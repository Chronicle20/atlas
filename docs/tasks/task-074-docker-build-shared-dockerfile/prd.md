# Docker Build Optimization via Shared Parameterized Dockerfile + buildx bake — PRD

Version: v1
Status: Draft
Created: 2026-05-21
---

## 1. Overview

Atlas currently maintains ~50 near-identical per-service Dockerfiles under `services/atlas-*/Dockerfile` (plus optional `Dockerfile.dev` and `Dockerfile.debug` variants for many of them). Each file is 80–88 lines of essentially the same template: pull `golang:1.25.5-alpine3.21`, install git, `COPY` every lib's `go.mod`/`go.sum`, synthesize a `go.work`, `go mod download`, `COPY` lib source trees, `go mod edit -replace=...` for every lib, `go mod tidy`, `go build`, then assemble a thin alpine runtime image. The only meaningful per-service variation is the service name itself. CLAUDE.md already calls out that adding a new lib dependency requires a four-place hand edit across each of these files and that `go build`/`go test` against `go.work` will not catch drift — only `docker build` will.

This duplication has two costs: (1) maintenance drag and a recurring source of CI-only failures when lib lists drift, and (2) CI build wall-time on PRs and `main` is large because the matrix fans out one runner per service, each paying runner provisioning, checkout, buildx setup, GHA cache restore, and `apk add git` overhead before `docker build` even starts. Lib bumps, `go.work` changes, and workflow changes amplify this to 50+ parallel cold-cache builds gated by the GHA concurrent-runner quota.

This task consolidates the ~50 Go-service Dockerfiles into one repo-root `Dockerfile` parameterized by `ARG SERVICE`, introduces a `docker-bake.hcl` enumerating all Go-service targets, switches PR-validation and main-publish CI from a per-service matrix to a single `docker buildx bake` invocation (still selecting only the changed services from `detect-changes`), and updates `deploy/compose/*.yml` to reference the shared Dockerfile. BuildKit cache mounts are added for `/go/pkg/mod` and `/root/.cache/go-build`. The `go mod edit -replace … && go mod tidy` block is evaluated empirically and removed if `go.work` alone suffices.

## 2. Goals

Primary goals:
- Eliminate the 50× duplication of per-service Go Dockerfiles. One Dockerfile, parameterized by service name.
- Reduce CI Docker-build wall time, particularly for PRs/main runs that touch many services (lib bumps, `go.work` changes, workflow changes).
- Remove the four-place lib-list drift hazard that CLAUDE.md currently documents.
- Preserve the existing CI selection model: `detect-changes` continues to drive which services build; this task only changes how those builds are executed.
- Keep local `docker compose build <service>` working with at most a mechanical update to compose files.

Non-goals:
- Migrating `services/atlas-ui/Dockerfile` (Node/nginx, unrelated build chain).
- Migrating `services/atlas-assets/Dockerfile`, `services/atlas-pr-bootstrap/Dockerfile`, and similar pure-nginx images that do not duplicate the Go build template.
- Switching CI cache backend from `type=gha` to `type=registry` (kept as a possible follow-up if measured wins justify it).
- Changing the runtime base image (alpine → distroless/scratch) or toggling `CGO_ENABLED`.
- Setting numeric CI wall-time targets. "Faster" is the goal; measurement and iteration are deferred.
- Sharding the consolidated bake job across multiple runners. One job, then observe.

## 3. User Stories

- As a backend engineer adding a new shared lib, I want to declare the dependency in exactly one place so I do not have to remember CLAUDE.md's four-place rule across 50 Dockerfiles.
- As an engineer pushing a PR that touches many services (lib bump, `go.work` change), I want CI Docker builds to complete faster than today's per-service matrix.
- As an engineer running CI on a single-service change, I want my build to be no slower than it is today.
- As an engineer building locally via `docker compose build <service>`, I want the command to work with at most a mechanical config update — no per-service knowledge to acquire.
- As an engineer running the mandatory `docker build` verification step before opening a PR, I want a single, well-known command that builds one service or all changed services.
- As an engineer reading CLAUDE.md, I want the "four-place lib list" verification rule to either disappear or shrink to a one-place rule that is hard to get wrong.

## 4. Functional Requirements

### 4.1 Shared Dockerfile

