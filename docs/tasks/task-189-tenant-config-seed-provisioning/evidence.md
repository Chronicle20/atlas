# task-189 — Diagnostic Evidence

Captured 2026-08-03 against the `atlas-main` namespace. This records the live
observations behind the PRD so the design phase does not have to re-derive them.

---

## Reproduction

Tenant `ec876921-c363-4cc6-9c51-5bb8d57f9553` (GMS v83), character 19.

1. Stand in map `270000100` (Temple of Time).
2. Enter portal `out00`.
3. Observed: character transforms into a draco and receives the buff (correct),
   then the client shows **"The flight service is currently unavailable. Please
   try again later."**
4. Expected: warp to `200090510` (the route's first transit map).

## Failure chain

**1 — Portal action fires correctly.**
`deploy/seed/gms/83_1/portal-actions/portals/portal-outTemple.json:13-26` defines
two operations: `apply_consumable_effect(itemId 2210016)` — the draco
transformation, which worked — followed by
`start_instance_transport(routeName: temple-of-time-return-flight)` with
`failureMessage` set to the observed string.

**2 — The saga cannot resolve the route name.**
`atlas-saga-orchestrator-6cfcb48969-7t4sl`, 2026-08-03T22:32:05Z:

```
Issuing [GET] request to
  http://atlas-ingress.atlas-main.svc.cluster.local:80/api/transports/instance-routes?page[number]=1&page[size]=250

error: "route not found: temple-of-time-return-flight"
error_code: "TRANSPORT_ROUTE_NOT_FOUND"
message: "Instance transport failed - emitting failure event"
```

The failure emits a `send_message` saga (`initiated_by: portal-action-transport-failure`)
carrying `PINK_TEXT` with the exact observed string.

**3 — atlas-transports holds no routes.**

```
GET /api/transports/instance-routes  (v83 tenant headers)  →  "total": 0
```

**4 — Because the source configuration is empty.**

```
GET /api/tenants/ec876921-.../configurations/instance-routes  →  "total": 0
```

**5 — And it is empty for every tenant and every resource.**
Pre-seed counts, all ten tenants:

```
v61    routes=0 vessels=0 instance-routes=0
v72    routes=0 vessels=0 instance-routes=0
v84    routes=0 vessels=0 instance-routes=0
v87    routes=0 vessels=0 instance-routes=0
v79    routes=0 vessels=0 instance-routes=0
jms185 routes=0 vessels=0 instance-routes=0
v95    routes=0 vessels=0 instance-routes=0
v92    routes=0 vessels=0 instance-routes=0
v48    routes=0 vessels=0 instance-routes=0
v83    routes=0 vessels=0 instance-routes=0
```

`rps-rewards` was also `0` on every tenant.

---

## Bug 1 — nothing calls the seed endpoints

Five seed endpoints are registered in
`services/atlas-tenants/atlas.com/tenants/configuration/resource.go`:

| Line | Endpoint |
|---|---|
| 1259 | `POST /tenants/{tenantId}/configurations/routes/seed` |
| 1267 | `POST /tenants/{tenantId}/configurations/vessels/seed` |
| 1275 | `POST /tenants/{tenantId}/configurations/instance-routes/seed` |
| 1283 | `POST /tenants/{tenantId}/configurations/rps-rewards/seed` |
| 1291 | `POST /tenants/{tenantId}/configurations/mts-configs/seed` |

Searches for callers found none in `deploy/k8s/`, `tools/`, or
`services/atlas-ui/`. `services/atlas-ui/src/services/api/seed.service.ts`
covers drops, gachapons, NPC conversations, quest conversations, NPC shops,
portal scripts, reactor scripts, and map action scripts — all script/content
services, none of the tenant-configuration resources.

Seed files ship in the atlas-tenants image at `/configurations/...` per
`Dockerfile:138,163`.

## Bug 2 — configuration-status events carry no tenant headers

After seeding, atlas-transports received the events and logged them with the
**correct** tenant from the event body, then reloaded against the **nil** tenant:

```
Configuration [instance-route] event [INSTANCE_ROUTE_CREATED] for resource [hak-to-orbis],
  reloading instance routes for tenant [ec876921-c363-4cc6-9c51-5bb8d57f9553].
Loading instance route configurations for tenant [00000000-0000-0000-0000-000000000000]
Loaded [0] instance routes for tenant [00000000-0000-0000-0000-000000000000]
```

Mechanism:

- `services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer.go:58`
  reads the tenant from the **context** (`tenant.MustFromContext`), populated by
  `consumer.TenantHeaderParser` from Kafka headers — not from `e.TenantId` in the body.
- `libs/atlas-kafka/producer/provider.go:16-17` — `ProviderImpl` always applies
  `TenantHeaderDecorator(ctx)`, so the producer *would* attach headers if the
  context had a tenant.
- `libs/atlas-kafka/producer/header.go:31-37` — `TenantHeaderDecorator` calls
  `tenant.FromContext(ctx)` and on error does `return headers, nil`: **empty
  headers, nil error**. A silent drop.
- `services/atlas-tenants/atlas.com/tenants/configuration/processor.go:24-91` —
  the whole configuration processor interface threads `tenantId uuid.UUID`, and
  `NewProcessor(l, ctx, db)` receives a tenant-free server context. So no
  `tenant.Model` is ever in the emit context.

Net effect: the hot-reload path has never worked. Only the startup bootstrap at
`services/atlas-transports/atlas.com/transports/main.go:101-120` — which iterates
tenants and builds a per-tenant context explicitly — loads routes.

Consumers of `EVENT_TOPIC_CONFIGURATION_STATUS`: **atlas-transports only**
(atlas-tenants produces). `rps-rewards`, `mts-configs`, and `rankings` events
have no consumer at all.

## Bug 3 — route ids are minted fresh on every load

After seeding and restarting atlas-transports (2 replicas), live counts were
**double** the configured counts:

```
v83 live instance-routes=24 routes=24     (configured: 12 and 12)
```

Confirmed on jms185, v95, v92, v48, v83.

Mechanism — neither `ExtractRoute` calls `SetId`, so both builders fall through
to `uuid.New()`:

| File | Line | Detail |
|---|---|---|
| `services/atlas-transports/atlas.com/transports/instance/config/rest.go` | 34 | `ExtractRoute` → `NewRouteBuilder(r.Name)`, no `SetId` |
| `services/atlas-transports/atlas.com/transports/instance/builder.go` | 26 | `id: uuid.New()` |
| `services/atlas-transports/atlas.com/transports/transport/config/rest.go` | 42 | `ExtractRoute` → `NewBuilder(r.Name)`, no `SetId` |
| `services/atlas-transports/atlas.com/transports/transport/builder.go` | 32 | `id: uuid.New()` |

The registry keys on `route.Id()`
(`services/atlas-transports/atlas.com/transports/instance/route_registry.go:36`),
so each replica writes its own 12 UUIDs into the shared Redis registry. The count
grows by (configured × replicas) on every restart.

Consequence beyond the count: two distinct route objects share one `name`, and
`GetRouteByName` in the saga-orchestrator resolves to one arbitrarily — so
boarders can split across "identical" routes. The Temple of Time flight has
`capacity: 1`, which masks this today.

Note the JSON:API resource id from atlas-tenants is the **slug**
(`hak-to-orbis`, `temple-of-time-return-flight`) — `InstanceRouteRestModel.Id`
is a `string` with `json:"-"`. No UUID is currently exposed.

---

## Remediation already applied to `atlas-main`

These were live operator actions on 2026-08-03, **not** code changes. The task
branch must still land the permanent fixes.

1. Seeded `routes`, `vessels`, and `instance-routes` for all ten tenants via the
   path-scoped endpoints. Every call reported
   `{"deletedCount":0,"createdCount":N,"failedCount":0}` with N = 12 / 6 / 12,
   matching the file counts in `services/atlas-tenants/configurations/`.
2. Restarted `atlas-transports` (required because of Bug 2) so the startup
   bootstrap would load the newly-seeded configuration.

Post-restart, `temple-of-time-return-flight` resolves for GMS v83. **The
duplicate registry entries described in Bug 3 remain in place** and are what
FR-5 must clean up.

## Deliberately not investigated

`atlas-transports` logs show `COMMAND_TOPIC_INSTANCE_TRANSPORT-main` wedging and
recreating its reader on a ~3 minute cycle (reached "attempt 997"). The same
pattern appears on every topic the service consumes
(`EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_MAP_STATUS`,
`EVENT_TOPIC_CONFIGURATION_STATUS`, `EVENT_TOPIC_CHANNEL_STATUS`), which is
consistent with an idle-topic heartbeat rather than a fault. **This is a
hypothesis, not a finding** — it was not verified and is out of scope for
task-189.
