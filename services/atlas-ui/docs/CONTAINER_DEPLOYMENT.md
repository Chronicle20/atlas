# Container deployment

Atlas UI is a static Vite-built SPA served by nginx. The production image has two stages: a `node:24-alpine` builder that runs `npm ci && npm run build`, and an `nginx:alpine` runtime that copies `dist/` to `/usr/share/nginx/html` and serves on port 80 with a `try_files $uri $uri/ /index.html` SPA fallback.

There is no Node.js at runtime. There is no image-optimisation layer. Plain `<img>` tags do their own thing; nothing in the container needs `DOCKER_ENV` or `DISABLE_IMAGE_OPTIMIZATION` to be set.

## Build

```bash
# From the atlas repo root (Dockerfile context is the repo root):
docker build -f services/atlas-ui/Dockerfile -t atlas-ui:local .
docker run --rm -p 3000:80 atlas-ui:local
```

The compose-mapped host port stays at 3000 for local-dev compatibility (`3000:80` in `deploy/compose/docker-compose.core.yml`).

## Environment variables

All runtime configuration is baked into the bundle at build time (that's how Vite env vars work — they're `import.meta.env.VITE_*`). Changing them after the fact requires a rebuild. Pass them at build time or in the compose / k8s build step:

| Variable | Purpose |
|---|---|
| `VITE_ROOT_API_URL` | Base URL for API requests (falls back to `window.location.origin` at runtime). |
| `VITE_BUILD_VERSION` | Attached to every error report. |
| `VITE_ERROR_ENDPOINT` | Remote error sink URL. If unset, remote error logging is disabled. |
| `VITE_ERROR_API_KEY` | Auth header for the error sink. |
| `VITE_ASSET_BASE_URL` | Override for the `/api/assets` prefix used by `getAssetIconUrl`. |

## Kubernetes

The deploy manifest lives at `deploy/k8s/atlas-ui.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-ui
  namespace: atlas
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atlas-ui
  template:
    metadata:
      labels:
        app: atlas-ui
    spec:
      containers:
      - name: ui
        image: ghcr.io/chronicle20/atlas-ui/atlas-ui:latest
        ports:
        - containerPort: 80
        env:
        - name: VITE_ROOT_API_URL
          value: "http://atlas-ingress.atlas.svc.cluster.local"
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-ui
  namespace: atlas
spec:
  selector:
    app: atlas-ui
  ports:
  - protocol: TCP
    port: 80
```

The ingress proxies the catch-all `location /` at `deploy/k8s/ingress.yaml` and `deploy/shared/routes.conf` both point at `atlas-ui:80`. The `/_next/webpack-hmr` blocks from the Next.js era have been removed.

## Troubleshooting

- **Blank page after navigation refresh**: verify `try_files $uri $uri/ /index.html` is in `services/atlas-ui/nginx.conf`. Without the SPA fallback, `curl http://host:80/accounts/123` returns 404 because there's no file at that path.
- **API calls fail with CORS errors**: the app expects `VITE_ROOT_API_URL` to be same-origin or behind a reverse proxy that injects the atlas tenant headers. The dev server handles this with a proxy in `vite.config.ts`; the production runtime relies on the ingress.
- **Tenant headers missing**: `TENANT_ID` / `REGION` / `MAJOR_VERSION` / `MINOR_VERSION` are set by `TenantProvider`'s effect calling `api.setTenant(activeTenant)`. If the console is full of 400s with "missing tenant", the tenant hasn't been picked from the tenant list yet — usually because `GET /api/tenants` returned an empty array.
- **Bundle looks stale after deploy**: Vite generates hashed filenames (`assets/index-abcd.js`). If the browser is loading an old hash, check the ingress's cache-control headers — CloudFront / Cloudflare in front of nginx sometimes caches `index.html` too aggressively.
