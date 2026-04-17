---
name: Deploy Reorg — Task Checklist
description: Progress checklist for the deployment reorganization task, tracked phase-by-phase.
type: tasks
task: task-001-deploy-reorg
---

# Tasks — Deployment Reorganization

Last Updated: 2026-04-16

Legend: effort = S (≤0.5d) / M (0.5–2d) / L (2–5d) / XL (>5d). Check boxes in order; phase ordering is load-bearing.

## Phase 0 — Safety rails (S)

- [ ] **0.1** Confirm working tree is clean except the pre-existing `tools/db-bootstrap.sh` change. *(effort: S)*
- [ ] **0.2** Create feature branch `deploy-reorg` off `main`. *(effort: S)*
- [ ] **0.3** Append to root `.gitignore`:
  ```
  deploy/compose/.env
  deploy/k8s/secrets.yaml
  ```
  *(effort: S)*
- [ ] **0.4** Create empty placeholder files and verify with `git check-ignore -v deploy/compose/.env deploy/k8s/secrets.yaml`. Remove placeholders. *(effort: S)*

**Acceptance:** `.gitignore` lists both paths; `git check-ignore -v` reports them as ignored.

## Phase 1 — `deploy/` skeleton + K8s shared files (M)

- [ ] **1.1** `mkdir -p deploy/k8s deploy/compose deploy/shared deploy/scripts`. *(effort: S)*
- [ ] **1.2** Split `base.yaml` → `deploy/k8s/namespace.yaml` (Namespace only). *(effort: S)*
- [ ] **1.3** Extract Secret block → `deploy/k8s/secrets.yaml` verbatim (preserves base64 values including the `\r\n`-trailing DB creds). *(effort: S)*
- [ ] **1.4** Create `deploy/k8s/secrets.example.yaml` with same schema, `data:` values replaced by `CHANGE_ME_BASE64`. *(effort: S)*
- [ ] **1.5** Extract nginx routes from `atlas-ingress.yml` → `deploy/shared/routes.conf`. Rewrite every `proxy_pass http://atlas-<svc>.atlas.svc.cluster.local:8080;` → `proxy_pass http://atlas-<svc>:8080;`. *(effort: M)*
- [ ] **1.6** Rewrite `deploy/k8s/ingress.yaml`: ConfigMap with `nginx.conf` key (K8s header + `include /etc/nginx/routes.conf;`) + `routes.conf` key (inlined shared content); Deployment volume mounts both keys. Preserve `metadata.name: atlas-ingress` and `atlas-ingress-configmap` (tools/debug-start.sh depends on these). *(effort: M)*
- [ ] **1.7** `git mv services/atlas-env.yaml deploy/k8s/env-configmap.yaml` (no content change). *(effort: S)*
- [ ] **1.8** Delete now-empty `base.yaml` and `atlas-ingress.yml` from repo root. *(effort: S)*
- [ ] **1.9** Dry-run apply each file: `kubectl --dry-run=client apply -f deploy/k8s/<file>` for all created files. *(effort: S)*

**Acceptance:** `deploy/k8s/` has 5 shared files; content of each is syntactically valid K8s; `deploy/shared/routes.conf` contains only `location` blocks with bare container-name proxy targets; `atlas-ingress-configmap` and `atlas-ingress` names preserved in `ingress.yaml`.

## Phase 2 — Move 54 per-service K8s manifests (M)

- [ ] **2.1** For each of the 54 services (all `services/atlas-*` except `atlas-families`, `atlas-marriages`) run `git mv services/atlas-<name>/atlas-<name>.yml deploy/k8s/atlas-<name>.yaml`. *(effort: M)*
- [ ] **2.2** Verify: `ls deploy/k8s/atlas-*.yaml | wc -l` == 54. *(effort: S)*
- [ ] **2.3** Verify per-file content diff is empty (pure rename): `git diff --stat HEAD` shows only renames. *(effort: S)*
- [ ] **2.4** `kubectl --dry-run=server apply -f deploy/k8s/` against a test cluster; all resources parse. *(effort: S)*

**Acceptance:** 54 manifests in `deploy/k8s/`; no content changes; server-side dry-run apply succeeds.

## Phase 3 — Compose skeleton + nginx + scripts (M)

