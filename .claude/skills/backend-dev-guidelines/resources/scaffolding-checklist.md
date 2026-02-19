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
**File:** `services/atlas-<service>/atlas-<service>.yml`

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
**File:** `atlas-ingress.yml`

Add a location block **alphabetically** in the nginx config section:
```nginx
location ~ ^/api/<service-path>(/.*)?$ {
  proxy_pass http://atlas-<service>.atlas.svc.cluster.local:8080;
}
```

## 6. Post-Scaffold Verification
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
- Steps 4 and 5 only apply to services that expose REST endpoints
- Kafka-only services (no REST API) skip Bruno and ingress
