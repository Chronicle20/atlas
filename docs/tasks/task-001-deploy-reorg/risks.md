# Risks — Deployment Reorganization

## R1. Real `secrets.yaml` / `.env` leaked via commit

**Likelihood:** Moderate. **Impact:** High (credential exposure).

A developer generates the real files for local use, then stages them accidentally (e.g., `git add .` from the repo root).

**Mitigations:**

- Add `.gitignore` entries **before** any generated file touches the working tree (Phase 0 of migration plan).
- After generation, run `git check-ignore -v deploy/compose/.env deploy/k8s/secrets.yaml` to confirm ignore coverage.
- Document the decode+generate commands in the PR description so reviewers don't copy the producer's local files through informal channels.
- Consider a `.gitattributes` entry flagging these paths as `-export-ignore` so `git archive` never bundles them even if they somehow get tracked.

## R2. Cluster applies stop working because of reordering

**Likelihood:** Low if file contents preserved. **Impact:** High (breaks cluster deploy workflow).

`kubectl apply -f <dir>` applies files in lexical order. Today `base.yaml` + `atlas-ingress.yml` are applied by explicit name in documented commands. After the reorg, if documentation or tooling starts doing `kubectl apply -f deploy/k8s/`, namespace-dependent resources may be applied before the namespace.

**Mitigations:**

- `deploy/k8s/namespace.yaml` sorts before every other file alphabetically (n < s). But `secrets.yaml` depends on the namespace and sorts *after* `namespace.yaml`, so a bare directory apply works by coincidence.
- Document the explicit two-step in README: apply `namespace.yaml` first, then the rest.
- Do not introduce any file that sorts before `namespace.yaml` without also being namespace-independent.

## R3. Cross-stack container name collisions

**Likelihood:** Low. **Impact:** Medium (local-dev friction).

If a developer runs `./up.sh core` in one terminal (project `atlas`) and `./up.sh socket` in another *without* the `--project-name atlas` flag (e.g., someone invokes `docker compose` directly), the two invocations become separate projects and containers with the same `container_name:` collide.

**Mitigations:**

