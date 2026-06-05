# Monster-Book Cover — Encode Mob ID in Character-Info (Crash Fix) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-05
---

## 1. Overview

Setting a monster-book cover crashes the v83 client whenever the player (or
anyone inspecting them) opens the Character Info window. The monster-book
plumbing is otherwise correct: the client's set-cover request (recv `0x39`)
sends a **card item id** (e.g. `2380000`), atlas-channel forwards it, and
atlas-monster-book stores it as `coverCardId`. That card-item-id value is the
*correct* id-space for the set-cover response (`0x54`), the monster-book
window, and the card list — all of which the client resolves card→mob through
its own loaded card data.

The Character Info packet is different. The v83 client treats the cover field
there as a **mob id**, not a card id. Verified in IDA
(`MapleStory_dump.exe`, v83): `CWvsContext::OnCharacterInfo` (`0xa2370b`) →
decoder `sub_684798` reads the monster-book block's 5th int (cover) and calls
`CMobTemplate::GetMobTemplate(cover)` **directly**, then dereferences the
returned template (`sub_685EB1` → indirect virtual call). A card item id
(`2380000`) is not a valid mob id, so the lookup yields an invalid/absent
template and the client crashes. `cover == 0` is guarded (`if (v3)`), which is
why the field was harmless until covers could actually be set.

By contrast, the monster-book *window* render (`CUIMonsterBook::UpdateUI` →
`sub_866C2C`) also calls `GetMobTemplate`, but on the mob id carried by the
card *object* it looked up from the cover card id — so the window is correct
with a stored card id. The fix must therefore change only the value sent in
the **Character Info** packet (and the analogous login-draw field, pending
verification) to be the cover card's mob id — without disturbing the card-id
paths.

This bug shipped latent in PR #659 ("display monster book cover + owned cards
at login"); it became reachable once the set-cover opcode was wired into the
live tenant config. Full root-cause evidence is captured in memory:
`bug_monsterbook_cover_charinfo_is_mobid.md`.

## 2. Goals

Primary goals:
- The Character Info window no longer crashes the v83 client when the
  inspected character has a monster-book cover set.
- The cover shown in Character Info resolves to the correct monster (the mob
  the cover card represents).
- The login-draw `CharacterData` cover field is verified against the client
  and corrected if it shares the same card-id-vs-mob-id defect.
- All other cover surfaces (set-cover request `0x39`, response `0x54`,
  monster-book window, card list) remain unchanged and correct.

Non-goals:
- Changing how the cover is *selected* or what the client *sends* in `0x39`
  (it correctly sends a card item id).
- Changing the monster-book window / card-list encoding (already correct in
  card-id space).
- The socket-opcode config gap (already fixed live for the affected tenant;
  auditing other tenants/versions for missing monster-book opcodes is a
  separate, later task).
- Re-architecting monster-book storage beyond adding the resolved mob id.

## 3. User Stories

- As a player, I can set a monster card as my monster-book cover and then open
  my Character Info (or have another player inspect me) without the client
  crashing.
- As a player inspecting another character, I see their chosen cover monster
  rendered correctly in the Character Info window.
- As a player who has not set a cover, Character Info continues to work exactly
  as before (no cover, no crash).
- As an operator, when a cover card cannot be resolved to a mob id, the system
  fails safe (no crash) and logs a warning so the gap is visible.

## 4. Functional Requirements

### 4.1 Cover mob-id resolution (chosen approach: resolve at set time)
- FR-1: When a cover is set (both entry points that reach
  `collection.SetCoverAndEmit`: the Kafka `SET_COVER` command and the REST
  `PATCH /characters/{id}/monster-book`), atlas-monster-book resolves the
  cover **card item id → mob id** and persists the mob id alongside the
  existing `coverCardId`.
- FR-2: Resolution uses atlas-data: `GET /api/data/consumables/{cardItemId}`
  returns `{ monsterBook: bool, monsterId: uint32 }` (`monsterId` is parsed
  from WZ `info/mob`, `consumable/reader.go:77`). The resolved mob id is
  `monsterId`.
