# Tenant-Configuration Seed Provisioning & Transport Registry Fixes — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-03
---

## 1. Overview

Transport configuration data — scheduled routes, vessels, and instance-transport
routes — is defined as JSON in `services/atlas-tenants/configurations/` and baked
into the atlas-tenants image. atlas-tenants exposes a working
`POST /tenants/{tenantId}/configurations/{resource}/seed` endpoint for each
resource. Nothing in the product ever calls those endpoints. Not the UI, not any
Kubernetes bootstrap job, not the baseline restore. The only way transport data
has ever entered a tenant is a hand-issued HTTP request.

The consequence is silent and total: every freshly provisioned tenant, and every
tenant rebuilt from a restored baseline, ships with zero boats, zero ships, and
zero instance flights. There is no error at startup, no warning in a log, and no
indication in the UI — the tenant simply behaves as though those features do not
exist. On 2026-08-03 all ten live tenants in `atlas-main` were confirmed to hold
`0` rows for `routes`, `vessels`, `instance-routes`, and `rps-rewards`. The
symptom that surfaced it was a player walking into the Temple of Time `out00`
portal and receiving "The flight service is currently unavailable" — the portal
action's `start_instance_transport` step failed with `TRANSPORT_ROUTE_NOT_FOUND`
because the route it names had never been provisioned.

Fixing the provisioning gap alone is not sufficient. Seeding the data live
exposed two further defects that prevented the seeded configuration from
reaching the atlas-transports runtime registry correctly: configuration-status
events are emitted without tenant headers (so the hot-reload path resolves the
nil tenant and loads nothing), and route models are assigned freshly-minted
UUIDs on every load (so each replica registers a duplicate copy of every route
into the shared Redis registry on every restart). This task closes all three.

## 2. Goals

Primary goals:

- Make transport configuration seeding a first-class, discoverable operation in
  the atlas-ui Setup page, consistent with the eight seed rows that already
  exist there.
- Migrate atlas-tenants configuration seeding onto the shared `libs/atlas-seeder`
  framework so it gains a status endpoint, per-tenant revision tracking, and UI
  parity without a bespoke second seeding idiom.
- Restore the configuration hot-reload path so a seed or edit propagates to
  atlas-transports without a pod restart.
- Give every configuration-derived route a stable identity, so repeated loads
  and multiple replicas converge on one registry entry per configured route.
- Remove the duplicate registry entries that the current defect has already
  written into the live `atlas-main` Redis registry.

Non-goals:

- Wiring hot-reload consumers for `rps-rewards`, `mts-configs`, or `rankings`.
  Those resources emit configuration-status events that no service consumes
  today; giving them consumers is separate work.
- Any change to transport gameplay behavior — boarding, capacity, travel
  timing, transit messaging, or the saga that drives them.
- The `COMMAND_TOPIC_INSTANCE_TRANSPORT` consumer reader-recreate churn observed
  in atlas-transports logs. It appears on every topic the service consumes and
  reads as an idle-topic heartbeat, but this is **unverified** and explicitly
  left out of scope.
- Seeding `mts-configs`. It shares the provisioning gap but has no reported
  symptom and no transport dependency (see §2 scope decision, resources limited
  to the three transport-related ones).

## 3. User Stories

- As an operator provisioning a new tenant, I want transport configuration to
  appear as a seedable row in the Setup page so that I do not have to know a
  hand-issued API call exists.
- As an operator, I want each transport seed row to show a current count so that
  I can tell at a glance whether a tenant is provisioned, rather than
  discovering it from a player bug report.
- As an operator editing a route in the UI, I want the change to take effect in
  atlas-transports without restarting the service.
- As a player, I want the Temple of Time flight — and every other boat, ship,
  and instance transport — to work on a freshly provisioned server.
- As an engineer, I want one route in configuration to mean exactly one route in
  the registry, regardless of replica count or restart history.

## 4. Functional Requirements

### FR-1 — Migrate configuration seeding onto `libs/atlas-seeder`

**FR-1.1** atlas-tenants MUST register `libs/atlas-seeder` `Group`s for the three
transport-related resources, replacing the bespoke seed handlers at
`services/atlas-tenants/atlas.com/tenants/configuration/resource.go:1259`
(`routes`), `:1267` (`vessels`), and `:1275` (`instance-routes`).

**FR-1.2** Each Group MUST expose the standard pair
`POST /<prefix>/seed` and `GET /<prefix>/seed/status`, tenant-*header* scoped via
`server.ParseTenant`, matching every other seeder consumer. The existing
tenant-*path*-scoped seed endpoints MUST be removed once the replacements are in
place; leaving both is an explicit non-goal (one idiom, not two).

