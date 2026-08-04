# task-189 Runbook — Transport Configuration Provisioning & Registry Convergence

Covers the rollout of task-189, the post-deploy verification that the
duplicate Redis registry entries are gone, and the manual fallback if the
automatic reconcile does not converge.

## What changes at deploy time

1. **atlas-tenants** gains a git-sync sidecar and reads its transport seed
   data from `deploy/seed/shared/all/` instead of the image. It also runs
   the `seed_state` AutoMigrate
   (`services/atlas-tenants/atlas.com/tenants/main.go`). Existing
   `configurations` rows are untouched and unchanged in shape.

   The sidecar comes from the existing
   `deploy/k8s/base/components/seed-catalog` component. It attaches because
   `deploy/k8s/base/atlas-tenants.yaml` now carries the
   `atlas.seed-catalog: "true"` Deployment label the component's
   `labelSelector` matches. The same file also declares `volumes: []` on the
   pod spec and `volumeMounts: []` on the `atlas-tenants` container — the
   component patches are JSON `add` operations against
   `/spec/template/spec/volumes/-` and
   `/spec/template/spec/containers/0/volumeMounts/-`, and kustomize fails the
   whole build ("doc is missing path") when those arrays are absent.
2. **atlas-transports** reconciles every tenant's route registries at
   startup: load → (on success) clear → add. Because route ids are now
   derived and stable, this converges each tenant to exactly the
   configured set and permanently removes the random-id duplicates.
3. **atlas-ui** gains three Setup rows: Transport Routes, Transport
   Vessels, Instance Transport Routes.

## Rollout order

1. Merge. `main-publish` stamps `CATALOG_REVISION` into every
   `deploy/seed/<region>/<version>/` directory — including
   `deploy/seed/shared/all/` — and bumps atlas-tenants, atlas-transports,
   and atlas-ui.
2. Let atlas-tenants roll first. Confirm the git-sync sidecar is running
   and the catalog is mounted:

   ```
   kubectl -n atlas-main get pod -l app=atlas-tenants \
     -o jsonpath='{.items[0].spec.containers[*].name}'
   kubectl -n atlas-main exec deploy/atlas-tenants -c atlas-tenants -- \
     ls /var/run/seed-catalog/catalog/deploy/seed/shared/all
   ```

   Expected: a `git-sync` container is present, and the listing shows
   `CATALOG_REVISION`, `instance-routes`, `routes`, `vessels`.
3. Let atlas-transports roll. Its bootstrap reconcile runs per tenant. If
   atlas-tenants has not finished rolling, atlas-transports derives the
   same ids locally and logs a WARN per route — that is expected during
   the window and self-corrects on the next reload. The WARN reads
   `Route [<slug>] has no uuid attribute for tenant [<id>]; deriving
   locally.` (and the `Instance route [...]` twin).

## Post-deploy verification (FR-5.3)

For every tenant, the live registry count must equal the configured
count. Configured counts on this branch are 12 routes, 6 vessels, and 12
instance routes (`deploy/seed/shared/all/{routes,vessels,instance-routes}`).

For each tenant, with that tenant's four headers set:

```
GET /api/tenants/configurations/routes/seed/status
GET /api/tenants/configurations/vessels/seed/status
GET /api/tenants/configurations/instance-routes/seed/status
```

and compare `subdomains.<resource>.count` against `meta.total` on:

```
GET /api/transports/routes             -> meta.total
GET /api/transports/instance-routes    -> meta.total
```

Both transports endpoints are paginated JSON:API; the collection size is
the top-level `meta.total`, not the length of `data` (which is capped by
`page[size]`). There is no live-registry endpoint for vessels — vessels
are held on the scheduled routes, so the routes count covers them.

A live `total` of **twice** the configured count means the reconcile did
not run — check that the atlas-transports pods actually restarted on the
new image. A live `total` **below** the configured count means the load
failed and the reconcile correctly skipped; check the atlas-transports
log for `leaving the ... registry untouched`.

The nil-tenant signature must appear in **no** atlas-transports
configuration-reload log line. It is not a bare uuid in the log text —
grep the two explicit messages instead:

```
# atlas-transports (consumer refused to act on a header-less event)
... arrived without tenant headers; skipping reload.

# atlas-tenants (producer enqueued a header-less message)
Enqueuing message without tenant headers; downstream consumers will resolve the nil tenant.
```

Either line is a regression, not a cosmetic issue: it means a
configuration-status event reached the consumer without tenant headers.

## Provisioning a new or restored tenant

Open Setup with the tenant selected and click Seed on all three transport
rows. Seed **both** Transport Routes and Transport Vessels — a scheduled
transport needs both to compute a schedule, and seeding only one leaves
the tenant quietly half-configured. Order does not matter: seeding either
triggers a full scheduled-transport reload (the atlas-transports consumer
treats `route` and `vessel` status events identically), so the second
seed converges.

The badges should read 12 routes / 6 vessels / 12 routes once each
background seed completes (the page polls status every 5 seconds).

## Manual fallback

Only if the automatic reconcile does not converge after a confirmed
restart on the new image:

1. Scale atlas-transports to 0 replicas.
2. Delete the tenant's registry keys (`instance-route:*` and
   `transport-route:*` under that tenant's registry prefix). Note that
   `tools/redis-key-guard.sh` bans keyed Redis commands in service code
   for good reason — this is an operator action, not something to
   automate into a service.
3. Scale atlas-transports back up. The bootstrap reconcile repopulates
   from configuration.

## Known startup window

Clear-then-add leaves a brief window during startup where a tenant's
registry is partial. It is bounded by the reconcile loop and acceptable
for a startup path. Concurrent replicas converge regardless of
interleaving, because both write the same stable ids. Note that a restart
already recomputes scheduled-route schedules — the 1-second
`UpdateRoutes()` tick in every replica is last-writer-wins today.