- A single `Dockerfile` at the repo root (path TBD in design — `Dockerfile` at root or `build/Dockerfile`).
- Accepts a build arg `SERVICE` whose value is the directory name under `services/` (e.g., `atlas-account`).
- Fails fast with a clear error early in the build if `SERVICE` is unset or if `services/${SERVICE}` does not exist.
- Discovers the inner Go module directory under `services/${SERVICE}/atlas.com/<name>/` rather than hard-coding it. Naming pattern is consistent across services and can be globbed/derived.
- `COPY`s the union of all atlas libs' `go.mod`/`go.sum`, then `COPY`s all atlas libs' source trees. The same Dockerfile is used for every service, so it must include every lib that any service consumes. Layer ordering must keep mod-only layers cacheable independently of source.
- Uses BuildKit `RUN --mount=type=cache,target=/go/pkg/mod` and `--mount=type=cache,target=/root/.cache/go-build` on `go mod download` and `go build` steps.
- Produces an output image identical in behavior to today's per-service image: same runtime base (`alpine:3.23`), same `EXPOSE`, same `CMD ["/server"]`, same `config.yaml` location at `/config.yaml`.
- Tags continue to flow through CI's existing tagging logic (`pr-<N>-<sha>`, `latest`, etc.); no change to how `image-name` and `tags` inputs are computed.

### 4.2 `go mod edit -replace` + `go mod tidy` evaluation

The current per-service Dockerfile has this block:

```
RUN cd services/<svc>/atlas.com/<name> && \
    go mod edit -replace=github.com/Chronicle20/atlas/libs/atlas-X=/app/libs/atlas-X ... \
    && go mod tidy
```

Implementation must empirically verify whether `go.work use(...)` alone resolves the unreachable `github.com/Chronicle20/atlas/libs/*` paths. If `go build` succeeds against `go.work` without the replace block, remove the entire block. If only `go mod tidy` is unnecessary but the `replace` lines are still required, keep `replace` and drop `tidy`. The decision and supporting evidence (the failing/passing build commands) is recorded in the design doc; the PRD does not pre-commit to either outcome.

### 4.3 `go.work` synthesis

The current Dockerfile synthesizes `go.work` inline via `RUN echo ... > go.work`. Implementation may either:
- Replace this with a static `go.work` file `COPY`ed in (small layer, always cache-hits), or
- Continue to synthesize it from a template if the service path varies.

Either is acceptable as long as the result is deterministic and cache-friendly.

### 4.4 docker-bake.hcl

- A `docker-bake.hcl` (or `docker-bake.json`) at the repo root enumerating one target per Go service.
- Each target sets `context = "."`, `dockerfile = "Dockerfile"`, `args = { SERVICE = "atlas-<name>" }`, and a `tags` value driven by an environment variable or HCL variable so CI can inject `image-name:tag` per service.
- A bake group named (e.g.) `all-go-services` containing every Go-service target.
- A way to invoke bake with a subset of targets passed from CI's `detect-changes` matrix output.
- Out of scope for the bake file: atlas-ui, atlas-assets, atlas-pr-bootstrap, atlas-wz-extractor's nginx portion (note: atlas-wz-extractor *is* a Go service per the existing Dockerfile and IS folded in).

### 4.5 Old Dockerfile removal

- All `services/atlas-*/Dockerfile` files corresponding to Go services in scope are deleted in this PR.
- All `services/atlas-*/Dockerfile.dev` and `services/atlas-*/Dockerfile.debug` files are deleted in this PR. (User confirmed: these can be removed; the design phase decides whether to fold their behavior into the shared Dockerfile as additional stages or to drop them entirely. If kept, they become additional bake targets or additional `--target` stages within the shared Dockerfile.)
- `services/atlas-ui/Dockerfile`, `services/atlas-assets/Dockerfile`, `services/atlas-pr-bootstrap/Dockerfile` remain untouched.

### 4.6 CI changes (PR validation + main publish)

- `.github/workflows/pr-validation.yml` `build-docker` job: replace the per-service matrix with a single job that invokes `docker buildx bake` against the targets selected by `detect-changes`.
- `.github/workflows/main-publish.yml`: equivalent change for the main-branch publish path.
- `.github/actions/docker-build/action.yml`: replaced or rewritten to wrap `docker buildx bake` instead of `docker/build-push-action@v6`. Inputs adjusted accordingly: the action takes a target list (or a bake file + group) rather than a single `context`/`dockerfile`.
- The PR-overlay-resolve job (`update-pr-overlay`) continues to consume `detect-changes.outputs.docker-services-matrix` to know which `images:` entries to bump. That matrix shape stays the same.
- GHA cache scope: collapse from per-service scopes (`<svc>-amd64`) to one or a small number of shared scopes. Specific scope key chosen in design.

