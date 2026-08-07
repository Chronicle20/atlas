# Tenant-Configuration Seed Provisioning & Transport Registry Fixes — Design

Task: `task-189-tenant-config-seed-provisioning`
Inputs: [`prd.md`](./prd.md), [`evidence.md`](./evidence.md)
Status: Draft for review

---

## 1. Scope recap

Three defects, one delivery:

1. **Nothing ever seeds tenant transport configuration.** Fix by moving the three
   transport resources onto `libs/atlas-seeder` and surfacing them on the Setup
   page (FR-1, FR-2).
2. **Configuration-status events carry no tenant headers**, so the hot-reload
   path in atlas-transports resolves the nil tenant and loads zero routes
   (FR-3).
3. **Route ids are minted fresh on every load**, so each replica/restart writes
   another full copy into the shared Redis registry (FR-4, FR-5).

Everything below is grounded in the code as it exists on this branch; file:line
references are to that state.

---

## 2. What the existing code actually does

These five facts constrain every decision that follows. Each was read, not
assumed.

**F1 — `configurations` is one row per `(tenant, resource_name)`, not one row
per entry.** `Entity` (`services/atlas-tenants/atlas.com/tenants/configuration/entity.go:11-17`)
holds `resource_data jsonb`; the blob is a JSON:API document whose `data` is an
**array** of entries. `CreateRoute` (`processor.go:181-273`) read-modify-writes
that array. "12 routes" is 12 array elements inside 1 DB row. `Entity` embeds
`gorm.Model`, so deletes are soft and `updated_at` is available.

**F2 — the stored entry shape is `{id, type, attributes}`.** `TransformRoute`
(`configuration/rest.go:41-48`) reads `data["id"]` and `data["attributes"]`.
The current seed files at `services/atlas-tenants/configurations/**` are exactly
that shape at top level — **not** wrapped in a `{"data": …}` envelope, which is
what `seeder.ParseEnvelope` (`libs/atlas-seeder/jsonapi.go:20-40`) requires.

**F3 — filename ≠ entity id.** `configurations/instance-routes/flight-temple-of-time-leafre.json`
has `"id": "temple-of-time-return-flight"`. So the seeder `Subdomain` must
return `EntityIDPattern() == nil` and take the id from `data.id`
(`libs/atlas-seeder/seed.go:140-152` handles exactly this case).

**F4 — the emit context is tenant-free, and both the producer and consumer
sides fail silently.** `ProcessorImpl` is built as `NewProcessor(l, ctx, db)`
with the server context (`processor.go:170-176`); all 18 `…AndEmit` methods
enqueue via `outbox.EmitProvider(p.l, p.ctx, tx)`. `outbox.EnqueueBuffer`
snapshots headers from that ctx at enqueue time (`libs/atlas-outbox/bridge.go:20-24, 42-57`),
and `producer.TenantHeaderDecorator` returns **empty headers with a nil error**
when the ctx has no tenant (`libs/atlas-kafka/producer/header.go:34-37`). On the
receiving end, `consumer.TenantHeaderParser` builds a tenant from whatever it
found and installs it unconditionally (`libs/atlas-kafka/consumer/header.go:61-65`),
so absent headers become the **zero** tenant rather than no tenant — which is
why `tenant.MustFromContext` does not panic and the reload runs against
`00000000-0000-0000-0000-000000000000`.

**F5 — the catalog toolchain keys off a two-level `<region>/<version>` layout,
and skips `_`-prefixed directories.** CI stamps `CATALOG_REVISION` with
`for dir in deploy/seed/*/*/` (`.github/workflows/main-publish.yml:287`,
`pr-validation.yml:619`, `reconcile-bump-prs.yml:198`). `tools/catalog-lint`
skips any directory whose name starts with `_`, at both the region scan
(`main.go:33`) and the walk (`main.go:55-57`), and only classifies files at
depth ≥ 4 (`main.go:70-73`).

