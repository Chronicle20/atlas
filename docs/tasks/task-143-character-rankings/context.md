# task-143 Character Rankings — Implementation Context

Companion to `plan.md`. Key files, decisions, and gotchas an implementer needs; all paths repo-relative.

## What is being built

1. **atlas-rankings** (new service, `services/atlas-rankings/atlas.com/rankings/`, module `atlas-rankings`) — leader-gated 60s ticker scans atlas-character per tenant, computes per-world overall + job-category rankings, upserts `character_rankings` / prunes stale rows, records `ranking_cycles`; serves `GET /api/rankings/characters?ids=…` (bulk) and `GET /api/rankings/characters/{id}` (single, 404 when absent). No Kafka at all.
2. **atlas-tenants** — new `rankings` configuration resource (generic JSONB `configurations` table, single-object `{"data": {…}}` variant of the routes/vessels pattern) carrying `recomputeIntervalMinutes`; GET/POST/PATCH/DELETE at `/tenants/{tenantId}/configurations/rankings`.
3. **atlas-login** — `character.Model` gets real `rank/rankMove/jobRank/jobRankMove` fields (replacing hardcoded-zero getters at `character/model.go:55-69`); `GetForWorld` applies a slice-level bulk rankings decoration (one call per char list, 2s timeout, fail-open to zeros). The packet writer needs **no change** — `socket/writer/character_list.go:61` already reads the getters.

## Pattern sources (copy these, don't invent)

| Need | Source |
|---|---|
| Service main.go shape (DB + REST, no Kafka) | `services/atlas-fame/atlas.com/fame/main.go` |
| Leader-gated ticker registration + env helpers | `services/atlas-monsters/atlas.com/monsters/main.go:100-132`, `leaderconfig.go` |
| Ticker registry (`Task` iface + `Register`) | `services/atlas-monsters/atlas.com/monsters/tasks/task.go` (copy verbatim) |
| Service-local `rest` package (HandlerDependency with DB) | `services/atlas-character/atlas.com/character/rest/handler.go` |
| Logger | `services/atlas-fame/atlas.com/fame/logger/init.go` (copy verbatim) |
| Tenant-enumeration REST client | `services/atlas-transports/atlas.com/transports/tenant/{rest,requests,processor}.go` (copy verbatim) |
| GORM resource handlers (Transform + MarshalResponse) | `services/atlas-character/atlas.com/character/character/resource.go:39-59` |
| Tenants config resource precedent (all touch points) | `configuration/rest.go` (RouteRestModel), `resource.go:88-131,637-670`, `processor.go:113+` (CreateRoute), `provider.go` (GetAllRoutesProvider), `kafka.go`, `mock/processor.go` |
| Base k8s manifest + probe | `deploy/k8s/base/atlas-gachapons.yaml` (shape), `deploy/k8s/base/atlas-world.yaml:31-37` (readinessProbe `/api/readyz`) |
| Bruno collection | `services/atlas-gachapons/.bruno/` |

## Load-bearing decisions (from design.md — do not relitigate)

- **REST scan, not Kafka projection.** One logical `GET /characters` per tenant per cycle. **Correction (found during implementation, task-143):** this endpoint is in fact **paginated** (`handleGetCharacters` → `paginate.ParseParams(..., DefaultPageSize=50, ...)`), not unpaginated as originally written here. The rankings character client therefore drains all pages via `requests.DrainProvider` (page size 250) rather than issuing a single unpaged GET — otherwise only the first 50 characters per tenant would be ranked. `PagedGetRequest` propagates the tenant/span headers, so the historical task-117 header-drop bug does not apply.
- **Config re-read every tick**, tenants re-enumerated every tick — never boot-time snapshots (avoids the atlas-transports staleness class). 404/error/zero → default 60 min.
- **Leader election via libs/atlas-lock** (`lock.New(rc, "rankings-recompute", …)`), 2 replicas. Single writer matters for correctness: concurrent recomputes would compute `move = prev − new` against each other's half-written rows. Crash mid-cycle is fine — the cycle is idempotent/convergent; do NOT rely on `ExecuteTransaction` (it is a no-op, task-119).
- **GM rule is `gm > 0`** (storage-level, `services/atlas-character/atlas.com/character/character/entity.go` `GM int`), NOT login's `gm == 1`.
- **Job move is computed against the previous job rank regardless of category change** (simplified FR-5 semantics, owner-approved).
- **Move signedness across the wire:** server-side moves are `int32`; login getters return `uint32(int32)` two's-complement pass-through (packet lib fields are `uint32`; v83 client does abs + sign-branch — IDA-verified at `0x60292F`).
- **Rank/JobRank derivation:** sort each world once by `level DESC, experience DESC, characterId ASC`; overall rank = position; job rank = running per-category counter over the same order (relative order within a category is preserved by restriction).