**FR-1.3** The catalog source MUST support version-agnostic seed data.
`services/atlas-tenants/configurations/` is a single shared set applied
identically to all tenants, whereas `filesystemSource.Roots` (`libs/atlas-seeder/catalog.go`)
currently resolves exactly one per-region/version root
(`<base>/<region>/<major>_<minor>`). `Roots` already returns `[]string`, so the
required change is additive: it MUST also return a shared root that is not
region- or version-qualified, and `Seed`/`Walk` MUST merge entries across the
returned roots. Duplicating twelve identical route files across ten version
directories is NOT acceptable.

**FR-1.4** When the same relative path exists in both the shared root and a
version-specific root, the version-specific entry MUST win, so a future
per-version override remains possible.

**FR-1.5** The seed data MUST remain a single source of truth. Whether it stays
at `services/atlas-tenants/configurations/` and is mounted, or moves under the
shared catalog tree consumed by `deploy/k8s/base/components/seed-catalog`, is a
design decision (see §9) — but it MUST NOT be forked into two copies.

**FR-1.6** atlas-tenants MUST be added to the `seed-catalog` kustomize component
consumer list (joining the nine services that already reference it in
`deploy/k8s/base/`) if and only if the design lands the data in the shared
catalog.

**FR-1.7** Seeding MUST remain idempotent: re-running a seed against an
already-seeded tenant MUST NOT create duplicate configuration rows.

### FR-2 — Setup page seed rows

**FR-2.1** `services/atlas-ui/src/pages/SetupPage.tsx` MUST gain **three**
additional `SeedRow` entries — one per domain, consistent with the existing
one-row-per-domain convention:

| Label | Resource | Badge |
|---|---|---|
| Transport Routes | `routes` | route count |
| Transport Vessels | `vessels` | vessel count |
| Instance Transport Routes | `instance-routes` | route count |

**FR-2.2** Each row MUST render a live count badge sourced from the
corresponding `GET /<prefix>/seed/status` endpoint, following the
`subdomainCount` pattern in `services/atlas-ui/src/services/api/seed.service.ts`.
A row with no data MUST render `—`, matching existing rows.

**FR-2.3** `seed.service.ts` MUST gain the corresponding `seed*` mutation methods
and `get*SeedStatus` query methods; `useSeed.ts` MUST gain the matching hooks and
query keys.

**FR-2.4** Seed rows MUST be tenant-scoped through the existing
`TenantProvider` header mechanism. No new tenant-selection UI.

**FR-2.5** Rows MUST follow the existing pending/disabled and toast-on-error
conventions used by the eight current rows.

### FR-3 — Tenant context on configuration-status events

**FR-3.1** atlas-tenants MUST emit `EVENT_TOPIC_CONFIGURATION_STATUS` messages
carrying the four tenant headers.

**FR-3.2** The root cause MUST be fixed at the context, not worked around at the
consumer. `producer.ProviderImpl` (`libs/atlas-kafka/producer/provider.go:16-17`)
already applies `TenantHeaderDecorator` unconditionally; that decorator reads
`tenant.FromContext(ctx)` and, on failure, returns empty headers with a **nil
error** (`libs/atlas-kafka/producer/header.go:31-37`) — a silent drop. The defect
is that atlas-tenants' configuration processor threads the tenant as a bare
`uuid.UUID` throughout its interface
(`services/atlas-tenants/atlas.com/tenants/configuration/processor.go:24-91`)
and constructs with a tenant-free server context via
`NewProcessor(l, ctx, db)`. The emit context MUST carry a fully-populated
`tenant.Model`. atlas-tenants owns the tenants table and therefore has the
region, major, and minor version needed to build one.

**FR-3.3** All configuration-status emitters MUST be fixed, not only the
instance-route path — `routes`, `vessels`, `instance-routes`, `rps-rewards`,
`mts-configs`, and `rankings` all emit through the same tenant-free context.

**FR-3.4** After the fix, a seed or edit MUST cause atlas-transports to reload
the correct tenant's routes with no restart. The service log MUST show the real
tenant UUID, never `00000000-0000-0000-0000-000000000000`.

**FR-3.5** Consider making `TenantHeaderDecorator`'s missing-tenant case
observable rather than silent (a warning, or a distinguishable error). This is
a `libs/atlas-kafka` change affecting all 59 services; it is in scope to
*evaluate* and may be deferred with a documented rationale, but silently
dropping tenant headers on a tenant-scoped topic MUST NOT be left undocumented.

### FR-4 — Stable route identity

**FR-4.1** atlas-tenants MUST expose a stable UUID for each configuration entry.
Today the JSON:API resource id is the human slug (e.g. `hak-to-orbis`,
`temple-of-time-return-flight`) and no UUID is surfaced.

