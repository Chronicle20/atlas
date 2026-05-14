# Service Scaffolding Checklist

When scaffolding a new Atlas service, complete ALL of these steps. Do not skip any.

## 1. GitHub Actions — services.json
**File:** `.github/config/services.json`

Add entry to the `services` array:
```json
{
  "name": "atlas-<service>",
  "type": "go-service",
  "path": "services/atlas-<service>",
  "module_path": "services/atlas-<service>/atlas.com/<service>",
  "docker_image": "ghcr.io/chronicle20/atlas-<service>/atlas-<service>",
  "docker_context": "."
}
```
Both workflows (`main-publish.yml`, `pr-validation.yml`) dynamically read this file — no YAML changes needed.

## 2. Kubernetes Manifest
**File:** `deploy/k8s/atlas-<service>.yaml`

Two resources: Deployment + Service. Pattern:
```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-<service>
  namespace: atlas
spec:
  replicas: 1
  selector:
    matchLabels:
      app: atlas-<service>
  template:
    metadata:
      labels:
        app: atlas-<service>
    spec:
      containers:
      - name: <service>
        image: ghcr.io/chronicle20/atlas-<service>/atlas-<service>:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: DB_NAME
          value: "atlas-<service>"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: DB_USER
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: DB_PASSWORD
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-<service>
  namespace: atlas
spec:
  selector:
    app: atlas-<service>
  ports:
  - protocol: TCP
    port: 8080
```

## 3. Dockerfile
**File:** `services/atlas-<service>/Dockerfile`

Multi-stage Go build. Key points:
- Builder: `golang:1.25.5-alpine3.21`
- Runtime: `alpine:3.23`
- Copy lib module defs first (dependency caching), then create `go.work`, then `go mod download`, then copy source, then build
- Libs to include: `atlas-constants`, `atlas-kafka`, `atlas-model`, `atlas-rest`, `atlas-tenant`
- Output binary: `/server`, expose 8080
- Copy `config.yaml` if present
- Install `libc6-compat` in runtime image

## 4. Bruno Collection (REST services only)
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

## 5. Ingress Route (REST services only)
**File:** `deploy/shared/routes.conf`

Add a location block **alphabetically** in the shared routes file (single-sourced for K8s and compose):
```nginx
location ~ ^/api/<service-path>(/.*)?$ {
  proxy_pass http://atlas-<service>:8080;
}
```

After editing, run `./deploy/scripts/sync-k8s-ingress-routes.sh` to regenerate the inlined K8s ConfigMap in `deploy/k8s/ingress.yaml`.

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
After scaffolding is complete, run these skills to verify the work:
1. `/service-doc` — generates/verifies service documentation
2. `/backend-audit` — audits against Atlas backend developer guidelines

## Database & Tenant Filtering Notes
- `database.Connect()` automatically registers GORM tenant-filtering callbacks — do NOT add `RegisterTenantCallbacks` to `main.go`
- Providers do NOT take `tenantId` — tenant filtering is automatic via `db.WithContext(ctx)`
- Only `create` functions need `tenantId` (to set the entity field)
- Test files using SQLite directly must call `database.RegisterTenantCallbacks(l, db)` after `gorm.Open()`
- Entity structs should use `TenantId` (not `TenantID`) for field naming consistency

## Conditional Steps
- Steps 4 and 5 only apply to services that expose REST endpoints. Kafka-only services skip Bruno and ingress.
- Step 6 only applies when the change introduces new atlas-channel packet writers or recv handlers. Pure-REST services and Kafka-only services skip the opcode template seed.
