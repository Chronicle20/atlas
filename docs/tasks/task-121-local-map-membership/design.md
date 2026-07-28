# Local Map Membership for Broadcasts (PS-2) — Design

Task: task-121-local-map-membership
Status: Proposed
Created: 2026-07-02
PRD: `docs/tasks/task-121-local-map-membership/prd.md`

---

## 1. Problem Recap

Every in-map broadcast in atlas-channel resolves its recipient set with a synchronous
REST call to atlas-maps (`services/atlas-channel/atlas.com/channel/map/processor.go:31-70`,
via `requests.SliceProvider` and `map/requests.go`). The resolved character ids are only
ever used to address **local** sessions (`session.Processor.ForEachByCharacterId` →
registry lookup; ids without a local session are skipped). Every local session already
carries its authoritative `field.Model`. The REST hop is therefore redundant state
transfer on the hottest path in the game loop.

## 2. Chosen Architecture

**Filter the local session registry by field, inside the `session` package; keep every
`map.Processor` method signature and its internal id→session delivery pipeline intact.**

Data flow after the change:

```
broadcast caller (consumer/handler)
  └─ map.Processor.ForSessionsInMap(f, op)            [signature unchanged]
       └─ CharacterIdsInMapModelProvider(f)           [signature unchanged, new impl]
            └─ session.Processor.InFieldModelProvider(f)     [NEW]
                 └─ registry.GetInTenant(tenantId)    [existing, RLock snapshot]
                      filter: s.CharacterId() != 0 && s.Field().Equals(f)
       └─ session.Processor.ForEachByCharacterId(...) [existing, unchanged]
```

### 2.1 Alternatives considered

| # | Approach | Verdict |
|---|----------|---------|
| A | **Linear scan of the tenant's sessions, filter by field (chosen)** | O(sessions-in-tenant) per broadcast with a trivial per-element comparison. At the design point in the PRD (hundreds to low thousands of sessions), a full scan is a handful of microseconds — noise next to packet encode + socket write, and ~3 orders of magnitude below the REST round-trip being removed. No new mutation logic; no index to drift. |
| B | Field-keyed index in the registry (`map[tenant]map[field][]sessionId`) | O(recipients) lookup, but every `Update` must detect field changes and move sessions between buckets under the write lock. `Update` is called by *every* mutator (`SetAccountId`, `UpdateLastRequest`, ping timestamps…), so the index pays bookkeeping on high-frequency non-field writes and introduces a drift-bug class the scan cannot have. Rejected per PRD Open Question 1 lean: measure first, and the estimate says the scan wins on simplicity at current scale. The provider seam chosen here means an index can be added later purely inside the `session` package, with zero caller churn. |
| C | Event-sourced projection from `MAP_STATUS` | Already rejected at PRD scope time (owner, 2026-07-02): cold-start/offset problems, second source of truth, and the session registry must be field-reliable anyway for gameplay to function. |

### 2.2 Where the filter lives (PRD Open Question 2)

