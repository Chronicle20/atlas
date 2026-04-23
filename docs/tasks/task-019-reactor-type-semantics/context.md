# Context — task-019 Reactor Type Semantics & Timer-Driven Progression

Quick-reference for an engineer picking up the plan cold. Pair with `prd.md` and `design.md`.

## Problem in one sentence

Hit-break reactors (e.g. `0002001`) get stuck at their terminal state instead of destroying-and-respawning, because atlas-data synthesises a phantom type-999 event for empty states and atlas-reactors treats type-999 as "persist forever." Timer-driven (type-101) reactors also never fire because `timeOut` is read with the wrong casing.

## Four root causes

1. `services/atlas-data/atlas.com/data/reactor/reader.go:124-130` fabricates `{Type:999, NextState:i+1}` for any state with no `event` subtree.
2. `services/atlas-reactors/atlas.com/reactors/reactor/processor.go:259-270` (`persistsAtFinalState`) returns `true` if any event anywhere is type 100 or 999 — falsely tripping for reactor 2001.
3. Matching scope is wrong even for true item reactors — it should be "the event that led to the end-state transition," not "any event anywhere."
4. `reader.go:89` reads lowercase `timeout` only; wz files use `timeOut` 157× vs `timeout` 3×.

## Reactor type taxonomy (from wz survey)

| type | meaning | end-state |
|------|---------|-----------|
| 0    | hit-break (default)    | destroy |
| 1, 2 | directional hit        | destroy |
| 5, 6, 7 | GPQ skill-gated     | **persist** |
| 100  | item-drop              | **persist** |
| 101  | timer auto-advance     | **persist (cyclic)** |

No `type=999` exists in any wz — every 999 in Atlas today is synthetic and will be removed.

## Key files

### atlas-data
- `services/atlas-data/atlas.com/data/reactor/reader.go` — wz → RestModel; remove synthesis, fix timeOut casing, extract timeoutNextState from type-101 events.
- `services/atlas-data/atlas.com/data/reactor/rest.go` — JSON model; add `TimeoutNextStateInfo map[int8]int8`.
- `services/atlas-data/atlas.com/data/reactor/reader_test.go` — existing fixtures `testXML` (1002000, type-101 at state 5), `infoFallbackTestXML` (2001, no event at state 2). Both are exercised; we add cases for "empty terminal state absent from StateInfo" and "type-101 populates TimeoutNextStateInfo."

### atlas-reactors
- `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go` — JSON:API RestModel + `Extract`; add `TimeoutNextStateInfo` field and carry into `Model`.
- `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go` — immutable `data.Model`; add `Timeout(state int8) int32` and `TimeoutNextState(state int8) (int8, bool)` accessors (neither exists today).
- `services/atlas-reactors/atlas.com/reactors/reactor/data/model_json.go` — local round-trip; add the new field.
- `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` — rewrite `persistsAtFinalState` → `persistsAtEndState(eventType int32)`; capture matched event type in `Hit`; wire both call sites.
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` — add persist-rule cases.
- `services/atlas-reactors/atlas.com/reactors/reactor/timer.go` — NEW; state-timer mechanism modeled on `item_reactor.go`'s `pendingActivations`.
- `services/atlas-reactors/atlas.com/reactors/reactor/timer_test.go` — NEW.
- `services/atlas-reactors/atlas.com/reactors/reactor/item_reactor.go` — already has the pattern we mirror (lines 26-29, 95-136, 161-179).
- `services/atlas-reactors/docs/domain.md:94-147` — rewrite state-transition section; drop type-999 language.

## Key invariants & gotchas

- `data.Model` is immutable — accessors only. New methods live on `data.Model` (value receiver, same as the others).
- `GetRegistry().Update(t, id, modifier)` is the only legal way to mutate a reactor. Takes `func(*ModelBuilder)`.
- `persistsAtEndState` takes a single `int32` event type (matched event from the triggering path) — not a stateInfo map.
- Timer callback MUST re-fetch the reactor (it may have been hit/destroyed between arming and firing) and compare `r.State()` before transitioning. If state changed, bail.
- Use `time.AfterFunc` + `sync.Mutex`-guarded `map[uint32]*time.Timer`, exact same shape as `item_reactor.go:26-29`.
- Cancel points: `Destroy` (before Remove), `Hit` (first thing, before reading current state), `Teardown` (via `cancelAllStateTimeouts` called alongside `CancelAllPendingActivations`).
- Arm points: `Create` (after `GetRegistry().Create`), and the nextState branches of `Hit` and timer-fire.
- For timer-fire transitions, use `characterId=0` — the Trigger command flows to atlas-reactor-actions which accepts 0 as "no player."
- `int8` for states, `uint32` for reactor IDs, `int32` for event type. Don't cross wires.

## Test strategy

- atlas-data: pure function reader tests, XML fixture driven — fast, no Redis.
- atlas-reactors persist-rule: extend existing processor_test.go. Uses miniredis (`setupTestRegistry`). Construct `data.Model` via `data.Extract(data.RestModel{...})`; reactors via `createTestReactor` then `GetRegistry().Update` to plant state/data.
- atlas-reactors timer: use very short delays (e.g. 50ms) + a small `time.Sleep(100*time.Millisecond)` to let `AfterFunc` fire. Deterministic enough for the behaviors we're testing; no fake clocks needed.

## What's explicitly out of scope

- `activateByTouch` (9 GPQ reactors) — deferred to docs/TODO.md.
- Redesign of atlas-maps's 10-second spawn tick.
- Moon Bunny (9101000) will remain broken post-change (no events AND no timeOut in its wz). Documented in design.md section "Risks & edge cases (4)" and will be called out in domain.md.
- The Kafka event envelope, tenant handling, and registry/Redis storage layout — unchanged.

## Build & verify commands

- Unit tests, atlas-data: `cd services/atlas-data/atlas.com/data && go test ./reactor/...`
- Unit tests, atlas-reactors: `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...`
- Full service build: `cd services/atlas-reactors/atlas.com/reactors && go build ./...`
- Docker build (when shared-lib changes land — not expected here): see per-service Dockerfile under each `services/atlas-*/`.

## Phasing (5 phases, 9 tasks)

1. atlas-data: stop synthesising + read both timeout casings + extract TimeoutNextStateInfo. Pure data layer — no behavior change downstream yet because atlas-reactors still uses the old rule. Safe to land independently.
2. atlas-reactors data model: add `TimeoutNextStateInfo` through RestModel / Extract / Model / MarshalJSON; add `Timeout()` and `TimeoutNextState()` accessors. No behavior change yet.
3. atlas-reactors persist rule: replace `persistsAtFinalState` with `persistsAtEndState(eventType)` + rewire `Hit`. This flips the behavior for reactor 2001 — now it destroys correctly.
4. Timer mechanism: new file + wire into Create/Hit/Destroy/Teardown. Type-101 reactors now cycle.
5. Docs: rewrite `services/atlas-reactors/docs/domain.md` state-transition section.
