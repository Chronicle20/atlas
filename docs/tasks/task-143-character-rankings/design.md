# Login-Screen Character Rankings — Design

Task: task-143-character-rankings
Status: Approved for planning
PRD: `docs/tasks/task-143-character-rankings/prd.md`
Created: 2026-07-09

---

## 1. Summary of Decisions

The PRD left four open questions. Grounded in the current source, this design resolves them:

| Question | Decision |
|---|---|
| Data acquisition (§9.1) | **REST scan** of atlas-character at compute time via the existing unfiltered `GET /characters` endpoint. No Kafka projection. task-117 is NOT a prerequisite. |
| Config shape (§9.2) | New **`rankings` configuration resource in atlas-tenants** (generic JSONB pattern, like `routes`/`vessels`), carrying `recomputeIntervalMinutes`. atlas-rankings **re-reads config every scheduler tick** — no config-status projection. |
| v95 zero-rank rendering (§9.3) | Deferred to implementation verification; not a design input. Zeros are the fail-open value regardless. |
| GM semantics (§9.4) | atlas-character stores `GM int` (default 0) — `services/atlas-character/atlas.com/character/entity.go:59`. **Exclude `gm > 0`** from rankings. (atlas-login's `Gm()` is `gm == 1` — `services/atlas-login/atlas.com/login/character/model.go:51` — but rankings uses the storage-level rule; a `gm >= 2` character simply gets no row → zeros → "Ranking not available", which is graceful.) |

Additional structural decisions:

- New service **atlas-rankings** modeled on **atlas-fame** (cleanest GORM + JSON:API DDD service), with the ticker pattern copied from atlas-expressions/atlas-monsters (`tasks/task.go`).
- **No Kafka at all in v1** — atlas-rankings neither consumes nor produces events. Character lifecycle (FR-6) is satisfied by the scan itself.
- Recompute is **leader-gated via `libs/atlas-lock`** (task-064; built exactly for gating ticker work to one replica) so the service can run the standard 2 replicas for REST availability without double-computing.
- atlas-login populates ranks via a **slice-level bulk decoration** in the character processor (one bulk call per `GetForWorld`, FR-8), failing open to zeros (FR-11).

## 2. Alternatives Considered

### 2.1 Data acquisition: REST scan vs Kafka projection

**Option A — REST scan at compute time (chosen).** Each cycle, atlas-rankings calls atlas-character's existing `GET /characters` (registered at `services/atlas-character/atlas.com/character/resource.go:28`, backed by `Processor.GetAll()` → `db.Find` with automatic tenant scoping, `provider.go:40-49`) once per tenant, filters and sorts in memory, and rewrites its rankings table.

**Option B — Kafka projection.** Maintain a local character snapshot from `EVENT_TOPIC_CHARACTER_STATUS` events (`CREATED`/`DELETED`/`LEVEL_CHANGED`/`EXPERIENCE_CHANGED`/`JOB_CHANGED`/`GM_CHANGED`), the way atlas-parties maintains its character registry (`services/atlas-parties/atlas.com/parties/character/registry.go`, consumer at `kafka/consumer/character/consumer.go:242-274`).

**Option C — Hybrid.** Projection for freshness + periodic REST reconciliation.

Why A wins:

1. **The projection cannot bootstrap.** The character status topic is delete-policy with default (~7d) retention (`deploy/k8s/base/atlas-kafka-precreate.yaml:42-53` creates all `EVENT_TOPIC_*` with no cleanup-policy override). Characters that exist but emit no events — exactly the long-tail population a leaderboard must still rank — would never enter a replay-built projection. Any projection therefore needs a REST backfill *and* a periodic REST reconciliation (missed `DELETED` events during downtime would otherwise leave stale ranked rows, violating FR-6). That reconciliation *is* Option A; B and C are A plus extra moving parts.
2. **Event gaps.** `StatusEventCreatedBody` carries only name/map (`kafka/message/character/kafka.go:253-257`) — no job/level/gm, so every CREATED needs a follow-up REST fetch; `GM_CHANGED` coerces the gm level to bool (`kafka.go:377-380`, producer `processor.go:1744-1747`), losing the numeric value.
3. **Load is acceptable.** One bulk read per tenant per hour (default cadence) against tens of thousands of rows is trivial for Postgres and far below the login-path traffic atlas-character already serves. Rankings freshness is hourly by requirement (non-goal: real-time updates), so the projection's only real advantage — freshness — buys nothing.
4. **No atlas-character changes needed.** `GET /characters` exists today. **Correction (found during implementation, task-143):** contrary to the original claim here, this endpoint **is paginated** (`handleGetCharacters` → `paginate.ParseParams(..., DefaultPageSize=50, ...)`). The rankings character client drains all pages via `requests.DrainProvider` (page size 250) so every character is ranked, not just the first 50. atlas-character itself still needs no change; task-117 is not a prerequisite (paged tenant/span headers already propagate).

Consequence to document in the service README: recompute cost scales with total tenant character count; the cycle is O(n log n) in memory and one full-table read on atlas-character per tenant per cycle.

### 2.2 Config: where the interval lives and how changes propagate

**Option A — atlas-tenants generic JSONB configurations resource (chosen).** New resource name `rankings` under the existing `configurations` table (`services/atlas-tenants/atlas.com/tenants/configuration/entity.go:11-22`), route `GET /tenants/{tenantId}/configurations/rankings`, following the `routes`/`vessels`/`instance-routes` precedent (`configuration/resource.go:647-668`). This is per-tenant by construction, matching FR-4, and `RouteRestModel`'s `CycleInterval` (`configuration/rest.go:8-20`) is the exact precedent for interval-typed attributes.

**Option B — atlas-configurations per-service config `tasks[]`** (the atlas-drops pattern, `services/atlas-drops/atlas.com/drops/main.go:65-94`). Rejected: that config is per-service, not per-tenant — it can't express "tenant X recomputes every 15 min, tenant Y hourly" without inventing a new shape.

**Option C — config-status Kafka projection** (task-090 pattern). Rejected for v1: it is copy-ported per service (explicit task-090 non-goal to lib-ify), adds a compacted-topic operational dependency that provisioning still doesn't guarantee (precreate job sets no `cleanup.policy=compact`), and buys push-latency we don't need — the scheduler ticks every minute anyway, so **re-reading the config on each tick** already satisfies "changes take effect without a service redeploy" (FR-4) with bounded staleness ≤ 1 tick.

Missing/404 config → default **60 minutes** (FR-4). No seeding required.

### 2.3 Scheduling: single-writer recompute

**Option A — leader-gated ticker via `libs/atlas-lock` (chosen).** Standard 2-replica deployment; the ticker body runs only while the pod holds the Redis lease. Precedent: atlas-lock exists specifically to gate ticker/sweep tasks to one replica (task-064).

**Option B — `replicas: 1`.** Simplest, no Redis dependency, but makes the REST endpoint a single point of unavailability during restarts and deviates from the standard manifest shape.

**Option C — hand-rolled DB claim** (conditional `UPDATE` on a cycle row). Works, but rolls our own lock when an audited lib already exists — rejected per the libs-first rule.

Why single-writer matters: the rank-move computation reads previous rows and rewrites them. Two concurrent recomputes interleaving would compute `move = prev − new` against each other's half-written state and zero out movement arrows. Correctness, not just efficiency.

Crash mid-cycle is still handled without the lock's help: the cycle is idempotent and convergent (see §3.4) — a re-run after a crash may reset some `*_move` values to 0 for one cycle, which the PRD's NFRs explicitly allow ("partial writes acceptable only if a re-run converges").

## 3. atlas-rankings Service Design

### 3.1 Layout

```
services/atlas-rankings/atlas.com/rankings/
├── main.go
├── logger/               (standard)
├── ranking/              core domain
│   ├── entity.go         character_rankings + ranking_cycles entities, Migration
│   ├── model.go          immutable Model + accessors
│   ├── builder.go        fluent Builder
│   ├── provider.go       DB read providers
│   ├── administrator.go  DB writes (batch upsert, prune, cycle rows)
│   ├── processor.go      Processor iface + Impl: compute, GetByCharacterIds, GetByCharacterId
│   ├── compute.go        pure ranking functions (sort/rank/move) — no side effects
│   ├── rest.go           JSON:API RestModel + Transform/Extract
│   └── resource.go       route registration + handlers
├── character/            REST client → atlas-character (ForeignRestModel: id, accountId, worldId, level, experience, jobId, gm)
│   └── requests.go       requests.RootUrl("CHARACTERS"), GET /characters
├── tenant/               REST client → atlas-tenants GET /tenants (atlas-transports precedent, services/atlas-transports/atlas.com/transports/tenant/requests.go:12-16)
├── configuration/        REST client → atlas-tenants GET /tenants/{id}/configurations/rankings; default on 404
└── tasks/                task.go (Task iface + Register, copied per convention from atlas-monsters/tasks/task.go:10-30) + recompute.go
```

`main.go` follows atlas-fame (`services/atlas-fame/atlas.com/fame/main.go`): teardown manager, tracer, `database.Connect(l, database.SetMigrations(ranking.Migration))`, REST server with `SetBasePath("/api/")`, plus `server.MountReadiness("/readyz", …)` (resolves to `/api/readyz` under the base path — the readiness-probe-path bug pattern) and the ticker registration `go tasks.Register(l, tdm.Context())(tasks.NewRecomputeTask(l, db, time.Minute))`. No Kafka consumer/producer registration.

Types: `world.Id`, `job.Id` from `libs/atlas-constants` (DOM-21). No locally invented job constants; the category is derived arithmetic (§3.4), stored as `uint16`.

### 3.2 Data model

Per PRD §6, one row per ranked character plus a small cycle-state table:

`character_rankings`
- `id` uuid PK (surrogate — never a natural PK; tenant-PK-collision pattern)
- `tenant_id` uuid, `character_id` uint32 — unique index `(tenant_id, character_id)`
- `world_id` (`world.Id`), `job_category` uint16
- `overall_rank` uint32, `overall_rank_move` int32, `job_rank` uint32, `job_rank_move` int32
- `computed_at` timestamp
- index `(tenant_id, world_id)` (future leaderboards)

`ranking_cycles` — one row per tenant: `id` uuid PK, `tenant_id` uuid (unique), `last_started_at`, `last_completed_at`, `characters_ranked` uint32, `duration_ms` uint32. Drives cadence ("is this tenant due?") robustly even when a tenant has zero eligible characters (MAX(computed_at) can't express that), survives restarts, and doubles as observability.

AutoMigrate on start; no legacy data. Tenant filtering is automatic via `database.Connect` GORM callbacks — providers never take tenantId; only creates set it (scaffolding rules, `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md:197-202`).

### 3.3 Scheduler (tasks/recompute.go)

The ticker fires every **60 s** (fixed base tick; the *tenant-visible* cadence is config-driven):

1. Acquire/verify leadership via atlas-lock; not leader → return.
2. `GET /tenants` from atlas-tenants **each tick** — never a boot-time snapshot (atlas-transports captures tenants once at `main.go:95` and never refreshes; that staleness class is exactly what task-090 fixed elsewhere — we avoid it by re-fetching, which also picks up newly provisioned tenants without redeploy).
3. Per tenant, build `ctx = tenant.WithContext(tdm.Context(), t)`, read the `rankings` config (404/error → default 60 min), read the tenant's `ranking_cycles` row, and if `now − last_started_at ≥ interval` (or no row) run the recompute.
4. Per-tenant failures are logged (`tenant`, `error`) and **skipped — never `log.Fatalf`** (crash-loop pattern; FR-7: one tenant's failure must not affect others).

Structured log per completed cycle: tenant, worlds, characters ranked, duration, per-world counts (NFR observability).

### 3.4 Recompute algorithm (ranking/compute.go — pure; processor orchestrates)

For one tenant:

1. `cycleTime := now`; write/claim the `ranking_cycles` row (`last_started_at = cycleTime`).
2. Fetch all characters: one `GET /characters` call (tenant headers from ctx via `TenantHeaderDecorator`, `libs/atlas-rest/requests/header.go:27-43`). Decode only needed attributes into the local `ForeignRestModel`.
3. Filter eligibility: drop `gm > 0` (FR-3 — excluded entirely, not counted).
4. Group by `world_id`. Per world:
   - **Overall rank:** sort `level DESC, experience DESC, characterId ASC`; assign 1-based positions. (With the characterId tiebreak the order is a strict total order, so "dense" and ordinal ranking coincide — every rank is unique; noted for FR-1.)
   - **Job rank:** `category := uint16(jobId / 100)` (Cosmic parity: 0=beginner…5=pirate; Cygnus 10–15, Aran 20–21 fall out of the same division for versions that have them). Within each `(world, category)` group, same sort, 1-based positions.
5. Load the tenant's existing `character_rankings` rows into `map[characterId]row`.
6. For each ranked character: `overall_rank_move = int32(prevOverall) − int32(newOverall)` (positive = up), same for job move **against the previous job rank regardless of category change** (FR-5 formula; a job-advanced character's move is computed across categories — semantics are "did my displayed number improve", matching the simplified PRD rule). No previous row → both moves 0.
7. Batch upsert with `ON CONFLICT (tenant_id, character_id) DO UPDATE`, all rows stamped `computed_at = cycleTime` (chunked, e.g. 500/batch).
8. Prune: `DELETE FROM character_rankings WHERE tenant_id = ? AND computed_at < cycleTime` — removes deleted characters and characters that became GM (FR-6: gone by the next recompute).
9. Complete the cycle row (`last_completed_at`, counts, duration).

**Idempotency/convergence (NFR):** every step is re-runnable. A crash between 7 and 8 leaves mixed `computed_at` values; the next run recomputes everything from a fresh scan and its prune uses the *new* cycleTime, so the table always converges to the latest scan. The only artifact of a re-run is that moves computed against half-updated rows read as 0 for one cycle — explicitly acceptable per PRD §8. Known repo caveat: `ExecuteTransaction` is currently a no-op (task-119), so the design deliberately does **not** rely on transactional atomicity for correctness — convergence does the work.

**Concurrent REST reads during a cycle** may see a mix of old and new rows for different characters. Accepted: transient (seconds, once per interval), and the client-facing consequence is at worst a stale arrow.

### 3.5 REST API (ranking/resource.go)

As specified in PRD §5, JSON:API resource type `rankings`, id = characterId (string):

- `GET /api/rankings/characters?ids={id},{id},…` — bulk. Parse the comma list; empty/unparseable → 400. Query `character_rankings` by `tenant + character_id IN (…)`; unknown ids simply absent from `data` (FR-9 — callers default to zeros). One DB query.
- `GET /api/rankings/characters/{characterId}` — single; 404 when absent.

Attributes: `worldId`, `rank`, `rankMove`, `jobRank`, `jobRankMove` (moves signed), `computedAt`. No write endpoints. Standard tenant middleware (missing headers → 400).

## 4. atlas-login Integration

### 4.1 Model (`services/atlas-login/atlas.com/login/character/model.go`)

Add four fields to `Model` — `rank uint32`, `rankMove int32`, `jobRank uint32`, `jobRankMove int32` — with builder setters (`SetRank`…) mirrored in `ToBuilder()`/`Build()`. Replace the hardcoded getters (`model.go:55-69`):

- `Rank()`, `JobRank()` return the stored values.
- `RankMove()`, `JobRankMove()` return `uint32(m.rankMove)` / `uint32(m.jobRankMove)` — **two's-complement pass-through**; the packet lib's fields are `uint32` (`libs/atlas-packet/model/character_list_entry.go:12-21`) and the v83 client reinterprets them signed (abs + sign-branch, IDA-verified per PRD §1). A unit test pins `int32(-1) → 0xFFFFFFFF`.

No packet-writer changes: `toCharacterListEntry` (`socket/writer/character_list.go:71-102`) already reads these getters, and the `rankEnabled` byte stays `!gm` (`character_list_entry.go:54`).

### 4.2 Fetch wiring

New login-side package `services/atlas-login/atlas.com/login/ranking/`:

- `requests.go`: `requests.RootUrl("RANKINGS")` → env `RANKINGS_SERVICE_URL` with automatic `BASE_SERVICE_URL` fallback (`libs/atlas-rest/requests/url.go:14-19`); bulk request `rankings/characters?ids=…`.
- `rest.go`: RestModel matching §3.5.

Integration point: `character/processor.go` `GetForWorld` (`processor.go:72-80`) — after models are fetched (and decorated), apply a **slice-level rankings decoration**: collect the ≤ ~15 character ids, make **one** bulk call (FR-8), and rebuild each model via `ToBuilder().SetRank(…)…Build()`. Both call paths converge here — world-selection (`socket/handler/character_list_world.go:48`) and view-all (`character_view_all.go:97`, one bulk call per world) — so no writer threading is needed. This is a slice transform, not a per-model `model.Decorator` (a per-model decorator would make N calls).

**Fail-open (FR-11):** any error/timeout → `l.WithError(err).Warnf(…)`, return the models unchanged (zero-valued rank fields). Template: the atlas-maps location fallback in `socket/writer/character_list.go:77-87`. The call must be bounded by a tight client-side timeout so login latency never rides on atlas-rankings health; the exact mechanism (requests-lib option vs dedicated client) is resolved at plan time against `libs/atlas-rest` — if the lib exposes no per-call timeout, add one rather than inheriting an unbounded default.

## 5. atlas-tenants Changes

New configuration resource `rankings` following the routes/vessels precedent — all five touch points (per the established checklist):

1. `configuration/rest.go` — `RankingsRestModel{ RecomputeIntervalMinutes uint32 }` + `GetName()` (`"rankings"`), Transform/Extract, JSON helpers.
2. `configuration/resource.go` — GET/POST/PATCH/DELETE handlers + routes at `/tenants/{tenantId}/configurations/rankings` (alongside `resource.go:647-668`).
3. `configuration/processor.go` — interface + impl additions.
4. `configuration/mock/processor.go` — matching mock funcs (**mandatory**; tests fail otherwise).
5. `configuration/administrator.go`/`provider.go` as needed by the generic layer.

No seed data: absent config = default 60 min in atlas-rankings.

## 6. Deployment & Scaffolding Checklist

New-service touch points (verified against current repo state):

| Artifact | Change |
|---|---|
| `.github/config/services.json` | Add `atlas-rankings` entry (name/type/path/module_path/docker_image/docker_context), alphabetical |
| `docker-bake.hcl` | Add `"atlas-rankings"` to `go_services` (hand-synced with services.json — HCL can't read JSON) |
| Repo-root `Dockerfile` | **No change** (only new shared libs require edits; none added) |
| `go.work` | Add `./services/atlas-rankings/atlas.com/rankings` to `use (…)` |
| `deploy/k8s/base/atlas-rankings.yaml` | Deployment (replicas 2, `envFrom atlas-env`, `DB_NAME=atlas-rankings`, `DB_USER/DB_PASSWORD` from `db-credentials`, Redis env matching other atlas-lock/atlas-redis adopters) + Service; `readinessProbe: httpGet /api/readyz:8080` (bug pattern: never bare `/readyz`) |
| `deploy/k8s/base/kustomization.yaml` | Register the manifest (alphabetical) |
| `deploy/k8s/overlays/{main,pr}/patches/db-name-suffix.yaml` | Per-service DB_NAME suffix blocks (`atlas-rankings-main` / `-PLACEHOLDER_ATLAS_ENV`) |
| `deploy/shared/routes.conf` | `location ~ ^/api/rankings(/.*)?$ { proxy_pass http://atlas-rankings:8080; }` (alphabetical) + run `./deploy/scripts/sync-k8s-ingress-routes.sh` |
| `deploy/k8s/base/env-configmap.yaml` | **No new keys.** No Kafka topics; login reaches rankings via `BASE_SERVICE_URL` fallback — do NOT hardcode `RANKINGS_SERVICE_URL` in base (hardcoded `*_SERVICE_URL` breaks env overlays; npc-shops precedent) |
| Bruno collection | `services/atlas-rankings/.bruno/` per scaffolding checklist |
| Service README | Endpoints table, config resource, recompute semantics, scaling note (§2.1) |

Existing-tenant note: no opcode/template seeding applies (no packet changes), and no live-tenant config patching is required — absent config falls back to the default.

Verification gate (CLAUDE.md): `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake atlas-rankings` (and `atlas-login`, `atlas-tenants` — their `go.mod`s are touched only if deps change, but bake them if so); `tools/redis-key-guard.sh` clean (atlas-lock usage is through the lib — compliant by construction).

## 7. Testing Strategy

**atlas-rankings (unit, table-driven; Builder pattern for setup — no `*_testhelpers.go`):**
- `compute.go`: ordering (`level DESC, exp DESC, characterId ASC`), 1-based uniqueness, job-category derivation incl. Cygnus/Aran ids (1000→10, 2100→21), GM exclusion (excluded *and* not counted), multi-world isolation within a tenant.
- Move math across two consecutive computes: up/down/unchanged, first-seen → 0, cross-category job move, rank vacated by deletion.
- Processor + sqlite (with `database.RegisterTenantCallbacks` in tests): upsert-then-prune leaves exactly the new set; deleted character's row gone; two-tenant isolation (rows and reads never cross tenants).
- REST handlers: bulk parse (400 on empty/garbage), unknown ids omitted, single 404.
- Scheduler unit: due/not-due decision from cycle row + interval; config 404 → 60 min default.

**atlas-login:**
- Getter conversion: `int32(-1) → uint32 0xFFFFFFFF`; builder round-trip for the four fields.
- Slice decoration: bulk response merges onto the right models; missing ids stay zero; error path returns originals + warning (fail-open).

**atlas-tenants:** resource CRUD tests mirroring the routes/vessels tests; mock updated.

**Manual (acceptance):** v83 tenant — pre-first-cycle character shows "Ranking not available"; post-cycle shows "Ranked at N" with correct arrows for +/−/0 movement; two tenants isolated. v95 zero-rank rendering checked opportunistically if a v95 tenant is in the test matrix (PRD §9.3).

## 8. Risks & Edge Cases

- **Unpaginated character scan:** acceptable at current populations (§2.1); revisit when task-117 lands. The scan decodes a trimmed ForeignRestModel to keep memory proportional to needed fields.
- **Tenant with zero eligible characters:** cycle row still updates (no busy-loop); prune clears any stale rows; login gets no records → zeros.
- **atlas-tenants down during a tick:** tenant enumeration fails → log warning, skip tick; rankings simply age. Login is unaffected (its data path is rankings-DB-only).
- **Redis unavailable (atlas-lock):** leadership can't be confirmed → recompute pauses (fail-closed on the writer side, correct), REST keeps serving existing rows.
- **Clock skew between replicas:** cadence decisions read DB timestamps written by the leader; only one writer exists at a time, so skew affects at worst cycle spacing by the skew amount.
- **Job id 0 division:** beginner category 0 is a valid category — job rank among beginners, matching Cosmic.
- **uint32 exp / byte level:** both absolute values from atlas-character (`entity.go:37-38`); exp is per-level progress, so the `level DESC, exp DESC` composite is correct.

## 9. Explicitly Out of Scope (per PRD)

Rank commands/notifications, UI leaderboards, real-time updates, last-login-aware move carry-over, fame/meso tiebreaks, packet changes.
