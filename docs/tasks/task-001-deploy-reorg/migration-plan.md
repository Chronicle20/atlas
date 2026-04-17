# Migration Plan — Deployment Reorganization

This document sequences the file moves, file additions, and reference updates needed to implement the PRD. Each phase is independently verifiable. Do not collapse phases — the ordering exists so intermediate states are either functional or unambiguously broken in a small, local way.

## Phase 0 — Safety rails

1. Confirm working tree is clean except for the pre-existing `tools/db-bootstrap.sh` change.
2. Confirm branch: create a feature branch (e.g., `deploy-reorg`) off of `main`.
3. Before touching any manifest, add the future gitignore entries so real secrets can't slip in when generated later:

   ```gitignore
   deploy/compose/.env
   deploy/k8s/secrets.yaml
   ```

   Append to the repo's root `.gitignore`. Verify with `git check-ignore -v deploy/compose/.env deploy/k8s/secrets.yaml` after creating placeholder empty files.

## Phase 1 — Create `deploy/` skeleton

1. `mkdir -p deploy/k8s deploy/compose deploy/shared deploy/scripts`.
2. Split `base.yaml` into two files inside `deploy/k8s/`:
   - `namespace.yaml`: only the `Namespace/atlas` resource.
   - `secrets.yaml` (gitignored): the `Secret/db-credentials` resource verbatim, preserving the current base64 values.
   - `secrets.example.yaml` (committed): same shape, with `data` values replaced by `CHANGE_ME_BASE64` or equivalent.
3. Extract routes from `atlas-ingress.yml`'s ConfigMap into `deploy/shared/routes.conf`:
   - Copy only the `location ~ ^/api/...` blocks (and any shared `proxy_set_header`/tenant-propagation directives that belong inside each route).
   - Rewrite every `proxy_pass http://atlas-<svc>.atlas.svc.cluster.local:8080;` → `proxy_pass http://atlas-<svc>:8080;`. Bare names work in both K8s (via search-domain resolution) and compose (via embedded DNS).
4. Rewrite `deploy/k8s/ingress.yaml`:
   - Keep the ConfigMap `nginx.conf` key holding the K8s-specific header (resolver `10.43.0.10`, `server_name dev.atlas.home`, http/server boilerplate) and an `include /etc/nginx/routes.conf;` directive.
   - Add a second ConfigMap key `routes.conf` whose value is the (indented) content of `deploy/shared/routes.conf`.
   - Update the Deployment volume mounts so the ConfigMap mounts both keys into `/etc/nginx/` (or whatever paths match the include directive).
5. Move `services/atlas-env.yaml` → `deploy/k8s/env-configmap.yaml`. No content changes.
6. Verify: `kubectl --dry-run=client apply -f deploy/k8s/namespace.yaml`, `kubectl --dry-run=client apply -f deploy/k8s/ingress.yaml`, etc. for each file.

## Phase 2 — Move per-service manifests

1. For each of the **54 services that have a manifest today** (all of `services/atlas-*` except `atlas-families` and `atlas-marriages`, per PRD §4.1), run:

   ```bash
   git mv services/atlas-<name>/atlas-<name>.yml deploy/k8s/atlas-<name>.yaml
   ```

   Filenames preserve the `atlas-` prefix (PRD §9 decision 1) and switch extension from `.yml` → `.yaml` for consistency with the new shared files.

2. Verify: `ls deploy/k8s/atlas-*.yaml | wc -l` reports `54`. Plus 5 shared files (`namespace.yaml`, `secrets.example.yaml`, `secrets.yaml` if already generated, `env-configmap.yaml`, `ingress.yaml`) = 58–59 files total in `deploy/k8s/` (59 once `secrets.yaml` exists).

3. Verify no manifest content changed: `git diff --stat HEAD~ deploy/k8s/` should show only adds/deletes (renames) and per-file diff should be empty.

4. Sanity-check against the live cluster: `kubectl --dry-run=server apply -f deploy/k8s/` (on a cluster you can target) must succeed before proceeding.

## Phase 3 — Create `deploy/compose/` skeleton

Create `deploy/compose/` with stub contents first, wire it up, then populate service definitions:

1. `deploy/compose/docker-compose.yml` (base file): shared `networks: atlas` block + `nginx` service (image `nginx:alpine`, mounts `./nginx.conf` and `./routes.conf`, publishes `${INGRESS_HOST_PORT:-8080}:80`, attached to network `atlas`).

2. `deploy/compose/nginx.conf`: compose-specific server/http header only (resolver `127.0.0.11`, `server_name _`, tenant/region `proxy_set_header` directives, underscore-headers setting, keepalive/timeout tuning) with an `include /etc/nginx/conf.d/routes.conf;` inside the `server {}` block. **Does not** contain any `location` blocks — those come from the shared file.