---

## 3. Architecture

### 3.1 Where the seed data lives — `deploy/seed/shared/all/`

**Decision (Open Question 1 & 2):** move
`services/atlas-tenants/configurations/{routes,vessels,instance-routes}/` into
the shared git-sync'd catalog at

```
deploy/seed/shared/all/routes/*.json
deploy/seed/shared/all/vessels/*.json
deploy/seed/shared/all/instance-routes/*.json
```

and label `deploy/k8s/base/atlas-tenants.yaml` with
`atlas.seed-catalog: "true"` so it gets the git-sync sidecar, the
`/var/run/seed-catalog` mount, and `SEED_CATALOG_ROOT` — the same three patches
the other nine consumers get from
`deploy/k8s/base/components/seed-catalog`. `services/atlas-tenants/configurations/`
and its two `Dockerfile` lines (`Dockerfile:138` list entry, `Dockerfile:163`
COPY) are deleted, so there is exactly one on-disk copy (FR-1.5, FR-1.6).

**Why `shared/all` and not `_shared`.** By F5, a directory named `_shared`
would be silently skipped by `catalog-lint` *and* silently missed by all three
CI `CATALOG_REVISION` stampers — no revision file, no envelope validation, no
error anywhere. That is precisely the silent-provisioning-failure class this
task exists to eliminate; we are not going to introduce a second instance of it
while fixing the first. `shared/all` is a region-shaped/version-shaped pair, so:

- `deploy/seed/*/*/` matches it → CI stamps its `CATALOG_REVISION` with no
  workflow change;
- `catalog-lint` treats `shared` as a region and `all` as a version, checks the
  revision file, and lints the JSON at `parts[2:] == routes|vessels|instance-routes`;
- `filesystemSource.Roots` never enumerates directories (it *formats*
  `<region>/<major>_<minor>` from the tenant, `catalog.go:57`), so `shared/all`
  can never be mistaken for a real tenant root — `all` is not `%d_%d`.

Three rules are added to `tools/catalog-lint/subdomains.go` (`routes`,
`vessels`, `instance-routes`, each `pattern: nil` per F3) so the shared tree is
linted rather than silently ignored.

**Cost accepted:** seed-data edits now require a merge to `main` (git-sync
`ref: main`) instead of an image rebuild. That is the same contract every other
seeded service already lives under, and the PR overlay already repoints
git-sync at the PR branch (`deploy/k8s/overlays/pr/patches/seed-catalog-ref.yaml`).

**File reshape (F2):** each file is rewrapped from `{id,type,attributes}` to
`{"data":{id,type,attributes}}`. This is mechanical over 30 files. The
`Subdomain.Build` then stores `env.Data` verbatim, so the resulting
`resource_data.data[]` entry is byte-identical to what the current bespoke
seeder writes — no change to `TransformRoute` inputs, no migration of existing
rows.

### 3.2 Shared-root support in `libs/atlas-seeder` (FR-1.3, FR-1.4)

`CatalogSource.Roots` already returns `[]string`; today `filesystemSource`
returns exactly one and `Seed`/`ReadStatus` use `roots[0]`
(`seed.go:48-64`, `status.go:22-26`). Three additive changes:

1. **Opt-in shared root.** Add
   `NewFilesystemCatalogSourceWithShared(envVar, fallbackRoot, sharedRel string)`.
   Its `Roots` returns `[]string{ <base>/<sharedRel>, <base>/<region>/<M>_<m> }`
   — **least-specific first**. `NewFilesystemCatalogSource` is unchanged and
   still returns one root, so the nine existing consumers see zero behavioural
   change. Only atlas-tenants constructs the shared variant, with
   `sharedRel = "shared/all"`.

   *Rejected:* unconditionally appending a shared root inside the existing
   constructor. It would fold the shared `CATALOG_REVISION` into all nine
   services' composite revision, producing a one-time spurious
   `seed catalog drift detected` warning across the fleet
   (`handlers.go:81-87`) for zero benefit.

