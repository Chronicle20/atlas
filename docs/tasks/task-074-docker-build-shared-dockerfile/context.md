# Task 074 — Context

> Companion to `prd.md`, `design.md`, `plan.md`. Read first to load the moving parts before executing.

## 1. Task at a glance

Consolidate ~50 per-service Go Dockerfiles into **one** parameterized `Dockerfile` at the repo root, driven by a `docker-bake.hcl` whose target list is materialized from `.github/config/services.json`. Collapse the per-service Docker CI matrix in `pr-validation.yml` and `main-publish.yml` into a single bake invocation, rewrite `.github/actions/docker-build/action.yml` to wrap bake, update `deploy/compose/*.yml` build blocks to reference the shared Dockerfile via `build.args.SERVICE`, retire `tools/inject-dockerfile-replace.sh`, simplify `tools/build-services.sh`, and rewrite the `CLAUDE.md` "Build & Verification" section.

This is a **rip-and-replace** PR: the old per-service Dockerfiles + dev/debug variants are deleted in the same commit set that lands the shared file. There is no flag, no transition window.

## 2. In-scope services (54 Go services)

Source of truth: `.github/config/services.json` (`.services[] | select(.type=="go-service")`).

Confirmed list (`jq -r '.services[] | select(.type=="go-service") | .name' .github/config/services.json`):

```
atlas-account               atlas-marriages
atlas-asset-expiration      atlas-merchant
atlas-ban                   atlas-messages
atlas-buddies               atlas-messengers
atlas-buffs                 atlas-monster-book
atlas-cashshop              atlas-monster-death
atlas-chairs                atlas-monsters
atlas-chalkboards           atlas-notes
atlas-channel               atlas-npc-conversations
atlas-character             atlas-npc-shops
atlas-character-factory     atlas-parties
atlas-configurations        atlas-party-quests
atlas-consumables           atlas-pets
atlas-data                  atlas-portal-actions
atlas-drop-information      atlas-portals
atlas-drops                 atlas-query-aggregator
atlas-effective-stats       atlas-quest
atlas-expressions           atlas-rates
atlas-fame                  atlas-reactor-actions
atlas-families              atlas-reactors
atlas-gachapons             atlas-saga-orchestrator
atlas-guilds                atlas-skills
atlas-inventory             atlas-storage
atlas-invites               atlas-tenants
atlas-keys                  atlas-transports
atlas-login                 atlas-world
atlas-map-actions           atlas-wz-extractor
atlas-maps
```

Count: **54** (atlas-wz-extractor is folded in per design §1).

## 3. Out of scope (untouched)

- `services/atlas-ui/Dockerfile` (Next.js / nginx).
- `services/atlas-assets/Dockerfile` (`type: static-service`; pure nginx).
- `services/atlas-pr-bootstrap/Dockerfile` (alpine + rpk; not Go-template-shaped; spot-checked in design §2.7).
- Cache backend switch, runtime base change, CGO toggles, numeric SLOs, bake-job sharding (PRD §2 non-goals).

## 4. The 17 atlas libs (full set; statically enumerated in the shared Dockerfile)

```
atlas-constants     atlas-rest
atlas-database      atlas-retry
atlas-kafka         atlas-saga
atlas-lock          atlas-script-core
atlas-model         atlas-service
atlas-object-id     atlas-socket
atlas-opcodes       atlas-tenant
atlas-packet        atlas-tracing
atlas-redis
```

Source: `ls libs/`. Every shared Dockerfile target COPYs the union; runtime image is unchanged (libs only exist in the build stage).

## 5. Inner-module naming pattern

For every service `services/<svc>/atlas.com/<inner>/`:

- Most services: `<inner>` matches the short-name (`atlas-account` → `account`).
- Edge cases (verified against `go.work`):
  - `atlas-drop-information` → `dis`
  - `atlas-monster-death` → `monster`
  - `atlas-npc-conversations` → `npc`
  - `atlas-npc-shops` → `npc`
  - `atlas-families` → `family`

The shared Dockerfile uses `ls -d services/${SERVICE}/atlas.com/*/ | head -1` to discover the inner directory at build time, so these edge cases need no per-service config (design §3.2).

## 6. CI surface — exact files touched

### Workflows

- `.github/workflows/pr-validation.yml` — `build-docker` job: collapse `strategy.matrix.service` to a single job that invokes bake against the targets from `detect-changes.outputs.docker-services-matrix`. Per-PR tag computation and the `deploy-env`-label-gated push semantics stay verbatim.
- `.github/workflows/main-publish.yml` — `build-amd64`, `build-arm64`, `create-manifest` jobs each collapse. `create-manifest` stays a per-service loop (no build, just `docker manifest create/push`).

