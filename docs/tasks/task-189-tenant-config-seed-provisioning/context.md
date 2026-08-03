# task-189 — Implementation Context

Companion to [`plan.md`](./plan.md). Everything here was read from the code on
this branch, not assumed. File:line references are to that state.

---

## 1. The three defects in one sentence each

1. **Nothing ever calls the seed endpoints.** Five `POST /tenants/{tenantId}/configurations/<res>/seed`
   routes exist (`configuration/resource.go:1259,1267,1275,1283,1291`); no caller
   exists in `deploy/k8s/`, `tools/`, or `services/atlas-ui/`. Every tenant ships
   with zero routes, vessels, and instance flights.
2. **Configuration-status events carry no tenant headers**, so atlas-transports'
   hot-reload path resolves the *zero* tenant and loads nothing.
3. **Route ids are minted fresh on every load**, so each replica/restart writes
   another full copy into the shared Redis registry (`24` live against `12`
   configured, on both scheduled and instance registries).

---

## 2. Five facts that constrain every decision

**F1 — `configurations` is one row per `(tenant, resource_name)`, not one row
per entry.** `Entity` (`configuration/entity.go:11-17`) holds
`resource_data jsonb`; the blob is a JSON:API document whose `data` is an
**array**. "12 routes" is 12 array elements inside 1 DB row. `Entity` embeds
`gorm.Model`, so deletes are soft and `updated_at` is available for the status
endpoint. `CreateRoute` (`processor.go:181-273`) read-modify-writes that array —
Task 5's `AppendConfigurationEntries` is that logic extracted once.

**F2 — the stored entry shape is `{id, type, attributes}`.** `TransformRoute`
(`configuration/rest.go:41-48`) reads `data["id"]` and `data["attributes"]`. The
pre-existing seed files at `services/atlas-tenants/configurations/**` are exactly
that shape *at top level* — **not** wrapped in `{"data": …}`, which is what
`seeder.ParseEnvelope` (`libs/atlas-seeder/jsonapi.go:20-40`) requires. Hence the
mechanical rewrap in Task 4.

**F3 — filename ≠ entity id.** `instance-routes/flight-temple-of-time-leafre.json`
has `"id": "temple-of-time-return-flight"`. The seeder `Subdomain` must therefore
return `EntityIDPattern() == nil` and take the id from `data.id`
(`libs/atlas-seeder/seed.go:140-152` handles exactly this case).

**F4 — the emit context is tenant-free, and both sides fail silently.**
`ProcessorImpl` is built as `NewProcessor(l, ctx, db)` with the server context
(`processor.go:170-176`); all 18 `…AndEmit` methods enqueue via
`outbox.EmitProvider(p.l, p.ctx, tx)`. `outbox.EnqueueBuffer` snapshots headers
from that ctx at enqueue time (`libs/atlas-outbox/bridge.go:20-24, 42-57`), and
`producer.TenantHeaderDecorator` returns **empty headers with a nil error** when
the ctx has no tenant (`libs/atlas-kafka/producer/header.go:34-37`). On the
receiving end, `consumer.TenantHeaderParser` builds a tenant from whatever it
found and installs it unconditionally, so absent headers become the **zero**
tenant rather than *no* tenant — which is why `tenant.MustFromContext` never
panicked and the reload ran against `00000000-0000-0000-0000-000000000000`.

**F5 — the catalog toolchain keys off a two-level `<region>/<version>` layout and
skips `_`-prefixed directories.** CI stamps `CATALOG_REVISION` with
`for dir in deploy/seed/*/*/` (`main-publish.yml:287`, `pr-validation.yml:619`,
`reconcile-bump-prs.yml:198`). `tools/catalog-lint` skips `_`-prefixed dirs at
both the region scan (`main.go:33`) and the walk (`main.go:55-57`), and only
classifies files at depth ≥ 4 (`main.go:70-73`). This is why the shared root is
`shared/all` and not `_shared`.

---

## 3. Key files by area

