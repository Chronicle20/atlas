# Isolation Audit — starting inventory

Supporting document for [prd.md](prd.md) FR-8. This is the **starting
inventory**, not the finished audit: it records what was verified during
specification so the audit begins from evidence rather than a blank sweep.

Decision D1 makes this load-bearing. Three substrates currently isolate ephemeral
environments by *deployment*; sparse mode removes all three, leaving `tenant_id`
as the data plane's only boundary and `environment` as the control plane's.

Verified at branch point `c8d44127c`.

---

## 1. The two planes

The audit's first job on any resource is to classify it, because the two planes
scope differently and applying the wrong rule produces incoherent schemas.

| | Data plane | Control plane |
|---|---|---|
| Contains | Gameplay state — characters, monsters, inventory, doors, summons | Tenant registry, socket templates, login/channel/drops service registrations, ingest runs |
| Scoped by | `tenant_id` | `environment` |
| Environment persisted? | No — derived from tenant (FR-7.3/7.4) | Yes |
| Rule of thumb | "belongs to one tenant" | "exists to provision or serve many tenants" |

Adding `tenant_id` to a control-plane table is a category error: the tenant
registry cannot be scoped to a tenant, and a socket template is deliberately
shared by every tenant on a version.

---

## 2. The three deployment-scoped substrates being removed

| Substrate | Current isolation | Source | Sparse-mode disposition |
|---|---|---|---|
| Postgres | `DB_NAME` → `<db>-<ATLAS_ENV>` | `overlays/pr/scripts/gen-db-name-suffix.sh` | Repointed at the baseline: `<db>-<baseline env>` |
| Kafka topics | all `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` → `-<ATLAS_ENV>` | `overlays/pr/scripts/gen-topic-config.sh` | Repointed at the baseline: `-<baseline env>` + ownership gate |
| Redis | prefix `<ATLAS_ENV>:atlas` | `libs/atlas-redis/keys.go:15-24` | Repointed at the baseline via `ATLAS_REDIS_ENV` |

**Corrected 2026-08-20.** These three dispositions originally read "Removed;
shared DB / shared topics / must be inert". "Removed" was read as "let
`deploy/k8s/base`'s unsuffixed defaults stand", and that is not the same thing
as "the baseline's". Every Atlas substrate in the cluster is named for the
environment that owns it — `EVENT_TOPIC_X-main`, `atlas-characters-main`,
`main:atlas:…` — so the unsuffixed name is a fourth, empty namespace rather
than the shared one. Sharing a substrate means *adopting the baseline's name
for it*. See [bug-sparse-baseline-scoping.md](bug-sparse-baseline-scoping.md).

The Redis prefix is computed in a **package-level `var` initializer** from
`os.Getenv("ATLAS_ENV")`, so it cannot vary per operation without restructuring.
All three remain in force for **isolated** mode.

---

## 3. Postgres

**89** `entity.go` files; **84** have a tenant-scoped primary entity.

```sh
find services -name entity.go -print0 | xargs -0 grep -L "TenantId\|TenantID"
```

Note the grep over-reports: `atlas-configurations/tenants/entity.go` matches only
because its `HistoryEntity` carries `TenantId` (line 29); the primary `Entity`
does not.

| Entity | Plane | Current key | Disposition |
|---|---|---|---|
| `atlas-tenants/.../tenant/entity.go` | Control | tenant id | **+ environment** (FR-8.2) |
| `atlas-configurations/.../tenants/entity.go` | Control | `(Id, Region, Major, Minor)` | **+ environment** (FR-8.2) |
| `atlas-configurations/.../templates/entity.go` | Control | `(Region, Major, Minor)` | **+ environment** (FR-8.2) |
| `atlas-configurations/.../services/entity.go` | Control | `Type` | **+ environment**; key becomes `(Type, environment)` (FR-8.2) |
| `atlas-quest/.../quest/medal/entity.go` | Data | `(QuestStatusId, MapId)` | Scoped transitively via a tenant-scoped parent; confirm FK integrity |

The `services` key change is what makes D6 work: `(Type, environment)` lets
`pr-123` hold its own login-service registration without touching `main`'s row.

**Not yet audited.** A `TenantId` *column* does not prove every *query* filters on
it. FR-8.1 requires the query-path sweep, which this inventory does not attempt.

---

## 4. Redis — the bare API is withdrawn (D7)

`libs/atlas-redis` splits into constructors that build keys via
`tenantEntityKey(namespace, t, entityKey)` and constructors that build them via
`namespacedKey(namespace, parts...)`.

**Tenant-scoped by construction:** `NewTenantRegistry`,
`NewTenantCoalescedRegistry`, `NewTenantSet`, `NewTenantKeyedSet`,
`NewTenantKeyedHash`, `NewTenantKeyedSortedSet`, `NewTenantCounter`.

