# Monster Book — Design

Version: v1
Status: Draft
Created: 2026-05-04
PRD: `prd.md` (approved)

---

## 1. Purpose & scope of this document

The PRD fixes *what* the Monster Book must do and *which* services it touches. This document fixes *how* the work is structured: which service owns which decision, what the Kafka and REST surfaces look like in detail, where the new code is grafted into existing flows, and what the alternatives were. It also resolves every PRD open question.

Sections that the PRD already nailed (data model, REST shape, acceptance criteria) are not duplicated here — they are referenced by section number.

## 2. Decisions index (resolves PRD open questions)

| # | PRD open question | Decision |
| - | --- | --- |
| 1 | EXP-bonus formula | `bonusPercent = bookLevel` (1% per level). §6.4. |
| 2 | Owner of consume-on-pickup wiring | atlas-inventory checks the flag pre-insert; emits generic `ITEM.CONSUMED_ON_PICKUP` event; atlas-consumables fans it into its existing effect dispatcher. §4.1. |
| 3 | Foreign-effect broadcast scope | Match Cosmic — broadcast to entire map via existing `EffectSimpleForeign` writer. §5.3. |
| 4 | Idempotency mechanism | Per-row `last_event_id` UUID columns on the two new tables, atomic upsert with `WHERE last_event_id IS DISTINCT FROM excluded.last_event_id`. No dedup table. §7. |
| 5 | atlas-data card metadata shape | No new resource. Existing `/data/consumables/{cardId}` already exposes `monsterId` parsed from WZ `info/mob`. atlas-monster-book validates by ID range only. §4.5. |
| 6 | Quest data backfill | Out of scope (per PRD). |
| 7 | Card stack/inventory side-effects | Card item is never inserted into inventory at any point. §4.1. |

Additional design-time decisions that weren't open in the PRD but bind here:

| # | Topic | Decision |
| - | --- | --- |
| 8 | Quest condition contract shape | `{type: "monsterBookCount", operator, value}` — uniform with existing atlas-query-aggregator conditions; supersedes the PRD's per-condition `requiredCards` field. §4.4. |
| 9 | Where the `monsterBookCount` condition is evaluated | atlas-query-aggregator (existing validation dispatcher) — not atlas-quest directly. §4.4. |
| 10 | Service template | atlas-keys (smallest service matching shape). §3.1. |
| 11 | Module name | `atlas-monster-book` (short form, matches `atlas-keys`/`atlas-fame` convention). §3.1. |
| 12 | Login decorator pattern | REST decorator on atlas-channel `character.Processor`, alongside the existing `SkillModelDecorator`/`QuestModelDecorator`/`PetAssetEnrichmentDecorator`. §5.4. |

## 3. atlas-monster-book service

### 3.1 Structure

Module: `atlas-monster-book`. Path: `services/atlas-monster-book/atlas.com/monster-book/`. Layout (mirrors atlas-keys):

```
monster-book/
├── main.go                       # bootstrap: db, migrate, consumers, REST
├── go.mod                        # module atlas-monster-book
├── logger/
├── collection/                   # per-character collection (cover + denormalised counts)
│   ├── model.go                  # immutable Model + getters
│   ├── builder.go                # ModelBuilder
│   ├── entity.go                 # GORM entity + Migration(db)
│   ├── administrator.go          # raw GORM read/write helpers (idempotent upsert)
│   └── processor.go              # Processor interface + Impl + factory
├── card/                         # per-character per-card row
│   ├── model.go
│   ├── builder.go
│   ├── entity.go
│   ├── administrator.go
│   └── processor.go
├── character/                    # consumer of EVENT_TOPIC_CHARACTER_STATUS
│   └── consumer.go               # cascade-delete on character deleted
├── kafka/
│   ├── consumer/
│   │   ├── monsterbook/          # MONSTER_BOOK.CARD_PICKED_UP, MONSTER_BOOK.SET_COVER
│   │   └── character/            # character lifecycle
│   ├── message/
│   │   ├── message.go            # Buffer, Emit, EmitWithResult
│   │   └── monsterbook/          # topic constants, status event types
│   └── producer/
│       └── producer.go
└── rest/
    ├── resource.go               # router + RegisterHandler/RegisterInputHandler wiring
    └── handler.go                # GET, PATCH handlers
```