- [ ] **3.1** Write `deploy/compose/docker-compose.yml` (base): `networks: atlas: name: atlas` + `nginx` service (image `nginx:alpine`, mounts `./nginx.conf` and `./routes.conf`, `${INGRESS_HOST_PORT:-8080}:80`). *(effort: S)*
- [ ] **3.2** Write `deploy/compose/nginx.conf` with compose-specific header (resolver `127.0.0.11 valid=30s`, `server_name _`, tenant/region `proxy_set_header` directives copied from today's ingress, `underscores_in_headers on`) and `include /etc/nginx/conf.d/routes.conf;` inside `server {}`. *(effort: S)*
- [ ] **3.3** Create compose routes symlink: `(cd deploy/compose && ln -sf ../shared/routes.conf routes.conf)`. *(effort: S)*
- [ ] **3.4** Write `deploy/scripts/sync-k8s-ingress-routes.sh`: reads shared `routes.conf`, re-indents to match YAML block-scalar in `deploy/k8s/ingress.yaml`, rewrites the `routes.conf: |` key. Support `--check` to exit non-zero on drift. Make executable. *(effort: M)*
- [ ] **3.5** Stub empty `deploy/compose/docker-compose.core.yml` and `docker-compose.socket.yml` (`services: {}`). *(effort: S)*
- [ ] **3.6** Write `up.sh` / `down.sh` / `logs.sh` per PRD §4.5: `set -euo pipefail`, resolve `SCRIPT_DIR`, validate stack selector `{core,socket,all}` (default `core`), enforce `--project-name atlas`, source `$SCRIPT_DIR/.env`, print friendly error if `.env` missing. `chmod +x`. *(effort: M)*
- [ ] **3.7** Write `deploy/compose/.env.example` enumerating every key any overlay will reference (infra hostnames, `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` list, `DB_USER=CHANGE_ME`, `DB_PASSWORD=CHANGE_ME`, `INGRESS_HOST_PORT=8080`, `ATLAS_IMAGE_TAG=local`, `BASE_SERVICE_URL=http://atlas-ingress:80/api/`). *(effort: M)*

**Acceptance:** `./deploy/compose/up.sh` (no args) runs, fails cleanly with missing-`.env` message; script is executable; `routes.conf` symlink resolves to `../shared/routes.conf`; sync script exits 0 when K8s ingress is in sync with shared file.

## Phase 4 — Populate compose overlays (L)

- [ ] **4.1** Write a generator (bash/awk or short Python) that reads each `deploy/k8s/atlas-<name>.yaml`, extracts literal-value `env:` entries (skip `configMapRef`/`secretKeyRef`), and emits a compose stanza per PRD §4.10. *(effort: M)*
- [ ] **4.2** Generate 52 stanzas into `docker-compose.core.yml` with: `container_name`, `build.context: ../..`, `build.dockerfile: services/atlas-<name>/Dockerfile`, `image: atlas-<name>:${ATLAS_IMAGE_TAG:-local}`, `env_file: .env`, `environment:` block from PRD §4.10, `extra_hosts:` mapping `{postgres,kafka,redis,tempo}.home:host-gateway`, `networks: [atlas]`, `restart: unless-stopped`. *(effort: L)*
- [ ] **4.3** Generate 2 stanzas into `docker-compose.socket.yml` for `atlas-login` and `atlas-channel` with the same template. *(effort: S)*
- [ ] **4.4** Add volume mounts:
  - `atlas-assets`: `../../tmp/assets:/usr/assets`
  - `atlas-data`: `../../tmp/data:/usr/data`
  - `atlas-wz-extractor`: `../../tmp/wz-input:/usr/wz-input`, `../../tmp/data:/usr/data`, `../../tmp/assets:/usr/assets`
  *(effort: S)*
- [ ] **4.5** Publish ports: `atlas-ui` → `3000:3000`; `atlas-login` → `1200,8300,8700,9200,9500,18500`; `atlas-channel` → `1201,8301,8701,18501`. Use `"${HOST_IFACE:-0.0.0.0}:<port>:<port>"` syntax. *(effort: S)*
- [ ] **4.6** Create empty `tmp/assets/` and `tmp/data/` directories (`tmp/wz-input/` already exists). Verify `tmp/` is gitignored (line 27 of root `.gitignore`). *(effort: S)*
- [ ] **4.7** `docker compose --project-name atlas --env-file /dev/null -f deploy/compose/docker-compose.yml -f deploy/compose/docker-compose.core.yml config` parses without error. Repeat for socket overlay and all-overlay combinations. *(effort: S)*

**Acceptance:** Compose config validates for `core`, `socket`, and `all`. All env vars from PRD §4.10 transcribed. Volume mounts present for 3 flagged services. Ports published only for the 3 services listed.

## Phase 5 — Generate real `.env` and `secrets.yaml` (S)

- [ ] **5.1** Decode `DB_USER` and `DB_PASSWORD` from base.yaml's secret block (`YXRsYXMgDQo=` → `atlas \r\n` — preserve trailing CR+LF exactly). *(effort: S)*
- [ ] **5.2** Flatten every `data:` key/value from `deploy/k8s/env-configmap.yaml` into `deploy/compose/.env` (`KEY=value` per line; unquote YAML-quoted values). Append decoded `DB_USER`/`DB_PASSWORD`. Add `INGRESS_HOST_PORT=8080`, `ATLAS_IMAGE_TAG=local`. Override `BASE_SERVICE_URL=http://atlas-ingress:80/api/`. *(effort: S)*
- [ ] **5.3** Verify `.env` is gitignored: `git status` does not list it. *(effort: S)*
- [ ] **5.4** `deploy/k8s/secrets.yaml` confirmed gitignored; `git status` does not list it. *(effort: S)*

**Acceptance:** `git status` shows neither real file as untracked. `docker compose config` expands `${DB_USER}` etc. correctly.

## Phase 6 — Reference sweep + legacy cleanup (M)

- [ ] **6.1** Delete all `services/atlas-*/docker-build.sh` and `services/atlas-*/docker-build.bat`. *(effort: S)*
- [ ] **6.2** Update `README.md` deployment section: replace old paths; add a `deploy/compose/` usage section with `up.sh`/`down.sh`/`logs.sh` examples; document single-source routes workflow. *(effort: M)*
- [ ] **6.3** Re-grep and update `CLAUDE.md`. *(effort: S)*
- [ ] **6.4** Re-grep and update `DOCS.md`. *(effort: S)*
- [ ] **6.5** Sweep all 56 `services/atlas-*/README.md` for old references; update or remove. *(effort: M)*
- [ ] **6.6** Sweep all `services/atlas-*/docs/*.md` for old references; update or remove. *(effort: M)*
- [ ] **6.7** Verify `tools/debug-start.sh` and `tools/debug-stop.sh` still match `atlas-ingress` + `atlas-ingress-configmap` + `atlas` namespace; no code change expected. *(effort: S)*
- [ ] **6.8** Re-grep `.github/workflows/*.yml` for old paths (expect zero). *(effort: S)*
- [ ] **6.9** Final sweep: `grep -rnE "base\.yaml|atlas-ingress\.yml|services/atlas-env\.yaml|services/atlas-[a-z-]+/atlas-[a-z-]+\.yml|docker-build\.(sh|bat)"` returns only results inside `docs/tasks/task-001-deploy-reorg/` and the new `deploy/` tree. *(effort: S)*

**Acceptance:** Grep sweep has no hits outside `docs/tasks/` and `deploy/`. README deployment section matches the new layout.

## Phase 7 — End-to-end verification (M)

- [ ] **7.1** `docker system prune -af` on the test machine. *(effort: S)*
- [ ] **7.2** `./deploy/compose/up.sh core --build`. Wait for builds (expect 10–20 min cold). *(effort: M)*
- [ ] **7.3** `curl -I http://localhost:8080/` returns valid nginx response. *(effort: S)*
- [ ] **7.4** `curl -H "TENANT_ID: <real-uuid>" -H "REGION: GMS0" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" http://localhost:8080/api/accounts` returns a JSON:API payload from `atlas-account`. *(effort: S)*
- [ ] **7.5** In a second terminal: `./deploy/compose/up.sh socket`. *(effort: S)*
- [ ] **7.6** `nc -zv localhost 1200` and `nc -zv localhost 1201` both succeed. *(effort: S)*
- [ ] **7.7** With both overlays up, confirm `atlas-channel` → `atlas-character` interaction via `docker logs atlas-channel` on a fresh login attempt (look for successful REST + Kafka exchanges). *(effort: S)*
- [ ] **7.8** `./deploy/compose/down.sh all` cleanly removes all containers. *(effort: S)*
- [ ] **7.9** On a test K8s cluster: `kubectl apply -f deploy/k8s/namespace.yaml && kubectl apply -f deploy/k8s/`. Smoke-test ingress endpoints. *(effort: M)*
- [ ] **7.10** `./deploy/scripts/sync-k8s-ingress-routes.sh --check` exits 0. *(effort: S)*

**Acceptance:** All acceptance checks from PRD §10 pass. No regressions in cluster or local workflows.

## Phase 8 — PR hygiene (S)

- [ ] **8.1** Commit 1: K8s moves (`git mv` renames + `base.yaml`/`atlas-ingress.yml` split). Preserves `git log --follow` per file. *(effort: S)*
- [ ] **8.2** Commit 2: New compose files + scripts + shared routes + sync tool. *(effort: S)*
- [ ] **8.3** Commit 3: Documentation updates + `docker-build.{sh,bat}` deletions + service README sweeps. *(effort: S)*
- [ ] **8.4** Pre-push: `git status` shows `deploy/compose/.env` and `deploy/k8s/secrets.yaml` as ignored, not tracked. *(effort: S)*
- [ ] **8.5** Open PR; body links back to `docs/tasks/task-001-deploy-reorg/`; includes decode+generate instructions for reviewers to create their own `.env` / `secrets.yaml` locally. *(effort: S)*

**Acceptance:** PR opened; three commits visible; CI (if any) passes; reviewers have reproducible local setup instructions.

## Out-of-scope follow-ups (do not do in this task)

- Fix `atlas-ui` `NEXT_PUBLIC_ROOT_API_URL` build-time bake (pass as build ARG or switch to runtime config endpoint).
- Add `atlas-families` and `atlas-marriages` manifests + compose entries once they're ready to deploy.
- CI drift check: run `sync-k8s-ingress-routes.sh --check` as a workflow step.
- Pre-commit hook that runs `sync-k8s-ingress-routes.sh` automatically.
- Compose profile pulling prebuilt `ghcr.io/chronicle20/...` images for contributors who don't need to iterate on code.
- Dockerfile build-optimization pass to reduce cache invalidation when shared libs change.