2. **Merge across roots.** `runSubdomain` walks every root and builds
   `map[filename]root`, later (more specific) roots overwriting earlier ones,
   then loads the merged set in sorted filename order. Version-specific wins
   (FR-1.4). `Walk` already returns `nil` for a missing directory
   (`catalog.go:84-87`), so a root that contributes nothing costs one
   `ReadDir` ENOENT.

3. **Composite revision (Open Question 5).** A new
   `revisionFor(src, roots) string` joins each root's non-empty `Revision` with
   `+` in root order. Single-root callers get the identical string they get
   today. atlas-tenants gets `<sharedSha>+<versionSha>` — both stamped by the
   same CI loop, so in practice `<sha>+<sha>`; drift is still detected if
   either root moves. Plain join, not a hash, so the value stays debuggable in
   a log line.

`Seed` and `ReadStatus` both route through the same two helpers, so status and
seed can never disagree about which roots are in play.

### 3.3 atlas-tenants adopts the seeder (FR-1.1, FR-1.2, FR-1.7)

Three `seeder.Group`s in a new `services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go`:

| Group `Name` | `URLPrefix` | Subdomain `Name()`/`Path()` | `Type()` |
|---|---|---|---|
| `routes` | `/tenants/configurations/routes` | `routes` | `routes` |
| `vessels` | `/tenants/configurations/vessels` | `vessels` | `vessels` |
| `instance-routes` | `/tenants/configurations/instance-routes` | `instance-routes` | `instance-routes` |

Yielding, under the `/api` base path (`libs/atlas-rest/server/server.go:69`):

```
POST /api/tenants/configurations/<resource>/seed          → 202
GET  /api/tenants/configurations/<resource>/seed/status    → 200
```

**Routing is already in place.** `deploy/shared/routes.conf:512` proxies
`^/api/tenants(/.*)?$` to `atlas-tenants:8080` with `$request_uri` intact — no
ingress change, no `tools/gen-routes.sh` change.

**No mux collision with the surviving CRUD routes.** `/tenants/configurations/routes/seed`
is 4 segments; the CRUD patterns are `/tenants/{tenantId}/configurations/<res>[/{id}]`,
whose 3rd segment is the literal `configurations`. Our 3rd segment is the
resource name, so no pattern can match. Same for the 5-segment
`…/seed/status`. The literal routes are nonetheless registered **before** the
`{tenantId}` patterns as belt-and-braces, and a route test asserts each of the
six new paths dispatches to the seeder handler.

The three bespoke `POST /tenants/{tenantId}/configurations/{routes|vessels|instance-routes}/seed`
handlers, `SeedRoutes`/`SeedVessels`/`SeedInstanceRoutes` on the `Processor`
interface, and the corresponding loaders in `configuration/seed.go` are
deleted. `rps-rewards/seed` and `mts-configs/seed` are untouched, so
`configuration/seed.go` keeps `LoadRpsRewardFiles`/`LoadMtsConfigFiles` and
their `/configurations/...` default paths — those two resources keep reading
from the image until a follow-up migrates them.

> **Consequence worth stating:** `rps-rewards` and `mts-configs` still read
> from `/configurations/rps-rewards` and `/configurations/mts-configs` inside
> the image, so `Dockerfile:163`'s `COPY /app/configurations /configurations`
> **stays**, and only the three migrated subdirectories are removed from
> `services/atlas-tenants/configurations/`. Deleting the whole tree would break
> the two out-of-scope seeders.

**Subdomain implementation.** Each subdomain is a thin adapter over the
existing administrator functions:

- `Decode(payload []byte) (map[string]any, error)` → `seeder.ParseEnvelope`,
  returning `env.Data` as a map (F2).
- `Build(t, entityID, entry) ([]entry, error)` → stamps `entry["id"] = entityID`
  and returns a one-element slice.
