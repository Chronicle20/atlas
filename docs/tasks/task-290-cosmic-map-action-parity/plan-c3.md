# Cosmic Map-Action Parity — Plan C, Part 3 (Tasks C16–C23)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish Plan C. This file holds G5's three bulk-mutation surfaces (C16–C18) and
its seeds (C19), the area-info persistence and condition G12 still needs (C20–C21), and
the explorer-quest capability G14 needs (C22–C23).

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md)). Read **[plan-c.md](plan-c.md)
first** — its Global Constraints, §0, §0.1 and §0.2 apply unchanged and are not repeated.

**Preconditions:** [plan.md](plan.md) green; [plan-b.md](plan-b.md) landed; Tasks C1–C15
landed.

---

## Task C16: `clear_drops` — a field-scoped drop clear in atlas-drops (G5)

Cosmic's no-arg `MapleMap.clearDrops()` iterates **every** `ITEM` map object on the map
(search center `(0,0)`, radius `POSITIVE_INFINITY`), removes each, decrements
`droppedItemCount`, and broadcasts `PacketCreator.removeItemFromMap(objectId, 0, 0)` with
owner id `0`. Whole map, not owner-filtered. (The other overload, `clearDrops(player)`,
is owner-scoped and is **not** what `922000000` calls.)

atlas-drops' field REST surface is read-only today:

```go
// services/atlas-drops/atlas.com/drops/map/resource.go:23-29
r := router.PathPrefix("/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/drops").Subrouter()
r.HandleFunc("", registerGet("get_drops_in_map", handleGetDropsInMap)).Methods(http.MethodGet)
```

`drop.Processor` exposes `Spawn/Reserve/CancelReservation/Gather/Consume/Expire/GetById/GetForMap`
(`services/atlas-drops/atlas.com/drops/drop/processor.go:113-328`) — no bulk clear.

### Files

- `services/atlas-drops/atlas.com/drops/drop/processor.go` — a `ClearForField` method
- `services/atlas-drops/atlas.com/drops/map/resource.go` — a `DELETE` route and handler
- `services/atlas-drops/atlas.com/drops/drop/processor_test.go` — extend
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/drops/{processor.go,requests.go,rest.go}` — **new package**
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}`
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json`

Module roots: `services/atlas-drops/atlas.com/drops`, `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `drop.Processor.Expire` (`drop/processor.go`, locate with
`grep -n "func (p \*ProcessorImpl) Expire" services/atlas-drops/atlas.com/drops/drop/processor.go`)
is the single-drop removal that already emits the client-facing removal event — the bulk
clear is that, per drop, over `GetForMap`'s result. Do **not** invent a new removal event;
reuse whatever `Expire` emits so atlas-channel needs no change. For the orchestrator's
domain client, copy
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` wholesale.

**Interfaces:**
- Produces:
  - `drop.Processor.ClearForField(f field.Model) (int, error)` — returns the number of drops removed
  - `DELETE /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/drops` → `204 No Content`
  - `saga.ClearDrops Action = "clear_drops"`; `saga.ClearDropsPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID}`
  - the `clear_drops` document operation, no required params

- [ ] **Step 1: Read the single-drop removal path**

Run:
```bash
grep -n "func (p \*ProcessorImpl)" services/atlas-drops/atlas.com/drops/drop/processor.go
sed -n "$(grep -n 'func (p \*ProcessorImpl) Expire' services/atlas-drops/atlas.com/drops/drop/processor.go | cut -d: -f1),+40p" services/atlas-drops/atlas.com/drops/drop/processor.go
```

Note the event it emits and the registry call it makes. `ClearForField` must do exactly
that per drop so the client sees each removal — a bulk delete that skips the event would
leave every client rendering ghost drops.

- [ ] **Step 2: Write the failing test**

Append to `services/atlas-drops/atlas.com/drops/drop/processor_test.go`, copying the
setup of the nearest existing `Spawn`/`Expire` test in that file.

**`TestClearForFieldRemovesEveryDrop`** — spawn three drops on field
`f = field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(922000000)).Build()`. Call
`ClearForField(f)`. Assert it returns `3, nil` and that `GetForMap(f)` returns an empty
slice.

**`TestClearForFieldIsFieldScoped`** — spawn two drops on field instance A and one on
field instance B of the same map. `ClearForField(fA)` returns `2, nil`; `GetForMap(fB)`
still returns 1.

**`TestClearForFieldOnEmptyFieldSucceeds`** — `ClearForField` on a field with no drops
returns `0, nil` and no error.

**`TestClearForFieldEmitsRemovalPerDrop`** — using whatever producer seam the existing
tests use to capture emitted events, assert one removal event per cleared drop, each
naming the correct drop id.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd services/atlas-drops/atlas.com/drops && go test ./drop/... -run TestClearForField -v`
Expected: compile failure — `ClearForField` undefined.

- [ ] **Step 4: Implement `ClearForField` and the DELETE route**

The processor method iterates `GetForMap(f)` and removes each drop through the same path
`Expire` uses, accumulating the count and returning the first error. The route handler
mirrors `handleGetDropsInMap`'s field-parameter parsing
(`services/atlas-drops/atlas.com/drops/map/resource.go:31`) and responds `204` on success.

Register it beside the GET:

```go
		r.HandleFunc("", registerDelete("clear_drops_in_map", handleClearDropsInMap)).Methods(http.MethodDelete)
```

using whatever registration helper the file already imports for non-GET methods — read
the file and match it; do not add a new helper.

- [ ] **Step 5: Add the saga action and orchestrator client**

`libs/atlas-saga/model.go`:

```go
	// ClearDrops removes every drop from a field. Cosmic's no-arg
	// MapleMap's no-arg drop clear is whole-map, not owner-filtered (task-290 G5).
	ClearDrops Action = "clear_drops"
```

with `ClearDropsPayload` as specified in Interfaces, the `unmarshal.go` case, and a
`TestUnmarshalClearDropsStep` round-trip test.

Create `saga-orchestrator/drops/` by copying `monster/`, with path constant
`worlds/%d/channels/%d/maps/%d/instances/%s/drops` and a `DELETE` request
(`requests.DeleteRequest`, or whatever `libs/atlas-rest/requests` exposes — check with
`grep -n "func Delete" libs/atlas-rest/requests/*.go`). Root URL token: find the existing
one with `grep -rn '"DROPS"' services/`.

Wire the seven orchestrator touchpoints per [plan-c.md](plan-c.md) Task C4 Step 5.

- [ ] **Step 6: Add the executor arm and schema block**

`case "clear_drops": return e.executeClearDrops(f, characterId, op)`. The method takes no
params; it builds the one-step saga directly with `SetInitiatedBy("map-action-clear-drops")`
and step id `fmt.Sprintf("clear-drops-%d", characterId)`.

