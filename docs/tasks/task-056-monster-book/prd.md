# Monster Book — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-04
---

## 1. Overview

The Monster Book is a per-character collection of "monster cards" earned by defeating mobs and picking up the card item they drop. Each card corresponds to a specific monster (item id `2380000`–`238xxxx`); each character can own up to **5 copies** of any given card, with each duplicate raising that card's *card level* by one. Total unique cards drives the character's *book level*, which in turn grants a passive EXP bonus and gates certain quests. The character may also designate a single owned card as their "cover," which is broadcast on character info.

This task introduces a new microservice — `atlas-monster-book` — that owns the per-character card collection, cover selection, derived stats (book level, normal/special counts, EXP bonus), and the Kafka and JSON:API surface needed by other services. Drop pickup, packet I/O, quest gating, and the UI all integrate with the new service via existing patterns; cosmetic and behavioural details (the card-gain chat line, the foreign card effect, the cover-set acknowledgement) match Cosmic for client compatibility.

The work has one notable upstream prerequisite: the `consumeOnPickup` flag on consumable items is currently plumbed through every data layer but never read on the server. Wiring that flag — so that picking up a card item on the floor causes the same use-effect that consuming it from inventory would — is in scope for this task because Monster Book is the first feature that depends on it.

## 2. Goals

Primary goals:
- Persist a per-character monster card collection (cardId → level, capped at 5) and cover, scoped by tenant.
- Add a card to the collection when the character picks up a card item that has `consumeOnPickup = true`, then trigger the same broadcast packets the v83 client expects.
- Compute book level, total/normal/special card counts, and a Monster Book EXP bonus consumable by `atlas-channel`.
- Allow other services and the UI to read the collection, cover, and derived stats over JSON:API.
- Expose the EXP bonus through the existing `EXPERIENCE_DISTRIBUTION:MONSTER_BOOK` Kafka pipeline so the right-side EXP gain panel shows it.
- Wire `MonsterBookCountRequirement` into `atlas-quest`'s requirement evaluator so quests requiring N total cards can be checked.
- Wire the `consumeOnPickup` flag on the drop-pickup path so that pickup-consumable items (cards being the first concrete example) trigger their use effect on the floor instead of going into the inventory.
- Surface the character's cover and full collection in the `atlas-ui` character detail page as a widget.

Non-goals:
- "Fill the Book" / card-exchange NPC scripts (e.g., the canonical Barry NPC) — these are scriptable on top of the new API and out of scope for v1.
- HP/MP bonuses or other book-level-driven stats beyond EXP bonus.
- Trading, dropping, or destroying owned cards.
- Card scrolls, card-set bonuses, or card decks beyond cover selection.
- Configurable per-tenant tuning of the EXP bonus or special-card threshold (a fixed Cosmic-parity formula ships in v1; tenant-configurable knobs may follow later).

## 3. User Stories

- As a player, when I kill a monster that drops its card and pick it up, I want the card added to my Monster Book and a "You got a card!" chat line shown to me so I know the pickup counted.
- As a player, when I pick up a duplicate of a card I already have, I want that card's level to increase up to 5; once at 5 the pickup should still be acknowledged but not increase the level further.
- As a player, I want nearby players to see a card-gain animation over my character so collecting cards feels public.
- As a player, I want to choose any card I already own as my "cover" and have that selection persist across logins.
- As a player, I want my Monster Book level to increase as I collect more unique cards, and I want to see a Monster Book EXP bonus contribution in the right-side EXP gain breakdown.
- As a quest designer, I want to gate a quest behind owning at least N total cards.
- As a server admin (atlas-ui user), I want to view a character's cover card and full collection (which cards, at what level) on the character detail page.

## 4. Functional Requirements

### 4.1 Card identity and validation

- A card item is identified by `itemId / 10000 == 238` (range `2380000`–`2389999` inclusive).
- A card is *special* iff `cardId / 1000 >= 2388`, i.e. cardIds in `2388000+`. All others are *normal*.
- The reference `cardId → mobId` mapping is owned by `atlas-data`. `atlas-monster-book` queries `atlas-data` to validate cards, list known cards, and look up the corresponding mob for display in clients/UI.
- The cover is valid iff it is `0` (cleared) or a cardId the character currently owns at level ≥ 1. (Cosmic only validates `id == 0 || id / 10000 == 238`; this PRD tightens that to "owned" because allowing un-owned covers leaks information and serves no gameplay purpose.)

