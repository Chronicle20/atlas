# Isolation Audit — starting inventory

Supporting document for [prd.md](prd.md) FR-8. This is the **starting
inventory**, not the finished audit: it records what was verified during
specification so the audit begins from evidence rather than from a blank sweep.
Design and implementation must complete it.

Decision D1 (shared datastores, tenant-scoped rows) is what makes this
load-bearing. Today three independent substrates isolate ephemeral environments
by *deployment*; sparse mode removes all three and leaves `tenant_id` as the only
boundary. Every row below is a place where that boundary must be shown to hold.

Verified at branch point `c8d44127c`.

---

## 1. The three deployment-scoped substrates being removed

| Substrate | Current isolation | Source | Sparse-mode disposition |
|---|---|---|---|
| Postgres | `DB_NAME` → `<db>-<ATLAS_ENV>` | `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh` | Removed; shared DB, `tenant_id` only |
| Kafka topics | all `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` → `-<ATLAS_ENV>` | `overlays/pr/scripts/gen-topic-config.sh` | Removed; shared topics + ownership gate |
| Redis | prefix `<ATLAS_ENV>:atlas` | `libs/atlas-redis/keys.go:15-24` | Removed; must be inert in sparse mode |

The Redis prefix is computed in a **package-level `var` initializer** from
`os.Getenv("ATLAS_ENV")`. It cannot vary per operation without restructuring, which
is why D1 chose to neutralize it rather than make it dynamic.

All three remain in force for **isolated** mode.

---

## 2. Postgres — entity tenant scoping

**89** `entity.go` files; **85** carry a `TenantId`/`TenantID` field.

```sh
find services -name entity.go -print0 | xargs -0 grep -L "TenantId\|TenantID"
```

| Entity | Finding | Disposition |
|---|---|---|
| `atlas-tenants/.../tenant/entity.go` | It is the tenant table itself | No action |
| `atlas-quest/.../quest/medal/entity.go` | Keyed `(QuestStatusId, MapId)`; scoped transitively through a tenant-scoped parent | Confirm FK integrity; no schema change expected |
| `atlas-configurations/.../templates/entity.go` | Keyed `(Region, MajorVersion, MinorVersion)` — **no tenant column** | **Blocking.** Schema change + migration (FR-8.3) |
| `atlas-configurations/.../services/entity.go` | Keyed `Type` only — **fully global** | **Blocking.** Schema change + migration (FR-8.3) |

> A PR editing a socket template or a service configuration would, under D1,
> mutate the exact row `main` reads. This is the clearest instance of the §14
> failure mode in the codebase.

**Not yet audited:** the presence of a `TenantId` *column* does not prove every
*query* filters on it. FR-8.1 requires the query-path sweep, which this
inventory does not attempt.

---

## 3. Redis — two families, only one type-safe

`libs/atlas-redis` splits into constructors that build keys via
`tenantEntityKey(namespace, t, entityKey)` and constructors that build them via
`namespacedKey(namespace, parts...)`.

**Tenant-scoped by construction** — safe:
`NewTenantRegistry`, `NewTenantCoalescedRegistry`, `NewTenantSet`,
`NewTenantKeyedSet`, `NewTenantKeyedHash`, `NewTenantKeyedSortedSet`,
`NewTenantCounter`.

**Not tenant-scoped by construction** — tenant appears only if the caller's
`keyFn` puts it there:
`NewRegistry`, `NewSet`, `NewHash`, `NewKeyedSet`, `NewKeyedHash`,
`NewCoalescedRegistry`, `NewIDGenerator`, `NewIDGeneratorWithStart`,
`NewGlobalIDGenerator`, `NewIndex`, `NewUint32Index`, `NewLock`,
`NewLockWithTTL`, `NewTTLRegistry`.

### Services using the second family

| Service | Audited? | Notes |
|---|---|---|
| `atlas-monsters` | Partial | `monsterKey(t tenant.Model, uniqueId uint32)` at `monster/registry.go:303` **does** embed tenant. Namespaces seen: `monster`, `monster-map`, `monster-cooldown`, `monster-attack-cooldown`, `monster-puppet`, `monster-puppet-field`. Each keyFn still needs individual confirmation. |
| `atlas-summons` | Not audited | `summon`, `summon-map`, `summon-owner` (`summon/registry.go:171-173`) |
| `atlas-doors` | Not audited | — |
| `atlas-storage` | Not audited | — |
| `atlas-npc-shops` | Not audited | — |
| `atlas-data` | Not audited | — |

The `atlas-monsters` spot-check is the reason FR-8.2 says *enumerate*, not
*assume*: the type system permits an unscoped key, and the one callsite checked
happened to be correct. Today the `ATLAS_ENV` prefix would mask any callsite that
forgot; removing it turns such a callsite into an immediate cross-environment
leak with no containing boundary.

Also note the deliberately global constructors — `NewGlobalIDGenerator`,
`NewLock`. A shared lock namespace across environments may be correct (mutual
exclusion on a genuinely shared resource) or a cross-environment stall. Each
needs a stated intent.

---

## 4. Other resources to enumerate (FR-8.5)

Not yet investigated during specification; listed so the audit does not stop at
Postgres and Redis:

- MinIO / canonical baseline objects — currently restored per environment
  (`docs/runbooks/ephemeral-pr-deployments.md` §9.1).
- Kafka consumer group names — already runtime-resolved
  (`libs/atlas-kafka/consumergroup/resolver.go`), listed for completeness.
- Any in-process cache or registry keyed without tenant.
- Scheduled/delayed work stores, including saga state in
  `atlas-saga-orchestrator`.
- Object-ID generation (`libs/atlas-object-id`) — uniqueness assumptions across
  environments sharing a keyspace.
- The outbox (`libs/atlas-outbox`) — whether rows carry enough context to restore
  environment on replay (PRD input §7.3).

---

## 5. How to finish this audit

1. Sweep query paths, not just schemas (FR-8.1) — a `tenant_id` column that no
   `WHERE` clause references is not isolation.
2. Enumerate every `keyFn` in the six services of §3 (FR-8.2).
3. Land the `atlas-configurations` migration first; it is the one known blocker
   (FR-8.3).
4. Add the CI guard before migrating services, so the audit cannot regress while
   the migration is in flight (FR-8.4).
5. Record a disposition for every §4 resource: tenant-scope it, or declare that
   it forces isolated mode (FR-9.3).

State findings as hypotheses until confirmed against the source. A spot-check
presented as a sweep is a false "verified" here in the most costly possible
place — it would be discovered as silent data corruption in `main`.