Executor test **`TestExecuteClearDrops`** — no params, field
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(922000000)).SetInstance(inst).Build()`.
Assert the payload equals
`saga.ClearDropsPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, MapId: 922000000, Instance: inst}`.

Schema `allOf`: `clear_drops` with no required params — emit the block anyway so the
generator's "every operation has an `allOf` block" assertion passes, with `params` an
empty-property object and a description explaining it takes none.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-drops/atlas.com/drops && go build ./... && go test ./... && cd -
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check
git add services/ libs/atlas-saga/
git commit -m "feat(drops): field-scoped clear and the clear_drops saga action (G5)"
```

---

## Task C17: `reset_reactors` and `shuffle_reactors` in atlas-reactors (G5)

Two Cosmic primitives, both on `MapleMap`:

- `resetReactors()` (`MapleMap.java:1545`) collects every live `Reactor` on the map and
  calls `resetReactors(list)`. `resetReactors(List<Reactor>)` (`MapleMap.java:1563`) skips
  any reactor whose `forceDelayedRespawn()` returns true (those respawn themselves), and
  for the rest locks the reactor, calls `resetReactorActions(0)`, `setAlive(true)` and
  broadcasts `triggerReactor(r, 0)`.
- `shuffleReactors()` (`MapleMap.java:1580`) collects the `Point` of every live reactor,
  `Collections.shuffle`s the list, and reassigns the shuffled positions back onto the same
  reactor objects. **It permutes positions only** — ids, states and identities are
  untouched.

**PRD correction.** PRD §4.3's G5 row asks for "`reset_reactors` (whole-map and
filtered-by-state)". There is no state-filtered Java overload. `926120300.js` computes the
filter *in JavaScript* — a local `getInactiveReactors(map)` helper walks `map.getReactors()`
and collects each reactor whose `getState() >= 7` (`Reactor.getState()`,
`server/maps/Reactor.java:100`), then passes that `List` to the same single
`resetReactors(List)` overload. So the server-side capability is one reset with an
**optional minimum-state filter**, not two methods.

atlas-reactors' state is a bare `int8` with no enum
(`services/atlas-reactors/atlas.com/reactors/reactor/model.go:16-30`, `Model.State()` at
line 67). Its processor has `GetById/GetInField/Create/DestroyInField/DestroyAll/DestroyInTenant/Hit/Touch/advance/Trigger/TriggerAndDestroy`
(`reactor/processor.go:60-339`) — no reset, no shuffle.

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` — `ResetInField` and `ShuffleInField`
- `services/atlas-reactors/atlas.com/reactors/reactor/resource.go` — two new routes
- `services/atlas-reactors/atlas.com/reactors/reactor/rest.go` — the reset input model
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` — extend
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/` — extend if the package exists (`HitReactor` already has a client), else create by copying `monster/`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}`
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json`

Module roots: `services/atlas-reactors/atlas.com/reactors`, `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `reactor.Processor.DestroyInField` — the existing field-scoped bulk
operation; read it and mirror its iteration, its event emission and its route
registration. For the orchestrator client, check first whether one already exists:
`ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/ | grep -i reactor`
(`HitReactor` is an existing action, `libs/atlas-saga/model.go` party-quest block).

**Interfaces:**
- Produces:
  - `reactor.Processor.ResetInField(f field.Model, minState *int8) (int, error)` — `nil` minState resets every reactor; non-nil resets only those with `State() >= *minState`. Returns the count reset.
  - `reactor.Processor.ShuffleInField(f field.Model) error`
  - `POST .../reactors/reset` with body `{"minState": <int8>}` (optional) and `POST .../reactors/shuffle`
  - `saga.ResetReactors Action = "reset_reactors"`; `saga.ResetReactorsPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID, MinState *int8}`
  - `saga.ShuffleReactors Action = "shuffle_reactors"`; `saga.ShuffleReactorsPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID}`
  - document operations `reset_reactors` (optional param `minState`) and `shuffle_reactors` (no params)

- [ ] **Step 1: Read the existing bulk path and the state model**

Run:
```bash
sed -n '1,80p' services/atlas-reactors/atlas.com/reactors/reactor/model.go
grep -n "func (p \*ProcessorImpl)" services/atlas-reactors/atlas.com/reactors/reactor/processor.go
sed -n "$(grep -n 'func (p \*ProcessorImpl) DestroyInField' services/atlas-reactors/atlas.com/reactors/reactor/processor.go | cut -d: -f1),+35p" services/atlas-reactors/atlas.com/reactors/reactor/processor.go
sed -n '20,45p' services/atlas-reactors/atlas.com/reactors/reactor/resource.go
```

Record two things in the commit body:
1. Whether atlas-reactors has an analogue of Cosmic's `forceDelayedRespawn()` — a reactor
   that respawns itself and must be skipped by a reset. If it has none, reset every
   reactor and say so.
2. What "reset" means against Atlas' model. Cosmic sets state to 0
   (`resetReactorActions(0)`) and `alive = true`. Match that: state `0`, alive, and emit
   whatever event `advance`/`Trigger` already emits for a state change so clients
   re-render.

- [ ] **Step 2: Write the failing tests**

Append to `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go`.

**`TestResetInFieldResetsEveryReactor`** — create three reactors on
`f = field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(922000000)).Build()` and advance
them to states `2`, `5`, `8`. Call `ResetInField(f, nil)`. Assert it returns `3, nil` and
that `GetInField(f)` reports `State() == 0` for all three.

**`TestResetInFieldHonorsMinState`** — same three reactors at states `2`, `5`, `8`. Call
`ResetInField(f, ptr(int8(7)))`. Assert it returns `1, nil`, that the state-8 reactor is
now `0`, and that the state-2 and state-5 reactors are unchanged. This is `926120300`'s
exact predicate.

**`TestResetInFieldIsFieldScoped`** — reactors on two instances of the same map; a reset
on instance A leaves instance B untouched.

**`TestShuffleInFieldPermutesPositionsOnly`** — create four reactors at four distinct
`(x, y)` positions with four distinct ids. Call `ShuffleInField(f)`. Assert: the multiset
of positions after equals the multiset before; the set of reactor ids is unchanged; every
reactor's `State()` is unchanged. Run the shuffle 50 times and assert at least one run
produced a different id→position mapping than the original, so a no-op implementation
fails.

**`TestShuffleInFieldWithOneReactorIsANoOp`** — a single reactor keeps its position and
returns no error.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/... -run 'TestResetInField|TestShuffleInField' -v`
Expected: compile failure — `ResetInField`/`ShuffleInField` undefined.

- [ ] **Step 4: Implement the two processor methods and their routes**

Mirror `DestroyInField`'s iteration and event emission. `ShuffleInField` uses
`math/rand/v2`'s `rand.Shuffle` over the collected positions, then writes each back.

Routes go on the existing field-scoped subrouter
(`services/atlas-reactors/atlas.com/reactors/reactor/resource.go:27-33`):

```go
		r.HandleFunc("/reset", registerPost("reset_reactors_in_map", handleResetReactorsInMap)).Methods(http.MethodPost)
		r.HandleFunc("/shuffle", registerPost("shuffle_reactors_in_map", handleShuffleReactorsInMap)).Methods(http.MethodPost)
