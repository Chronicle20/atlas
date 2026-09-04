# Service Scaffolding Checklist

When scaffolding a new Atlas service, complete ALL of these steps. Do not skip any.

> **MANDATORY companion:** `docs/adding-a-new-service.md` (repo root) is the
> canonical checklist of every file a new service must be enumerated in —
> CI lists, `docker-bake.hcl`, `go.work`, the k8s base, BOTH kustomize
> overlays (main + pr), database creation, and ingress. Several of those
> fail *silently* when missed (unpinned `:latest` image, dropped topic env
> vars, unsuffixed Kafka topics). Work through that doc in full; this file
> only covers the code-level scaffolding.

## 1. Build & CI registration
Covered by `docs/adding-a-new-service.md` §1: `.github/config/services.json`,
`docker-bake.hcl` (hand-synced!), `go.work`. There is NO per-service
Dockerfile — the repo-root `Dockerfile` is shared and parameterized by
`ARG SERVICE`; verify with `docker buildx bake atlas-<service>`.

## 2. Kubernetes wiring
Covered by `docs/adding-a-new-service.md` §2–§6: base manifest at
`deploy/k8s/base/atlas-<service>.yaml` (no `namespace:` — overlays set it;
`DB_NAME` gets the unsuffixed base value), base `kustomization.yaml`
resources entry, base `env-configmap.yaml` topic vars, the main overlay's
four enumerations (db-name-suffix patch, ATLAS_ENV patch, `images:` pin,
topic literals), the pr overlay's five, and database creation.

## 3. Bruno Collection (REST services only)
**Directory:** `services/atlas-<service>/.bruno/`

Minimum files:
```
.bruno/
├── bruno.json
├── collection.bru
└── environments/
    ├── Local.bru
    ├── Local Debug.bru
    └── Atlas - K3S.bru
```

**bruno.json:**
```json
{
  "version": "1",
  "name": "atlas-<service>",
  "type": "collection",
  "ignore": ["node_modules", ".git"]
}
```

**collection.bru:**
```
headers {
  TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
  REGION: GMS
  MAJOR_VERSION: 83
  MINOR_VERSION: 1
}
```

**environments/Local.bru:**
```
vars {
  host: localhost
  port: 8080
  scheme: http
}
```

**environments/Local Debug.bru:**
```
vars {
  host: localhost
  port: 8081
  scheme: http
}
```

**environments/Atlas - K3S.bru:**
```
vars {
  host: atlas-nginx
  port: 80
  scheme: http
}
```

Optionally add sample request `.bru` files for the service's endpoints.

## 4. REST Handler Scaffolding (REST services only)
**File:** `services/atlas-<service>/atlas.com/<svc>/rest/handler.go`

`rest/handler.go` declares only aliases over `libs/atlas-rest/server` — copy the
pattern from `services/atlas-mts/atlas.com/mts/rest/handler.go`, not from
an older service:

- `HandlerDependency`, `HandlerContext`, and `GetHandler` alias the shared
  scaffolding types.
- `InputHandler[M]` is added only if the service has an input handler.
- `var RegisterHandler = server.RegisterHandler`, and, where needed, the
  generic `RegisterInputHandler` wrapper.
- Do not declare a local `ParseInput` wrapper — no service calls one.

A DB-backed service closes its `*gorm.DB` over the handler constructor —
`func handleX(db *gorm.DB) rest.GetHandler` — and never puts it on
`HandlerDependency`. Reference:
`services/atlas-configurations/atlas.com/configurations/environments/resource.go:45`.

Service-specific path-parameter helpers stay in the service's `rest` package
and delegate to `server.ParseIntId` / `server.ParseUUIDId` /
`server.ParseStringId` when the path segment is a plain int, UUID, or string.

## 5. Ingress Route (REST services only)
**File:** `deploy/shared/routes.conf`

Add a location block **alphabetically** in the shared routes file (single-sourced for K8s and compose):
```nginx
location ~ ^/api/<service-path>(/.*)?$ {
  proxy_pass http://atlas-<service>:8080;
}
```

After editing, run `tools/gen-routes.sh` to regenerate
`deploy/k8s/base/routes.conf.template.generated` from the shared source, and
commit both files. (`deploy/scripts/sync-k8s-ingress-routes.sh` is **dead** — it
targets a `deploy/k8s/ingress.yaml` that no longer exists. Do not run it.)

## 6. Tenant Opcode Template (atlas-channel packet writers/handlers only)
**File:** `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`

Atlas tenants are seeded from these JSON templates the first time they are created. If your service introduces new packet writers or recv handlers in `atlas-channel` (i.e., the change touches `libs/atlas-packet/character/{clientbound,serverbound}/<feature>/` or registers new `Writer`/`Handler` constants in `services/atlas-channel/atlas.com/channel/main.go`), seed the corresponding opcode rows in **every targeted template** so fresh tenants get the mappings without manual operator action.

Two top-level arrays:

- `handlers[]` — recv side. Add an entry with `opCode`, `validator`, and `handler` name (the constant string registered in `main.go`):
  ```json
  {
    "opCode": "0x39",
    "validator": "LoggedInValidator",
    "handler": "MonsterBookCover"
  }
  ```
- `writers[]` — send side. Add `opCode` + `writer` name:
  ```json
  { "opCode": "0x53", "writer": "MonsterBookSetCard" }
  ```

Insert each entry in numeric order alongside neighbouring opcodes. Match the indentation and trailing-comma style of adjacent entries; the file is plain JSON and must remain valid (`python3 -m json.tool` validates).

If the feature targets a single client version (e.g. v83-only), only that template needs the entries — but document the scope decision in the design doc so future client-version expansions know to add them.