## Gotchas that will bite

- **jsonapi relationship stubs** (`SetToOneReferenceID`/`SetToManyReferenceIDs`) are REQUIRED on the atlas-rankings character client RestModel — atlas-character responses carry `relationships`; api2go errors without them and it surfaces as a bogus "not found" (libs/atlas-rest/CLAUDE.md). The plan's httptest fixture includes a relationships block specifically to catch this.
- **Tenant filtering is automatic** via `database.Connect` GORM callbacks. Providers take no tenantId; only creates set it; sqlite tests must call `database.RegisterTenantCallbacks` themselves. `pruneBefore` relies on the delete callback — always call through `db.WithContext(ctx)`.
- **`requests.GetRequest` takes no configurators.** The login-side 2s timeout must go through `requests.MakeGetRequest[…](url, spanDecorator, tenantDecorator, requests.SetTimeout(2*time.Second))` inside a hand-rolled `requests.Request` closure (see plan Task 12).
- **Readiness probe** must target `/api/readyz` (MountReadiness mounts under `SetBasePath("/api/")`) — bare `/readyz` wedges rollouts (task-090 postmortem).
- **docker-bake.hcl `go_services` is hand-synced** with `.github/config/services.json` — update BOTH, alphabetically (`atlas-rankings` between `atlas-quest` and `atlas-rates`); same slot in `go.work` and `deploy/k8s/base/kustomization.yaml`.
- **No `RANKINGS_SERVICE_URL` in env-configmap** — hardcoded `*_SERVICE_URL` in base breaks env overlays (npc-shops precedent); `requests.RootUrl("RANKINGS")` falls back to `BASE_SERVICE_URL`.
- **atlas-tenants mock must be updated** in the same commit as the Processor interface (compile-time check `var _ configuration.Processor = (*ProcessorMock)(nil)` breaks the build otherwise).
- **`go mod tidy` only after imports exist**; never `go work sync`. Run repo-root guard scripts (`tools/redis-key-guard.sh`) WITHOUT a `GOWORK=off` prefix.
- **db-name-suffix patches** go in BOTH `deploy/k8s/overlays/main` (`atlas-rankings-main`) and `deploy/k8s/overlays/pr` (`atlas-rankings-PLACEHOLDER_ATLAS_ENV`); container name is `rankings`.
- After editing `deploy/shared/routes.conf`, run `./deploy/scripts/sync-k8s-ingress-routes.sh` and commit its output.

## Data model (final)

`character_rankings`: uuid surrogate PK, unique `(tenant_id, character_id)`, index `(tenant_id, world_id)`; `world_id world.Id`, `job_category uint16` (= jobId/100 at compute time), `overall_rank`/`job_rank uint32` (1-based, never 0), `overall_rank_move`/`job_rank_move int32`, `computed_at`. Upsert = `ON CONFLICT (tenant_id, character_id) DO UPDATE` (works on sqlite for tests); prune = `DELETE … WHERE computed_at < cycleTime` (tenant-scoped).

`ranking_cycles`: one row per tenant (unique `tenant_id`): `last_started_at` (drives IsDue), `last_completed_at`, `characters_ranked`, `duration_ms`.

## Dependencies / order

Plan tasks 1→8 are sequential within atlas-rankings (each compiles + tests green). Task 9 (deploy) needs 1–8. Task 10 (atlas-tenants) is independent of 1–9. Tasks 11→12 (atlas-login) are sequential; 12 needs 11 only (it talks to atlas-rankings over the wire, so no code dependency on tasks 1–9). Task 13 sweeps everything; `superpowers:requesting-code-review` runs BEFORE any PR.

## Deferred / out of scope

Rank commands, UI leaderboards, real-time updates, last-login-aware move carry-over, fame/meso tiebreaks, packet changes. v95 zero-rank rendering is unverified (v83 verified) — check opportunistically during manual acceptance if a v95 tenant is in the test matrix; zeros are the fail-open value regardless.
