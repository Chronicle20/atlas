---
name: Deploy Reorg — Implementation Plan
description: Comprehensive, phased plan to consolidate Atlas deployment artifacts under deploy/ and add a local docker-compose stack.
type: plan
task: task-001-deploy-reorg
---

# Deployment Reorganization — Implementation Plan

Last Updated: 2026-04-16
Companion docs: `prd.md`, `migration-plan.md`, `compose-design.md`, `risks.md`

## Executive Summary

Atlas scatters deployment artifacts across the repository: per-service Kubernetes manifests live inside each `services/atlas-<name>/` directory, shared infra manifests sit at the repo root (`base.yaml`, `atlas-ingress.yml`), and the environment ConfigMap lives under `services/`. There is no local Docker Compose stack at all — running Atlas locally requires either a Kubernetes cluster or manual per-service `go run` / `docker-build.sh` workflows.

This task consolidates all deployment artifacts under a single top-level `deploy/` tree (mirroring `home-hub`), splits the nginx ingress routes into a single source shared between K8s and compose, and ships a three-file compose stack (`docker-compose.yml` base + `core.yml` + `socket.yml` overlays) that brings up the full Atlas service graph against host-provided infra (Postgres/Redis/Kafka/Tempo).

**This is a packaging/layout change only.** No service code, Go module, API surface, or data schema is modified. Live cluster migration is out of scope — this PR changes repo layout; a follow-up operational step re-applies from the new paths.

## Current State Analysis

**Kubernetes manifests today.**
- Root: `base.yaml` (Namespace + `db-credentials` Secret), `atlas-ingress.yml` (Ingress Deployment + ConfigMap embedding ~230 lines of nginx routes).
- `services/atlas-env.yaml`: shared ConfigMap with infra endpoints, Kafka topics, DB host/port.
- `services/atlas-<name>/atlas-<name>.yml`: per-service manifest (54 services have one; `atlas-families` and `atlas-marriages` do not).
- Per-service `docker-build.sh` / `docker-build.bat`: legacy one-off build scripts, duplicated across services.

**Local dev today.** Developers either stand up k3s/minikube, run `docker-build.sh` per-service, or `go run` services manually. No one-command local stack exists.

**Route duplication risk latent.** Any move that creates a separate compose nginx config guarantees drift between the K8s ConfigMap (230 lines inlined) and the compose file. This plan addresses that up front via a single `deploy/shared/routes.conf`.

