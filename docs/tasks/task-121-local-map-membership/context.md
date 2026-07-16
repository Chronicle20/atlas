# task-121-local-map-membership — Implementation Context

Companion to `plan.md`. Summarizes the key files, locked decisions, and dependencies an implementer needs; the authoritative rationale is `design.md`.

## What this task does

Removes the synchronous REST call to atlas-maps from atlas-channel's broadcast recipient-resolution path (finding PS-2, `docs/architectural-improvements.md`). Recipients are resolved by filtering the local in-process session registry by `field.Model` — every local session already carries its authoritative field, and delivery is only ever possible to local sessions anyway.

## Key files

| File | Role |
|------|------|
| `services/atlas-channel/atlas.com/channel/map/processor.go` | The swap site. `CharacterIdsInMapModelProvider` (:31) and `CharacterIdsInMapAllInstancesModelProvider` (:49) currently use `requests.SliceProvider` → HTTP. All other methods compose over these two and are untouched. |
| `services/atlas-channel/atlas.com/channel/map/requests.go`, `map/rest.go` | REST plumbing, deleted in Task 5 once unreferenced. |
| `services/atlas-channel/atlas.com/channel/session/processor.go` | Gains `InFieldModelProvider` / `InMapAllInstancesModelProvider` (Task 1). `SetField` (:237) sets map+instance; `SetMapId` (:226) is deleted in Task 2. `Create` (:287) fixes world/channel at session creation. |
| `services/atlas-channel/atlas.com/channel/session/registry.go` | `GetInTenant` (:76) — RLock snapshot returning value copies; the only shared-state access the new providers use. |
| `services/atlas-channel/atlas.com/channel/session/model.go` | `Field()` (:180) returns the session's `field.Model`; `WorldId/ChannelId/MapId/Instance` delegate to it. `Model.setMapId` (:152) stays (used by `SetField`). |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:190` | The FR-1.2 gap: login bootstrap calls `SetMapId(s.SessionId(), f.MapId())`, dropping the instance from `location.GetField`. Fixed to `SetField(s.SessionId(), f)`. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:249` | The `MAP_CHANGED` consumer — already correct: `SetField(sessionId, targetField)` before the warp packet and spawn. Referenced by the audit doc, not edited. |
| `services/atlas-channel/atlas.com/channel/session/registry_test_helper.go` | Exported test helpers `AddSessionToRegistry` / `ClearRegistryForTenant` — the sanctioned way tests populate the registry. |
| `services/atlas-channel/atlas.com/channel/test/context.go` | `DefaultTenantId`, `CreateTestContext`, `CreateTestContextWithTenant`, `CreateDefaultMockTenant` (GMS v83). |
| `libs/atlas-constants/field/model.go` | `Equals` (:92, includes instance), `SameMap` (:101), `NewBuilder` (:163). |

## Locked decisions (do not relitigate)

1. **Linear scan, no field-keyed index** (design §2.1-A vs B). Scan of `GetInTenant` snapshot is O(sessions-in-tenant) with trivial comparison — negligible at design scale. The provider seam means an index could later be added inside `session` with zero caller churn.
2. **Filter lives in the `session` package** as `model.Provider`-returning methods (design §2.2); registry internals stay encapsulated.
3. **No shadow verification** (design §2.3): FR-4 tests + staging playtest; keeping REST alive for comparison would contradict FR-3.2.
4. **Every `map.Processor` exported signature is frozen** — 32 caller files must compile unchanged. The id→session delivery indirection (`ForEachByCharacterId` re-resolving ids) is deliberately retained (design §3.2).
5. **Character-id dedup is mandatory** in the new id providers: the registry can transiently hold two sessions for one character (stale socket + reconnect); atlas-maps never returned duplicates.
6. **`Processor.SetMapId` is deleted** after the bootstrap fix (zero non-test callers remain); its two tests (`TestSetMapId`, `TestSetMapId_NonExistent`) are replaced with `SetField` equivalents — the instance-preservation test doubles as the FR-1.2 regression guard.
7. **Bootstrap fix is a prerequisite**, not a side quest: without it, exact-field matching would exclude characters who logged in inside an instanced map from instance-addressed broadcasts (design §4.1).

## Dependencies and gotchas

- **Module:** `services/atlas-channel/atlas.com/channel`, module name `atlas-channel`. Map package is named `_map` (import alias `_map "atlas-channel/map"`); in its test file import the constants package under a different alias (plan uses `mapid`).
- **Test pattern:** `session.NewSession(id, tenant, 0, nil)` (nil conn is fine — nothing in these paths writes to it) + `AddSessionToRegistry` + processor mutators. Sessions created this way sit at world 0 / channel 0; world/channel discrimination is tested by querying a *different* world/channel, not by mutating the session (no exported world/channel setter exists outside `Create`).
- **`SetField` sets map + instance only** — world/channel are fixed at `Create`. Fine for both production paths and tests.
- **Test files reference internals** (project gotcha): deleting `SetMapId` breaks `TestSetMapId`/`TestSetMapId_NonExistent` — Task 2 handles both sides in one commit.
- **`field.Builder` also has a `SetMapId` method** (`session/model.go:154` calls it inside `Model.setMapId`) — a bare grep for `SetMapId` legitimately keeps one hit there after Task 2.
- **Semantic deltas are favorable, not regressions** (design §5 / NFR-3): registry is fresher than atlas-maps' async projection; the whole-broadcast abort on a stale id (`SliceMap` fails the entire mapping on first `not found`, `libs/atlas-model/model/processor.go:419`) shrinks to the resolution→delivery window; REST failure mode disappears.
- **Out of scope:** PS-1/PS-3/PS-4, any atlas-maps change, the `MAP_STATUS` consumer, ENTER/EXIT emission, index optimization, pipeline collapse.

## Verification gate (CLAUDE.md)

From the module: `go test -race ./...`, `go vet ./...`, `go build ./...`.
From the worktree root: `docker buildx bake atlas-channel`, `tools/redis-key-guard.sh`.
Deliverable doc: `docs/tasks/task-121-local-map-membership/field-transition-audit.md` (FR-1.3, Task 6).
Playtest per PRD acceptance happens at review/PR time (needs a deployed build).