Operators creating a tenant from a snapshot taken before this change still need the rows applied via `atlas-tenants` admin; the seed templates only affect tenants instantiated post-merge.

## 7. Post-Scaffold Verification
After scaffolding is complete:
1. Run `tools/service-registration-guard.sh` (machine-checks every registration list; also a CI job), then the remaining commands in `docs/adding-a-new-service.md` §Verification (overlay renders, ghcr tag existence, bake build)
2. `/service-doc` — generates/verifies service documentation
3. `/backend-audit` — audits against Atlas backend developer guidelines

## Database & Tenant Filtering Notes
- `database.Connect()` automatically registers GORM tenant-filtering callbacks — do NOT add `RegisterTenantCallbacks` to `main.go`
- Providers do NOT take `tenantId` — tenant filtering is automatic via `db.WithContext(ctx)`
- Only `create` functions need `tenantId` (to set the entity field)
- Test files using SQLite directly must call `database.RegisterTenantCallbacks(l, db)` after `gorm.Open()`
- Entity structs should use `TenantId` (not `TenantID`) for field naming consistency

## Conditional Steps
- Steps 3, 4, and 5 only apply to services that expose REST endpoints. Kafka-only services skip Bruno, REST handler scaffolding, and ingress.
- Step 6 only applies when the change introduces new atlas-channel packet writers or recv handlers. Pure-REST services and Kafka-only services skip the opcode template seed.

---

## Audit verification — SCAFFOLD-01..10

Rule IDs are defined in [audit-checklist.md](audit-checklist.md). These checks
trigger when the diff:

- adds a `services/atlas-<service>/` directory — detect with
  `git diff --name-status <base>..HEAD | awk '$1 == "A" && $2 ~ /^services\/atlas-[^/]+\/.+\/main\.go$/'`; **or**
- registers a new `Writer` / `Handler` constant in
  `services/atlas-channel/atlas.com/channel/main.go`, or adds a package under
  `libs/atlas-packet/character/{clientbound,serverbound}/<feature>/`
  (SCAFFOLD-07 only).

Canonical source for the registration lists is `docs/adding-a-new-service.md`;
this section is the audit's verification form of it.

| ID | How to verify | Pass criteria |
|----|---------------|---------------|
| SCAFFOLD-01 | `jq '.services[] \| select(.name == "atlas-<service>")' .github/config/services.json` | Returns a non-empty object with `type: "go-service"`. CI's change detection reads this file; without an entry the service never builds. |
| SCAFFOLD-02 | `test -f deploy/k8s/base/atlas-<service>.yaml` and grep the filename in `deploy/k8s/base/kustomization.yaml` `resources:` | Base manifest exists with `Deployment` + `Service`, no `namespace:` (overlays set it), unsuffixed `DB_NAME`, `containerPort: 8080`, db creds from the `db-credentials` secret — and it is listed in the base kustomization. |
| SCAFFOLD-03 | Grep `"atlas-<service>"` in `docker-bake.hcl`'s `go_services` list and `./services/atlas-<service>/atlas.com/<svc>` in `go.work`. | Both present. Go services are built from the shared repo-root `Dockerfile`, parameterized by `ARG SERVICE` (see [patterns-deploy.md](patterns-deploy.md)); a new Go service needs no Dockerfile of its own, and adding one is a finding. `docker-bake.hcl` is hand-synced with services.json; adding to one does not add to the other. |
| SCAFFOLD-04 | `grep -F "atlas-<service>:" deploy/shared/routes.conf` | At least one `location` block routes to the service, using the bare container name. Skip for Kafka-only services (no `rest/` package, no REST handlers in `main.go`). |
| SCAFFOLD-05 | Re-run `tools/gen-routes.sh` and `git diff --exit-code deploy/k8s/base/routes.conf.template.generated` | Exit 0 — the generated template is in sync with `deploy/shared/routes.conf` and both are committed. |
| SCAFFOLD-06 | `grep -F "atlas-<service>:" deploy/compose/docker-compose.core.yml` | Service block exists alongside peers. |
| SCAFFOLD-07 | For each new `Writer` / `Handler` constant, grep its name in the targeted `services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`. | Each new `Writer` appears as a `"writer": "<Name>"` row in `writers[]`; each new recv `Handler` appears as a `"handler": "<Name>"` row in `handlers[]`. The targeted client version(s) must match what the design doc declared. Pure-REST and Kafka-only services skip this check. |
| SCAFFOLD-08 | `test -d services/atlas-<service>/.bruno && test -f services/atlas-<service>/.bruno/bruno.json` | Directory exists with `bruno.json`, `collection.bru`, and an `environments/` directory. Skip for Kafka-only services. |
| SCAFFOLD-09 | `tools/service-registration-guard.sh` | Exit 0. This structurally checks the enumerations that fail *silently* when missed: both overlays' `images:` pins, the main `ATLAS_ENV` and db-name-suffix patches, `ATLAS_DB_NAMES`, `tools/db-bootstrap.sh`, base kustomization membership, and atlas-env key parity between base and overlays. Exit 2 means "cannot verify" (fail-closed) — record that as a FAIL, not a PASS. |
| SCAFFOLD-10 | `grep -c '= server.HandlerDependency' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, `grep -c 'type HandlerDependency struct' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, `grep -c 'gorm.io/gorm' services/atlas-<service>/atlas.com/<svc>/rest/handler.go`, and `grep -rc 'd\.DB()' services/atlas-<service>/` | First is ≥1; the other three are 0. `rest/handler.go` aliases the shared scaffolding (`libs/atlas-rest/server`) and declares no scaffolding of its own. A DB-backed service closes its `*gorm.DB` over the handler constructor — `func handleX(db *gorm.DB) rest.GetHandler` — and never puts it on `HandlerDependency`. Skip for Kafka-only services (no `rest/` package). |