### 4.2 Collection — add card

When `atlas-monster-book` receives a `CARD_PICKED_UP` (or equivalent) Kafka event from the drops pipeline, it MUST:

1. Validate the cardId is in `2380000`–`2389999`. Reject otherwise.
2. Look up the current `(charId, cardId)` row.
3. If absent, insert a new row at level 1. Increment the in-memory `normalCard` or `specialCard` counter and recompute book level.
4. If present and `level < 5`, increment the level by 1. Do not recompute book level (only first-acquisition affects unique-card count, which drives book level).
5. If present and `level == 5`, do not change persisted state. Emit a "card already at max" status for downstream packet construction.
6. Emit a Kafka status event (`MONSTER_BOOK_CARD_ADDED` with subtype `ADDED` or `ALREADY_FULL`) with `{characterId, cardId, newLevel, full}` for `atlas-channel` to translate into:
   - `MONSTER_BOOK_SET_CARD` (SendOpcode `0x53`) packet to the owner: `flag=1` (added) or `flag=0` (already full), cardId, level.
   - `SHOW_ITEM_GAIN_INCHAT` (`0x0D`) "got a card" chat line to the owner — only on `ADDED`, not on `ALREADY_FULL`.
   - `SHOW_FOREIGN_EFFECT` (`0x0D`) broadcast to the rest of the map for the foreign card-gain animation — only on `ADDED`.

### 4.3 Cover

- A character has a single optional cover cardId (default `0`).
- A read-modify-write API allows the character to set their cover. Validation rules from §4.1 apply.
- On valid update, `atlas-monster-book` emits a `MONSTER_BOOK_COVER_CHANGED` Kafka status with the new cover, which `atlas-channel` translates to `MONSTER_BOOK_SET_COVER` (`0x54`) for the owner.
- The current cover is queryable both via REST and via Kafka request/response so login/channel flows can decorate the character info packet without reading the database directly. (See §6 and §7.)

### 4.4 Book level and counts

- `bookLevel` is derived from `totalUniqueCards = normalCount + specialCount` using Cosmic's formula:
  ```
  level = 0
  expToNext = 1
  do {
      level += 1
      expToNext += level * 10
  } while (totalUniqueCards >= expToNext)
  // bookLevel = level
  ```
- `bookLevel` is recalculated on first-time acquisition of a card (not on level-up of an existing card).
- `bookLevel`, `normalCount`, `specialCount`, and `totalUniqueCards` are exposed both via REST and as fields on the per-character status emitted on any change.

### 4.5 EXP bonus