- `DeleteAllForTenant(db)` → `DeleteConfigurationByResourceName` scoped by the
  `libs/atlas-database` tenant callback (`tenant_scope.go:35, 77`), which
  injects `tenant_id = <ctx tenant>` automatically. `seeder.Seed` guarantees
  the ctx tenant (`seed.go:43`), so the cross-tenant NFR is satisfied by
  existing plumbing rather than a hand-written predicate.
- `BulkCreate(db, entries)` → **create-or-append** into the single
  `(tenant, resource_name)` row: the same read-modify-write `CreateRoute`
  already performs, extracted into one shared
  `AppendConfigurationEntries(db, resourceName, entries []map[string]any) error`
  used by all three subdomains.
- `Count(db)` → `jsonb_array_length(resource_data->'data')` for the row, with a
  `CASE` returning 1 for the legacy single-object shape and 0 when no row
  exists; `updated_at` supplies the status `updatedAt`.

`BulkCreate` runs once per file, so a 12-file seed performs 12 appends. That is
serialized and safe: `seeder.Seed` holds a per-`(tenant, group)` mutex for the
whole run (`seed.go:32-40`), and the three subdomains live in *different*
groups so they never contend on the same row. Idempotency (FR-1.7) comes free
from delete-all-then-append.

*Rejected:* teaching `libs/atlas-seeder` a "collect all files, write once"
mode. It changes the `Subdomain` contract for all nine consumers to serve one
storage shape. 12 sequential JSONB updates on a rarely-run admin operation is
not a cost worth a library redesign.

**`seed_state`** is added to atlas-tenants' migration list exactly as the other
consumers do — `func(db *gorm.DB) error { return db.AutoMigrate(&seeder.SeedState{}) }`
alongside the existing `database.SetMigrations(...)` call at
`services/atlas-tenants/atlas.com/tenants/main.go:48`.

### 3.4 Emitting the reload event: `Group.AfterSeed`

The seeder's `postSeed` seeds in a background goroutine and returns 202
(`handlers.go:47-71`); nothing in the library can emit a domain event. Add one
optional field:

```go
type Group struct {
    Name       string
    URLPrefix  string
    Subdomains []SubdomainAny
    // AfterSeed, when non-nil, runs once after a successful Seed with the
    // tenant-bearing seed context. Errors are logged, not returned to the
    // caller — the seed itself has already committed.
    AfterSeed  func(ctx context.Context, db *gorm.DB, res Result) error
}
```

Nil for the nine existing groups; purely additive. atlas-tenants uses it to
enqueue **one** configuration-status event per group through the outbox, in a
`database.ExecuteTransaction`, using the `bgCtx` `postSeed` already builds with
`tenant.WithContext` (`handlers.go:53`) — so the tenant headers are populated by
construction.

Event shape reuses the existing providers (`configuration/kafka.go`) with a
synthetic resource id, since the consumer switches on `ResourceType` only and
uses `ResourceId` for logging
(`services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer.go:42-68`):

| Group | provider | `Type` | `ResourceId` |
|---|---|---|---|
| `routes` | `CreateRouteStatusEventProvider` | `ROUTE_UPDATED` | `*` |
| `vessels` | `CreateVesselStatusEventProvider` | `VESSEL_UPDATED` | `*` |
| `instance-routes` | `CreateInstanceRouteStatusEventProvider` | `INSTANCE_ROUTE_UPDATED` | `*` |

One event per group, not one per file: the consumer's handler is
`ClearTenant()` + full reload, so 12 events would trigger 12 reloads and — worse
— a reload could land mid-seed and load a partial set.

*Rejected:* emitting from inside `BulkCreate` by recovering the context off
`db.Statement.Context`. It works, but it produces the per-file storm above and
smuggles domain events through a persistence hook.

### 3.5 Tenant context on every configuration emitter (FR-3.1–FR-3.4)