**FR-4.2** atlas-transports MUST use that stable UUID as the route id.
Both `ExtractRoute` implementations currently omit `SetId` —
`services/atlas-transports/atlas.com/transports/instance/config/rest.go:34` and
`services/atlas-transports/atlas.com/transports/transport/config/rest.go:42` —
so the builders fall through to `uuid.New()`
(`instance/builder.go:26`, `transport/builder.go:32`).

**FR-4.3** The id MUST be stable across: repeated loads within one process,
multiple replicas, service restarts, and re-seeding the same configuration. It
MUST differ across tenants for the same slug.

**FR-4.4** After the fix, the live route count MUST equal the configured route
count exactly. The registry key is `route.Id()`
(`instance/route_registry.go:36`), so stable ids make `AddTenant` idempotent.

**FR-4.5** Both registries MUST be fixed — scheduled `transport` and
`instance`. The live drift was observed in both (`routes=24` and
`instance-routes=24` against 12 and 12 configured entries).

### FR-5 — One-time drift cleanup

**FR-5.1** The task MUST remove the duplicate entries already written to the
live `atlas-main` Redis registry. As of 2026-08-03 every tenant holds 24
registry entries per 12 configured entries.

**FR-5.2** Cleanup MUST be safe to run against a live cluster and MUST be
documented as a runbook step in the task folder, not left as tribal knowledge.

**FR-5.3** The cleanup MUST be verified by asserting live counts match
configured counts for all ten tenants after it runs.

## 5. API Surface

### Added (atlas-tenants)

Tenant-header scoped, provided by `libs/atlas-seeder.RegisterRoutes`:

```
POST /tenants/configurations/routes/seed                 → 202 Accepted
GET  /tenants/configurations/routes/seed/status          → 200 { catalogRevision, subdomains: { routes: { count } } }
POST /tenants/configurations/vessels/seed                → 202 Accepted
GET  /tenants/configurations/vessels/seed/status         → 200 { … }
POST /tenants/configurations/instance-routes/seed        → 202 Accepted
GET  /tenants/configurations/instance-routes/seed/status → 200 { … }
```

Exact URL prefixes are a design decision — they MUST NOT collide with the
existing `/tenants/{tenantId}/configurations/...` CRUD routes, which remain.

Required headers on all six: `TENANT_ID`, `REGION`, `MAJOR_VERSION`,
`MINOR_VERSION`. Missing headers → `400`, per `server.ParseTenant`.

Note the semantic change: `libs/atlas-seeder`'s `postSeed` returns **202** and
seeds in a background goroutine, whereas the endpoints being replaced return
**200** with a synchronous `{deletedCount, createdCount, failedCount}` body. The
UI MUST be written against the 202 + poll-status contract.

### Removed (atlas-tenants)

```
POST /tenants/{tenantId}/configurations/routes/seed
POST /tenants/{tenantId}/configurations/vessels/seed
POST /tenants/{tenantId}/configurations/instance-routes/seed
```

`rps-rewards/seed` (`resource.go:1283`) and `mts-configs/seed` (`:1291`) are out
of scope and MUST be left untouched.

### Modified (atlas-tenants)

The instance-route, route, and vessel configuration read models MUST surface a
stable UUID per FR-4.1, additively — the existing slug id MUST keep working, as
configuration-status events reference resources by slug.

## 6. Data Model

- **`seed_state`** — atlas-tenants gains the `libs/atlas-seeder` GORM entity
  keyed `(tenant_id, group_name) -> catalog_revision`, one row per tenant per
  Group. Requires a migration in atlas-tenants.
- **Configuration rows** — the existing JSONB-backed configuration storage is
  unchanged in shape. If FR-4.1 stores a UUID rather than deriving one, that is
  a schema addition plus a backfill for existing rows.
- **Redis route registries** — `instance-route` and the scheduled-transport
  equivalent are keyed by route UUID under the tenant registry. No shape change;
  FR-4 changes only how the key is derived, and FR-5 purges the existing
  duplicates.
