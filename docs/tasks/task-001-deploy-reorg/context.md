---
name: Deploy Reorg — Context
description: Key files, decisions, and dependencies for the deployment reorganization task.
type: context
task: task-001-deploy-reorg
---

# Context — Deployment Reorganization

Last Updated: 2026-04-16

## Key Files (current repo, to be touched)

### K8s manifests (moved/deleted)
- `base.yaml` — Namespace + `db-credentials` Secret. Split into `deploy/k8s/namespace.yaml` + `deploy/k8s/secrets{,.example}.yaml`.
- `atlas-ingress.yml` — Ingress Deployment + Service + ConfigMap. Becomes `deploy/k8s/ingress.yaml`; route bodies extracted to `deploy/shared/routes.conf`.
- `services/atlas-env.yaml` — shared ConfigMap. Moved to `deploy/k8s/env-configmap.yaml`.
- `services/atlas-<name>/atlas-<name>.yml` — 54 per-service manifests. `git mv` to `deploy/k8s/atlas-<name>.yaml` (extension normalized to `.yaml`).

### Build scripts (deleted)
- `services/atlas-<name>/docker-build.sh` — delete all.
- `services/atlas-<name>/docker-build.bat` — delete all.

### Documentation (swept for path updates)
- `README.md` — deployment section.
- `CLAUDE.md` — re-grep.
- `DOCS.md` — re-grep.
- `services/atlas-*/README.md` (×56) — sweep for old manifest/build paths.
- `services/atlas-*/docs/*.md` — sweep.
- `.github/workflows/*.yml` — re-grep (current sweep: no references).

### Tooling (verify, not rewrite)
- `tools/debug-start.sh`, `tools/debug-stop.sh` — hardcode `NAMESPACE=atlas`, `CONFIGMAP_NAME=atlas-ingress-configmap`, `INGRESS_DEPLOYMENT=atlas-ingress`. These `metadata.name` values do NOT change in this task; scripts should Just Work after the move. Verify with `grep -E "name: atlas-ingress|name: atlas-ingress-configmap|namespace: atlas" deploy/k8s/ingress.yaml`.

### .gitignore
- Add: `deploy/compose/.env`, `deploy/k8s/secrets.yaml`.
- Already covered: `tmp/` (line 27).

## Key Files (new, to be created)

- `deploy/k8s/{namespace,secrets.example,secrets,env-configmap,ingress}.yaml`
- `deploy/k8s/atlas-<name>.yaml` ×54
- `deploy/shared/routes.conf`
- `deploy/compose/docker-compose.yml` (base)
- `deploy/compose/docker-compose.core.yml` (52 services)
- `deploy/compose/docker-compose.socket.yml` (2 services)
- `deploy/compose/nginx.conf`
- `deploy/compose/routes.conf` (symlink → `../shared/routes.conf`)
- `deploy/compose/up.sh`, `down.sh`, `logs.sh`
- `deploy/compose/.env.example`, `.env` (gitignored)
- `deploy/scripts/sync-k8s-ingress-routes.sh`
- `tmp/assets/`, `tmp/data/` (empty dirs; `tmp/wz-input/` already exists)

## Service Inventory

**54 services with K8s manifest today** (all go into `deploy/k8s/`):

Core overlay (52, in `docker-compose.core.yml`):
`atlas-account, atlas-asset-expiration, atlas-assets, atlas-ban, atlas-buddies, atlas-buffs, atlas-cashshop, atlas-chairs, atlas-chalkboards, atlas-character, atlas-character-factory, atlas-configurations, atlas-consumables, atlas-data, atlas-drop-information, atlas-drops, atlas-effective-stats, atlas-expressions, atlas-fame, atlas-gachapons, atlas-guilds, atlas-inventory, atlas-invites, atlas-keys, atlas-map-actions, atlas-maps, atlas-merchant, atlas-messages, atlas-messengers, atlas-monster-death, atlas-monsters, atlas-notes, atlas-npc-conversations, atlas-npc-shops, atlas-parties, atlas-party-quests, atlas-pets, atlas-portal-actions, atlas-portals, atlas-query-aggregator, atlas-quest, atlas-rates, atlas-reactor-actions, atlas-reactors, atlas-saga-orchestrator, atlas-skills, atlas-storage, atlas-tenants, atlas-transports, atlas-ui, atlas-world, atlas-wz-extractor`