### Composite actions

- `.github/actions/docker-build/action.yml` — rewritten. New inputs: `targets` (JSON array), `image-name-map` (JSON object), `tags` (comma-separated tag list), `platform`, `push`, `cache-scope`, registry creds. Internally generates `--set "${target}.tags=…"` flags and runs `docker buildx bake`.
- `.github/actions/detect-changes/action.yml` — **unchanged**. Its `docker-services-matrix` output already provides `{name, path, docker_context, docker_image}` — the bake-wrapping action consumes that shape directly.

### Cache scopes

- Today: `${{ matrix.service.name }}-amd64` and `${{ matrix.service.name }}-arm64` (one per service per arch ≈ 108 scopes).
- After: two shared scopes — `atlas-bake-amd64`, `atlas-bake-arm64`.

## 7. Compose blocks — counts

- `deploy/compose/docker-compose.yml` — **0** go-service build blocks (only `nginx`). No edits.
- `deploy/compose/docker-compose.core.yml` — **52** go-service build blocks (every go service except `atlas-login` and `atlas-channel`).
- `deploy/compose/docker-compose.socket.yml` — **2** go-service build blocks (`atlas-login`, `atlas-channel`).
- `atlas-assets`'s block uses `context: ../../services/atlas-assets` + service-local `Dockerfile` → unchanged.

Each go-service block changes from:

```yaml
build:
  context: ../..
  dockerfile: services/atlas-<svc>/Dockerfile
```

to:

```yaml
build:
  context: ../..
  dockerfile: Dockerfile
  args:
    SERVICE: atlas-<svc>
```

## 8. Tools — disposition table

| Tool | Today | After |
|------|-------|-------|
| `tools/inject-dockerfile-replace.sh` | Injects the per-Dockerfile `go mod edit -replace` block | **Deleted** — single Dockerfile makes injection meaningless |
| `tools/import-lib.sh` | Imports a new lib repo into `libs/` | **Updated docstring** noting the new "append two COPY lines to /Dockerfile + add to go.work" rule. No automation added in this task (manual edit is one append per surface). |
| `tools/import-service.sh` | Imports a new service repo into `services/` | **Updated docstring** noting that no Dockerfile generation is needed; just append to `services.json` + `go.work`. |
| `tools/build-services.sh` | Loops `docker build` per service Dockerfile | **Rewritten** to a one-liner: `exec docker buildx bake all-go-services "$@"` |
| `tools/test-all-go.sh`, `tools/tidy-all-go.sh`, `tools/scripts/*`, `tools/cideps/*`, `tools/packet-audit/*`, etc. | Unrelated | Untouched |

## 9. Key design decisions to remember (cross-ref design.md)

