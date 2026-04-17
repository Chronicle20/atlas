# Compose Design — Deployment Reorganization

This document drills into the design of the Docker Compose portion of the reorg. It exists to resolve the non-obvious decisions up front so implementation is mechanical.

## File split & why

Three files:

| File                            | Purpose                                                          | Always loaded?          |
| ------------------------------- | ---------------------------------------------------------------- | ----------------------- |
| `docker-compose.yml`            | Base: `atlas` network declaration + `nginx` reverse-proxy entry  | Yes (every invocation)  |
| `docker-compose.core.yml`       | 52 services (51 HTTP/Kafka + `atlas-ui`). Excludes `atlas-families` and `atlas-marriages` — no K8s manifest today, parity with K8s. | When `STACK=core\|all`  |
| `docker-compose.socket.yml`     | `atlas-login` + `atlas-channel` (raw TCP socket servers)         | When `STACK=socket\|all`|

**Why separate files (not profiles).** The user's constraint: "run core services, and still keep existing login/channel running separate." That implies the socket servers might be running outside compose entirely (e.g., bare `go run` under a debugger, or a previous compose session that hasn't been torn down). Separate files — combined with a stable project name and an externally-named network — let each overlay be started, stopped, rebuilt, and logged independently without clobbering the other. Compose profiles *could* work, but they share one file and one up/down invocation cycle, which is awkward when one half of the stack lives in a debugger and the other in compose.

**Why nginx lives in the base file.** nginx is needed whenever HTTP is. Putting it in `core.yml` would mean `socket`-only sessions couldn't later add HTTP routes if a developer brought up `core` in a second terminal. Putting it in the base guarantees one nginx, addressable at the same host port, regardless of overlay mix. When only `socket` is loaded, nginx still starts but its upstreams fail to resolve — that's harmless because no one's calling it. (The `resolver ... valid=30s;` directive means it'll start succeeding as core services come online without an nginx restart.)

**Why a named network (`name: atlas`).** Default compose networks are auto-named `<project>_default`, which is stable as long as `--project-name` is stable. Explicitly naming the network makes the topology visible in `docker network ls` and makes it trivial to attach one-off debug containers: `docker run --rm -it --network atlas --entrypoint sh alpine`.

## Build context & caching strategy

Every service's production `Dockerfile` (not `Dockerfile.dev`) uses the **repo root** as its build context. Concretely, compose entries look like:

```yaml
  atlas-account:
    build:
      context: ../..                  # repo root relative to deploy/compose/
      dockerfile: services/atlas-account/Dockerfile
    image: atlas-account:${ATLAS_IMAGE_TAG:-local}
```

**Caching behavior.** Each Dockerfile starts by copying every `libs/atlas-<x>/go.mod` + `go.sum`, writing a minimal `go.work`, running `go mod download`, then copying sources and `go build`. Under BuildKit, the first 20ish layers of every image are identical (same `libs/*` go.mod set). After the first service builds, those layers are cached and reused for the next 55. Cold build time is roughly "time-to-build-one-service × 1 + time-to-compile × 55" rather than "× 56" — call it 10–20 minutes on a typical machine. Warm rebuilds of a single service touch only that service's COPY + build steps — seconds.

**Cache invalidation trap.** Changing a shared lib (e.g., `libs/atlas-kafka/consumer.go`) invalidates the `COPY libs/atlas-kafka go.mod go.sum` layer for every service. That's unavoidable with the current Dockerfile shape and is the same cost already paid in CI. Don't try to "optimize" this by restructuring Dockerfiles in this task — it's out of scope and belongs in a separate build-optimization pass.

**Why not `Dockerfile.dev`?** `Dockerfile.dev` uses a service-local context (`ADD ./atlas.com/<svc>`) and doesn't copy shared libs. It only works for services whose only dependency is `go.work`-resolved against the host filesystem — it's a convenience for local `docker-build.sh` runs, not a parity build with CI. Using it in compose would create images that diverge from what ships to the cluster. Stick with `Dockerfile`.

## Environment wiring

### Flow

```
┌──────────────────────┐          ┌────────────────────────┐
│ deploy/compose/.env  │ ──load── │ every compose service  │
└──────────────────────┘          │ via `env_file: .env`   │
                                  └──────────────┬─────────┘
                                                 │
                                     env vars passed to container
                                                 │
                                                 ▼
                                       service reads via os.Getenv
```