Socket overlay (2, in `docker-compose.socket.yml`):
`atlas-login, atlas-channel`

**Explicitly excluded (no K8s manifest today):**
`atlas-families`, `atlas-marriages`. Add when manifests are authored.

## Key Decisions (PRD §9)

| # | Decision |
| - | -------- |
| 1 | K8s filenames keep `atlas-` prefix; extension normalized to `.yaml`. |
| 2 | nginx host port default `8080`, overridable via `INGRESS_HOST_PORT`. |
| 3 | `atlas-wz-extractor` is long-running — include in core overlay, no profile gating. |
| 4 | Per-service `DB_NAME` literal in each service's compose `environment:` block. No `.env` proliferation. |
| 5 | Compose always builds (`build:` directive); no `ghcr.io` pull path in this task. |
| 6 | Cluster→compose ingress rerouting already supported by `tools/debug-start.sh --target`; no new tooling. |
| 7 | `atlas-ui` lives in core overlay, not its own file. |
| 8 | Stack selector is a positional arg on `up.sh`/`down.sh`/`logs.sh`: `core` (default), `socket`, `all`. |
| 9 | `.env` lives at `deploy/compose/.env` (not repo root); scripts use `$SCRIPT_DIR/.env`. |
| 10 | Single-source nginx routes via `deploy/shared/routes.conf`; `sync-k8s-ingress-routes.sh` keeps K8s ConfigMap in sync. |
| 11 | Smoke-test ports: TCP 1200 (login), 1201 (channel). |
| 12 | `atlas-families`, `atlas-marriages` excluded — no K8s manifest. |
| 13 | Shared-volume host path is `tmp/` at repo root. |

## External Dependencies

- **Host-provided infra:** Postgres (`postgres.home`), Kafka (`kafka.home:9093`), Redis (`redis.home`), Tempo (`tempo.home:4317`). Compose reaches them via `extra_hosts: <host>:host-gateway`.
- **Docker Engine ≥ 23** with BuildKit enabled.
- **For K8s smoke test:** a writable test cluster. Production cluster not required.

## Cross-System Dependencies (none)

- No service code changes, no Kafka topic changes, no DB migrations, no JSON:API schema changes, no library re-exports.
- No coordination with UI team or other service owners required — the change is internal to repo layout + local tooling.

## Sequence Dependencies within the Task

1. **Phase 0 MUST precede Phase 5.** `.gitignore` entries must exist before real `.env` or `secrets.yaml` hits the working tree.
2. **Phase 1 MUST precede Phase 2.** `deploy/k8s/` must exist before manifests are moved into it.
3. **Phase 1 `routes.conf` extraction MUST precede Phase 3 compose nginx wiring.** Compose bind-mounts the shared file; if it doesn't exist, nginx boot fails.
4. **Phase 3 overlay stubs MUST precede Phase 4.** Generator emits into existing overlay files.
5. **Phase 4 MUST precede Phase 7.** Can't verify a stack with no services defined.
6. **Phase 6 MUST follow all file creation/moves.** Sweep catches everything only once the target paths are stable.

## Reference Documents

- `prd.md` — product requirements, acceptance criteria, per-service env transcription table (§4.10).
- `migration-plan.md` — phase-by-phase execution sequence with verification commands.
- `compose-design.md` — rationale for file split, build caching, nginx single-source, host-gateway, port mapping.
- `risks.md` — R1–R12a risk register with mitigations.
- `CLAUDE.md` (project root) — multi-service build/verify guidance; re-check at implementation time.
- Home-hub reference: mirror `home-hub/deploy/` layout for k8s flat + compose + scripts conventions.