### `libs/atlas-seeder`
| File | Why it matters |
|---|---|
| `catalog.go:22-59` | `filesystemSource.Roots` *formats* `<region>/<major>_<minor>` — it never enumerates directories, so `shared/all` can't be mistaken for a tenant root (`all` is not `%d_%d`). |
| `seed.go:42-84` | `Seed` holds a per-`(tenant, group)` mutex for the whole run; subdomains within a group run concurrently via errgroup. |
| `seed.go:86-121` | `runSubdomain` — where the multi-root merge lands. |
| `seed.go:123-161` | `loadOne` — nil `EntityIDPattern` ⇒ entity id comes from `data.id`; the full file bytes are handed to `Decode`, not just `attributes`. |
| `subdomain.go:12-22` | The `Subdomain[J, M]` contract the three transport subdomains implement. |
| `handlers.go:47-71` | `postSeed` returns **202** and seeds in a background `routine.Go` with `bgCtx = tenant.WithContext(...)`. This is where `AfterSeed` fires. |
| `handlers.go:81-87` | Emits `seed catalog drift detected` when `CatalogRevision != *TenantSeededRevision`. This is why the shared root is opt-in: folding it into all nine consumers would trip this once, fleet-wide, for no benefit. |
| `status.go:22-26` | `ReadStatus` reads the revision from `roots[0]` today; must route through the same helper as `Seed`. |
| `state.go:13-21` | `SeedState` — the `(tenant_id, group_name)` table each consumer AutoMigrates. |

### `services/atlas-tenants`
| File | Why it matters |
|---|---|
| `configuration/entity.go:11-17` | The single-row-per-resource shape (F1). |
| `configuration/administrator.go:87-93` | `DeleteConfigurationByResourceName` — what the subdomain's `DeleteAllForTenant` calls. |
| `configuration/provider.go:15-22` | `GetByTenantIdAndResourceNameProvider` — the read side of append/count. |
| `configuration/processor.go:20-160` | The `Processor` interface; note every method threads `tenantId uuid.UUID`. |
| `configuration/processor.go:1151-1262` | The three bespoke `Seed*` methods being deleted. |
| `configuration/processor.go:1640-1652` | `CreateRankingsAndEmit` — verified during design to share the F4 defect. |
| `configuration/kafka.go:11-31,41-111` | Event type constants and the six `Create*StatusEventProvider` functions the `AfterSeed` hooks reuse. |
| `configuration/rest.go:10-22,362-372` | The three rest models gaining `Uuid`. |
| `configuration/resource.go:1246-1305` | `RegisterRoutes` — the CRUD patterns the new literal seed paths must not collide with. |
| `main.go:44-85` | Migration list and route initializers; the seed initializer goes *before* `configuration.RegisterRoutes`. |
| `kafka/message/*.go:46,63` | `message.Emit(p)(func(*Buffer) error) error` and `message.EmitWithResult[M, B]`. |

### `services/atlas-transports`
| File | Why it matters |
|---|---|
| `instance/config/rest.go:35-45` | `ExtractRoute` — no `SetId`, so `instance/builder.go:26` falls through to `uuid.New()`. |
| `transport/config/rest.go:41-58` | Same defect for scheduled routes (`transport/builder.go:32`). |
| `transport/config/rest.go:85-94` | `ExtractVessel` **does** call `SetId(v.Id)` — this is precisely why vessels never drifted, and why they stay untouched. |
| `instance/route_registry.go:33-38,73-81` | Registry keys on `route.Id()`; `AddTenant` is a per-route `Put`, `ClearTenant` a per-route `Remove`. Stable ids make `Put` idempotent but remove nothing — hence the reconcile. |
| `instance/config/processor.go:38-54`, `transport/config/processor.go:47-76` | `requests.DrainProvider` takes the mapper as a **value** (`libs/atlas-rest/requests/paged.go:113`), which is what lets `ExtractRouteFor(l, t)` be tenant-aware. |
| `kafka/consumer/configuration/consumer.go:41-72` | The reload handler: reads the tenant from **context** (not `e.TenantId`), and today clears **before** loading. |
| `main.go:98-120` | The bootstrap loop; today it falls through to `AddTenant` with empty slices on a load error — harmless while additive, destructive under clear-then-add. |

