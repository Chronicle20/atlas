# task-278 — Planning Context

Companion to `plan.md`. Records the discovery findings the plan assumes, the
places the plan deliberately departs from `design.md` and `prd.md`, and the
sizing decisions.

## Key files, by seam

| Seam | File | Why it matters |
|---|---|---|
| Shared enum home | `libs/atlas-constants/field/constants.go` (7 lines today: `IdFormat`, `type Id string`) | `ObjectKind` has no existing definition anywhere; per repo convention the shared enum belongs here, not in a service |
| Saga contract | `libs/atlas-saga/model.go:285-290`, `payloads.go:1300-1310`, `unmarshal.go:636-647` | `FieldEffectWeather` / `PlayJukebox` are the exact precedent pair; new arms go immediately after |
| Registry precedent | `services/atlas-maps/atlas.com/maps/map/weather/registry.go:1-57` | Singleton `once.Do` + `FieldKey{Tenant, Field}` + RWMutex. `map/jukebox/` is a byte-for-byte clone; `map/jukebox/registry_test.go` is the registry-test precedent |
| Maps command consumer | `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:32-44` (`InitHandlers`), `:46-75` (`handleWeatherStartCommand`) | Registration and handler shape copied verbatim |
| Maps REST | `services/atlas-maps/atlas.com/maps/map/weather/resource.go:22-57`; `rest/handler.go:24,30,34-48` | `ParseWorldId/ChannelId/MapId/InstanceId` and `ParseInput`/`RegisterInputHandler` all exist as design assumed |
| Exit funnel | `services/atlas-maps/atlas.com/maps/map/processor.go:105-110` | `Exit` is the single funnel for logout, warp, and channel change |
| Orchestrator handler | `services/atlas-saga-orchestrator/.../saga/handler.go:3715-3742` | `handleFieldEffectWeather` — type-assert, `logActionError`, `StepCompleted` |
| Channel REST client | `services/atlas-channel/atlas.com/channel/weather/{processor,requests,rest}.go` + `mock/` | Four-file shape reproduced for `environment/`; only difference is `SliceProvider` instead of `Provider` |
| Channel announce seam | `services/atlas-channel/.../kafka/consumer/map/consumer.go:769-779` (`var doorAnnounce = func(...)`) | A package-level `var`, already stubbed by tests at `consumer_test.go:254-262`. Every new emit routes through it so writer selection is testable without a socket |
| Enter replay site | `services/atlas-channel/.../kafka/consumer/map/consumer.go:358-383` | The three existing `routine.Go` blocks in `SpawnForSelf`; the new block goes after `announceActiveJukebox` |
| Writer wrappers | `services/atlas-channel/atlas.com/channel/socket/writer/{set_object_state,field_obstacle_on_off,field_obstacle_on_off_list,field_obstacle_all_reset}.go` | All four exist and are used as-is; registered at `main.go:835,836,871,872` |
| Script executors | `services/atlas-reactor-actions/.../script/executor.go:220-243,278-292,308-314`; `services/atlas-map-actions/.../script/executor.go:35-50,63-88` | Saga-builder shape and missing-param error style |

## Corrections to `design.md` carried into the plan

1. **§3.4 mislabels the teardown seam.** There is no `CHARACTER_EXIT` handler in
   `services/atlas-maps/.../kafka/consumer/character/consumer.go`. The cited
   `consumer.go:144` is inside `handleStatusEventLogoutFunc`, and the method is
   `_map.Processor.ExitAndEmit` (`map/processor.go:112-116`), **not**
   `mapcharacter.Processor.ExitAndEmit` — no such method exists on
   `map/character.Processor`.

   The plan puts the empty-field check inside `_map.ProcessorImpl.Exit`
   (`map/processor.go:105-110`) instead. `Exit` is the single funnel that
   `ExitAndEmit`, `TransitionMap`, and `TransitionChannel` all route through, so
   one edit covers logout, warp, and channel change; patching the logout handler
   would have covered only one of the three. This also delivers the design's own
   stated behaviour ("a field that empties transiently loses its state") for the
   PQ stage-warp case, which a logout-only hook would not.

2. **§3.4's `ExitAll` change is bigger than "return the affected keys."**
   `ExitAll` and its backing `RemoveCharacterFromAllMaps` are both `void` today
   (`map/character/processor.go:66-69`, `registry.go:105-114`), so both change.
   `ExitAll` has exactly one caller repo-wide
   (`kafka/consumer/character/consumer.go:194`), so the blast radius is two
   files plus the interface.

3. **§3.5's Kafka-manifest contingency resolves to "no manifest."**
   `task-276-kafka-topic-manifest` has not landed: no `docs/tasks/task-276-*`
   directory exists and no `*manifest*` file enumerates Kafka topics. Topic
   message types live in per-service `docs/kafka.md`. Task 14 updates those.

4. **§6.3 cites the map-actions switch at `executor.go:36-52`; it is at
   `:35-50`.** Cosmetic, but the plan's `### Files` entries use the real range.

