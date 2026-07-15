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

## 4. Ingress Route (REST services only)
**File:** `deploy/shared/routes.conf`

Add a location block **alphabetically** in the shared routes file (single-sourced for K8s and compose):
```nginx
location ~ ^/api/<service-path>(/.*)?$ {
  proxy_pass http://atlas-<service>:8080;
}
```

After editing, run `./deploy/scripts/sync-k8s-ingress-routes.sh` to regenerate the inlined K8s ConfigMap in `deploy/k8s/ingress.yaml`.

## 5. Tenant Opcode Template (atlas-channel packet writers/handlers only)
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

## 6. Post-Scaffold Verification
After scaffolding is complete:
1. Run the verification commands in `docs/adding-a-new-service.md` §Verification (overlay renders, configmap key diff, hand-maintained list greps, bake build)
2. `/service-doc` — generates/verifies service documentation
3. `/backend-audit` — audits against Atlas backend developer guidelines

## Database & Tenant Filtering Notes
- `database.Connect()` automatically registers GORM tenant-filtering callbacks — do NOT add `RegisterTenantCallbacks` to `main.go`
- Providers do NOT take `tenantId` — tenant filtering is automatic via `db.WithContext(ctx)`
- Only `create` functions need `tenantId` (to set the entity field)
- Test files using SQLite directly must call `database.RegisterTenantCallbacks(l, db)` after `gorm.Open()`
- Entity structs should use `TenantId` (not `TenantID`) for field naming consistency

## Conditional Steps
- Steps 3 and 4 only apply to services that expose REST endpoints. Kafka-only services skip Bruno and ingress.
- Step 5 only applies when the change introduces new atlas-channel packet writers or recv handlers. Pure-REST services and Kafka-only services skip the opcode template seed.