### 4.7 compose files

- `deploy/compose/docker-compose.yml`, `docker-compose.core.yml`, `docker-compose.socket.yml`: every Go-service `build:` block updated to reference the shared Dockerfile and pass `SERVICE` via `build.args`.
- `atlas-assets`'s non-standard `build.context: ../../services/atlas-assets` is left alone (it's out of scope; it points at the nginx Dockerfile which stays).
- Image tagging via `${ATLAS_IMAGE_TAG:-local}` is unchanged.
- `docker compose build <service>` and `docker compose up` must continue to work without additional flags.

### 4.8 Tool / script cleanup

- `tools/inject-dockerfile-replace.sh`, `tools/import-service.sh`, `tools/import-lib.sh`: audit and either update to match the new layout or remove if their purpose was tied to maintaining the per-service Dockerfile template. Design phase decides per-tool.
- `tools/build-services.sh`: audit; may become trivially `docker buildx bake all-go-services`.

### 4.9 Documentation updates

- `CLAUDE.md`'s "Build & Verification" section: rewrite the four-place-lib-list paragraph. The new instruction is to update the shared Dockerfile (one place) and run `docker buildx bake <service>` (or the equivalent local command) to verify.
- Mention `docker buildx bake all-go-services` as the local "build everything" command.

## 5. API Surface

No HTTP/API surface change. The change is internal to the build system.

Build CLI surface:
- `docker buildx bake <target>` for a single service.
- `docker buildx bake all-go-services` (or equivalent group) for everything.
- `docker compose build <service>` continues to work after compose-file updates.
- `docker build -f Dockerfile --build-arg SERVICE=atlas-<name> .` works as a fallback that bypasses bake.

## 6. Data Model

N/A. No data model changes.

## 7. Service Impact

| Area | Change |
|------|--------|
| All Go services (~50) | Per-service `Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug` deleted. Runtime image bytes unchanged (same alpine base, same `/server` binary, same `config.yaml`). |
| atlas-ui | Untouched. |
| atlas-assets | Untouched. |
| atlas-pr-bootstrap | Untouched (if Go-build-template-shaped, fold in; if pure-nginx-shaped, leave alone — design phase confirms). |
| atlas-wz-extractor | Folded into shared Dockerfile (it is a Go service with the same template). |
| `deploy/compose/*.yml` | `build.context` + `build.dockerfile` + new `build.args.SERVICE` for every Go-service block. |
| `.github/workflows/pr-validation.yml` | `build-docker` job switched to single bake invocation. |
| `.github/workflows/main-publish.yml` | Equivalent switch. |
| `.github/actions/docker-build/action.yml` | Replaced/rewritten. |
| `tools/*.sh` | Audit + update or delete. |
| `CLAUDE.md` | Build & Verification section rewritten. |

## 8. Non-Functional Requirements

### 8.1 Performance

- Single-service CI Docker build should be no slower than today's per-service matrix build (accounting for the fact that the new path adds one bake-startup cost but removes per-runner setup cost).
- Multi-service CI Docker build (e.g., lib bump touching 20+ services) should be measurably faster than today, primarily because runner provisioning + checkout + buildx setup + cache restore are paid once instead of N times, and the shared base layers (golang base pull, lib `go.mod` COPYs, `go mod download`) are computed once instead of N times.
- No numeric SLO. The acceptance criterion is "the maintainer can observe a real reduction in wall time on lib-bump-style PRs." Measurement strategy is recorded in the design doc.

### 8.2 Correctness

- Built images must be byte-equivalent (or behaviorally equivalent) to today's images for the same source. Specifically: same `CMD`, same `EXPOSE`, same `/config.yaml` content, same `/server` binary build flags.
- The `docker build` mandatory-verification rule in CLAUDE.md continues to provide drift detection — now for one Dockerfile instead of 50.

### 8.3 CI ergonomics

- One bake job log instead of per-service tiles is acceptable (user confirmed). The bake job's log must clearly separate per-target output so a failure points unambiguously at the offending service.
- `detect-changes` selection model is preserved; PRs that touch no services still skip the Docker job.

