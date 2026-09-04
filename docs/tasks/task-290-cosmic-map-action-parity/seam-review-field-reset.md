# Seam review: G5 "reset the field" (clear_drops, reset_reactors, shuffle_reactors, reset_field) — shard 2/3

Range: `9613e7259..d058fd34a`. Scope: the four G5 producer/consumer seams (task C16-C19)
only. Style/guideline and plan-adherence checklists are out of scope — those already ran
clean elsewhere on this branch.

## C16 — `clear_drops`: atlas-map-actions → saga-orchestrator → atlas-drops

**Producer (executor):** `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:419` builds
`saga.ClearDropsPayload{CharacterId, WorldId, ChannelId, MapId, Instance}` for a one-step saga
(action `saga.ClearDrops`).

**Saga handler:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2010`
(`handleClearDrops`) rebuilds `field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).SetInstance(payload.Instance).Build()`
and calls `h.dropsP.ClearDrops(f)`.

**Orchestrator client:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/drops/requests.go:21`
issues `DELETE {root}worlds/%d/channels/%d/maps/%d/instances/%s/drops` — world/channel/map/instance all
taken from `f`.

**Consumer (atlas-drops):**
`services/atlas-drops/atlas.com/drops/map/resource.go:28` registers the same path
(`/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/drops`) as `DELETE`, calling
`drop.Processor.ClearForField` (`services/atlas-drops/atlas.com/drops/drop/processor.go:318`), which
iterates `GetForMap(f)` (field/instance-scoped registry lookup) and removes each drop through `Expire`,
so the per-drop `EnvEventTopicDropStatus` removal event that atlas-channel already consumes still fires —
no consumer-side change needed and none was made.

**Scoping:** identical on both sides (world, channel, map, **and instance**) — no under/over-scoping found.

**Test asserting the new contract:** `services/atlas-drops/atlas.com/drops/drop/processor_test.go:1009`
(`TestClearForField`, four sub-tests: whole-map removal, instance scoping, empty-field no-op, per-drop
emission) and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/drops/drops_test.go:24/69`
(right-field DELETE, upstream-failure propagation) and
`services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go:1242` (`TestExecuteClearDrops`).
All three hops are covered by a test that fails without the change (each asserts on the new
method/route, not a coincidental pass).

**Verdict: PASS.**

## C17 — `reset_reactors` / `shuffle_reactors`: saga-orchestrator → atlas-reactors

**Producer:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/requests.go:30-48`
posts to `.../reactors/reset` with body `ResetReactorsInputRestModel{MinState *int8}` (json tag
`minState,omitempty`) and to `.../reactors/shuffle` with an empty `ShuffleReactorsInputRestModel{}`, both
via `requests.PostRequest`, which marshals the body with `jsonapi.Marshal` (confirmed at
`libs/atlas-rest/requests/post.go:26`, not a raw JSON post).

**Consumer:** `services/atlas-reactors/atlas.com/reactors/reactor/resource.go:34-35` registers both
routes with `rest.RegisterInputHandler[ResetInputRestModel]` / `rest.RegisterInputHandler[ShuffleInputRestModel]`,
which decodes the body with `jsonapi.Unmarshal` (`libs/atlas-rest/server/context.go:51`).

I did not inherit the prior reviewer's claim that this round-trips — I verified it directly. I wrote and
ran (then removed; tree is clean, confirmed with `git status --short`) a probe test inside
`services/atlas-reactors/atlas.com/reactors/reactor` that does `jsonapi.Marshal(ResetInputRestModel{MinState: ptr(7)})`
→ `jsonapi.Unmarshal(..., &ResetInputRestModel{})` and the reverse for the shuffle model. Both round-tripped:
`{"data":{"type":"reactors","id":"","attributes":{"minState":7}}}` decoded back to `MinState == 7`, and the
empty shuffle body decoded cleanly. The two independently-defined structs (orchestrator's
`reactor/rest.go:38-49` and reactors' `reactor/rest.go:49-61`) have identical field names, identical
`json:"minState,omitempty"` tags, and the same `GetName() == "reactors"` — the wire contract agrees.

`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor_test.go:134`
(`TestResetReactorsCarriesMinState`) independently confirms the same thing over real HTTP: it decodes
`r.Body` as a JSON:API document and asserts `data.attributes.minState` is present/absent per case.

**Processor (atlas-reactors):** `ResetInField` (`reactor/processor.go:357`) and `ShuffleInField`
(`reactor/processor.go:386`) both scope through `GetRegistry().GetInField(t, f)` — tenant + field (world,
channel, map, instance) — matching the plan's field-scoping requirement. `ResetInField`'s `minState`
filter (`r.State() < *minState → skip`) matches `926120300.js`'s `state >= 7` predicate exactly (`<` skip
is the same as `>=` keep).