### `services/atlas-ui`
| File | Why it matters |
|---|---|
| `services/api/seed.service.ts:74-91,152-170` | The generic `SeedStatus` shape and `subdomainCount(s, key)` projection helper. |
| `services/api/seed.service.ts:173-175` | `seedDrops` — the 202 + poll-status idiom the three new seed methods mirror. |
| `lib/hooks/api/useSeed.ts:39-58` | Query-key convention (`[name, tenantId] as const`). |
| `lib/hooks/api/useSeed.ts:256-267` | Status-query convention: `enabled: !!activeTenant, staleTime: 0, refetchInterval: 5000`. |
| `pages/SetupPage.tsx:176-272` | The `SeedRow` interface and the eight existing rows; `formatBadge` is a thunk closing over its own `status.data` so the `map()` loop stays uniform. |

### Deploy / tooling
| File | Why it matters |
|---|---|
| `deploy/k8s/base/components/seed-catalog/` | The label-selected component: git-sync sidecar + `/var/run/seed-catalog` mount + `SEED_CATALOG_ROOT=/var/run/seed-catalog/catalog/deploy/seed`. |
| `deploy/k8s/base/atlas-drop-information.yaml:7` | The exact label form to copy: `atlas.seed-catalog: "true"`. |
| `deploy/shared/routes.conf:512` | `^/api/tenants(/.*)?$` already proxies to atlas-tenants with `$request_uri` intact — **no ingress or `tools/gen-routes.sh` change is needed**. |
| `Dockerfile:47,77` | `libs/atlas-seeder` is already COPY'd (mod block + source block) — the bake is confirmation, not a change. |
| `Dockerfile:138,163` | `COPY /app/configurations /configurations` — **stays**, because `rps-rewards`/`mts-configs` still read from the image. |
| `tools/catalog-lint/subdomains.go:12-28` | Where the three new `pattern: nil` rules go. |

---

## 4. Decisions already made (don't relitigate)

| Open question | Decision | Where |
|---|---|---|
| Where does shared seed data live? | `deploy/seed/shared/all/` in the git-sync'd catalog; the three source dirs are deleted from the image. | design §3.1, plan Task 4 |
| Shared root name and precedence? | `shared/all`, returned **first** so the version-specific root wins. Not `_shared`: catalog-lint and all three CI stampers skip `_`-prefixed dirs (F5). | design §3.1, plan Task 1/4 |
| Derived or stored UUID? | **Derived.** `uuid.NewSHA1(tenantId, "<resource>/<slug>")`, homed once in `libs/atlas-tenant.DerivedId`. No migration, no backfill, correct for rows that already exist in `atlas-main`. | design §3.7, plan Task 3 |
| Does FR-3.5 land here? | Partially. Producer-side WARN (atlas-outbox) and consumer-side ERROR guard (atlas-transports) land. Changing `TenantHeaderDecorator`/`TenantHeaderParser` is **deferred** — both are hot paths in 59 services and the parser change would convert a silent zero-tenant into a `MustFromContext` panic fleet-wide. Rationale recorded in plan Task 9. | design §3.6 |
| What is `CATALOG_REVISION` with two roots? | Plain `+`-join in root order (`<sharedSha>+<versionSha>`), not a hash, so it stays debuggable in a log line. Single-root callers get the identical string as before. | design §3.2, plan Task 1 |
| Does `rankings` share the F4 defect? | **Yes, verified.** `CreateRankingsAndEmit` (`processor.go:1640-1652`) uses the same tenant-free `p.ctx`. All 18 `…AndEmit` methods are fixed. | design §3.5, plan Task 7 |
| One event per file or per group? | **Per group.** The consumer's handler is `ClearTenant()` + full reload, so 12 events would trigger 12 reloads and a reload could land mid-seed on a partial set. | design §3.4, plan Task 6 |
| Rewrite the processor to read tenant from ctx? | **No.** It is the right end-state but rewrites a 1731-line processor, a 1305-line resource file, and their tests for a defect a 5-line helper closes — and the REST layer is path-scoped, so handlers would build the context anyway. Noted as future cleanup. | design §3.5, plan Task 7 |
| Teach the seeder a "write once" mode? | **No.** Changing the `Subdomain` contract for all nine consumers to serve one storage shape isn't worth 12 sequential JSONB updates on a rarely-run admin operation. | design §3.3 |
| Three Setup rows or one grouped row? | **Three**, as FR-2.1 mandates. The partial-seed hazard is documented in plan Task 12 as a cheap follow-up. | design §3.9, plan Task 12 |