3. `deploy/compose/routes.conf`: create as a **symlink** to `../shared/routes.conf` so compose bind-mount reads the live shared source:

   ```bash
   (cd deploy/compose && ln -sf ../shared/routes.conf routes.conf)
   ```

4. `deploy/scripts/sync-k8s-ingress-routes.sh`: bash script that reads `deploy/shared/routes.conf`, re-indents it to match the YAML block-scalar indentation in `deploy/k8s/ingress.yaml`, and rewrites the `routes.conf: |` key's value block in-place. Idempotent; optionally supports `--check` to exit non-zero on drift (for future CI).

5. `deploy/compose/docker-compose.core.yml`: empty `services:` block to start. Populate in Phase 4.

6. `deploy/compose/docker-compose.socket.yml`: empty `services:` block. Populate in Phase 4.

7. Scripts: write `up.sh`, `down.sh`, `logs.sh` per PRD §4.5 — they read `.env` from `$SCRIPT_DIR/.env` (not repo root). Make executable (`chmod +x`).

8. `.env.example`: enumerate every variable the compose stack will eventually reference. Use sanitized placeholders.

## Phase 4 — Populate overlay files

For each service, add an entry to the appropriate overlay (`core` for 52, `socket` for `atlas-login` + `atlas-channel`). The full per-service `environment:` content is in PRD §4.10. Suggested minimum entry template:

```yaml
  atlas-<name>:
    container_name: atlas-<name>
    build:
      context: ../..
      dockerfile: services/atlas-<name>/Dockerfile
    image: atlas-<name>:${ATLAS_IMAGE_TAG:-local}
    env_file: .env                    # loads deploy/compose/.env
    environment:                      # per PRD §4.10 transcription table
      LOG_LEVEL: "debug"
      DB_NAME: "atlas-<name>"
    extra_hosts:
      - "postgres.home:host-gateway"
      - "kafka.home:host-gateway"
      - "redis.home:host-gateway"
      - "tempo.home:host-gateway"
    networks: [atlas]
    restart: unless-stopped
```

**Volume-mount additions.** Three services need explicit `volumes:` entries per PRD §4.3:

```yaml
  atlas-assets:
    # ...
    volumes:
      - ../../tmp/assets:/usr/assets

  atlas-data:
    # ...
    volumes:
      - ../../tmp/data:/usr/data

  atlas-wz-extractor:
    # ...
    volumes:
      - ../../tmp/wz-input:/usr/wz-input
      - ../../tmp/data:/usr/data
      - ../../tmp/assets:/usr/assets
```

Ensure `tmp/assets/` and `tmp/data/` exist (create empty) — `tmp/wz-input/` already exists per the current repo.

**Published ports.**

- `atlas-ui`: `3000:3000`
- `atlas-login`: `1200:1200`, `8300:8300`, `8700:8700`, `9200:9200`, `9500:9500`, `18500:18500`
- `atlas-channel`: `1201:1201`, `8301:8301`, `8701:8701`, `18501:18501`

No other services publish ports — they're reachable via the compose nginx only.

**Automation tip.** The template is mechanical; a small shell/awk script that reads each `services/atlas-<name>/atlas-<name>.yml` and emits a compose stanza will be faster and less error-prone than hand-authoring 52 entries. The env-survey approach from spec review (awk extracting `env:` block entries with literal `value:`) produces exactly the table in PRD §4.10. Skip env entries backed by `configMapRef` / `secretKeyRef` — those come from `.env` via `env_file`.

## Phase 5 — Generate real `.env` and `secrets.yaml`

1. Decode the base64 values in `base.yaml`:
   - `DB_USER: YXRsYXMgDQo=` → `atlas ` (note trailing space + CR — preserve exactly; the current secret contains `\r\n` artifacts that have been tolerated in production).
   - `DB_PASSWORD: YXRsYXMgDQo=` → same.

2. Compose `deploy/compose/.env` from:
   - Every key/value pair in `deploy/k8s/env-configmap.yaml` (flattened, `KEY=value` per line, double-quoted values unquoted).
   - Decoded `DB_USER` and `DB_PASSWORD`.
   - Compose-specific additions: `INGRESS_HOST_PORT=8080`, `ATLAS_IMAGE_TAG=local`.

3. Verify the file is gitignored: `git status` must not list it as untracked.

4. `deploy/k8s/secrets.yaml`: copy the `Secret/db-credentials` block out of the historical `base.yaml` verbatim, place it in `deploy/k8s/secrets.yaml`, confirm gitignored.