```

matching the file's existing registration helper for POST.

- [ ] **Step 5: Add the two saga actions and the orchestrator client**

Both actions, both payloads, both `unmarshal.go` cases, and round-trip tests
`TestUnmarshalResetReactorsStep` (with and without `minState`) and
`TestUnmarshalShuffleReactorsStep`.

`MinState *int8` is a pointer so "no filter" and "filter at 0" are distinguishable; the
JSON tag is `\`json:"minState,omitempty"\``.

Wire the seven orchestrator touchpoints for each.

- [ ] **Step 6: Add the executor arms and schema blocks**

`case "reset_reactors":` and `case "shuffle_reactors":`.

`executeResetReactors` reads the optional `minState` param with
`strconv.ParseInt(s, 10, 8)`, erroring `invalid minState [%s]` on failure, and sets the
payload pointer only when the param is present.

Executor tests:

| test | params | expected payload |
|---|---|---|
| `TestExecuteResetReactorsAll` | `{}` | `ResetReactorsPayload{..., MinState: nil}` |
| `TestExecuteResetReactorsFiltered` | `{"minState":"7"}` | `MinState` points to `int8(7)` |
| `TestExecuteResetReactorsBadMinState` | `{"minState":"x"}` | error containing `invalid minState [x]` |
| `TestExecuteResetReactorsMinStateOverflow` | `{"minState":"300"}` | error containing `invalid minState [300]` |
| `TestExecuteShuffleReactors` | `{}` | `ShuffleReactorsPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, MapId: 922000000, Instance: inst}` |

Schema blocks: `reset_reactors` with optional `minState` (`"Reset only reactors whose state is at least this value; omit to reset all"`), `shuffle_reactors` with no params.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./... && cd -
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check
git add services/ libs/atlas-saga/
git commit -m "feat(reactors): field reset with an optional min-state filter, and shuffle (G5)

Cosmic has no state-filtered resetReactors overload; 926120300 computes the
state>=7 filter in script and calls resetReactors(List). Modelled here as one
reset with an optional minState, not two methods."
```

---

## Task C18: `reset_field` — Cosmic's `resetPQ` (G5)

**PRD correction (plan-c.md §0.1).** `MapleMap.resetPQ(int difficulty)`
(`MapleMap.java:3962-3975`) is:

```text
// Cosmic Java, MapleMap.java:3962-3975
public void resetPQ(int difficulty) { resetMapObjects(difficulty, true); }
public void resetMapObjects(int difficulty, boolean isPq) {
    clearMapObjects();
    restoreMapSpawnPoints();
    instanceMapFirstSpawn(difficulty, isPq);
}
```

It clears every live map object, restores the map's static spawn points, and re-runs the
first-spawn pass at the given difficulty with `isPq = true` hard-coded. The owner is the
**field**, not `atlas-party-quests` — whose REST surface is read-only
(`instance/resource.go:24-28`, five GET routes) and whose only `Reset*` symbol is the
test-only `Registry.ResetForTesting()` (`instance/registry.go:162`).

atlas-maps owns the spawn-point registry
(`services/atlas-maps/atlas.com/maps/map/monster/registry.go`, with
`InitializeForMap`, `Reset`, `ReserveEligibleSpawnPoints`), so the field reset lands
there, coordinating with atlas-monsters for the object clear.

### Files

- `services/atlas-maps/atlas.com/maps/map/<the package that owns field reset>/processor.go` — a `ResetField` method
- `services/atlas-maps/atlas.com/maps/.../resource.go` — a `POST .../reset` route
- `services/atlas-maps/atlas.com/maps/.../processor_test.go` — extend
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}`
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json`
- `services/atlas-monsters/atlas.com/monsters/world/resource.go` — read-only; `handleDeleteMonstersInMap` (line 83) is the existing whole-field monster clear the reset composes with

Module roots: `services/atlas-maps/atlas.com/maps`, `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/monster/registry.go`'s `Reset`
for the spawn-point restore half, and
`services/atlas-monsters/atlas.com/monsters/world/resource.go:83` (`handleDeleteMonstersInMap`)
for the object-clear half — that route already exists and should be called, not
reimplemented.

**Interfaces:**
- Produces: `saga.ResetField Action = "reset_field"`; `saga.ResetFieldPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID, Difficulty int}`; `POST .../reset` on the field; the `reset_field` document operation with optional param `difficulty` (default `1`).

- [ ] **Step 1: Establish which package owns a field reset, and what "first spawn" means**

Run:
```bash
ls services/atlas-maps/atlas.com/maps/map/
grep -rn "func (p \*ProcessorImpl)" services/atlas-maps/atlas.com/maps/map/monster/processor.go 2>/dev/null | head -20
grep -rn "InitializeForMap\|Reset(" services/atlas-maps/atlas.com/maps/map/monster/registry.go
grep -rn "difficulty\|Difficulty" services/atlas-maps/atlas.com/maps/ --include='*.go' | head
```

Two decisions, both recorded in the commit body:

1. **Does Atlas model spawn difficulty at all?** Cosmic's `instanceMapFirstSpawn(difficulty, isPq)`
   selects a difficulty bucket. If atlas-maps' spawn-point model has no difficulty
   concept, the payload still carries `Difficulty` (all four G5 scripts pass `1`, so the
   value is constant today) but the implementation ignores it — say so in a comment on the
   field rather than dropping it, because dropping it would silently lose the only signal
   Cosmic gives.
2. **Which objects does "clear" cover?** Cosmic's `clearMapObjects()` removes monsters,
   drops, reactors and NPCs. Decide the Atlas composition explicitly: at minimum the
   monster clear (atlas-monsters' existing `DELETE .../monsters`) plus the spawn-point
   restore. Write down which object classes are and are not cleared; do not leave it
   implicit.

- [ ] **Step 2: Write the failing test**

**`TestResetFieldRestoresSpawnPointsAndClearsMonsters`** — on field
`f = field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()`,
with the spawn-point registry initialized and some points reserved, and with monsters
present via the monster client seam the package's existing tests use:
call `ResetField(f, 1)`. Assert the spawn-point registry reports all points unreserved
(mirror whatever `Reset` already asserts in `registry_test.go`) and that the monster clear
was invoked once for `f`.

**`TestResetFieldIsFieldScoped`** — a reset on instance A leaves instance B's reserved
spawn points and monsters untouched.

**`TestResetFieldOnUnknownMapErrors`** — a field whose map has no spawn-point data returns
an error rather than silently succeeding.

Match the surrounding tests' seam style — if the package injects its monster client
through a package-level var or a constructor parameter, use that; do not add a new one.

- [ ] **Step 3: Run the test and confirm it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./map/... -run TestResetField -v`
Expected: compile failure — `ResetField` undefined.

- [ ] **Step 4: Implement `ResetField` and its route**