In the `session` package, as new providers on `session.Processor`. The registry's
internals (tenant map, mutex) stay encapsulated; `map.Processor` consumes a
`model.Provider[[]session.Model]` exactly the way it consumes providers today. Decision:
**session package** (matches the PRD's stated preference).

### 2.3 Shadow verification (PRD Open Question 3)

Not carried. FR-4 unit tests + staging playtest per the PRD default. Rationale: the
equivalence argument is structural (see §5), the delivery pipeline downstream of the
provider is byte-identical, and shadow mode would keep the REST plumbing alive that
FR-3.2 requires deleted.

## 3. Component Design

### 3.1 `session` package — new providers

```go
// InFieldModelProvider returns local sessions whose field exactly matches f
// (world, channel, map, instance) and which have an assigned character.
func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model]

// InMapAllInstancesModelProvider returns local sessions on the given
// world/channel/map across all instances, with an assigned character.
func (p *Processor) InMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]Model]
```

Implementation notes:

- Both snapshot via the existing `getRegistry().GetInTenant(p.t.Id())` (RLock, returns
  value copies) and filter the snapshot outside the lock — FR-2.5 / NFR-5 satisfied with
  no new locking.
- Exact match uses `field.Model.Equals` (includes instance —
  `libs/atlas-constants/field/model.go:92`); all-instances match uses per-component
  world/channel/map comparison (equivalent to `field.Model.SameMap`).
- `s.CharacterId() != 0` excludes pre-login / character-select sessions (FR-2.4).
- Both return `model.Provider[[]Model]` closures so they compose with the existing
  `model` pipeline; they cannot fail (the error return exists only to satisfy the
  `Provider` contract).

### 3.2 `map` package — providers re-implemented, signatures frozen

`map/processor.go` keeps every exported method with its exact current signature
(PRD §5). New bodies:

```go
func (p *Processor) CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32] {
    return characterIds(p.sp.InFieldModelProvider(f))
}

func (p *Processor) CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]uint32] {
    return characterIds(p.sp.InMapAllInstancesModelProvider(worldId, channelId, mapId))
}
```

where `characterIds` is a package-private helper that maps `[]session.Model` →
`[]uint32` and **deduplicates** character ids. Dedup preserves today's semantics:
atlas-maps returns each character at most once, whereas the registry could transiently
hold two sessions for one character id (stale socket + reconnect); without dedup the
delivery operator would run twice against the same first-found session.

Unchanged: `GetCharacterIdsInMap`, `ForSessionsInSessionsMap`, `ForSessionsInMap`,
`ForSessionsInMapAllInstances`, `NotCharacterIdFilter`,
`OtherCharacterIdsInMapModelProvider`, `ForOtherSessionsInMap` — they already compose
over the two providers above and need no edits beyond what the provider swap gives them
(FR-2.3). `NotCharacterIdFilter` keeps operating on the id slice via
`model.FilteredProvider`, so filtering semantics are untouched.

**Deliberately retained indirection:** recipient resolution still produces character
*ids*, and `ForEachByCharacterId` still re-resolves each id to a session at delivery
time. This looks like wasted work (we had the sessions in hand) but is kept because
(a) it makes the diff a pure provider swap — every downstream behavior including
`model.ParallelExecute` delivery is provably identical; (b) the delivery-time re-fetch
picks up the freshest session snapshot, exactly as today. The extra O(N) per recipient
lookup is in-memory and irrelevant next to the removed REST call. Collapsing the
pipeline to iterate sessions directly is a possible later simplification, not part of
this task.

### 3.3 Deletions (FR-3.2)

Once the providers are swapped, delete:

- `atlas.com/channel/map/requests.go` (`getBaseRequest`, `requestCharactersInMap`,
  `requestCharactersInMapAllInstances`, resource constants)
- `atlas.com/channel/map/rest.go` (`RestModel`, `Extract`) — used nowhere else
  (verified by grep; only `processor.go` references them)
- the `requests` import from `map/processor.go`

`grep -r requestCharactersInMap` over non-test code must return nothing (acceptance
criterion). atlas-channel's other atlas-maps interactions are untouched (FR-3.3):
`maps/location.GetField` (login bootstrap, different endpoint), map ENTER/EXIT command
emission, and the `MAP_STATUS` consumer.

## 4. FR-1 Session-Field Write-Path Audit

All world/channel/map/instance mutations of a session funnel through exactly three
sites (full-file grep of `atlas.com/channel` for registry field mutators; the audit
document required by FR-1.3 will restate this with final line numbers):

| # | Transition | Site | Registry update | Verdict |
|---|-----------|------|-----------------|---------|
| 1 | Socket connect | `session/processor.go:287` `Create` — `setWorldId` + `setChannelId` before `Add` | world/channel fixed for the session's lifetime at creation | ✅ correct |
| 2 | Login spawn-in | `kafka/consumer/session/consumer.go:190` — `SetMapId(f.MapId())` where `f` comes from `location.GetField` (includes instance, `maps/location/requests.go:78`) | map set, **instance silently dropped** | ❌ **GAP — fix in this task** |
| 3 | Every map/instance change (portal warp, GM warp, revive/forced return, transport arrival, instance enter/exit) | `kafka/consumer/character/consumer.go:249` — `SetField(sessionId, targetField)` from the `MAP_CHANGED` character status event (carries `TargetMapId` + `TargetInstance`), **before** the SetField packet write and `SpawnForSelf` | full field set before dependent broadcasts | ✅ correct |
| — | Channel change | `kafka/consumer/session/consumer.go:126-138` — old session is destroyed; the client reconnects to the target channel pod, which runs path 1 + 2 fresh | no in-place field mutation exists | ✅ correct by construction |

All the PRD's enumerated transition kinds (map change, portal warp, instance enter/exit,
revive, GM warp, transports) are server-driven through the single `MAP_CHANGED` status
event consumed at path 3 — atlas-channel is a projection of atlas-character/atlas-maps
state and has no other field-writing entry point. Ordering is safe: path 3 runs
`SetField` synchronously before announcing the warp packet and spawning, so any
broadcast triggered by the arrival sees the new field.