5. **The map message types are duplicated three times**, not once:
   `services/atlas-maps/.../kafka/message/map/`,
   `services/atlas-saga-orchestrator/.../kafka/message/map/`, and
   `services/atlas-channel/.../kafka/message/map/` each keep their own copy.
   The design describes only the `atlas-maps` copy. Tasks 4, 7, and 10 each add
   their service's copy, and Task 14's review step explicitly checks the three
   are byte-identical on the wire. This is the highest-risk seam in the change.

   The three copies are not even laid out the same way: `atlas-maps` splits
   commands into `command.go` and events into `kafka.go`; the orchestrator and
   the channel each keep everything in a single `kafka.go`. `plan-lint` caught
   this — the plan's first draft named a `command.go` under the orchestrator
   that does not exist.

6. **`weather.Registry.Get` does not copy** (`registry.go:46-51` returns the
   value directly). That is safe for `WeatherEntry` (a scalar struct) and unsafe
   for `[]ObjectEntry` (a slice header). The environment registry's copy-on-read
   is new behaviour, not a pattern inherited from weather, so Task 3 tests it
   explicitly.

## Corrections to `prd.md` already made by `design.md`, restated

- **FR-15 is wrong and is not implemented.** `gms_48_1` routes no
  `FieldObstacleOnOff` at all and `gms_61_1` routes neither
  `FieldObstacleOnOff` nor `FieldObstacleAllReset`, so "on GMS<61 send one
  `FieldObstacleOnOff` per obstacle" is unimplementable. The plan uses the
  writer-availability fallback to `SetObjectState` (design.md §1.3). The PRD
  acceptance criterion that reads "A GMS<61 tenant receives per-obstacle
  `FieldObstacleOnOff` packets on bulk replay" is therefore **not** satisfiable
  as written; Task 11's `TestAnnounceEnvironmentState_NoObstacleWritersAtAll` is
  the honest version of it.
- **FR-12's `ENVIRONMENT_RESET` body is not empty.** It carries the cleared
  objects, because `FieldObstacleAllReset` restores only the client's obstacle
  list (design.md §1.2) and the channel keeps no registry of its own.
- **PRD §5's `instance` query parameter is a path segment.** All three routes
  take `/instances/{instanceId}/environment`.
- **OQ-3 resolved as "no bulk status event."** No `ENVIRONMENT_STATE_LIST` type
  is added. `FieldObstacleOnOffListWriter` gets its emitting call site from the
  enter replay (Task 11) instead.

## Task sizing

14 tasks. No task exceeds 6 files or crosses a service boundary. Ordering
dependencies:

```
Task 1 (constants)
  └─ Task 2 (atlas-saga)
       ├─ Task 3 (maps registry/processor/rest)
       │    ├─ Task 4 (maps kafka contract + consumer)
       │    │    └─ Task 5 (maps REST resource)
       │    └─ Task 6 (maps teardown)
       ├─ Task 7 (orchestrator map_command)
       │    └─ Task 8 (orchestrator saga wiring)
       ├─ Task 12 (reactor-actions)
       └─ Task 13 (map-actions)
Task 5 ─ Task 9 (channel REST client)
Task 4 ─ Task 10 (channel broadcast)   [needs Task 4's wire strings]
Task 9 + Task 10 ─ Task 11 (channel enter replay)
everything ─ Task 14 (docs + gate + review)
```

Tasks 3–6 are all `atlas-maps` and could in principle run back to back in one
context; they are split because each has an independently reviewable deliverable
(registry semantics, wire contract, REST surface, lifecycle hook) and the review
loop is per task.

No task was deliberately left large.

## Things the implementer must resolve, not guess

- **Task 9, `requests.MakeGetRequest[[]RestModel]` is a placeholder name.**
  `requests.SliceProvider` needs a `Request[[]A]`, and `requests.GetRequest[M]`
  returns `Request[M]`. The plan instructs the implementer to grep an existing
  `SliceProvider` caller and copy its request constructor rather than invent
  one. This is the one unresolved external-API signature in the plan.
- **Task 12/13, `operation.Model`'s constructor.** `operation.Model` lives in
  `libs/atlas-script-core/operation` (reached via the `replace` directive at
  `services/atlas-reactor-actions/.../go.mod:99`) and exposes only
  `Type() string` and `Params() map[string]string`. Neither script package has
  an existing test file, so the constructor for a test `operation.Model` must be
  found in that library.
- **Task 7, a `map_command.Processor` mock may exist** under
  `services/atlas-saga-orchestrator/`. Adding two interface methods breaks it if
  so; the plan says to grep for it.
- **Task 6, `_map.ProcessorImpl`'s field names.** The plan's snippet uses
  `p.l` and `p.ctx`; `NewProcessor(l, ctx, p, db)` implies both exist, but the
  struct should be read before editing.

## Registry singleton hazard

`map/environment`'s registry is a process-wide singleton reached through
`once.Do`, exactly like weather and jukebox. Every test in Tasks 3, 4, 5, and 6
must create a **fresh** tenant with `tenant.Create(uuid.New(), "GMS", 83, 1)` so
tests cannot leak state into one another. The plan states this in each affected
task; it is the most likely source of a flaky failure in this change.