**Verified (Open Question 6): `rankings` shares the defect.**
`CreateRankingsAndEmit` (`processor.go:1640-1652`) uses the identical
`outbox.EmitProvider(p.l, p.ctx, tx)` with the tenant-free `p.ctx`. All six
resources — `routes`, `vessels`, `instance-routes`, `rps-rewards`,
`mts-configs`, `rankings` — across **18** `…AndEmit` methods are affected
(FR-3.3).

**Fix:** a single private helper on `ProcessorImpl`:

```go
func (p *ProcessorImpl) tenantCtx(tenantId uuid.UUID) (context.Context, error)
```

It loads the tenant row via the service's own `tenant.Processor.GetById`
(`tenant/processor.go:251`), builds a `tenant.Model` with
`libs/atlas-tenant`'s `Create(id, region, major, minor)`, and returns
`tenant.WithContext(p.ctx, t)`. Every `…AndEmit` resolves it first and threads
it into both `p.db.WithContext(...)` and `outbox.EmitProvider(p.l, ctx, tx)`.
A resolution failure aborts the write with an error rather than emitting
tenant-free — the operation is meaningless for an unknown tenant.

Cost: one extra `SELECT` per configuration write. Configuration writes are
operator-rate; no cache.

*Rejected:* dropping `tenantId uuid.UUID` from the `Processor` interface and
reading the tenant from ctx (the idiomatic Atlas shape). It is the right
end-state, but it rewrites a 1731-line processor, a 1305-line resource file,
and their tests for a defect that a 5-line helper closes — and the REST layer is
path-scoped (`/tenants/{tenantId}/…`), so handlers would have to build the
context anyway. Noted as future cleanup, not done here.

### 3.6 FR-3.5 — making the silent drop observable

Two guards land here; one library-wide change is explicitly deferred with the
reason recorded.

**Lands now (producer side):** `libs/atlas-outbox.EnqueueBuffer` already holds a
`logrus.FieldLogger`. After `headerMap(ctx)` it checks for `tenant.ID` and, when
absent, logs a `WARN` naming the topic. Every configuration-status event travels
this path, so the whole class becomes visible without touching a single error
contract.