### 4.1 The path-2 fix (FR-1.2)

Change `kafka/consumer/session/consumer.go:190` from
`s = sp.SetMapId(s.SessionId(), f.MapId())` to
`s = sp.SetField(s.SessionId(), f)`.

Why it matters: a character logging in while located in an **instanced** map currently
gets a session whose field has `instance == uuid.Nil`. Under today's REST resolution
this character is *already* mishandled in one direction (queries built from
`s.Field()` ask atlas-maps for the Nil-instance bucket), but broadcasts addressed by
event-derived fields (which carry the real instance) still reach them because
atlas-maps knows their true instance. Under local resolution, exact-field matching
would exclude them from those broadcasts — the gap would widen from a self-query quirk
into a delivery regression. Setting the full field at bootstrap closes both. This is
precisely the audit outcome FR-1 anticipated.

## 5. Caller Audit (FR-3.1) and Equivalence Argument

32 non-test files call the recipient providers (enumerated by grep; the FR-1.3 audit
doc will carry the full file list). They fall into three usage categories:

1. **Packet fan-out** (the overwhelming majority): consumers under `kafka/consumer/*`
   (monster, map, summon, pet, guild, mist, drop, reactor, mount, message, merchant,
   door, chalkboard, chair, buff, quest, party_quest, monsterbook, expression,
   consumable, asset, route), `movement/processor.go`, `skill/handler/*`, and the
   `socket/handler/character_*` handlers. Result feeds `ForEachByCharacterId` →
   `session.Announce`. Local-session semantics are sufficient by definition: delivery
   is only possible to local sessions.
2. **Membership enumeration for spawn logic**: `kafka/consumer/map/consumer.go:127,359`
   (`fetchOtherCharactersInMap`, `enterMap`) — ids are used to fetch character models
   and then, again, to address local sessions. Sufficient: a character standing in a
   map on channel *c* necessarily holds its socket session on the pod serving channel
   *c*; there is no cross-pod population of the same (world, channel, map, instance).
3. **Side-effecting iteration**: `kafka/consumer/map/consumer.go:726` (weather /
   consumable-effect saga applies a saga per character in map). Same sufficiency
   argument as category 2 — the map's population *is* this pod's session set for that
   field. The saga targets characters, not sessions, but the membership universe is
   identical.

No caller requires atlas-maps' authoritative view; no named REST escape hatch is
needed (the PRD expected none).

**Semantic deltas, all favorable (NFR-3):**

- *Freshness*: the registry is updated synchronously in the warp path, whereas
  atlas-maps' view is an async Kafka projection. A character mid-transition is seen at
  its true field immediately instead of after the Enter command lands. Strictly less
  stale.