### 8.4 Local dev ergonomics

- Local `docker compose build <service>` works after the compose file update with no additional setup.
- The bake command works on a fresh clone with no setup other than Docker + buildx (already required today).

### 8.5 Multi-tenancy / security / observability

- N/A for this task. Build-system change; no runtime tenant boundary, secret handling, or observability surface is touched.

## 9. Open Questions

These are surfaced for the design phase, not blockers for the PRD:

1. Where does the shared Dockerfile live — repo root (`./Dockerfile`) or a build directory (`./build/Dockerfile`)? Either works; design picks.
2. `go.work` strategy — static file `COPY`ed in, vs synthesized inline. Either works; design picks based on whether the service path needs to vary.
3. Dockerfile.dev / Dockerfile.debug — confirmed deletable, but the design phase decides whether to (a) drop entirely, (b) re-create as additional stages of the shared Dockerfile guarded by `--target dev` / `--target debug`, or (c) re-create as additional bake targets. The choice should be informed by whether any current workflow actually uses the dev/debug variants.
4. atlas-pr-bootstrap — Go-template-shaped or pure-nginx-shaped? Spot check in design and decide.
5. Whether to keep the `go mod edit -replace ...` directives even after confirming `go.work` resolves the libs locally. Design phase tests both configurations and picks the simplest working one.
6. GHA cache scope key — one shared scope (`atlas-bake-amd64`) or grouped by lib-fingerprint? Single shared scope is the simplest starting point.
7. `tools/build-services.sh` and friends — full audit happens in design.
8. Migration plan for the single-PR cutover: any need to keep the old Dockerfiles temporarily reachable via a feature flag on CI, or just rip-and-replace? Default is rip-and-replace per user decision.

## 10. Acceptance Criteria

- [ ] One shared `Dockerfile` (parameterized by `ARG SERVICE`) exists at the agreed location, builds every in-scope Go service to a behaviorally equivalent image.
- [ ] All per-service `services/atlas-*/Dockerfile` files for in-scope Go services are deleted.
- [ ] All `services/atlas-*/Dockerfile.dev` and `services/atlas-*/Dockerfile.debug` files are deleted (or replaced by stages/targets in the shared Dockerfile, per design).
- [ ] `services/atlas-ui/Dockerfile`, `services/atlas-assets/Dockerfile`, and any other pure-nginx Dockerfiles confirmed out of scope are untouched.
- [ ] `docker-bake.hcl` (or `.json`) at repo root defines a target per Go service plus an `all-go-services` group.
- [ ] BuildKit cache mounts for `/go/pkg/mod` and `/root/.cache/go-build` are present on `go mod download` and `go build` steps.
- [ ] The `go mod edit -replace` + `go mod tidy` block is either removed or has its inclusion justified in the design doc with concrete evidence.
- [ ] `.github/workflows/pr-validation.yml` `build-docker` job is a single bake invocation; `detect-changes` selection logic still drives which targets build; the existing tagging behavior (`pr-<N>-<sha>`, `latest`, etc.) is preserved; the `update-pr-overlay` job continues to receive `docker-services-matrix` and bump the correct image tags.
- [ ] `.github/workflows/main-publish.yml` has the equivalent change.
- [ ] `.github/actions/docker-build/action.yml` is rewritten or replaced to wrap bake.
- [ ] All three `deploy/compose/*.yml` files build Go services via the shared Dockerfile with `build.args.SERVICE`. `docker compose build <service>` and `docker compose up` work on a fresh clone.
- [ ] `tools/inject-dockerfile-replace.sh`, `tools/import-service.sh`, `tools/import-lib.sh`, `tools/build-services.sh` are audited; obsolete ones removed, surviving ones updated.
- [ ] CLAUDE.md "Build & Verification" section rewritten. The four-place-lib-list paragraph is gone or reduced to a one-place instruction.
- [ ] A representative single-service PR build runs through CI green.
- [ ] A representative multi-service change (e.g., a lib `go.mod` touch that triggers many services) runs through CI green and the maintainer observes a noticeable reduction in wall time vs the pre-change matrix path. (Qualitative; no numeric SLO.)
- [ ] PR overlay resolution (`update-pr-overlay`) still produces a correct `bot/pr-<N>-resolved` branch with the right image tags for the built services.
