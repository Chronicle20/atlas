# Task-118 Shared Service Bootstrap — Context

Companion to `plan.md`. Key files, locked decisions, dependencies, and the gotchas an implementer must not rediscover the hard way.

## Key files

| File | Role |
|---|---|
| `libs/atlas-service/teardown.go` | Existing `Manager`/`GetTeardownManager` singleton — Bootstrap wraps it, does NOT change it |
| `libs/atlas-service/bootstrap.go` (new) | `Bootstrap`, `Option`, `Runtime`, readiness controller |
| `libs/atlas-service/logger.go` (new) | `CreateLogger` — canonical body from the 53-way-identical service copies |
| `libs/atlas-service/fieldnorm.go` (new) | snake_case emit-time normalizer hook (CP-9) |
| `libs/atlas-service/projection.go` (new) | `Projection` iface, `ProjectionFuncs`, `WithConfigProjection`, `AwaitProjectionCatchUp`, `parseProjectionCatchupTimeout` |
| `libs/atlas-kafka/producer/provider.go` (new) | `Provider` + `ProviderImpl` — verbatim move of the 51-way-identical wrapper |
| `libs/atlas-kafka/producer/producer_test.go` | Has `MockWriter` fake — reuse it in `provider_test.go` |
| `libs/atlas-rest/server/server.go:34` | `MountReadiness(path, fn)` — existing contract, unchanged |
| `libs/atlas-tracing/tracing.go` | `InitTracer(name) (*sdktrace.TracerProvider, error)`, `Teardown(l)(tp) func()` |
| `services/atlas-fame/atlas.com/fame/main.go` | Pilot service; the canonical "Cohort A" shape |
| `services/atlas-{login,channel,world,character-factory}/.../main.go` | The 4 projection services (Cohorts D1/D2) |
| `services/atlas-renders/atlas.com/renders/main.go` | The one non-standard service (raw mux, no tracer, `/healthz`) |
| `services/atlas-merchant/atlas.com/merchant/service/teardown.go` | Private teardown-manager copy — delete, rewire imports to the lib |
| `deploy/k8s/base/atlas-{world,character-factory,renders}.yaml` | Live probe paths — must remain valid; NO manifest edits in this task |

## Locked decisions (from design.md)

- **D1** Producer move is verbatim; return type becomes the named `Provider` (identical underlying type, assignable everywhere). No behavior change.
- **D3** snake_case via emit-time hook, registered LAST in `CreateLogger`. Collision rule: explicit snake_case key wins, camelCase duplicate dropped, deterministic via sorted rename order. Safe to mutate `entry.Data` in place — logrus v1.9.4 `entry.Dup()` fires hooks on a per-emission copy.
- **D4** Functional options + `Runtime` handle. Bootstrap owns logger/tdm/tracer/readiness/projection; `main.go` keeps DB/Redis/consumers/REST builder/tasks/producer-close teardown.
- **D5** Readiness mounts via one explicit `MountReadiness("/readyz", rt.Ready)` line per main.go under `SetBasePath("/api/")` → effective `/api/readyz`. NO root-mount relocation (that's OPS-3, decided with OPS-1's manifests).
- **D6** Lib owns projection *choreography* only, via the 2-method `Projection` interface + `ProjectionFuncs` adapter; the 4 service-local `configuration/projection` packages are NOT extracted. Catch-up gate position stays with main.go (`rt.AwaitProjectionCatchUp()`); readiness gates are service-supplied (`caughtUp.CaughtUpNow` ×3, `configuration.SnapshotReady` for world).
- **D7** atlas-renders: `WithoutTracer()`, keeps raw mux + `/healthz`, gains root `/readyz` + graceful shutdown. Log format change to ecslogrus is the accepted observable change.
- Accepted micro-changes (must be called out in Cohort D commit messages): projection subscriber starts earlier (inside Bootstrap, before consumer registration); warn condition unifies to "tenant topic unset".

## Dependencies & sequencing