The two domain packages (`collection`, `card`) are split because the row lifecycles differ: a collection row is created lazily on first valid card-add and updated on cover changes; card rows are upsert-on-pickup. Both processors are constructed via `NewProcessor(l, ctx, db)` and accept a `*gorm.DB` (transactional ops use `db.Transaction(...)`).

### 3.2 Domain types

Both packages follow the standard immutable-model + builder pattern.

`collection.Model` fields (private + getters): `tenantId uuid.UUID`, `characterId uint32`, `coverCardId uint32`, `bookLevel uint16`, `normalCount uint16`, `specialCount uint16`, `expBonusPercent uint16`, `lastCoverEventId *uuid.UUID`, `createdAt time.Time`, `updatedAt time.Time`.

`card.Model` fields: `tenantId uuid.UUID`, `characterId uint32`, `cardId uint32`, `level uint8`, `isSpecial bool`, `lastEventId *uuid.UUID`, `firstAcquiredAt time.Time`, `updatedAt time.Time`.

Per CLAUDE.md DOM-21, before introducing new numeric types we check `libs/atlas-constants/`. Existing types reused: `world.Id` (byte), `channel.Id` (byte), `_map.Id` (uint32), `character.Id` (uint32), `tenant.Model` from context. Card IDs are plain `uint32` — there is no `card.Id` type yet, and the cardId carries no semantic distinction from a regular item id (it *is* an item id). If shared usage emerges later (e.g., cards referenced by another service), we can promote to a typed alias under `libs/atlas-constants/item/card/`.

### 3.3 Tables

Per PRD §6.1 plus the two `last_event_id` columns required by §7 of this design:

```sql
TABLE monster_book_collections
  tenant_id              UUID    NOT NULL
  character_id           BIGINT  NOT NULL
  cover_card_id          INT     NOT NULL DEFAULT 0
  book_level             INT     NOT NULL DEFAULT 1
  normal_count           INT     NOT NULL DEFAULT 0
  special_count          INT     NOT NULL DEFAULT 0
  exp_bonus_percent      INT     NOT NULL DEFAULT 0
  last_cover_event_id    UUID    NULL              -- new in this design
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
  PRIMARY KEY (tenant_id, character_id)

TABLE monster_book_cards
  tenant_id              UUID    NOT NULL
  character_id           BIGINT  NOT NULL
  card_id                INT     NOT NULL
  level                  SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 5)
  is_special             BOOLEAN NOT NULL
  last_event_id          UUID    NULL               -- new in this design
  first_acquired_at      TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
  PRIMARY KEY (tenant_id, character_id, card_id)
  FOREIGN KEY (tenant_id, character_id)
      REFERENCES monster_book_collections (tenant_id, character_id) ON DELETE CASCADE
  INDEX (tenant_id, character_id, is_special)
```

Migration is `gorm.AutoMigrate(&collectionEntity{}, &cardEntity{})` invoked from `main.go` at boot, mirroring atlas-keys. No backfill.

## 4. Cross-service flows

### 4.1 Card pickup (consume-on-pickup wiring)

Order of operations from client packet to monster-book row:

1. **atlas-channel** `socket/handler/drop_pick_up.go:15` decodes the pickup packet and calls `drop.NewProcessor(l, ctx).RequestReservation(...)`. *(unchanged)*
2. **atlas-drops** reserves the drop with a `transactionId` (UUID). *(unchanged)*
3. **atlas-inventory** consumes the reservation (`kafka/consumer/drop/consumer.go:43`) and routes to `compartment.AttemptItemPickUpAndEmit(...)` (`compartment/processor.go:1142`). **CHANGE:** before computing the target slot/insert, fetch the consumable record via the existing `data/consumable` REST/model layer (where `ConsumeOnPickup bool` is already extracted — `data/consumable/model.go:59`, `data/consumable/rest.go:20,117`) and check it. If true:
   - Skip the asset insert entirely. No `INVENTORY_OPERATION` packet is emitted.
   - Buffer a `ITEM.CONSUMED_ON_PICKUP {tenantId, characterId, itemId, transactionId, fieldKey}` Kafka command on the dedicated topic `EVENT_TOPIC_ITEM_CONSUMED_ON_PICKUP` (new constant in `libs/...` or atlas-inventory's message package, mirroring how other inventory-emitted commands are placed today).
   - Still call `dropProcessor.RequestPickUp()` to remove the drop from the field — the player did successfully pick it up.
   The change is gated by item type to keep blast radius small: only items where `inventory.TypeFromItemId == TypeValueUse` are checked. (Equipables and setup never carry `consumeOnPickup=true` in current data.)
4. **atlas-consumables** grows a new consumer `kafka/consumer/pickup/consumer.go` for `ITEM.CONSUMED_ON_PICKUP`. The handler:
   - For cards (`itemId / 10000 == 238`): produce `MONSTER_BOOK.CARD_PICKED_UP {tenantId, characterId, cardId, eventId, source: "drop_pickup"}`. The `eventId` is the upstream `transactionId`.
   - For all other use-type items (potions etc., reserved for future pickup-consumables): call into the existing `consumable.RequestItemConsume` effect dispatcher (`consumable/processor.go:181`). The current dispatcher takes `slot.Position`; we will add a "pickup mode" overload that doesn't require a slot — `ApplyItemEffects` itself does not depend on slot, only the inventory removal step does. That step is no-op under pickup mode because nothing was inserted.
   The card branch is the only code path exercised in this task; the reserved future-consumables branch is a parameterless `default:` returning `nil` for v1 unless a non-card pickup-consumable is encountered, in which case we log-and-skip. Aggressive scope discipline: implementing the generic path for items we do not have data for would be speculative.
5. **atlas-monster-book** `kafka/consumer/monsterbook/` handles `MONSTER_BOOK.CARD_PICKED_UP`:
   - Validate `2380000 <= cardId <= 2389999`. Reject out-of-range with a logged warning.
   - Compute `isSpecial = (cardId / 1000) >= 2388`.
   - Run the idempotent upsert (see §7) inside a single transaction that also touches the collection row.
   - Emit `MONSTER_BOOK.CARD_ADDED {tenantId, characterId, cardId, newLevel, full, eventId}` via the message buffer. If the upsert detected a duplicate event_id, return early without emitting (the player has already received their packet on the first delivery).
   - If `bookLevel` changed (only on first acquisition), also emit `MONSTER_BOOK.STATS_CHANGED {bookLevel, normalCount, specialCount, totalUniqueCards, expBonusPercent}` and an `EXPERIENCE_DISTRIBUTION:MONSTER_BOOK` event (see §6.4) so atlas-channel updates the right-side EXP panel.

### 4.2 Cover change

1. **atlas-channel** registers a new RecvOpcode `0x39` handler in `socket/handler/monster_book_cover.go`. Body: one int (cardId, `0` to clear). The handler calls `monsterbook.NewProcessor(l, ctx).RequestSetCover(s.CharacterId(), cardId)` which produces `MONSTER_BOOK.SET_COVER {tenantId, characterId, coverCardId, eventId: uuid.New()}`.
2. **atlas-monster-book** consumes `MONSTER_BOOK.SET_COVER`:
   - Validate `coverCardId == 0 || (2380000 <= coverCardId <= 2389999 AND character owns it at level >= 1)`.
   - Update `monster_book_collections.cover_card_id` with the same `last_cover_event_id`-guarded upsert.
   - Emit `MONSTER_BOOK.COVER_CHANGED {tenantId, characterId, coverCardId, eventId}`.
3. **atlas-channel** Kafka consumer for `MONSTER_BOOK.COVER_CHANGED` writes `MONSTER_BOOK_SET_COVER` (SendOpcode `0x54`) to the owning session.

### 4.3 Login decorator

When atlas-channel assembles the character info packet (`socket/handler/character_info_request.go:20-66`), it appends a new `MonsterBookCoverDecorator` to the existing decorator slice. The decorator calls `monsterbook.NewProcessor(l, ctx).GetByCharacterId(charId)` over REST, returning a small struct with `coverCardId` (and `bookLevel`, `expBonusPercent` if any character-info field needs them in v1; otherwise just cover). Errors are logged and swallowed — the player gets `coverCardId=0` rather than a failed login.

This mirrors `PetAssetEnrichmentDecorator`/`SkillModelDecorator`/`QuestModelDecorator`. No DB-side denormalization onto `atlas-character`.

### 4.4 Quest requirement

PRD §4.8 specifies the capability. Concrete plumbing:

- **atlas-quest** adds `MonsterBookCountCondition = "monsterBookCount"` to `data/validation/model.go`. In `data/validation/processor.go`'s `buildStartConditions()` (line 55) and the symmetric end-requirement construction inside `ValidateEndRequirements()` (line 208), when a quest definition references this requirement type, build `ConditionInput{Type: MonsterBookCountCondition, Operator: ">=", Value: requiredCards, ReferenceId: 0}`.
- **atlas-query-aggregator** adds a new case in its validation dispatcher (the existing service that consumes `data/validation/requests.go:15-21`'s POST). The case calls `GET /characters/{characterId}/monster-book` on atlas-monster-book, reads `totalUniqueCards`, and applies the standard operator/value comparison.

Why atlas-query-aggregator owns the actual evaluator: every other quest condition type is dispatched from there. Putting `monsterBookCount` anywhere else fragments the validation surface.

### 4.5 atlas-data integration

No atlas-data code changes. `services/atlas-data/atlas.com/data/consumable/reader.go:77` already parses `info/mob` from the WZ XML into `MonsterId` and exposes it as JSON:API `monsterId` at `GET /data/consumables/{itemId}`.

- **atlas-monster-book** does not call atlas-data on the hot path (validation is by ID range; cardId is upstream-trusted from the drop chain).
- **atlas-ui** chains `GET /data/consumables/{cardId}` → read `monsterId` → `GET /data/monsters/{monsterId}` for the mob name. Mirrors the way the UI already resolves item icon → monster references.

## 5. atlas-channel additions

### 5.1 Opcodes & writers

- **RecvOpcode `0x39`** `MONSTER_BOOK_COVER`: register in the channel handler map; body is one Int (cardId).
- **SendOpcode `0x53`** `MONSTER_BOOK_SET_CARD`: new writer in `libs/atlas-packet/character/clientbound/monsterbook/set_card.go`. Body: `byte(flag) + int(cardId) + int(level)` (Cosmic format). `flag=1` = added/levelled, `flag=0` = already at max.
- **SendOpcode `0x54`** `MONSTER_BOOK_SET_COVER`: new writer in `libs/atlas-packet/character/clientbound/monsterbook/set_cover.go`. Body: `int(cardId)`.

Both writers are registered in atlas-channel's `main.go` writer list alongside the existing `CharacterEffectWriter` and friends.

### 5.2 Effect packets (already present)

`libs/atlas-packet/character/clientbound/effect.go` already declares an `EffectSimple` mode `MonsterBookCardGet`, `EffectSimpleForeign` for map-broadcast effects, and `CharacterEffectWriter` / `CharacterEffectForeignWriter` (the 0x0D opcode pair). The Kafka consumer for `MONSTER_BOOK.CARD_ADDED` emits:

- `MONSTER_BOOK_SET_CARD` (`0x53`) to the owner with the appropriate flag — always.
- `EffectSimple{mode: MonsterBookCardGet}` to the owner — only when `full == false`.
- `EffectSimpleForeign{characterId, mode: MonsterBookCardGet}` broadcast to map peers — only when `full == false`. Broadcast scope = entire map (matches Cosmic; future optimisation deferred).

PRD §4.2 also calls out a "SHOW_ITEM_GAIN_INCHAT" chat-line packet on `ADDED`. In v83 the `EffectSimple{MonsterBookCardGet}` write generally drives the chat-line render client-side; whether a separate ItemGainInChat packet is also required is verified at plan-phase by inspecting the v83 client / Cosmic packet log. If yes, it is emitted alongside the EffectSimple, gated on `full == false` per PRD.

### 5.3 Foreign-effect scope confirmation

PRD open question 3: scope is whole map. The `EffectSimpleForeign` writer already supports map-broadcast via the existing channel session-foreman. No changes needed to enable it.

### 5.4 Decorator placement

`character_info_request.go:27-31` already constructs a decorator slice. Add a single line: `decorators = append(decorators, cp.MonsterBookCoverDecorator)`. The decorator method lives on `character.Processor` next to the existing decorators in `character/processor.go:70-130`.

### 5.5 EXP-bonus pipe (zero-change)

`atlas-channel/kafka/consumer/character/consumer.go:269-270` already consumes `ExperienceDistributionTypeMonsterBook` and writes `experience_status.MonsterBookBonus`. atlas-monster-book emits these events on book-level changes (§6.4) and no atlas-channel code changes are needed.

## 6. atlas-monster-book internals

### 6.1 REST surface

Per PRD §5.1; minor refinements:

- `GET /characters/{characterId}/monster-book` — JSON:API resource type `monster-book`. Returns the collection row's denormalised stats. If no row exists yet, return a 200 with default attributes (no 404 — avoids special-casing on the consumer side, matches PRD §5.1 note).
- `GET /characters/{characterId}/monster-book/cards?page[offset]=&page[limit]=&filter[isSpecial]=` — JSON:API resource type `monster-book-card`. Paginates because §8 requires sub-50ms p95 on summaries; the unbounded card list could exceed that for completionist characters. Default `limit = 100`, max `200`.
- `GET /characters/{characterId}/monster-book/cards/{cardId}` — single card lookup.
- `PATCH /characters/{characterId}/monster-book` — accepts only `coverCardId`. Validation per §4.2. On success, returns the updated collection resource.

The `GET /reference/cards` endpoint floated in PRD §5.1 is **dropped**: atlas-data already exposes per-card metadata via `/data/consumables/{cardId}`; a re-shaping layer in atlas-monster-book would only duplicate that, and atlas-ui is the only realistic consumer.

### 6.2 Kafka surface

Topics (constant names confirmed at implementation time):

| Direction | Topic | Body |
| --- | --- | --- |
| In (cmd) | `MONSTER_BOOK.CARD_PICKED_UP` | `{tenantId, characterId, cardId, eventId, source}` |
| In (cmd) | `MONSTER_BOOK.SET_COVER` | `{tenantId, characterId, coverCardId, eventId}` |
| Out (status) | `MONSTER_BOOK.CARD_ADDED` | `{tenantId, characterId, cardId, newLevel, full, eventId}` |
| Out (status) | `MONSTER_BOOK.COVER_CHANGED` | `{tenantId, characterId, coverCardId, eventId}` |
| Out (status) | `MONSTER_BOOK.STATS_CHANGED` | `{tenantId, characterId, bookLevel, normalCount, specialCount, totalUniqueCards, expBonusPercent}` |
| Out (status) | `EXPERIENCE_DISTRIBUTION` (existing topic, reused) | `{characterId, distributionType: "MONSTER_BOOK", amount: expBonusPercent}` |

`STATS_CHANGED` is emitted only when the collection row's denormalised values change — i.e., on first-time card acquisition, never on a level-up of an existing card. The card-add transaction either updates the collection row (if first-time) and emits `STATS_CHANGED`, or doesn't touch it (if level-up only).

### 6.3 Processors

`collection.Processor`:

- `GetByCharacterId(charId)` — returns Model or "default" (cover=0, all counts=0) if absent.
- `SetCoverAndEmit(mb, eventId, charId, cardId)` — validates ownership via `card.Processor`, runs the guarded upsert, buffers `MONSTER_BOOK.COVER_CHANGED`.
- `RecomputeAndEmit(mb, eventId, charId)` — internal helper called by the card-add path; recomputes counts/level/expBonusPercent, persists, buffers `MONSTER_BOOK.STATS_CHANGED` and `EXPERIENCE_DISTRIBUTION` if the values changed.

`card.Processor`:

- `GetByCharacterId(charId, page)` — paginated list.
- `GetByCharacterIdAndCardId(charId, cardId)` — single lookup; used by `SetCover` validation.
- `AddAndEmit(mb, eventId, charId, cardId)` — main pickup handler. Returns `(added bool, full bool, newLevel uint8, firstAcquisition bool, duplicate bool)`. The status emission is composed in the consumer based on these flags, not deep in the processor — keeps `Processor` concerned with state, message buffer with packets.

Both processors take `*gorm.DB` and run inside a single `db.Transaction(...)` for the card-add flow so card upsert + collection upsert + message buffer commit (via `message.Emit(p)`) are atomic.

### 6.4 EXP-bonus formula

`expBonusPercent = uint16(bookLevel)`. With the PRD's level formula, this caps practical bonuses around 5-10% in normal play. Encoded directly in `collection.Processor.RecomputeAndEmit` and emitted on the `EXPERIENCE_DISTRIBUTION` topic on every change.

A future task can add a tenant-configurable formula per `atlas-tenants` resource `monster-book` — out of scope here.

### 6.5 Character-deletion cascade

`character/consumer.go` subscribes to `EVENT_TOPIC_CHARACTER_STATUS` and filters for `StatusEventTypeDeleted`. On match, deletes the matching `monster_book_collections` row inside the tenant scope; the `monster_book_cards` rows cascade via FK. Mirrors atlas-keys's pattern verbatim (`atlas-keys/atlas.com/keys/kafka/consumer/character/consumer.go:56-68`).

## 7. Idempotency

All Kafka deliveries in Atlas are at-least-once. Card-add is `level := LEAST(level + 1, 5)` — non-idempotent if naively replayed. Mechanism:

- Each `MONSTER_BOOK.CARD_PICKED_UP` carries an `eventId` (UUID) sourced from the upstream drop reservation's `transactionId`.
- The card upsert is guarded:
  ```sql
  INSERT INTO monster_book_cards (tenant_id, character_id, card_id, level, is_special, last_event_id, first_acquired_at, updated_at)
  VALUES ($1, $2, $3, 1, $4, $5, now(), now())
  ON CONFLICT (tenant_id, character_id, card_id) DO UPDATE
    SET level = LEAST(monster_book_cards.level + 1, 5),
        last_event_id = excluded.last_event_id,
        updated_at = now()
  WHERE monster_book_cards.last_event_id IS DISTINCT FROM excluded.last_event_id
  RETURNING (xmax = 0) AS inserted, level;
  ```
- A duplicate redelivery (same `eventId` for the same `(tenant, character, card)`) hits the `WHERE` guard, no rows update, `RETURNING` is empty → consumer treats it as a no-op and skips both the collection-row update and the downstream Kafka emissions.
- The cover-change path uses the same trick on `monster_book_collections.last_cover_event_id`.

What this *doesn't* defend against:
- Logical re-emissions with a *different* eventId for the same physical pickup. That would be an upstream contract violation (atlas-consumables handing us two distinct eventIds for the same drop). Not in scope.
- Cross-card replay (a single eventId associated with a different cardId on replay). Also an upstream violation.

A separate dedup table was rejected: it'd require a TTL purge job, an extra write per event, and decouples idempotency from the data being protected. The per-row column is one nullable UUID and one extra `WHERE` clause.

## 8. Concurrency

Concurrent card-add for the same `(tenant, character, card)` is serialized by the `(tenant_id, character_id, card_id)` unique constraint — Postgres takes the row lock implicitly during `INSERT ... ON CONFLICT`. The collection row is updated inside the same transaction with `SELECT ... FOR UPDATE` to prevent two parallel first-acquisitions from racing on `bookLevel` recomputation.

Concurrent cover changes are racey by definition (player toggles fast); last-write-wins is acceptable given the eventId guard prevents *replay* races, and both updates' downstream packets reach the client in send order.

## 9. Observability

Per PRD §8 plus:

- Prometheus counters: `monster_book_cards_added_total{result=added|levelled|already_full|duplicate}`, `monster_book_covers_changed_total`, `monster_book_book_level_ups_total`, `monster_book_validation_rejections_total{reason=out_of_range|unowned}`.
- OpenTelemetry spans on all `Processor.*AndEmit` methods, propagating tenant + character attributes.
- Standard Atlas log lines on every state change: tenant, character, eventId, before/after level, action.

## 10. Testing

- **atlas-monster-book unit**: processors with an in-memory sqlite (or a test container — match atlas-keys's existing approach). Coverage for: first acquisition, level-up, level-cap, duplicate eventId no-op, cover validation (unowned, out-of-range, zero), book-level recomputation, EXP bonus emission.
- **atlas-monster-book integration**: REST `GET`/`PATCH` round-trip with tenant header propagation; Kafka consumer dispatch for both inbound topics; cascade-delete on character deletion.
- **atlas-inventory**: targeted test for the `consumeOnPickup` branch — verify suppressed insert, drop reservation still released, command emitted.
- **atlas-consumables**: test for the new `ITEM.CONSUMED_ON_PICKUP` consumer's card branch.
- **atlas-channel**: handler test for the `0x39` decode → command emission; consumer tests for `0x53`/`0x54` packet emission given canned status events.
- **End-to-end smoke**: a single happy-path test that drops a card item on a map, picks it up, and asserts (a) no inventory row, (b) one card row, (c) collection counts, (d) `0x53` packet observed by an in-test session, (e) `EXPERIENCE_DISTRIBUTION` event observed.

DOM-* (backend) and FE-* (frontend) checklists run via `superpowers:requesting-code-review` at the end of execute phase per CLAUDE.md.

## 11. Frontend (atlas-ui)

Per PRD §4.9. Implementation specifics deferred to FE design; no architecture choices need locking here. The widget consumes:

- `GET /api/monster-book/characters/{characterId}/monster-book` (atlas-monster-book)
- `GET /api/monster-book/characters/{characterId}/monster-book/cards?page[...]` (atlas-monster-book)
- `GET /api/data/consumables/{cardId}` for `monsterId` (atlas-data, unchanged)
- `GET /api/data/monsters/{monsterId}` for mob name (atlas-data, unchanged)

Pagination on the cards list is enforced by the API; the UI uses TanStack React Query's infinite-query pattern per the frontend guidelines skill.

## 12. Rollout & risks

- **Sequencing**: atlas-data is unchanged, so deploy order is atlas-monster-book first (creates topics + tables on AutoMigrate at boot), then atlas-consumables (new consumer), then atlas-inventory (the consumeOnPickup branch — flips the behavior live), then atlas-channel (opcode handlers + decorator), then atlas-quest + atlas-query-aggregator (requirement support), then atlas-ui. Each step is additive; the only behavior-flipping step is atlas-inventory's pre-insert branch, and that runs only when `consumeOnPickup=true` — currently zero items at runtime, so deployment is safe even before atlas-monster-book is reachable. Cards become consumeOnPickup only via atlas-data WZ data; no existing card data has it set inadvertently (verify during execute).

- **Risk: atlas-inventory branch fires for an item where atlas-monster-book is unreachable.** Mitigation: the produce of `ITEM.CONSUMED_ON_PICKUP` is fire-and-forget; if Kafka is down the player loses the card and the drop is gone. This matches existing pickup semantics for any other cross-service flow (we don't roll back drop pickup on Kafka failures elsewhere either). Acceptable.

- **Risk: WZ data has no `info/mob` for some card item**. atlas-data exposes `monsterId=0` in that case; atlas-ui shows blank mob name. Loud warn-log in atlas-ui telemetry is acceptable v1 behavior.

- **Risk: a future tenant configures a non-card item with `consumeOnPickup=true`**. atlas-consumables' default branch logs and skips, with a Prometheus counter `monster_book_unsupported_pickup_consumable_total`. Not a regression since today the flag is universally ignored.

## 13. Out of scope (for clarity)

- Tenant-configurable EXP-bonus formula and special-card threshold.
- Card scrolls, card-set bonuses, card decks beyond cover.
- "Fill the Book" / Barry NPC scripts.
- HP/MP bonuses or other book-level-driven stats beyond EXP bonus.
- Backfilling existing quest data with `monsterBookCount` requirements.
- Trading, dropping, or destroying owned cards.