## Phase 6 — Reference updates + legacy cleanup

**Delete legacy per-service build scripts** (superseded by compose `build:`):

```bash
for svc in services/atlas-*/; do
  rm -f "$svc/docker-build.sh" "$svc/docker-build.bat"
done
git add -A services/
```

**Update in-repo references** (re-grep at implementation time to catch anything missed):

| File                                    | Change                                                                                                                                                  |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `README.md`                             | Replace deployment section paths (`base.yaml` → `deploy/k8s/namespace.yaml` + `deploy/k8s/secrets.yaml`; `atlas-ingress.yml` → `deploy/k8s/ingress.yaml` + `deploy/shared/routes.conf`; `services/atlas-<name>/atlas-<name>.yml` → `deploy/k8s/atlas-<name>.yaml`; `services/atlas-env.yaml` → `deploy/k8s/env-configmap.yaml`). Add a section describing `deploy/compose/` + `up.sh`/`down.sh`/`logs.sh` + the `deploy/shared/` single-source routes pattern. |
| `CLAUDE.md`                             | Re-grep and update.                                                                                                                                     |
| `DOCS.md`                               | Re-grep and update.                                                                                                                                     |
| `services/atlas-*/README.md`            | Sweep all 56 service READMEs for references to the old manifest path (`services/atlas-<name>/atlas-<name>.yml`), `base.yaml`, `atlas-ingress.yml`, `docker-build.sh`. Update or remove.                                                             |
| `services/atlas-*/docs/*.md`            | Same sweep across every `docs/` subfolder under each service.                                                                                           |
| `tools/debug-start.sh`, `debug-stop.sh` | Confirm `CONFIGMAP_NAME=atlas-ingress-configmap` still matches the ConfigMap `metadata.name` in `deploy/k8s/ingress.yaml` (unchanged). Update only if that metadata drifted. |
| `.github/workflows/*.yml`               | Re-grep. Current sweep shows no references.                                                                                                             |
| `.gitignore`                            | Already updated in Phase 0 (`deploy/compose/.env`, `deploy/k8s/secrets.yaml`). `tmp/` already covered (line 27).                                        |

Confirm the sweep: `grep -rnE "base\.yaml|atlas-ingress\.yml|services/atlas-env\.yaml|services/atlas-[a-z-]+/atlas-[a-z-]+\.yml|docker-build\.(sh|bat)"` must return only results inside `docs/tasks/task-001-deploy-reorg/` and the new `deploy/` tree.

## Phase 7 — End-to-end verification

1. **Compose up — core.** `./deploy/compose/up.sh core --build` from a clean `docker system prune -af` state. Expect ~10–20 minutes cold build; subsequent `up.sh core` completes in <60s. Confirm nginx answers `curl -I http://localhost:8080/` with a valid response and several services reply via `curl -H "TENANT_ID: <uuid>" -H "REGION: GMS0" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" http://localhost:8080/api/accounts`.
2. **Compose up — socket.** In a second terminal: `./deploy/compose/up.sh socket`. Confirm `nc -zv localhost 1200` succeeds (atlas-login) and `nc -zv localhost 1201` succeeds (atlas-channel). These are the pinned smoke-test ports per PRD §9 decision 11.
3. **Cross-overlay.** With both `core` and `socket` up, confirm `atlas-channel` can reach `atlas-character` via the `atlas` network (check `atlas-channel` logs for successful Kafka+REST interactions on a fresh login).
4. **Compose down.** `./deploy/compose/down.sh all` removes containers. The `atlas` network is removed only when no compose invocation still has services up.
5. **K8s apply.** On a test cluster: `kubectl apply -f deploy/k8s/namespace.yaml && kubectl apply -f deploy/k8s/secrets.yaml && kubectl apply -f deploy/k8s/env-configmap.yaml && kubectl apply -f deploy/k8s/ingress.yaml && kubectl apply -f deploy/k8s/` — the final apply of the whole folder should noop on the already-applied shared resources and create/update the 56 per-service resources. Smoke-test a few endpoints via the cluster ingress.

## Phase 8 — PR hygiene

1. Commit moves as their own commit (so `git log --follow` preserves history per file).
2. Commit new compose files + scripts in a second commit.
3. Commit documentation updates in a third commit.
4. Pre-push check: `git status` shows `deploy/compose/.env` and `deploy/k8s/secrets.yaml` as **ignored, not tracked**.
5. Open PR; the PR description links back to this task folder and calls out that real `.env` / `secrets.yaml` must be generated locally by each reviewer (provide decode commands in the PR body if helpful).