- Enforce `--project-name atlas` inside `up.sh`/`down.sh`/`logs.sh` so casual users can't hit the footgun.
- Document: "never invoke `docker compose` directly for this stack, use the scripts." Put a note in a short `deploy/compose/README.md` (add only if content justifies it — else leave to the repo's main README).

## R4. `extra_hosts: host-gateway` doesn't reach the developer's infra

**Likelihood:** Moderate on non-default setups. **Impact:** Medium (local dev broken).

`host-gateway` maps to the Docker host's gateway, which works great when Postgres/Kafka run on the same host. If a developer runs infra on a different LAN machine, or behind a VPN, or with host-networked Docker Desktop peculiarities, resolution fails.

**Mitigations:**

- Document the override pattern explicitly: set `DB_HOST` (etc.) in `.env` to the reachable hostname/IP; remove or rewrite the `extra_hosts` entry if the IP resolution changes.
- Compose file's `extra_hosts` references variables where possible: `- "${DB_HOSTNAME:-postgres.home}:host-gateway"`.
- Add a troubleshooting section to the compose README.

## R5. Cold build exceeds developer patience

**Likelihood:** High on first run. **Impact:** Low (annoyance, not correctness).

56 services × full Go compile = tens of minutes on a fresh checkout.

**Mitigations:**

- Document the one-time cost in the compose README.
- Ensure `up.sh` defaults to `--build` on first run but supports `--no-build` for subsequent runs.
- Shipping a compose profile that pulls prebuilt `ghcr.io/chronicle20/...` images is a potential follow-up (out of scope for this task per PRD §9 decision 5) for contributors who don't need to iterate on code.

## R6. Debug-start.sh references broken by rename

**Likelihood:** Low. **Impact:** Medium (debugging workflow broken).

`tools/debug-start.sh` hardcodes `NAMESPACE=atlas`, `CONFIGMAP_NAME=atlas-ingress-configmap`, `INGRESS_DEPLOYMENT=atlas-ingress`. If the reorg accidentally changes any of these `metadata.name` values, the script breaks silently (it operates via `kubectl`, not by reading the file paths we're moving).

**Mitigations:**

- Reorg is a **move only**, not a rename of K8s resources. Content is preserved.
- Add a test-phase verification: `grep -E "name: atlas-ingress|name: atlas-ingress-configmap|namespace: atlas" deploy/k8s/ingress.yaml` succeeds.

## R7. Socket-server port conflicts on the host

**Likelihood:** Moderate. **Impact:** Medium (socket stack won't start).

`atlas-login`/`atlas-channel` publish multiple low-numbered and mid-range ports (1200, 1201, 8300/8301, 8700/8701, 18500/18501, 9200, 9500). Common ports (8300 is Consul's HTTP port; 9200 is Elasticsearch) may collide with other dev tools.

**Mitigations:**

- Document known-conflict ports and suggest `HOST_IFACE=127.0.0.1` to at least avoid LAN exposure.
- Allow per-port override via `.env` variables (`LOGIN_PORT_1200=1200`, etc.) if collisions are common. Default to 1:1.
- Honestly, conflicts are rare in practice — most devs don't run Consul + Elasticsearch + Atlas on the same box. Don't over-engineer.

## R8. `BASE_SERVICE_URL` misrouting in compose

**Likelihood:** High if the K8s value is used unchanged. **Impact:** High (internal REST calls fail).

Today `BASE_SERVICE_URL=http://atlas-ingress.atlas.svc.cluster.local:80/api/`. Inside compose, that hostname doesn't resolve.

**Mitigations:**

- Compose `.env` sets `BASE_SERVICE_URL=http://atlas-ingress:80/api/` (nginx container name).
- Acceptance test §10 validates `atlas-channel` → `atlas-tenants` call chain works end-to-end in compose.

## R9. Stale documentation references

**Likelihood:** Moderate. **Impact:** Low-to-medium (new contributors mislead).

README, DOCS, service-level docs may reference deployment paths that no longer exist.

**Mitigations:**

- Phase 6 of migration plan performs an exhaustive sweep.
- PR checklist item: grep for old paths, expect zero hits outside `docs/tasks/`.

## R10. `atlas-wz-extractor` volume mount paths wrong

**Likelihood:** Low. **Impact:** Low.

`atlas-wz-extractor` is confirmed long-running (PRD §9 decision 3) and mounts `tmp/wz-input:ro` + `tmp/assets`. If the repo layout changes or the in-container paths differ from `/tmp/wz-input` + `/tmp/assets`, mounts silently bind empty directories and extraction results vanish.

**Mitigations:**

- Verify in-container paths against the service's `main.go`/config before writing the compose entry.
- Confirm host-side `tmp/wz-input/` and `tmp/assets/` exist (they already do per the current repo).
- Smoke-test the service end-to-end after compose bring-up: place a known WZ file in `tmp/wz-input/`, verify extraction artifacts appear in `tmp/assets/`.

## R11. File-rename loses `git log --follow` history

**Likelihood:** High if `git mv` is skipped. **Impact:** Low (forensic annoyance).

If implementer uses `mv` + `git add` instead of `git mv`, Git may fail to detect the rename for files with heavy content changes, making historical spelunking harder.

**Mitigations:**

- Migration plan explicitly says `git mv`.
- The manifest content doesn't change in this task, so rename detection will work even with plain `mv` as long as added+deleted are in the same commit.

## R12a. nginx K8s ConfigMap drift from shared `routes.conf`

**Likelihood:** Moderate. **Impact:** High (K8s routing breaks silently).

A developer edits `deploy/shared/routes.conf`, tests in compose (bind-mount picks up the change live), and forgets to run `deploy/scripts/sync-k8s-ingress-routes.sh`. The K8s `ingress.yaml` ConfigMap still has the old inlined routes. When the K8s deploy happens, stale routes ship.

**Mitigations:**

- Document: editing routes requires running the sync script before commit.
- Add a `--check` flag to the sync script; run it as a pre-commit hook or CI step (future task; not in scope).
- Acceptance criterion: running `sync-k8s-ingress-routes.sh --check` on a fresh checkout of the reorg commit exits 0.
- Review-time check: PRs touching `deploy/shared/routes.conf` should also touch `deploy/k8s/ingress.yaml`.

## R12. Multi-file compose merge semantics surprise

**Likelihood:** Low. **Impact:** Low.

Developers unfamiliar with `-f file1 -f file2` layering may assume later files fully replace earlier ones. Compose actually deep-merges, with special rules for list fields like `environment` (append) vs. `ports` (replace).

**Mitigations:**

- Our base file only contains shared resources (network + nginx); overlays add new services (no key conflicts). Merge behavior is irrelevant for the happy path.
- Keep overlays strictly additive — never redeclare `networks` or `nginx` in a service overlay. That rule is easy to enforce in review.