**Test asserting the new contract:** `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go`
(grep confirms `TestResetInField...` / `TestShuffleInField...` present per plan) plus the orchestrator-side
`TestResetReactorsCarriesMinState` / `TestShuffleReactorsIssuesPost` above.

**Verdict: PASS.** The BE2A `RegisterInputHandler` re-registration is compatible with the caller; verified
empirically, not inherited.

## C18 — `reset_field` (Cosmic `resetPQ`)

**Producer:** `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:510`
(`executeResetField`) builds `saga.ResetFieldPayload{..., Difficulty}` (default 1, `strconv.Atoi`).

**Saga handler:** `saga/handler.go:2085` (`handleResetField`) → `h.fieldP.ResetField(f, payload.Difficulty)`.

**Orchestrator client:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/field/requests.go:16`
posts to `{root}worlds/%d/channels/%d/maps/%d/instances/%s/reset` with `ResetFieldInputRestModel{Difficulty}`
(json tag `difficulty`).

**Consumer (atlas-maps):** `services/atlas-maps/atlas.com/maps/map/resource.go:34` registers the same
path; `handleResetField` calls `map/monster.ProcessorImpl.ResetField(f, difficulty)`
(`services/atlas-maps/atlas.com/maps/map/monster/processor.go:183`), which composes, in Cosmic's order:

1. `p.mp.DeleteInMap(f)` → HTTP `DELETE {MONSTERS_root}worlds/%d/channels/%d/maps/%d/instances/%s/monsters`
   — this is the **existing** atlas-monsters route (`handleDeleteMonstersInMap`,
   `services/atlas-monsters/atlas.com/monsters/world/resource.go:83`), reused rather than reimplemented,
   as the plan required. Path parameters agree field-for-field on both sides.
2. `GetRegistry().RestoreSpawnPoints(p.ctx, mapKey)` — an in-process (not cross-service) call against
   atlas-maps' own spawn-point registry, scoped by `character.MapKey{Tenant, Field}`.

`difficulty` is accepted but intentionally unused, with a doc comment
(`map/monster/processor.go:172-176`) explaining atlas-maps has no difficulty-bucket concept — this
matches the plan's Step 1 instruction to record rather than silently drop the signal.

**Partial-failure behavior — NOT fully coherent, non-blocking finding:**
`ResetField` (`map/monster/processor.go:183-201`) clears monsters first, then restores spawn points, with
no compensation if the second call fails after the first succeeds. `TestResetFieldOnUnknownMapErrors`
(`services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:1303`) exercises exactly this
window: `DeleteInMap` (mocked, always succeeds) runs, then `RestoreSpawnPoints` fails because the map has
no spawn-point data, and the method returns an error to the caller/saga. In a real deployment this means
a `reset_field` step can report failure to the saga (and, upstream, to the character/script) *after*
already having deleted every monster on the field — the destructive half completed, the restorative half
did not, and nothing signals that asymmetry to the saga's error path. Retrying is safe for the monster-clear
leg (idempotent, nothing left to delete) but for a map that genuinely has no spawn-point data the retry
loops forever on the same error. This is not called out anywhere in the code or plan as an accepted trade-off.
Not blocking — the scenario requires a script author to route `reset_field` at a mapId with no known
spawn data, which none of the four G5 seeds do (922000000/926000000/926000010/926120300 all have spawn
data) — but it should be tracked as a known gap rather than left silent.

**Ordering with `spawn_monster` (926000000.json):** verified below under C19; the two-step-single-saga fix
is present and tested.

**Verdict: PASS with one non-blocking finding** (partial-failure asymmetry in `ResetField`, documented
above, `services/atlas-maps/atlas.com/maps/map/monster/processor.go:183-201`).

## C19 — the four G5 seed documents

Checked all four documents (`deploy/seed/gms/83_1/map-actions/onUserEnter/map-{922000000,926000000,
926000010,926120300}.json`, and confirmed identical `type`/`params` shape across the other 10 seeded
roots) against `executor.go`'s `ExecuteOperation` switch (`script/executor.go:38-83`) and the generated
schema (`services/atlas-map-actions/docs/map_script_schema.json`):

| doc | operation `type` | `params` keys | executor arm | schema block |
|---|---|---|---|---|
| 922000000 | `clear_drops`, `reset_reactors`, `shuffle_reactors` | none, none, none | `executor.go:76,78,80` | `map_script_schema.json:707-770` |
| 926000000 | `reset_field`, `spawn_monster` | `difficulty`; `monsterId`,`spawnIfAbsent`,`x`,`y` | `executor.go:82` + existing spawn arm | `map_script_schema.json:773-793` (+ existing spawn_monster block) |
| 926000010 | `reset_field` | `difficulty` | `executor.go:82` | same as above |
| 926120300 | `reset_reactors` | `minState` | `executor.go:78` | `map_script_schema.json:734-751` |

All four `type` strings and all `params` keys (`minState`, `difficulty`, plus the pre-existing
`monsterId`/`spawnIfAbsent`/`x`/`y`) match exactly what the executor arms read
(`strconv.ParseInt(params["minState"], 10, 8)` at `executor.go:453`; `strconv.Atoi(params["difficulty"])`
at `executor.go:534`). `./tools/gen-map-action-schema.sh --check` (run during this review) reports the
schema up to date with the executor's switch.

**Race the plan itself flagged, verified fixed:** 926000000.json's `reset_field` immediately followed by
`spawn_monster` is a genuine hazard — two independent sagas have no cross-saga ordering guarantee, so a
naive implementation could let the spawn land before (or be wiped by) the reset. `ExecuteOperations`
(`script/executor.go:100-121`) detects this exact adjacent pair and calls
`executeResetFieldThenSpawnMonster` (`script/executor.go:557-580`), which emits **one** saga with the
reset step before the spawn step. I confirmed the ordering guarantee this depends on actually holds in
the orchestrator: `saga/processor.go:526` ("Steps dispatch serially, so the earliest pending step IS the
in-flight one") and `FindEarliestPendingStepIndex`/`MarkEarliestPendingStep*` (`saga/processor.go:696-716`)
implement strict serial, earliest-pending-first dispatch — the executor's comment is not aspirational,
it's backed by the actual dispatch code.

**Test asserting the new contract:**
`services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go:1480`
(`TestExecuteOperationsCombinesResetFieldThenSpawnMonster`) asserts exactly one saga is created with two
steps in the correct order and correct payloads for the 926000000.json shape. This test would fail against
a naive "call ExecuteOperation per op" implementation (it would see two sagas), so it is not a
coincidental pass.

The other three sibling operations in 922000000.json (`clear_drops` → `reset_reactors` → `shuffle_reactors`)
are dispatched as three independent sagas with no such combining. I checked whether that is also a hazard:
it is not — `clear_drops` touches atlas-drops only; `reset_reactors` only ever writes `State`
(`reactor/processor.go:366-368`); `shuffle_reactors` only ever writes `(x,y)`
(`reactor/processor.go:407-408`). None of the three read a field the others write, so there is no
observable difference if they race or interleave, unlike the reset_field/spawn_monster case where the
reset's monster-clear could destroy the very entity the spawn just created.

**Verdict: PASS.**

## Overall

No blocking findings. One non-blocking finding on `reset_field`'s partial-failure asymmetry
(`services/atlas-maps/atlas.com/maps/map/monster/processor.go:183-201`) — recommend a follow-up
task/comment before this ever targets a field known to lack spawn-point data, but it does not affect any
of the four G5 seeds landed in this branch.

All four seams (clear_drops, reset_reactors, shuffle_reactors, reset_field) have producer and consumer
field-sets/types/enum values in agreement, and each hop is backed by a test that would fail without the
change.