- FR-3: `coverCardId == 0` (clear cover) stores mob id `0` and performs no
  lookup.
- FR-4: If resolution fails (atlas-data error, card not found, `monsterBook`
  false, or `monsterId == 0`), store mob id `0` and **log a warning**
  identifying the character and card id. The cover card id may still be stored
  (it remains valid for the window/`0x54`), but the mob id used by Character
  Info is `0` (safe no-op in the client).
- FR-5: Setting the cover must not fail/blow up if atlas-data is unavailable —
  ownership validation still applies, but a resolution failure degrades to mob
  id `0` + warning rather than rejecting the set.

### 4.2 Exposing the resolved mob id
- FR-6: The resolved cover mob id is exposed to atlas-channel via the
  monster-book REST model (`GET /characters/{id}/monster-book`) so the
  Character Info and login-draw encoders can read it without a per-view
  atlas-data lookup.
- FR-7: The `COVER_CHANGED` status event continues to carry the cover **card
  id** (used by the `0x54` set-cover response / window). Whether it also needs
  to carry the mob id depends on the encode paths in 4.3 (design decides).

### 4.3 Packet encoding (atlas-channel)
- FR-8: The Character Info packet
  (`libs/atlas-packet/character/clientbound/info.go` cover field, populated by
  `services/atlas-channel/.../socket/writer/character_info.go`) writes the
  cover **mob id** (or `0`), not the card item id.
- FR-9: The set-cover response (`0x54`,
  `clientbound/monsterbook/set_cover.go`) continues to write the cover **card
  id** (unchanged — correct for the window).
- FR-10: Login-draw `CharacterData` cover field
  (`libs/atlas-packet/character/data.go:701`): verify against the v83 client
  how the login monster-book block's cover is consumed (decompile the login
  `CharacterData` monster-book decoder). If consumed as a mob id, change it to
  emit the mob id; if consumed as a card id (window-rendered), leave it.
  Document the finding.
- FR-11: The monster-book field type stays a 32-bit int on the wire; only the
  *semantic value* changes for the affected packet(s).

### 4.4 Behavior parity
- FR-12: Card list (`0x53` / `set_card.go`) and any per-card encoding remain
  card-id space (unchanged).
- FR-13: The fix is gated to the same version/region conditions that already
  gate the monster-book block (`GMS <= 87 || JMS`), matching existing
  `info.go` / `data.go` gating.

## 5. API Surface

### Modified — atlas-monster-book REST model
- `GET /characters/{id}/monster-book` response attributes gain a resolved
  cover mob id field (e.g. `coverMonsterId`), in addition to existing
  `coverCardId`. JSON:API resource type `monster-book`, tenant-scoped (tenant
  headers required, as today).
- `PATCH /characters/{id}/monster-book` request is unchanged (still sends
  `coverCardId`); the response reflects the newly resolved `coverMonsterId`.

### Consumed — atlas-data (existing, no change)
- `GET /api/data/consumables/{cardItemId}` → `{ monsterBook, monsterId, ... }`.
  Tenant-scoped (`TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION`
  headers).

### Kafka
- `COVER_CHANGED` status event: unchanged for `coverCardId`; may add the mob id
  if a downstream encoder needs it (design decision per FR-7/FR-10).

## 6. Data Model

- atlas-monster-book `monster_book_collections`: add `cover_mob_id` (uint32,
  not null, default 0) alongside existing `cover_card_id`. GORM migration.
- Backfill: existing rows with a non-zero `cover_card_id` set before this
  change have `cover_mob_id = 0`. Options (design decides):
  - Lazy: resolve on next set; until then Character Info shows no cover (safe).
  - One-time backfill job/migration that resolves existing covers via
    atlas-data.
  - Given the live tenant's only affected cover was already cleared to `0`,
    lazy backfill is likely sufficient.
