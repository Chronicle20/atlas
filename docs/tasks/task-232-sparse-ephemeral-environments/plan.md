# Sparse Ephemeral Environments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a per-PR ephemeral environment deploy only its changed services plus a two-workload socket floor, with every other service served by the `main` baseline, while every operation stays logically inside its environment across REST, Kafka, sagas, scheduled work and autonomous loops.

**Architecture:** Environment becomes a property of the *operation*, carried on `context.Context` exactly as tenant already is, propagated as one flat header on REST and one message header on Kafka. A per-process in-memory registry — projected from a compacted Kafka topic using the machinery `libs/atlas-service` already has for configuration — answers "which deployment is the effective implementation for `(environment, service)`" with no I/O. Outbound REST targets the operation's environment's own nginx ingress (whose per-service upstream table is static for that environment's life); inbound Kafka runs an ownership gate in `libs/atlas-kafka` before any domain handler. The data plane shares `main`'s Postgres/Kafka/Redis and is isolated by `tenant_id`; the control plane (`atlas-configurations`, `atlas-tenants`) gains an `environment` column and is isolated by it.

**Tech Stack:** Go 1.26 (per-module `go 1.25.5`), `go.work` monorepo, GORM + Postgres, `segmentio/kafka-go`, go-redis v9, logrus + ECS JSON, nginx ingress, Kustomize + Argo CD, GitHub Actions.

**Spec:** [`design.md`](design.md) (mechanisms M1–M5, corrections C1–C4, deviations V1–V4). PRD: [`prd.md`](prd.md). Starting inventory: [`isolation-audit.md`](isolation-audit.md).

---

## Global Constraints

Every task's requirements implicitly include this section.

- **Inert by default (FR-1.8, NFR-7, NG6).** With no Environment record present, every mechanism must return exactly today's answer: `env.Id("")` resolves to the local deployment for all registry queries, the Kafka gate processes everything, `RootUrl` returns exactly what it returns today. `main`'s observable behaviour, configuration and deployment must not change at any point. Every library task carries an explicit "empty environment ⇒ legacy behaviour" test.
- **Fail closed (D4).** An operation whose environment cannot be resolved is dropped and counted, never executed by the baseline. REST rejects; Kafka acknowledges-without-processing and increments an alertable counter. Never fall back to the baseline.
- **No environment logic in domain packages (NG5, FR-4.5).** Only `main.go`, `kafka/`, `rest/` and `libs/` may import `libs/atlas-env`. Enforced by `env-domain-guard` (Task 27).
- **No I/O on the resolution path (FR-1.6, NG4, NFR-1).** All four registry queries are in-memory map lookups. No REST, Kafka, database or Kubernetes API call.
- **Environment is never persisted on data-plane rows (FR-7.4).** `tenant_id` stays their only scoping column. Environment *is* persisted on control-plane rows.
- **`main` is never mutated to serve an ephemeral environment (G7, NG6).** No write path in this plan may create, update or delete a row whose `environment` differs from the caller's.
- **Header spelling.** `ENVIRONMENT`, flat uppercase-underscore, on both REST and Kafka. The context key is the same string. Constant: `env.Key`.
- **Environment id vocabulary.** `env.Id` is a string. `"main"` is the baseline id in this repo but MUST NOT be hard-coded anywhere except test fixtures and the `main` overlay — the baseline is a field on each environment record (FR-1.5).
- **Verification scope.** Implementers run module-local `go build ./... && go test ./...` only. Repo-wide `tools/verify.sh` is the `atlas-verifier` agent's job, in its own context. The flagless `tools/verify.sh` must exit 0 before the branch is claimed done.
- **Commit discipline.** One commit per task minimum; the TDD steps within a task may commit more often. Never commit to `main`.
- **Test helpers.** Use the project's Builder pattern for test setup. Do not create `*_testhelpers.go` files.
- **Line endings.** Preserve; do not normalise CRLF→LF as a side effect.

### Decisions this plan takes that `design.md` left open (§17)

| # | Open item | Decision | Rationale |
|---|---|---|---|
| **P1** | `atlas-environments` new service vs. package in `atlas-configurations` | **Package inside `atlas-configurations`** (`environments/`), published on `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS` through the existing transactional outbox | Reuses the outbox → compacted-topic → projection path verbatim (the whole argument of design §4.1); adds no 65th Deployment and no new migration path; and makes FR-9.3's "changes to `atlas-configurations` escalate to isolated" automatically cover environment-schema changes, which design §17.1 calls "arguably correct" |
| **P2** | `env.Id` validation | Constrained at *ingest* to `^[a-z][a-z0-9-]{0,30}[a-z0-9]$`; **not** constrained per-operation, and the namespace is read from the record's `namespace` field, never derived from the id | Keeps FR-1.5's second-baseline promise fully open (any id, any namespace) while making malformed records rejectable at the one place they enter the system |
| **P3** | The `NS_*` variable list (design §5.3) | **Derived** by `tools/gen-routes.sh` from the distinct `atlas-<svc>` upstreams it already rewrites in `deploy/shared/routes.conf`; name = `NS_` + service name uppercased with `-`→`_` (`atlas-party-quests` → `NS_ATLAS_PARTY_QUESTS`). Never hand-listed | The 114 `location` blocks and the 64 Deployments do not correspond 1:1; deriving from the file that already exists removes the discrepancy instead of resolving it by hand |
| **P4** | Module layout for `libs/atlas-env` | `libs/atlas-env` is a **leaf module** (depends only on `libs/atlas-model`). The Kafka-fed projection that populates its registry lives in `libs/atlas-service` | `libs/atlas-kafka` and `libs/atlas-rest` must import `atlas-env` for the gate and the header. If `atlas-env` also imported `atlas-kafka` for its projection the module graph would be mutually recursive. `libs/atlas-service` already owns bootstrap-time projection wiring and is imported by no library |

### Phase map

| Phase | Tasks | Deliverable |
|---|---|---|
| **A — Prerequisites (FR-8, blocking)** | 1–16 | Postgres query-path sweep recorded; Redis data-plane migration and the `atlas-storage` defect fix; control-plane `environment` column + backfill + scoped reads/writes; scope guards |
| **B — Libraries (all inert)** | 17–29 | `libs/atlas-env`; environment records in `atlas-configurations`; projection + registry wiring in `libs/atlas-service`; REST and Kafka header plumbing; the ownership gate; the autonomous-iteration helper; guards; observability |
| **C — Services** | 30–41 | 62 Go services wire the registry; the 18 ticker-bearing loops classified and converted |
| **D — Deployment** | 42–48 | Per-service `NS_*` ingress routing; the sparse overlay; consumer-offset seeding; bootstrap create-own-row (C2); teardown ordering |
| **E — Mode selection** | 49–51 | Affected-service determination, escalation, PR reporting |
| **F — Enable and prove** | 52–55 | `env-bootstrap-guard` flips green, the Kafka fan-out cost is measured, sparse becomes the default, the §17 proof is executed |

---
# Phase A — Prerequisites (FR-8, blocking)

Nothing in Phase B–F is safe to enable until Phase A lands. Phase A is
independently valuable: Tasks 4 and 5 fix a live cross-tenant defect that
exists today with or without this project.

---

### Task 1: Query-path sweep, part 1 of 3 (`atlas-account` … `atlas-fame`)

Deliverable is a recorded audit, not code. FR-8.1 requires a **sweep, not a
sample**: a `TenantId` column proves nothing about the `WHERE` clause.

**Files:**
- Create: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — the audit table; this task creates it and fills rows for the first third
- Read-only: every `services/atlas-{account,asset-expiration,ban,buddies,buffs,cashshop,chairs,chalkboards,channel,character,character-factory,configurations,consumables,data,doors,drop-information,drops,effective-stats,expressions,fame}/**/entity.go`, `provider.go`, `administrator.go`, `processor.go`

Patterns to copy: none — this is an audit. The evidence format is fixed below.

**Interfaces:**
- Produces: `query-scope-audit.md` with one row per `(service, table)` and the columns defined below. Tasks 2 and 3 append rows to the same file. Task 15 (`tenant-scope-guard`) reads its allowlist from the `FORCES-ISOLATED` and `TRANSITIVE` rows.

- [ ] **Step 1: Create the audit document skeleton**

```markdown
# Query-path scope audit (FR-8.1)

Every Postgres access path in the fleet, classified. A `tenant_id` column is
not evidence; the evidence is the `WHERE` clause of every read and the
assignment of every write.

Verdicts:
- `SCOPED` — every query path filters on tenant (data plane) or environment
  (control plane). Cite the file:line of the narrowest query builder.
- `TRANSITIVE` — no direct scoping column; every access goes through a
  `SCOPED` parent. Cite the parent and the join.
- `UNSCOPED` — at least one query path does not filter. **Blocking.** Cite it.
- `CONTROL` — control plane; scoped by environment after Task 10/11.
- `FORCES-ISOLATED` — cannot be scoped; the service must escalate to isolated
  mode (FR-9.3). Requires a written reason.

| Service | Table / entity | Plane | Verdict | Evidence (file:line) | Notes |
|---|---|---|---|---|---|
```

- [ ] **Step 2: Enumerate the entities in this third**

Run:

```sh
for s in account asset-expiration ban buddies buffs cashshop chairs \
         chalkboards channel character character-factory configurations \
         consumables data doors drop-information drops effective-stats \
         expressions fame; do
  find "services/atlas-$s" -name entity.go -printf '%p\n'
done
```

Expected: a list of `entity.go` paths. Every one gets a row.

- [ ] **Step 3: For each entity, read its query builders and classify**

For each `entity.go`, read the `provider.go` and `administrator.go` in the
same package (Atlas keeps read builders in `provider.go` and writes in
`administrator.go`). The check is mechanical:

```sh
# every read builder for one package
grep -n "db.Where\|db.Model\|First(\|Find(\|Take(" services/atlas-<svc>/atlas.com/<svc>/<pkg>/provider.go
# every write path
grep -n "Create(\|Save(\|Updates\?(\|Delete(" services/atlas-<svc>/atlas.com/<svc>/<pkg>/administrator.go
```

A path is `SCOPED` only if `tenant_id` (or the entity's tenant field) appears
in the `Where` of every read and in the struct literal of every write. Record
the file:line of the *narrowest* builder as evidence — a package-level
`func entityByTenant(...)` helper that every caller routes through is the
strongest evidence and should be cited once.

- [ ] **Step 4: Record every row, including the negatives**

Append one row per entity. `UNSCOPED` rows are the point of the exercise —
do not omit or soften them. If this third finds an `UNSCOPED` row other than
the two already known (`atlas-storage` `npc-context` is Redis, not Postgres),
add a follow-up task to this plan file under Phase A and say so in the task
report.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "docs(task-232): query-path scope audit, part 1 of 3"
```

---

### Task 2: Query-path sweep, part 2 of 3 (`atlas-families` … `atlas-portals`)

**Files:**
- Modify: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — append rows
- Read-only: every `entity.go`/`provider.go`/`administrator.go` under `services/atlas-{families,guilds,inventory,invites,keys,kites,login,map-actions,maps,marriages,merchant,messages,messengers,mini-games,monster-book,monster-death,monsters,mounts,mts,notes,npc-conversations,npc-shops,parties,party-quests,pets,portal-actions,portals}`

**Interfaces:**
- Consumes: the audit document and verdict vocabulary created in Task 1.
- Produces: appended rows in the same table.

- [ ] **Step 1: Enumerate the entities in this third**

```sh
for s in families guilds inventory invites keys kites login map-actions maps \
         marriages merchant messages messengers mini-games monster-book \
         monster-death monsters mounts mts notes npc-conversations npc-shops \
         parties party-quests pets portal-actions portals; do
  find "services/atlas-$s" -name entity.go -printf '%p\n'
done
```

- [ ] **Step 2: Classify each with the Task 1 Step 3 procedure**

Same commands, same evidence rule. `atlas-quest`'s `quest/medal/entity.go` is
covered in Task 3 and is the known `TRANSITIVE` candidate; nothing in this
third is pre-classified.

- [ ] **Step 3: Append the rows and commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "docs(task-232): query-path scope audit, part 2 of 3"
```

---

### Task 3: Query-path sweep, part 3 of 3 (`atlas-query-aggregator` … `atlas-world`) and the FR-8.6 disposition list

**Files:**
- Modify: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — append the final rows and the §2 disposition section
- Read-only: every `entity.go`/`provider.go`/`administrator.go` under `services/atlas-{query-aggregator,quest,rankings,rates,reactor-actions,reactors,renders,reward-pools,rps,saga-orchestrator,skills,storage,summons,tenants,trades,transports,world}`
- Read-only: `libs/atlas-object-id/`, `libs/atlas-outbox/`, `libs/atlas-lock/` — for the FR-8.6 disposition section

**Interfaces:**
- Consumes: the audit document from Tasks 1–2.
- Produces: the completed audit plus a `## 2. Non-Postgres deployment-scoped resources (FR-8.6)` section that Task 16 does *not* duplicate.

- [ ] **Step 1: Enumerate and classify the final third**

Same procedure as Task 1 Step 3. Two entities are pre-identified and must be
confirmed rather than assumed:

- `services/atlas-quest/atlas.com/quest/quest/medal/entity.go` — keyed
  `(QuestStatusId, MapId)`, expected `TRANSITIVE` through the tenant-scoped
  quest-status parent. Cite the join and confirm the FK integrity: a medal row
  whose parent is deleted must not survive.
- `services/atlas-tenants/atlas.com/tenants/tenant/entity.go` — `CONTROL`,
  scoped by environment in Task 11.

- [ ] **Step 2: Write the FR-8.6 disposition section**

```markdown
## 2. Non-Postgres deployment-scoped resources (FR-8.6)

Every resource whose isolation currently depends on deployment identity, with
a disposition: **scope it**, or **forces isolated mode**.

| Resource | Where | Current isolation | Disposition |
|---|---|---|---|
| Redis key prefix | `libs/atlas-redis/keys.go:15` | `ATLAS_ENV` package-level var | Scoped: data plane → tenant-scoped API (Tasks 4–8); prefix stays load-bearing for isolated mode only |
| Kafka topic names | `overlays/pr/scripts/gen-topic-config.sh` | `-<ATLAS_ENV>` suffix | Scoped: sparse consumes unsuffixed topics + ownership gate (Task 25) |
| Postgres DB names | `overlays/pr/scripts/gen-db-name-suffix.sh` | `<db>-<ATLAS_ENV>` | Scoped: sparse shares `main`'s databases (D1) |
| Consumer group ids | `libs/atlas-kafka/consumergroup/resolver.go` | runtime-resolved | Already correct; no change |
| Object id allocation | `libs/atlas-object-id` | *fill in from reading* | *fill in* |
| Outbox advisory lock | `libs/atlas-outbox/lock.go` | single constant key per DB | Deliberately global; now serialises drainers across environments — throughput coupling, not correctness (design §8.4) |
| MinIO canonical objects | `services/atlas-pr-bootstrap/scripts/reconcile-minio.sh` | per-environment restore | *fill in* |
| Login/channel ports + advertised IP | `tools/gen-lb-ports.sh`, `services/atlas-pr-bootstrap/scripts/version-ports.sh` | per-namespace LoadBalancer | Scoped in Task 46 |
```

Read each *fill in* resource before writing its row. Do not guess a
disposition; if a resource genuinely cannot be scoped, write
`forces isolated mode` and the reason, and add the path to Task 50's
escalation rule — say so in the task report so the Task 50 implementer does
not have to re-derive it.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "docs(task-232): query-path scope audit, part 3 of 3 + FR-8.6 dispositions"
```

---

### Task 4: Fix the `atlas-storage` `npc-context` cross-tenant defect

This is a **live defect today**, independent of this project: the cache keys
on character id alone, and character ids are per-tenant sequences, so two
tenants collide inside multi-tenant `main`. It lands on its own commit so the
fix is reviewable without the migration around it (design §9).

**Files:**
- Modify: `services/atlas-storage/atlas.com/storage/storage/cache.go` — interface and implementation take a `tenant.Model`
- Modify: `services/atlas-storage/atlas.com/storage/storage/cache_test.go` — existing tests pass a tenant; add the collision test
- Modify: `services/atlas-storage/atlas.com/storage/kafka/consumer/storage/consumer.go:169` — caller
- Modify: `services/atlas-storage/atlas.com/storage/kafka/consumer/character/consumer.go:88` — caller
- Read-only: `libs/atlas-redis/tenant_registry.go:26` — `NewTenantRegistry` signature; its methods take `t tenant.Model` explicitly
- Read-only: `services/atlas-storage/atlas.com/storage/main.go:65` — `InitNpcContextCache(rc)` call, unchanged

Module root for `go build`/`go test`: `services/atlas-storage/atlas.com/storage`.

**Interfaces:**
- Produces: `NpcContextCacheInterface` with tenant-carrying methods:
  ```go
  Get(t tenant.Model, characterId uint32) (uint32, bool)
  Put(t tenant.Model, characterId uint32, npcId uint32, ttl time.Duration)
  Remove(t tenant.Model, characterId uint32)
  ```

- [ ] **Step 1: Write the failing collision test**

Add to `services/atlas-storage/atlas.com/storage/storage/cache_test.go`:

```go
func TestNpcContextCacheIsTenantScoped(t *testing.T) {
	client := newTestRedis(t) // existing helper in this file
	InitNpcContextCache(client)

	t1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	t2, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	GetNpcContextCache().Put(t1, 12345, 9001, 30*time.Minute)
	GetNpcContextCache().Put(t2, 12345, 9002, 30*time.Minute)

	got1, ok := GetNpcContextCache().Get(t1, 12345)
	if !ok || got1 != 9001 {
		t.Fatalf("tenant 1: got (%d, %v), want (9001, true)", got1, ok)
	}
	got2, ok := GetNpcContextCache().Get(t2, 12345)
	if !ok || got2 != 9002 {
		t.Fatalf("tenant 2: got (%d, %v), want (9002, true)", got2, ok)
	}

	GetNpcContextCache().Remove(t1, 12345)
	if _, ok := GetNpcContextCache().Get(t2, 12345); !ok {
		t.Fatal("removing tenant 1's entry removed tenant 2's")
	}
}
```

If `cache_test.go` has no redis test helper, use the same construction the
existing tests in that file use (they call `InitNpcContextCache(client)` with
a client built at the top of the file); reuse it rather than adding a new one.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-storage/atlas.com/storage && go test ./storage/ -run TestNpcContextCacheIsTenantScoped -v`
Expected: FAIL to compile — `too many arguments in call to GetNpcContextCache().Put`.

- [ ] **Step 3: Migrate the cache to the tenant-scoped registry**

In `cache.go`:

```go
type NpcContextCacheInterface interface {
	Get(t tenant.Model, characterId uint32) (uint32, bool)
	Put(t tenant.Model, characterId uint32, npcId uint32, ttl time.Duration)
	Remove(t tenant.Model, characterId uint32)
}

type NpcContextCache struct {
	reg *atlasredis.TenantRegistry[uint32, uint32]
}

func InitNpcContextCache(client *goredis.Client) {
	npcContextCache = &NpcContextCache{
		reg: atlasredis.NewTenantRegistry[uint32, uint32](
			client,
			"npc-context",
			func(characterId uint32) string {
				return strconv.FormatUint(uint64(characterId), 10)
			},
		),
	}
}

func (c *NpcContextCache) Get(t tenant.Model, characterId uint32) (uint32, bool) {
	npcId, err := c.reg.Get(context.Background(), t, characterId)
	if err != nil {
		return 0, false
	}
	return npcId, true
}

func (c *NpcContextCache) Put(t tenant.Model, characterId uint32, npcId uint32, ttl time.Duration) {
	_ = c.reg.PutWithTTL(context.Background(), t, characterId, npcId, ttl)
}

func (c *NpcContextCache) Remove(t tenant.Model, characterId uint32) {
	_ = c.reg.Remove(context.Background(), t, characterId)
}
```

Read `libs/atlas-redis/tenant_registry.go` and match the exact method names
and signatures it exposes (`PutWithTTL`, `Remove`) — if a method the code
above names does not exist on `TenantRegistry`, use the one that does and note
it in the task report rather than adding a method to the library.

- [ ] **Step 4: Update the two callers**

Both call sites are inside Kafka consumers, which already have the tenant on
`ctx`. In `kafka/consumer/storage/consumer.go:169` and
`kafka/consumer/character/consumer.go:88`, change:

```go
cache := storage.GetNpcContextCache()
```

to resolve the tenant first and thread it:

```go
t := tenant.MustFromContext(ctx)
cache := storage.GetNpcContextCache()
// ...every cache.Get/Put/Remove call gains `t` as its first argument
```

Use `tenant.FromContext(ctx)()` and return the error if the surrounding
function returns one; only use `MustFromContext` where the surrounding code
already does.

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-storage/atlas.com/storage && go build ./... && go test ./...`
Expected: PASS, including `TestNpcContextCacheIsTenantScoped`.

- [ ] **Step 6: Update the service docs**

`services/atlas-storage/docs/domain.md:191-199` documents `NpcContextCache`.
Update the key description to state the key is tenant-scoped.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-storage
git commit -m "fix(atlas-storage): tenant-scope the npc-context cache

The cache keyed on character id alone. Character ids are per-tenant
sequences, so two tenants collided inside multi-tenant main; the ATLAS_ENV
key prefix masked it only across environments. Pre-existing defect, not
introduced by sparse environments."
```

---

### Task 5: Migrate `atlas-monsters` Redis state to the tenant-scoped API

`monsterKey(t tenant.Model, uniqueId uint32)` already embeds the tenant, so
this migration is **behaviour-preserving in key shape** only if the new
`tenantEntityKey` layout is accepted as a key change. It is: sparse mode does
not share a Redis keyspace with a running `main` until Phase F, and the
registries are caches of live state that repopulate.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go` — the six namespaces below
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go` (or the tests beside it) — assert per-tenant separation
- Read-only: `libs/atlas-redis/tenant_registry.go`, `libs/atlas-redis/keyed_set.go:99` (`NewTenantKeyedSet`), `libs/atlas-redis/keyed_hash.go:20`, `libs/atlas-redis/set.go:65`, `libs/atlas-redis/counter.go:67`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/isolation-audit.md` §4.2 — the namespace inventory

Namespaces to migrate (isolation-audit §4.2): `monster`, `monster-map`,
`monster-cooldown`, `monster-attack-cooldown`, `monster-puppet`,
`monster-puppet-field`.

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: no exported signature change outside the service. The registry's
  own methods gain a `tenant.Model` parameter wherever they did not have one.

- [ ] **Step 1: Inventory the current constructors**

Run:

```sh
grep -n "atlasredis.New" services/atlas-monsters/atlas.com/monsters/monster/registry.go
```

Expected: one line per namespace, each naming a **bare** constructor
(`NewRegistry`, `NewKeyedSet`, `NewSet`, `NewIndex`, …). Record the mapping
bare → tenant-scoped before editing:

| Bare | Tenant-scoped replacement |
|---|---|
| `NewRegistry` | `NewTenantRegistry` |
| `NewSet` | `NewTenantSet` |
| `NewKeyedSet` | `NewTenantKeyedSet` |
| `NewKeyedHash` | `NewTenantKeyedHash` |
| `NewCoalescedRegistry` | `NewTenantCoalescedRegistry` |
| `NewKeyedSortedSet` | `NewTenantKeyedSortedSet` |

`NewIndex` / `NewUint32Index` / `NewTTLRegistry` have **no** tenant-scoped
equivalent today. If the file uses one, stop and report it: Task 9 adds the
missing tenant-scoped constructor, and this task should be re-ordered after
it rather than working around the gap.

- [ ] **Step 2: Write the failing per-tenant separation test**

For each namespace, one test of the shape (adapt names to the actual registry
methods):

```go
func TestMonsterRegistryIsTenantScoped(t *testing.T) {
	r := newTestRegistry(t)
	t1 := testTenant(t)
	t2 := testTenant(t)

	r.Add(context.Background(), t1, monsterWithUniqueId(1))
	got, err := r.GetAll(context.Background(), t2)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("tenant 2 saw %d of tenant 1's monsters, want 0", len(got))
	}
}
```

Use the Builder pattern already present in this service for `monsterWith…`;
do not add a `*_testhelpers.go` file.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TenantScoped -v`
Expected: FAIL (compile error on the new tenant parameter, or a non-zero
count if the current key already separates and only the constructor changes —
in the latter case the test is a regression pin and may pass; say so in the
report and keep it).

- [ ] **Step 4: Swap the constructors and drop the tenant out of the keyFn**

The tenant-scoped constructors build the key as
`<prefix>:<namespace>:<tenantKey>:<entityKey>` via `tenantEntityKey`, so the
`keyFn` must no longer embed the tenant:

```go
// before
atlasredis.NewRegistry[uint32, Model](client, "monster", func(id uint32) string {
	return monsterKey(t, id)
})

// after
atlasredis.NewTenantRegistry[uint32, Model](client, "monster", func(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
})
```

Delete `monsterKey` and any sibling key builders that exist only to splice the
tenant in. Thread `tenant.Model` through the registry methods to the
`TenantRegistry` calls.

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monsters
git commit -m "refactor(atlas-monsters): move Redis state to the tenant-scoped API"
```

---

### Task 6: Migrate `atlas-summons` and `atlas-npc-shops` Redis state

Two small services, same mechanical change as Task 5. `atlas-npc-shops`'
consumable cache keys on the shop `uuid` alone — a uuid may be globally
unique but is not tenant-scoped *by construction*, which is exactly the
property D7 requires.

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/registry.go` (namespaces `summon`, `summon-map`, `summon-owner`)
- Modify: the summon registry's test file beside it
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/cache.go:32` (namespace `npc-shop:consumables`)
- Modify: the npc-shops cache test file beside it
- Read-only: `libs/atlas-redis/tenant_registry.go` and the constructor table in Task 5 Step 1

Module roots: `services/atlas-summons/atlas.com/summons`,
`services/atlas-npc-shops/atlas.com/npc`.

**Interfaces:**
- Consumes: the bare→tenant-scoped constructor table from Task 5 Step 1.
- Produces: no cross-service signature change.

- [ ] **Step 1: Audit the summon keyFns before touching them**

Run:

```sh
grep -n "atlasredis.New" -A 6 services/atlas-summons/atlas.com/summons/summon/registry.go
```

isolation-audit §4.2 marks these "audit keyFns first" — a keyFn that already
embeds the tenant must have the tenant removed from it (Task 5 Step 4), and a
keyFn that does not was leaking. Record which of the three it was in the task
report; a leaking namespace is a second live defect and belongs in the commit
message.

- [ ] **Step 2: Write the failing per-tenant separation tests**

One per service, same shape as Task 5 Step 2: write under tenant 1, read
under tenant 2, assert empty.

- [ ] **Step 3: Run to verify they fail**

Run:
```sh
cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run TenantScoped -v
cd services/atlas-npc-shops/atlas.com/npc && go test ./shops/ -run TenantScoped -v
```
Expected: FAIL.

- [ ] **Step 4: Swap the constructors in both services**

Same edit shape as Task 5 Step 4.

- [ ] **Step 5: Run both module test suites**

Run:
```sh
cd services/atlas-summons/atlas.com/summons && go build ./... && go test ./...
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons services/atlas-npc-shops
git commit -m "refactor(atlas-summons,atlas-npc-shops): move Redis state to the tenant-scoped API"
```

---

### Task 7: Migrate `atlas-doors` Redis state (callers build the keys)

`atlas-doors` passes an **identity `keyFn`** — the keys are built by callers,
so the scoping cannot be read off the constructor. isolation-audit §4.1 marks
this "Unknown — must trace callers." Tracing is the first half of the task.

**Files:**
- Modify: `services/atlas-doors/atlas.com/doors/door/registry.go:69-76` — the four namespaces `door`, `door-field`, `door-owner`, `door-town`
- Modify: every caller that builds a key for those registries (found in Step 1)
- Modify: the door registry test file beside `registry.go`
- Read-only: `libs/atlas-redis/tenant_registry.go`

Module root: `services/atlas-doors/atlas.com/doors`.

**Interfaces:**
- Consumes: the constructor table from Task 5 Step 1.
- Produces: no cross-service signature change.

- [ ] **Step 1: Trace every caller and record the key each builds**

Run:

```sh
grep -rn "door\(Registry\|Field\|Owner\|Town\)\|GetRegistry()" services/atlas-doors/atlas.com/doors
```

For each call site, write down the literal key string it produces. Record the
result in the task report as a table (call site → key → contains tenant?).
This table is the finding; a migration done without it is a guess.

- [ ] **Step 2: Write the failing per-tenant separation test for all four namespaces**

Same shape as Task 5 Step 2, one sub-test per namespace.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TenantScoped -v`
Expected: FAIL.

- [ ] **Step 4: Move to the tenant-scoped constructors and strip the tenant from caller-built keys**

The identity `keyFn` becomes the caller's key minus any tenant component; the
`TenantRegistry` re-adds the tenant via `tenantEntityKey`. Every caller loses
whatever tenant splicing it was doing and gains a `tenant.Model` argument.

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-doors
git commit -m "refactor(atlas-doors): move Redis state to the tenant-scoped API"
```

---

### Task 8: Add the environment-scoped Redis API and migrate `atlas-data`'s `ingestrun`

`atlas-data`'s `ingestrun` is **legitimately cross-tenant** — it is control
plane. It does not move to the tenant-scoped API; it moves to a new, narrowly
named environment-scoped one (FR-8.4).

**Files:**
- Create: `libs/atlas-redis/environment.go` — `EnvironmentRegistry` plus `environmentEntityKey`
- Create: `libs/atlas-redis/environment_test.go`
- Modify: `libs/atlas-redis/keys.go` — add `environmentEntityKey` beside `tenantEntityKey` (line 46)
- Modify: `libs/atlas-redis/go.mod` — add the `libs/atlas-env` requirement (Task 17 must land first)
- Modify: `services/atlas-data/atlas.com/data/ingestrun/ingestrun.go:119-129` — swap the constructor
  (correction, controller addendum #1: the file does not live under
  `runtime/ingest/`; the package is `services/atlas-data/atlas.com/data/ingestrun/`)
- Read-only: `libs/atlas-redis/tenant_registry.go` — the shape to mirror
- Read-only: `libs/atlas-env/env.go` (Task 17) — `env.Id`

Module roots: `libs/atlas-redis`, `services/atlas-data/atlas.com/data`.

**Ordering:** this task depends on Task 17 (`libs/atlas-env`). Run it after
Phase B Task 17 if the executor is working strictly in order; the rest of
Phase A does not depend on it.

**Interfaces:**
- Consumes: `env.Id` from Task 17.
- Produces:
  ```go
  func NewEnvironmentRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *EnvironmentRegistry[K, V]
  func (r *EnvironmentRegistry[K, V]) Get(ctx context.Context, e env.Id, key K) (V, error)
  func (r *EnvironmentRegistry[K, V]) Put(ctx context.Context, e env.Id, key K, value V) error
  func (r *EnvironmentRegistry[K, V]) Remove(ctx context.Context, e env.Id, key K) error
  func (r *EnvironmentRegistry[K, V]) GetAllValues(ctx context.Context, e env.Id) ([]V, error)
  ```

  Correction (controller addendum #1): the shipped `libs/atlas-redis/environment.go`
  exports considerably more than the five methods above. At minimum add
  `UpdateWithTTL(ctx context.Context, e env.Id, key K, ttl time.Duration, fn func(V) V) (V, error)`
  (`environment.go:206`). The other shipped methods, for reference: `PutWithTTL`,
  `RemoveExisting`, `Update`, `Exists`, `GetAllEntries`, `ClearByPrefix`,
  `Client`, `Namespace`, `Clear` — listed here so the plan matches what
  landed.

- [ ] **Step 1: Write the failing key-shape test**

`libs/atlas-redis/environment_test.go`:

```go
func TestEnvironmentEntityKey(t *testing.T) {
	got := environmentEntityKey("ingestrun", env.Id("pr-123"), "run-7")
	want := "atlas:ingestrun:pr-123:run-7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEnvironmentEntityKeyLegacyEnvironmentIsUnprefixed(t *testing.T) {
	// An empty environment is the legacy value: the key must be
	// byte-identical to what namespacedKey produced before this type
	// existed, so main's existing Redis state stays addressable (NFR-7).
	got := environmentEntityKey("ingestrun", env.Id(""), "run-7")
	want := "atlas:ingestrun:run-7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-redis && go test ./... -run EnvironmentEntityKey -v`
Expected: FAIL — `undefined: environmentEntityKey`.

- [ ] **Step 3: Implement the key builder**

In `libs/atlas-redis/keys.go`, beside `tenantEntityKey`:

```go
// environmentEntityKey scopes a control-plane key by environment. The empty
// environment is the legacy value and produces exactly the key
// namespacedKey produced before environments existed, so main's existing
// state stays addressable (NFR-7).
func environmentEntityKey(namespace string, e env.Id, entityKey string) string {
	if e == "" {
		return namespacedKey(namespace, entityKey)
	}
	return namespacedKey(namespace, string(e), entityKey)
}

func environmentScanPattern(namespace string, e env.Id) string {
	if e == "" {
		return namespacedKey(namespace, "*")
	}
	return namespacedKey(namespace, string(e), "*")
}
```

- [ ] **Step 4: Implement `EnvironmentRegistry`**

`libs/atlas-redis/environment.go` mirrors `tenant_registry.go` exactly,
substituting `env.Id` for `tenant.Model` and `environmentEntityKey` for
`tenantEntityKey`. Copy the file, rename, and change those two things — do
not redesign the surface.

- [ ] **Step 5: Run the library tests**

Run: `cd libs/atlas-redis && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Migrate `atlas-data`'s `ingestrun`**

Swap `atlasredis.NewRegistry` → `atlasredis.NewEnvironmentRegistry` and pass
`env.Self()` at the call sites. `ingestrun` is a process-local control-plane
concern, so `env.Self()` — not an environment off a context — is the correct
source (design §4.3: a pod's own environment cannot go stale).

- [ ] **Step 7: Run the atlas-data tests**

Run: `cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-redis services/atlas-data
git commit -m "feat(atlas-redis): add the environment-scoped registry; migrate atlas-data ingestrun"
```

---

### Task 9: Withdraw the bare Redis constructors and extend `rediskeyguard`

After Tasks 4–8 no data-plane caller uses a bare constructor. This task makes
a new one fail CI (FR-8.4/8.5).

**Files:**
- Modify: `libs/atlas-redis/registry.go:25`, `set.go:27`, `hash.go:16`, `keyed_set.go:20`, `keyed_hash.go:58`, `coalesced.go:64`, `ttl.go:23`, `index.go:21`, `index.go:74`, `id.go:21`, `id.go:29` — the bare constructors
- Modify: `tools/rediskeyguard/` — the analyzer; add the bare-constructor check
- Modify: `tools/redis-key-guard.sh` — if the allowlist lives there
- Create: `tools/rediskeyguard/testdata/src/bareconstructor/` — analyzer fixture
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — the surviving legitimate call sites

Module roots: `libs/atlas-redis`, `tools/rediskeyguard`.

**Interfaces:**
- Consumes: the migrations from Tasks 4–8.
- Produces: a CI-enforced rule that `redis.New{Registry,Set,Hash,KeyedSet,KeyedHash,CoalescedRegistry,TTLRegistry,Index,Uint32Index,IDGenerator,IDGeneratorWithStart}` may not be called from `services/*` outside an explicit allowlist.

- [ ] **Step 1: Read the existing guard to find its extension point**

Run: `ls tools/rediskeyguard && sed -n 1,80p tools/rediskeyguard/*.go | head -120`

The guard already exists (design §9). Find the function that walks call
expressions and the allowlist mechanism; the new check goes beside them.

- [ ] **Step 2: Write the failing analyzer fixture**

`tools/rediskeyguard/testdata/src/bareconstructor/main.go`:

```go
package bareconstructor

import (
	goredis "github.com/redis/go-redis/v9"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

func bad(c *goredis.Client) {
	_ = atlasredis.NewRegistry[uint32, string](c, "thing", nil) // want `bare Redis constructor`
}

func ok(c *goredis.Client) {
	_ = atlasredis.NewTenantRegistry[uint32, string](c, "thing", nil)
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd tools/rediskeyguard && go test ./...`
Expected: FAIL — the analyzer reports no diagnostic where the fixture wants one.

- [ ] **Step 4: Implement the check**

Report a diagnostic on any selector call whose package is
`github.com/Chronicle20/atlas/libs/atlas-redis` and whose name is in the bare
set, unless the containing file's package path is in the allowlist. Seed the
allowlist with only what Tasks 4–8 could not migrate; every entry needs a
one-line reason comment.

- [ ] **Step 5: Run the guard against the fleet**

Run: `./tools/redis-key-guard.sh`
Expected: exit 0. If it reports a call site Tasks 4–8 missed, migrate it here
rather than allowlisting it — an allowlist entry with no reason is a
regression.

- [ ] **Step 6: Unexport what nothing legitimately calls**

For each bare constructor with zero surviving external callers, lowercase it.
Keep exported only what the allowlist justifies. Run
`cd libs/atlas-redis && go build ./... && go test ./...` and then
`./tools/test-all-go.sh` if available; otherwise build each service that
imports `atlas-redis`.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-redis tools/rediskeyguard tools/redis-key-guard.sh
git commit -m "feat(atlas-redis): withdraw the bare constructors; guard against new use"
```

---

### Task 10: Record the intent of every deliberately-global Redis resource

`NewGlobalIDGenerator`, `NewLock`, `NewLockWithTTL` are global *by intent*.
Under D1 "global" now means "global across environments", which is either
correct (mutual exclusion on a genuinely shared resource) or a
cross-environment stall. Neither can be assumed (isolation-audit §4.3).

**Files:**
- Modify: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — new `## 3. Deliberately-global Redis resources` section
- Modify: each surviving call site — add a one-line comment stating the intent
- Read-only: `libs/atlas-redis/id.go:66`, `libs/atlas-redis/lock.go:29`, `libs/atlas-redis/lock.go:37`

- [ ] **Step 1: Enumerate the call sites**

```sh
grep -rn "NewGlobalIDGenerator\|NewLockWithTTL\|NewLock(" services libs
```

- [ ] **Step 2: Classify each**

For each, answer in one sentence: *is the resource it protects shared across
environments under D1?* Two verdicts only:

- `GLOBAL-CORRECT` — the resource genuinely is one thing across environments
  (e.g. an id space that must not collide). Leave as is; add the comment.
- `SHOULD-BE-SCOPED` — the lock or generator protects per-environment state
  and would now serialise or collide across environments. Convert it to the
  environment-scoped API from Task 8, or to the tenant-scoped one if the
  state is data plane.

- [ ] **Step 3: Convert every `SHOULD-BE-SCOPED` site**

Do not defer these. A cross-environment stall is a live production symptom
once sparse mode is enabled, and the conversion is a constructor swap.

- [ ] **Step 4: Write the audit section and run the affected module tests**

Run `go build ./... && go test ./...` in every module a site was changed in.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md services libs
git commit -m "docs(task-232): state the intent of every globally-scoped Redis resource"
```

---

### Task 11: Add the `environment` column to `atlas-configurations`' three control-plane tables

D5: the control plane is environment-scoped, not tenant-scoped. This task is
schema + backfill only; the scoped read/write paths are Task 13.

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/entity.go` — add `Environment string`
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/entity.go` — add `Environment string`
- Modify: `services/atlas-configurations/atlas.com/configurations/services/entity.go` — add `Environment string` to `Entity` and `HistoryEntity`
- Create: `services/atlas-configurations/atlas.com/configurations/environmentcol/migration.go` — the backfill, run from `Migration`
- Create: `services/atlas-configurations/atlas.com/configurations/environmentcol/migration_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go:43` — the `database.SetMigrations(...)` list gains the backfill
- Read-only: `services/atlas-configurations/atlas.com/configurations/services/backfill.go` and `tenants/backfill.go` — the existing backfill pattern to copy
- Read-only: `libs/atlas-database/` — `Connect` / `SetMigrations`

Module root: `services/atlas-configurations/atlas.com/configurations`.

**Interfaces:**
- Produces: `Entity.Environment string` on all three tables, non-null,
  defaulting to the baseline environment name read from `ATLAS_ENVIRONMENT`
  (falling back to `"main"` **only** in the backfill, which is a one-time
  data migration and is the single place the literal is allowed).
- Produces: `func BackfillEnvironment(db *gorm.DB) error` — assigns every
  pre-existing row to the baseline environment.

- [ ] **Step 1: Write the failing backfill test**

`environmentcol/migration_test.go`, using the service's existing sqlite/pg
test harness (`services/atlas-configurations/atlas.com/configurations/` has
no `test/` package today — copy the harness shape from
`services/atlas-tenants/atlas.com/tenants/test/database.go`, which builds an
in-memory GORM database):

```go
func TestBackfillAssignsExistingRowsToTheBaseline(t *testing.T) {
	db := testDatabase(t)
	if err := db.AutoMigrate(&tenants.Entity{}, &templates.Entity{}, &services.Entity{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// A row written before the column existed has the zero value.
	db.Exec(`INSERT INTO templates (id, region, major_version, minor_version, data, environment) VALUES (?, 'GMS', 83, 1, '{}', '')`, uuid.New())

	if err := BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment: %v", err)
	}

	var got string
	db.Raw(`SELECT environment FROM templates LIMIT 1`).Scan(&got)
	if got != "main" {
		t.Fatalf("environment = %q, want \"main\"", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./environmentcol/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Add the columns**

In each of the three `entity.go` files:

```go
	// Environment scopes this control-plane row to one execution
	// environment (task-232 D5). Never a tenant column: this table exists
	// to provision and serve many tenants.
	Environment string `gorm:"not null;default:''"`
```

`services.Entity`'s key is **unchanged** — it is already `Id`-keyed and
consumers select by `SERVICE_ID` (design C1/V4). Do not add a composite
unique index on `(Type, environment)`.

- [ ] **Step 4: Implement the backfill**

```go
// BackfillEnvironment assigns every control-plane row with an empty
// environment to the baseline. Idempotent: rows already carrying an
// environment are untouched, so re-running a migration is safe.
func BackfillEnvironment(db *gorm.DB, baseline string) error {
	for _, table := range []string{"tenants", "templates", "services", "service_history"} {
		if err := db.Exec(
			"UPDATE "+table+" SET environment = ? WHERE environment = '' OR environment IS NULL",
			baseline,
		).Error; err != nil {
			return fmt.Errorf("backfill %s: %w", table, err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Wire it into `main.go`**

```go
db := database.Connect(l, database.SetMigrations(
	templates.Migration, tenants.Migration, services.Migration, outboxlib.Migration,
	environmentcol.Migration, // must run last: it backfills the columns the three above create
))
```

with

```go
func Migration(db *gorm.DB) error {
	return BackfillEnvironment(db, environmentBaseline())
}

// environmentBaseline is the environment existing rows belong to. main is
// the only baseline this repo ships; the literal appears here and nowhere
// else (FR-1.5 keeps the runtime baseline a record field).
func environmentBaseline() string {
	if v := os.Getenv("ATLAS_ENVIRONMENT"); v != "" {
		return v
	}
	return "main"
}
```

- [ ] **Step 6: Run the tests**

Run: `cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations
git commit -m "feat(atlas-configurations): add the environment dimension to the control-plane tables"
```

---

### Task 12: Add the `environment` column to `atlas-tenants`

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/entity.go` — add `Environment string`
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/model.go` — the immutable model gains `environment` + accessor
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/builder.go` and `entity_builder.go` — the builders gain `SetEnvironment`
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/rest.go` — the REST model gains the attribute
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/kafka.go` — the emitted event body gains it
- Create: `services/atlas-tenants/atlas.com/tenants/tenant/environment_migration.go` — the backfill
- Modify: `services/atlas-tenants/atlas.com/tenants/main.go` — register the backfill migration
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/provider_test.go` (or the nearest existing test file) — round-trip test
- Read-only: `services/atlas-tenants/atlas.com/tenants/test/database.go` — the test database harness

Module root: `services/atlas-tenants/atlas.com/tenants`.

**Interfaces:**
- Produces: `Model.Environment() string`, `Builder.SetEnvironment(string)`, and
  the `environment` attribute on the tenant REST model and Kafka event. Task 21
  projects this attribute into the in-memory tenant→environment map.

- [ ] **Step 1: Write the failing round-trip test**

```go
func TestTenantEnvironmentRoundTrips(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), context.Background(), db)

	m, err := p.Create(NewBuilder().SetRegion("GMS").SetMajorVersion(83).
		SetMinorVersion(1).SetEnvironment("pr-123").Build())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := p.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Environment() != "pr-123" {
		t.Fatalf("Environment() = %q, want \"pr-123\"", got.Environment())
	}
}
```

Adapt the constructor/processor names to what `tenant/processor.go` actually
exposes — read it first.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./tenant/ -run Environment -v`
Expected: FAIL — `SetEnvironment` undefined.

- [ ] **Step 3: Thread the field through entity → model → builder → rest → kafka**

Follow the existing shape of `Region` in each of those five files exactly.
The column carries `gorm:"not null;default:''"` for the same reason as Task 11.

- [ ] **Step 4: Add the backfill**

Same shape as Task 11 Step 4, over the `tenants` table in this service's
database, registered in this service's `database.SetMigrations` list.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-tenants
git commit -m "feat(atlas-tenants): add the environment dimension to the tenant table"
```

---

### Task 13: Environment-scoped reads and writes in `atlas-configurations`

The schema exists (Task 11); this task makes it load-bearing. Two strategies,
split by row kind (design §8.1): **strict** for `services` and `tenants`,
**baseline fallback** for `templates` (deviation V3).

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/provider.go` — strict environment filter on every read
- Modify: `services/atlas-configurations/atlas.com/configurations/tenants/administrator.go` — set environment on write; reject cross-environment writes
- Modify: `services/atlas-configurations/atlas.com/configurations/services/provider.go` and `services/administrator.go` — same
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/provider.go` — my environment's row if present, else the baseline's
- Modify: `services/atlas-configurations/atlas.com/configurations/templates/administrator.go` — strict write
- Create: `services/atlas-configurations/atlas.com/configurations/scope/scope.go` — `ErrCrossEnvironmentWrite`, `Strict`, `AuthorizeWrite`
- Create: `services/atlas-configurations/atlas.com/configurations/scope/scope_test.go`
- Create: `services/atlas-configurations/atlas.com/configurations/templates/overlay.go` — the key-aware template overlay (Step 4a)
- Create: `services/atlas-configurations/atlas.com/configurations/templates/overlay_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/rest/handler.go` — surface `ErrCrossEnvironmentWrite` as a 403
- Read-only: `libs/atlas-env/env.go` (Task 17) — `env.FromContext`
- Read-only: `libs/atlas-rest/server/register.go` — where the environment lands on the request context (Task 22)

Module root: `services/atlas-configurations/atlas.com/configurations`.

**Ordering:** depends on Tasks 11, 17 and 22.

**Interfaces:**
- Produces:
  ```go
  package scope

  var ErrCrossEnvironmentWrite = errors.New("write targets another environment")

  // Strict filters db to rows owned by e. An empty e is the legacy value
  // and applies no filter (FR-1.8).
  func Strict(db *gorm.DB, e env.Id) *gorm.DB

  // AuthorizeWrite returns ErrCrossEnvironmentWrite when target != caller.
  func AuthorizeWrite(caller env.Id, target env.Id) error
  ```
  and, in the `templates` package (not `scope` — the overlay key is
  templates-specific):
  ```go
  // OverlaySingle scopes a lookup of ONE version key: e's row if present,
  // else the baseline's.
  func OverlaySingle(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB

  // OverlayCollection scopes a Find/paged read to e's rows PLUS the baseline
  // rows whose version key e has no row for. An anti-join, not an ORDER BY.
  func OverlayCollection(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB

  // VisibleById scopes a UUID lookup to rows e may read: its own or its
  // baseline's. A UUID is unique, so there is nothing to fall back to.
  func VisibleById(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB
  ```

**Why the fallback is not a generic GORM scope.** A `CASE`-based `ORDER BY`
resolves a `First()` on one exact `(region, major_version, minor_version)`
key, and nothing else. Applied to a collection read it returns the union, not
the overlay:

```
main:    GMS 83.1, GMS 95.1
pr-123:  GMS 83.1

union   -> 3 rows: main/83.1, main/95.1, pr-123/83.1   ← wrong
overlay -> 2 rows: pr-123/83.1, main/95.1              ← required
```

The three read paths in `templates/provider.go` are distinct and each needs
its own treatment. Read the file first — these are the only paths, so the API
above is complete rather than open-ended:

| Provider | Query | Treatment |
|---|---|---|
| `byRegionVersionEntityProvider` | `WHERE region=? AND major_version=? AND minor_version=?` → `First()` | `OverlaySingle` |
| `getAll` | `database.PagedQuery[Entity]` | `OverlayCollection` (anti-join) |
| `byIdEntityProvider` | `WHERE id=?` → `First()` | `VisibleById` |

- [ ] **Step 1: Write the failing scope tests**

`scope/scope_test.go`:

```go
func TestStrictWithEmptyEnvironmentAppliesNoFilter(t *testing.T) {
	db := testDatabase(t)
	seedTemplates(t, db, "main", "pr-123")

	var rows []templates.Entity
	if err := Strict(db.Model(&templates.Entity{}), env.Id("")).Find(&rows).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("legacy read returned %d rows, want 2 (unfiltered)", len(rows))
	}
}

func TestStrictFiltersToTheCallersEnvironment(t *testing.T) {
	db := testDatabase(t)
	seedTemplates(t, db, "main", "pr-123")

	var rows []templates.Entity
	if err := Strict(db.Model(&templates.Entity{}), env.Id("pr-123")).Find(&rows).Error; err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(rows) != 1 || rows[0].Environment != "pr-123" {
		t.Fatalf("got %d rows (%v), want exactly pr-123's", len(rows), rows)
	}
}

func TestAuthorizeWriteRejectsAnotherEnvironment(t *testing.T) {
	if err := AuthorizeWrite(env.Id("pr-123"), env.Id("main")); !errors.Is(err, ErrCrossEnvironmentWrite) {
		t.Fatalf("got %v, want ErrCrossEnvironmentWrite", err)
	}
	if err := AuthorizeWrite(env.Id("pr-123"), env.Id("pr-123")); err != nil {
		t.Fatalf("same-environment write rejected: %v", err)
	}
	if err := AuthorizeWrite(env.Id(""), env.Id("")); err != nil {
		t.Fatalf("legacy write rejected: %v", err)
	}
}
```

- [ ] **Step 2: Write the failing template-fallback test**

```go
func TestTemplatesFallBackToTheBaselineRow(t *testing.T) {
	db := testDatabase(t)
	// Only main has a v83.1 template.
	seedTemplate(t, db, "main", "GMS", 83, 1)

	got, err := templates.NewProcessor(testLogger(t), envContext(t, "pr-123"), db).
		GetByVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("GetByVersion: %v", err)
	}
	if got.Environment != "main" {
		t.Fatalf("pr-123 got environment %q, want the baseline's row", got.Environment)
	}
}

func TestTemplatesPreferTheOwnEnvironmentRow(t *testing.T) {
	db := testDatabase(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := templates.NewProcessor(testLogger(t), envContext(t, "pr-123"), db).
		GetByVersion("GMS", 83, 1)
	if err != nil {
		t.Fatalf("GetByVersion: %v", err)
	}
	if got.Environment != "pr-123" {
		t.Fatalf("got environment %q, want pr-123's own row to win", got.Environment)
	}
}

func TestTemplateCollectionIsAnOverlayNotAUnion(t *testing.T) {
	// The case an ORDER BY cannot express. main ships two versions; pr-123
	// overrides one of them. The collection read must return pr-123's 83.1
	// and main's 95.1 — two rows, not three.
	db := testDatabase(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "main", "GMS", 95, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := templates.NewProcessor(testLogger(t), envContext(t, "pr-123"), db).
		GetAll(model.Page{Number: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("got %d rows, want 2 (overlay, not union): %+v", len(got.Items), got.Items)
	}
	byVersion := map[string]string{}
	for _, e := range got.Items {
		byVersion[fmt.Sprintf("%s%d.%d", e.Region, e.MajorVersion, e.MinorVersion)] = e.Environment
	}
	if byVersion["GMS83.1"] != "pr-123" {
		t.Fatalf("GMS83.1 came from %q, want pr-123's overriding row", byVersion["GMS83.1"])
	}
	if byVersion["GMS95.1"] != "main" {
		t.Fatalf("GMS95.1 came from %q, want the inherited baseline row", byVersion["GMS95.1"])
	}
}

func TestTemplateCollectionOnMainIsUnchanged(t *testing.T) {
	// NG6/NFR-7: the baseline's own collection read must be byte-identical to
	// today's — it inherits from nothing and must not see other environments.
	db := testDatabase(t)
	seedTemplate(t, db, "main", "GMS", 83, 1)
	seedTemplate(t, db, "main", "GMS", 95, 1)
	seedTemplate(t, db, "pr-123", "GMS", 83, 1)

	got, err := templates.NewProcessor(testLogger(t), envContext(t, "main"), db).
		GetAll(model.Page{Number: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("main saw %d rows, want its own 2", len(got.Items))
	}
	for _, e := range got.Items {
		if e.Environment != "main" {
			t.Fatalf("main's collection read returned a %q row", e.Environment)
		}
	}
}

func TestTemplateByIdRejectsAnotherEnvironmentsRow(t *testing.T) {
	// A UUID is unique, so there is nothing to fall back to. pr-123 may read
	// its own rows and its baseline's (templates are a shared read-only
	// source), and nothing else.
	db := testDatabase(t)
	mine := seedTemplate(t, db, "pr-123", "GMS", 83, 1)
	inherited := seedTemplate(t, db, "main", "GMS", 95, 1)
	foreign := seedTemplate(t, db, "pr-999", "GMS", 83, 1)

	p := templates.NewProcessor(testLogger(t), envContext(t, "pr-123"), db)
	if _, err := p.GetById(mine.Id); err != nil {
		t.Fatalf("own row: %v", err)
	}
	if _, err := p.GetById(inherited.Id); err != nil {
		t.Fatalf("baseline row: %v", err)
	}
	if _, err := p.GetById(foreign.Id); err == nil {
		t.Fatal("read another environment's template by id; want not-found")
	}
}
```

- [ ] **Step 3: Run to verify they fail**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./scope/ ./templates/ -v`
Expected: FAIL — package `scope` does not exist.

- [ ] **Step 4: Implement `scope`**

```go
func Strict(db *gorm.DB, e env.Id) *gorm.DB {
	if e == "" {
		return db // legacy: unfiltered, byte-identical to pre-change behaviour
	}
	return db.Where("environment = ?", string(e))
}

func AuthorizeWrite(caller env.Id, target env.Id) error {
	if caller == target {
		return nil
	}
	return fmt.Errorf("%w: caller=%q target=%q", ErrCrossEnvironmentWrite, caller, target)
}
```

- [ ] **Step 4a: Implement the templates overlay**

`services/atlas-configurations/atlas.com/configurations/templates/overlay.go`.
Note this lives in `templates`, not `scope`: the overlay is defined in terms
of the template version key, which no other table shares.

```go
// overlayKey is the identity a template row is unique on. Fallback is
// defined ONLY in terms of this key: a baseline row is visible to e exactly
// when e has no row with the same key. Templates are keyed by a VERSION, not
// by an environment, and the PR bootstrap already treats them as a shared
// read-only source it clones from (design V3).
var overlayKey = []string{"region", "major_version", "minor_version"}

// OverlaySingle scopes a lookup of one version key. e's row wins; the
// baseline's fills in when e has none.
func OverlaySingle(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	return db.Where("environment IN (?, ?)", string(e), string(baseline)).
		Order(clause.Expr{
			SQL:  "CASE WHEN environment = ? THEN 0 ELSE 1 END",
			Vars: []interface{}{string(e)},
		})
}

// OverlayCollection scopes a Find/paged read: e's rows, plus the baseline
// rows whose version key e has no row for. This is an anti-join — an ORDER BY
// cannot express it, because a collection read returns every matching row
// rather than the first.
//
// NOT EXISTS rather than DISTINCT ON: it composes with database.PagedQuery's
// LIMIT/OFFSET, and it runs on both Postgres and the sqlite test harness.
func OverlayCollection(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	anti := `environment = ? OR (environment = ? AND NOT EXISTS (
	           SELECT 1 FROM templates o
	           WHERE o.environment = ?
	             AND o.region = templates.region
	             AND o.major_version = templates.major_version
	             AND o.minor_version = templates.minor_version))`
	return db.Where(anti, string(e), string(baseline), string(e))
}

// VisibleById scopes a UUID lookup. A UUID is unique, so there is nothing to
// fall back to — this is a visibility rule, not an overlay: e may read its
// own rows and its baseline's, and nothing else.
func VisibleById(db *gorm.DB, e env.Id, baseline env.Id) *gorm.DB {
	if e == "" || e == baseline {
		return scope.Strict(db, e)
	}
	return db.Where("environment IN (?, ?)", string(e), string(baseline))
}
```

Verify the table name in the anti-join matches `Entity.TableName()`
(`"templates"`) and that GORM does not alias it in the paged query — if
`database.PagedQuery` introduces an alias, the correlated reference must use
that alias instead. Check by logging the generated SQL in the test, not by
assuming.

Wire each provider to its matching helper per the table in the Interfaces
block. `templates/administrator.go` stays **strict** on every write: a PR
overrides a template by inserting its own row, never by updating the
baseline's.

- [ ] **Step 5: Apply the filters at every read and write path**

Every function in the four `provider.go` / `administrator.go` files that
builds a query goes through `scope.Strict` (or, in `templates` only, the
matching overlay helper from Step 4a);
every write calls `scope.AuthorizeWrite(callerEnv, targetEnv)` first and sets
`Environment: string(callerEnv)` on insert. The caller's environment comes
from `env.FromContext(ctx)` — never from a request body.

- [ ] **Step 6: Map the error to a 403 in the REST layer**

In `rest/handler.go`, add `ErrCrossEnvironmentWrite` to the error mapping so
it surfaces as `403 Forbidden`, not a 500.

- [ ] **Step 7: Write the C2 regression test (FR-7.8, G7)**

The negative test design §8.2 calls "what makes G7 verified rather than
intended" — a `pr-*` PATCH against a `main`-owned service row is rejected and
the row is byte-identical afterwards:

```go
func TestPrEnvironmentCannotPatchAMainOwnedServiceRow(t *testing.T) {
	db := testDatabase(t)
	before := seedService(t, db, "main", services.ServiceTypeLogin)

	err := services.NewProcessor(testLogger(t), envContext(t, "pr-123"), db).
		Update(before.Id, patchWithNewTenant(t))
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("got %v, want ErrCrossEnvironmentWrite", err)
	}

	var after services.Entity
	if err := db.First(&after, "id = ?", before.Id).Error; err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if !bytes.Equal(after.Data, before.Data) {
		t.Fatalf("main's row changed:\n before %s\n after  %s", before.Data, after.Data)
	}
}
```

- [ ] **Step 8: Run the module tests**

Run: `cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...`
Expected: PASS, including the C2 regression test.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-configurations
git commit -m "feat(atlas-configurations): environment-scoped control-plane reads and writes"
```

---

### Task 14: Environment-scoped reads and writes in `atlas-tenants`

`AllProvider(page)` is the function F13's autonomous-loop pattern calls;
making it environment-filtered is what stops a baseline pod from picking up
an ephemeral tenant it does not own. (Correction, controller addendum #2:
`atlas-tenants`' `tenant.Processor` has no `GetAll()` method — the real
interface method is `AllProvider(page model.Page) model.Provider[model.Paged[Model]]`,
confirmed at `services/atlas-tenants/atlas.com/tenants/tenant/processor.go:19,45`.
The shipped tests in `services/atlas-tenants/atlas.com/tenants/tenant/scope_test.go`
correctly target `AllProvider`.)

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/provider.go` — environment filter on every read
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/administrator.go` — set environment on write; reject cross-environment writes
- Modify: `services/atlas-tenants/atlas.com/tenants/tenant/processor.go` — `AllProvider` gains the filter
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/provider.go` and `configuration/administrator.go` — same treatment
- Modify: `services/atlas-tenants/atlas.com/tenants/rest/handler.go` — 403 mapping
- Read-only: `services/atlas-configurations/atlas.com/configurations/scope/scope.go` (Task 13) — copy the two helpers into this service rather than importing across a service boundary

Module root: `services/atlas-tenants/atlas.com/tenants`.

**Note on duplication:** copying `scope.go` into `atlas-tenants` is
deliberate. Per CLAUDE.md ("don't break service boundaries by having one
layer call another's internals directly"), one service must not import
another's package. Three ten-line functions is the right amount of
duplication; do not create a shared library for it.

**Interfaces:**
- Consumes: `env.FromContext` (Task 17), the `Environment` field (Task 12).
- Produces: `Processor.AllProvider(page)` returns only tenants of the
  caller's environment. Task 27's `ForEachOwnedEnvironment` depends on this
  filter. (Correction, controller addendum #2: not `GetAll()` — see the note
  above the Files block.)

- [ ] **Step 1: Write the failing `AllProvider` filter test**

Shipped as `services/atlas-tenants/atlas.com/tenants/tenant/scope_test.go`
(correction, controller addendum #2 — reproduced here matching the real
`AllProvider(page)`/`model.Page`/`model.Paged` call shape, not `GetAll()`):

```go
func TestGetAllReturnsOnlyTheCallersEnvironmentsTenants(t *testing.T) {
	db := testDatabase(t)
	seedTenant(t, db, "main")
	seedTenant(t, db, "pr-123")

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	paged, err := NewProcessor(testLogger(t), ctx, db).AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 1 || paged.Items[0].Environment() != "pr-123" {
		t.Fatalf("got %d tenants (%v), want pr-123's only", len(paged.Items), paged.Items)
	}
}

func TestGetAllWithLegacyEnvironmentReturnsEverything(t *testing.T) {
	db := testDatabase(t)
	seedTenant(t, db, "main")
	seedTenant(t, db, "pr-123")

	paged, err := NewProcessor(testLogger(t), context.Background(), db).AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 2 {
		t.Fatalf("legacy GetAll returned %d, want 2 (FR-1.8)", len(paged.Items))
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./tenant/ -run GetAll -v`
Expected: FAIL — the first test returns 2.

- [ ] **Step 3: Copy `scope.go` into this service and apply it**

Create `services/atlas-tenants/atlas.com/tenants/scope/scope.go` with
`ErrCrossEnvironmentWrite`, `Strict` and `AuthorizeWrite`, copied from Task 13
Step 4. No overlay helpers: nothing in `atlas-tenants` falls back — a tenant
belongs to exactly one environment (FR-7.1). Apply `Strict` to every read and
`AuthorizeWrite` + `Environment:` to every write.

- [ ] **Step 4: Run the module tests**

Run: `cd services/atlas-tenants/atlas.com/tenants && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-tenants
git commit -m "feat(atlas-tenants): environment-scoped tenant reads and writes"
```

---

### Task 15: The scope guards — `control-plane-scope-guard` and `tenant-scope-guard`

FR-8.5: a new unscoped entity must fail CI, so the Phase A audit cannot
regress while Phases B–F are in flight.

**Files:**
- Create: `tools/scopeguard/main.go`, `tools/scopeguard/go.mod` — the analyzer
- Create: `tools/scopeguard/testdata/src/{unscopeddata,unscopedcontrol,allowlisted}/main.go` — fixtures
- Create: `tools/scopeguard/allowlist.txt` — seeded from the `TRANSITIVE` and `FORCES-ISOLATED` rows of `query-scope-audit.md`
- Create: `tools/scope-guard.sh` — the shell entry point, mirroring `tools/redis-key-guard.sh`
- Modify: `tools/verify.sh` — add the step beside the other guards (see line 293's `touched` gate pattern)
- Read-only: `tools/rediskeyguard/` — the analyzer shape to copy
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — the allowlist source

**Interfaces:**
- Produces: `tools/scope-guard.sh`, exit 0 clean. Two rules:
  1. a `services/*/**/entity.go` primary `Entity` struct with no `TenantId`
     field, in a service not in `{atlas-configurations, atlas-tenants}`,
     fails unless allowlisted with a reason;
  2. a `services/atlas-{configurations,tenants}/**/entity.go` primary
     `Entity` struct with no `Environment` field fails, no allowlist.

- [ ] **Step 1: Write the failing fixtures**

`testdata/src/unscopeddata/main.go`:

```go
package unscopeddata

type Entity struct { // want `data-plane entity without TenantId`
	Id   uint32
	Name string
}
```

`testdata/src/unscopedcontrol/main.go`:

```go
package unscopedcontrol

type Entity struct { // want `control-plane entity without Environment`
	Id   uint32
	Type string
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd tools/scopeguard && go test ./...`
Expected: FAIL — no analyzer.

- [ ] **Step 3: Implement the analyzer**

Copy `tools/rediskeyguard`'s `analysis.Analyzer` skeleton. Inspect
`*ast.TypeSpec` whose name is `Entity`, in a file named `entity.go`, and
check its field names. Plane is decided by the file path, not by heuristics.

- [ ] **Step 4: Seed the allowlist from the audit**

Every entry is `<repo-relative path> # <reason>`. An entry with no reason is
a lint failure of the allowlist file itself — assert that in a test.

- [ ] **Step 5: Wire into `tools/verify.sh`**

Beside the existing guards, gated on a changed `entity.go`:

```sh
if touched '^services/.*/entity\.go$|^tools/scopeguard/'; then
    step "scope guard" ./tools/scope-guard.sh
else
    skip "scope guard (no entity changed)"
fi
```

- [ ] **Step 6: Run the guard against the fleet**

Run: `./tools/scope-guard.sh`
Expected: exit 0. Anything it reports that the audit did not is an audit gap
— fix the code or add the allowlist entry with its reason, and correct the
audit document.

- [ ] **Step 7: Commit**

```bash
git add tools/scopeguard tools/scope-guard.sh tools/verify.sh docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "feat(tools): scope guard for unscoped data-plane and control-plane entities"
```

---

### Task 16: Phase A gate — run the full verification and record the prerequisite sign-off

FR-8 is blocking: nothing in Phase B–F is safe until this passes.

**Files:**
- Modify: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — add the `## 4. Sign-off` section
- Read-only: `tools/verify.sh`, `docs/verification.md`

- [ ] **Step 1: Run the flagless verification gate**

Run: `tools/verify.sh --base <the branch-point commit>`
Expected: exit 0. Launch it in the background and do other work; do not idle
waiting on it. `--quick` does **not** count — the bake step is what catches a
missing `COPY libs/...` for the new `libs/atlas-redis` dependency.

- [ ] **Step 2: Confirm each FR-8 acceptance criterion against evidence**

Write the sign-off section with one line per criterion and the artifact that
proves it:

```markdown
## 4. Sign-off (PRD §12 prerequisites)

| Criterion | Evidence |
|---|---|
| FR-8.1 every query path scoped | §1 of this document, 89 entities, N `UNSCOPED` remaining (must be 0) |
| FR-8.2 control-plane environment dimension | Tasks 11–12; `main` resolution unchanged proven by the `Strict(db, "")` legacy tests |
| FR-8.3 data-plane Redis on the tenant-scoped API | Tasks 4–7; `atlas-storage` defect fixed in its own commit |
| FR-8.4 bare constructors withdrawn | Task 9; allowlist has N entries, each with a reason |
| FR-8.5 guards fail CI on regression | Tasks 9, 15; `tools/verify.sh` steps present |
| FR-8.6 every deployment-scoped resource dispositioned | §2 of this document |
```

If any row cannot be filled with an artifact, Phase A is not done. Say so and
stop — do not proceed into Phase B on an unfilled row.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "docs(task-232): Phase A prerequisite sign-off"
```

---
# Phase B — Libraries (every mechanism inert until Phase F)

Everything in this phase must be a no-op on `main`. The FR-1.8 test in each
task is not a formality — it is what makes the 64-service migration in Phase C
safe to merge incrementally (design §12).

---

### Task 17: `libs/atlas-env` — environment on the context

**Files:**
- Create: `libs/atlas-env/go.mod` — module `github.com/Chronicle20/atlas/libs/atlas-env`, `go 1.25.5`, requiring only `github.com/Chronicle20/atlas/libs/atlas-model`
- Create: `libs/atlas-env/env.go`
- Create: `libs/atlas-env/env_test.go`
- Create: `libs/atlas-env/.gitignore` — copy `libs/atlas-tenant/.gitignore`
- Create: `libs/atlas-env/README.md` — one paragraph; copy the shape of `libs/atlas-tenant/README.md`
- Modify: `go.work` — add `./libs/atlas-env`
- Read-only: `libs/atlas-tenant/processor.go:12-17,56-97` — the idiom this mirrors exactly
- Read-only: `docs/adding-a-new-service.md` — the checklist for a new module in this repo (`go.work`, `docker-bake.hcl`, Dockerfile `COPY libs/...`), of which only the `go.work` entry applies to a library

Module root: `libs/atlas-env`.

**Interfaces:**
- Produces (every later task in Phases B–F consumes these):
  ```go
  package env

  const Key = "ENVIRONMENT" // context key AND REST/Kafka header name

  type Id string

  func WithContext(ctx context.Context, id Id) context.Context
  func FromContext(ctx context.Context) model.Provider[Id]
  func MustFromContext(ctx context.Context) Id
  func Self() Id
  func Valid(id Id) bool
  ```

- [ ] **Step 1: Write the failing context round-trip test**

`libs/atlas-env/env_test.go`:

```go
package env

import (
	"context"
	"testing"
)

func TestWithContextRoundTrips(t *testing.T) {
	ctx := WithContext(context.Background(), Id("pr-123"))
	got, err := FromContext(ctx)()
	if err != nil {
		t.Fatalf("FromContext: %v", err)
	}
	if got != Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\"", got)
	}
}

func TestFromContextWithNoEnvironmentIsTheLegacyValue(t *testing.T) {
	// FR-1.8: absence is not an error. A context with no environment is a
	// legacy operation and resolves to the empty id, which every registry
	// query answers with "the local deployment owns it".
	got, err := FromContext(context.Background())()
	if err != nil {
		t.Fatalf("FromContext on a bare context returned an error: %v", err)
	}
	if got != Id("") {
		t.Fatalf("got %q, want the empty id", got)
	}
}

func TestSelfReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "pr-123")
	if got := Self(); got != Id("pr-123") {
		t.Fatalf("Self() = %q, want \"pr-123\"", got)
	}
}