- The Monster Book EXP bonus is a non-negative integer percentage value contributed to the right-side EXP panel via the existing `EXPERIENCE_DISTRIBUTION:MONSTER_BOOK` distribution type.
- v1 formula (Cosmic-parity, fixed): `bonusPercent = bookLevel`. (Cosmic's exact formula varies by fork; the v1 formula will be confirmed during the design phase, but the integration shape is fixed: a percent value driven by book level.)
- The bonus is recomputed and emitted on any book-level change for that character.
- `atlas-channel` already consumes `EXPERIENCE_DISTRIBUTION:MONSTER_BOOK` and writes to `experience_status.MonsterBookBonus`. No change is required there.

### 4.6 Login decoration

- Login/channel flows MUST query `atlas-monster-book` for the character's cover when assembling the character info packet, rather than reading from the character table or from a cached field on the character model. This is the decorator pattern already used elsewhere in the channel for character-adjacent data.
- For book-level-dependent fields in the character info packet (if any are wired in v1), the same decorator pattern applies.

### 4.7 Drop pickup → consume-on-pickup wiring (prerequisite)

- The `consumeOnPickup` flag is currently exposed by `atlas-data`, `atlas-consumables`, `atlas-inventory`, and `atlas-npc-shops` consumable models, but is never read at runtime.
- This task wires the flag into the drop pickup path: when a character picks up a consumable drop and `consumeOnPickup = true`, the server MUST trigger the consumable's use-effect (the same effect that would fire if the character used the item from inventory) without inserting the item into inventory.
- For card items (`itemId / 10000 == 238`), the use-effect MUST emit a Kafka `CARD_PICKED_UP` (or equivalent) event into `atlas-monster-book`'s consumer.
- This wiring is implemented in whichever service owns the post-pickup consumable use orchestration (likely `atlas-inventory` or `atlas-consumables`, to be decided in design); it is **not** specific to cards. Once wired, any future consume-on-pickup item type will benefit.

### 4.8 Quest requirement

- Add a `MonsterBookCount` requirement type to `atlas-quest`'s requirement evaluator.
- Configuration shape: `{type: "monster_book_count", requiredCards: N}`.
- Evaluator queries `atlas-monster-book` for `totalUniqueCards` and passes iff `total >= N`.
- Existing quest data sourced from WZ should expose this requirement (out of scope to backfill quest data in this task; the *capability* must exist).

### 4.9 UI widget

- The atlas-ui character detail page gets a new "Monster Book" widget showing:
  - The cover card (image and name) or a placeholder if cover is `0`.
  - Book level, total unique cards, normal count, special count.
  - A scrollable/searchable list of owned cards: cardId, mob name (resolved via atlas-data), and current card level (1–5).
- Read-only in v1 (no admin "set cover" or "add card" actions from the UI).

## 5. API Surface

### 5.1 atlas-monster-book — REST (JSON:API)

All endpoints are tenant-scoped via the standard tenant header.

- `GET /characters/{characterId}/monster-book`
  Resource type: `monster-book`. Attributes: `bookLevel`, `totalUniqueCards`, `normalCount`, `specialCount`, `coverCardId`, `expBonusPercent`. Relationships: `cards` (to-many `monster-book-card`).
- `GET /characters/{characterId}/monster-book/cards`
  Resource type: `monster-book-card`. Attributes: `cardId`, `level`, `firstAcquiredAt`. Supports filter `?filter[isSpecial]=true|false`.
- `GET /characters/{characterId}/monster-book/cards/{cardId}`
- `PATCH /characters/{characterId}/monster-book` — accepts `{coverCardId}` updates only. Validation per §4.3.
- `GET /reference/cards` — proxies/joins atlas-data card metadata for UI convenience (cardId, mobId, mob name, isSpecial). May be deferred to UI-side composition if simpler.

Errors:
- `404` if character has no Monster Book row (auto-created on first valid event; until then, return `200` with empty/default attributes).
- `422` for invalid cover (non-card id, or unowned card).

### 5.2 Kafka — commands / events

Topic naming follows existing conventions; final names confirmed in design.

Inbound commands (consumed by `atlas-monster-book`):
- `MONSTER_BOOK.CARD_PICKED_UP` `{tenantId, characterId, cardId, source: "drop_pickup" | "admin"}` — the canonical add path.
- `MONSTER_BOOK.SET_COVER` `{tenantId, characterId, coverCardId}` — initiated by atlas-channel on receiving v83 RecvOpcode `0x39` (MONSTER_BOOK_COVER).

Outbound status events (produced by `atlas-monster-book`):
- `MONSTER_BOOK.CARD_ADDED` `{tenantId, characterId, cardId, newLevel, full: bool}` — `full=true` means the card was already at level 5; consumers use this to decide which client packets to send.
- `MONSTER_BOOK.COVER_CHANGED` `{tenantId, characterId, coverCardId}`.
- `MONSTER_BOOK.STATS_CHANGED` `{tenantId, characterId, bookLevel, normalCount, specialCount, totalUniqueCards, expBonusPercent}` — fired whenever any of these change. Drives EXP-distribution emission and any other downstream consumers.

Existing topic reuse:
- `atlas-monster-book` (or a thin shim service) MUST emit `EXPERIENCE_DISTRIBUTION:MONSTER_BOOK` updates for the affected character whenever `expBonusPercent` changes, using the same shape `atlas-channel` already consumes.

### 5.3 atlas-channel — packet handlers

- Inbound: register handler for v83 RecvOpcode `0x39` (MONSTER_BOOK_COVER). Body: one Int (cardId, `0` to clear). On receive, emit `MONSTER_BOOK.SET_COVER` Kafka command.
- Outbound: subscribe to `MONSTER_BOOK.CARD_ADDED` and `MONSTER_BOOK.COVER_CHANGED`; translate to v83 SendOpcodes `0x53` and `0x54` respectively, plus the `0x0D` chat / foreign-effect packets per §4.2.
- Login: when assembling the character info packet, query `atlas-monster-book` for cover (decorator pattern, no DB join).

### 5.4 atlas-quest — requirement evaluator

- New requirement `monster_book_count` with `requiredCards` config field. Evaluator calls `atlas-monster-book` REST.

## 6. Data Model

### 6.1 atlas-monster-book schema

```
TABLE monster_book_collections
  tenant_id          UUID NOT NULL
  character_id       BIGINT NOT NULL
  cover_card_id      INT NOT NULL DEFAULT 0
  book_level         INT NOT NULL DEFAULT 1
  normal_count       INT NOT NULL DEFAULT 0
  special_count      INT NOT NULL DEFAULT 0
  exp_bonus_percent  INT NOT NULL DEFAULT 0
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
  PRIMARY KEY (tenant_id, character_id)

TABLE monster_book_cards
  tenant_id           UUID NOT NULL
  character_id        BIGINT NOT NULL
  card_id             INT NOT NULL          -- 2380000..2389999
  level               SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 5)
  is_special          BOOLEAN NOT NULL       -- denormalised from card_id for index/filter use
  first_acquired_at   TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
  PRIMARY KEY (tenant_id, character_id, card_id)
  FOREIGN KEY (tenant_id, character_id) REFERENCES monster_book_collections (tenant_id, character_id) ON DELETE CASCADE
INDEX (tenant_id, character_id, is_special)
```

Notes:
- `book_level`, `normal_count`, `special_count`, `exp_bonus_percent` are denormalised onto the collection row to avoid recomputing on every read. They are recomputed and updated transactionally on every card insert.
- `monster_book_collections` is created lazily on first card-add for that character; reads return defaults if absent (see §5.1).
- The reference table `monstercarddata` from Cosmic has **no equivalent** in this schema — atlas-data owns that mapping.

### 6.2 atlas-character

- No schema changes. Cover is **not** added to `atlas-character`'s character table; it stays in `atlas-monster-book`. Login decorator pattern (§4.6) is the integration surface.

### 6.3 Migration

- A single new migration in `atlas-monster-book` creates both tables.
- No backfill (collections are empty for all existing characters, which is correct).
- Character deletion: `atlas-monster-book` subscribes to the existing character-deleted Kafka event and cascades deletion of the collection + cards rows. (Same pattern as other per-character services.)

## 7. Service Impact

- **atlas-monster-book** (new): full service per the existing template — DB, processors, REST handlers, Kafka producer/consumer, configuration loader, character-lifecycle subscriber.
- **atlas-data**: confirm/expose `cardId → mobId` (and mob-name resolution) reference data via REST. May require a new endpoint if not already present.
- **atlas-drops** *or* **atlas-inventory** *or* **atlas-consumables**: implement the `consumeOnPickup` runtime branch (§4.7). The owning service will be selected during design; the decision criterion is "wherever the post-pickup item-flow currently dispatches consumable use effects."
- **atlas-channel**:
  - Register RecvOpcode `0x39` handler emitting `MONSTER_BOOK.SET_COVER` command.
  - Add SendOpcode `0x53` / `0x54` packet builders.
  - Add foreign-effect (`0x0D`) and in-chat (`0x0D`) builders if not present.
  - Add Kafka consumers for the new `MONSTER_BOOK.CARD_ADDED` and `MONSTER_BOOK.COVER_CHANGED` events.
  - Add login-time decorator that fetches the cover for the character info packet.
- **atlas-quest**: implement the `monster_book_count` requirement evaluator and wire it into the requirement dispatcher.
- **atlas-ui**: new "Monster Book" widget on the character detail page; consumes `atlas-monster-book` REST + atlas-data card metadata.
- **atlas-character**: no code changes. (Specifically: no `monster_book_cover` column added.)
- **atlas-tenants**: no v1 configuration knobs.

## 8. Non-Functional Requirements

- **Multi-tenancy**: every read/write/event is scoped by `tenant_id` from `tenant.MustFromContext(ctx)` per Atlas convention. All composite keys lead with `tenant_id`.
- **Concurrency**: card-add from drop pickup is high-frequency. The processor MUST handle concurrent inserts for the same `(tenant, character, card)` deterministically — `INSERT ... ON CONFLICT DO UPDATE SET level = LEAST(level + 1, 5)` is the expected idiom, executed in a transaction that also updates the collection row's denormalised counts and `book_level` via row-locked read-modify-write.
- **Idempotency**: each `MONSTER_BOOK.CARD_PICKED_UP` command should carry a producer-generated event id. Duplicate ids within a short window MUST NOT double-add. (Exact mechanism — dedup table vs. consumer offset reasoning — decided at design.)
- **Observability**: standard Atlas logging (request id, tenant, character) on every state change; Prometheus counters for `cards_added_total`, `covers_changed_total`, `book_level_ups_total`; OpenTelemetry spans on processor methods.
- **Performance**: REST `GET /characters/{id}/monster-book` must return in <50ms p95 for collections up to ~1000 cards. The denormalised counts on the collection row are why the summary endpoint is fast; the per-card list endpoint should support pagination.
- **Backwards compatibility**: existing `experience_status.MonsterBookBonus` field continues to be populated via the existing pipeline; this task only ensures a real producer drives the value.
- **Security**: REST endpoints require the standard tenant + character authorization per Atlas convention. Setting another character's cover is rejected with 403 at the handler layer.

## 9. Open Questions

1. **Exact EXP-bonus formula.** v1 ships a Cosmic-parity formula driven by book level; the precise mapping (linear `level%`, stepped table, capped, etc.) needs confirmation from the Cosmic source / fork the project tracks. Resolved at design phase.
2. **Owner of consume-on-pickup wiring.** Most likely `atlas-drops` triggers, `atlas-inventory` orchestrates, and an item-type dispatcher routes the use-effect, but the exact location and contract are open until the existing post-pickup flow is read end-to-end. Design phase.
3. **Foreign-effect broadcast scope.** Cosmic broadcasts to the entire map. We can keep that; future optimisation could limit to nearby viewers via the existing channel-foreman scope. v1 = match Cosmic.
4. **Idempotency mechanism for card-add commands.** Dedup table vs. consumer-offset reasoning vs. requiring producers to be idempotent — chosen at design.
5. **Reference data shape from atlas-data.** Whether the existing endpoints already cover `cardId → mobId` or a new endpoint is needed. To verify in design.
6. **Quest data backfill.** This task delivers the *capability* to evaluate `monster_book_count`. Whether any existing quest configurations actually use it (and thus need data updates) is a separate question deferred to the quest data owner.
7. **Card stack/inventory side-effects.** Cards normally enter the etc. inventory with stacking; on consume-on-pickup, no inventory entry is created. Confirm this matches the v83 client expectation (i.e., the client does not expect to see the card item in inventory for any duration).

## 10. Acceptance Criteria

- [ ] `atlas-monster-book` service exists, builds, has working Docker image, and ships with migrations that create the two tables.
- [ ] Picking up a card item (`238xxxx`) on the floor for a character with no prior collection results in: (a) a row in `monster_book_cards` at level 1, (b) a `monster_book_collections` row with appropriate counts and `book_level`, (c) a `MONSTER_BOOK_SET_CARD` packet to the owner, (d) `SHOW_ITEM_GAIN_INCHAT` to the owner, (e) `SHOW_FOREIGN_EFFECT` to other players on the map, (f) the card item NOT inserted into the character's inventory.
- [ ] Picking up the same card again increments level up to 5, with the corresponding `MONSTER_BOOK_SET_CARD` packet (`flag=1`).
- [ ] Picking up a card already at level 5 sends `MONSTER_BOOK_SET_CARD` with `flag=0` and does NOT send the chat / foreign-effect packets and does NOT modify persisted level.
- [ ] Sending v83 RecvOpcode `0x39` with a valid owned cardId updates the cover, persists, and echoes `MONSTER_BOOK_SET_COVER`.
- [ ] Sending RecvOpcode `0x39` with `cardId=0` clears the cover.
- [ ] Sending RecvOpcode `0x39` with an unowned or non-card id is rejected without state change.
- [ ] Book level recalculates correctly per the formula on first acquisition; total/normal/special counts are accurate.
- [ ] `experience_status.MonsterBookBonus` reflects the character's current `expBonusPercent` and updates within one event-loop tick after a book-level change.
- [ ] A quest configured with `monster_book_count: N` requirement is gated correctly: `totalUniqueCards >= N` passes, `<` fails.
- [ ] On character delete, all `monster_book_collections` and `monster_book_cards` rows for that character are removed.
- [ ] Cover and collection are tenant-isolated: a character in tenant A cannot read or modify any monster-book data for a same-id character in tenant B.
- [ ] atlas-ui character detail page shows the Monster Book widget with cover, book level, counts, and a paginated card list (cardId, mob name from atlas-data, level).
- [ ] DOM-* / FE-* checklists pass for new code (run reviewer agents as part of execute phase).
- [ ] All affected services (`atlas-monster-book`, `atlas-channel`, `atlas-quest`, the consume-on-pickup owner, atlas-ui) build cleanly and their tests pass.