- *Stale-id abort eliminated*: today, if atlas-maps returns a character whose session
  just vanished, `SliceMap` inside `ForEachByCharacterId` aborts the **entire**
  broadcast on the first `not found` (`libs/atlas-model/model/processor.go:419` —
  any transformer error fails the whole mapping). Locally-resolved ids always have a
  backing session at resolution time, shrinking that failure window to the
  resolution→delivery gap instead of the Kafka-lag window.
- *REST failure mode gone*: recipient resolution can no longer fail with a network
  error; an atlas-maps or nginx outage stops affecting fan-out (NFR-2).

## 6. Multi-Tenancy & Concurrency

- Tenant scoping is inherited from `session.Processor` (`tenant.MustFromContext` at
  construction; `GetInTenant(p.t.Id())`) — NFR-4 holds with no new code.
- No new goroutines, no new locks. The only shared-state access is the existing
  `GetInTenant` RLock snapshot. `go test -race` gates it (NFR-5).

## 7. Testing (FR-4)

All tests use the existing Builder-style setup: `session.NewSession` (with a fake
`net.Conn`) shaped through `session.Processor` mutators (`SetCharacterId`,
`SetField`), registered via the exported `AddSessionToRegistry` /
`ClearRegistryForTenant` test helpers (`session/registry_test_helper.go`). No new
`*_testhelpers.go` constructors.

1. **`session` package** — `InFieldModelProvider` / `InMapAllInstancesModelProvider`:
   - exact-field matching, including instance discrimination (same map, different
     instance UUIDs → disjoint sets);
   - all-instances matching unions the instance buckets;
   - sessions with `characterId == 0` excluded;
   - other-tenant sessions excluded;
   - empty result (no error) for an unpopulated field.
2. **`map` package** — provider composition:
   - `OtherCharacterIdsInMapModelProvider` / `NotCharacterIdFilter` excludes the
     reference character;
   - id dedup when two sessions share a character id;
   - `GetCharacterIdsInMap` returns the local set with no HTTP server running
     (regression proof that REST is gone).
3. **Transition correctness (FR-4.2)** — build sessions A-in-map-1 and B-in-map-1,
   assert both resolve for map 1; apply `SetField` moving B to map 2; assert map 1
   resolves exactly {A} and map 2 exactly {B} — no state in which B is in both or
   neither. Since `Update` replaces the single registry entry under one write lock,
   the invariant is structural, and the test pins it.
4. **Login-bootstrap instance fix** — unit-level: after the path-2 fix, a bootstrap
   field with a non-Nil instance yields a session whose `Field().Equals(f)` is true
   (guards against regression to `SetMapId`).

Existing suites (`kafka/consumer/{door,mist,monster,mount}/consumer_test.go`, movement,
skill handlers) must pass unchanged — they compile against frozen signatures.

## 8. Verification & Rollout

- `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-channel;
  `docker buildx bake atlas-channel`; `tools/redis-key-guard.sh` (per CLAUDE.md).
- FR-1.3 audit document committed as
  `docs/tasks/task-121-local-map-membership/field-transition-audit.md` (the §4 table
  with final file:line references, produced during implementation).
- Playtest per PRD acceptance: two characters co-located see each other's
  movement/chat/emotes; third character in another instance/map excluded; warp updates
  visibility immediately; atlas-maps access logs show no character-enumeration calls
  from atlas-channel.

## 9. Risks

| Risk | Mitigation |
|------|-----------|
| An unenumerated field-transition path exists (audit miss) | §4 grep is exhaustive over registry mutators (`SetField`/`SetMapId`/`setWorldId`/`setChannelId`/`setInstance` call sites); FR-1.3 doc re-verified at review; transition test + playtest catch drift symptomatically. |
| Duplicate sessions per character double-deliver | id dedup in `characterIds` (§3.2). |
| Scan cost surprises at higher session counts | provider seam confines any future field-index to the `session` package (§2.1-B); no caller churn to adopt it. |
| Hidden consumer of `map` REST artifacts | deletion compiles the whole service; grep acceptance criterion. |