func TestSelfWithNoVariableIsTheLegacyValue(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "")
	if got := Self(); got != Id("") {
		t.Fatalf("Self() = %q, want the empty id", got)
	}
}

func TestValid(t *testing.T) {
	for _, ok := range []Id{"main", "pr-123", "staging-2", ""} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false, want true", ok)
		}
	}
	for _, bad := range []Id{"PR-123", "pr_123", "-pr", "pr-", "a", "x/y"} {
		if Valid(bad) {
			t.Errorf("Valid(%q) = true, want false", bad)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-env && go test ./...`
Expected: FAIL — no such module.

- [ ] **Step 3: Implement `env.go`**

```go
// Package env carries the execution environment of an operation on
// context.Context, exactly as libs/atlas-tenant carries the tenant.
//
// Environment is a property of the OPERATION, not of the deployment
// processing it. A baseline pod may process an operation belonging to an
// ephemeral environment; it reads the environment off the operation and
// hands it back out unchanged.
//
// This package is deliberately a leaf: it depends only on atlas-model, so
// atlas-kafka and atlas-rest can import it without a module cycle. The
// registry implementation that this package's Registry interface describes
// is populated in libs/atlas-service, which owns the Kafka projection.
package env

import (
	"context"
	"os"
	"regexp"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Key is both the context key and the REST/Kafka header name, matching
// libs/atlas-tenant's ID = "TENANT_ID" convention.
const Key = "ENVIRONMENT"

// SelfVar is the environment variable a pod reads its own environment from.
const SelfVar = "ATLAS_ENVIRONMENT"

// Id identifies one execution environment ("main", "pr-123"). The empty Id
// is the legacy value: it means "not environment-aware" and every registry
// query answers it with the local deployment (FR-1.8).
type Id string

// idPattern constrains ids at INGEST only (task-232 P2). Operations are
// never revalidated: a record that entered the registry is trusted, and a
// per-operation regex would be I/O-free but pointless work on the hot path.
var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}[a-z0-9]$`)

// Valid reports whether id is a well-formed environment id. The empty id is
// valid — it is the legacy value.
func Valid(id Id) bool {
	if id == "" {
		return true
	}
	return idPattern.MatchString(string(id))
}

func WithContext(ctx context.Context, id Id) context.Context {
	return context.WithValue(ctx, Key, id)
}

// FromContext never errors: a context with no environment yields the empty
// id, which is the legacy value (FR-1.8). It returns a Provider anyway so
// callers compose with the rest of the atlas-model pipeline.
func FromContext(ctx context.Context) model.Provider[Id] {
	id, _ := ctx.Value(Key).(Id)
	return model.FixedProvider(id)
}

func MustFromContext(ctx context.Context) Id {
	id, _ := FromContext(ctx)()
	return id
}

// Self is this process's own environment, read from ATLAS_ENVIRONMENT. It
// never fails and never consults the registry: a pod knows its own
// environment even during a registry outage, which is what keeps main fully
// functional when the projection is unavailable (design §4.3).
func Self() Id {
	return Id(os.Getenv(SelfVar))
}
```

`context.WithValue` with a plain `string` key is what `libs/atlas-tenant`
does (`processor.go:92-95`). Match it rather than introducing a private key
type — a mismatch would make the two packages' context conventions diverge.

- [ ] **Step 4: Run the tests**

Run: `cd libs/atlas-env && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Register the module in `go.work`**

Add `./libs/atlas-env` in sorted position. Run `go work sync` is **not**
required (see `tools/lint.sh` header); just verify:

Run: `go build ./libs/atlas-env/...`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-env go.work
git commit -m "feat(atlas-env): environment on the context"
```

---

### Task 18: `libs/atlas-env` — the registry interface, the in-memory implementation, and staleness

FR-1's four queries, all resolving from an in-memory map with no I/O.

**Files:**
- Create: `libs/atlas-env/registry.go`
- Create: `libs/atlas-env/registry_test.go`
- Create: `libs/atlas-env/record.go` — the environment record type the projection decodes into
- Read-only: `libs/atlas-env/env.go` (Task 17)
- Read-only: `design.md` §4.2–4.4 — the query semantics and the record shape

Module root: `libs/atlas-env`.

**Interfaces:**
- Produces:
  ```go
  // Record is one environment as published on the environment-status topic.
  type Record struct {
      Name      Id                `json:"name"`
      Baseline  Id                `json:"baseline"`
      Namespace string            `json:"namespace"`
      Tenant    string            `json:"tenant"`    // uuid string; "" for main
      Overrides map[string]string `json:"overrides"` // service name -> namespace
      Phase     string            `json:"phase"`     // PROVISIONING|ACTIVE|DEACTIVATING|DELETED
  }

  type Registry interface {
      // EnvironmentNamespace is the environment's OWN namespace — where its
      // own ingress lives. Never falls back to the baseline.
      EnvironmentNamespace(e Id) (string, error)
      // ServiceNamespace is the namespace of the effective implementation of
      // service for e: the override's namespace if e overrides it, else the
      // baseline's namespace. FR-1.2.
      ServiceNamespace(e Id, service string) (string, error)
      EnvironmentsOwnedBy(service string) []Id        // FR-1.3
      IsOwner(e Id, service string) bool              // FR-1.4
      IsActive(e Id) bool                             // FR-1.4
      Stale() bool                                    // FR-1.7
  }

  func NewMapRegistry(self Id, clock func() time.Time) *MapRegistry
  func (r *MapRegistry) Apply(rec Record)
  func (r *MapRegistry) ApplyTombstone(name Id)
  func (r *MapRegistry) Observe(at time.Time) // heartbeat / any record seen

  func SetRegistry(r Registry)
  func CurrentRegistry() Registry // never nil; a legacy no-op registry before SetRegistry
  ```
- The `PhaseActive` etc. constants live in `record.go`.

**Two namespace queries, not one — this distinction is load-bearing.**
`EnvironmentNamespace` and `ServiceNamespace` answer different questions and
must not be collapsed:

| Query | `pr-123` (overrides `atlas-character` only) |
|---|---|
| `EnvironmentNamespace("pr-123")` | `atlas-pr-123` — always, override or not |
| `ServiceNamespace("pr-123", "atlas-character")` | `atlas-pr-123` |
| `ServiceNamespace("pr-123", "atlas-monsters")` | `atlas-main` |

Task 23's outbound REST resolution uses **`EnvironmentNamespace`**, because
every sparse environment deploys its own ingress and the per-service
override/baseline decision is made *inside* that ingress by the `NS_*`
routing table (Task 43). Using `ServiceNamespace(e, "atlas-ingress")` there
would be a bug: `pr-123` does not override `atlas-ingress` in its record's
`overrides` map, so the query would fall back to `atlas-main` and a baseline
pod handling a `pr-123` operation would send its next call into `main` —
exactly the leak this project exists to close.

`ServiceNamespace` exists for the ingress-configuration generation and for
diagnostics. Nothing on the REST hot path calls it.

**Design notes bound into the implementation:**
- `EnvironmentsOwnedBy` takes only the service name; a process only ever asks
  about itself and `env.Self()` plus `SERVICE_NAME` already identify it
  (design §4.2). Do not add a deployment parameter.
- The baseline is `Record.Baseline`, never the literal `"main"` (FR-1.5).
- Staleness: heartbeat every 30 s, `Stale()` true after 120 s without one
  (design §4.3). `Self()`'s own environment is exempt from every fail-closed
  path — a pod's own environment comes from an env var and cannot go stale.

- [ ] **Step 1: Write the failing registry tests**

`libs/atlas-env/registry_test.go`:

```go
package env

import (
	"testing"
	"time"
)

func active(name, baseline, ns string, overrides map[string]string) Record {
	return Record{Name: Id(name), Baseline: Id(baseline), Namespace: ns,
		Overrides: overrides, Phase: PhaseActive}
}

func TestLegacyEnvironmentResolvesToTheLocalDeployment(t *testing.T) {
	// FR-1.8: with no records, every query returns the legacy answer.
	r := NewMapRegistry(Id("main"), time.Now)

	if !r.IsOwner(Id(""), "atlas-monsters") {
		t.Fatal("IsOwner(\"\") = false, want true (legacy owns everything)")
	}
	if !r.IsActive(Id("")) {
		t.Fatal("IsActive(\"\") = false, want true")
	}
	if got := r.EnvironmentsOwnedBy("atlas-monsters"); len(got) != 1 || got[0] != Id("") {
		t.Fatalf("EnvironmentsOwnedBy = %v, want [\"\"]", got)
	}
}

func TestServiceNamespaceFallsBackToTheBaseline(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-123", "main", "atlas-pr-123", map[string]string{
		"atlas-character": "atlas-pr-123",
	}))

	got, err := r.ServiceNamespace(Id("pr-123"), "atlas-character")
	if err != nil || got != "atlas-pr-123" {
		t.Fatalf("override: got (%q, %v), want (\"atlas-pr-123\", nil)", got, err)
	}
	got, err = r.ServiceNamespace(Id("pr-123"), "atlas-monsters")
	if err != nil || got != "atlas-main" {
		t.Fatalf("fallback: got (%q, %v), want (\"atlas-main\", nil)", got, err)
	}
}

func TestEnvironmentNamespaceNeverFallsBackToTheBaseline(t *testing.T) {
	// The bug this test exists to prevent: an environment's OWN namespace is
	// where its OWN ingress lives. It is not subject to the per-service
	// override/baseline decision, because the record's `overrides` map does
	// not (and must not) list atlas-ingress. If this returned the baseline's
	// namespace, a baseline pod handling a pr-123 operation would send its
	// next REST call into main and the operation would silently change
	// environment mid-flight (G4).
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-123", "main", "atlas-pr-123", map[string]string{
		"atlas-character": "atlas-pr-123", // note: no atlas-ingress entry
	}))

	got, err := r.EnvironmentNamespace(Id("pr-123"))
	if err != nil || got != "atlas-pr-123" {
		t.Fatalf("got (%q, %v), want (\"atlas-pr-123\", nil)", got, err)
	}

	// And the contrast, stated explicitly so a future refactor that collapses
	// the two queries fails here rather than in production.
	svcNs, err := r.ServiceNamespace(Id("pr-123"), "atlas-ingress")
	if err != nil {
		t.Fatalf("ServiceNamespace: %v", err)
	}
	if svcNs == got {
		t.Fatal("ServiceNamespace(e, \"atlas-ingress\") happens to equal EnvironmentNamespace(e); " +
			"the fixture no longer distinguishes the two queries — fix the fixture, not the assertion")
	}
}

func TestNamespaceQueriesNeverHardCodeMain(t *testing.T) {
	// FR-1.5: a second baseline must require no code change.
	r := NewMapRegistry(Id("staging"), time.Now)
	r.Apply(active("staging", "staging", "atlas-staging", nil))
	r.Apply(active("pr-9", "staging", "atlas-pr-9", nil))

	got, err := r.ServiceNamespace(Id("pr-9"), "atlas-monsters")
	if err != nil || got != "atlas-staging" {
		t.Fatalf("ServiceNamespace: got (%q, %v), want (\"atlas-staging\", nil)", got, err)
	}
	got, err = r.EnvironmentNamespace(Id("pr-9"))
	if err != nil || got != "atlas-pr-9" {
		t.Fatalf("EnvironmentNamespace: got (%q, %v), want (\"atlas-pr-9\", nil)", got, err)
	}
}

func TestNamespaceQueriesOfAnUnknownEnvironmentError(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))

	if _, err := r.ServiceNamespace(Id("pr-999"), "atlas-monsters"); err == nil {
		t.Fatal("ServiceNamespace resolved an unknown environment; want an error (D4 fail closed)")
	}
	if _, err := r.EnvironmentNamespace(Id("pr-999")); err == nil {
		t.Fatal("EnvironmentNamespace resolved an unknown environment; want an error (D4 fail closed)")
	}
}

func TestIsOwnerIsExactlyOneDeployment(t *testing.T) {
	// FR-4.6: for a given (environment, service) exactly one deployment
	// satisfies IsOwner, and every pod projects the same log.
	overrides := map[string]string{"atlas-character": "atlas-pr-123"}
	baselinePod := NewMapRegistry(Id("main"), time.Now)
	overridePod := NewMapRegistry(Id("pr-123"), time.Now)
	for _, r := range []*MapRegistry{baselinePod, overridePod} {
		r.Apply(active("main", "main", "atlas-main", nil))
		r.Apply(active("pr-123", "main", "atlas-pr-123", overrides))
	}

	if baselinePod.IsOwner(Id("pr-123"), "atlas-character") {
		t.Fatal("baseline claims ownership of an overridden service (FR-6.3)")
	}
	if !overridePod.IsOwner(Id("pr-123"), "atlas-character") {
		t.Fatal("override does not own its own service")
	}
	if !baselinePod.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("baseline does not own a non-overridden service for pr-123")
	}
	if overridePod.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("override claims a service it does not override")
	}
}

func TestNonActivePhaseIsNotActiveAndNotOwned(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	rec := active("pr-123", "main", "atlas-pr-123", nil)
	rec.Phase = PhaseProvisioning
	r.Apply(rec)

	if r.IsActive(Id("pr-123")) {
		t.Fatal("PROVISIONING environment reported ACTIVE")
	}
	if r.IsOwner(Id("pr-123"), "atlas-monsters") {
		t.Fatal("PROVISIONING environment reported owned; overrides must receive no work (FR-5.2)")
	}
}

func TestTombstoneRemovesTheEnvironment(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("pr-123", "main", "atlas-pr-123", nil))
	r.ApplyTombstone(Id("pr-123"))

	if r.IsActive(Id("pr-123")) {
		t.Fatal("tombstoned environment still ACTIVE (FR-5.7)")
	}
}

func TestEnvironmentsOwnedByExcludesOverriddenServices(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.Apply(active("main", "main", "atlas-main", nil))
	r.Apply(active("pr-1", "main", "atlas-pr-1", map[string]string{"atlas-character": "atlas-pr-1"}))
	r.Apply(active("pr-2", "main", "atlas-pr-2", nil))

	got := r.EnvironmentsOwnedBy("atlas-character")
	if len(got) != 2 { // main + pr-2, NOT pr-1
		t.Fatalf("EnvironmentsOwnedBy(atlas-character) = %v, want main and pr-2", got)
	}
}

func TestStaleAfterFourMissedHeartbeats(t *testing.T) {
	now := time.Unix(0, 0)
	r := NewMapRegistry(Id("main"), func() time.Time { return now })
	r.Observe(now)

	if r.Stale() {
		t.Fatal("fresh registry reported stale")
	}
	now = now.Add(119 * time.Second)
	if r.Stale() {
		t.Fatal("stale at 119s; bound is 120s")
	}
	now = now.Add(2 * time.Second)
	if !r.Stale() {
		t.Fatal("not stale at 121s; bound is 120s")
	}
}

func TestCurrentRegistryIsNeverNil(t *testing.T) {
	if CurrentRegistry() == nil {
		t.Fatal("CurrentRegistry() = nil before SetRegistry; must be the legacy no-op")
	}
	if !CurrentRegistry().IsOwner(Id(""), "atlas-anything") {
		t.Fatal("the default registry is not the legacy no-op")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-env && go test ./... -run 'Registry|Namespace|Owner|Stale|Legacy|Tombstone|Phase'`
Expected: FAIL — `undefined: NewMapRegistry`.

- [ ] **Step 3: Implement `record.go`**

```go
package env

const (
	PhaseProvisioning = "PROVISIONING"
	PhaseActive       = "ACTIVE"
	PhaseDeactivating = "DEACTIVATING"
	PhaseDeleted      = "DELETED"
)

// Record is one environment as published on the environment-status topic.
// Overrides maps a service name to the NAMESPACE that serves it — not to a
// Deployment name. Deployment names are identical across namespaces in
// Atlas (atlas-character everywhere); the namespace is what varies and what
// the REST routing mechanism needs (design §4.4).
type Record struct {
	Name      Id                `json:"name"`
	Baseline  Id                `json:"baseline"`
	Namespace string            `json:"namespace"`
	Tenant    string            `json:"tenant"`
	Overrides map[string]string `json:"overrides"`
	Phase     string            `json:"phase"`
}

func (r Record) Active() bool { return r.Phase == PhaseActive }
```

- [ ] **Step 4: Implement `registry.go`**

```go
package env

import (
	"fmt"
	"sync"
	"time"
)

// StaleAfter is the FR-1.7 staleness bound: four missed 30s heartbeats.
const StaleAfter = 120 * time.Second

type Registry interface {
	EnvironmentNamespace(e Id) (string, error)
	ServiceNamespace(e Id, service string) (string, error)
	EnvironmentsOwnedBy(service string) []Id
	IsOwner(e Id, service string) bool
	IsActive(e Id) bool
	Stale() bool
}

type MapRegistry struct {
	mu       sync.RWMutex
	self     Id
	records  map[Id]Record
	lastSeen time.Time
	now      func() time.Time
}

func NewMapRegistry(self Id, clock func() time.Time) *MapRegistry {
	if clock == nil {
		clock = time.Now
	}
	return &MapRegistry{self: self, records: map[Id]Record{}, now: clock}
}

func (r *MapRegistry) Apply(rec Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec.Phase == PhaseDeleted {
		delete(r.records, rec.Name)
	} else {
		r.records[rec.Name] = rec
	}
	r.lastSeen = r.now()
}

func (r *MapRegistry) ApplyTombstone(name Id) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, name)
	r.lastSeen = r.now()
}

// Observe records that the projection is alive at t — a heartbeat record or
// any other message on the topic.
func (r *MapRegistry) Observe(t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSeen = t
}

func (r *MapRegistry) Stale() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lastSeen.IsZero() {
		return false // nothing ever seen: no records exist, legacy mode
	}
	return r.now().Sub(r.lastSeen) > StaleAfter
}

// EnvironmentNamespace is the environment's OWN namespace — where its own
// ingress lives. It NEVER falls back to the baseline: every environment
// deploys its own ingress, and the per-service override/baseline decision is
// made inside that ingress by its NS_* routing table (Task 43). Falling back
// here would send a baseline pod's downstream call for pr-123 into main.
func (r *MapRegistry) EnvironmentNamespace(e Id) (string, error) {
	if e == "" {
		return "", nil // legacy: caller keeps its own BASE_SERVICE_URL
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok {
		return "", fmt.Errorf("environment %q is not in the registry", e)
	}
	return rec.Namespace, nil
}

// ServiceNamespace is the namespace of the effective implementation of
// service for e: e's own namespace when e overrides the service, otherwise
// e's baseline's namespace (FR-1.2). Used to generate the ingress routing
// table and for diagnostics — NOT on the REST hot path, which wants
// EnvironmentNamespace.
func (r *MapRegistry) ServiceNamespace(e Id, service string) (string, error) {
	if e == "" {
		return "", nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok {
		return "", fmt.Errorf("environment %q is not in the registry", e)
	}
	if ns, ok := rec.Overrides[service]; ok {
		return ns, nil
	}
	base, ok := r.records[rec.Baseline]
	if !ok {
		return "", fmt.Errorf("baseline %q of environment %q is not in the registry", rec.Baseline, e)
	}
	return base.Namespace, nil
}

func (r *MapRegistry) IsActive(e Id) bool {
	if e == "" {
		return true // FR-1.8
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	return ok && rec.Active()
}

// IsOwner reports whether THIS process's deployment is the effective
// implementation of service for environment e. It is a pure function of the
// projected log plus self, so exactly one deployment satisfies it and every
// pod agrees (FR-4.6).
func (r *MapRegistry) IsOwner(e Id, service string) bool {
	if e == "" {
		return true // FR-1.8: the local deployment owns everything
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[e]
	if !ok || !rec.Active() {
		return false // D4: unknown or not-yet-active is never owned
	}
	if _, overridden := rec.Overrides[service]; overridden {
		return e == r.self
	}
	return rec.Baseline == r.self
}

func (r *MapRegistry) EnvironmentsOwnedBy(service string) []Id {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.records) == 0 {
		return []Id{""} // FR-1.8 / FR-6.6: exactly today's single iteration
	}
	out := make([]Id, 0, len(r.records))
	for name, rec := range r.records {
		if !rec.Active() {
			continue
		}
		if _, overridden := rec.Overrides[service]; overridden {
			if name == r.self {
				out = append(out, name)
			}
			continue
		}
		if rec.Baseline == r.self {
			out = append(out, name)
		}
	}
	return out
}

// legacyRegistry is the process-wide default before SetRegistry runs: every
// query returns the legacy answer, so a service that has not yet been
// migrated behaves exactly as it does today (FR-1.8).
type legacyRegistry struct{}

func (legacyRegistry) EnvironmentNamespace(Id) (string, error)      { return "", nil }
func (legacyRegistry) ServiceNamespace(Id, string) (string, error)  { return "", nil }
func (legacyRegistry) EnvironmentsOwnedBy(string) []Id       { return []Id{""} }
func (legacyRegistry) IsOwner(Id, string) bool               { return true }
func (legacyRegistry) IsActive(Id) bool                      { return true }
func (legacyRegistry) Stale() bool                           { return false }

var (
	currentMu sync.RWMutex
	current   Registry = legacyRegistry{}
)

// SetRegistry installs the process-wide registry. Called once, from
// libs/atlas-service's bootstrap wiring. Never call it from a domain
// package (env-domain-guard).
func SetRegistry(r Registry) {
	currentMu.Lock()
	defer currentMu.Unlock()
	current = r
}

// CurrentRegistry is never nil: before SetRegistry it is the legacy no-op.
func CurrentRegistry() Registry {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}
```

Note `EnvironmentsOwnedBy` returns `[]Id{""}` when the map is empty but the
per-record loop when it is not — that is deliberate: `FR-6.6` requires a
deployment owning only `main` to perform exactly today's work, and on `main`
with a single `main` record the loop yields exactly one environment.

- [ ] **Step 5: Run the tests**

Run: `cd libs/atlas-env && go build ./... && go test ./...`
Expected: PASS, all tests from Step 1.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-env
git commit -m "feat(atlas-env): the environment registry and its four queries"
```

---

### Task 19: Environment records in `atlas-configurations`

Decision P1: environment records are a fourth resource in
`atlas-configurations`, published through the transactional outbox onto a
compacted topic, exactly like `tenants` and `services` already are.

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/environments/entity.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/model.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/builder.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/provider.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/administrator.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/processor.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/rest.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/resource.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/heartbeat.go`
- Create: `services/atlas-configurations/atlas.com/configurations/environments/processor_test.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/outbox/envelopes.go` — add `NewEnvironmentEnvelope`
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go` — register the migration, the route initializer, and the heartbeat goroutine
- Modify: `deploy/k8s/base/env-configmap.yaml` — add `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS`
- Modify: `deploy/k8s/base/atlas-kafka-precreate.yaml` — add the new topic to the **compacted** list (it is a projection topic; see the comment at line 50)
- Read-only: `services/atlas-configurations/atlas.com/configurations/services/*.go` — the full resource shape to mirror
- Read-only: `libs/atlas-env/record.go` (Task 18) — the wire shape the projection decodes

Module root: `services/atlas-configurations/atlas.com/configurations`.

**Interfaces:**
- Consumes: `env.Record` (Task 18) as the JSON shape of the envelope's
  `config` field; `scope.AuthorizeWrite` (Task 13).
- Produces: `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS`, keyed
  `environment:<name>`, carrying an outbox envelope whose `config` decodes to
  `env.Record`. Task 20 consumes it. Tombstone = empty value, same convention
  as the tenant/service topics (`projection.IsTombstone`).
- Produces: REST `/api/configurations/environments` (JSON:API, resource type
  `environments`) for the sparse overlay's bootstrap Job to POST into.

- [ ] **Step 1: Write the failing publish test**

`environments/processor_test.go`:

```go
func TestCreatingAnEnvironmentEnqueuesAnOutboxEnvelope(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	_, err := p.Create(NewBuilder().
		SetName("pr-123").
		SetBaseline("main").
		SetNamespace("atlas-pr-123").
		SetTenant(tenantId.String()).
		SetOverride("atlas-login", "atlas-pr-123").
		SetOverride("atlas-channel", "atlas-pr-123").
		SetPhase(env.PhaseProvisioning).
		Build())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rows := readOutbox(t, db)
	if len(rows) != 1 {
		t.Fatalf("outbox has %d rows, want 1", len(rows))
	}
	if got := string(rows[0].Key); got != "environment:pr-123" {
		t.Fatalf("key = %q, want \"environment:pr-123\"", got)
	}

	var envelope struct {
		Config env.Record `json:"config"`
	}
	if err := json.Unmarshal(rows[0].Payload, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Config.Namespace != "atlas-pr-123" ||
		envelope.Config.Overrides["atlas-login"] != "atlas-pr-123" ||
		envelope.Config.Phase != env.PhaseProvisioning {
		t.Fatalf("envelope config = %+v", envelope.Config)
	}
}

func TestCreatingAnEnvironmentRejectsAMalformedName(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	if _, err := p.Create(NewBuilder().SetName("PR_123").
		SetBaseline("main").SetNamespace("x").Build()); err == nil {
		t.Fatal("malformed environment name accepted; ingest must validate (P2)")
	}
}
```

Read `services/atlas-configurations/atlas.com/configurations/outbox/` and the
`services` package's administrator to find how a row is enqueued and how the
test reads it back; mirror that helper rather than inventing `readOutbox`.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test ./environments/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the entity and model**

```go
// entity.go
type Entity struct {
	Id        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	Name      string          `gorm:"not null;uniqueIndex"`
	Baseline  string          `gorm:"not null"`
	Namespace string          `gorm:"not null"`
	Tenant    string          `gorm:"not null;default:''"`
	Overrides json.RawMessage `gorm:"type:json;not null"`
	Phase     string          `gorm:"not null"`
}

func (e Entity) TableName() string { return "environments" }

func Migration(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
```

Note this table carries **no** `environment` column: it *is* the environment
list, exactly as the tenant table is the tenant list (PRD §3.5's category
argument). Add it to `tools/scopeguard/allowlist.txt` with that reason.

- [ ] **Step 4: Implement the processor with ingest validation**

`Create`/`Update` call `env.Valid(env.Id(name))` and reject a malformed name
(P2), then write the row and enqueue the outbox envelope inside the same
transaction, mirroring `services/administrator.go`.

- [ ] **Step 5: Add the envelope constructor**

In `outbox/envelopes.go`, beside `NewTenantEnvelope`:

```go
// NewEnvironmentEnvelope serializes an environment record for the
// EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS topic. Shares the envelope
// shape with services and tenants; kept as its own constructor so it can
// diverge without ricochet.
func NewEnvironmentEnvelope(name string, config any, emittedAt time.Time) ([]byte, error) {
	return json.Marshal(envelope{
		SchemaVersion: CurrentSchemaVersion,
		Id:            name,
		Config:        config,
		EmittedAt:     emittedAt.UTC().Format(time.RFC3339),
	})
}
```

- [ ] **Step 6: Implement the heartbeat**

`environments/heartbeat.go` re-publishes the baseline environment's record
every 30 s so consumers can detect staleness (design §4.3, Task 18's
`StaleAfter`):

```go
// StartHeartbeat republishes the baseline environment record every 30s.
// Consumers use the arrival of ANY message on the topic as liveness, so the
// payload is deliberately the unchanged baseline record: compaction keeps
// exactly one copy per key regardless of how often it is written.
func StartHeartbeat(l logrus.FieldLogger, ctx context.Context, p Processor) {
	routine.Go(l, ctx, func(_ context.Context) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := p.Republish(env.Self()); err != nil {
					l.WithError(err).Warn("environment heartbeat failed")
				}
			}
		}
	})
}
```

- [ ] **Step 7: Wire `main.go`**

Add `environments.Migration` to `database.SetMigrations`, add
`.AddRouteInitializer(environments.InitResource(GetServer())(db))`, and call
`environments.StartHeartbeat(l, rt.Context(), environments.NewProcessor(l, rt.Context(), db))`
after the drainer starts.

- [ ] **Step 8: Add the topic to the ConfigMap and the precreate Job**

`deploy/k8s/base/env-configmap.yaml`:

```yaml
  EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS: EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS
```

In `deploy/k8s/base/atlas-kafka-precreate.yaml`, add it to the **compacted**
topic list beside the other two `CONFIGURATION_*_STATUS` topics — read the
comment at lines 50-58 first; a DELETE-retention projection topic empties
after ~7 days and every later boot replays nothing.

- [ ] **Step 9: Run the module tests**

Run: `cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-configurations deploy/k8s/base/env-configmap.yaml deploy/k8s/base/atlas-kafka-precreate.yaml tools/scopeguard/allowlist.txt
git commit -m "feat(atlas-configurations): environment records published on a compacted topic"
```

---

### Task 20: Project the environment topic into the registry from `libs/atlas-service`

Decision P4: the projection lives here, not in `libs/atlas-env`, to keep the
module graph acyclic. It reuses the exact machinery `projection.go` already
has for configuration: per-process consumer group replaying from
`FirstOffset`, an end-offset catch-up gate, composed into readiness.

**Files:**
- Create: `libs/atlas-service/envregistry.go` — the subscriber, the catch-up gate, and `WithEnvironmentRegistry`
- Create: `libs/atlas-service/envregistry_test.go`
- Modify: `libs/atlas-service/bootstrap.go` — `bootstrapConfig` gains the option; `Bootstrap` starts it
- Modify: `libs/atlas-service/go.mod` — add `libs/atlas-env`, `libs/atlas-kafka`
- Read-only: `libs/atlas-service/projection.go:60-120` — the per-process group id, the catch-up gate, and the Fatal semantics to mirror
- Read-only: `services/atlas-channel/atlas.com/channel/configuration/projection/subscriber.go` — the consumer registration and end-offset snapshot
- Read-only: `services/atlas-channel/atlas.com/channel/configuration/projection/caughtup.go` — the gate; copy its semantics, not its code
- Read-only: `libs/atlas-env/registry.go` (Task 18)

Module root: `libs/atlas-service`.

**Interfaces:**
- Consumes: `env.NewMapRegistry`, `env.SetRegistry`, `env.Record` (Task 18);
  the topic from Task 19.
- Produces:
  ```go
  // WithEnvironmentRegistry makes Bootstrap subscribe to the environment
  // status topic, project it into an env.MapRegistry, install it with
  // env.SetRegistry, and AND its catch-up into Runtime.Ready().
  func WithEnvironmentRegistry(serviceName string) Option
  ```
  Every service's `main.go` in Phase C passes exactly this one option.

- [ ] **Step 1: Write the failing projection tests**

```go
func TestEnvironmentProjectionAppliesRecords(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	payload := mustEnvelope(t, env.Record{
		Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"},
		Phase:     env.PhaseActive,
	})
	if _, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "t", Partition: 0, Offset: 0,
			Key: []byte("environment:pr-123"), Value: payload}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if !reg.IsActive(env.Id("pr-123")) {
		t.Fatal("record not applied")
	}
}

func TestEnvironmentProjectionAppliesTombstones(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	if _, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "t", Key: []byte("environment:pr-123"), Value: nil}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if reg.IsActive(env.Id("pr-123")) {
		t.Fatal("tombstone not applied")
	}
}

func TestEnvironmentProjectionIgnoresAnUnreadableSchema(t *testing.T) {
	// Forward-compatible, matching the configuration projection: a schema we
	// cannot read is acknowledged, not retried.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	cont, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "t", Key: []byte("environment:x"),
			Value: []byte(`{"schema_version":999}`)})
	if err != nil || !cont {
		t.Fatalf("got (cont=%v, err=%v), want (true, nil)", cont, err)
	}
}

func TestBootstrapWithoutTheOptionLeavesTheLegacyRegistry(t *testing.T) {
	// FR-1.8: a service that has not yet been migrated keeps today's
	// behaviour exactly.
	if !env.CurrentRegistry().IsOwner(env.Id(""), "atlas-anything") {
		t.Fatal("default registry is not the legacy no-op")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-service && go test ./... -run Environment -v`
Expected: FAIL — `undefined: envSubscriber`.

- [ ] **Step 3: Implement the subscriber**

Mirror `services/atlas-channel/.../projection/subscriber.go` structurally:
snapshot end offsets with `consumer.ReadReplayableEndOffsets`, register one
consumer with `consumer.SetStartOffset(kafka.FirstOffset)` and a per-process
group id, decode the envelope, and apply to the registry. On every message,
call `registry.Observe(time.Now())` **before** decoding so an unreadable
message still counts as liveness for the staleness bound.

The group id follows `projection.go:74-77` exactly:

```go
groupId := fmt.Sprintf("%s - environments - %s", serviceName, uuid.New().String())
```

- [ ] **Step 4: Implement `WithEnvironmentRegistry` and wire it into `Bootstrap`**

```go
func WithEnvironmentRegistry(serviceName string) Option {
	return func(c *bootstrapConfig) { c.envRegistry = &envRegistryConfig{serviceName: serviceName} }
}

func (r *Runtime) startEnvironmentRegistry(c *envRegistryConfig) {
	topic := os.Getenv("EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS")
	reg := env.NewMapRegistry(env.Self(), time.Now)
	env.SetRegistry(reg)
	if topic == "" {
		// No topic configured: the registry stays empty and every query
		// returns the legacy answer (FR-1.8). This is the state main runs
		// in until Phase F, and it must not be fatal.
		r.logger.Warn("environment registry: EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS unset; running in legacy single-environment mode")
		return
	}
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), topic: topic}
	groupId := fmt.Sprintf("%s - environments - %s", c.serviceName, uuid.New().String())
	if err := s.Start(r.tdm.Context(), r.logger, r.tdm.WaitGroup(), groupId); err != nil {
		r.logger.WithError(err).Fatal("Unable to start environment registry subscriber.")
	}
	r.envCaughtUp = s.caughtUp
	r.gates = append(r.gates, s.caughtUp.CaughtUpNow)
}
```

`Bootstrap` calls `startEnvironmentRegistry` when `cfg.envRegistry != nil`,
before `startProjection`. The catch-up gate is ANDed into `Ready()` directly
here rather than requiring each service to pass `WithReadinessGate` — this is
the one gate every service needs identically, so the option owns it.

- [ ] **Step 5: Run the tests**

Run: `cd libs/atlas-service && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-service
git commit -m "feat(atlas-service): project the environment topic into the registry"
```

---

### Task 21: Project tenant → environment, and the FR-7.7 mismatch error

FR-7.3: environment must be derivable from tenant identity so autonomous work
starting from persisted, tenant-owned state resolves its environment. Per
design §8.3 this is structural — the tenant row carries its environment — and
it is the recovery path for every delayed or persisted work item.

**Files:**
- Create: `libs/atlas-env/tenants.go` — the tenant→environment map and its accessors
- Create: `libs/atlas-env/tenants_test.go`
- Modify: `libs/atlas-service/envregistry.go` — subscribe to `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` as a second source, feeding the tenant map
- Modify: `libs/atlas-service/envregistry_test.go`
- Read-only: `services/atlas-channel/atlas.com/channel/configuration/projection/subscriber.go:117-152` — the tenant envelope decode and tombstone key convention (`tenant:<uuid>`)
- Read-only: `services/atlas-tenants/atlas.com/tenants/tenant/kafka.go` (Task 12) — the `environment` attribute added to the emitted event

Module roots: `libs/atlas-env`, `libs/atlas-service`.

**Interfaces:**
- Produces:
  ```go
  // TenantEnvironments maps tenant id -> environment. Populated by the same
  // projection that populates the registry.
  func (r *MapRegistry) ApplyTenant(tenantId string, e Id)
  func (r *MapRegistry) RemoveTenant(tenantId string)
  func (r *MapRegistry) EnvironmentOfTenant(tenantId string) (Id, bool)

  // ErrEnvironmentMismatch is FR-7.7: a hard error, never a reconciliation.
  var ErrEnvironmentMismatch = errors.New("environment header disagrees with the tenant's environment")

  // Reconcile returns the operation's environment or ErrEnvironmentMismatch.
  func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error)
  ```

- [ ] **Step 1: Write the failing reconciliation tests**

```go
func TestReconcileAgreesWhenHeaderMatchesTenant(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	got, err := Reconcile(r, Id("pr-123"), "t-1")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileRejectsADisagreement(t *testing.T) {
	// FR-7.7: a mismatch is a hard error, not a reconciliation. Silently
	// preferring either side is how an operation changes environment
	// mid-flight.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	if _, err := Reconcile(r, Id("main"), "t-1"); !errors.Is(err, ErrEnvironmentMismatch) {
		t.Fatalf("got %v, want ErrEnvironmentMismatch", err)
	}
}

func TestReconcileDerivesFromTenantWhenNoHeaderIsPresent(t *testing.T) {
	// The autonomous / persisted-work recovery path (design §8.3): a saga
	// sweeper reads a row, gets a tenant, and needs an environment.
	r := NewMapRegistry(Id("main"), time.Now)
	r.ApplyTenant("t-1", Id("pr-123"))

	got, err := Reconcile(r, Id(""), "t-1")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileWithAnUnknownTenantTrustsTheHeader(t *testing.T) {
	// A tenant the projection has not seen yet is possible during
	// activation (design §7.3): the tenant and environment records travel
	// on different topics. Trusting the header here does NOT weaken D4 —
	// the ownership gate still rejects an unknown ENVIRONMENT.
	r := NewMapRegistry(Id("main"), time.Now)

	got, err := Reconcile(r, Id("pr-123"), "t-unknown")
	if err != nil || got != Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", nil)", got, err)
	}
}

func TestReconcileWithNeitherIsTheLegacyValue(t *testing.T) {
	r := NewMapRegistry(Id("main"), time.Now)
	got, err := Reconcile(r, Id(""), "")
	if err != nil || got != Id("") {
		t.Fatalf("got (%q, %v), want (\"\", nil)", got, err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-env && go test ./... -run Reconcile -v`
Expected: FAIL — `undefined: Reconcile`.

- [ ] **Step 3: Implement `tenants.go`**

```go
var ErrEnvironmentMismatch = errors.New("environment header disagrees with the tenant's environment")

// TenantResolver is the half of Registry that answers "which environment
// does this tenant belong to". Separate interface so Reconcile is testable
// against a fake without the four routing queries.
type TenantResolver interface {
	EnvironmentOfTenant(tenantId string) (Id, bool)
}

// Reconcile resolves the operation's environment from the header and the
// tenant, and returns ErrEnvironmentMismatch when they disagree (FR-7.7).
// An unknown tenant trusts the header: during activation the tenant and
// environment records arrive on different topics and therefore different
// partitions, so a tenant may be visible before or after its environment
// (design §7.3). This does not weaken D4 — an unknown ENVIRONMENT is still
// rejected by the ownership gate.
func Reconcile(r Registry, headerEnv Id, tenantId string) (Id, error) {
	tr, ok := r.(TenantResolver)
	if !ok || tenantId == "" {
		return headerEnv, nil
	}
	tenantEnv, known := tr.EnvironmentOfTenant(tenantId)
	if !known {
		return headerEnv, nil
	}
	if headerEnv == "" {
		return tenantEnv, nil
	}
	if headerEnv != tenantEnv {
		return "", fmt.Errorf("%w: header=%q tenant=%q(%s)",
			ErrEnvironmentMismatch, headerEnv, tenantEnv, tenantId)
	}
	return headerEnv, nil
}
```

Add `tenants map[string]Id` to `MapRegistry` (guarded by the same mutex) plus
`ApplyTenant`, `RemoveTenant`, `EnvironmentOfTenant`.

- [ ] **Step 4: Subscribe to the tenant-status topic in the projection**

In `libs/atlas-service/envregistry.go`, register a second consumer on
`EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` with the same per-process group id
suffix and `FirstOffset`, decoding the tenant envelope's `environment`
attribute into `registry.ApplyTenant`. Handle the `tenant:<uuid>` tombstone
key exactly as `subscriber.go:117-152` does. Both topics feed the same
catch-up gate.

- [ ] **Step 5: Run the tests**

Run:
```sh
cd libs/atlas-env && go build ./... && go test ./...
cd libs/atlas-service && go build ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-env libs/atlas-service
git commit -m "feat(atlas-env): derive environment from tenant; reject header/tenant mismatch"
```

---

### Task 22: `libs/atlas-rest` inbound — parse the `ENVIRONMENT` header

**Files:**
- Modify: `libs/atlas-rest/server/handler.go` — add `ParseEnvironment` beside `ParseTenant` (line 34)
- Modify: `libs/atlas-rest/server/register.go` — compose it into all four `Register*Handler` functions
- Modify: `libs/atlas-rest/server/handler_test.go` — the new cases
- Modify: `libs/atlas-rest/go.mod` — add `libs/atlas-env`
- Modify: `deploy/k8s/base/atlas-ingress.yaml:38-41` — forward the header
- Read-only: `libs/atlas-rest/server/handler.go:34-100` — `ParseTenant`, the shape to mirror
- Read-only: `libs/atlas-env/env.go`, `libs/atlas-env/tenants.go`

Module root: `libs/atlas-rest`.

**Interfaces:**
- Consumes: `env.Key`, `env.WithContext`, `env.Reconcile`, `env.CurrentRegistry`.
- Produces:
  ```go
  type EnvironmentHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc
  func ParseEnvironment(l logrus.FieldLogger, ctx context.Context, next EnvironmentHandler) http.HandlerFunc
  ```
  Ordering in `register.go`: `RetrieveSpan` → `ParseEnvironment` → `ParseTenant`
  → handler, so the environment is on the context before the tenant is parsed
  and the tenant-derived reconciliation (FR-7.7) happens inside `ParseTenant`'s
  successor.

- [ ] **Step 1: Write the failing handler tests**

Add to `libs/atlas-rest/server/handler_test.go`:

```go
func TestParseEnvironmentPutsTheHeaderOnTheContext(t *testing.T) {
	var got env.Id
	h := ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				got = env.MustFromContext(ctx)
				w.WriteHeader(http.StatusOK)
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-123")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\"", got)
	}
}

func TestParseEnvironmentWithNoHeaderIsTheLegacyPath(t *testing.T) {
	// FR-1.8 / NFR-7: an unheadered request is exactly today's request.
	called := false
	h := ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, ctx context.Context) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				called = true
				if got := env.MustFromContext(ctx); got != env.Id("") {
					t.Errorf("environment = %q, want the empty id", got)
				}
				w.WriteHeader(http.StatusOK)
			}
		})

	h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler not reached; an unheadered request must pass through")
	}
}

func TestParseEnvironmentRejectsAnUnknownEnvironment(t *testing.T) {
	// FR-3.6: a request naming an unknown or inactive environment is
	// rejected. Never served by the baseline (G4).
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	h := ParseEnvironment(testLogger(t), context.Background(),
		func(_ logrus.FieldLogger, _ context.Context) http.HandlerFunc {
			return func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler reached for an unknown environment")
			}
		})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(env.Key, "pr-999")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
```

`env.SetRegistry(nil)` must restore the legacy no-op rather than storing nil
— fix `SetRegistry` in `libs/atlas-env/registry.go` to substitute
`legacyRegistry{}` for a nil argument, and add a test for that there.

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-rest && go test ./server/ -run ParseEnvironment -v`
Expected: FAIL — `undefined: ParseEnvironment`.

- [ ] **Step 3: Implement `ParseEnvironment`**

```go
type EnvironmentHandler func(logrus.FieldLogger, context.Context) http.HandlerFunc

// ParseEnvironment reads the ENVIRONMENT header onto the context. An absent
// header is the legacy value and passes through unchanged (FR-1.8). A
// present header naming an environment the registry does not know, or knows
// as inactive, is rejected with 400 — never served by the baseline (FR-3.6,
// D4).
func ParseEnvironment(l logrus.FieldLogger, ctx context.Context, next EnvironmentHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := env.Id(r.Header.Get(env.Key))
		if id != "" && !env.CurrentRegistry().IsActive(id) {
			l.WithField(env.Key, string(id)).Error("Request names an unknown or inactive environment.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		el := l
		if id != "" {
			el = l.WithField("environment", string(id))
		}
		next(el, env.WithContext(ctx, id))(w, r)
	}
}
```

- [ ] **Step 4: Compose it into all four registrars**

In `register.go`, wrap each existing body. `RegisterHandler` becomes:

```go
return RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
	fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
	return ParseEnvironment(fl, sctx, func(el logrus.FieldLogger, ectx context.Context) http.HandlerFunc {
		return ParseTenant(el, ectx, func(tl logrus.FieldLogger, tctx context.Context) http.HandlerFunc {
			return handler(&HandlerDependency{l: tl, ctx: tctx}, &HandlerContext{si: si})
		})
	})
})
```

`RegisterSimpleHandler` and `RegisterSimpleInputHandler` get
`ParseEnvironment` but **not** `ParseTenant` — those are the tenant-free
control-plane routes (e.g. `atlas-drops` fetching its own service config),
and for them the header is the only source of environment. That is why the
header exists at all rather than deriving environment from tenant everywhere
(design §5.2).

- [ ] **Step 5: Add the FR-7.7 reconciliation inside `ParseTenant`'s successor**

At the end of `ParseTenant`, after `tctx := tenant.WithContext(ctx, t)`:

```go
		resolved, err := env.Reconcile(env.CurrentRegistry(), env.MustFromContext(tctx), t.Id().String())
		if err != nil {
			l.WithError(err).Error("Environment header disagrees with the tenant's environment.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tctx = env.WithContext(tctx, resolved)
		next(tl, tctx)(w, r)
```

Add a test asserting a mismatched pair yields 400 and the handler is not
reached.

- [ ] **Step 6: Forward the header at the ingress**

`deploy/k8s/base/atlas-ingress.yaml`, after line 41:

```
        proxy_set_header ENVIRONMENT $http_environment;
```

`underscores_in_headers on` is already set at line 27; no other nginx change
is needed for the inbound direction.

- [ ] **Step 7: Run the tests**

Run: `cd libs/atlas-rest && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-rest deploy/k8s/base/atlas-ingress.yaml
git commit -m "feat(atlas-rest): parse and forward the ENVIRONMENT header"
```

---

### Task 23: `libs/atlas-rest` outbound — target the operation's own ingress

This is the mechanism that closes the leak the PRD identified but did not
localise: without it, a baseline pod serving `pr-123` sends its downstream
call to `atlas-main`'s ingress and the operation silently changes environment
mid-flight (design §5.1).

**Files:**
- Modify: `libs/atlas-rest/requests/url.go` — add the environment-aware resolver
- Create: `libs/atlas-rest/requests/url_test.go`
- Modify: `libs/atlas-rest/requests/header.go` — add `EnvHeaderDecorator` beside `TenantHeaderDecorator` (line 29)
- Modify: `libs/atlas-rest/requests/decorated.go` and/or `client.go` — wire `EnvHeaderDecorator` into the canonical decorator set so no service sets the header by hand (FR-3.1)
- Modify: `libs/atlas-rest/requests/client_test.go` — assert the header on an outbound request
- Read-only: `deploy/k8s/overlays/main/kustomization.yaml:39` and `deploy/k8s/overlays/pr/kustomization.yaml:160` — the `BASE_SERVICE_URL` shape being rewritten

Module root: `libs/atlas-rest`.

**Interfaces:**
- Consumes: `env.FromContext`, `env.CurrentRegistry().Namespace` (Tasks 17–18).
- Produces:
  ```go
  // RootUrlFor resolves the base URL for a domain in the environment carried
  // on ctx. Falls back to RootUrl(domain) for the legacy empty environment.
  func RootUrlFor(ctx context.Context, domain string) (string, error)
  func EnvHeaderDecorator(ctx context.Context) HeaderDecorator
  ```
  `RootUrl(domain string) string` keeps its signature and behaviour so no
  caller breaks; every caller migrates to `RootUrlFor` in Phase C.

- [ ] **Step 1: Write the failing URL tests**

```go
func TestRootUrlForWithNoEnvironmentIsUnchanged(t *testing.T) {
	// NFR-7: byte-identical to today for a legacy operation.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")

	got, err := RootUrlFor(context.Background(), "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	if want := RootUrl("characters"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootUrlForTargetsTheEnvironmentsIngress(t *testing.T) {
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive})
	reg.Apply(env.Record{Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	want := "http://atlas-ingress.atlas-pr-123.svc.cluster.local:80/api/"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootUrlForANonOverriddenServiceStillTargetsTheEnvironmentsIngress(t *testing.T) {
	// The regression this whole mechanism turns on. pr-123 overrides only
	// atlas-character. A call to atlas-monsters must STILL leave through
	// pr-123's ingress — that ingress's NS_ATLAS_MONSTERS then points at
	// atlas-main, so the request reaches the baseline pod WITH the
	// ENVIRONMENT header intact. Resolving the upstream namespace here
	// instead would strip the environment from the operation.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive})
	reg.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"},
		Phase:     env.PhaseActive})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "monsters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	want := "http://atlas-ingress.atlas-pr-123.svc.cluster.local:80/api/"
	if got != want {
		t.Fatalf("got %q, want %q — a non-overridden service must not be "+
			"resolved to the baseline's ingress", got, want)
	}
}

func TestRootUrlForAnUnknownEnvironmentErrorsAndNeverFallsBack(t *testing.T) {
	// G4 / FR-3.5: an operation must never silently transition to main.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-999"))
	got, err := RootUrlFor(ctx, "characters")
	if err == nil {
		t.Fatalf("got %q with no error; want an error, never a baseline URL", got)
	}
}

func TestRootUrlForHonoursAPerDomainOverride(t *testing.T) {
	// A *_SERVICE_URL override bypasses the ingress entirely (local debug).
	// It must keep winning, and must NOT be namespace-rewritten.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	t.Setenv("CHARACTERS_SERVICE_URL", "http://localhost:9999/api/")

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	if got != "http://localhost:9999/api/" {
		t.Fatalf("got %q, want the per-domain override verbatim", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-rest && go test ./requests/ -run RootUrlFor -v`
Expected: FAIL — `undefined: RootUrlFor`.

- [ ] **Step 3: Implement `RootUrlFor`**

```go
// selfNamespace is the namespace this process runs in, used to rewrite
// BASE_SERVICE_URL for another environment. POD_NAMESPACE is already
// injected into the ingress Deployment; services get it the same way.
const podNamespaceVar = "POD_NAMESPACE"

// RootUrlFor resolves the base URL for domain in the environment carried on
// ctx. Every inter-service call in Atlas already goes through an ingress
// whose address is BASE_SERVICE_URL, and every environment deploys its own
// ingress — so environment-aware routing is a namespace substitution in
// that one string, resolved from the in-memory registry with no I/O
// (design §5.1). It NEVER falls back to the baseline: an unresolvable
// environment is an error before the call is made (FR-3.5, G4).
func RootUrlFor(ctx context.Context, domain string) (string, error) {
	if val, ok := os.LookupEnv(strings.ToUpper(domain) + ServiceSuffix); ok {
		return val, nil // per-domain override wins and is never rewritten
	}
	base := os.Getenv(BaseService)
	e, _ := env.FromContext(ctx)()
	if e == "" {
		return base, nil // legacy: byte-identical to RootUrl
	}
	// EnvironmentNamespace, NOT ServiceNamespace(e, "atlas-ingress"): every
	// environment deploys its own ingress, and the record's `overrides` map
	// does not list it. ServiceNamespace would therefore fall back to the
	// baseline and send this call into main — the exact leak M3 exists to
	// close. See Task 18's TestEnvironmentNamespaceNeverFallsBackToTheBaseline.
	ns, err := env.CurrentRegistry().EnvironmentNamespace(e)
	if err != nil {
		return "", fmt.Errorf("resolving ingress for environment %q: %w", e, err)
	}
	if ns == "" {
		return base, nil
	}
	rewritten, err := replaceNamespace(base, ns)
	if err != nil {
		return "", fmt.Errorf("rewriting %q for namespace %q: %w", base, ns, err)
	}
	return rewritten, nil
}

// replaceNamespace rewrites the namespace label of a cluster-local URL:
//
//	http://atlas-ingress.atlas-main.svc.cluster.local:80/api/
//	                     ^^^^^^^^^^
//
// A URL that does not match that shape (a local-debug host, an external
// address) is an error rather than a silent pass-through — passing it
// through would send the operation to the wrong environment.
func replaceNamespace(raw string, ns string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host, port, splitErr := net.SplitHostPort(u.Host)
	if splitErr != nil {
		host = u.Host
	}
	parts := strings.Split(host, ".")
	if len(parts) < 5 || parts[2] != "svc" || parts[3] != "cluster" {
		return "", fmt.Errorf("host %q is not a cluster-local FQDN", host)
	}
	parts[1] = ns
	host = strings.Join(parts, ".")
	if port != "" {
		u.Host = net.JoinHostPort(host, port)
	} else {
		u.Host = host
	}
	return u.String(), nil
}
```

- [ ] **Step 4: Implement `EnvHeaderDecorator` and wire it centrally**

```go
// EnvHeaderDecorator sets the ENVIRONMENT header from the operation's
// context. Set centrally so no service sets it by hand (FR-3.1); a service
// handling a request for pr-123 emits pr-123 on every downstream call
// regardless of which deployment it is (FR-3.2).
func EnvHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func(h http.Header) {
		e, _ := env.FromContext(ctx)()
		if e == "" {
			return // legacy: no header, byte-identical to today
		}
		h.Set(env.Key, string(e))
	}
}
```

Find where `TenantHeaderDecorator` is composed into the default decorator set
(`grep -rn "TenantHeaderDecorator" libs/atlas-rest services | grep -v _test`)
and add `EnvHeaderDecorator` at every one of those sites in `libs/`. Sites in
`services/` are Phase C work — record them in the task report, do not edit
them here.

- [ ] **Step 5: Run the tests**

Run: `cd libs/atlas-rest && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-rest
git commit -m "feat(atlas-rest): route outbound calls to the operation's own ingress"
```

---

### Task 24: `libs/atlas-kafka` producer — the `ENVIRONMENT` message header

**Files:**
- Modify: `libs/atlas-kafka/producer/header.go` — add `EnvHeaderDecorator` beside `TenantHeaderDecorator`
- Modify: `libs/atlas-kafka/producer/provider.go:14-22` — `ProviderImpl` composes it
- Create: `libs/atlas-kafka/producer/header_test.go` (or extend the existing producer tests)
- Modify: `libs/atlas-kafka/go.mod` — add `libs/atlas-env`
- Modify: `services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go:97`
- Modify: `services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go:28`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go:103`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go:175`
- Create: `tools/producerseamguard/` + `tools/producer-seam-guard.sh` — forbid new direct `producer.Produce` outside `libs/`
- Modify: `tools/verify.sh` — add the guard step
- Read-only: `libs/atlas-kafka/producer/provider.go` — `ProviderImpl` is the canonical producer at 211 call sites

Module roots: `libs/atlas-kafka`, `services/atlas-quest/atlas.com/quest`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `tools/producerseamguard`.

**Interfaces:**
- Produces: `func EnvHeaderDecorator(ctx context.Context) HeaderDecorator`,
  emitting `map[env.Key] = string(id)` and nothing when the id is empty.
- Produces: `tools/producer-seam-guard.sh`, exit 0.

- [ ] **Step 1: Write the failing decorator test**

```go
func TestEnvHeaderDecoratorEmitsTheEnvironment(t *testing.T) {
	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := EnvHeaderDecorator(ctx)()
	if err != nil {
		t.Fatalf("decorator: %v", err)
	}
	if got[env.Key] != "pr-123" {
		t.Fatalf("headers = %v, want %s=pr-123", got, env.Key)
	}
}

func TestEnvHeaderDecoratorEmitsNothingForTheLegacyEnvironment(t *testing.T) {
	// NFR-7: a main-only message is byte-identical to today's.
	got, err := EnvHeaderDecorator(context.Background())()
	if err != nil {
		t.Fatalf("decorator: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("headers = %v, want none", got)
	}
}

func TestProviderImplComposesTheEnvironmentDecorator(t *testing.T) {
	// FR-4.1: written centrally. A service must never add this by hand.
	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	msgs := captureProduced(t, func() {
		ProviderImpl(testLogger(t))(ctx)("SOME_TOPIC")(fixedMessages(t))
	})
	if !hasHeader(msgs[0], env.Key, "pr-123") {
		t.Fatalf("produced message headers = %v, want %s=pr-123", msgs[0].Headers, env.Key)
	}
}
```

Read `libs/atlas-kafka/producer/` for the existing test harness before
writing `captureProduced` / `fixedMessages`; reuse whatever the producer
tests already use for a fake writer.

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-kafka && go test ./producer/ -run EnvHeader -v`
Expected: FAIL — `undefined: EnvHeaderDecorator`.

- [ ] **Step 3: Implement the decorator and compose it**

```go
// EnvHeaderDecorator writes the operation's environment as a message
// header, mirroring TenantHeaderDecorator. Message BODIES are untouched —
// no domain schema changes anywhere, which is what keeps the 64-service
// migration mechanical (design §6.1).
func EnvHeaderDecorator(ctx context.Context) HeaderDecorator {
	return func() (map[string]string, error) {
		headers := make(map[string]string)
		e, _ := env.FromContext(ctx)()
		if e == "" {
			return headers, nil
		}
		headers[env.Key] = string(e)
		return headers, nil
	}
}
```

and in `provider.go`:

```go
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider {
	return func(ctx context.Context) Provider {
		sd := SpanHeaderDecorator(ctx)
		td := TenantHeaderDecorator(ctx)
		ed := EnvHeaderDecorator(ctx)
		return func(token string) MessageProducer {
			return Produce(l)(ManagerWriterProvider(l)(token))(sd, td, ed)
		}
	}
}
```

- [ ] **Step 4: Add the decorator at the four direct call sites**

Each of the four already passes `(sd, td)` explicitly; add `ed`:

```go
ed := producer.EnvHeaderDecorator(ctx)
// ...
producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td, ed)
```

- [ ] **Step 5: Write the guard fixture and implement the guard**

`tools/producerseamguard/testdata/src/directproduce/main.go`:

```go
package directproduce

import "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

func bad() {
	_ = producer.Produce(nil) // want `direct producer.Produce outside libs/`
}
```

The analyzer reports any call to `producer.Produce` whose file lives under
`services/` and is not in the four-entry allowlist. Wire into `verify.sh`
gated on `touched '^services/.*producer.*\.go$|^libs/atlas-kafka/'`.

- [ ] **Step 6: Run everything**

Run:
```sh
cd libs/atlas-kafka && go build ./... && go test ./...
cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
./tools/producer-seam-guard.sh
```
Expected: all PASS / exit 0.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka services/atlas-quest services/atlas-saga-orchestrator tools/producerseamguard tools/producer-seam-guard.sh tools/verify.sh
git commit -m "feat(atlas-kafka): write the ENVIRONMENT message header centrally"
```

---

### Task 25: `libs/atlas-kafka` consumer — parse the header and reconcile against the tenant

**Files:**
- Modify: `libs/atlas-kafka/consumer/header.go` — add `EnvHeaderParser` beside `TenantHeaderParser`
- Create: `libs/atlas-kafka/consumer/header_env_test.go`
- Modify: every `consumer.SetHeaderParsers(...)` call site in `libs/` — add the new parser
- Read-only: `libs/atlas-kafka/consumer/header.go:28-66` — `TenantHeaderParser`, the shape to mirror
- Read-only: `libs/atlas-env/tenants.go` (Task 21) — `Reconcile`

Module root: `libs/atlas-kafka`.

**Interfaces:**
- Produces: `func EnvHeaderParser(ctx context.Context, headers []kafka.Header) context.Context`.
  It must run **after** `TenantHeaderParser` in the parser slice so the
  tenant is already on the context when it reconciles (FR-7.7).

- [ ] **Step 1: Write the failing parser tests**

```go
func TestEnvHeaderParserPutsTheHeaderOnTheContext(t *testing.T) {
	ctx := EnvHeaderParser(context.Background(), []kafka.Header{
		{Key: env.Key, Value: []byte("pr-123")},
	})
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\"", got)
	}
}

func TestEnvHeaderParserWithNoHeaderIsTheLegacyValue(t *testing.T) {
	ctx := EnvHeaderParser(context.Background(), nil)
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("got %q, want the empty id", got)
	}
}

func TestEnvHeaderParserDerivesFromTenantWhenNoHeaderIsPresent(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant(tenantId.String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	base := tenant.WithContext(context.Background(), testTenant(t))
	ctx := EnvHeaderParser(base, nil)
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\" derived from the tenant", got)
	}
}

func TestEnvHeaderParserMarksAMismatch(t *testing.T) {
	// FR-7.7. The parser cannot return an error (its signature is
	// context-in/context-out), so it records the mismatch on the context
	// and the gate drops the message.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant(tenantId.String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	base := tenant.WithContext(context.Background(), testTenant(t))
	ctx := EnvHeaderParser(base, []kafka.Header{{Key: env.Key, Value: []byte("main")}})
	if !env.Mismatched(ctx) {
		t.Fatal("mismatch not recorded; the gate would process the message")
	}
}
```

`env.Mismatched(ctx) bool` and its setter `env.WithMismatch(ctx)` are new;
add them to `libs/atlas-env/tenants.go` with their own unit test in that
package.

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run EnvHeaderParser -v`
Expected: FAIL — `undefined: EnvHeaderParser`.

- [ ] **Step 3: Implement the parser**

```go
// EnvHeaderParser reads the ENVIRONMENT message header onto the context and
// reconciles it against the tenant already on the context (FR-7.7). It must
// be registered AFTER TenantHeaderParser.
//
// A mismatch cannot be returned — the HeaderParser signature is
// context-in/context-out — so it is recorded on the context and the
// ownership gate drops the message with the alertable counter.
func EnvHeaderParser(ctx context.Context, headers []kafka.Header) context.Context {
	var id env.Id
	for _, h := range headers {
		if h.Key == env.Key {
			id = env.Id(h.Value)
			break
		}
	}

	tenantId := ""
	if t, err := tenant.FromContext(ctx)(); err == nil {
		tenantId = t.Id().String()
	}

	resolved, err := env.Reconcile(env.CurrentRegistry(), id, tenantId)
	if err != nil {
		return env.WithMismatch(env.WithContext(ctx, id))
	}
	return env.WithContext(ctx, resolved)
}
```

- [ ] **Step 4: Register the parser wherever `TenantHeaderParser` is registered in `libs/`**

Run `grep -rn "TenantHeaderParser" libs/ services/ | grep -v _test` and add
`consumer.EnvHeaderParser` **after** it at each `libs/` site. Sites in
`services/` are Phase C work — record them in the report; there will be one
per consumer registration and the Phase C batches convert them.

- [ ] **Step 5: Run the tests**

Run: `cd libs/atlas-kafka && go build ./... && go test ./...` and
`cd libs/atlas-env && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka libs/atlas-env
git commit -m "feat(atlas-kafka): parse the ENVIRONMENT message header and reconcile it against the tenant"
```

---

### Task 26: The Kafka ownership gate

The correctness of the whole project lives here: exactly one deployment
processes any environment-scoped message (FR-4.6), and an unresolvable
environment is dropped and alerted rather than executed by the baseline
(FR-4.7, D4).

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go:610-620` — the gate goes in `processMessage`, immediately after header parsing and before the tracing span
- Create: `libs/atlas-kafka/consumer/gate.go` — the decision function and its counters
- Create: `libs/atlas-kafka/consumer/gate_test.go`
- Modify: `libs/atlas-kafka/consumer/manager.go` — `Consumer` gains a `service string` field, set from `SERVICE_NAME` at construction
- Read-only: `libs/atlas-kafka/consumer/manager.go:611-650` — `processMessage`; note the return-value contract (`true` = success = offset advances)
- Read-only: `libs/atlas-kafka/consumer/cursor.go` — the prefix-commit cursor a `false` return would block
- Read-only: `libs/atlas-env/registry.go` (Task 18)

Module root: `libs/atlas-kafka`.

**Interfaces:**
- Produces:
  ```go
  type gateVerdict int

  const (
      gateProcess gateVerdict = iota
      gateSkipNotOwner
      gateDropUnresolvable
  )

  // decide is a pure function of the registry state and the message's
  // environment — no I/O, no clock beyond the registry's own staleness.
  func decide(r env.Registry, self env.Id, service string, msgEnv env.Id, mismatched bool) gateVerdict
  ```
- Produces three Prometheus counters labelled `{service, environment}`:
  `atlas_kafka_gate_processed_total`, `atlas_kafka_gate_skipped_not_owner_total`,
  `atlas_kafka_gate_dropped_unresolvable_total`.

- [ ] **Step 1: Write the failing gate tests**

```go
func TestGateProcessesTheLegacyEnvironment(t *testing.T) {
	// FR-1.8: with no records, everything is processed exactly as today.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id(""), false); got != gateProcess {
		t.Fatalf("verdict = %v, want gateProcess", got)
	}
}

func TestGateProcessesWhenThisDeploymentIsTheOwner(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateProcess {
		t.Fatalf("verdict = %v, want gateProcess (baseline owns a non-overridden service)", got)
	}
}

func TestGateSkipsWhenAnotherDeploymentIsTheOwner(t *testing.T) {
	// FR-4.4: acknowledged without domain processing.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"}, Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-character", env.Id("pr-123"), false); got != gateSkipNotOwner {
		t.Fatalf("verdict = %v, want gateSkipNotOwner", got)
	}
}

func TestGateDropsAnUnknownEnvironment(t *testing.T) {
	// FR-4.7 / D4: not processed by ANY deployment, acked, alertable.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-999"), false); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestGateDropsAfterDeletion(t *testing.T) {
	// FR-5.7: after DELETED, surviving delayed work never executes against
	// the baseline. This is satisfied by the gate's drop path, not by
	// draining (design §7.4).
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	r.ApplyTombstone(env.Id("pr-123"))

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestGateDropsAMismatch(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("main"), true); got != gateDropUnresolvable {
		t.Fatalf("verdict = %v, want gateDropUnresolvable for a header/tenant mismatch", got)
	}
}

func TestGateWhenStaleProcessesOnlyItsOwnEnvironment(t *testing.T) {
	// design §4.3: a registry outage degrades a baseline pod to exactly its
	// pre-project behaviour — it serves main and nothing else — rather than
	// taking it down.
	now := time.Unix(0, 0)
	r := env.NewMapRegistry(env.Id("main"), func() time.Time { return now })
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	now = now.Add(5 * time.Minute) // stale

	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("main"), false); got != gateProcess {
		t.Fatalf("own environment while stale: verdict = %v, want gateProcess", got)
	}
	if got := decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-123"), false); got != gateDropUnresolvable {
		t.Fatalf("foreign environment while stale: verdict = %v, want gateDropUnresolvable", got)
	}
}

func TestExactlyOneDeploymentProcesses(t *testing.T) {
	// FR-4.6, stated as the property rather than as two separate asserts.
	overrides := map[string]string{"atlas-character": "atlas-pr-123"}
	records := []env.Record{
		{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive},
		{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Overrides: overrides, Phase: env.PhaseActive},
	}
	for _, svc := range []string{"atlas-character", "atlas-monsters"} {
		for _, msgEnv := range []env.Id{"main", "pr-123"} {
			processors := 0
			for _, self := range []env.Id{"main", "pr-123"} {
				r := env.NewMapRegistry(self, time.Now)
				for _, rec := range records {
					r.Apply(rec)
				}
				if decide(r, self, svc, msgEnv, false) == gateProcess {
					processors++
				}
			}
			if processors != 1 {
				t.Errorf("service=%s env=%s: %d deployments would process, want exactly 1", svc, msgEnv, processors)
			}
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run Gate -v`
Expected: FAIL — `undefined: decide`.

- [ ] **Step 3: Implement `gate.go`**

```go
func decide(r env.Registry, self env.Id, service string, msgEnv env.Id, mismatched bool) gateVerdict {
	if mismatched {
		return gateDropUnresolvable // FR-7.7
	}
	if msgEnv == "" {
		return gateProcess // FR-1.8
	}
	if r.Stale() && msgEnv != self {
		// A pod's own environment comes from an env var and cannot go
		// stale; every other environment fails closed (design §4.3).
		return gateDropUnresolvable
	}
	if !r.IsActive(msgEnv) {
		return gateDropUnresolvable // FR-4.7 / D4
	}
	if !r.IsOwner(msgEnv, service) {
		return gateSkipNotOwner // FR-4.4
	}
	return gateProcess
}
```

- [ ] **Step 4: Wire the gate into `processMessage`**

```go
func (c *Consumer) processMessage(l logrus.FieldLogger, ctx context.Context, msg kafka.Message) bool {
	wctx := ctx
	for _, p := range c.headerParsers {
		wctx = p(wctx, msg.Headers)
	}

	// Ownership gate (FR-4.4). Runs BEFORE the tracing span and before any
	// domain handler, in consumer infrastructure — no domain processor
	// contains environment-conditional logic (FR-4.5, NG5).
	//
	// Both drop paths return TRUE: the message is acknowledged and the
	// offset advances. A drop is not a failure — returning false would
	// block the prefix-commit cursor and wedge the partition.
	msgEnv, _ := env.FromContext(wctx)()
	switch decide(env.CurrentRegistry(), env.Self(), c.service, msgEnv, env.Mismatched(wctx)) {
	case gateDropUnresolvable:
		gateDroppedUnresolvable.WithLabelValues(c.service, string(msgEnv)).Inc()
		l.WithField("environment", string(msgEnv)).WithField("topic", msg.Topic).
			Error("Dropping message: environment is unresolvable. No deployment will process it.")
		return true
	case gateSkipNotOwner:
		gateSkippedNotOwner.WithLabelValues(c.service, string(msgEnv)).Inc()
		return true
	}
	gateProcessed.WithLabelValues(c.service, string(msgEnv)).Inc()

	var span trace.Span
	// ...unchanged from here
```

`c.service` is a new field on `Consumer`, populated from `SERVICE_NAME` in
`AddConsumer`. If that variable is not set on every Deployment today, check
`deploy/k8s/base/env-configmap.yaml` and add it in Task 42 rather than
inventing a fallback — a gate that resolves the wrong service name is worse
than one that fails to start.

- [ ] **Step 5: Write the counter test**

Assert that a drop increments `atlas_kafka_gate_dropped_unresolvable_total`
and that the domain handler is not invoked. Use a `Consumer` with one
recording handler and `testutil.ToFloat64` from `prometheus/client_golang`.

- [ ] **Step 6: Run the tests**

Run: `cd libs/atlas-kafka && go build ./... && go test ./...`
Expected: PASS, including `TestExactlyOneDeploymentProcesses`.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka
git commit -m "feat(atlas-kafka): the ownership gate — exactly one processor, fail closed"
```

---

### Task 27: `ForEachOwnedEnvironment` — the autonomous-iteration helper

FR-6: every background loop iterates the environments it owns and executes
independently within each, both dimensions resolved **fresh each tick**
(FR-6.4) — which is exactly what correction C4 says the current code does not
do.

**Files:**
- Create: `libs/atlas-service/foreach.go`
- Create: `libs/atlas-service/foreach_test.go`
- Read-only: `libs/atlas-routine/routine.go:15` — `routine.Go`, the panic recovery per iteration
- Read-only: `services/atlas-transports/atlas.com/transports/main.go:93-151` — the C4 defect this replaces (tenants loaded once, closed over by the ticker goroutine)
- Read-only: `libs/atlas-env/registry.go` (Task 18), `libs/atlas-env/tenants.go` (Task 21)

Module root: `libs/atlas-service`.

**Interfaces:**
- Produces:
  ```go
  // TenantLister is the per-environment tenant source. Services pass their
  // own tenant processor's GetAll, bound to the environment's context.
  type TenantLister func(ctx context.Context) ([]tenant.Model, error)

  // ForEachOwnedEnvironment runs body once per (environment, tenant) this
  // deployment owns, SERIALLY, resolving BOTH dimensions fresh on every call.
  func ForEachOwnedEnvironment(l logrus.FieldLogger, ctx context.Context,
      service string, tenants TenantLister, body func(context.Context))

  // ForEachOwnedEnvironmentConcurrently is the same iteration with each
  // (environment, tenant) body in its own goroutine. Use it ONLY in a loop
  // that already ran its tenants concurrently.
  func ForEachOwnedEnvironmentConcurrently(l logrus.FieldLogger, ctx context.Context,
      service string, tenants TenantLister, body func(context.Context))
  ```

**Serial by default — this is a deliberate constraint, not an oversight.**
Every class-1 loop today is `for _, t := range tenants { work }`: sequential.
Making the helper concurrent would turn a one-second ticker into a burst of
goroutines across every tenant of every environment — a behavioural change
with real blast radius (connection-pool pressure, downstream fan-out, tick
overlap) that has nothing to do with environment isolation. This task adds
the environment dimension and **preserves each loop's existing concurrency
shape**. The concurrent variant exists for the loops that already
parallelised their tenants; Task 41/42 must state, per loop, which one it
used and why.

Fault isolation (FR-6.5) is obtained from a per-iteration `recover`, not from
a goroutine. Panic recovery and concurrency are separate concerns and
`routine.Go` happens to bundle them.

- [ ] **Step 1: Write the failing iteration tests**

```go
func TestForEachOwnedEnvironmentRunsOncePerTenantPerOwnedEnvironment(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	seen := map[string]int{}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(ctx context.Context) {
			seen[string(env.MustFromContext(ctx))]++
		})

	if seen["main"] != 2 || seen["pr-123"] != 2 {
		t.Fatalf("seen = %v, want 2 iterations each for main and pr-123", seen)
	}
}

func TestForEachOwnedEnvironmentRunsSerially(t *testing.T) {
	// The helper must preserve each loop's existing shape: today every
	// class-1 loop is `for _, t := range tenants { work }`. An unsynchronised
	// counter is the assertion — this test is run under -race, so a
	// concurrent implementation fails it rather than passing flakily.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	inFlight := 0
	maxInFlight := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(context.Context) {
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			time.Sleep(time.Millisecond)
			inFlight--
		})

	if maxInFlight != 1 {
		t.Fatalf("maxInFlight = %d, want 1 — the default helper must not "+
			"parallelise tenants that the caller's loop ran serially", maxInFlight)
	}
}

func TestForEachOwnedEnvironmentSkipsEnvironmentsThisDeploymentDoesNotOwn(t *testing.T) {
	// FR-6.3: a baseline deployment must not originate work for an
	// environment that overrides its service.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-monsters": "atlas-pr-123"}, Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	seen := map[string]int{}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		oneTenantPerEnvironment(t), func(ctx context.Context) {
			seen[string(env.MustFromContext(ctx))]++
		})

	if seen["pr-123"] != 0 {
		t.Fatalf("baseline originated %d iterations for an environment that overrides it", seen["pr-123"])
	}
	if seen["main"] != 1 {
		t.Fatalf("seen[main] = %d, want 1", seen["main"])
	}
}

func TestForEachOwnedEnvironmentWithNoRecordsDoesExactlyTodaysWork(t *testing.T) {
	// FR-6.6 / NFR-2: a deployment owning only main performs the same work
	// it does today, and no extra background work.
	env.SetRegistry(nil)

	iterations := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		func(context.Context) ([]tenant.Model, error) { return []tenant.Model{testTenant(t)}, nil },
		func(context.Context) { iterations++ })

	if iterations != 1 {
		t.Fatalf("iterations = %d, want 1", iterations)
	}
}

func TestForEachOwnedEnvironmentIsolatesFaults(t *testing.T) {
	// FR-6.5: one environment's panic does not stop another's iteration.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	completed := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		oneTenantPerEnvironment(t), func(ctx context.Context) {
			if env.MustFromContext(ctx) == env.Id("main") {
				panic("boom")
			}
			completed++
		})

	if completed != 1 {
		t.Fatalf("completed = %d; a panic in one environment stopped another", completed)
	}
}

func TestForEachOwnedEnvironmentConcurrentlyRunsBodiesInParallel(t *testing.T) {
	// The opt-in variant, for loops that ALREADY parallelised their tenants.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	started := make(chan struct{}, 4)
	release := make(chan struct{})
	go func() {
		for i := 0; i < 4; i++ {
			<-started
		}
		close(release)
	}()

	ForEachOwnedEnvironmentConcurrently(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(context.Context) {
			started <- struct{}{}
			<-release // deadlocks unless all four run concurrently
		})
}

func TestForEachOwnedEnvironmentReresolvesOwnershipEveryCall(t *testing.T) {
	// FR-6.4: a loop must not cache an ownership set across ticks. This is
	// the C4 defect stated as a test.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	count := func() int {
		n := 0
		ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
			oneTenantPerEnvironment(t), func(context.Context) { n++ })
		return n
	}

	if got := count(); got != 1 {
		t.Fatalf("first tick = %d, want 1", got)
	}
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	if got := count(); got != 2 {
		t.Fatalf("second tick = %d, want 2 — a new environment must be picked up without a restart", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-service && go test ./... -run ForEachOwnedEnvironment -v`
Expected: FAIL — `undefined: ForEachOwnedEnvironment`.

- [ ] **Step 3: Implement `foreach.go`**

```go
// ForEachOwnedEnvironment runs body once per (environment, tenant) pair this
// deployment owns. BOTH dimensions are resolved fresh on every call
// (FR-6.4): a tenant provisioned after this pod started must be picked up
// without a restart, because a baseline pod cannot be redeployed to serve an
// ephemeral environment (G7, NG6). The pre-existing pattern this replaces
// loaded the tenant list once before the ticker and closed over the slice
// (design C4) — do not reintroduce that shape.
//
// Iteration is SERIAL, preserving the shape of the loops it replaces — every
// class-1 loop today is `for _, t := range tenants { work }`. Adding
// concurrency here would be a behavioural change unrelated to environment
// isolation, so it is opt-in via ForEachOwnedEnvironmentConcurrently.
//
// Each (environment, tenant) body runs under its own recover so one
// environment's fault does not stop another's (FR-6.5). Recovery does NOT
// require a goroutine; routine.Go merely bundles the two.
func ForEachOwnedEnvironment(l logrus.FieldLogger, ctx context.Context,
	service string, tenants TenantLister, body func(context.Context)) {

	eachOwned(l, ctx, service, tenants, func(el logrus.FieldLogger, c context.Context) {
		safely(el, c, body)
	})
}

// ForEachOwnedEnvironmentConcurrently runs each (environment, tenant) body in
// its own goroutine and blocks until all complete. Use it ONLY where the loop
// being converted already ran its tenants concurrently — otherwise a
// one-second ticker becomes a burst of goroutines across every tenant of
// every environment.
func ForEachOwnedEnvironmentConcurrently(l logrus.FieldLogger, ctx context.Context,
	service string, tenants TenantLister, body func(context.Context)) {

	var wg sync.WaitGroup
	eachOwned(l, ctx, service, tenants, func(el logrus.FieldLogger, c context.Context) {
		wg.Add(1)
		routine.Go(el, c, func(gc context.Context) {
			defer wg.Done()
			body(gc)
		})
	})
	wg.Wait()
}

// eachOwned resolves the (environment, tenant) pairs this deployment owns and
// hands each to visit. BOTH dimensions are resolved fresh on every call
// (FR-6.4): a tenant provisioned after this pod started must be picked up
// without a restart, because a baseline pod cannot be redeployed to serve an
// ephemeral environment (G7, NG6). The pre-existing pattern this replaces
// loaded the tenant list once before the ticker and closed over the slice
// (design C4) — do not reintroduce that shape.
func eachOwned(l logrus.FieldLogger, ctx context.Context, service string,
	tenants TenantLister, visit func(logrus.FieldLogger, context.Context)) {

	for _, e := range env.CurrentRegistry().EnvironmentsOwnedBy(service) {
		ectx := env.WithContext(ctx, e)
		el := l.WithField("environment", string(e))
		ts, err := tenants(ectx)
		if err != nil {
			el.WithError(err).Error("Unable to list tenants; skipping this environment's iteration.")
			continue
		}
		for _, t := range ts {
			visit(el, tenant.WithContext(ectx, t))
		}
	}
}

// safely runs body with panic recovery, on the CALLING goroutine. It is
// deliberately not routine.Go: fault isolation and concurrency are separate
// concerns, and this helper needs only the first.
func safely(l logrus.FieldLogger, ctx context.Context, body func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			l.WithField("panic", fmt.Sprintf("%v", r)).
				WithField("stack", string(debug.Stack())).
				Error("Recovered panic in a per-environment iteration.")
		}
	}()
	body(ctx)
}
```

`tools/goroutine-guard.sh` enforces DOM-25 (background goroutines go through
`routine.Go`, never a bare `go`). `safely` spawns nothing, so it does not
trip the guard; `ForEachOwnedEnvironmentConcurrently` uses `routine.Go` as
required. Confirm the guard passes rather than assuming:
`./tools/goroutine-guard.sh`.

A tenant whose environment is not in the registry is never reached at all —
`EnvironmentsOwnedBy` only yields environments the registry knows, and the
tenant lister is already environment-filtered by Task 14. That is the
skip-on-unknown rule design §7.1 requires, obtained structurally rather than
as a check.

- [ ] **Step 4: Run the tests, including under `-race`**

Run: `cd libs/atlas-service && go build ./... && go test ./... && go test -race ./...`
Expected: PASS. `-race` matters here: `TestForEachOwnedEnvironmentRunsSerially`
uses unsynchronised counters on purpose, so a concurrent implementation fails
loudly instead of passing intermittently.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-service
git commit -m "feat(atlas-service): per-environment autonomous iteration helper"
```

---

### Task 28: `env-domain-guard` and `env-bootstrap-guard`

`env-domain-guard` enforces NG5/FR-4.5. `env-bootstrap-guard` is the gate on
enabling sparse mode at all (design §12): sparse is not selectable until every
service wires the registry, and CI proves it.

**Files:**
- Create: `tools/envguard/main.go`, `tools/envguard/go.mod`
- Create: `tools/envguard/testdata/src/{domainimport,missingbootstrap,ok}/` — fixtures
- Create: `tools/env-domain-guard.sh`
- Create: `tools/env-bootstrap-guard.sh`
- Create: `tools/envguard/bootstrap-allowlist.txt` — services legitimately exempt (`atlas-renders` if it does not use `service.Bootstrap`; check before writing)
- Modify: `tools/verify.sh` — add both steps
- Read-only: `tools/rediskeyguard/` — the analyzer skeleton
- Read-only: `libs/atlas-service/envregistry.go` (Task 20) — the option name the bootstrap guard greps for

**Interfaces:**
- Produces: `tools/env-domain-guard.sh` — fails when
  `github.com/Chronicle20/atlas/libs/atlas-env` is imported from a file under
  `services/` that is not `main.go` and not under a `kafka/` or `rest/`
  directory.
- Produces: `tools/env-bootstrap-guard.sh` — fails when a
  `services/*/atlas.com/*/main.go` calls `service.Bootstrap` without
  `service.WithEnvironmentRegistry`. Phase C makes this pass; **it is expected
  to fail until then** and is therefore added to `verify.sh` in Task 52, not
  here.

- [ ] **Step 1: Write the failing domain-guard fixture**

`tools/envguard/testdata/src/domainimport/processor.go`:

```go
package domainimport

import "github.com/Chronicle20/atlas/libs/atlas-env" // want `atlas-env imported from a domain package`

var _ = env.Key
```

and an `ok` fixture under a `kafka/` path that imports it cleanly.

- [ ] **Step 2: Run to verify it fails**

Run: `cd tools/envguard && go test ./...`
Expected: FAIL — no analyzer.

- [ ] **Step 3: Implement both checks**

The domain check is an import-path rule over file paths. The bootstrap check
is a source scan of `main.go` files, not an analyzer — a shell + `grep`
implementation matching `tools/service-registration-guard.sh`'s style is
appropriate and simpler than an AST pass.

- [ ] **Step 4: Run the domain guard against the fleet**

Run: `./tools/env-domain-guard.sh`
Expected: exit 0. It should pass trivially today — nothing in `services/`
imports `atlas-env` yet — and stay passing through Phase C, which touches
only `main.go`, `kafka/` and `rest/`.

- [ ] **Step 5: Run the bootstrap guard and record the expected failure**

Run: `./tools/env-bootstrap-guard.sh`
Expected: **non-zero**, listing all 64 services. Capture the count in the task
report; Task 52 asserts it has reached zero.

- [ ] **Step 6: Wire only the domain guard into `verify.sh`**

```sh
if touched '^services/.*\.go$|^tools/envguard/'; then
    step "env domain guard" ./tools/env-domain-guard.sh
else
    skip "env domain guard (no service Go file changed)"
fi
```

- [ ] **Step 7: Commit**

```bash
git add tools/envguard tools/env-domain-guard.sh tools/env-bootstrap-guard.sh tools/verify.sh
git commit -m "feat(tools): env domain guard (enforced) and env bootstrap guard (staged for Phase F)"
```

---

### Task 29: Observability — the log field, the metric labels, and the alert

FR-10: environment is emitted by the logging setup, not by callers.

**Files:**
- Modify: `libs/atlas-service/logger.go:31-43` — add an environment hook beside `serviceNameHook`
- Modify: `libs/atlas-service/logger_test.go`
- Modify: `libs/atlas-rest/requests/metrics.go` — add the `environment` label to the outbound REST counter
- Modify: `deploy/k8s/base/atlas-ingress.yaml` — add `$http_environment` to the nginx access log format
- Create: `docs/runbooks/sparse-environments.md` — the operator runbook, including the alert and the Loki selectors
- Read-only: `libs/atlas-service/fieldnorm.go` — the normalizer hook must stay LAST
- Read-only: `libs/atlas-kafka/consumer/gate.go` (Task 26) — the three counters

Module roots: `libs/atlas-service`, `libs/atlas-rest`.

**Interfaces:**
- Produces: an `environment` field on every log record emitted by a
  `CreateLogger` logger, sourced from `env.Self()`.
- Produces: `docs/runbooks/sparse-environments.md`.

**Cardinality budget (FR-10.3, design §14):** `environment` is a label on
exactly the three gate counters and the existing REST and Kafka message
counters. It is **excluded** from anything already labelled by
topic × partition × handler. Budget: ~10 concurrent environments, i.e. a
bounded small multiple on a handful of series. Write this paragraph into the
runbook.

- [ ] **Step 1: Write the failing logger test**

```go
func TestLoggerEmitsTheEnvironmentField(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "pr-123")
	var buf bytes.Buffer
	l := CreateLogger("atlas-test")
	l.SetOutput(&buf)
	l.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec["environment"] != "pr-123" {
		t.Fatalf("environment = %v, want \"pr-123\"", rec["environment"])
	}
}

func TestLoggerOmitsTheEnvironmentFieldOnMain(t *testing.T) {
	// NFR-7: main's log records are byte-identical to today's.
	t.Setenv("ATLAS_ENVIRONMENT", "")
	var buf bytes.Buffer
	l := CreateLogger("atlas-test")
	l.SetOutput(&buf)
	l.Info("hello")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := rec["environment"]; present {
		t.Fatal("environment field emitted with no ATLAS_ENVIRONMENT set")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-service && go test ./... -run LoggerEmitsTheEnvironment -v`
Expected: FAIL.

- [ ] **Step 3: Implement the hook**

```go
type environmentHook struct{ environment string }

func (h environmentHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h environmentHook) Fire(entry *logrus.Entry) error {
	if h.environment == "" {
		return nil // main: byte-identical to today's records (NFR-7)
	}
	if _, present := entry.Data["environment"]; !present {
		entry.Data["environment"] = h.environment
	}
	return nil
}
```

Register it in `CreateLogger` **after** `newServiceNameHook` and **before**
`fieldKeyNormalizerHook{}` — the normalizer must stay the last registered
hook so it sees keys added by earlier ones (the comment at `logger.go:9-11`).
A record whose context already carries a different environment keeps it: the
hook fills in, it does not overwrite.

- [ ] **Step 4: Add the nginx access-log field**

In the `http {}` block of `deploy/k8s/base/atlas-ingress.yaml`, define a log
format including `env=$http_environment` and set `access_log` to use it.
`underscores_in_headers on` at line 27 makes `$http_environment` resolve.

- [ ] **Step 5: Write the runbook**

`docs/runbooks/sparse-environments.md` covers: the mode-selection rules
(FR-9.3), the sparse floor (D6), how to read the three gate counters, the
alert, and the Loki selectors. Loki has **no `app` label** in this cluster —
selectors are:

```
{service_name="atlas-monsters"} | environment="pr-123"
```

The alert:

```
atlas_kafka_gate_dropped_unresolvable_total > 0
```

is the P0 signal for cross-environment leakage (FR-10.4). It should be zero
in steady state in every environment; a non-zero value means an operation
carried an environment nobody could resolve, and the message it names was not
processed by anyone.

- [ ] **Step 6: Run the tests**

Run: `cd libs/atlas-service && go build ./... && go test ./...` and
`cd libs/atlas-rest && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-service libs/atlas-rest deploy/k8s/base/atlas-ingress.yaml docs/runbooks/sparse-environments.md
git commit -m "feat(observability): environment log field, metric labels, and the leakage alert"
```

---
# Phase C — Services

64 Go services (`services/*/atlas.com/*/main.go`) wire the registry. The
change is mechanical and **inert**: with no Environment record present every
mechanism returns exactly today's answer, so each batch is independently
mergeable and observably changes nothing (design §12).

Three things change per service, and nothing else:

1. `main.go` passes `service.WithEnvironmentRegistry(serviceName)` to `service.Bootstrap`.
2. Every `consumer.SetHeaderParsers(...)` call gains `consumer.EnvHeaderParser`
   **after** `consumer.TenantHeaderParser`.
3. Every `requests.RootUrl(...)` call in the service's REST clients becomes
   `requests.RootUrlFor(ctx, ...)`, propagating the returned error.

**No domain package is touched.** If a domain package needs editing, the
abstraction is in the wrong place (NG5) — stop and report it rather than
working around it. `env-domain-guard` (Task 28) will catch the attempt.

---

### Task 29A: Provision `SERVICE_NAME` on every Deployment — BLOCKS Task 30

Inserted during execution. Task 26 gave `Consumer` a `service` field populated
from `SERVICE_NAME`, and its brief said to provision the variable "in Task 42".
Two facts found while reviewing Task 26 make that wrong:

1. `grep -rn "SERVICE_NAME" deploy/` returns **nothing** — the variable is
   absent from every base manifest and every overlay.
2. **Task 42 does not mention `SERVICE_NAME`.** Followed literally, the plan
   never provisions it at all.

**Why the ordering matters.** With `service == ""`, `MapRegistry.IsOwner`
(`libs/atlas-env/registry.go:186-200`) looks up `rec.Overrides[""]`, which never
matches a real service key. The override branch is therefore dead and ownership
collapses to `rec.Baseline == r.self`. For an environment overriding
`atlas-character`:

- `self=main,   service="", msgEnv=pr-123` → `gateProcess` (the baseline steals it)
- `self=pr-123, service="", msgEnv=pr-123` → `gateSkipNotOwner` (the owner refuses it)

That is not a race and not a dropped message — it is a **silent misroute** in
which the baseline permanently consumes every environment's overridden-service
traffic, while FR-4.6's "exactly one processor" still holds numerically and every
counter and test looks correct. The isolation guarantee behind FR-1.2 is what
breaks.

The defect is inert only while no domain consumer registers `EnvHeaderParser`:
`decide` returns at the `msgEnv == ""` guard before `service` is read. **Task 30
Step 2 is what makes it live**, and Tasks 33–40 then replicate it across ~215
sites — all of them landing *before* Task 42 at the end of Phase C. So the
variable must be provisioned here, ahead of Task 30, not at the end of the phase.

**Files:**
- Modify: `deploy/k8s/base/env-configmap.yaml` (or the per-service Deployment
  template that already injects `POD_NAMESPACE`) — inject `SERVICE_NAME` for
  every service, sourced from the existing per-service name rather than a new
  hand-maintained list
- Read-only: `libs/atlas-kafka/consumer/manager.go` — how `c.service` is read
- Read-only: `deploy/k8s/overlays/main/kustomization.yaml`,
  `deploy/k8s/overlays/pr/kustomization.yaml`

**Interfaces:** none — deployment configuration only.

- [ ] **Step 1: Confirm the injection point.** Find how `POD_NAMESPACE` reaches
  every service today and follow that mechanism exactly. Do not invent a second
  one, and do not hand-maintain a per-service list if the existing mechanism can
  derive the name.

- [ ] **Step 2: Inject `SERVICE_NAME` for every service**, in both the `main`
  and `pr` overlays.

- [ ] **Step 3: Prove it covers every service.** Render the manifests
  (`kustomize build` on both overlays) and assert every Deployment carries a
  non-empty `SERVICE_NAME` whose value matches the ownership key used in
  `Record.Overrides` — an env var that resolves to the wrong *form* of the name
  (e.g. `monsters` where the registry expects `atlas-monsters`) misroutes exactly
  as silently as an empty one. Quote the rendered output; do not assert by
  inspection of the template.

- [ ] **Step 4: Add a guard or a rendered-manifest assertion** so a newly-added
  service cannot land without `SERVICE_NAME`. Follow the change-gate lesson from
  Task 24: verify the guard's trigger predicate fires, not just its check.

---

### Task 30: The service-wiring recipe, established on `atlas-monsters`

This task's deliverable is as much the written recipe as the code: Tasks
31–40 follow it verbatim, so it must be complete and correct before they run.

**Prerequisite: Task 29A must have landed.** This task's Step 2 is what first
registers `EnvHeaderParser` on a domain consumer, which is the moment the
ownership gate starts reading `c.service`. Running it before `SERVICE_NAME` is
provisioned opens a silent-misroute window — see Task 29A.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go` — the `Bootstrap` option
- Modify: every `services/atlas-monsters/atlas.com/monsters/kafka/consumer/**/*.go` that calls `consumer.SetHeaderParsers`
- Modify: every `services/atlas-monsters/atlas.com/monsters/**/rest.go` that calls `requests.RootUrl`
- Create: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md` — the recipe Tasks 31–40 consume
- Read-only: `libs/atlas-service/envregistry.go` (Task 20) — `WithEnvironmentRegistry`
- Read-only: `libs/atlas-kafka/consumer/header.go` (Task 25) — `EnvHeaderParser`
- Read-only: `libs/atlas-rest/requests/url.go` (Task 23) — `RootUrlFor`

Module root: `services/atlas-monsters/atlas.com/monsters`.

**Interfaces:**
- Consumes: Tasks 20, 23, 25.
- Produces: `service-wiring-recipe.md`, the exact three edits with their
  before/after snippets and the two commands that find every site.

- [ ] **Step 1: Find every site in this service**

```sh
S=services/atlas-monsters/atlas.com/monsters
grep -rn "service.Bootstrap"        "$S"
grep -rn "SetHeaderParsers"         "$S"
grep -rn "requests.RootUrl("        "$S"
```

Record the counts. They are the recipe's checklist.

- [ ] **Step 2: Write the failing wiring test**

```go
// services/atlas-monsters/atlas.com/monsters/wiring_test.go
package main

import (
	"os"
	"strings"
	"testing"
)

// TestMainWiresTheEnvironmentRegistry pins the one line every service must
// carry. It is a source assertion rather than a behavioural one because the
// wiring's effect is inert until an Environment record exists (FR-1.8), so
// there is nothing observable to assert at this point in the migration.
func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

This same test file is copied into every service by Tasks 31–40. It duplicates
what `env-bootstrap-guard` checks fleet-wide, deliberately: the guard tells
the whole fleet's story at CI time, and the per-service test fails the batch's
own `go test ./...` immediately.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-monsters/atlas.com/monsters && go test . -run TestMainWires -v`
Expected: FAIL.

- [ ] **Step 4: Apply the three edits**

```go
// main.go — before
rt := service.Bootstrap(serviceName)
// main.go — after
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
// kafka/consumer/... — before
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)
// after — EnvHeaderParser must come AFTER TenantHeaderParser so the tenant
// is on the context when it reconciles (FR-7.7).
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
// rest client — before
func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, requests.RootUrl("MONSTERS"), id))
}
// after — the environment on ctx decides which ingress this call targets.
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "MONSTERS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

`requests.ErrorRequest[T]` may not exist. Check
`libs/atlas-rest/requests/provider.go` first; if there is no error-carrying
`Request` constructor, add one in `libs/atlas-rest` as part of this task —
it is a three-line helper and every one of the 64 services needs it, so
inventing a per-service workaround is the wrong shape.

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Write the recipe document**

`service-wiring-recipe.md` contains: the three grep commands from Step 1, the
three before/after snippets from Step 4 verbatim, the `wiring_test.go` source
from Step 2, the `EnvHeaderParser`-goes-last ordering rule and why, the
"no domain package is touched" rule, and the per-batch commands:

```sh
cd services/<svc>/atlas.com/<name> && go build ./... && go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters libs/atlas-rest docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md
git commit -m "feat(atlas-monsters): wire the environment registry; establish the service-wiring recipe"
```

---

### Task 31: `atlas-login` — registry wiring and socket-edge environment origination

`atlas-login` is one of the two mandatory overrides (D6) and one of the four
origination points for environment context (FR-2.2). A game client connects
to the PR's **own** login service, so the environment is established at the
edge the client actually connects to — from `env.Self()`, not from a header.

**Files:**
- Modify: `services/atlas-login/atlas.com/login/main.go` — the `Bootstrap` option
- Modify: `services/atlas-login/atlas.com/login/main.go:54` region — `SERVICE_ID` resolution stays unchanged (design C1: the row is already `Id`-keyed)
- Modify: the socket session/handler entry point — put `env.Self()` on the session context
- Modify: every `SetHeaderParsers` and `requests.RootUrl` site in the service
- Create: `services/atlas-login/atlas.com/login/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md` (Task 30)
- Read-only: `libs/atlas-socket/` — where a session's context is constructed

Module root: `services/atlas-login/atlas.com/login`.

**Interfaces:**
- Produces: every operation originating from a login socket session carries
  `env.Self()` on its context, and therefore emits it on every downstream
  REST call and Kafka message.

- [ ] **Step 1: Find the session context construction**

```sh
grep -rn "context.Background()\|WithContext" services/atlas-login/atlas.com/login | grep -iv test
```

The socket layer builds a context per session or per packet. Identify the one
place where the tenant is put on it — the environment goes on the same
context, at the same place.

- [ ] **Step 2: Write the failing origination test**

```go
func TestSocketSessionContextCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := newSessionContext(testTenant(t)) // the function found in Step 1
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}

func TestSocketSessionContextOnMainIsTheLegacyValue(t *testing.T) {
	t.Setenv(env.SelfVar, "")
	ctx := newSessionContext(testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("environment = %q, want the empty id", got)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-login/atlas.com/login && go test ./... -run SocketSessionContext -v`
Expected: FAIL.

- [ ] **Step 4: Apply the recipe plus the origination edit**

The three recipe edits, plus `ctx = env.WithContext(ctx, env.Self())` beside
the `tenant.WithContext` call in the session context builder.

`env.Self()` — not a registry lookup — is deliberate: a pod always knows its
own environment even when the registry is unreachable, which is what keeps
`main` fully functional during a registry outage (design §3).

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-login/atlas.com/login && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-login
git commit -m "feat(atlas-login): wire the environment registry; originate environment at the socket edge"
```

---

### Task 32: `atlas-channel` — registry wiring and socket-edge environment origination

Same shape as Task 31, on the second mandatory override. `atlas-channel` also
runs a configuration projection of its own (`configuration/projection/`) —
that projection is control-plane and is **not** converted here; its
disposition is Task 42.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go` — the `Bootstrap` option (note `main.go:176` resolves `SERVICE_ID`; unchanged)
- Modify: the socket session context builder — `env.Self()` on the context
- Modify: every `SetHeaderParsers` and `requests.RootUrl` site in the service
- Create: `services/atlas-channel/atlas.com/channel/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`
- Read-only: `services/atlas-channel/atlas.com/channel/configuration/projection/` — **not** modified by this task

Module root: `services/atlas-channel/atlas.com/channel`.

**Note on size:** `atlas-channel` is the largest service in the repo. If the
`SetHeaderParsers` / `RootUrl` site count exceeds what fits the tool-call
budget, commit what is done and report `PARTIAL` with the remaining file
list — do not compress by skipping the per-site build.

- [ ] **Step 1: Count the sites**

```sh
S=services/atlas-channel/atlas.com/channel
grep -rn "SetHeaderParsers" "$S" | wc -l
grep -rn "requests.RootUrl(" "$S" | wc -l
```

Record both counts in the task report before editing; they are the
completion criterion.

- [ ] **Step 2: Write the failing origination and wiring tests**

Same two tests as Task 31 Step 2, plus the `wiring_test.go` from the recipe.

- [ ] **Step 3: Run to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./... -run 'SocketSession|MainWires' -v`
Expected: FAIL.

- [ ] **Step 4: Apply the recipe plus the origination edit**

- [ ] **Step 5: Run the module tests**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Verify the site counts reached zero**

Run:
```sh
S=services/atlas-channel/atlas.com/channel
grep -rn "SetHeaderParsers" "$S" | grep -v EnvHeaderParser
grep -rn "requests.RootUrl(" "$S"
```
Expected: no output from either.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(atlas-channel): wire the environment registry; originate environment at the socket edge"
```

---

## The batch procedure (Tasks 33–40)

Tasks 33–40 each apply `service-wiring-recipe.md` (Task 30) to one batch of
services. The procedure is identical for every batch and is spelled out in
full inside each task, because an implementer sees only its own task.

Three edits per service, and nothing else:

1. `main.go` passes `service.WithEnvironmentRegistry(serviceName)` to `service.Bootstrap`.
2. Every `consumer.SetHeaderParsers(...)` gains `consumer.EnvHeaderParser`
   **after** `consumer.TenantHeaderParser`.
3. Every `requests.RootUrl(...)` becomes `requests.RootUrlFor(ctx, ...)`,
   propagating the returned error.

**Batch allocation:**

| Task | Services |
|---|---|
| 33 | `atlas-account`, `atlas-asset-expiration`, `atlas-ban`, `atlas-buddies`, `atlas-buffs`, `atlas-cashshop`, `atlas-chairs`, `atlas-chalkboards` |
| 34 | `atlas-character`, `atlas-character-factory`, `atlas-configurations`, `atlas-consumables`, `atlas-data`, `atlas-doors`, `atlas-drop-information`, `atlas-drops` |
| 35 | `atlas-effective-stats`, `atlas-expressions`, `atlas-fame`, `atlas-families`, `atlas-guilds`, `atlas-inventory`, `atlas-invites`, `atlas-keys` |
| 36 | `atlas-kites`, `atlas-map-actions`, `atlas-maps`, `atlas-marriages`, `atlas-merchant`, `atlas-messages`, `atlas-messengers`, `atlas-mini-games` |
| 37 | `atlas-monster-book`, `atlas-monster-death`, `atlas-mounts`, `atlas-mts`, `atlas-notes`, `atlas-npc-conversations`, `atlas-npc-shops`, `atlas-parties` |
| 38 | `atlas-party-quests`, `atlas-pets`, `atlas-portal-actions`, `atlas-portals`, `atlas-query-aggregator`, `atlas-quest`, `atlas-rankings`, `atlas-rates` |
| 39 | `atlas-reactor-actions`, `atlas-reactors`, `atlas-renders`, `atlas-reward-pools`, `atlas-rps`, `atlas-saga-orchestrator`, `atlas-skills`, `atlas-storage` |
| 40 | `atlas-summons`, `atlas-tenants`, `atlas-trades`, `atlas-transports`, `atlas-world` |

---

### Task 33: Wire the environment registry — batch 1

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md` — the three before/after snippets and the `wiring_test.go` source

**Services in this batch:** `atlas-account`, `atlas-asset-expiration`,
`atlas-ban`, `atlas-buddies`, `atlas-buffs`, `atlas-cashshop`,
`atlas-chairs`, `atlas-chalkboards`.

**Interfaces:**
- Consumes: `service.WithEnvironmentRegistry` (Task 20),
  `consumer.EnvHeaderParser` (Task 25), `requests.RootUrlFor` (Task 23), and
  the recipe (Task 30).
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8), so this batch is independently mergeable and observably
  changes nothing.

**No domain package is touched.** If a domain package appears to need
editing, the abstraction is in the wrong place (NG5) — stop and report it
rather than working around it.

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-account atlas-asset-expiration atlas-ban atlas-buddies \
         atlas-buffs atlas-cashshop atlas-chairs atlas-chalkboards; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

Record the counts; they are this task's completion criterion.

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

Copy this file verbatim into each service's `main` package directory as
`wiring_test.go`:

```go
package main

import (
	"os"
	"strings"
	"testing"
)

// TestMainWiresTheEnvironmentRegistry pins the one line every service must
// carry. It is a source assertion rather than a behavioural one because the
// wiring's effect is inert until an Environment record exists (FR-1.8).
func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
// main.go — before
rt := service.Bootstrap(serviceName)
// after
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
// kafka/consumer/... — before
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)
// after — EnvHeaderParser must come AFTER TenantHeaderParser so the tenant is
// on the context when it reconciles (FR-7.7).
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
// rest client — before
func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, requests.RootUrl("ACCOUNTS"), id))
}
// after — the environment on ctx decides which ingress this call targets.
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "ACCOUNTS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-account atlas-asset-expiration atlas-ban atlas-buddies \
         atlas-buffs atlas-cashshop atlas-chairs atlas-chalkboards; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-account atlas-asset-expiration atlas-ban atlas-buddies \
         atlas-buffs atlas-cashshop atlas-chairs atlas-chalkboards; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-account services/atlas-asset-expiration services/atlas-ban \
        services/atlas-buddies services/atlas-buffs services/atlas-cashshop \
        services/atlas-chairs services/atlas-chalkboards
git commit -m "feat(services): wire the environment registry — batch 1"
```

---

### Task 34: Wire the environment registry — batch 2

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-character`, `atlas-character-factory`,
`atlas-configurations`, `atlas-consumables`, `atlas-data`, `atlas-doors`,
`atlas-drop-information`, `atlas-drops`.

**Two notes specific to this batch:**

- `atlas-character` is one of the largest services in the repo. If the site
  count exceeds the tool-call budget, commit what is done and report
  `PARTIAL` with the remaining file list — do not compress by skipping the
  per-module build.
- `atlas-configurations` is the service that *publishes* the environment
  topic. It still **consumes** it: it needs the registry for its own
  `scope.Strict` calls (Task 13). Wire it exactly like any other service.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-character atlas-character-factory atlas-configurations \
         atlas-consumables atlas-data atlas-doors atlas-drop-information atlas-drops; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "CHARACTERS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-character atlas-character-factory atlas-configurations \
         atlas-consumables atlas-data atlas-doors atlas-drop-information atlas-drops; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-character atlas-character-factory atlas-configurations \
         atlas-consumables atlas-data atlas-doors atlas-drop-information atlas-drops; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character services/atlas-character-factory \
        services/atlas-configurations services/atlas-consumables \
        services/atlas-data services/atlas-doors \
        services/atlas-drop-information services/atlas-drops
git commit -m "feat(services): wire the environment registry — batch 2"
```

---

### Task 35: Wire the environment registry — batch 3

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-effective-stats`, `atlas-expressions`,
`atlas-fame`, `atlas-families`, `atlas-guilds`, `atlas-inventory`,
`atlas-invites`, `atlas-keys`.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-effective-stats atlas-expressions atlas-fame atlas-families \
         atlas-guilds atlas-inventory atlas-invites atlas-keys; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "GUILDS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-effective-stats atlas-expressions atlas-fame atlas-families \
         atlas-guilds atlas-inventory atlas-invites atlas-keys; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-effective-stats atlas-expressions atlas-fame atlas-families \
         atlas-guilds atlas-inventory atlas-invites atlas-keys; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-effective-stats services/atlas-expressions services/atlas-fame \
        services/atlas-families services/atlas-guilds services/atlas-inventory \
        services/atlas-invites services/atlas-keys
git commit -m "feat(services): wire the environment registry — batch 3"
```

---

### Task 36: Wire the environment registry — batch 4

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-kites`, `atlas-map-actions`, `atlas-maps`,
`atlas-marriages`, `atlas-merchant`, `atlas-messages`, `atlas-messengers`,
`atlas-mini-games`.

`atlas-marriages` also has two ticker-bearing schedulers; those are **Task
41's** work, not this batch's. Touch only `main.go`, `kafka/` and `rest/`
here.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-kites atlas-map-actions atlas-maps atlas-marriages \
         atlas-merchant atlas-messages atlas-messengers atlas-mini-games; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "MAPS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-kites atlas-map-actions atlas-maps atlas-marriages \
         atlas-merchant atlas-messages atlas-messengers atlas-mini-games; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-kites atlas-map-actions atlas-maps atlas-marriages \
         atlas-merchant atlas-messages atlas-messengers atlas-mini-games; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-kites services/atlas-map-actions services/atlas-maps \
        services/atlas-marriages services/atlas-merchant services/atlas-messages \
        services/atlas-messengers services/atlas-mini-games
git commit -m "feat(services): wire the environment registry — batch 4"
```

---

### Task 37: Wire the environment registry — batch 5

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-monster-book`, `atlas-monster-death`,
`atlas-mounts`, `atlas-mts`, `atlas-notes`, `atlas-npc-conversations`,
`atlas-npc-shops`, `atlas-parties`.

`atlas-mts` also has a ticker-bearing periodic task; that is **Task 42's**
work, not this batch's.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-monster-book atlas-monster-death atlas-mounts atlas-mts \
         atlas-notes atlas-npc-conversations atlas-npc-shops atlas-parties; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "PARTIES")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-monster-book atlas-monster-death atlas-mounts atlas-mts \
         atlas-notes atlas-npc-conversations atlas-npc-shops atlas-parties; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-monster-book atlas-monster-death atlas-mounts atlas-mts \
         atlas-notes atlas-npc-conversations atlas-npc-shops atlas-parties; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book services/atlas-monster-death services/atlas-mounts \
        services/atlas-mts services/atlas-notes services/atlas-npc-conversations \
        services/atlas-npc-shops services/atlas-parties
git commit -m "feat(services): wire the environment registry — batch 5"
```

---

### Task 38: Wire the environment registry — batch 6

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-party-quests`, `atlas-pets`,
`atlas-portal-actions`, `atlas-portals`, `atlas-query-aggregator`,
`atlas-quest`, `atlas-rankings`, `atlas-rates`.

Two notes: `atlas-party-quests` has a ticker in `main.go` — that conversion
is **Task 42's** work; here, add only the `Bootstrap` option. `atlas-quest`
contains two of the four direct `producer.Produce` call sites, already fixed
in Task 24; do not re-edit them.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-party-quests atlas-pets atlas-portal-actions atlas-portals \
         atlas-query-aggregator atlas-quest atlas-rankings atlas-rates; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "QUESTS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-party-quests atlas-pets atlas-portal-actions atlas-portals \
         atlas-query-aggregator atlas-quest atlas-rankings atlas-rates; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-party-quests atlas-pets atlas-portal-actions atlas-portals \
         atlas-query-aggregator atlas-quest atlas-rankings atlas-rates; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-party-quests services/atlas-pets services/atlas-portal-actions \
        services/atlas-portals services/atlas-query-aggregator services/atlas-quest \
        services/atlas-rankings services/atlas-rates
git commit -m "feat(services): wire the environment registry — batch 6"
```

---

### Task 39: Wire the environment registry — batch 7

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-reactor-actions`, `atlas-reactors`,
`atlas-renders`, `atlas-reward-pools`, `atlas-rps`,
`atlas-saga-orchestrator`, `atlas-skills`, `atlas-storage`.

Two notes: `atlas-renders` may call
`service.Bootstrap(serviceName, service.WithoutTracer())` — add the new
option alongside the existing one; if it does not call `Bootstrap` at all,
add it to `tools/envguard/bootstrap-allowlist.txt` with that reason and say
so in the report. `atlas-saga-orchestrator` has a ticker in `main.go` and
contains two direct `producer.Produce` sites already fixed in Task 24; here,
add only the `Bootstrap` option and the parser/URL edits.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-reactor-actions atlas-reactors atlas-renders atlas-reward-pools \
         atlas-rps atlas-saga-orchestrator atlas-skills atlas-storage; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all eight (or seven plus an allowlist entry for
`atlas-renders`).

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "SKILLS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-reactor-actions atlas-reactors atlas-renders atlas-reward-pools \
         atlas-rps atlas-saga-orchestrator atlas-skills atlas-storage; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed**

```sh
for s in atlas-reactor-actions atlas-reactors atlas-renders atlas-reward-pools \
         atlas-rps atlas-saga-orchestrator atlas-skills atlas-storage; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
```
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-reactor-actions services/atlas-reactors services/atlas-renders \
        services/atlas-reward-pools services/atlas-rps services/atlas-saga-orchestrator \
        services/atlas-skills services/atlas-storage tools/envguard
git commit -m "feat(services): wire the environment registry — batch 7"
```

---

### Task 40: Wire the environment registry — batch 8

**Files:**
- Modify, per service: `services/<svc>/atlas.com/<name>/main.go`
- Modify, per service: every file matching `grep -rln "SetHeaderParsers" services/<svc>`
- Modify, per service: every file matching `grep -rln "requests.RootUrl(" services/<svc>`
- Create, per service: `services/<svc>/atlas.com/<name>/wiring_test.go`
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md`

**Services in this batch:** `atlas-summons`, `atlas-tenants`, `atlas-trades`,
`atlas-transports`, `atlas-world`.

This is the last batch. After it, every one of the 64 Go services is wired —
`tools/env-bootstrap-guard.sh` should reach exit 0, which Task 52 asserts.

`atlas-transports` has the reference C4 ticker; converting it is **Task 41's**
work, not this batch's. Here, add only the `Bootstrap` option and the
parser/URL edits.

**Interfaces:**
- Consumes: Tasks 20, 23, 25, 30.
- Produces: nothing new. Every edit is inert until an Environment record
  exists (FR-1.8).

**No domain package is touched** (NG5).

- [ ] **Step 1: Count the sites for every service in the batch**

```sh
for s in atlas-summons atlas-tenants atlas-trades atlas-transports atlas-world; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  printf '%s parsers=%s rooturl=%s\n' "$s" \
    "$(grep -rl 'SetHeaderParsers' "$d" | wc -l)" \
    "$(grep -rl 'requests.RootUrl(' "$d" | wc -l)"
done
```

- [ ] **Step 2: Add the failing wiring test to every service in the batch**

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}
```

Run, per service: `cd <module root> && go test . -run TestMainWires -v`
Expected: FAIL for all five.

- [ ] **Step 3: Apply the three edits to every service in the batch**

```go
rt := service.Bootstrap(serviceName, service.WithEnvironmentRegistry(serviceName))
```

```go
consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser)
```

```go
func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := requests.RootUrlFor(ctx, "WORLDS")
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getById, root, id))
}
```

- [ ] **Step 4: Build and test every module in the batch**

```sh
for s in atlas-summons atlas-tenants atlas-trades atlas-transports atlas-world; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $s"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 5: Verify no site was missed, fleet-wide**

```sh
for s in atlas-summons atlas-tenants atlas-trades atlas-transports atlas-world; do
  d=$(dirname services/$s/atlas.com/*/main.go)
  grep -rn "SetHeaderParsers" "$d" | grep -v EnvHeaderParser
  grep -rn "requests.RootUrl(" "$d"
done
./tools/env-bootstrap-guard.sh; echo "guard exit=$?"
```
Expected: no output from the loop. The guard should now exit 0 — if it does
not, the services it names were missed by an earlier batch; report them
rather than fixing them silently here, so the gap is visible.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons services/atlas-tenants services/atlas-trades \
        services/atlas-transports services/atlas-world
git commit -m "feat(services): wire the environment registry — batch 8 (final)"
```

---

### Task 41: Per-environment iteration for the eight per-tenant ticker loops

FR-6.1. Design §7.2 classifies the 18 ticker-bearing files into three
classes; this task converts **class 1**, the eight that do per-tenant domain
work. Correction C4 is the reason each one needs converting rather than
wrapping: they load the tenant list once, before the ticker, and the ticker
goroutine closes over the slice.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/main.go:93-151` — the reference case
- Modify: `services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go`
- Modify: `services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go`
- Modify: `services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go`
- Create/modify the matching test file beside each
- Read-only: `libs/atlas-service/foreach.go` (Task 27) — `ForEachOwnedEnvironment`
- Read-only: `libs/atlas-service/foreach_test.go` — the fault-isolation and re-resolution tests, which cover the helper; per-service tests assert only the call shape

Module roots: `services/atlas-transports/atlas.com/transports`,
`services/atlas-marriages/atlas.com/marriages`,
`services/atlas-asset-expiration/atlas.com/asset-expiration`.

**Interfaces:**
- Consumes: `service.ForEachOwnedEnvironment`, `service.TenantLister` (Task 27).
- Produces: no exported change.

- [ ] **Step 1: Write the failing "no cached tenant list" test for `atlas-transports`**

```go
// services/atlas-transports/atlas.com/transports/tick_test.go
package main

import (
	"os"
	"strings"
	"testing"
)

// TestTickerDoesNotCloseOverACachedTenantList pins design C4: the tenant
// list must be resolved inside the tick, not before it. A tenant
// provisioned after this pod started must be picked up without a restart,
// because a baseline pod cannot be redeployed to serve an ephemeral
// environment (G7, NG6).
func TestTickerDoesNotCloseOverACachedTenantList(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "service.ForEachOwnedEnvironment") {
		t.Fatal("tick loop does not use service.ForEachOwnedEnvironment")
	}
	tickerAt := strings.Index(s, "time.NewTicker")
	getAllAt := strings.Index(s, "NewProcessor(l, rt.Context()).GetAll()")
	if getAllAt >= 0 && getAllAt < tickerAt {
		t.Fatal("tenant list is still loaded before the ticker and closed over (design C4)")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-transports/atlas.com/transports && go test . -run TestTickerDoesNot -v`
Expected: FAIL.

- [ ] **Step 3: Convert `atlas-transports`**

```go
// before — tenants loaded once, closed over by the ticker goroutine (C4)
tenants, err := tenant2.NewProcessor(l, rt.Context()).GetAll()
...
routine.Go(l, rt.Context(), func(_ context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rt.Context().Done():
			return
		case <-ticker.C:
			for _, t := range tenants {
				ctx := tenant.WithContext(rt.Context(), t)
				transport.NewProcessor(l, ctx).UpdateRoutes()
				ip := instance.NewProcessor(l, ctx)
				_ = ip.TickBoardingExpirationAndEmit()
				_ = ip.TickArrivalAndEmit()
				_ = ip.TickStuckTimeoutAndEmit()
			}
		}
	}
})

// after — both dimensions resolved fresh every tick (FR-6.4)
listTenants := func(ctx context.Context) ([]tenant.Model, error) {
	return tenant2.NewProcessor(l, ctx).GetAll()
}
routine.Go(l, rt.Context(), func(_ context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rt.Context().Done():
			return
		case <-ticker.C:
			service.ForEachOwnedEnvironment(l, rt.Context(), serviceName, listTenants,
				func(ctx context.Context) {
					transport.NewProcessor(l, ctx).UpdateRoutes()
					ip := instance.NewProcessor(l, ctx)
					_ = ip.TickBoardingExpirationAndEmit()
					_ = ip.TickArrivalAndEmit()
					_ = ip.TickStuckTimeoutAndEmit()
				})
		}
	}
})
```

`ForEachOwnedEnvironment` — the **serial** variant — is correct here because
the loop it replaces is serial (`for _, t := range tenants`). Record in the
task report, for each of the four loops in this task, which variant you used
and what the pre-existing loop's concurrency shape was. Reaching for
`ForEachOwnedEnvironmentConcurrently` on a loop that was serial is a
behavioural change unrelated to this project and must not happen silently.

The **startup reconciliation** at `main.go:100-107` and the **graceful
shutdown** loop at `main.go:145-151` iterate the same cached slice and get
the same treatment: both become `service.ForEachOwnedEnvironment` calls with
the same `listTenants`. A shutdown that iterates a stale list would skip an
environment's characters entirely.

- [ ] **Step 4: Convert the three remaining class-1 files in this task**

Same shape. `atlas-marriages`' two schedulers and
`atlas-asset-expiration`'s periodic task each have their own tenant loop;
read each before editing and record in the report whether it exhibited the
C4 cached-list defect.

- [ ] **Step 5: Run the module tests**

```sh
cd services/atlas-transports/atlas.com/transports && go build ./... && go test ./...
cd services/atlas-marriages/atlas.com/marriages && go build ./... && go test ./...
cd services/atlas-asset-expiration/atlas.com/asset-expiration && go build ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports services/atlas-marriages services/atlas-asset-expiration
git commit -m "feat(services): per-environment iteration for the transports, marriages and asset-expiration loops"
```

---

### Task 42: The remaining ticker loops, and the class-2 / class-3 dispositions

Four more class-1 conversions, then the ten files that are **not** converted —
each with the evidence design §7.2 demands. "It looks local" is not a
disposition.

**Files:**
- Modify: `services/atlas-mts/atlas.com/mts/task/periodic.go`
- Modify: `services/atlas-party-quests/atlas.com/party-quests/main.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`
- Modify: `services/atlas-login/atlas.com/login/main.go` — its ticker
- Create: `docs/tasks/task-232-sparse-ephemeral-environments/ticker-dispositions.md`
- Read-only (class 2, no change): `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/reservation/cache.go`, `services/atlas-channel/atlas.com/channel/character/chakra/registry.go`, `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`, `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go`, `services/atlas-data/atlas.com/data/runtime/ingest/heartbeat.go`, `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go`
- Read-only (class 3, control-plane): `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go`, `services/atlas-login/atlas.com/login/configuration/projection/loop.go`, `services/atlas-character-factory/atlas.com/character-factory/configuration/bridge.go`, `services/atlas-world/atlas.com/world/configuration/bridge.go`
- Read-only: `libs/atlas-service/foreach.go` (Task 27)

**Interfaces:**
- Produces: `ticker-dispositions.md` — all 18 files, each with a class, a
  disposition, and a file:line citation. FR-6.1 says "the 18 ticker-bearing
  files are the starting inventory; design must confirm completeness" — this
  document is that confirmation.

- [ ] **Step 1: Re-derive the inventory rather than trusting the count**

```sh
grep -rl "NewTicker" services | sort
```

Expected: 18 paths. If the count differs from 18, the new files are the
finding — classify them too and say so in the report.

Also check for tickers that are not `time.NewTicker`:

```sh
grep -rn "time.After\|time.Tick(\|cron\." services | grep -v _test | grep -v vendor
```

Anything that schedules recurring work is in scope for FR-6.1 regardless of
how it is spelled.

- [ ] **Step 2: Convert the four remaining class-1 loops**

Same shape as Task 41 Step 3, including the concurrency-shape record: name
the variant used per loop and the pre-existing loop's shape.
`atlas-saga-orchestrator`'s loop is a
**persisted-work path**: its sweeper reads saga rows and must reconstruct the
environment from the tenant on the row (design §8.3), not from `env.Self()`.
`ForEachOwnedEnvironment` gives it that for free — the tenant lister is
environment-filtered (Task 14), so a saga row belonging to a tenant of an
unknown environment is simply never reached.

- [ ] **Step 3: Write the failing per-loop tests**

Copy `tick_test.go` from Task 41 Step 1 into each of the four services,
adapting the file it reads.

- [ ] **Step 4: Write the class-2 dispositions with evidence**

For each of the six class-2 files, cite the lines proving it neither reads
tenant-scoped state nor emits a message:

```markdown
### `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go`

**Class 2 — process-local watchdog. No change.**

- No tenant loop: `grep -n "tenant\." <file>` → *paste the output, or "no matches"*
- No emitted work: `grep -n "producer\.\|Emit\|requests\." <file>` → *paste*
- What it does instead: *one sentence, with the line range*
```

If any of the six turns out to read tenant-scoped state or emit a message, it
is class 1 and must be converted in this task. Do not record a class-2
disposition you cannot evidence.

- [ ] **Step 5: Write the class-3 dispositions**

The four control-plane projections are environment-scoped, not tenant-scoped
(D5). Each already replays a compacted topic from `FirstOffset` under a
per-process consumer group and needs no per-environment iteration: a sparse
override projects its own environment's configuration because its
`SERVICE_ID` selects its own row (design C1). State that per file, with the
`SERVICE_ID` resolution line as the citation.

- [ ] **Step 6: Run the module tests**

```sh
for d in services/atlas-mts/atlas.com/mts \
         services/atlas-party-quests/atlas.com/party-quests \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         services/atlas-login/atlas.com/login; do
  ( cd "$d" && go build ./... && go test ./... ) || echo "FAILED: $d"
done
```
Expected: no `FAILED:` lines.

- [ ] **Step 7: Commit**

```bash
git add services docs/tasks/task-232-sparse-ephemeral-environments/ticker-dispositions.md
git commit -m "feat(services): finish per-environment iteration; record every ticker disposition"
```

---
# Phase D — Deployment

---

### Task 43: Per-service namespace variables in the ingress routing table

Design §5.3, decision P3. The routing table becomes **static for an
environment's entire life** — the override set is fixed when the environment
is created — so there is no reload, no ConfigMap propagation delay, and no
per-request registry lookup. PRD open question 5 dissolves: neither a
regenerated ConfigMap nor `njs` is needed.

**Files:**
- Modify: `tools/gen-routes.sh` — emit `${NS_<SERVICE>}` instead of `${POD_NAMESPACE}`, and a companion variable list
- Modify: `deploy/k8s/base/routes.conf.template.generated` — regenerated output (never hand-edited)
- Create: `deploy/k8s/base/ns-vars.generated.yaml` — the `NS_*` env block, generated by the same script, patched into the ingress Deployment
- Modify: `deploy/k8s/base/atlas-ingress.yaml:82-89` — `NGINX_ENVSUBST_FILTER` widened to the `NS_` prefix; the generated env block included
- Modify: `tools/verify.sh:331` — the existing `deploy/` gate gains `./tools/gen-routes.sh --check`
- Create: `tools/gen-routes_test.sh` — the drift check's own test, mirroring `tools/gen-lb-ports_test.sh`
- Read-only: `deploy/shared/routes.conf` — the source of truth for the upstream list
- Read-only: `tools/gen-lb-ports.sh:1-40` — the `--check` / marker-block convention to copy

**Interfaces:**
- Produces: `tools/gen-routes.sh --check` — exit 1 on drift, no files modified.
- Produces: one `NS_<SERVICE>` variable per distinct `atlas-<svc>` upstream in
  `deploy/shared/routes.conf`, named by uppercasing the service name and
  replacing `-` with `_` (`atlas-party-quests` → `NS_ATLAS_PARTY_QUESTS`).
  In `main` and in isolated mode every one is `$(POD_NAMESPACE)` — byte-identical
  behaviour (NFR-7).

- [ ] **Step 1: Write the failing generator test**

`tools/gen-routes_test.sh`:

```sh
#!/usr/bin/env sh
set -eu
REPO_ROOT="$(git rev-parse --show-toplevel)"
fail() { echo "FAIL: $*" >&2; exit 1; }

# 1. Every upstream in the generated routes uses a per-service NS_ variable.
if grep -q '\${POD_NAMESPACE}' "$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated"; then
    fail "routes still reference \${POD_NAMESPACE}; expected per-service NS_ variables"
fi

# 2. Every NS_ variable referenced in the routes is defined in the env block.
for v in $(grep -o 'NS_[A-Z0-9_]*' "$REPO_ROOT/deploy/k8s/base/routes.conf.template.generated" | sort -u); do
    grep -q "name: $v" "$REPO_ROOT/deploy/k8s/base/ns-vars.generated.yaml" \
        || fail "$v referenced in routes but not defined in ns-vars.generated.yaml"
done

# 3. Every defined variable defaults to POD_NAMESPACE (NFR-7).
grep -c 'value: \$(POD_NAMESPACE)' "$REPO_ROOT/deploy/k8s/base/ns-vars.generated.yaml" >/dev/null \
    || fail "ns-vars defaults are not \$(POD_NAMESPACE)"

# 4. --check is clean against the checked-in output.
"$REPO_ROOT/tools/gen-routes.sh" --check || fail "gen-routes drift"

echo "PASS"
```

- [ ] **Step 2: Run to verify it fails**

Run: `./tools/gen-routes_test.sh`
Expected: FAIL at check 1.

- [ ] **Step 3: Rewrite `gen-routes.sh`**

Replace the three `sed -E` substitutions with ones that derive the variable
name from the captured service name. In POSIX shell the per-line
uppercase-and-substitute is easiest with `awk`:

```sh
awk '
  function nsvar(svc,  v) { v = toupper(svc); gsub(/-/, "_", v); return "NS_" v }
  {
    if (match($0, /set \$u "(atlas-[a-z-]+):8080"/, m)) {
      printf "  set $u \"%s.${%s}.svc.cluster.local:8080\";\n", m[1], nsvar(m[1])
      next
    }
    print
  }
' "$SRC" > "$OUT"
```

`gawk`'s three-argument `match` is not POSIX. If the CI image has only
`mawk`/`busybox awk`, use `sed` to extract the service name into a shell loop
instead — verify which `awk` the image ships before choosing, and record the
answer in the task report.

Emit `ns-vars.generated.yaml` from the same pass:

```yaml
# GENERATED by tools/gen-routes.sh — do not edit.
# One variable per routed service. In main and in isolated mode every value
# is $(POD_NAMESPACE), which reproduces the pre-task-232 routing table
# byte-for-byte (NFR-7). A sparse overlay overrides only the services it
# does not deploy, pointing them at the baseline's namespace.
- name: NS_ATLAS_ACCOUNT
  value: $(POD_NAMESPACE)
```

Add `--check`, copying `gen-lb-ports.sh`'s implementation: generate to a temp
directory, diff against the checked-in files, exit 1 on drift, modify nothing.

- [ ] **Step 4: Wire the variables into the ingress Deployment**

`deploy/k8s/base/atlas-ingress.yaml`: widen the filter and include the block.

```yaml
        - name: NGINX_ENVSUBST_FILTER
          # POD_NAMESPACE is still substituted for any upstream that has not
          # been given its own variable; NS_ covers the per-service ones.
          # Everything else ($request_uri, $u, $http_*) stays untouched.
          value: "POD_NAMESPACE|NS_"
```

Verify `NGINX_ENVSUBST_FILTER` accepts a regex in the nginx image version
pinned here (the existing comment says "requires nginx image >= 1.25.4").
If it only accepts a literal prefix, emit `POD_NAMESPACE` as one of the
`NS_*`-shaped variables instead and set the filter to `NS_` alone — do not
guess; check the image's `docker-entrypoint.d/20-envsubst-on-templates.sh`.

- [ ] **Step 5: Regenerate and run the test**

Run: `./tools/gen-routes.sh && ./tools/gen-routes_test.sh`
Expected: PASS.

- [ ] **Step 6: Prove `main` is byte-identical**

```sh
git stash && ./tools/gen-routes.sh && cp deploy/k8s/base/routes.conf.template.generated /tmp/before.conf
git stash pop && ./tools/gen-routes.sh
POD_NAMESPACE=atlas-main envsubst < deploy/k8s/base/routes.conf.template.generated > /tmp/after-raw.conf
```

Then substitute every `NS_*` to `atlas-main` in `/tmp/after-raw.conf` and
diff against `/tmp/before.conf` rendered with `POD_NAMESPACE=atlas-main`.
Expected: empty diff. Paste the diff (or its absence) into the task report —
NFR-7 is asserted here, not assumed.

- [ ] **Step 7: Add the drift check to `verify.sh`**

At the existing `deploy/` gate (line 331), beside `gen-lb-ports.sh --check`:

```sh
    step "routes drift"        ./tools/gen-routes.sh --check
```

- [ ] **Step 8: Commit**

```bash
git add tools/gen-routes.sh tools/gen-routes_test.sh tools/verify.sh deploy/k8s/base
git commit -m "feat(deploy): per-service namespace variables in the ingress routing table"
```

---

### Task 44: The sparse overlay

**Files:**
- Create: `deploy/k8s/overlays/pr-sparse/kustomization.yaml`
- Create: `deploy/k8s/overlays/pr-sparse/environment-record.yaml` — the Job that POSTs the environment record to `atlas-configurations`
- Create: `deploy/k8s/overlays/pr-sparse/ns-overrides.yaml` — the `NS_*` patch pointing non-overridden services at the baseline namespace
- Create: `deploy/k8s/overlays/pr-sparse/README.md`
- Read-only: `deploy/k8s/overlays/pr/kustomization.yaml` — the isolated overlay; the sparse one is derived from it by **removing** the three isolation substrates
- Read-only: `deploy/k8s/overlays/pr/patches/{db-name-suffix,consumer-group-env,lb-allocate,ingress-host}.yaml`
- Read-only: `deploy/k8s/base/ns-vars.generated.yaml` (Task 43)

**Interfaces:**
- Consumes: `NS_*` (Task 43), the environments REST resource (Task 19).
- Produces: an overlay that builds **only** the override set's Deployments
  plus the ingress, and an environment record naming them.

**What the sparse overlay does NOT carry**, relative to `overlays/pr`
(this is the whole point of D1):

| Removed | Why |
|---|---|
| `patches/db-name-suffix.yaml` | Shared databases (D1) |
| topic suffixing in the `atlas-env` `configMapGenerator` | Sparse consumes the unsuffixed baseline topics (FR-4.8) |
| `ATLAS_ENV` in the `atlas-env` ConfigMap | Makes the Redis key prefix inert (design §9); `computeKeyPrefix("")` is already the legacy path, so no code change |
| `wave0-create-dbs.yaml` | No per-environment databases |

**What it keeps:** `patches/lb-allocate.yaml`, `patches/ingress-host.yaml`,
`patches/consumer-group-env.yaml`, `ingress-route.yaml`, the replica-1 patch,
and the sync-bootstrap / predelete-purge / pihole Jobs.

**What it adds:** `ATLAS_ENVIRONMENT=pr-<N>` in the `atlas-env` ConfigMap
(read by `env.Self()`), the `NS_*` overrides, and the environment-record Job.

- [ ] **Step 1: Enumerate the base resources so the overlay can subtract**

```sh
grep -l "^kind: Deployment" deploy/k8s/base/atlas-*.yaml | sed 's|.*/||;s|\.yaml||' | sort
```

Expected: 64 names. The sparse overlay includes `../../base` and then deletes
every Deployment **not** in the override set, using the `$patch: delete`
idiom already used in `overlays/pr/kustomization.yaml` for the minio
reconcile resources. The delete list is generated at CI time by Task 50, so
the checked-in overlay carries a `PLACEHOLDER_DELETE_BLOCK` sentinel in the
same style as the existing `PLACEHOLDER_*` sentinels.

- [ ] **Step 2: Write the overlay**

Header comment first, stating the sentinel contract exactly as
`overlays/pr/kustomization.yaml:1-20` does — the CI job `sed`-substitutes
`PLACEHOLDER_PR_NUMBER`, `PLACEHOLDER_FULL_SHA`, `PLACEHOLDER_DELETE_BLOCK`
and `PLACEHOLDER_NS_OVERRIDES` and force-pushes to `bot/pr-<N>-resolved`.

- [ ] **Step 3: Write the environment-record Job**

A sync-wave Job that POSTs to `/api/configurations/environments`:

```yaml
# The environment record is the single source of truth every pod projects.
# It is created in PROVISIONING: during PROVISIONING baseline deployments
# continue to own the environment's services, overrides receive no work, and
# the ingress does not route (FR-5.2). Task 45's offset seeding and the
# override rollout happen next; the phase is flipped to ACTIVE last.
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-environment-record
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    argocd.argoproj.io/sync-options: Force=true,Replace=true
```

Its body builds the record from the `NS_*` override list and POSTs it with
`phase: PROVISIONING`.

- [ ] **Step 4: Verify the overlay builds**

Run:
```sh
sed -e 's/PLACEHOLDER_PR_NUMBER/999/g' -e 's/PLACEHOLDER_FULL_SHA/deadbeef/g' \
    -e 's/PLACEHOLDER_DELETE_BLOCK//' -e 's/PLACEHOLDER_NS_OVERRIDES//' \
    -i.bak deploy/k8s/overlays/pr-sparse/*.yaml
kustomize build deploy/k8s/overlays/pr-sparse | grep -c "^kind: Deployment"
```
Expected: a build with no error, and a Deployment count equal to the base's
(the delete block is empty in this smoke test). Restore the `.bak` files
afterwards.

- [ ] **Step 5: Verify the subtraction works with a real delete block**

Fill `PLACEHOLDER_DELETE_BLOCK` with delete patches for every Deployment
except `atlas-ingress`, `atlas-login`, `atlas-channel` and
`atlas-character`, rebuild, and assert exactly 4 Deployments remain. This is
the mechanical half of the §17 proof and must pass before Task 55 relies on it.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/overlays/pr-sparse
git commit -m "feat(deploy): the sparse PR overlay"
```

---

### Task 45: Seed override consumer-group offsets at provisioning time

FR-4.9 and PRD open question 2. A new group on a long-lived shared topic
either replays the whole retention window (`FirstOffset`, the library default
at `libs/atlas-kafka/consumer/config.go:37`) or loses anything produced
between group creation and first poll (`LastOffset`). Neither is acceptable.

**Files:**
- Modify: `deploy/k8s/base/atlas-kafka-precreate.yaml` — the Job gains an offset-seeding pass
- Create: `deploy/k8s/base/atlas-kafka-precreate_test.sh` — a shell test of the seeding logic against a local Kafka, skipped when `BOOTSTRAP_SERVERS` is unset
- Modify: `docs/runbooks/sparse-environments.md` (Task 29) — how to verify a group is seeded
- Read-only: `libs/atlas-kafka/consumergroup/resolver.go:38-50` — how group names are resolved
- Read-only: `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml` — the per-PR group naming, unchanged

**Interfaces:**
- Produces: for each override consumer group × subscribed topic, a committed
  offset at end-of-log, created **while the group is empty and therefore
  resettable**, at sync-wave 0 — before any Deployment starts. Argo CD
  already health-checks this Job and holds Deployments (sync-wave 10) until
  it completes, so the ordering is enforced by existing machinery.

**Why this is provably lossless, not merely likely-lossless (design §6.3):**

1. Offsets are committed at end-of-log at instant *T*.
2. Committed offsets take precedence over `startOffset`, so the replay
   question disappears.
3. No message for this environment can exist before *T*: the environment's
   tenant does not exist and its ingress does not route until activation,
   which is strictly after *T*.
4. Every message after *T* is consumed.

- [ ] **Step 1: Write the failing seeding test**

```sh
#!/usr/bin/env sh
# deploy/k8s/base/atlas-kafka-precreate_test.sh
# Requires a reachable Kafka; skips otherwise.
set -eu
[ -n "${BOOTSTRAP_SERVERS:-}" ] || { echo "SKIP: BOOTSTRAP_SERVERS unset"; exit 0; }

TOPIC="atlas-precreate-test-$$"
GROUP="atlas-precreate-test-group-$$"
kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$TOPIC" --partitions 2

# Produce three messages BEFORE seeding: a correctly seeded group must not
# replay them.
printf 'a\nb\nc\n' | kafka-console-producer.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC"

seed_group "$GROUP" "$TOPIC"   # the function under test, sourced from the Job script

for p in 0 1; do
    off=$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --group "$GROUP" --describe 2>/dev/null \
          | awk -v p="$p" '$3==p {print $4}')
    [ -n "$off" ] && [ "$off" != "-" ] \
        || { echo "FAIL: partition $p has no committed offset"; exit 1; }
done
echo "PASS"
```

Extract the Job's inline script into a sourceable shell file so the test can
call `seed_group` — a script that exists only inside a YAML heredoc cannot be
tested, and this one carries a correctness argument.

- [ ] **Step 2: Run to verify it fails**

Run: `BOOTSTRAP_SERVERS=<broker> ./deploy/k8s/base/atlas-kafka-precreate_test.sh`
Expected: FAIL — `seed_group: not found`.

- [ ] **Step 3: Implement `seed_group`**

```sh
# Commit end-of-log offsets for an override consumer group on one topic,
# while the group is empty and therefore resettable. Runs at sync-wave 0,
# before any Deployment starts (design §6.3).
seed_group() {
    group="$1"; topic="$2"
    kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$group" --topic "$topic" --reset-offsets --to-latest --execute >/dev/null
}
```

The Job enumerates override groups from the `KAFKA_CONSUMER_GROUP` value in
its `envFrom`-injected `atlas-env` ConfigMap and the topic list it already
builds, and calls `seed_group` for each pair. Skip the whole pass when
`KAFKA_CONSUMER_GROUP` is unset — that is `main`, whose groups already exist
and must not be reset (NG6). Assert that skip in the test.

- [ ] **Step 4: Add the readiness observation**

The activation gate (FR-5.3) needs an **observable** signal that the group is
initialized, not an inference. Add to the Job a verification pass that
`--describe` reports a committed offset on every partition of every
subscribed topic, and fail the Job if it does not — so Argo CD's health check
carries the signal.

- [ ] **Step 5: Run the test**

Run: `BOOTSTRAP_SERVERS=<broker> ./deploy/k8s/base/atlas-kafka-precreate_test.sh`
Expected: PASS. If no broker is reachable in the development environment, say
so in the task report and mark this as verified only at deploy time — do not
claim a pass the test did not produce.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/base docs/runbooks/sparse-environments.md
git commit -m "feat(deploy): seed override consumer-group offsets at sync-wave 0"
```

---

### Task 46: Confirm and wire per-environment socket port and IP allocation

PRD open question 8. The hypothesis from reading
`deploy/k8s/overlays/pr/patches/lb-allocate.yaml` is that this is **already
solved**: each PR namespace's `atlas-login-lb` and `atlas-channel-lb` request
a dynamic IP from MetalLB's `pr-pool` (192.168.23.210-.229, `autoAssign:
false`, opt-in by annotation), and ports are version-derived and therefore
identical across environments — different IP, same port. This task confirms
that against reality and wires the confirmed answer, rather than assuming it.

**Files:**
- Modify: `deploy/k8s/overlays/pr-sparse/kustomization.yaml` (Task 44) — keep `patches/lb-allocate.yaml`
- Modify: `docs/runbooks/sparse-environments.md` — the pool-capacity note
- Modify: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` §2 — fill in the ports/IP disposition row
- Read-only: `deploy/k8s/overlays/pr/patches/lb-allocate.yaml`
- Read-only: `services/atlas-pr-bootstrap/scripts/version-ports.sh` — `derive_login_port` / `derive_channel_port`
- Read-only: `tools/gen-lb-ports.sh` — the version→port formula and its duplicate-major guard

- [ ] **Step 1: Confirm the pool has capacity for the intended concurrency**

The pool is 20 addresses and each sparse environment needs **two** (login and
channel), so the ceiling is 10 concurrent environments — which is exactly the
metric-cardinality budget stated in Task 29. Confirm the pool range from the
cluster's MetalLB configuration rather than from the comment:

```sh
kubectl -n metallb-system get ipaddresspool -o yaml
```

Record the actual range. If it differs from the comment, the comment is wrong
and gets fixed here.

- [ ] **Step 2: Confirm ports do not need to vary per environment**

Read `version-ports.sh`. Ports are derived from `majorVersion` alone. Two
environments on the same version therefore bind the **same port on different
IPs**, which is correct for a LoadBalancer Service and is why D6's "each
sparse environment needs its own port/IP allocation" resolves to "its own IP,
the shared port formula".

Write that conclusion into the audit row. If reading the script contradicts
it, the contradiction is the finding — report it and stop, because the sparse
floor's addressability depends on the answer.

- [ ] **Step 3: Confirm the sparse overlay keeps the patch**

Run: `grep -n "lb-allocate" deploy/k8s/overlays/pr-sparse/kustomization.yaml`
Expected: present. If Task 44 dropped it, add it back.

- [ ] **Step 4: Add the pool-exhaustion failure mode to the runbook**

`bootstrap.sh:296` already fails loudly when
`atlas-channel-lb` has no allocated IP ("MetalLB pool exhausted?"). Document
the symptom, the `kubectl get svc -A | grep pending` diagnostic, and the
remedy (tear down an idle environment, or widen the pool).

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/overlays/pr-sparse docs/runbooks/sparse-environments.md docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md
git commit -m "docs(deploy): confirm per-environment socket IP allocation; document the pool ceiling"
```

---

### Task 47: Bootstrap creates its own service rows instead of merging into `main`'s

**The single most dangerous piece of existing code in this migration**
(design C2, risk table row 1). `upsert_service_config` reads the live row for
a canonical **pinned** `SERVICE_ID`, merges this PR's tenant entry into its
`tenants[]` array, and writes the merged result back. That is safe today
because each PR has its own `atlas-configurations` database. Under D1 it
would merge the PR's tenant and port into **`main`'s** service row — the
NG6/G7 violation D6 exists to prevent, arriving through the bootstrap script
rather than through service code. `tools/verify.sh` cannot see it.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/bootstrap.sh:303-320` — `upsert_service_config` becomes `create_service_config` in sparse mode
- Modify: `services/atlas-pr-bootstrap/scripts/service-config.sh` — `merge_tenant_entry` gains a sparse-mode sibling that builds a fresh row
- Modify: `services/atlas-pr-bootstrap/test/service_config_test.bats` — the negative test
- Modify: `services/atlas-pr-bootstrap/canonical/` — the canonical payloads keep their pinned ids for isolated mode; sparse generates fresh UUIDs
- Modify: `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — the sparse `SERVICE_ID` values come from the generated row ids
- Read-only: `services/atlas-configurations/atlas.com/configurations/services/entity.go` — the table is `Id`-keyed; multiple rows of the same `Type` are already representable (design C1)

**Interfaces:**
- Produces: in sparse mode, a **new** `services` row per service type with a
  fresh UUID and `environment = <this environment>`; `main`'s row is never
  read, written or merged into. Isolated mode keeps the pinned-id merge
  unchanged.

- [ ] **Step 1: Write the failing negative test**

`services/atlas-pr-bootstrap/test/service_config_test.bats`:

```bash
@test "sparse mode never reads or writes the pinned main service row" {
    export ATLAS_MODE=sparse
    export TENANT_ID="$(uuidgen)"
    export MAJOR_VERSION=83
    export LB_IP=192.168.23.211

    run build_service_config login "$CANONICAL/login.json"
    [ "$status" -eq 0 ]

    # A fresh id, NOT the canonical pinned one.
    pinned=$(jq -r '.data.id' "$CANONICAL/login.json")
    got=$(echo "$output" | jq -r '.data.id')
    [ "$got" != "$pinned" ]
    [ "$got" != "null" ]

    # Exactly this environment's one tenant — never a merge of main's list.
    [ "$(echo "$output" | jq '.data.attributes.tenants | length')" -eq 1 ]
    [ "$(echo "$output" | jq -r '.data.attributes.tenants[0].id')" = "$TENANT_ID" ]

    # The environment is stamped so teardown and write-authorisation can scope it.
    [ "$(echo "$output" | jq -r '.data.attributes.environment')" = "$ATLAS_ENVIRONMENT" ]
}

@test "isolated mode still merges into the pinned row" {
    export ATLAS_MODE=isolated
    run build_service_config login "$CANONICAL/login.json"
    [ "$status" -eq 0 ]
    pinned=$(jq -r '.data.id' "$CANONICAL/login.json")
    [ "$(echo "$output" | jq -r '.data.id')" = "$pinned" ]
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-pr-bootstrap && bats test/service_config_test.bats`
Expected: FAIL — `build_service_config: command not found`.

- [ ] **Step 3: Implement `build_service_config`**

```sh
# Build the service-config payload for this environment.
#
# ATLAS_MODE=sparse: a NEW row with a fresh UUID carrying exactly this
# environment's single tenant entry. main's row is never read, written or
# merged into (G7, NG6). The services table is Id-keyed and consumers select
# by SERVICE_ID, so multiple rows of one Type are already representable
# (design C1) — no key change is needed to make this work.
#
# ATLAS_MODE=isolated: unchanged. The pinned id is merged into as before,
# which is safe because an isolated environment owns its own database.
build_service_config() {
    shape="$1"; tmpl="$2"
    case "$shape" in
        login)   entry=$(build_login_entry) ;;
        channel) entry=$(build_channel_entry "$tmpl") ;;
        none)    entry="" ;;
        *)       log error "build_service_config: unknown shape '$shape'"; return 1 ;;
    esac

    if [ "${ATLAS_MODE:-isolated}" = "sparse" ]; then
        jq -c --arg id "$(uuidgen)" --arg envn "$ATLAS_ENVIRONMENT" --argjson entry "${entry:-null}" '
            .data.id = $id
            | .data.attributes.environment = $envn
            | if $entry == null then . else .data.attributes.tenants = [$entry] end
        ' "$tmpl"
    else
        jq -c '.data.attributes' "$tmpl" | merge_tenant_entry "$entry" | rewrap_attributes "$tmpl"
    fi
}
```

Read the existing `upsert_service_config` body before writing
`rewrap_attributes` — the isolated branch must reproduce today's behaviour
exactly, including the GET-merge-PATCH network calls, which belong outside
this pure function.

- [ ] **Step 4: Route the sparse `SERVICE_ID` values back into the overlay**

The generated row ids must reach the override Deployments' `SERVICE_ID` env
vars. The bootstrap Job writes them into a ConfigMap the Deployments read, or
patches the Deployments directly — choose whichever fits the existing
sync-wave ordering and state the choice in the task report. A hardcoded
`SERVICE_ID` in the sparse overlay would reintroduce the pinned-row problem
in a different place.

- [ ] **Step 5: Run the bats suite**

Run: `cd services/atlas-pr-bootstrap && bats test/`
Expected: PASS, both new tests and every pre-existing one.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pr-bootstrap deploy/k8s/overlays/pr-sparse
git commit -m "fix(pr-bootstrap): sparse mode creates its own service rows instead of merging into main's

Under a shared control-plane database, upsert_service_config would have
merged the PR's tenant and port into main's service row — mutating main to
serve a PR (G7/NG6). Sparse mode now creates a fresh Id-keyed row per
environment; isolated mode is unchanged."
```

---

### Task 48: Teardown — deactivate before destroy, and reclaim the control plane

FR-5.5 (routing stops before any override workload is destroyed) and FR-5.7
(no surviving delayed work executes against the baseline).

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh:240` — the `PHASES` list gains `deactivate` (first) and `drop-control-plane`
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh` — the two new phase functions
- Modify: `services/atlas-pr-bootstrap/test/` — bats coverage for both
- Modify: `docs/runbooks/ephemeral-pr-deployments.md` §9.4 — the new phases in the `failed_phases` reporting
- Read-only: `services/atlas-pr-bootstrap/scripts/cleanup.sh:238-250` — the `PHASES` orchestration and `run_phase` / `summarize_phases` contract
- Read-only: `libs/atlas-env/registry.go` — `PhaseDeactivating`, `PhaseDeleted`

**Interfaces:**
- Produces: `do_deactivate` — PATCHes the environment record to
  `DEACTIVATING`, waits for the staleness bound to elapse so every pod has
  projected it, then PATCHes to `DELETED`. Runs **first**, before any
  destructive phase.
- Produces: `do_drop_control_plane` — deletes this environment's
  `atlas-configurations` `services` / `tenants` / `templates` rows and its
  `atlas-tenants` row.

**Drain semantics (design §7.4) — work is discarded, never handed back:**

| Resource | Behaviour |
|---|---|
| In-flight REST | The environment's own ingress stops routing. Established connections complete under nginx's existing `proxy_read_timeout 30` (`atlas-ingress.yaml:22`). Bounded, no handover. |
| Unacknowledged Kafka | Override pods terminate; the override group is deleted. Uncommitted messages are never consumed by anyone: baseline pods see an environment no longer in the registry and ack-and-drop (FR-4.7). |
| Open game sockets | Override login/channel pods terminate; clients disconnect. No baseline handover is possible — the tenant is gone. |
| Scheduled / persisted work | Orphaned rows are inert: a saga row whose tenant no longer resolves to a known environment is never reached by the sweeper. Storage waste, not a correctness defect. Task 49 reclaims it. |

FR-5.7 is satisfied **by the gate's drop path**, not by draining. Teardown
does not have to be complete to be correct.

- [ ] **Step 1: Write the failing phase-ordering test**

```bash
@test "deactivate runs before every destructive phase" {
    run grep -n 'deactivate\|do_drop_dbs\|do_drop_topics\|do_drop_groups\|do_drop_redis' \
        "$SCRIPTS/cleanup.sh"
    [ "$status" -eq 0 ]
    deact=$(echo "$output" | grep -n 'deactivate  *do_deactivate' | cut -d: -f1)
    first_destructive=$(echo "$output" | grep -n 'do_drop_' | head -1 | cut -d: -f1)
    [ "$deact" -lt "$first_destructive" ]
}

@test "drop-control-plane deletes only this environment's rows" {
    run do_drop_control_plane --dry-run
    [ "$status" -eq 0 ]
    # Every emitted request must carry this environment; none may be unscoped.
    ! echo "$output" | grep -qv "environment=$ATLAS_ENVIRONMENT"
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-pr-bootstrap && bats test/cleanup_test.bats`
Expected: FAIL.

- [ ] **Step 3: Implement `do_deactivate`**

```sh
# Stop new work reaching this environment BEFORE anything is destroyed
# (FR-5.5). Two steps, because every pod's registry cache updates
# independently: DEACTIVATING stops the ingress routing and the gate
# accepting; after StaleAfter has elapsed every pod has projected it, and
# DELETED removes the record entirely.
do_deactivate() {
    curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -d "{\"data\":{\"type\":\"environments\",\"id\":\"$ATLAS_ENVIRONMENT\",\"attributes\":{\"phase\":\"DEACTIVATING\"}}}" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null || return 1
    sleep "${ATLAS_DEACTIVATE_SETTLE_S:-35}"
    curl -fsS -X PATCH \
        -H 'Content-Type: application/vnd.api+json' \
        -d "{\"data\":{\"type\":\"environments\",\"id\":\"$ATLAS_ENVIRONMENT\",\"attributes\":{\"phase\":\"DELETED\"}}}" \
        "$ATLAS_UI_BASE/api/configurations/environments/$ATLAS_ENVIRONMENT" >/dev/null
}
```

The 35 s settle is one heartbeat interval plus margin, not the 120 s
staleness bound: the projection is push-based, so a phase change propagates
in the time it takes one Kafka message to reach every consumer. State that
reasoning in a comment. If measurement during Task 55 shows it is
insufficient, raise it there rather than guessing higher now.

- [ ] **Step 4: Implement `do_drop_control_plane`**

DELETEs this environment's rows through the REST API, scoped by
`environment` — never by a bare table sweep. The write-authorisation from
Task 13 makes a cross-environment delete impossible; assert that with a
negative bats case.

- [ ] **Step 5: Add both phases to `PHASES`**

```sh
PHASES=(
    deactivate           do_deactivate          # FR-5.5 — must be first
    drop-control-plane   do_drop_control_plane
    drop-dbs             do_drop_dbs            # no-op in sparse mode
    drop-topics          do_drop_topics         # no-op in sparse mode
    drop-groups          do_drop_groups
    drop-redis           do_drop_redis          # no-op in sparse mode
    drop-images          do_drop_images
    drop-dns             do_drop_dns
    drop-branch          do_drop_branch
)
```

`drop-dbs`, `drop-topics` and `drop-redis` remain in the list: they are
no-ops in sparse mode and load-bearing in isolated mode. Make each detect the
mode and log "skipped (sparse)" rather than silently succeeding — a phase
that reports success without doing anything is indistinguishable from one
that failed to find its target.

- [ ] **Step 6: Run the bats suite**

Run: `cd services/atlas-pr-bootstrap && bats test/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-pr-bootstrap docs/runbooks/ephemeral-pr-deployments.md
git commit -m "feat(pr-bootstrap): deactivate before destroy; reclaim control-plane rows"
```

---

### Task 49: The tenant-keyed orphan sweeper

Design §7.5, honestly stated: today teardown drops the PR's databases and
everything the environment ever wrote vanishes with them. Under D1 the
databases are shared, so teardown becomes "delete every row belonging to the
ephemeral tenant across ~85 shared databases". This is the largest cost the
design pays for D1's benefits.

It is tractable because **correctness does not depend on teardown
completeness** — orphaned rows are inert, skipped by every sweeper under the
unknown-environment rule — so this is storage reclamation, not a correctness
gate.

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` — add the tenant-keyed pass
- Modify: `services/atlas-pr-bootstrap/test/` — bats coverage
- Modify: `docs/runbooks/sparse-environments.md` — how to run it and what it does not guarantee
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` — the per-service entity list is the sweeper's driver
- Read-only: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` — the existing structure

**Interfaces:**
- Produces: `sweep_tenant <tenant-uuid>` — one generic "delete rows for tenant
  T" pass driven from the audit's entity list, run against the shared
  databases. **One mechanism, not 85 per-service cleanup scripts.**

- [ ] **Step 1: Write the failing sweeper test**

```bash
@test "sweep_tenant deletes only the named tenant's rows" {
    seed_row characters "$TENANT_A" 1
    seed_row characters "$TENANT_B" 2

    run sweep_tenant "$TENANT_A"
    [ "$status" -eq 0 ]

    [ "$(count_rows characters "$TENANT_A")" -eq 0 ]
    [ "$(count_rows characters "$TENANT_B")" -eq 1 ]
}

@test "sweep_tenant refuses an empty tenant id" {
    run sweep_tenant ""
    [ "$status" -ne 0 ]
}
```

The second test is not defensive padding: `DELETE FROM characters WHERE
tenant_id = ''` against a shared database is the failure mode that makes this
script dangerous.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-pr-bootstrap && bats test/sweep_orphans_test.bats`
Expected: FAIL.

- [ ] **Step 3: Implement `sweep_tenant`**

Driven by a table list generated from the audit:

```sh
# Delete every data-plane row belonging to one tenant, across the shared
# databases. Driven from the entity list the FR-8.1 audit produced — one
# mechanism, not 85 per-service scripts.
#
# Correctness does NOT depend on this completing: an orphaned row whose
# tenant no longer resolves to a known environment is never reached by any
# sweeper or loop. This reclaims storage.
sweep_tenant() {
    tenant="$1"
    [ -n "$tenant" ] || { log error "sweep_tenant: refusing an empty tenant id"; return 1; }
    while read -r db table; do
        [ -n "$db" ] || continue
        psql_exec "$db" "DELETE FROM \"$table\" WHERE tenant_id = '$tenant'"
    done < "$TENANT_TABLES"
}
```

`$TENANT_TABLES` is a checked-in `<database> <table>` list generated from the
audit. Generate it, do not hand-write it, and add a CI check that it matches
the audit — a table missing from the list is silently never reclaimed.

- [ ] **Step 4: Wire it into teardown**

Add `sweep-tenant` to `cleanup.sh`'s `PHASES` after `drop-control-plane`,
sparse-mode only. Its failure must not fail the teardown — record it in
`failed_phases` and continue, because it is reclamation, not correctness.

- [ ] **Step 5: Run the bats suite**

Run: `cd services/atlas-pr-bootstrap && bats test/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-pr-bootstrap docs/runbooks/sparse-environments.md
git commit -m "feat(pr-bootstrap): tenant-keyed orphan sweeper for shared-database teardown"
```

---

# Phase E — Mode selection

---

### Task 50: Affected-service determination and automatic escalation

FR-9.2/9.3. `tools/cideps` already computes the affected-module set from a
changed-file list **including transitive module dependencies** — it is what
`.github/workflows/pr-validation.yml`'s `detect-changes` job feeds every
downstream matrix from. Affected-service determination is that same
computation, not a new one.

**Files:**
- Create: `tools/mode-select.sh` — the mode decision and the override-set computation
- Create: `tools/mode-select_test.sh` — the decision table as a test
- Modify: `.github/actions/detect-changes/action.yml` — emit `mode` and `override-services`
- Modify: `.github/workflows/pr-validation.yml` — the `update-pr-overlay` job branches on `mode`
- Read-only: `tools/cideps/` — `select.go` and `graph.go`, the affected-module computation
- Read-only: `.github/workflows/pr-validation.yml:46-75` — the `detect-changes` job's outputs
- Read-only: `docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit.md` §2 — any resource dispositioned `forces isolated mode`

**Interfaces:**
- Produces: `tools/mode-select.sh <changed-file-list> → {sparse|isolated} <override service list>`.
- Produces: `detect-changes` outputs `mode` and `override-services`.

**The escalation rules (FR-9.3), evaluated over the changed-file list before
the override set is computed:**

| Trigger | Path prefix |
|---|---|
| Deployment or ingress configuration | `deploy/k8s/base/`, `deploy/shared/routes.conf`, `tools/gen-routes.sh` |
| Kafka infrastructure or message contracts | `libs/atlas-kafka/`, `services/*/kafka/message/` |
| Database migrations | any `entity.go`, any `migration*.go` |
| The control plane itself | `services/atlas-configurations/`, `services/atlas-tenants/` |
| Cross-cutting libraries | `libs/atlas-kafka/`, `libs/atlas-rest/`, `libs/atlas-tenant/`, `libs/atlas-redis/`, `libs/atlas-env/`, `libs/atlas-service/` |
| Affected-service determination unreliable | `cideps` returns an error, or any changed path matches none of its known roots |
| Anything dispositioned `forces isolated mode` | from the audit §2 |

**How often this fires (design §13):** measured over the 2492 commits between
2026-02-15 and 2026-08-15 on `main`, 127 (5.6 %) touched
`atlas-configurations`/`atlas-tenants` and 85 (3.7 %) touched the four
cross-cutting libraries — roughly one commit in eleven, and the true per-PR
rate is higher because a PR escalates if *any* of its commits qualifies. The
answer to PRD open question 12 is therefore: no, the shared control plane
does not defeat the purpose, but isolated mode must stay a first-class,
tested, scheduled path (FR-9.6), not a vestigial fallback.

- [ ] **Step 1: Write the failing decision-table test**

```sh
#!/usr/bin/env sh
# tools/mode-select_test.sh
set -eu
fail() { echo "FAIL: $*" >&2; exit 1; }
check() { # <expected-mode> <changed files...>
    want="$1"; shift
    got=$(printf '%s\n' "$@" | ./tools/mode-select.sh | head -1)
    [ "$got" = "$want" ] || fail "files [$*]: mode=$got, want=$want"
}

check sparse   services/atlas-monsters/atlas.com/monsters/monster/processor.go
check isolated deploy/k8s/base/atlas-monsters.yaml
check isolated libs/atlas-kafka/consumer/manager.go
check isolated libs/atlas-rest/requests/url.go
check isolated services/atlas-configurations/atlas.com/configurations/main.go
check isolated services/atlas-tenants/atlas.com/tenants/main.go
check isolated services/atlas-monsters/atlas.com/monsters/monster/entity.go
check isolated some/unknown/path.txt
check sparse   docs/tasks/task-232-sparse-ephemeral-environments/plan.md

# The override set ALWAYS includes the mandatory floor (FR-9.4, D6).
set -- services/atlas-monsters/atlas.com/monsters/monster/processor.go
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | tail -1)
for required in atlas-login atlas-channel atlas-monsters; do
    echo "$overrides" | tr ' ' '\n' | grep -qx "$required" \
        || fail "override set [$overrides] is missing $required"
done

echo "PASS"
```

The `docs/` case matters: a docs-only PR is the worst case for sparse mode
and must produce **two** workloads (the floor), not zero and not 64.

- [ ] **Step 2: Run to verify it fails**

Run: `./tools/mode-select_test.sh`
Expected: FAIL — `mode-select.sh: not found`.

- [ ] **Step 3: Implement `mode-select.sh`**

Reads the changed-file list on stdin. First applies the escalation table
above — **conservatively**: an unmatched path escalates rather than being
ignored. Then, for sparse, calls `tools/cideps` for the affected-service set
and unions in `atlas-login` and `atlas-channel` (FR-9.4). Prints the mode on
line 1 and the space-separated override set on line 2.

- [ ] **Step 4: Emit the outputs from `detect-changes`**

Add `mode` and `override-services` to the composite action's outputs,
computed by `mode-select.sh` from the same changed-file list the existing
outputs use. Do not re-derive the file list.

- [ ] **Step 5: Branch the overlay job on the mode**

In `pr-validation.yml`'s `update-pr-overlay` job, select
`deploy/k8s/overlays/pr-sparse` or `deploy/k8s/overlays/pr` on `mode`, and
substitute `PLACEHOLDER_DELETE_BLOCK` / `PLACEHOLDER_NS_OVERRIDES` from
`override-services` for the sparse path. Keep the existing `sed`-and-
force-push-to-`bot/pr-<N>-resolved` mechanism unchanged — this adds a branch,
not a new delivery model.

- [ ] **Step 6: Run the test**

Run: `./tools/mode-select_test.sh`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/mode-select.sh tools/mode-select_test.sh .github
git commit -m "feat(ci): affected-service determination and automatic escalation to isolated mode"
```

---

### Task 51: The per-PR label override and the mode report

FR-9.5 (overridable per PR by an explicit label, **in both directions**) and
FR-9.7 (the chosen mode, the override set, and the reason are reported on the
PR).

**Files:**
- Modify: `tools/mode-select.sh` — honour `ATLAS_FORCE_MODE`
- Modify: `tools/mode-select_test.sh` — the override cases
- Modify: `.github/workflows/pr-validation.yml` — read the labels; post the report comment
- Modify: `docs/runbooks/ephemeral-pr-deployments.md` — document both labels
- Read-only: `.github/workflows/pr-validation.yml:30-40` — the `labeled`-event concurrency lane, which already exists and must keep working

**Interfaces:**
- Produces: labels `atlas:sparse` and `atlas:isolated`, each forcing its mode.
  Both present is an error, not a precedence rule — a PR that asks for both is
  a mistake, and picking one silently hides it.
- Produces: a single PR comment, updated in place, naming the mode, the
  override set, and the reason.

- [ ] **Step 1: Write the failing override tests**

```sh
check_forced() { # <env-value> <expected-mode> <changed file>
    got=$(printf '%s\n' "$3" | ATLAS_FORCE_MODE="$1" ./tools/mode-select.sh | head -1)
    [ "$got" = "$2" ] || fail "forced=$1 file=$3: mode=$got, want=$2"
}

check_forced sparse   sparse   libs/atlas-kafka/consumer/manager.go  # down-force
check_forced isolated isolated services/atlas-monsters/atlas.com/monsters/monster/processor.go  # up-force

# Both labels at once is an error, not a precedence rule.
printf 'x\n' | ATLAS_FORCE_MODE="sparse isolated" ./tools/mode-select.sh \
    && fail "conflicting force labels accepted"
```

- [ ] **Step 2: Run to verify it fails**

Run: `./tools/mode-select_test.sh`
Expected: FAIL.

- [ ] **Step 3: Implement the override**

```sh
case "${ATLAS_FORCE_MODE:-}" in
    "")                 : ;;                      # no label; compute normally
    sparse)   MODE=sparse;   REASON="forced by the atlas:sparse label" ;;
    isolated) MODE=isolated; REASON="forced by the atlas:isolated label" ;;
    *) echo "mode-select: conflicting or unknown ATLAS_FORCE_MODE '$ATLAS_FORCE_MODE'" >&2; exit 2 ;;
esac
```

`ATLAS_FORCE_MODE=sparse` on a PR that would otherwise escalate is a
deliberate, documented risk: the PR author is asserting the change is safe to
validate against the shared control plane. The report comment must say so
explicitly.

- [ ] **Step 4: Post the report comment**

```markdown
### Ephemeral environment: **sparse**

| | |
|---|---|
| Mode | `sparse` |
| Reason | affected services: `atlas-monsters` |
| Workloads deployed | 3 of 64 |
| Override set | `atlas-login`, `atlas-channel`, `atlas-monsters` |
| Everything else | served by `main` |

Change the mode with the `atlas:isolated` label.
```

Update the existing comment in place rather than appending a new one per
push; find it by a hidden marker comment.

- [ ] **Step 5: Confirm the scheduled full-stack run still exists (FR-9.6)**

Run: `grep -n "schedule:" -A 5 .github/workflows/*.yml`
Expected: the scheduled isolated deployment is present and untouched. If it
is not there, that is a finding — report it; do not add one silently.

- [ ] **Step 6: Run the test**

Run: `./tools/mode-select_test.sh`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/mode-select.sh tools/mode-select_test.sh .github docs/runbooks/ephemeral-pr-deployments.md
git commit -m "feat(ci): per-PR mode override labels and the mode report comment"
```

---

# Phase F — Enable and prove

---

### Task 52: Turn on `env-bootstrap-guard`

Design §12: the gate on enabling sparse mode is not a code freeze but this
guard — sparse mode is not selectable until every service wires the registry,
and CI proves it.

**Files:**
- Modify: `tools/verify.sh` — add the `env-bootstrap-guard` step
- Modify: `tools/envguard/bootstrap-allowlist.txt` — final content, every entry with a reason
- Read-only: `tools/env-bootstrap-guard.sh` (Task 28)

- [ ] **Step 1: Run the guard and confirm it reaches zero**

Run: `./tools/env-bootstrap-guard.sh`
Expected: exit 0. Compare against the count Task 28 recorded — every service
it listed must now be either wired or allowlisted with a reason. Paste both
numbers into the task report; "it passes now" without the before-count is not
evidence that the fleet was covered.

- [ ] **Step 2: Wire it into `verify.sh`**

```sh
if touched '^services/.*/main\.go$|^tools/envguard/'; then
    step "env bootstrap guard" ./tools/env-bootstrap-guard.sh
else
    skip "env bootstrap guard (no service main.go changed)"
fi
```

- [ ] **Step 3: Run the flagless verification gate**

Run: `tools/verify.sh`
Expected: exit 0. Flagless — `--quick` skips the bake, and Phase B added a new
`libs/atlas-env` dependency to `atlas-rest`, `atlas-kafka`, `atlas-redis` and
`atlas-service`, which every service Dockerfile must `COPY`. That is exactly
the class of failure `go build` against `go.work` cannot catch.

If the bake fails on a missing `COPY libs/atlas-env`, fix the shared
Dockerfile here — it is a prerequisite this task can produce itself.

- [ ] **Step 4: Commit**

```bash
git add tools/verify.sh tools/envguard
git commit -m "feat(tools): enforce env-bootstrap-guard — every service wires the registry"
```

---

### Task 53: Measure the Kafka fan-out cost before enabling sparse by default

The ownership gate is logically sound but it is **not free**. Sparse
environments consume the unsuffixed baseline topics (FR-4.8), so every
override consumer group reads the service's *entire* topic and gate-drops the
traffic belonging to other environments. With ten concurrent environments
overriding `atlas-character`, there are eleven consumer groups
(`atlas-character-main` plus ten `atlas-character-pr-N`) each seeing
essentially all character messages.

The trade is:

```
before:  60 pods × environment
after:   topic traffic × (number of overrides of that service)
```

That is very likely still a large win — a dropped message costs a header
parse and two map lookups, not a domain handler — but "very likely" is not a
measurement, and the cost is superlinear in the one dimension the project is
designed to increase. This task measures it before Task 54 makes sparse the
default. **If the numbers are bad, the remedy is a design change** (per-service
topic partitioning by environment, or reinstating topic suffixing for
high-volume services in sparse mode), which is far cheaper to discover here
than after the flip.

**Files:**
- Create: `docs/tasks/task-232-sparse-ephemeral-environments/kafka-fanout-measurement.md` — the measured result
- Modify: `docs/runbooks/sparse-environments.md` — the concurrency ceiling this establishes, and the symptom to watch for
- Read-only: `libs/atlas-kafka/consumer/gate.go` (Task 26) — the three counters that make the drop rate observable
- Read-only: `libs/atlas-kafka/consumer/manager.go` — `Snapshot()` / the `/debug/consumers` handler, for per-consumer lag

**Interfaces:**
- Consumes: a live cluster with the Phase D deployment machinery working.
- Produces: `kafka-fanout-measurement.md` with four numbers per configuration
  and a stated verdict. Task 54 Step 1 cites it.

- [ ] **Step 1: Pick the highest-volume service and establish its baseline**

Identify the service with the highest message rate on `main`, from the
existing Kafka metrics rather than by assumption:

```promql
topk(5, sum by (topic) (rate(kafka_server_brokertopicmetrics_messagesin_total[15m])))
```

Record, for `main` alone: broker ingress bytes/s for that topic, the
consumer group's CPU (`rate(container_cpu_usage_seconds_total[5m])` for the
pod), resident memory, and consumer lag. These four are the baseline row.

- [ ] **Step 2: Deploy 5 concurrent sparse environments overriding that one service**

Five PRs, each overriding only the chosen service. All five plus `main`
consume the same unsuffixed topic. Re-record the same four numbers, per
group and in total, after a 15-minute soak under representative load.

- [ ] **Step 3: Scale to 10 and re-record**

Ten is the MetalLB `pr-pool` ceiling established in Task 46 (20 addresses,
two per environment), so it is the real upper bound on concurrency — measure
at the ceiling, not at a comfortable point below it.

- [ ] **Step 4: Record the drop ratio**

```
atlas_kafka_gate_skipped_not_owner_total / atlas_kafka_gate_processed_total
```

per group. With N environments and traffic concentrated in `main`, an
override group's ratio should approach `(N-1)/1` — i.e. it discards almost
everything it reads. Confirm the observed ratio matches that prediction; a
large divergence means the gate is not seeing the traffic you think it is,
and the measurement is invalid.

- [ ] **Step 5: Write the verdict**

`kafka-fanout-measurement.md`:

```markdown
# Kafka fan-out under concurrent sparse environments

Measured <date>, on <cluster>, against `<service>` / `<topic>`.

| Environments | Broker egress (MB/s) | Consumer CPU (cores, total) | Peak lag (msgs) | Drop ratio |
|---|---|---|---|---|
| 1 (main only, baseline) | | | | n/a |
| 6 (main + 5 overrides) | | | | |
| 11 (main + 10 overrides) | | | | |

**Verdict:** <acceptable | needs mitigation>

**Threshold used:** <state it — e.g. total consumer CPU across all groups
must stay under X cores, and no group's lag may exceed Y at steady state>

**If mitigation is needed**, the options in preference order are:
1. Restrict sparse mode for this specific service — escalate PRs touching it
   to isolated (a one-line addition to Task 50's escalation table).
2. Reinstate topic suffixing for the identified high-volume topics in sparse
   mode, accepting per-environment topics for those alone.
3. Partition the topic by environment and assign partitions by ownership.
```

State the threshold **before** reading the numbers, and record it in the
document. A threshold chosen after seeing the result is not a threshold.

- [ ] **Step 6: If the verdict is "needs mitigation", stop and raise it**

Do not proceed to Task 54. Mitigation option 1 is a plan edit this task can
make itself (add the service's path to Task 50's escalation table); options 2
and 3 are design changes and need sign-off. Report which applies.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/kafka-fanout-measurement.md \
        docs/runbooks/sparse-environments.md
git commit -m "docs(task-232): measured Kafka fan-out cost under concurrent sparse environments"
```

---

### Task 54: Make sparse the default mode

**Files:**
- Modify: `tools/mode-select.sh` — the default when no escalation trigger fires becomes `sparse`
- Modify: `docs/runbooks/sparse-environments.md` — the "how to escalate" section becomes the primary path
- Modify: `docs/runbooks/ephemeral-pr-deployments.md` — note that sparse is now the default and isolated is the documented escalation
- Read-only: `tools/mode-select_test.sh` — already asserts `sparse` for a plain service change

- [ ] **Step 1: Confirm every precondition**

Before flipping, each of these must hold. Check them, do not assume them:

```sh
./tools/scope-guard.sh            # Task 15
./tools/redis-key-guard.sh        # Task 9
./tools/env-domain-guard.sh       # Task 28
./tools/env-bootstrap-guard.sh    # Task 52
./tools/producer-seam-guard.sh    # Task 24
./tools/gen-routes.sh --check     # Task 43
./tools/mode-select_test.sh       # Tasks 50-51
```

Expected: all exit 0. Paste the outputs into the task report.

Also confirm Task 53's verdict is **acceptable**. Quote the table from
`kafka-fanout-measurement.md` here — a fan-out cost that has not been
measured is not a reason to flip the default, and the cost grows with exactly
the concurrency this change is meant to encourage.

Also re-check the NetworkPolicy risk (design §16): cross-namespace ingress
calls are what M3 depends on, and a future NetworkPolicy would break it
silently.

```sh
grep -rn NetworkPolicy deploy/k8s/
kubectl get networkpolicy -A
```

Expected: nothing in either. If a NetworkPolicy exists, M3 must be validated
against it before this flip — that is a blocker, not a note.

- [ ] **Step 2: Flip the default**

In `mode-select.sh`, the no-trigger branch sets `MODE=sparse` (it was
`isolated` while the mechanism was being built).

- [ ] **Step 3: Run the mode tests**

Run: `./tools/mode-select_test.sh`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add tools/mode-select.sh docs/runbooks
git commit -m "feat(ci): sparse is now the default ephemeral environment mode"
```

---

### Task 55: The §17 proof — environment is not deployment

The PRD's acceptance criteria, executed against a real cluster. Every item
below is a **measurement**, not an inspection; a claim without pasted output
is not a pass.

**Files:**
- Create: `docs/tasks/task-232-sparse-ephemeral-environments/proof.md` — the executed proof with pasted evidence
- Read-only: everything

**Interfaces:**
- Consumes: a live sparse environment created from a real PR.
- Produces: `proof.md`, the artifact the PRD §12 acceptance checklist is
  signed off against.

- [ ] **Step 1: Deploy a sparse environment overriding one service**

Push a branch changing only `atlas-character`. Wait for the rollout.

```sh
kubectl -n atlas-pr-<N> get deploy -o name | wc -l
kubectl -n atlas-pr-<N> get deploy -o name
```

Expected: **4** — `atlas-ingress`, `atlas-login`, `atlas-channel`,
`atlas-character`. Paste the list. The remaining 60 are not deployed.

- [ ] **Step 2: Prove a client session stays in its environment**

Connect a game client to `pr-<N>`'s login service. Then:

```sh
# every log line for this session, across override AND baseline pods
logcli query '{service_name=~"atlas-.*"} | environment="pr-<N>"' --limit 500
```

Expected: the session's records appear with `environment=pr-<N>` and
`tenant=<ephemeral>` throughout, including records emitted by
`atlas-monsters` and other **baseline** pods. Paste a sample showing at
least one baseline pod's record carrying `pr-<N>`.

- [ ] **Step 3: Prove a request executes through a mixed path and never leaves**

Pick an operation that traverses both an override and a baseline service.
Trace it:

```sh
logcli query '{service_name=~"atlas-.*"} | trace_id="<id>"' --limit 200
```

Expected: every hop carries `environment=pr-<N>`. **Zero** hops carry
`environment=main` or no environment.

- [ ] **Step 4: Prove `main` is unaffected**

Run the equivalent operation in `main` and assert the behaviour and the
records are unchanged. Then:

```sh
kubectl -n atlas-main get deploy -o json \
  | jq -r '.items[] | "\(.metadata.name) \(.metadata.generation) \(.status.observedGeneration)"'
```

Expected: no Deployment's `generation` advanced across the sparse
environment's whole lifecycle (G7, FR-7.8). Capture the values before and
after.

- [ ] **Step 5: Prove autonomous work is originated per environment**

Pick a baseline deployment with a ticker (`atlas-monsters` if it has one;
otherwise `atlas-transports`). Assert:

```sh
logcli query '{service_name="atlas-transports"} | environment="pr-<N>"' --limit 50
logcli query '{service_name="atlas-transports"} | environment="main"'    --limit 50
```

Expected: **both** non-empty — the same baseline pod originates work
separately for `main` and for `pr-<N>`.

Then deploy a second sparse environment that **overrides** `atlas-transports`
and assert:

```sh
logcli query '{service_name="atlas-transports", namespace="atlas-main"} | environment="pr-<M>"' --limit 50
```

Expected: **empty** — a baseline deployment originates no autonomous work for
an environment that overrides its service (FR-6.3).

- [ ] **Step 6: Prove the gate drops nothing in steady state**

```sh
# in every namespace
curl -s <pod>:<port>/metrics | grep atlas_kafka_gate_dropped_unresolvable_total
```

Expected: `0` everywhere. A non-zero value is cross-environment leakage and
blocks sign-off (FR-10.4).

- [ ] **Step 7: Prove teardown leaves nothing executing against `main`**

Delete the PR. After teardown:

```sh
logcli query '{service_name=~"atlas-.*", namespace="atlas-main"} | environment="pr-<N>"' \
  --since 10m --limit 100
```

Expected: empty after the deactivation point, and any messages that *were*
in flight show up as `dropped_unresolvable` increments rather than as
processed work (FR-5.7).

- [ ] **Step 8: Prove concurrent environments cost what NFR-3 says**

Run three concurrent sparse environments. Sum their pod counts and compare to
3 × 64.

Expected: proportional to the override sets, not to 64.

- [ ] **Step 9: Record the startup-latency measurement (NFR-4)**

Time from PR push to `ACTIVE` for a sparse environment and for an isolated
one. The ~60 s baseline restore leaves the critical path when the environment
shares `main`'s data — confirm that with the two numbers rather than
asserting it.

- [ ] **Step 10: Write `proof.md` and commit**

One section per step, each with the command and its pasted output. Any step
that could not be executed is recorded as **not executed** with the reason —
never as passed.

```bash
git add docs/tasks/task-232-sparse-ephemeral-environments/proof.md
git commit -m "docs(task-232): the §17 proof — executed, with evidence"
```

---

## Self-review record

Run at the end of plan authoring, per the writing-plans skill.

**Spec coverage.** Every design section maps to at least one task:
M1 → 17; M2 → 18–20; M3 → 22–23, 43; M4 → 24–26; M5 → 27, 41–42;
C1 → 11 (no key change), 47; C2 → 47; C3 → confirmed in the PRD reading, no
mandatory-floor change needed, recorded in 44/46; C4 → 27, 41–42;
§8 control plane → 11–14; §9 Redis → 4–10; §10 guards → 9, 15, 24, 28, 52;
§11 deviations V1–V4 → carried in the Global Constraints and in Tasks 20, 26,
13, 11 respectively; §12 sequencing → the phase map; §13 mode selection →
50–51; §14 observability → 29; §15 testing → the TDD steps plus 55;
§16 risks → 47 (C2), 1–3 (sweep), 49 (teardown), 20/26 (registry outage), 45
(offset seeding), 54 Step 1 (NetworkPolicy), 52 (migration drift);
§17 open items → decisions P1–P4 in the Global Constraints.

One cost the design does not bound and this plan adds a task for: sparse
environments consume the unsuffixed baseline topics (FR-4.8), so every
override consumer group reads its service's whole topic and gate-drops other
environments' traffic — superlinear in the concurrency the project exists to
increase. **Task 53** measures broker egress, consumer CPU, lag and drop
ratio at 1 / 6 / 11 concurrent environments against a pre-stated threshold,
and gates Task 54.

PRD requirements with no design mechanism and therefore no task: none found.
PRD FR-5.4 is implemented as design's V2 refinement (at most one owner ever;
zero owners only while no operation for that environment can exist) —
Task 18's `IsOwner` returns false for unknown and non-`ACTIVE` environments,
and Task 26's gate drops them, which is that invariant.

**Placeholder scan.** No `TBD`, no "implement later", no "add appropriate
error handling", no "similar to Task N" in place of repeated code. Three
tasks (1–3) are audits whose deliverable is a document; their steps give the
exact commands and the exact table format rather than code, which is the
right shape for an audit.

**Type consistency.** `env.Id`, `env.Key`, `env.Record`, `env.Registry`,
`env.Self()`, `env.Reconcile`, `env.Mismatched`, `service.WithEnvironmentRegistry`,
`service.ForEachOwnedEnvironment`, `service.TenantLister`,
`requests.RootUrlFor`, `requests.EnvHeaderDecorator`, `producer.EnvHeaderDecorator`,
`consumer.EnvHeaderParser`, `scope.Strict`, `scope.AuthorizeWrite`,
`scope.ErrCrossEnvironmentWrite`, and `templates.{OverlaySingle,
OverlayCollection, VisibleById}` are each defined in exactly one task and used
with the same spelling everywhere after it.

Two API shapes deliberately depart from the PRD/design wording, and the
departure is recorded at the point of definition:

- `Registry.EnvironmentNamespace` / `Registry.ServiceNamespace` replace the
  PRD's single `Resolve(environment, service) → deployment`. M3 needs a
  namespace rather than a Deployment name (Deployment names are identical
  across namespaces in Atlas — design §4.4), and the two questions have
  different answers: the outbound REST path wants the environment's own
  ingress namespace, which must never fall back to the baseline.
- The V3 template fallback is expressed as three key-aware helpers in the
  `templates` package rather than one generic GORM scope, because a
  collection read needs an anti-join and an `ORDER BY` only resolves a
  single-key `First()`.