**Lands now (consumer side):** atlas-transports' `handleConfigurationStatus`
rejects a nil-tenant context — if `tenant.MustFromContext(ctx).Id() == uuid.Nil`,
log `ERROR` ("configuration-status event without tenant headers; skipping
reload") and return. This is what makes the NFR "the nil-tenant signature MUST
be treated as a regression" mechanically true instead of aspirational: a
regression now produces an error and *no* destructive `ClearTenant`, rather than
a quiet clear-and-load-zero.

**Deferred, with rationale:** changing `TenantHeaderDecorator`
(`libs/atlas-kafka/producer/header.go:31-44`) to return an error, or changing
`TenantHeaderParser` (`libs/atlas-kafka/consumer/header.go:61-65`) to install no
tenant instead of the zero tenant. Both are hot paths in all 59 services, and
the parser change in particular would convert today's silent zero-tenant into a
`MustFromContext` panic in every consumer that currently tolerates it. Landing
either safely requires first auditing which emitters legitimately produce
without a tenant — an audit with a blast radius far larger than this task, and
one this task's WARN is the right instrument to *drive*. Recorded here so it is
documented rather than forgotten (FR-3.5's explicit escape clause).

### 3.7 Stable route identity (FR-4)

**Decision (Open Question 3): derive, do not store.**

```
routeUUID = uuid.NewSHA1(tenantId, []byte(resourceName + "/" + slug))
```

UUIDv5 with the tenant id as namespace. Stable across loads, replicas,
restarts and re-seeds; different per tenant for the same slug (FR-4.3);
different across `routes` and `instance-routes` for a shared slug. No schema
change, no backfill, and — decisively — it is correct for rows that already
exist in `atlas-main` today.

*Rejected:* a stored `uuid` column on each entry. It needs a migration plus a
backfill, and the value it buys (operator-editable ids) is not a requirement.

**atlas-tenants exposes it (FR-4.1).** `RouteRestModel` and
`InstanceRouteRestModel` in `configuration/rest.go` gain
a `Uuid` field tagged `json:"uuid"`; `TransformRoute`/`TransformInstanceRoute` gain a
`tenantId uuid.UUID` parameter and populate it. The JSON:API resource `id`
stays the slug (FR-4.1's "additively") because configuration-status events and
the CRUD routes reference resources by slug.

**atlas-transports consumes it (FR-4.2, FR-4.5).** Both rest models gain
the same `Uuid` field tagged `json:"uuid"`. `ExtractRoute` becomes tenant-aware —
`ExtractRouteFor(t tenant.Model) func(RouteRestModel) (transport.Model, error)`,
passed as the mapper to `requests.DrainProvider` (which already takes the
mapper as a value:
`instance/config/processor.go:41`) — and calls `SetId(parsed)`.

When `uuid` is absent or unparseable, atlas-transports logs a `WARN` and derives
the same UUIDv5 locally. This keeps a staggered rollout (transports up before
tenants) correct rather than merely non-crashing, and because the formula is
identical the two paths can never diverge. The formula lives once, in
`libs/atlas-tenant` as `func DerivedId(tenantId uuid.UUID, parts ...string) uuid.UUID`,
with a table test pinning known vectors so a future edit to either consumer
cannot silently re-key the registry.

**Vessels are not affected.** `ExtractVessel` already calls `SetId(v.Id)` with
the slug (`transport/config/rest.go:87-93`), which is why the evidence records
drift on `routes` and `instance-routes` (24 vs 12 each) but not on vessels.

### 3.8 FR-5 — converging the existing drift

The duplicates in `atlas-main` are entries keyed by ids nothing will ever
generate again. Stable ids make `Put` idempotent, but nothing *removes* the old
random-id entries — so a purge is required.

**Mechanism: reconcile at bootstrap, not by hand.**
`services/atlas-transports/atlas.com/transports/main.go:101-120` becomes
**load → (on success) clear → add**, for both registries:

```
routes, vessels, err := configProcessor.LoadConfigurationsForTenant(t)
if err != nil { log ERROR; continue }          // leave the registry untouched
transport.NewProcessor(l, ctx).ClearTenant()
transport.NewProcessor(l, ctx).AddTenant(routes, vessels)
```

Note the ordering: today a load failure falls through to `AddTenant` with an
empty slice (`main.go:106-110`); under clear-then-add that would **wipe** a
healthy registry. Loading first and skipping the whole reconcile on error is
required, not incidental.

This makes the deploy itself the cleanup: one rolling restart converges every
tenant to exactly the configured count, permanently, with no `redis-cli`
surgery (which `tools/redis-key-guard.sh` discourages for good reason) and no
step an operator can forget. It also self-heals any future drift.

Two properties, stated rather than glossed:

- **Concurrent replicas converge.** Both write the same stable ids, so
  regardless of clear/add interleaving the end state is exactly the configured
  set.
- **A restart clobbers live scheduled-route state.** It already does: with
  stable ids, replica B's `Put` overwrites replica A's entry with a
  freshly-computed schedule, and the 1-second `UpdateRoutes()` tick in both
  replicas is already last-writer-wins (`main.go:123-140`). Clear-then-add adds
  a brief window where the registry is partial during startup, bounded by the
  reconcile loop. Acceptable for a startup path; documented in the runbook.

**FR-5.2 runbook** — `docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md`
covers: rollout order, the post-deploy verification loop (per tenant,
`GET /api/transports/routes` and `/api/transports/instance-routes` `total` must
equal the seeded configuration count from
`GET /api/tenants/configurations/<res>/seed/status`), what a mismatch means, and
the manual fallback (scale atlas-transports to 0, delete the
`instance-route:*` / `transport-route:*` tenant keys, scale back) for the case
where the automatic reconcile does not converge.

### 3.9 Setup page (FR-2)

Three `SeedRow` entries appended to `seedRows` in
`services/atlas-ui/src/pages/SetupPage.tsx:183`, following the eight existing
rows exactly (`mutation` + `formatBadge`, `—` when the query has no data):

| Label | Icon (lucide-react) | Status key | Badge |
|---|---|---|---|
| Transport Routes | `Ship` | `routes` | `N routes` |
| Transport Vessels | `Anchor` | `vessels` | `N vessels` |
| Instance Transport Routes | `Plane` | `instance-routes` | `N routes` |

`seed.service.ts` gains `seedRoutes`/`seedVessels`/`seedInstanceRoutes` (each
`api.post("/api/tenants/configurations/<res>/seed", {})`, mirroring `seedDrops`
— the 202 + poll-status contract, no result body) and three
`get*SeedStatus` methods projecting `subdomainCount(s, "<res>")` and
`s.tenantSeededAt ?? s.updatedAt`. `useSeed.ts` gains three query keys, three
`useSeed*` mutations invalidating their key, and three status queries with the
established `staleTime: 0, refetchInterval: 5000`. Tenant scoping is the
existing `TenantProvider` → `tenantHeaders` path; no new UI (FR-2.4).

> **Design concern, raised and deferred to the PRD.** FR-2.1 mandates three
> independent rows. Scheduled transports need **both** `routes` and `vessels`
> to compute a schedule, so an operator who seeds one and not the other leaves
> the tenant in a quietly-broken state — the same failure class this task
> fixes. A single "Transport Configuration" group with three subdomains and one
> badge (`12 routes / 6 vessels / 12 instance routes`) would make partial
> seeding impossible and matches the multi-subdomain convention Drops and
> Reward Pools already use. The PRD is explicit, so this design implements
> three rows; the consolidation is a cheap follow-up if the hazard proves real.
> Mitigation within scope: seeding either `routes` or `vessels` triggers a full
> scheduled-transport reload, so the order does not matter and the second seed
> converges.

---

## 4. Data flow

**Seed (operator clicks "Transport Routes"):**

```
SetupPage → useSeedRoutes → POST /api/tenants/configurations/routes/seed
  → nginx (^/api/tenants) → atlas-tenants
  → seeder.postSeed: 202, background goroutine with tenant-bearing ctx
  → seeder.Seed: lock(tenant,"routes")
       → Roots = [<catalog>/shared/all, <catalog>/gms/83_1]
       → merge filenames, version-specific wins
       → DeleteAllForTenant (tenant callback scopes by tenant_id)
       → per file: ParseEnvelope → append entry to configurations.resource_data.data[]
       → UpsertSeedState(tenant, "routes", "<sharedSha>+<versionSha>")
  → Group.AfterSeed: outbox.Enqueue(ROUTE_UPDATED, headers from tenant ctx)
  → outbox drainer → EVENT_TOPIC_CONFIGURATION_STATUS (with TENANT_ID …)
  → atlas-transports consumer: TenantHeaderParser → real tenant
       → guard: tenant id non-nil
       → ClearTenant + LoadConfigurationsForTenant + AddTenant
       → GET /api/tenants/{realTenantId}/configurations/routes (paginated drain)
       → ExtractRouteFor(t): SetId(uuid from `uuid` attr, else derived) 
       → registry keys = stable ids → idempotent
SetupPage's 5s poll of /seed/status shows the new count.
```

**Bootstrap (pod start):** per tenant, load → clear → add, same stable ids.

---

## 5. Testing

**`libs/atlas-seeder`** (extend `catalog_test.go`, `seed_test.go`):
merge precedence — a relPath present in both roots resolves to the
version-specific file; union across roots; a missing shared root is a no-op;
composite revision equals the single revision for a one-root source and
`a+b` for two; `AfterSeed` runs once on success with a tenant-bearing ctx and a
nil hook is a no-op.

**atlas-tenants:** per-subdomain `Decode`/`Build` round-trip proving the stored
entry is byte-identical to the pre-migration shape (F2); `Count` for the empty,
array and legacy-single-object cases; **a cross-tenant test** seeding tenant A
and asserting tenant B's rows are untouched (NFR — most severe possible
regression); a route test asserting all six new paths dispatch and that
`/tenants/{uuid}/configurations/routes` still reaches the CRUD handler; and a
**regression test that a configuration-status event carries all four tenant
headers**, asserted at the outbox row (`Entity.Headers`) so it covers the
enqueue-time snapshot that actually failed.

**atlas-transports:** `ExtractRouteFor` returns the same id across repeated
calls and differs across tenants for one slug (FR-4 acceptance); the absent-`uuid`
fallback derives the identical id and warns; `handleConfigurationStatus`
with a nil-tenant ctx logs and performs **no** `ClearTenant`; bootstrap
reconcile skips clear+add when the load errors.

**atlas-ui:** vitest for the three new hooks/service methods; `npm run build`
(not vitest alone) per the per-task verification rule.

**Live (acceptance):** on a tenant provisioned only through the Setup page,
enter `270000100` via `out00` and confirm the warp to `200090510` with no
"flight service is currently unavailable"; confirm
`00000000-0000-0000-0000-000000000000` appears in no atlas-transports reload
log; confirm live counts equal configured counts on all ten tenants after a
restart with ≥2 replicas.

---

## 6. Rollout

1. Merge; `main-publish` stamps `CATALOG_REVISION` into `deploy/seed/shared/all/`
   and bumps atlas-tenants, atlas-transports, atlas-ui.
2. atlas-tenants restarts with the git-sync sidecar and the `seed_state`
   migration. Existing configuration rows are untouched.
3. atlas-transports restarts; the bootstrap reconcile purges the duplicate
   entries (FR-5.1) using ids derived locally if atlas-tenants has not rolled
   yet.
4. Operator opens Setup on each tenant and confirms the three badges show
   12 / 6 / 12. Freshly provisioned tenants now seed from the UI.

**Rollback:** revert the images. The `seed_state` table and the reshaped
catalog files are inert to the previous version, and configuration rows are
unchanged in shape, so rollback is a pure image revert. The only lost behaviour
is the reconcile, which means the old random-id duplicates would begin
re-accumulating from that point.

---

## 7. Verification gates (CLAUDE.md §Build & Verification)

Changed Go modules: `libs/atlas-seeder`, `libs/atlas-outbox`,
`libs/atlas-tenant`, `services/atlas-tenants`, `services/atlas-transports`.

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in each.
- `docker buildx bake atlas-tenants atlas-transports` — both `go.mod`s gain
  dependencies (atlas-seeder for tenants; atlas-tenant is already present in
  transports but `libs/atlas-seeder`'s COPY lines must be confirmed for the
  tenants image).
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/lint.sh --check` clean.
- `tools/service-registration-guard.sh` — `deploy/k8s/base/atlas-tenants.yaml`
  changed.
- `go run ./tools/catalog-lint deploy/seed` clean, including the new
  `shared/all` tree.
- atlas-ui: `npm run build` plus `npm run test`.

---

## 8. Open items for the plan phase

- Confirm the repo-root `Dockerfile` already COPYs `libs/atlas-seeder` for the
  shared build (it must, since nine services use it) — verify rather than
  assume before the bake.
- Decide the exact wording of the three Setup badges against the existing
  `pluralize`/`formatCount` helpers.
- The outbox nil-tenant `WARN` (§3.6) may be noisy if some existing emitters
  legitimately have no tenant. Validate in the ephemeral PR environment; if it
  floods, downgrade to a counter and keep the atlas-transports `ERROR` guard,
  which is the one that actually protects the registry.