- **HARD GATE (Task 0):** task-114 (outbox adoption) and task-116 (processor gen3 unification) must be merged to main, and this branch rebased, before any code task. Both rewrite producer wiring / main.go surface. All measured shapes expire at rebase; re-measure and re-derive verbatim bodies.
- `libs/atlas-service` gains deps: logrus v1.9.4, ecslogrus v1.0.0, google/uuid v1.6.0, `libs/atlas-tracing` (require + `replace ../atlas-tracing`). No import cycle (atlas-tracing → otel+logrus only).
- No Dockerfile or go.work edits: both libs already have COPY lines and go.work entries.
- Lib tasks (1–5) strictly before any migration; pilot (6) before cohort sweeps (7–12); docs (13) and verification (14) last.

## Gotchas (measured 2026-07-02; several are design-doc corrections)

1. **57 local logger packages, not 56**: atlas-monster-book's is `logger/logger.go`, byte-identical (md5 `473b31e275b2900d442a9915fb6a095a`). Acceptance grep must match both filenames.
2. **`services/atlas-cashshop/.../cashshop/inventory/rest_test.go`** imports the local logger — the only non-main importer fleet-wide. Rewrite to `service.CreateLogger` in cashshop's commit.
3. **atlas-storage's `projection` package is storage-domain**, not config projection. Cohort A, no projection option.
4. **Replace-directive semantics**: `replace` lines are only honored from the main module. Services must KEEP their `atlas-tracing` require+replace even after main.go stops importing tracing (atlas-service depends on it). Do NOT `go mod tidy` services — except atlas-renders, which genuinely gains deps.
5. **atlas-renders' `tenantMiddleware`** rejects requests without tenant headers; `/readyz` must be added to the `/healthz` bypass or probes get 400s.
6. **Dual-import files** (~20+, worst in atlas-saga-orchestrator, e.g. `saga/producer.go` with `kproducer` alias): after the R1 sed they become duplicate imports — keep one `producer` import, rename alias uses.
7. **atlas-quest's wrapper is `Provider`-type-only** (no `ProviderImpl`); migration must not force `ProviderImpl` adoption (FR-1.4).
8. **`zz_lifecycle_test.go` ordering**: the SIGTERM lifecycle test closes the process-wide teardown-manager singleton; the `zz_` filename makes it run last in the package. Don't add later-sorting test files to `libs/atlas-service`.
9. **Normalizer hook ordering caveat**: keys added by hooks registered after the normalizer escape normalization. `CreateLogger` registers it last; zero `AddHook` calls exist outside the (deleted) `logger/` packages.
10. **Probe-path trap** (`bug_readiness_probe_path_under_api_basepath`): effective path is `/api/readyz`, and atlas-world/atlas-character-factory manifests already probe exactly that. This task must not change any effective URL.
11. **logrus `Fatal` testing**: set `rt.Logger().ExitFunc` to capture exit instead of letting the test binary die (used in the projection timeout test).
12. **Group-id pattern is load-bearing**: `"<base> - projection - <uuid>"` per process forces FirstOffset replay of the compacted config log; a shared group id would leave the projection state empty on restart.

## Cohort map (58 services)

- **Pilot:** fame.
- **Cohort A (44):** account, asset-expiration, ban, buddies, buffs, cashshop, chairs, chalkboards, character, consumables, data, doors, drops, effective-stats, expressions, families, guilds, inventory, invites, keys, map-actions, maps, marriages, messages, messengers, monster-book, monster-death, monsters, mounts, notes, npc-conversations, npc-shops, parties, party-quests, pets, portal-actions, portals, reactor-actions, reactors, skills, storage, summons, tenants, transports.
- **Cohort B (5, no producer wrapper):** configurations, drop-information, gachapons, query-aggregator, rates.
- **Cohort C (3):** quest, merchant, saga-orchestrator.
- **Cohort D (4):** world, character-factory (D1); login, channel (D2).
- **Cohort E (1):** renders.

## Post-rebase measurements

(Filled in by Task 0 at execution time.)

## Acceptance evidence

(Filled in by Task 14 at execution time.)