Compose the two halves in Cosmic's order: clear first, then restore spawn points, then
(if Step 1 found a difficulty-aware first-spawn) run it.

- [ ] **Step 5: Add the saga action and wire the orchestrator**

`libs/atlas-saga/model.go`:

```go
	// ResetField clears a field's objects and restores its spawn points —
	// Cosmic's MapleMap resetPQ, which despite the name is a field
	// reset and not a party-quest state reset (task-290 G5).
	ResetField Action = "reset_field"
```

Payload as in Interfaces, `unmarshal.go` case, `TestUnmarshalResetFieldStep`, then the
seven orchestrator touchpoints with a domain client pointed at the new atlas-maps route.

- [ ] **Step 6: Add the executor arm and schema block**

`case "reset_field": return e.executeResetField(f, characterId, op)`. Optional
`difficulty` param, `strconv.Atoi`, default `1`, error `invalid difficulty [%s]`.

Executor tests:

| test | params | expected payload field |
|---|---|---|
| `TestExecuteResetFieldDefaultsDifficultyToOne` | `{}` | `Difficulty: 1` |
| `TestExecuteResetFieldExplicitDifficulty` | `{"difficulty":"2"}` | `Difficulty: 2` |
| `TestExecuteResetFieldBadDifficulty` | `{"difficulty":"x"}` | error containing `invalid difficulty [x]` |

Schema block: `reset_field`, optional `difficulty`, description
`"Spawn difficulty bucket for the post-reset first spawn (default 1)"`.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... && cd -
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check
git add services/ libs/atlas-saga/
git commit -m "feat(maps): reset_field action for Cosmic resetPQ (G5)