All globally-shared values live in `.env` (infra endpoints, Kafka topics, feature flags). Per-service-unique values (today only `DB_NAME` in most K8s manifests) go in the service's compose `environment:` block, transcribed from its K8s manifest.

### Variables sourced from today's `atlas-env.yaml`

Every key/value pair in the ConfigMap becomes one line in `.env`. Notable ones:

- Infrastructure: `BASE_SERVICE_URL`, `BOOTSTRAP_SERVERS`, `DB_HOST`, `DB_PORT`, `REDIS_URL`, `TRACE_ENDPOINT`, `REST_PORT`.
- ~50 `COMMAND_TOPIC_*` and `EVENT_TOPIC_*` entries — straight copy.

### Variables sourced from today's `base.yaml` secret

Decoded from base64 and written verbatim into `.env`:

- `DB_USER`
- `DB_PASSWORD`

### Compose-specific additions

- `INGRESS_HOST_PORT` — host port for the nginx container. Defaults to `8080`.
- `ATLAS_IMAGE_TAG` — local image tag suffix. Defaults to `local`. Allows a developer to tag a working build and roll back.

### Override semantics

- `.env` is the "real" values file (gitignored, generated during Phase 5).
- `.env.example` is the committed template with placeholder values for every key.
- Developers who run against non-standard infra override `.env` locally — the file is theirs, never overwritten.
- `BASE_SERVICE_URL` specifically needs to be overridden for compose: the K8s-internal value `http://atlas-ingress.atlas.svc.cluster.local:80/api/` won't resolve inside the compose network. Suggested compose default: `http://atlas-ingress:80/api/` (uses the nginx container's DNS name).

## Host-infra reachability (`extra_hosts`)

Every compose service receives:

```yaml
extra_hosts:
  - "postgres.home:host-gateway"
  - "kafka.home:host-gateway"
  - "redis.home:host-gateway"
  - "tempo.home:host-gateway"
```

`host-gateway` is a magic value supported by Docker Engine 20.10+ that maps the hostname to the host's gateway IP as seen by the container. When the developer's host DNS resolves `postgres.home` to a real address (via `/etc/hosts`, `.home` domain local resolver, mDNS, etc.), that resolution happens on the host side — the container just needs to reach the host. This approach:

- **Doesn't require host networking.** Containers keep their own network namespace, preserving per-container name resolution.
- **Doesn't hardcode IPs.** If the developer moves their Postgres to a different host, they update their host DNS (not compose files).
- **Allows per-developer overrides.** If someone doesn't have `*.home` DNS, they set `DB_HOST=192.168.1.50` in their `.env`; `extra_hosts` becomes a no-op for that key.

**Caveat on Docker Desktop (macOS/Windows).** `host-gateway` resolves to `host.docker.internal` semantics — i.e., reaches the host's loopback. If the infra is running on the host's loopback, that works. If it's on a separate machine in the LAN, update `.env` with the machine's actual hostname/IP instead of relying on `host-gateway`.

## nginx design — single-sourced routes

Route bodies are identical between K8s and compose because:

- All HTTP services listen on container port `8080` in both environments.
- Bare service names (`atlas-account`) resolve correctly in both: in K8s via `resolv.conf` search-list (`atlas.svc.cluster.local`), in compose via Docker's embedded DNS.

So we split nginx config into a **shared** routes file plus **environment-specific** headers:

| File                              | Environment  | Contents                                                                                                                        |
| --------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| `deploy/shared/routes.conf`       | Both         | Every `location ~ ^/api/...` block. `proxy_pass http://atlas-<svc>:8080;` (bare container names, no cluster DNS suffix).       |
| `deploy/k8s/ingress.yaml`         | K8s only     | ConfigMap with two keys: `nginx.conf` (K8s-specific header + `include /etc/nginx/routes.conf;`) and `routes.conf` (inlined).   |
| `deploy/compose/nginx.conf`       | Compose only | Compose-specific header + `include /etc/nginx/conf.d/routes.conf;`.                                                             |
| `deploy/compose/routes.conf`      | Compose only | Symlink → `../shared/routes.conf`. Bind-mounted into the nginx container at `/etc/nginx/conf.d/routes.conf`.                   |

Environment-specific header differences:

| Directive                                       | K8s                                            | Compose                      |
| ----------------------------------------------- | ---------------------------------------------- | ---------------------------- |
| `resolver`                                      | `10.43.0.10 valid=30s;` (CoreDNS)              | `127.0.0.11 valid=30s;` (Docker DNS) |
| `server_name`                                   | `dev.atlas.home;`                              | `_;` (match any Host header) |
| Tenant/region `proxy_set_header` + `underscores_in_headers on;` + keepalive timeouts | Identical (in the header file in both envs) | Identical                    |