---

## 5. Deliberate additions beyond the literal design text

- **The consumer's reload is reordered to load-then-clear-then-add** (plan Task 11),
  not just guarded. `consumer.go:48,61` clears *before* loading today, so a load
  failure already leaves the tenant with zero routes. It is the same defect
  design §3.8 identifies in `main.go`, in the same delivery.
- **`VesselRestModel` also gains `Uuid`** (plan Task 8). Design §3.7 says "vessels
  are not affected" for *registry identity* (true — `ExtractVessel` already sets a
  stable slug id), but PRD §5 "Modified" explicitly lists the vessel read model
  among those that must surface a stable UUID. It is additive and costs nothing.

---

## 6. Task dependency order

```
1 (seeder shared root) ─┐
2 (seeder AfterSeed)  ──┤
5 (tenants append/count)┼─> 6 (tenants adopts seeder) ─> 7 (tenant emit ctx)
4 (catalog data move) ──┘
3 (tenant.DerivedId) ─────> 8 (tenants exposes uuid) ─┐
                       └──> 10 (transports uses uuid) ┼─> 11 (reconcile + guard)
9 (outbox WARN) ───────────────────────────────────────┘
                                                        12 (atlas-ui)
                                                        13 (deploy + runbook + gates)
```

Tasks 6 and 7 both edit `configuration/processor.go` — run 6 first (it deletes
the three `Seed*` methods) so 7 doesn't have to transform code that is about to
disappear. Tasks 9 and 12 are independent and can run any time. Task 13 must be
last: it is the full-sweep gate.

---

## 7. Traps

- **Do not run `go work sync`** for the atlas-tenants `go.mod` change — use
  `go mod edit` + `go mod tidy` (see the go-workspace-guard-footguns note).
- **`libs/atlas-seeder`'s nine existing consumers must see zero behavioural
  change.** `NewFilesystemCatalogSource` keeps returning exactly one root, and
  `revisionFor` over one root yields the identical string. Task 1 has a test for
  each.
- **Putting a tenant into the configuration processor's context activates
  `libs/atlas-database`'s tenant callback** for `configurations` (it has a
  `tenant_id` column). The injected `tenant_id = <ctx tenant>` predicate is
  identical to the explicit one the existing code already passes, because the
  context tenant is built from the same path-scoped `tenantId`. Task 7 Step 6
  is where a mismatch would surface.
- **The three `Transform*` functions have twelve call sites** across
  `resource.go` (four each). The grep in Task 8 Step 4 is what catches a miss.
- **`configuration/seed.go` keeps `LoadRpsRewardFiles`/`LoadMtsConfigFiles` and
  their `/configurations/...` default paths.** Deleting the whole file, or the
  whole `services/atlas-tenants/configurations/` tree, breaks the two
  out-of-scope seeders.
- **`docker buildx bake atlas-tenants atlas-transports` is mandatory**, not
  optional — `go build` against `go.work` cannot catch a missing `COPY libs/...`
  line in the shared Dockerfile.
- **atlas-ui needs `npm run build`, not just `npm run test`** — `tsc -b` is what
  catches type errors vitest doesn't.
- The `packet-*` playbooks and matrix tooling are **not** involved in this task;
  no packet, opcode, or template change is in scope.

---

## 8. Live-acceptance checks (post-deploy, from the runbook)

- On a tenant provisioned only through the Setup page UI, a character entering
  map `270000100` via portal `out00` transforms and warps to `200090510` with no
  "The flight service is currently unavailable" message.
- `00000000-0000-0000-0000-000000000000` appears in **no** atlas-transports
  configuration-reload log line.
- Live route counts equal configured counts (12 / 6 / 12) for all ten tenants,
  after a restart and with ≥2 replicas, on **both** registries.