| # | Decision | Where |
|---|----------|-------|
| 1 | Shared Dockerfile lives at `./Dockerfile` (repo root) | §3.1 |
| 2 | `docker-bake.hcl` at repo root, matrix expanded from `.github/config/services.json` via `jsondecode(file(...))` | §3.4 |
| 3 | `# syntax=docker/dockerfile:1.24` directive (matches existing `services/atlas-pr-bootstrap/Dockerfile`) | §3.2 |
| 4 | Builder base `golang:${GO_VERSION}-alpine${ALPINE_VERSION}` with defaults `1.25.5` / `3.21` | §3.2 |
| 5 | Runtime base `alpine:3.23` + `apk add libc6-compat` (unchanged from today) | §3.2 |
| 6 | `COPY go.work go.work.sum ./` instead of synthesizing inline | §3.3 |
| 7 | All 17 libs `COPY`d (mod-only block + source block, in that order) for every target | §3.2 |
| 8 | Inner-module discovery: `MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1)` with `test -f $MOD_DIR/go.mod` guard | §3.2 |
| 9 | Build invocation: `go build -C "$MOD_DIR" -o /server` with BuildKit cache mounts on `/go/pkg/mod` and `/root/.cache/go-build` | §3.2 |
| 10 | `RUN cp "$MOD_DIR/config.yaml" /app/config.yaml` + `COPY --from=build-env /app/config.yaml /` in runtime stage (config path varies by service so it's resolved at build time) | §3.2 |
| 11 | Empirical test: drop the `go mod edit -replace … && go mod tidy` block; if `go.work use(...)` doesn't resolve, reinstate `-replace` (still drop `tidy`) | §2.3, §3.2 |
| 12 | `Dockerfile.dev` / `Dockerfile.debug` deleted outright; not folded into the shared file as `--target` stages | §2.6 |
| 13 | Two GHA cache scopes total: `atlas-bake-amd64`, `atlas-bake-arm64` | §3.4 |
| 14 | Migration: rip-and-replace in one atomic PR; revert = clean rollback | §6 |

## 10. Equivalence smoke tests (per design §4)

Per-service smoke executed during plan execution for at least:

1. `atlas-account` — REST + DB representative.
2. `atlas-channel` — socket-service representative (different inner module shape: still `channel`, but socket service path).

For each: `docker inspect` of the new image must match the old image's `Cmd`, `ExposedPorts`, `WorkingDir`, `Env`, and the running `/server` binary must start (or print its `--help` cleanly) under `docker run --rm`.

## 11. Hazards to watch during execution

| Hazard | Detection | Response |
|--------|-----------|----------|
| `go.work` resolution fails without `-replace` block | First `docker buildx bake atlas-account` errors with `cannot find module providing package github.com/Chronicle20/atlas/libs/...` | Reinstate parameterized `-replace` block per design §2.3 fallback; still drop `tidy` |
| `go.work` `use(...)` warning for missing service dirs in single-target builds | `go: warning: directory ./services/atlas-X does not exist` during bake of service Y | Tolerate (warnings, not errors). If a warning becomes an error, slim go.work inline per design §3.3 mitigation |
| Inner-module glob picks the wrong directory | `test -f $MOD_DIR/go.mod` guard fails fast in the build | Investigate which service violates the one-inner-dir convention; fix per-service rather than per-Dockerfile |
| `update-pr-overlay` job loses dependency on `build-docker` | Argo overlay never gets resolved image tags | The `update-pr-overlay` job's `needs: [detect-changes, build-docker]` and consumption of `docker-services-matrix` must remain literally unchanged |
| Per-PR tag (`pr-<N>-<sha>`) push semantics drift | Image with wrong tag pushed, or tag missing entirely | Verify the bake-action's `--set "<target>.tags=<image>:<tag>"` produces the same `<image>:<tag>` strings the current matrix loop produces; smoke against a one-service deploy-env PR |
| `create-manifest` job picks wrong arch tags | `docker manifest create` references non-existent tags | The manifest job's loop body is unchanged; only its trigger (now a single bake completing instead of N) is collapsed |
| Bake parallelism saturates 2-vCPU runner on large PRs | `go build` steps slow dramatically; runner OOMs | Add `--parallelism 4` to the bake invocation if observed; sharding is a deferred non-goal |
| Cache mount semantics confusion | `/go/pkg/mod` cache appears empty across runs on ephemeral runners | Cache mounts only persist intra-builder; the layer cache (`type=gha`) handles cold-start. This is expected — don't chase it. |

## 12. Verification gate (mandatory per CLAUDE.md "Build & Verification")

Before invoking `superpowers:finishing-a-development-branch`:

1. `go test -race ./...` in every changed Go module (no Go module is functionally changed, but `go.work` editions and any `go.work.sum` regeneration may dirty the workspace — run anyway as a sanity check).
2. `go vet ./...` clean in every changed module.
3. `go build ./...` clean in every changed service.
4. **`docker buildx bake atlas-<svc>`** from the worktree root for every service whose Dockerfile is being replaced (which is all 54). Practically: `docker buildx bake all-go-services` once after the cutover and confirm every target succeeds.

## 13. Files inventory — at-a-glance

**Created**
- `Dockerfile` (repo root)
- `docker-bake.hcl` (repo root)

**Modified**
- `.github/actions/docker-build/action.yml`
- `.github/workflows/pr-validation.yml`
- `.github/workflows/main-publish.yml`
- `deploy/compose/docker-compose.core.yml`
- `deploy/compose/docker-compose.socket.yml`
- `tools/build-services.sh`
- `tools/import-lib.sh` (docstring only)
- `tools/import-service.sh` (docstring only)
- `CLAUDE.md` (Build & Verification section)

**Deleted**
- `services/atlas-*/Dockerfile` × 54 (every entry in §2 above)
- `services/atlas-*/Dockerfile.dev` × N (every existing one — `find services -maxdepth 2 -name 'Dockerfile.dev'`)
- `services/atlas-*/Dockerfile.debug` × N (every existing one)
- `tools/inject-dockerfile-replace.sh`

**Untouched**
- `services/atlas-ui/Dockerfile`
- `services/atlas-assets/Dockerfile`
- `services/atlas-pr-bootstrap/Dockerfile`
- `.github/actions/detect-changes/action.yml`
- `.github/config/services.json`
- `go.work`
- `deploy/compose/docker-compose.yml`
- All other `tools/*` scripts