- All storage stays tenant-scoped. No cross-tenant reads are introduced.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-tenants` | Adopt `libs/atlas-seeder`; remove three bespoke seed handlers; thread a real `tenant.Model` into the emit context across all configuration emitters; expose stable configuration UUIDs; `seed_state` migration. |
| `atlas-transports` | `SetId` from the stable UUID in both `ExtractRoute`s; verify `AddTenant` is idempotent in both registries. No consumer change expected once FR-3 lands. |
| `atlas-ui` | Three Setup rows; `seed.service.ts` methods; `useSeed.ts` hooks and query keys; tests. |
| `libs/atlas-seeder` | Shared/version-agnostic root support in `filesystemSource.Roots` plus merge-across-roots in the walk (FR-1.3/1.4). |
| `libs/atlas-kafka` | Only if FR-3.5 lands the observability change. |
| `deploy/k8s` | `seed-catalog` component wiring for atlas-tenants, if the design puts the data in the shared catalog. |

## 8. Non-Functional Requirements

- **Multi-tenancy** — every new endpoint is tenant-header scoped; seeding one
  tenant MUST NOT touch another's rows. A cross-tenant leak here is the most
  severe possible regression and MUST have a test.
- **Idempotency** — re-seeding and re-loading are both no-ops on an
  already-correct tenant (FR-1.7, FR-4.4).
- **Observability** — a configuration reload MUST log the resolved tenant id.
  The nil-tenant signature `00000000-0000-0000-0000-000000000000` appearing in
  an atlas-transports reload log MUST be treated as a regression.
- **No silent failure** — the class of bug being fixed is precisely "silently
  does nothing." A seed that loads zero files, or a reload that resolves no
  tenant, MUST surface at WARN or above.
- **Backward compatibility** — an already-provisioned tenant MUST continue to
  work across the deploy; the stable-id change MUST converge rather than
  duplicate (FR-5).
- **Verification** — per `CLAUDE.md`: `go test -race ./...`, `go vet ./...`,
  `go build ./...` clean in every changed module; `docker buildx bake` for every
  service whose `go.mod` is touched; `tools/lint.sh --check`;
  `tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`. atlas-ui needs
  `npm run build`, not just vitest.

## 9. Open Questions

1. **Where does the shared seed data live?** Keep it at
   `services/atlas-tenants/configurations/` and point a `CatalogSource` at it, or
   move it under the shared catalog tree that `deploy/k8s/base/components/seed-catalog`
   mounts? The latter is more consistent but the catalog is currently
   per-region/version and this data is not. FR-1.3 constrains the outcome; the
   mechanism is for `/design-task`.
2. **What is the shared root's name and precedence?** `_common`, `_shared`,
   `common/`? Note `Walk` skips `_`-prefixed *entries within* a root, which does
   not conflict with a `_`-prefixed root directory — but this needs confirming
   rather than assuming.
3. **Derived or stored UUID?** A `uuid.NewSHA1(tenantId, slug)` derivation needs
   no migration and is stable by construction; a stored column is explicit and
   editable. FR-4.1 says "expose a stable UUID" without mandating either.
4. **Does FR-3.5 land in this task?** Making `TenantHeaderDecorator` loud touches
   a library all 59 services use. Fix here, or split with a documented rationale?
5. **What does `CATALOG_REVISION` mean for version-agnostic data?** The revision
   is currently read per-root; with two roots contributing, the status endpoint
   needs a defined composite.
6. **Does `rankings` share the FR-3 defect?** It emits through the same processor
   and is presumed affected, but this was not verified — confirm during design.

## 10. Acceptance Criteria

- [ ] Three seed rows — Transport Routes, Transport Vessels, Instance Transport
      Routes — render on the Setup page with live count badges.
- [ ] Clicking each row seeds that resource for the active tenant; the badge
      reflects the new count after the background seed completes.
- [ ] `GET /<prefix>/seed/status` returns a correct count for all three
      resources, for a seeded and an unseeded tenant.
- [ ] The three bespoke path-scoped seed endpoints are removed; `rps-rewards`
      and `mts-configs` seed endpoints are untouched.
- [ ] Version-agnostic seed data is resolved from a single on-disk copy; no
      file is duplicated per version directory.
- [ ] Editing or seeding a configuration causes atlas-transports to reload
      **without a restart**, and the reload log shows the real tenant UUID.
- [ ] `00000000-0000-0000-0000-000000000000` never appears in an
      atlas-transports configuration-reload log.
- [ ] A regression test asserts configuration-status events carry all four
      tenant headers.
- [ ] Live route count equals configured route count for both the scheduled and
      instance registries — verified after a restart and with ≥2 replicas.
- [ ] A test asserts route ids are stable across repeated `ExtractRoute` calls
      and differ across tenants for the same slug.
- [ ] The existing duplicate entries are purged from `atlas-main`; all ten
      tenants show live counts equal to configured counts.
- [ ] The cleanup procedure is documented as a runbook in this task folder.
- [ ] End-to-end: on a tenant provisioned only through the Setup page UI, a
      character entering `270000100` via `out00` transforms and is warped to
      `200090510` with no "flight service is currently unavailable" message.
- [ ] All `CLAUDE.md` §Build & Verification gates pass, including
      `docker buildx bake` for every service with a touched `go.mod`.