- All fields tenant-scoped via existing `tenant_id` column.

## 7. Service Impact

- **atlas-monster-book**:
  - New outbound dependency on atlas-data (consumables lookup). The service
    currently has **no** outbound REST client — add a `requests`/provider
    using the existing `atlas-rest` + `atlas-tenant` libs and a `DATA` root-url
    env var; wire it into the deployment config.
  - `collection.SetCoverAndEmit`: resolve + persist mob id (FR-1..FR-5).
  - New `cover_mob_id` column + migration; expose in REST model + transform.
- **atlas-channel**:
  - Monster-book model/rest gains the cover mob id; Character Info writer emits
    it (FR-8). Login-draw writer updated only if FR-10 verification requires.
  - `0x54` / card-list paths unchanged.
- **libs/atlas-packet**:
  - `clientbound/info.go` cover field semantics (value now mob id); possibly
    `character/data.go` login-draw cover (pending FR-10). Update affected
    tests.
- **atlas-data**: no change (read-only consumer of the existing endpoint).

## 8. Non-Functional Requirements

- **Reliability / fail-safe**: a resolution failure must never produce a value
  that can crash the client; it degrades to `0` + warning (FR-4).
- **Performance**: card→mob resolution happens only on cover *set* (rare), not
  on every Character Info view; Character Info encoding stays a local read.
- **Observability**: warning log on resolution failure with character id + card
  id (FR-4).
- **Multi-tenancy**: atlas-data lookup is tenant-scoped (correct region/version
  headers from context); resolved mob id stored per tenant row.
- **Verification**: per CLAUDE.md, packet-encoding correctness must be verified
  against the v83 client / WZ source, not memory. The login-draw decision
  (FR-10) requires a client decompile; the char-info crash is already
  IDA-verified.

## 9. Open Questions

- OQ-1 (FR-10): Does the login-draw `CharacterData` cover field need mob id or
  card id? Resolve by decompiling the v83 client's login `CharacterData`
  monster-book decoder during design. (Decision: in scope for this task.)
- OQ-2 (Data Model): Lazy vs. one-time backfill for pre-existing covers.
  Recommended: lazy (the only live affected cover is already cleared).
- OQ-3 (FR-7): Does any encoder consume the mob id off the `COVER_CHANGED`
  event (vs. the REST model), requiring the event to carry it?
- OQ-4 (Architecture): Confirm acceptance of atlas-monster-book taking an
  outbound atlas-data dependency. Alternative is encode-time resolution in
  atlas-channel (no monster-book schema/dependency change, but a lookup per
  Character Info view). Chosen direction is set-time resolution in
  atlas-monster-book per stakeholder preference; design should record the
  trade-off explicitly.

## 10. Acceptance Criteria

- [ ] With a cover set, opening Character Info on the cover's owner (self and
      remote inspect) does not crash the v83 client; the correct cover monster
      renders.
- [ ] With no cover (`coverCardId == 0`), Character Info behaves exactly as
      before.
- [ ] The Character Info packet's cover field carries the cover card's mob id
      (or `0`), verified by byte-level/encoder test and against the v83 client
      decode (`GetMobTemplate` receives a valid mob id or `0`).
- [ ] Setting a cover resolves and persists `cover_mob_id` via atlas-data;
      `coverCardId` storage is unchanged.
- [ ] A cover card that cannot be resolved stores mob id `0` and logs a
      warning; the set still succeeds and the client does not crash.
- [ ] The set-cover response (`0x54`), monster-book window, and card list are
      unchanged and still correct (card-id space).
- [ ] Login-draw `CharacterData` cover field decision (FR-10) is documented and
      implemented accordingly (mob id or card id), verified against the client.
- [ ] atlas-monster-book schema migration adds `cover_mob_id`; REST model
      exposes it; existing covers are handled per the chosen backfill approach.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every
      changed module; `docker buildx bake` for every service whose `go.mod`
      changed; `tools/redis-key-guard.sh` clean.