**Keeping K8s and shared in sync.** `deploy/scripts/sync-k8s-ingress-routes.sh` reads `deploy/shared/routes.conf`, indents it to match the YAML block scalar in `deploy/k8s/ingress.yaml`, and rewrites the `routes.conf: |` key's value block. Developers run this after editing `routes.conf`; it exits 0 when already in sync. A future CI check can run `--check` mode to fail on drift.

**Why not generate the ConfigMap at deploy time** (e.g., `kubectl create configmap --from-file`)? That would change the K8s deploy workflow from `kubectl apply -f deploy/k8s/` to a two-step process. Keeping `ingress.yaml` self-contained preserves the current deploy ergonomics at the cost of needing the sync script.

### What nginx can't do

It can't route TCP for the socket servers. Game clients must connect directly to the host ports published by `atlas-login` and `atlas-channel` — nginx is HTTP-only. That's fine because game clients are expected to know their endpoints; the HTTP ingress is only for REST API callers and the admin UI.

## Socket-server ports

`atlas-login` and `atlas-channel` publish their TCP ports 1:1 to the host so game clients can connect to `localhost:<port>`.

Ports (verified against current K8s manifests):

- `atlas-login`: 1200, 8300, 8700, 9200, 9500, 18500
- `atlas-channel`: 1201, 8301, 8701, 18501

Port binding syntax: `"${HOST_IFACE:-0.0.0.0}:1200:1200"` (via env override). Default binds to all interfaces; developers who don't want these exposed to their LAN can set `HOST_IFACE=127.0.0.1`.

**Port conflict risk.** If a developer also runs `atlas-login`/`atlas-channel` outside compose (e.g., in a debugger on the host), the host-port publish will fail. That's the *intended* behavior: the separate-file split means those services are opt-in. Don't bring up `socket` if you're already running them elsewhere.

## `atlas-wz-extractor` special-casing

This service reads `tmp/wz-input/` and writes `tmp/assets/`. It's a **long-running service** (confirmed during spec review), so no profile gating. In compose:

- Lives in `docker-compose.core.yml` alongside the other core services.
- `restart: unless-stopped` (default — no override).
- Needs volume mounts for input/output:

  ```yaml
  volumes:
    - ../../tmp/wz-input:/tmp/wz-input:ro
    - ../../tmp/assets:/tmp/assets
  ```

  Host paths are repo-root-relative (`tmp/wz-input/`, `tmp/assets/` already exist per the current repo layout).

## `atlas-ui` wiring

- Container port 3000, published as `3000:3000`.
- Lives in `docker-compose.core.yml`.
- `NEXT_PUBLIC_ROOT_API_URL`: set to `http://atlas-ingress:8080` to match the K8s runtime-env value (cluster DNS rewritten to compose container name). **Caveat**: the Dockerfile runs `npm run build` without this ARG, so Next.js bakes `undefined` into the client bundle at build time; the runtime env only affects server-side code (server components, API routes). This is pre-existing behavior in the K8s deployment and is intentionally **replicated verbatim** here — see PRD §8.3.
- If browser-side API calls need to reach the host-published nginx directly, either the UI code must fall back to same-origin relative URLs (most likely what happens today) or a follow-up task must switch to a build-time ARG.

## Start-order & health

- No `depends_on` conditions with `service_healthy` — adds latency and fragility. Services already retry Kafka/REST/DB on startup.
- Minimal `depends_on: [<name>]` (no condition) is OK where it makes cold-start order more readable (e.g., nginx "after" core services) but is not required for correctness.
- Do not add health checks to service entries in this task. Consistent health-check wiring is valuable but it's a separate concern — handle in a follow-up.

## What's explicitly **not** in compose

Called out to make the boundary clear:

- Postgres, Redis, Kafka, Zookeeper, Tempo, Grafana — supplied externally by the host.
- TLS termination — HTTP only.
- Kubernetes ingress controller replica — single nginx, single port, no Traefik/Contour.
- Service-mesh concerns (Istio, Linkerd) — N/A locally.
- CI-style image registry pulls — compose always builds. Pulling prebuilt images from GHCR is explicitly out of scope (PRD §9 decision 5) and can be a future enhancement.