**Pre-existing quirks preserved (not fixed here).**
- `atlas-ui` bakes `NEXT_PUBLIC_ROOT_API_URL` as `undefined` at build time (Next.js bakes `NEXT_PUBLIC_*` during `npm run build`, which the Dockerfile doesn't pass). K8s tolerates this today; compose replicates the behavior verbatim. Fix is a separate follow-up task.
- `base.yaml` Secret's base64 values decode with trailing `\r\n` artifacts (`atlas \r\n`). Preserve exactly — production has been tolerating this.

## Proposed Future State

```
deploy/
├── k8s/
│   ├── namespace.yaml                 # extracted from base.yaml
│   ├── secrets.example.yaml           # committed, sanitized
│   ├── secrets.yaml                   # gitignored, real values
│   ├── env-configmap.yaml             # moved from services/atlas-env.yaml
│   ├── ingress.yaml                   # K8s nginx ConfigMap + Deployment + Service
│   └── atlas-<name>.yaml              # 54 per-service manifests
├── compose/
│   ├── docker-compose.yml             # base: atlas network + nginx
│   ├── docker-compose.core.yml        # 52 services (51 HTTP/Kafka + atlas-ui)
│   ├── docker-compose.socket.yml      # atlas-login + atlas-channel
│   ├── nginx.conf                     # compose-specific header
│   ├── routes.conf                    # symlink → ../shared/routes.conf
│   ├── up.sh / down.sh / logs.sh      # wrappers accepting {core,socket,all}
│   ├── .env.example                   # committed template
│   └── .env                           # gitignored, real values
├── shared/
│   └── routes.conf                    # single-source location blocks, bare container names
└── scripts/
    └── sync-k8s-ingress-routes.sh     # regenerates ingress.yaml's inlined routes
```

**Key invariants.**
- Every K8s resource keeps its current `metadata.name`, `metadata.namespace`, labels, image refs, and container ports — purely a file move + rename + nginx single-sourcing.
- Every compose service builds from the **production** `Dockerfile` (not `Dockerfile.dev`) with build context `../..` (repo root).
- All three compose invocation modes (`core`, `socket`, `all`) share one compose project (`--project-name atlas`) and one externally-named network (`name: atlas`) so cross-overlay traffic works regardless of invocation order.

## Implementation Phases

Phased ordering matches `migration-plan.md`; do not collapse phases — intermediate states are designed to fail loudly rather than corrupt silently.

### Phase 0 — Safety rails (S)
Add `deploy/compose/.env` and `deploy/k8s/secrets.yaml` to `.gitignore` **before** generating any real file. Confirm coverage with `git check-ignore -v`. Start feature branch.

### Phase 1 — deploy/ skeleton and K8s shared files (M)
Create `deploy/{k8s,compose,shared,scripts}/`. Split `base.yaml` into `namespace.yaml` + `secrets{,.example}.yaml`. Extract nginx routes from `atlas-ingress.yml` into `deploy/shared/routes.conf` (rewriting `proxy_pass` to bare container names). Rewrite `deploy/k8s/ingress.yaml` with a K8s-header `nginx.conf` key + regenerated `routes.conf` key. Move `services/atlas-env.yaml` → `deploy/k8s/env-configmap.yaml`. Dry-run `kubectl apply` on each file.

### Phase 2 — Move per-service K8s manifests (M)
`git mv services/atlas-<name>/atlas-<name>.yml deploy/k8s/atlas-<name>.yaml` for all 54 services with manifests. Verify: count is 54; per-file content diff is empty (pure rename); `kubectl --dry-run=server apply -f deploy/k8s/` succeeds on a live cluster.

### Phase 3 — Compose skeleton + nginx wiring + sync script (M)
Write `docker-compose.yml` (network + nginx), `nginx.conf` (compose header), symlink `routes.conf → ../shared/routes.conf`, stub empty `core.yml`/`socket.yml`, and authoring `up.sh`/`down.sh`/`logs.sh` with `{core,socket,all}` stack selector and enforced `--project-name atlas`. Author `deploy/scripts/sync-k8s-ingress-routes.sh` (idempotent, supports `--check`). Write `.env.example` enumerating every variable any overlay will reference.

### Phase 4 — Populate compose overlays (L)
Generate 52 service stanzas into `core.yml` and 2 into `socket.yml`. **Use a small shell/awk generator** rather than hand-authoring — the input is each service's `atlas-<name>.yml` K8s manifest, the output is mechanical. Transcribe per-service env per PRD §4.10 (literal-value `env:` entries only; skip anything backed by `configMapRef`/`secretKeyRef`). Add volume mounts for `atlas-assets`, `atlas-data`, `atlas-wz-extractor`. Publish TCP ports 1:1 for `atlas-login`/`atlas-channel` and `3000:3000` for `atlas-ui`.

### Phase 5 — Generate real `.env` and `secrets.yaml` (S)
Flatten today's `atlas-env.yaml` ConfigMap into `deploy/compose/.env`. Decode `DB_USER`/`DB_PASSWORD` from `base.yaml` (preserve `\r\n` artifacts exactly). Add compose-specific keys: `INGRESS_HOST_PORT=8080`, `ATLAS_IMAGE_TAG=local`, override `BASE_SERVICE_URL=http://atlas-ingress:80/api/`. Copy the Secret block verbatim into `deploy/k8s/secrets.yaml`. Re-verify gitignore coverage with `git status`.

### Phase 6 — Reference sweep + legacy cleanup (M)
Delete every `services/atlas-*/docker-build.{sh,bat}`. Sweep `README.md`, `CLAUDE.md`, `DOCS.md`, all 56 `services/*/README.md`, all `services/*/docs/*.md`, `tools/debug-{start,stop}.sh`, `.github/workflows/*.yml`. Replace every reference to old paths. Confirm with `grep -rnE "base\.yaml|atlas-ingress\.yml|services/atlas-env\.yaml|services/atlas-[a-z-]+/atlas-[a-z-]+\.yml|docker-build\.(sh|bat)"` — expect results only inside `docs/tasks/` and `deploy/`.

### Phase 7 — End-to-end verification (M)
Cold build `up.sh core --build` from a pruned Docker state; hit `/api/accounts` via localhost:8080 with tenant headers. Bring up `socket` in a second terminal; confirm `nc -zv localhost 1200` and `nc -zv localhost 1201`. With both up, validate `atlas-channel` → `atlas-character` interaction via logs. Tear down with `down.sh all`. Apply all K8s files to a test cluster; smoke-test a few endpoints via cluster ingress.

### Phase 8 — PR hygiene (S)
Commit in three logical units: (1) K8s moves (preserves `git log --follow`), (2) new compose + scripts + shared routes, (3) documentation updates. Pre-push check: `git status` shows real `.env` and `secrets.yaml` as ignored, not tracked. Open PR linking back to `docs/tasks/task-001-deploy-reorg/`.

## Task Breakdown Structure

See `tasks.md` for the per-task checklist. Each task inherits the phase's effort size unless explicitly re-estimated.

## Risk Assessment and Mitigation Strategies

Full risk register in `risks.md` (R1–R12). Top risks summarized:

| Risk | Impact | Primary mitigation |
| --- | --- | --- |
| R1 — Real `.env`/`secrets.yaml` committed | High | Gitignore **before** generation; `git check-ignore -v` verification; `.gitattributes -export-ignore` |
| R8 — `BASE_SERVICE_URL` misroutes in compose | High | Override to `http://atlas-ingress:80/api/` in `.env`; acceptance test hits cross-service call chain |
| R12a — K8s ingress.yaml drifts from shared routes | High | `sync-k8s-ingress-routes.sh --check` in acceptance criteria; future pre-commit hook |
| R2 — kubectl apply ordering | High (low likelihood) | Namespace sorts first alphabetically; README calls out two-step apply |
| R7 — Socket port conflicts | Medium | Document `HOST_IFACE=127.0.0.1` override; per-port `.env` overrides if adoption pain surfaces |
| R4 — `host-gateway` fails on non-default setups | Medium | Document override via `DB_HOST`/`BOOTSTRAP_SERVERS` in `.env`; troubleshooting section |

## Success Metrics

Concrete acceptance criteria (full list in PRD §10):

- `deploy/k8s/` contains exactly 54 `atlas-<name>.yaml` files + 5 shared files (`namespace`, `secrets`, `secrets.example`, `env-configmap`, `ingress`).
- `deploy/compose/up.sh core` + `up.sh socket` in separate terminals yield one shared compose project (`atlas`) and one shared network; `docker network inspect atlas` shows containers from both overlays.
- `curl -H "TENANT_ID: <uuid>" -H "REGION: GMS0" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" http://localhost:8080/api/accounts` returns JSON:API from `atlas-account` via compose nginx.
- `nc -zv localhost 1200` and `nc -zv localhost 1201` succeed after `up.sh socket`.
- `kubectl apply -f deploy/k8s/namespace.yaml && kubectl apply -f deploy/k8s/` brings up a functioning cluster equivalent to today's.
- Grep for legacy paths returns only `docs/tasks/` and `deploy/` hits.
- `deploy/scripts/sync-k8s-ingress-routes.sh --check` exits 0 on the merge commit.

## Required Resources and Dependencies

**Host prerequisites for compose end-to-end test.**
- Docker Engine ≥ 23 with BuildKit enabled (default).
- Reachable Postgres, Redis, Kafka, Tempo at the hostnames declared in `.env` (defaults `postgres.home`, `kafka.home:9093`, `redis.home`, `tempo.home:4317`).
- Available host ports: `8080` (nginx), `3000` (atlas-ui), and the 10 socket ports published by `atlas-login`/`atlas-channel`.

**Cluster prerequisites for K8s smoke test.**
- A writable test cluster (any — k3s/kind/minikube/shared dev) where `kubectl apply -f deploy/k8s/` can run without affecting production.

**Tooling.** `git mv`, `kubectl`, `docker compose`, `nc`, `awk`/`sed`. No new dependencies introduced.

**No code/library dependencies.** Task touches only deploy artifacts, documentation, and `.gitignore`.

## Timeline Estimates

Effort sizes (S ≤ 0.5d, M = 0.5–2d, L = 2–5d, XL > 5d):

| Phase | Effort | Notes |
| --- | --- | --- |
| Phase 0 — Safety rails | S | Mostly `.gitignore` hygiene. |
| Phase 1 — Skeleton + K8s shared files | M | ingress.yaml rewrite is the chunky part. |
| Phase 2 — Move 54 per-service manifests | M | Mechanical `git mv` loop; dry-run apply. |
| Phase 3 — Compose skeleton + scripts + sync tool | M | Scripts and nginx header. |
| Phase 4 — Populate 54 compose stanzas | L | Generator script amortizes the labor; env-table transcription from PRD §4.10. |
| Phase 5 — Generate real `.env`/`secrets.yaml` | S | Mostly copy-paste; verify gitignore. |
| Phase 6 — Reference sweep + legacy cleanup | M | 56 service READMEs to scan; re-grep. |
| Phase 7 — E2E verification | M | Cold build can be 10–20 min; two terminals for core+socket; one cluster apply. |
| Phase 8 — PR hygiene | S | Three logical commits; PR body. |

**Total estimate: 1–2 developer weeks** depending on how aggressive Phase 4's generator is vs. manual transcription, and how many service-local docs turn up stale references in Phase 6.

## Non-Goals (explicit, to prevent scope creep)

- No Helm/Kustomize/Timoni templating.
- No Dockerfile rewrites or build-optimization passes.
- No CI workflow changes unless a reference is actually discovered.
- No service code, Go module, or API surface changes.
- No live cluster migration (that's an operator step after merge).
- No backing infra (Postgres/Redis/Kafka/Tempo) in compose — host-provided only.
- No `NEXT_PUBLIC_ROOT_API_URL` build-ARG fix for `atlas-ui` (separate follow-up).
- No `atlas-families` or `atlas-marriages` compose entries (no K8s manifest today → parity).
- No automatic cluster→compose ingress rerouting wiring (`tools/debug-start.sh --target` already supports the manual path).
