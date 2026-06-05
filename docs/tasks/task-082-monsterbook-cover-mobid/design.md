# Monster-Book Cover — Encode Mob ID in Character-Info (Crash Fix) — Design

Task: task-082-monsterbook-cover-mobid
Status: Approved
Created: 2026-06-05
Inputs: `prd.md` (approved), `bug_monsterbook_cover_charinfo_is_mobid.md` (root cause)

---

## 1. Problem Recap

The v83 client's `CWvsContext::OnCharacterInfo` decoder reads the monster-book
block's 5th int (cover) and calls `CMobTemplate::GetMobTemplate(cover)`
**directly**, then dereferences the result. atlas-channel currently writes the
stored **cover card item id** (e.g. `2380000`) into that field. A card item id
is not a valid mob id, so the lookup yields an invalid template and the client
crashes. `cover == 0` is guarded client-side (`if (v3)`), which is why the field
was harmless until covers could actually be set (PR #659, reachable once the
`0x39` set-cover opcode was wired into the live tenant config).

The fix: the **Character Info** packet must carry the cover card's **mob id**
(or `0`), while every other cover surface — the `0x39` set request, the `0x54`
set response, the monster-book window, the card list, and the login-draw block
— stays in **card-id** space (those paths resolve card→mob client-side and are
already correct).

## 2. Chosen Architecture

**Set-time resolution in atlas-monster-book** (PRD OQ-4, confirmed in design).

When a cover is set, atlas-monster-book resolves the cover **card item id → mob
id** via a new outbound call to atlas-data, and persists the resolved mob id
(`cover_mob_id`) alongside the existing `cover_card_id`. The resolved mob id is
exposed on the existing `GET /characters/{id}/monster-book` REST model.
atlas-channel reads it from that response — which the Character-Info handler
**already fetches per view** via `MonsterBookDecorator` — and writes it into the
Character-Info packet's cover field. No new per-view round-trip is introduced on
the hot path.

```
                          ┌──────────────────────── atlas-data ───────────────┐
                          │ GET /api/data/consumables/{cardId}                 │
                          │   → { monsterBook: bool, monsterId: uint32, ... }  │
                          └────────────────────────────▲───────────────────────┘
                                                        │ (set-time, rare)
 set-cover entry points                                 │
 ─────────────────────                                  │
  kafka SET_COVER  ─┐                                    │
                    ├─► collection.SetCoverAndEmit ──────┘ resolve card→mob
  REST PATCH       ─┘        │  persist cover_card_id + cover_mob_id
                             │  emit COVER_CHANGED (card id, unchanged)
                             ▼
                   monster_book_collections row
                   (cover_card_id, cover_mob_id)
                             │
            GET /characters/{id}/monster-book  (adds coverMonsterId)
                             │ (per-view, already happening)
                             ▼
   atlas-channel character_info.go ──► MonsterBookInfo.Cover = mob id  ──► v83 client
                                                                          GetMobTemplate(mobId|0) ✓
```

### 2.1 Alternatives considered & rejected

- **B2 — encode-time resolution in atlas-channel.** Channel resolves
  `coverCardId → mobId` when building the Character-Info packet; monster-book is
  untouched (no column, no migration, no REST field). *Rejected:* atlas-channel
  has **no** atlas-data item/consumable client today and caches nothing for item
  data, so this requires a brand-new client **and** a new cache (mirroring the
  map cache) to avoid an atlas-data lookup on every inspect of a covered
  character. The dependency is not avoided — only relocated to a service that
  also lacks caching — and it adds more moving parts, not fewer. Its only edge
  (frozen monster-book schema) does not outweigh the new client+cache.
- **B1 — resolve in channel's `0x39` set handler, carry mob id in the
  `SET_COVER` command body.** *Rejected (strict loser):* still needs a new
  atlas-data client in channel, **and** the REST `PATCH` entry point bypasses
  channel entirely, so PATCH-set covers would never be resolved (mob id stays
  `0`, Character Info silently shows no cover). Closing that hole would require
  the resolver in monster-book too — both services then carry the dependency.