**Not tenant-scoped by construction:** `NewRegistry`, `NewSet`, `NewHash`,
`NewKeyedSet`, `NewKeyedHash`, `NewCoalescedRegistry`, `NewIDGenerator`,
`NewIDGeneratorWithStart`, `NewGlobalIDGenerator`, `NewIndex`, `NewUint32Index`,
`NewLock`, `NewLockWithTTL`, `NewTTLRegistry`.

### 4.1 Why the policy is "mandatory", not "audited"

Four callsites were checked during specification. One is already wrong:

| Callsite | Key | Verdict |
|---|---|---|
| `atlas-monsters` `monster/registry.go:303` | `monsterKey(t tenant.Model, uniqueId uint32)` | Scoped — correct by convention |
| `atlas-storage` `storage/cache.go:31-36` | `strconv(characterId)` → `atlas:npc-context:<id>` | **UNSCOPED — live defect** |
| `atlas-npc-shops` `shops/cache.go:32` | shop `uuid` only | Suspect; uuid may be globally unique but is not tenant-scoped by construction |
| `atlas-doors` `door/registry.go:69-76` | identity `keyFn`; keys built by callers | Unknown — must trace callers |

`atlas-storage`'s `npc-context` cache keys on character id alone. Character IDs
are per-tenant sequences, so two tenants can collide **today** inside multi-tenant
`main`; the `ATLAS_ENV` prefix hides it only across environments. This is a
pre-existing cross-tenant defect, not something sparse mode introduces — but
sparse mode removes the prefix that limits its blast radius.

One in four wrong on the first sample is the argument for D7: the type system
should make the unscoped key unrepresentable rather than leaving it to per-callsite
discipline.

### 4.2 Required migration

| Service | Namespaces seen | Action |
|---|---|---|
| `atlas-monsters` | `monster`, `monster-map`, `monster-cooldown`, `monster-attack-cooldown`, `monster-puppet`, `monster-puppet-field` | Migrate to tenant-scoped API; behavior-preserving |
| `atlas-summons` | `summon`, `summon-map`, `summon-owner` | Migrate; audit keyFns first |
| `atlas-doors` | `door`, `door-field`, `door-owner`, `door-town` | Trace callers, then migrate |
| `atlas-npc-shops` | `npc-shop:consumables` | Migrate |
| `atlas-storage` | `npc-context` | **Fix the defect**, then migrate |
| `atlas-data` | `ingestrun` | Control plane — move to the new environment-scoped API, not the tenant one |

### 4.3 The deliberately global constructors

`NewGlobalIDGenerator`, `NewLock`, `NewLockWithTTL` are global by intent. Each
surviving use needs a stated intent, because "shared across environments" is
either correct (mutual exclusion on a genuinely shared resource) or a
cross-environment stall. Neither can be assumed.

### 4.4 End state

- Data-plane state: tenant-scoped API only.
- Control-plane state: a narrow, explicitly-named environment-scoped API.
- Bare constructors: removed, unexported, or guarded so a new use fails CI
  (FR-8.4/8.5).

---

## 5. Other resources to enumerate (FR-8.6)

Not investigated during specification; listed so the audit does not stop at
Postgres and Redis:

- **Login/channel port and IP allocation** — each sparse environment needs unique
  ports and an advertised `ChannelTenantRestModel.IPAddress`. The PR overlay's
  existing `lb-allocate` patch may already cover this (PRD open question 8).
- MinIO / canonical baseline objects — currently restored per environment.
- Kafka consumer group names — already runtime-resolved
  (`libs/atlas-kafka/consumergroup/resolver.go`); listed for completeness.
- Any in-process cache or registry keyed without tenant.
- Scheduled/delayed work stores, including saga state in
  `atlas-saga-orchestrator`.
- `libs/atlas-object-id` — uniqueness assumptions across environments sharing a
  keyspace.
- `libs/atlas-outbox` — whether rows carry enough context to restore environment
  on replay (PRD input §7.3).

---

## 6. How to finish this audit

1. Classify every resource as data plane or control plane before scoping it (§1).
2. Sweep query paths, not just schemas (FR-8.1) — a `tenant_id` column no `WHERE`
   clause references is not isolation.
3. Land the control-plane environment dimension first (FR-8.2); it is the known
   blocker and it unlocks D6.
4. Fix the `atlas-storage` defect independently — it does not need this project.
5. Migrate the Redis callsites, then withdraw the bare constructors (FR-8.3/8.4).
6. Add the CI guards before migrating services, so the audit cannot regress while
   the migration is in flight (FR-8.5).
7. Record a disposition for every §5 resource: scope it, or declare that it forces
   isolated mode (FR-9.3).

State findings as hypotheses until confirmed against source. A spot-check
presented as a sweep would be a false "verified" in the most costly possible
place — it surfaces as silent data corruption in `main`.