resetPQ(n) is clearMapObjects + restoreMapSpawnPoints + instanceMapFirstSpawn,
i.e. a field reset. PRD 4.3 assigned it to a party-quest service; corrected."
```

---

## Task C19: the four G5 seeds

### Files

Create under all 11 roots at `map-actions/onUserEnter/`: `map-922000000.json`,
`map-926000000.json`, `map-926000010.json`, `map-926120300.json` — 44 files total.

All four maps are **instanced**, which is why Plan A Task A5's `f.Instance()` fix
(design F3) had to land first: without it every spawn and every reset in these documents
would target the non-instanced field.

- [ ] **Step 1: Author `map-922000000.json`**

Cosmic order: `map.clearDrops()`, `map.resetReactors()`, `map.shuffleReactors()`.

```json
{
  "data": {
    "attributes": {
      "description": "Map 922000000 - clears drops, resets every reactor and shuffles reactor positions on entry",
      "rules": [
        {
          "conditions": [],
          "id": "reset_stage",
          "operations": [
            {
              "type": "clear_drops"
            },
            {
              "type": "reset_reactors"
            },
            {
              "type": "shuffle_reactors"
            }
          ]
        }
      ],
      "scriptName": "922000000"
    },
    "id": "922000000",
    "type": "map-action"
  }
}
```

- [ ] **Step 2: Author `map-926000000.json`**

Cosmic: `map.resetPQ(1)`, then a separate guard
`if (map.countMonster(9100013) == 0) { spawn 9100013 @ (82, 200) }`. Per
[plan-c.md](plan-c.md) §0.1, `countMonster(...) == 0` is `spawnIfAbsent`.

One rule, two operations, in Cosmic's order:

```json
{
  "data": {
    "attributes": {
      "description": "Map 926000000 - resets the field and spawns 9100013 at (82, 200) if absent",
      "rules": [
        {
          "conditions": [],
          "id": "reset_and_spawn",
          "operations": [
            {
              "params": {
                "difficulty": "1"
              },
              "type": "reset_field"
            },
            {
              "params": {
                "monsterId": "9100013",
                "spawnIfAbsent": "true",
                "x": "82",
                "y": "200"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "926000000"
    },
    "id": "926000000",
    "type": "map-action"
  }
}
```

Ordering caveat to check and record: `reset_field` clears the field's monsters, and the
spawn follows it. The two operations become two saga steps; confirm the executor's
`ExecuteOperations` ordering guarantee actually holds across saga boundaries here — if
each operation is its own independent saga, the spawn may race the reset. Read
`e.sagaP.Create` and the orchestrator's step sequencing; if they can race, emit both steps
in **one** saga rather than two, and note the change.

- [ ] **Step 3: Author `map-926000010.json`**

Cosmic: `map.resetPQ(1)` only.

```json
{
  "data": {
    "attributes": {
      "description": "Map 926000010 - resets the field on entry",
      "rules": [
        {
          "conditions": [],
          "id": "reset_stage",
          "operations": [
            {
              "params": {
                "difficulty": "1"
              },
              "type": "reset_field"
            }
          ]
        }
      ],
      "scriptName": "926000010"
    },
    "id": "926000010",
    "type": "map-action"
  }
}
```

- [ ] **Step 4: Author `map-926120300.json`**

Cosmic: `map.resetReactors(getInactiveReactors(map))`, where the helper collects reactors
with `getState() >= 7`.

```json
{
  "data": {
    "attributes": {
      "description": "Map 926120300 - resets only reactors whose state is 7 or higher",
      "rules": [
        {
          "conditions": [],
          "id": "reset_inactive_reactors",
          "operations": [
            {
              "params": {
                "minState": "7"
              },
              "type": "reset_reactors"
            }
          ]
        }
      ],
      "scriptName": "926120300"
    },
    "id": "926120300",
    "type": "map-action"
  }
}
```

- [ ] **Step 5: Replicate and verify**

```bash
for f in 922000000 926000000 926000010 926120300; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `44`.

- [ ] **Step 6: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): four G5 stage-reset map-actions"
```

---

## Task C20: area-info persistence and the `areaInfo` condition (G12)

Cosmic's area info is per-character server state:
`Character.area_info` is a `Map<Short, String>` (`client/Character.java:326`) persisted to
an `area_info` table (`Character.java:7087-7092` load, `8540-8544` save,
`2289` delete). `containsAreaInfo(area, info)` is
`area_info.get(area).contains(info)` — a **substring** test
(`Character.java:9782-9788`). `updateAreaInfo(area, info)` is a full **replace** of the
stored string, plus the client packet (`Character.java:9790-9793`).

Atlas stores none of this: `UpdateAreaInfo` (Task C3) is fire-and-forget to the client, and
`grep -rli areaInfo services/atlas-character` finds nothing. So `rienArrow` and `rien`,
which both gate on `containsAreaInfo`, cannot work without server-side state.

This task adds the persistence, makes `UpdateAreaInfo` write it, and adds the aggregator
condition that reads it.

**This task widens the canonical wire contract.** See [plan-c.md](plan-c.md) §0.2: it adds
`ValueString string \`json:"valueString,omitempty"\`` to
`saga.ValidationConditionInput`, because the operand is intrinsically a string and no
existing field can carry it. Confirm the reviewer agrees before starting.

### Files

- `services/atlas-character/atlas.com/character/area_info/{entity.go,model.go,builder.go,administrator.go,processor.go,rest.go,resource.go}` — **new package**
- `services/atlas-character/atlas.com/character/area_info/processor_test.go` — **new file**
- the atlas-character migration/AutoMigrate registration for the new entity
- `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go` — read-only; the packet side is unchanged
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — `handleUpdateAreaInfo` also persists
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/` (or a new `area_info` client) — the persist call
- `libs/atlas-saga/validation.go` — `AreaInfoCondition` constant and the `ValueString` field
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/{model.go,builder.go,rest.go,context.go}` — the condition's touchpoints and a `GetAreaInfo` getter
- `services/atlas-query-aggregator/atlas.com/query-aggregator/area_info/` — **new REST client package**
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go` — extend
- `services/atlas-map-actions/atlas.com/map-actions/script/rest.go`, `entity.go`, `evaluator.go` — carry `valueString`
- `services/atlas-map-actions/docs/map_script_schema.json` — the condition's `valueString` property

Module roots: `services/atlas-character/atlas.com/character`, `libs/atlas-saga`,
`services/atlas-query-aggregator/atlas.com/query-aggregator`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: any existing small per-character GORM-backed package in atlas-character
— locate one with `ls services/atlas-character/atlas.com/character/` and pick the
smallest entity+model+builder+administrator+processor+rest+resource set, then mirror it
exactly. For the aggregator condition's nine touchpoints, [context.md](context.md)'s
`transportAvailable` checklist again; unlike Task C10 this one **does** need a new
`ValidationContext` getter, modelled on `GetTransportState`
(`query-aggregator/validation/context.go:346-359`) and its `transportP` wiring
(`context.go:51,74,98,417`; `builder.go:38,61,85,172`).

**Interfaces:**
- Produces:
  - atlas-character: `GET /characters/{characterId}/area-info` and `GET /characters/{characterId}/area-info/{area}`; `PUT /characters/{characterId}/area-info/{area}` with body `{"info": "..."}`
  - `saga.AreaInfoCondition = "areaInfo"`
  - `saga.ValidationConditionInput.ValueString string`
  - `validation.ConditionInput.ValueString string` in map-actions
  - `condition.Model.ValueString() string` is **not** added — the document's existing `value` field carries the substring, and map-actions copies it into `ValueString` when the condition type is `areaInfo`. See Step 6.
  - the document condition `{"type": "areaInfo", "operator": "=", "referenceId": "<area>", "value": "<substring>"}`

- [ ] **Step 1: Confirm the wire widening with the reviewer, then add it**

In `libs/atlas-saga/validation.go`'s `ValidationConditionInput` (lines 65-75), after
`Values`:

```go
	// ValueString carries a string operand for conditions whose comparand is
	// not numeric. Its only user today is areaInfo, whose Cosmic semantics are
	// a substring test against a free-form per-character string (task-290 G12).
	// Numeric conditions continue to use Value/Values and never set this.
	ValueString string `json:"valueString,omitempty"`
```

Add the condition constant beside the others:

```go
	AreaInfoCondition = "areaInfo"
```

Run `cd libs/atlas-saga && go build ./... && go test ./...` and confirm every dependent
module still builds:

```bash
for m in services/atlas-query-aggregator/atlas.com/query-aggregator \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         services/atlas-map-actions/atlas.com/map-actions \
         services/atlas-npc-conversations/atlas.com/npc-conversations \
         services/atlas-party-quests/atlas.com/party-quests; do
  (cd "$m" && go build ./...) || echo "BROKE: $m"
done
```
Expected: no `BROKE` lines — the field is additive.

- [ ] **Step 2: Write the failing atlas-character tests**

`services/atlas-character/atlas.com/character/area_info/processor_test.go`, using the
project's Builder pattern for setup (no `*_testhelpers.go`).

**`TestUpsertAreaInfoReplacesWholeString`** — upsert `(character 1, area 21019, "miss=o;helper=clear")`,
then upsert `(1, 21019, "miss=o;arr=o;helper=clear")`. Assert a `GetByArea(1, 21019)`
returns exactly `"miss=o;arr=o;helper=clear"` — a replace, not a merge, matching
`Character.updateAreaInfo`.

**`TestGetByAreaMissingReturnsEmpty`** — an unset area returns the empty string and no
error (Cosmic's `area_info.get(area)` on a missing key is handled as "does not contain").

**`TestAreaInfoIsPerCharacter`** — character 1 and character 2 hold independent values for
the same area.

**`TestAreaInfoIsTenantScoped`** — the same character id under two tenant contexts holds
independent values.

- [ ] **Step 3: Implement the atlas-character package and its routes**

Entity fields: tenant id, character id, `area uint16`, `info string`, with a unique index
on (tenant, character, area). Register it for AutoMigrate the same way the sibling package
you copied does.

- [ ] **Step 4: Make `UpdateAreaInfo` persist as well as announce**

`handleUpdateAreaInfo` (`saga/handler.go:2879`) currently only calls
`h.systemMessageP.UpdateAreaInfo(...)`. It must also persist through a client pointed at
the new atlas-character route, **before** announcing — so a client that re-reads the value
immediately after the packet sees the stored one.

Add a handler test in the orchestrator asserting both calls happen and in that order,
copying the shape of an existing two-call handler test in
`saga/handler_test.go`.

- [ ] **Step 5: Add the `areaInfo` aggregator condition**

Nine touchpoints per [context.md](context.md), plus a new `GetAreaInfo` getter:

```go
// GetAreaInfo returns the character's stored area-info string for area, or ""
// when unset or unavailable. Degrades to "" rather than erroring, matching
// GetTransportState's posture.
func (ctx ValidationContext) GetAreaInfo(area uint16) string
```

backed by a new `query-aggregator/area_info` REST client, wired through
`ValidationContextBuilder` exactly as `transportP` is (`builder.go:85`).

The `Evaluate` arm:

```go
	case AreaInfoCondition:
		stored := ctx.GetAreaInfo(uint16(c.referenceId))
		if strings.Contains(stored, c.valueString) {
			actualValue = 1
		} else {
			actualValue = 0
		}
```

with `c.valueString` populated from the input's `ValueString`. `containsAreaInfo` is a
substring test — **not** equality, and not a parse of the `k=v;` pairs. Cosmic compares the
literal substring and this must do the same, or `rienArrow`'s
`"miss=o;helper=clear"` guard would stop matching the stored
`"miss=o;arr=o;helper=clear"` in the way Cosmic's does. Verify that specific pair in the
test below.

Validation in all four places (`builder.go:210` accepted list, `builder.go` `FromInput`,
`builder.go` `Validate()`, `rest.go`): `referenceId` is required, and `valueString` must be
non-empty. Message: `referenceId and valueString are required for areaInfo conditions`.

- [ ] **Step 6: Write the aggregator test**

Append to `query-aggregator/validation/model_test.go`, copying the `transportAvailable`
test's setup.

**`TestAreaInfoCondition`** — table-driven, `referenceId` `21019`, `operator` `=`,
`value` `1`:

| subtest | stored area info | `valueString` | expect passed |
|---|---|---|---|
| `exact match` | `miss=o;helper=clear` | `miss=o;helper=clear` | `true` |
| `substring of a longer stored value` | `miss=o;arr=o;helper=clear` | `miss=o` | `true` |
| `rienArrow guard before the update` | `miss=o;helper=clear` | `miss=o;helper=clear` | `true` |
| `rien guard after rienArrow ran` | `miss=o;arr=o;helper=clear` | `miss=o;arr=o;helper=clear` | `true` |
| `rienArrow guard after rienArrow ran` | `miss=o;arr=o;helper=clear` | `miss=o;helper=clear` | `false` |
| `not present` | `miss=o` | `arr=o` | `false` |
| `unset area` | `` | `miss=o` | `false` |

The fifth row is the load-bearing one: it is why Cosmic's `rienArrow` fires at most once,
and a naive "parse the pairs and compare sets" implementation would get it wrong.

**`TestAreaInfoRequiresReferenceIdAndValueString`** — both missing-field cases error with
the message above.

**`TestAreaInfoAcceptedBySetType`** — `SetType("areaInfo")` returns no error.

- [ ] **Step 7: Carry `valueString` through map-actions**

`RestConditionModel` and `jsonCondition` (Plan A Task A2) gain
`ValueString string \`json:"valueString,omitempty"\``, `condition.Builder` gains
`SetValueString`/`Model.ValueString()` (mirroring Plan A Task A1's `values` addition in
`libs/atlas-script-core`), and `evaluateViaQueryAggregator` copies it into
`validation.ConditionInput.ValueString`.

Add the schema's condition property:

```json
        "valueString": {
          "type": "string",
          "description": "String operand for conditions whose comparand is not numeric (areaInfo)"
        }
```

Regenerate, `--check`.

Extend Plan A Task A3's `TestEvaluateViaQueryAggregatorCarriesEveryField` with
`ValueString`, so the F1 regression test covers the new field too.

- [ ] **Step 8: Build, test, commit**

```bash
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd libs/atlas-script-core && go build ./... && go test ./... && cd -
cd services/atlas-character/atlas.com/character && go build ./... && go test ./... && cd -
cd services/atlas-query-aggregator/atlas.com/query-aggregator && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
git add libs/ services/
git commit -m "feat(character): persist area info; add the areaInfo aggregator condition (G12)

Widens saga.ValidationConditionInput with ValueString (additive, omitempty) —
containsAreaInfo is a substring test against a free-form string and no
existing field can carry the operand. See plan-c.md section 0.2."
```

---

## Task C21: the two area-info seeds — `rienArrow` and `rien` (G12)

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-rienArrow.json` — **new file**
- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-rien.json` — **new file**

22 files total.

Patterns to copy: Task C3's `map-Resi_tutor30.json` for the `update_area_info` +
`show_info` pair; Task C5's documents for a multi-condition rule.

- [ ] **Step 1: Author `map-rienArrow.json`**

Cosmic: `if (ms.containsAreaInfo(21019, "miss=o;helper=clear")) { ms.updateAreaInfo(21019, "miss=o;arr=o;helper=clear"); ms.showInfo("Effect/OnUserEff.img/guideEffect/aranTutorial/tutorialArrow3"); }`

```json
{
  "data": {
    "attributes": {
      "description": "Rien arrow tutorial - once the miss=o;helper=clear area info is present, records arr=o and shows the tutorialArrow3 guide effect",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "21019",
              "type": "areaInfo",
              "value": "1",
              "valueString": "miss=o;helper=clear"
            }
          ],
          "id": "advance_arrow_tutorial",
          "operations": [
            {
              "params": {
                "area": "21019",
                "info": "miss=o;arr=o;helper=clear"
              },
              "type": "update_area_info"
            },
            {
              "params": {
                "path": "Effect/OnUserEff.img/guideEffect/aranTutorial/tutorialArrow3"
              },
              "type": "show_info"
            }
          ]
        }
      ],
      "scriptName": "rienArrow"
    },
    "id": "rienArrow",
    "type": "map-action"
  }
}
```

`value` is `"1"` — the condition evaluates to 1 when the substring is present, and the
document asserts it equals 1. The substring itself lives in `valueString`.

- [ ] **Step 2: Author `map-rien.json`**

Cosmic, in order:
1. `if (ms.isQuestCompleted(21101) && ms.containsAreaInfo(21019, "miss=o;arr=o;helper=clear")) { ms.updateAreaInfo(21019, "miss=o;arr=o;ck=1;helper=clear"); }`
2. `ms.unlockUI();` — **unconditional**, outside the guard, and it runs whether or not the
   guard passed.

Two rules, because the second operation has no conditions:

```json
{
  "data": {
    "attributes": {
      "description": "Rien - records ck=1 once quest 21101 is complete and the arrow tutorial has run; always unlocks the UI",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "21101",
              "type": "questStatus",
              "value": "3"
            },
            {
              "operator": "=",
              "referenceId": "21019",
              "type": "areaInfo",
              "value": "1",
              "valueString": "miss=o;arr=o;helper=clear"
            }
          ],
          "id": "record_checkpoint",
          "operations": [
            {
              "params": {
                "area": "21019",
                "info": "miss=o;arr=o;ck=1;helper=clear"
              },
              "type": "update_area_info"
            }
          ]
        },
        {
          "conditions": [],
          "id": "unlock",
          "operations": [
            {
              "type": "unlock_ui"
            }
          ]
        }
      ],
      "scriptName": "rien",
      "type": "map-action"
    },
    "id": "rien",
    "type": "map-action"
  }
}
```

Remove the stray `"type": "map-action"` from inside `attributes` when authoring — it
belongs only at `data.type`. (It is shown here so the mistake is caught in review rather
than in the file.)

`isQuestCompleted(21101)` → `questStatus referenceId 21101 = 3` (Cosmic COMPLETED(2) + 1).

- [ ] **Step 3: Confirm the multi-rule semantics**

`rien` depends on **both** rules running when both match, not just the first. Task C5
Step 3 recorded whether the engine is first-match-wins or run-all-matching. If it is
first-match-wins, this document is wrong as written: fold `unlock_ui` into the first
rule's operations *and* add a second rule with the inverse guard, or restructure per
whatever the engine actually does. Do not land it until Task C5's finding is applied.

- [ ] **Step 4: Replicate and verify**

```bash
for f in rienArrow rien; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `22`.

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): rienArrow and rien area-info gated tutorials (G12)"
```

---

## Task C22: `explorer_quest` (G14)

**Not reducible to existing actions.** Design D10 asked whether `explorerQuest` collapses
to `SetQuestProgress`/`StartQuest`. It does not.
`MapScriptMethods.explorerQuest(questid, questName)`
(`<cosmic-root>/src/main/java/scripting/map/MapScriptMethods.java:104-139`) does five
things:

1. returns immediately if the quest is already completed;
2. force-starts it via NPC `9000066` if not started (`quest.forceStart(getPlayer(), 9000066)`);
3. `qs.addMedalMap(getPlayer().getMapId())` — appends the current map to a **deduplicated
   per-quest visited-map set**, returning false (and aborting) if it was already there;
4. writes the resulting visited **count** as quest progress under the quest's own
   `infoNumber`: `setQuestProgress(quest.getId(), quest.getInfoNumber(qs.getStatus()), status)`;
5. compares that count to the quest's `infoEx(0)` and sends either
   `getShowQuestCompletion(questId)` or `earnTitleMessage("<n>/<m> regions explored.")`.

Step 3 is the part with no Atlas equivalent: a per-character, per-quest set of visited map
ids with dedup. That is the capability this task adds, in atlas-quest.

### Files

- `services/atlas-quest/atlas.com/quest/medal_map/{entity.go,model.go,builder.go,administrator.go,processor.go,rest.go,resource.go}` — **new package** 
- `services/atlas-quest/atlas.com/quest/medal_map/processor_test.go` — **new file**
- the atlas-quest AutoMigrate registration
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/processor.go` — a `RequestExplorerQuest` method
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}`
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json`

Module roots: `services/atlas-quest/atlas.com/quest`, `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: Task C20's atlas-character `area_info` package — the same shape
(tenant + character + key → value, unique index, small REST surface). The orchestrator
already has a `quest` domain client used by `handleStartQuest`
(`saga/handler.go:2165`) and `handleSetQuestProgress` (`saga/handler.go:2202`); extend it
rather than creating a new package.

**Interfaces:**
- Produces:
  - atlas-quest: `POST /characters/{characterId}/quests/{questId}/medal-maps` with body `{"mapId": <id>}` → `201` when newly recorded, `200` when already present, and a response body carrying the resulting `{"count": <n>, "threshold": <m>, "completed": <bool>}`
  - `saga.ExplorerQuest Action = "explorer_quest"`; `saga.ExplorerQuestPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, QuestId uint32, MapId _map.Id, AreaName string}`
  - the `explorer_quest` document operation with required params `questId` and `areaName`

- [ ] **Step 1: Establish atlas-quest's layout and where `infoNumber`/`infoEx` live**

The service directory is **`services/atlas-quest`** (singular), and its Go module root is
`services/atlas-quest/atlas.com/quest`. Confirm and survey it:

```bash
ls services/atlas-quest/atlas.com/quest/
grep -rn "InfoNumber\|infoNumber\|InfoEx\|infoEx" services/atlas-quest --include='*.go' | head -20
```

Two things to record in the commit body:
1. **Does Atlas expose the quest's `infoNumber` and `infoEx[0]`?** Cosmic reads both from
   WZ quest Act data. If atlas-quest (or atlas-data) already serves them, use them. If
   neither does, the completion comparison (Cosmic step 5) cannot be made faithfully —
   in that case record the visited count as progress and **skip** the completion packet,
   stating that explicitly in the response model and the commit body. Do not fabricate a
   threshold.
2. **Is there an existing force-start path?** `handleStartQuest` already exists; the
   explorer flow can compose `StartQuest` (npcId `9000066`) with the medal-map record
   rather than duplicating the start logic.

- [ ] **Step 2: Write the failing atlas-quest tests**

**`TestRecordMedalMapDeduplicates`** — record map `100000000` for character 1, quest
`29005`. Assert the result reports `count == 1` and "newly recorded". Record the same map
again; assert `count == 1` and "already present".

**`TestRecordMedalMapCountsDistinctMaps`** — record `100000000`, `100000001`, `100000000`
again, `100000002`. Assert the final count is `3`.

**`TestMedalMapsArePerQuest`** — recording `100000000` under quest `29005` does not affect
quest `29006`'s count.

**`TestMedalMapsArePerCharacterAndTenant`** — independent counts across characters and
across tenant contexts.

- [ ] **Step 3: Implement the atlas-quest package**

Entity: tenant, character id, quest id, map id, with a unique index on all four so the
dedup is enforced by the database and not by a read-then-write.

- [ ] **Step 4: Add the saga action and wire the orchestrator**

`libs/atlas-saga/model.go`:

```go
	// ExplorerQuest credits one exploration region: force-start the quest if
	// needed, append the current map to the quest's deduplicated visited-map
	// set, and write the resulting count as quest progress. Not reducible to
	// StartQuest + SetQuestProgress, because the dedup and the count are
	// server-side state neither of those carries (task-290 G14).
	ExplorerQuest Action = "explorer_quest"
```

Payload as in Interfaces, `unmarshal.go` case, `TestUnmarshalExplorerQuestStep`, then the
seven orchestrator touchpoints. The handler composes: force-start (npc `9000066`) → record
medal map → write progress, aborting silently if the map was already recorded, matching
Cosmic's `if (!qs.addMedalMap(...)) return;`.

`AreaName` is carried because Cosmic passes it and it appears in the player-facing message;
if Step 1 concluded the completion/progress messages cannot be sent faithfully, keep the
field on the payload (the seed supplies it and it is the only human-readable label of the
region) and note that it is currently unused.

- [ ] **Step 5: Add the executor arm and schema block**

`case "explorer_quest": return e.executeExplorerQuest(f, characterId, op)`, reading
`questId` (required, uint32) and `areaName` (required, string), with the payload's `MapId`
taken from `f.MapId()` — never from a param, because Cosmic credits the map the character
is actually standing in.

Executor test **`TestExecuteExplorerQuest`** — params
`{"areaName":"Beginner Explorer","questId":"29005"}`, field
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(104000000)).Build()`. Assert the
payload equals
`saga.ExplorerQuestPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, QuestId: 29005, MapId: 104000000, AreaName: "Beginner Explorer"}`.

Validation cases: missing `questId` → `explorer_quest operation missing questId parameter`;
missing `areaName` → `explorer_quest operation missing areaName parameter`;
`{"questId":"x","areaName":"a"}` → `invalid questId [x]`.

Schema `allOf`: `explorer_quest` requiring `questId` and `areaName`.

- [ ] **Step 6: Build, test, commit**

```bash
cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./... && cd -
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check
git add services/ libs/atlas-saga/
git commit -m "feat(quests): medal-map set and the explorer_quest saga action (G14)"
```

---

## Task C23: `explorationPoint` (G14)

Nine rules. Eight map-id range branches crediting an exploration region, plus one
independent `field_effect` on map `104000000`.

Two structural facts from the Cosmic source that the PRD's one-line summary omits, and
that determine whether the document is correct:

1. **The eight range branches are a single `if / else if` chain** — at most one fires per
   entry, in the literal order below.
2. **The `104000000` check is a separate top-level `if`, not part of the chain.** Map
   `104000000` satisfies branch 1's range (`>= 100000000 && < 105040300`), so on that map
   Cosmic fires **both** `explorerQuest(29005, "Beginner Explorer")` and
   `mapEffect("maplemap/enter/104000000")`.

Fact 2 means the document needs the engine to run **all** matching rules, not just the
first. Task C5 Step 3 established which the engine does. Apply that finding here before
authoring.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-explorationPoint.json` — **new file**, plus the other ten roots (11 total)

Patterns to copy: Plan B Task B1's `field_effect` operation shape; Task C5's multi-rule
document.

- [ ] **Step 1: Apply Task C5 Step 3's rule-evaluation finding**

If the engine runs every matching rule, author the nine rules as below. If it is
first-match-wins, the `104000000` effect must instead be folded into branch 1 as a second
operation guarded by its own rule placed **before** branch 1 and carrying both operations
— restructure accordingly and record the restructuring in the commit body. Do not author
until this is settled.

- [ ] **Step 2: Author the document**

Branch 1's condition is a disjunction (`mapId == 110000000 || (mapId >= 100000000 && mapId < 105040300)`),
and rule conditions are AND-only, so it becomes two rules with the same operation.
Branch 1's upper bound is **exclusive** (`<`); every other bound is inclusive.

| rule id | conditions | operation |
|---|---|---|
| `beginner_exact` | `map_id = 110000000` | `explorer_quest` 29005 / `Beginner Explorer` |
| `beginner_range` | `map_id >= 100000000`, `map_id < 105040300` | `explorer_quest` 29005 / `Beginner Explorer` |
| `sleepywood` | `map_id >= 105040300`, `map_id <= 105090900` | `explorer_quest` 29014 / `Sleepywood Explorer` |
| `el_nath` | `map_id >= 200000000`, `map_id <= 211041800` | `explorer_quest` 29006 / `El Nath Mts. Explorer` |
| `ludus_lake` | `map_id >= 220000000`, `map_id <= 222020000` | `explorer_quest` 29007 / `Ludus Lake Explorer` |
| `undersea` | `map_id >= 230000000`, `map_id <= 230040401` | `explorer_quest` 29008 / `Undersea Explorer` |
| `mu_lung` | `map_id >= 250000000`, `map_id <= 251010500` | `explorer_quest` 29009 / `Mu Lung Explorer` |
| `nihal_desert` | `map_id >= 260000000`, `map_id <= 261030000` | `explorer_quest` 29010 / `Nihal Desert Explorer` |
| `minar_forest` | `map_id >= 240000000`, `map_id <= 240050000` | `explorer_quest` 29011 / `Minar Forest Explorer` |
| `enter_effect_104000000` | `map_id = 104000000` | `field_effect` `maplemap/enter/104000000` |

Ten rules, because branch 1 splits. Area-name strings are literal, including
`El Nath Mts. Explorer`'s abbreviation and period.

Each `explorer_quest` operation:

```json
            {
              "params": {
                "areaName": "<areaName>",
                "questId": "<questId>"
              },
              "type": "explorer_quest"
            }
```

The `map_id` conditions are evaluated locally (`evaluator.go:33`), so this document costs
no aggregator round-trip — which matters, because it fires on every map entry in eight
whole regions. That is the PRD §8 performance note in practice.

The `>`/`<`/`>=`/`<=` operators on `map_id` are Plan A Task A3's addition; without it
every rule here fails with `unsupported operator`.

- [ ] **Step 3: Verify the boundary arithmetic**

Assert the ranges against the branch order with a quick check rather than by eye:

```bash
python3 - <<'PY'
import json
d = json.load(open('deploy/seed/gms/83_1/map-actions/onUserEnter/map-explorationPoint.json'))
for r in d['data']['attributes']['rules']:
    conds = [(c['type'], c['operator'], c['value']) for c in r['conditions']]
    ops = [(o['type'], o.get('params', {})) for o in r['operations']]
    print(r['id'], conds, ops)
PY
```

Check each printed line against the table above, then confirm the two boundary cases by
hand: map `105040300` must match `sleepywood` and **not** `beginner_range` (exclusive
upper bound), and map `110000000` must match `beginner_exact` and **not** any range rule.

- [ ] **Step 4: Replicate and verify**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-explorationPoint.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-explorationPoint.json"
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `11`.

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): explorationPoint region crediting across eight map-id ranges (G14)"
```

---

## Plan C completion gate

- [ ] **Confirm the document count**

Plan C lands 39 documents. With Plan A's 9 pre-existing and Plan B's 27, every root should
hold 74 `onUserEnter` documents and 1 `onFirstUserEnter`.

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/83_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  printf '%s onUserEnter=%s onFirstUserEnter=%s\n' "$r" \
    "$(ls deploy/seed/$r/map-actions/onUserEnter/ | wc -l)" \
    "$(ls deploy/seed/$r/map-actions/onFirstUserEnter/ | wc -l)"
done
```
Expected: `onUserEnter=74 onFirstUserEnter=1` for all eleven.

- [ ] **Confirm byte-identity across the roots**

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  diff -r deploy/seed/gms/83_1/map-actions deploy/seed/$r/map-actions || echo "DRIFT in $r"
done
```
Expected: no output.

- [ ] **Confirm every executor arm has a schema block and vice versa**

Run: `./tools/gen-map-action-schema.sh --check`
Expected: exit 0. The generator asserts every operation in the switch has an `allOf`
block, so this is also the FR-3.0 parity check.

- [ ] **Run the full verification gate**

Run: `./tools/verify.sh`
Expected: exit 0. `--quick` and `--no-docker` do **not** satisfy this.

- [ ] **Requirement coverage**

| Gap | Engine task | Seed task | Scripts |
|---|---|---|---|
| G1a warp-to-map | C12 | C13 | 12 |
| G1b docked state | C10 | C11, C13 | — |
| G1c music + boat effect | C9 | C11 | 2 |
| G2 `spawn_npc` | C14 | C15 | 5 |
| G3 `set_quest_progress` | C1 | C1 | 3 |
| G4 `start_quest` | C1 | C1 | 1 |
| G5 drops / reactors / field reset | C16, C17, C18 | C19 | 4 |
| G6 `questProgress` + job band | — (Plan A) | C5 | 2 |
| G7 randomized spawn | — (Plan A) | C6 | 1 |
| G8 direction | C7 (`play_sound` only) | C8 | 2 of 4 |
| G9 `open_npc` | C2 | C2 | 2 of 3 |
| G10 `set_direction_mode` | not converted (§0) | — | 0 of 2 |
| G11 `set_standalone_mode` | not converted (§0) | — | 0 of 1 |
| G12 area info | C3, C20 | C3, C21 | 3 |
| G13 skill clear | C4 | C4 | 1 |
| G14 `explorer_quest` + ranges | C22 (+ Plan A for ranges) | C23 | 1 |
| | | **total** | **39 of 45** |

The six not converted — `cannon_tuto_01`, `cannon_tuto_direction`, `Resi_tutor10`,
`Resi_tutor50`, `Resi_tutor70`, `Resi_tutor80` — are recorded with their evidence in
[plan-c.md](plan-c.md) §0 and stay on issue #1624.