**Why A wins:** the dependency is unavoidable wherever it lands (monster-book
cannot resolve from its own data — it stores cards by card id, not mob id);
monster-book is the authoritative owner of the cover; set-time resolution runs
once per (rare) cover *set* rather than per (frequent) inspect; and the
Character-Info hot path already fetches the monster-book model, so A adds **zero**
extra round-trips where latency matters.

## 3. Component Design

### 3.1 atlas-monster-book — new `data/consumable` outbound client

New package `services/atlas-monster-book/.../data/consumable/`, mirroring the
canonical `services/atlas-npc-shops/.../data/consumable/` pattern:

- `requests.go`:
  - `Resource = "data/consumables"`, `ById = Resource + "/%d"`
  - `getBaseRequest() = requests.RootUrl("DATA")` (falls back to
    `BASE_SERVICE_URL` via ingress when `DATA_SERVICE_URL` is unset)
  - `requestById(id uint32) requests.Request[RestModel]`
- `rest.go`:
  - `RestModel` with at minimum `Id uint32`, `MonsterBook bool
    json:"monsterBook"`, `MonsterId uint32 json:"monsterId"` (the fields the
    resolver needs; other consumable fields omitted — only what we read).
  - JSON:API plumbing: `GetName() = "consumables"`, `GetID`/`SetID`, and the
    **`SetToOneReferenceID` / `SetToManyReferenceIDs` no-op stubs** to avoid the
    api2go "does not implement UnmarshalToManyRelations" trap (surfaces as a
    generic not-found otherwise).
  - `Extract(RestModel) (Model, error)` → internal `Model` exposing
    `MonsterBook() bool` and `MonsterId() uint32`.
- `processor.go`:
  - `Processor` interface + `ProcessorImpl{l, ctx}`, `NewProcessor(l, ctx)`.
  - `GetById(itemId uint32) (Model, error)` →
    `requests.Provider[RestModel, Model](l, ctx)(requestById(itemId), Extract)()`.
  - Tenant/region/version headers are propagated automatically by
    `requests.GetRequest` (`TenantHeaderDecorator`), so the lookup is correctly
    tenant/version-scoped — no manual header wiring.

### 3.2 `collection` — resolve + persist mob id

**Resolver wiring.** `ProcessorImpl` gains a consumable processor field, built
in `NewProcessor` and carried through `WithTransaction` (the consumable client
is stateless w.r.t. the DB tx, so `WithTransaction` just copies the same
instance — it makes no DB calls).

```go
type ProcessorImpl struct {
    l    logrus.FieldLogger
    ctx  context.Context
    db   *gorm.DB
    t    tenant.Model
    cp   card.Processor
    dp   consumable.Processor   // new: atlas-data consumable lookup
}
```

**`resolveCoverMobId`** — a small private helper implementing FR-2..FR-5:

```go
// resolveCoverMobId resolves a cover card item id to its mob id via atlas-data.
// cardId == 0 returns 0 with no lookup. Any failure (atlas-data error, card not
// found, monsterBook == false, or monsterId == 0) returns 0 and logs a warning;
// it never returns an error, so a resolution failure can never reject the set
// or produce a client-crashing value (FR-4, FR-5, NFR fail-safe).
func (p *ProcessorImpl) resolveCoverMobId(characterId character.Id, cardId item.Id) uint32 {
    if cardId == 0 {
        return 0
    }
    m, err := p.dp.GetById(uint32(cardId))
    if err != nil || !m.MonsterBook() || m.MonsterId() == 0 {
        p.l.WithError(err).Warnf("Unable to resolve monster-book cover card [%d] to a mob id for character [%d]; storing cover mob id 0.", cardId, characterId)
        return 0
    }
    return m.MonsterId()
}
```

**`SetCoverAndEmit`** keeps its existing ownership validation (unchanged:
`cardId == 0` clears; otherwise `card.IsCardId` + ownership at level ≥ 1).
Resolution happens **after** validation passes and **before/within** the emit
transaction. The resolved mob id is threaded into `setCover`:

- `setCover(...)` (administrator.go) gains a `coverMobId uint32` parameter and
  adds `"cover_mob_id": coverMobId` to its `Updates(map[...])`. The event-id
  idempotency guard (`last_cover_event_id`) is unchanged, so a duplicate
  `eventId` still no-ops and does not re-resolve a stale value.

**Idempotency note.** Resolution is a pure read of static WZ data; resolving the
same card id twice yields the same mob id. The `last_cover_event_id` guard still
prevents duplicate *writes*. Worst case on a replayed command: one redundant
atlas-data read whose result is discarded because `setCover` reports
`changed == false`. Acceptable.

### 3.3 Schema, entity, model

- `entity.go`: add `CoverMobId uint32 gorm:"not null;default:0"`. GORM
  `AutoMigrate` adds the column with default `0` (no manual migration).
- `model.go`: add `coverMobId uint32` field + `CoverMobId() uint32` getter; map
  it in `ToEntity`.
- `builder.go`: add `coverMobId` field, `SetCoverMobId(uint32)`, thread through
  `CloneModelBuilder`, `Build`, and `Make` (entity → Model).

### 3.4 REST exposure (FR-6)

- `collection/rest.go`: `RestModel` gains `CoverMonsterId uint32
  json:"coverMonsterId"`; `Transform` sets it from `m.CoverMobId()`.
- `PatchInput` unchanged (still only `coverCardId`). The PATCH handler reloads
  and returns the model after `SetCoverAndEmit`, so the response reflects the
  freshly resolved `coverMonsterId` (already the handler's behavior).

### 3.5 atlas-channel — write mob id into Character-Info

- `monsterbook/rest.go`: `CollectionRestModel` gains `CoverMonsterId uint32
  json:"coverMonsterId"`; `Extract` maps it into the `Collection` domain model
  (`coverMonsterId` field + `CoverMonsterId() uint32` getter on `Collection`).
- `socket/writer/character_info.go`: set
  `MonsterBookInfo.Cover = mb.CoverMonsterId()` instead of
  `uint32(mb.CoverCardId())`. When no cover is set the field resolves to `0`
  (safe client no-op). Version/region gating in `info.go`
  (`GMS <= 87 || JMS`) is unchanged.
- `libs/atlas-packet/character/clientbound/info.go`: **no structural change** —
  `MonsterBookInfo.Cover` stays `uint32` / `WriteInt`. Only the *semantic value*
  supplied by the writer changes (FR-11).

## 4. Resolved Open Questions

- **OQ-3 / FR-7 — `COVER_CHANGED` stays card-id only; no kafka change.** The
  only channel consumer of `COVER_CHANGED` is `handleCoverChanged`, which sends
  the `0x54 SetCover` packet using the **card id** (correct for the monster-book
  window, which resolves card→mob client-side). No encoder consumes a mob id off
  the event, so the event body and `CoverChangedBody` are unchanged.
- **OQ-1 / FR-10 — login-draw `CharacterData` cover stays the card id; no change
  to `data.go`.** Decision and evidence:
  1. The live crash manifested **only** when opening Character Info — never on
     map-entry/login despite the cover being set. If the login `CharacterData`
     decoder called `GetMobTemplate(cover)`, setting a cover would have crashed
     at login too. It did not, which is strong behavioral evidence the login
     cover is **not** consumed as a mob id.
  2. The monster-book *window* (fed by the login block) resolves card→mob
     itself from the cover card object — consistent with a card id on the wire.
  So `encodeMonsterBook` in `data.go` (cover = full card item id, flag
  `0x20000`) is left unchanged, and `character_data.go` continues to emit
  `CoverCardId()`.
  **Verification gate (execution):** confirm via IDA that the login
  `CharacterData` monster-book decoder does **not** call `GetMobTemplate` on the
  cover field. The default position is the no-change one, so if IDA is
  unavailable the safe path is to ship the Character-Info fix and leave
  login-draw untouched; only if the client proves it consumes a mob id does
  `data.go`/`character_data.go` change (and then it would mirror §3.5). Document
  the finding in the task.
- **OQ-2 — lazy backfill.** Existing rows keep `cover_mob_id = 0` (safe: renders
  as no cover, never crashes) until the cover is next set. The one live affected
  cover is already cleared. No backfill job/migration.
- **OQ-4 — set-time resolution in monster-book** (this design's §2).

## 5. Data Flow Summary

1. **Set (kafka `SET_COVER` or REST `PATCH`)** → `SetCoverAndEmit`:
   validate ownership → `resolveCoverMobId(cardId)` (atlas-data, fail-safe → 0
   + warn) → `setCover` persists `cover_card_id` + `cover_mob_id` under the
   `last_cover_event_id` guard → on change, emit `COVER_CHANGED` (card id).
2. **`COVER_CHANGED` consumed by channel** → `0x54 SetCover` to owner with the
   **card id** (unchanged; window correctness).
3. **Character-Info request** → `MonsterBookDecorator` fetches the monster-book
   model (now including `coverMonsterId`) → `character_info.go` writes the
   **mob id** into the cover field → client `GetMobTemplate(mobId | 0)` succeeds.
4. **Login-draw / SetField** → `BuildCharacterData` writes the **card id**
   (unchanged, per FR-10).

## 6. Error Handling & Fail-Safe

- atlas-data unavailable / card not found / `monsterBook == false` /
  `monsterId == 0` → `cover_mob_id = 0` + **warning** (character id + card id).
  The set still succeeds; `cover_card_id` is still stored (valid for window /
  `0x54`). Character Info shows no cover — never a crash (FR-4, FR-5).
- `MonsterBookDecorator` already fails open on REST errors (renders an empty
  book), so a monster-book outage degrades Character Info to no-cover, not a
  crash.
- Resolution never returns an error to `SetCoverAndEmit`, so it cannot convert a
  resolvable cover into a rejected set.

## 7. Testing Strategy

- **`libs/atlas-packet` `info_test.go`** — byte-level/round-trip assertion that
  `MonsterBookInfo.Cover` carries the supplied value (the writer now supplies a
  mob id); existing gating tests unchanged. (`info.go` itself is unchanged, so
  this is mostly guarding the contract the writer relies on.)
- **`libs/atlas-packet` `data_test.go`** — unchanged; login-draw still emits the
  card id (regression guard that FR-10's no-change decision holds).
- **monster-book `collection` processor tests** — table-driven over the new
  `data/consumable` processor (faked): (a) successful resolve persists the mob
  id and exposes it via `Transform`; (b) `cardId == 0` stores `0`, no lookup;
  (c) resolve failure / `monsterBook == false` / `monsterId == 0` → stores `0`
  + warning, set still succeeds, `cover_card_id` still written; (d) duplicate
  `eventId` still no-ops. Use the project Builder pattern for setup — no
  `*_testhelpers.go`.
- **monster-book `data/consumable`** — `Extract` mapping + JSON:API unmarshal of
  a representative atlas-data response (incl. a `relationships` block to prove
  the stubs work).
- **atlas-channel** — `monsterbook.Extract` maps `coverMonsterId`;
  `character_info.go` writer sets `Cover` from `CoverMonsterId()`;
  `character_data.go` / `BuildCharacterData` regression that login-draw cover is
  still the card id.

## 8. Verification Checklist (per CLAUDE.md)

- [ ] `go test -race ./...` clean in every changed module
  (atlas-monster-book, atlas-channel, libs/atlas-packet).
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean.
- [ ] `docker buildx bake atlas-monster-book` — its `go.mod` gains the
  atlas-rest client; the shared Dockerfile must already `COPY` the needed libs
  (no new shared lib is introduced, so likely no Dockerfile edit — confirm).
  `docker buildx bake atlas-channel` if its `go.mod` is touched (it is not
  expected to be — only Go source changes there).
- [ ] `tools/redis-key-guard.sh` clean (no new redis usage introduced).
- [ ] IDA confirmation of the FR-10 login-draw decision recorded in the task.

## 9. Deployment / Config

- Add `DATA_SERVICE_URL` to atlas-monster-book's deploy config for explicitness
  (`deploy/k8s/base/atlas-monster-book.yaml` or the shared `atlas-env`
  configmap). Functionally optional — `requests.RootUrl("DATA")` falls back to
  `BASE_SERVICE_URL` (ingress) — but declaring it documents the new dependency
  and allows direct routing.

## 10. Out of Scope (per PRD non-goals)

- Changing what the client sends in `0x39` (correctly a card item id).
- Monster-book window / card-list / `0x54` encoding (already correct, card-id
  space).
- The socket-opcode config gap (separate task; already fixed live).
- Re-architecting monster-book storage beyond adding `cover_mob_id`.
