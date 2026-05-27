# Docker Build Optimization via Shared Parameterized Dockerfile + buildx bake — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ~50 per-service `services/atlas-*/Dockerfile` files with a single repo-root `Dockerfile` parameterized by `ARG SERVICE`, driven by `docker-bake.hcl` and a collapsed CI matrix, so adding a new shared lib is a one-place edit and CI builds for lib-bump-style PRs run measurably faster.

**Architecture:** One shared multi-stage `Dockerfile` at repo root copies the union of all 17 atlas libs + the requested service's source, discovers the inner module path via glob, builds against the repo-root `go.work` (no per-service synthesis), and emits a behaviorally identical runtime image. `docker-bake.hcl` enumerates one target per `type == "go-service"` entry in `.github/config/services.json` via HCL `jsondecode(file(...))`. CI's `detect-changes` action is untouched; the `docker-build` composite action is rewritten to wrap `docker buildx bake` with per-target tag injection; the matrix `build-docker` (PR) and `build-amd64`/`build-arm64` (main) jobs collapse to single-job invocations. Per-service Dockerfiles, `Dockerfile.dev`/`Dockerfile.debug` variants, and the `inject-dockerfile-replace.sh` helper are deleted in the same PR. Compose files reference the shared Dockerfile via `build.args.SERVICE`.

**Tech Stack:** Docker BuildKit (syntax `1.24`), `docker buildx bake` (HCL config), GitHub Actions, Go 1.25.5 workspace mode (`go.work`), Alpine 3.21 (builder) / 3.23 (runtime).

---

## Pre-flight

### Pre-flight Task A: Verify worktree state

**Files:** none (sanity check)

- [ ] **Step 1: Confirm cwd**

Run: `pwd`
Expected: `.worktrees/task-074-docker-build-shared-dockerfile`

If wrong, `cd .worktrees/task-074-docker-build-shared-dockerfile`.

- [ ] **Step 2: Confirm branch**

Run: `git branch --show-current`
Expected: `task-074-docker-build-shared-dockerfile`

If wrong, STOP. Investigate before doing anything.

- [ ] **Step 3: Confirm prerequisite artifacts exist**

Run:
```bash
ls docs/tasks/task-074-docker-build-shared-dockerfile/{prd.md,design.md,context.md,plan.md}
```
Expected: all four files listed.

- [ ] **Step 4: Confirm tree is clean (no untracked or modified files outside .worktrees)**

Run: `git status --short`
Expected: empty output, or only this `plan.md`/`context.md` if you reach this step before the first commit.

If the worktree has modifications from earlier exploration, decide explicitly before proceeding (likely `git stash` or `git checkout --` after diffing).

---

## Phase 1 — Build the shared Dockerfile and prove one service builds

The goal of Phase 1 is to land a working shared `Dockerfile` that builds **one** service (atlas-account) on the current developer machine, before touching any CI or compose surface. Phase 1 is the gate that empirically decides whether the `go mod edit -replace` block from design §2.3 is needed.

### Task 1: Create the shared `Dockerfile` at repo root

**Files:**
- Create: `Dockerfile`

- [ ] **Step 1: Write the shared Dockerfile**

Create `Dockerfile` (repo root) with these exact contents:

```dockerfile
# syntax=docker/dockerfile:1.24
#
# Shared Atlas Dockerfile. One file builds every Go service in
# .github/config/services.json (.services[] | select(.type=="go-service")).
#
# Usage:
#   docker build -f Dockerfile --build-arg SERVICE=atlas-<name> .
# Preferred:
#   docker buildx bake atlas-<name>
# Build everything:
#   docker buildx bake all-go-services
#
# Adding a new shared lib requires appending two COPY lines to this
# file (one to the mod-only block, one to the source block) AND adding
# the lib to /go.work. That's it — no per-service edits.
ARG GO_VERSION=1.25.5
ARG ALPINE_VERSION=3.21

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env

ARG SERVICE
RUN test -n "${SERVICE}" || (echo "ERROR: build arg SERVICE is required (e.g., atlas-account)" >&2 && exit 1)

RUN apk add --no-cache git

WORKDIR /app

# Layer: repo go.work (cheap; invalidates when libs or services are added/removed).
COPY go.work go.work.sum ./

# Layer: all 17 atlas libs' go.mod/go.sum (lib-mod-only layer; shared across every target).
COPY libs/atlas-constants/go.mod   libs/atlas-constants/go.sum   libs/atlas-constants/
COPY libs/atlas-database/go.mod    libs/atlas-database/go.sum    libs/atlas-database/
COPY libs/atlas-kafka/go.mod       libs/atlas-kafka/go.sum       libs/atlas-kafka/
COPY libs/atlas-lock/go.mod        libs/atlas-lock/go.sum        libs/atlas-lock/
COPY libs/atlas-model/go.mod       libs/atlas-model/go.sum       libs/atlas-model/
COPY libs/atlas-object-id/go.mod   libs/atlas-object-id/go.sum   libs/atlas-object-id/
COPY libs/atlas-opcodes/go.mod     libs/atlas-opcodes/go.sum     libs/atlas-opcodes/
COPY libs/atlas-packet/go.mod      libs/atlas-packet/go.sum      libs/atlas-packet/
COPY libs/atlas-redis/go.mod       libs/atlas-redis/go.sum       libs/atlas-redis/
COPY libs/atlas-rest/go.mod        libs/atlas-rest/go.sum        libs/atlas-rest/
COPY libs/atlas-retry/go.mod       libs/atlas-retry/go.sum       libs/atlas-retry/
COPY libs/atlas-saga/go.mod        libs/atlas-saga/go.sum        libs/atlas-saga/
COPY libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/
COPY libs/atlas-service/go.mod     libs/atlas-service/go.sum     libs/atlas-service/
COPY libs/atlas-socket/go.mod      libs/atlas-socket/go.sum      libs/atlas-socket/
COPY libs/atlas-tenant/go.mod      libs/atlas-tenant/go.sum      libs/atlas-tenant/
COPY libs/atlas-tracing/go.mod     libs/atlas-tracing/go.sum     libs/atlas-tracing/

# Layer: this service's tree (per-target; brings in its go.mod and source).
COPY services/${SERVICE}/atlas.com/ services/${SERVICE}/atlas.com/

# Layer: all 17 atlas libs' source trees (shared across every target; invalidates
# when any lib source changes — same invalidation profile as today).
COPY libs/atlas-constants   libs/atlas-constants
COPY libs/atlas-database    libs/atlas-database
COPY libs/atlas-kafka       libs/atlas-kafka
COPY libs/atlas-lock        libs/atlas-lock
COPY libs/atlas-model       libs/atlas-model
COPY libs/atlas-object-id   libs/atlas-object-id
COPY libs/atlas-opcodes     libs/atlas-opcodes
COPY libs/atlas-packet      libs/atlas-packet
COPY libs/atlas-redis       libs/atlas-redis
COPY libs/atlas-rest        libs/atlas-rest
COPY libs/atlas-retry       libs/atlas-retry
COPY libs/atlas-saga        libs/atlas-saga
COPY libs/atlas-script-core libs/atlas-script-core
COPY libs/atlas-service     libs/atlas-service
COPY libs/atlas-socket      libs/atlas-socket
COPY libs/atlas-tenant      libs/atlas-tenant
COPY libs/atlas-tracing     libs/atlas-tracing

# Discover the inner module dir (services/${SERVICE}/atlas.com/<inner>) and build.
# Atlas convention: exactly one inner directory per service.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && test -n "$MOD_DIR" || (echo "ERROR: no module dir under services/${SERVICE}/atlas.com/" >&2 && exit 1) \
    && test -f "${MOD_DIR}go.mod" || (echo "ERROR: ${MOD_DIR}go.mod missing" >&2 && exit 1) \
    && go build -C "$MOD_DIR" -o /server

# Stash this service's config.yaml in a known location for the runtime stage to COPY.
RUN MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && cp "${MOD_DIR}config.yaml" /app/config.yaml

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /
COPY --from=build-env /app/config.yaml /

CMD ["/server"]
```

- [ ] **Step 2: Confirm the file landed and BuildKit syntax pragma is line 1**

Run: `head -1 Dockerfile`
Expected: `# syntax=docker/dockerfile:1.24`

(The pragma must be line 1 — comments above it disable it.)

- [ ] **Step 3: Lint via hadolint if available (optional, do not block on it)**

Run: `command -v hadolint && hadolint Dockerfile || echo "hadolint not installed; skipping"`
Expected: either clean hadolint output, or the skip line.

- [ ] **Step 4: Commit**

```bash
git add Dockerfile
git commit -m "build(docker): add shared parameterized Dockerfile for all Go services

Single file at repo root, ARG SERVICE selects which services/atlas-<svc>
to build. COPYs the union of all 17 atlas libs + the service tree; uses
the repo-root go.work for resolution (no inline synthesis). Inner module
dir is discovered via glob. BuildKit cache mounts on /go/pkg/mod and
/root/.cache/go-build.

Per-service Dockerfiles and bake config land in follow-up commits."
```

### Task 2: Empirical test — does `go.work` alone resolve atlas-* modules without the `-replace` block?

This is the design §2.3 / §3.2 gate. Outcome dictates whether Task 3 reinstates a parameterized `-replace` step.

**Files:** none (test-only — no edits unless Task 3 is needed)

- [ ] **Step 1: Build atlas-account from the shared Dockerfile (no -replace block present)**

Run:
```bash
DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg SERVICE=atlas-account -t atlas-account:test074 .
```
Expected: success — image built.

- [ ] **Step 2: Interpret result**

If Step 1 succeeds → workspace mode alone resolves the libs. Skip Task 3 entirely. Record the outcome in `docs/tasks/task-074-docker-build-shared-dockerfile/design.md` §3.2's "Recorded outcome" line via a one-paragraph append. Then move to Task 4.

If Step 1 fails with `cannot find module providing package github.com/Chronicle20/atlas/libs/...` (or any go-build resolution error pointing at the atlas libs) → workspace mode is insufficient. Execute Task 3.

If Step 1 fails for any other reason (e.g., missing config.yaml, glob mismatch, BuildKit cache mount syntax error) → that's a Task 1 bug. Fix it and re-run Step 1 before deciding Task 3.

- [ ] **Step 3: Append the recorded outcome to design.md**

Edit `docs/tasks/task-074-docker-build-shared-dockerfile/design.md` §3.2's "Recorded outcome" entry. Replace the placeholder bullet with the actual one-paragraph result. Example:

```markdown
  - **Recorded outcome:** [2026-MM-DD] `docker build --build-arg SERVICE=atlas-account .` succeeded with no `go mod edit -replace` block. Confirmed `go.work use(...)` resolves all 17 atlas libs in build-env. Block stays dropped.
```

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-074-docker-build-shared-dockerfile/design.md
git commit -m "docs(task-074): record empirical outcome of go.work-only resolution"
```

### Task 3: (Conditional — only if Task 2 Step 1 failed) Add parameterized `-replace` block

**Skip this task entirely if Task 2 Step 1 succeeded.**

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1: Inject parameterized `-replace` block in `build-env`**

Edit `Dockerfile`. Find the section that ends with the source-tree `COPY libs/atlas-tracing libs/atlas-tracing` line. Insert this `RUN` between that line and the existing build step's `RUN --mount=type=cache,...` block:

```dockerfile
# Force local resolution of Chronicle20/atlas-* modules against the in-repo
# libs (mirrors the historical per-service block). The list is the union of
# all 17 libs; -replace for a module the service does not require is harmless.
RUN MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && cd "$MOD_DIR" \
    && go mod edit \
        -replace=github.com/Chronicle20/atlas/libs/atlas-constants=/app/libs/atlas-constants \
        -replace=github.com/Chronicle20/atlas/libs/atlas-database=/app/libs/atlas-database \
        -replace=github.com/Chronicle20/atlas/libs/atlas-kafka=/app/libs/atlas-kafka \
        -replace=github.com/Chronicle20/atlas/libs/atlas-lock=/app/libs/atlas-lock \
        -replace=github.com/Chronicle20/atlas/libs/atlas-model=/app/libs/atlas-model \
        -replace=github.com/Chronicle20/atlas/libs/atlas-object-id=/app/libs/atlas-object-id \
        -replace=github.com/Chronicle20/atlas/libs/atlas-opcodes=/app/libs/atlas-opcodes \
        -replace=github.com/Chronicle20/atlas/libs/atlas-packet=/app/libs/atlas-packet \
        -replace=github.com/Chronicle20/atlas/libs/atlas-redis=/app/libs/atlas-redis \
        -replace=github.com/Chronicle20/atlas/libs/atlas-rest=/app/libs/atlas-rest \
        -replace=github.com/Chronicle20/atlas/libs/atlas-retry=/app/libs/atlas-retry \
        -replace=github.com/Chronicle20/atlas/libs/atlas-saga=/app/libs/atlas-saga \
        -replace=github.com/Chronicle20/atlas/libs/atlas-script-core=/app/libs/atlas-script-core \
        -replace=github.com/Chronicle20/atlas/libs/atlas-service=/app/libs/atlas-service \
        -replace=github.com/Chronicle20/atlas/libs/atlas-socket=/app/libs/atlas-socket \
        -replace=github.com/Chronicle20/atlas/libs/atlas-tenant=/app/libs/atlas-tenant \
        -replace=github.com/Chronicle20/atlas/libs/atlas-tracing=/app/libs/atlas-tracing
```

**Important:** do NOT add `go mod tidy` — design §2.3 explicitly drops it (workspace makes it unnecessary; tidy mutates go.mod and produces diff noise).

- [ ] **Step 2: Rebuild atlas-account**

Run:
```bash
DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg SERVICE=atlas-account -t atlas-account:test074 .
```
Expected: success.

- [ ] **Step 3: Append the recorded outcome to design.md**

Edit `docs/tasks/task-074-docker-build-shared-dockerfile/design.md` §3.2's "Recorded outcome" line. Example:

```markdown
  - **Recorded outcome:** [2026-MM-DD] `go.work use(...)` alone failed with `<error>`; parameterized `go mod edit -replace=...` block reinstated covering all 17 libs. `go mod tidy` remains dropped.
```

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docs/tasks/task-074-docker-build-shared-dockerfile/design.md
git commit -m "build(docker): reinstate parameterized -replace for atlas-* libs

go.work alone did not resolve in build-env (recorded in design.md §3.2).
Single parameterized RUN injects -replace for all 17 libs; -replace for
a module a service does not require is a harmless no-op."
```

### Task 4: Smoke-test the built `atlas-account` image

**Files:** none

- [ ] **Step 1: Inspect runtime metadata**

Run:
```bash
docker inspect atlas-account:test074 --format '{{json .Config}}' | jq '{Cmd, ExposedPorts, WorkingDir}'
```
Expected:
```json
{
  "Cmd": ["/server"],
  "ExposedPorts": {"8080/tcp": {}},
  "WorkingDir": "/"
}
```

- [ ] **Step 2: Confirm `/server` binary and `/config.yaml` exist**

Run:
```bash
docker run --rm --entrypoint sh atlas-account:test074 -c 'ls -la /server /config.yaml'
```
Expected: both files listed, `/server` executable, `/config.yaml` non-empty.

- [ ] **Step 3: Compare against the legacy image**

Build the old image first so we have a baseline:
```bash
DOCKER_BUILDKIT=1 docker build -f services/atlas-account/Dockerfile -t atlas-account:legacy074 .
docker inspect atlas-account:legacy074 --format '{{json .Config}}' | jq '{Cmd, ExposedPorts, WorkingDir}'
```
Expected: same `Cmd`, `ExposedPorts`, `WorkingDir` as Step 1.

If they differ, fix the shared Dockerfile until they match. Do NOT proceed.

- [ ] **Step 4: Confirm the binary starts (best-effort exec; service may exit fast without env)**

Run:
```bash
docker run --rm atlas-account:test074 /server --help 2>&1 | head -5 || true
```
Expected: either a help banner, a usage line, or a "missing config" error — but NOT a "no such file" / segfault on `/server`. Any of these confirms the binary exists, is linked correctly, and runs.

- [ ] **Step 5: Repeat Steps 1-4 for atlas-channel (socket service representative)**

Run:
```bash
DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg SERVICE=atlas-channel -t atlas-channel:test074 .
DOCKER_BUILDKIT=1 docker build -f services/atlas-channel/Dockerfile -t atlas-channel:legacy074 .
docker inspect atlas-channel:test074 --format '{{json .Config}}' | jq '{Cmd, ExposedPorts, WorkingDir}'
docker inspect atlas-channel:legacy074 --format '{{json .Config}}' | jq '{Cmd, ExposedPorts, WorkingDir}'
```
Expected: both inspect outputs identical (`Cmd: ["/server"]`, `ExposedPorts: {"8080/tcp": {}}`, `WorkingDir: "/"`).

- [ ] **Step 6: Clean up the smoke-test tags**

Run:
```bash
docker rmi atlas-account:test074 atlas-account:legacy074 atlas-channel:test074 atlas-channel:legacy074
```
Expected: removed (or "no such image" if already cleaned — fine).

- [ ] **Step 7: Commit (no-op if Phase 1 introduced no new files in this task)**

This task is verification-only; nothing to commit unless Step 3 forced a Dockerfile fix. If a fix was committed in this task, the commit message should reference "fix shared Dockerfile to match legacy atlas-<svc>:<aspect>".

---

## Phase 2 — Add `docker-bake.hcl` and prove every service builds end-to-end

### Task 5: Create `docker-bake.hcl` at repo root

**Files:**
- Create: `docker-bake.hcl`

- [ ] **Step 1: Write the bake file**

Create `docker-bake.hcl` (repo root) with these exact contents:

```hcl
# Atlas docker bake file. Drives all Go-service image builds.
#
# Single source of truth for the service list: .github/config/services.json.
#
#   docker buildx bake                                  # all-go-services (default group)
#   docker buildx bake all-go-services                  # explicit: every Go service
#   docker buildx bake atlas-account                    # one
#   docker buildx bake atlas-account atlas-ban          # subset
#
# CI overrides tags per-target via --set "<target>.tags=<image>:<tag>".

variable "SERVICES_CONFIG" {
  default = "./.github/config/services.json"
}

variable "ATLAS_IMAGE_TAG" {
  # Used by local builds (matches the deploy/compose ${ATLAS_IMAGE_TAG:-local} pattern).
  default = "local"
}

variable "GO_VERSION" {
  default = "1.25.5"
}

variable "ALPINE_VERSION" {
  default = "3.21"
}

locals {
  config      = jsondecode(file("${SERVICES_CONFIG}"))
  go_services = [for s in local.config.services : s if s.type == "go-service"]
}

# One target per Go service, expanded from the JSON at parse time.
target "go-service" {
  matrix = {
    svc = [for s in local.go_services : s.name]
  }
  name       = svc
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    SERVICE        = svc
    GO_VERSION     = "${GO_VERSION}"
    ALPINE_VERSION = "${ALPINE_VERSION}"
  }
  # Default local tag. CI overrides per-target via --set.
  tags = ["${svc}:${ATLAS_IMAGE_TAG}"]
}

group "all-go-services" {
  targets = [for s in local.go_services : s.name]
}

# Default group: same as all-go-services.
group "default" {
  targets = ["all-go-services"]
}
```

- [ ] **Step 2: Verify bake parses the file and emits the expected target list**

Run:
```bash
docker buildx bake --print 2>&1 | jq -r '.target | keys | sort | .[]' | head -60
```
Expected: 54 target names (`atlas-account`, `atlas-asset-expiration`, …, `atlas-world`, `atlas-wz-extractor`).

If `--print` outputs JSON wrapped in extra log lines, drop the `jq` and inspect manually. The key thing: 54 target entries, no parse errors.

- [ ] **Step 3: Verify a single-target print**

Run:
```bash
docker buildx bake --print atlas-account 2>&1 | jq '.target."atlas-account"'
```
Expected: a JSON object with `dockerfile: "Dockerfile"`, `context: "."`, `args.SERVICE: "atlas-account"`, `args.GO_VERSION: "1.25.5"`, `args.ALPINE_VERSION: "3.21"`, `tags: ["atlas-account:local"]`.

- [ ] **Step 4: Build one target via bake to confirm end-to-end wiring**

Run:
```bash
docker buildx bake atlas-account
```
Expected: build succeeds; image `atlas-account:local` is produced. Confirm:
```bash
docker image inspect atlas-account:local --format '{{.Config.Cmd}}'
```
Expected: `[/server]`.

Clean up: `docker rmi atlas-account:local`.

- [ ] **Step 5: Commit**

```bash
git add docker-bake.hcl
git commit -m "build(docker): add docker-bake.hcl driven by services.json

HCL matrix expands one target per .services[] | select(.type=='go-service')
in .github/config/services.json — single source of truth, no drift.

Local: 'docker buildx bake atlas-<svc>' or 'docker buildx bake' for all.
CI overrides tags per-target via --set."
```

### Task 6: Full-fleet local bake — prove all 54 services build against the shared Dockerfile

**Files:** none

- [ ] **Step 1: Bake every Go service**

Run:
```bash
docker buildx bake all-go-services 2>&1 | tee /tmp/task074-bake-all.log
```
Expected: all 54 targets succeed.

If any target fails:
1. Note the service name and the error.
2. If it's a glob/inner-module issue (no go.mod found), inspect `services/<svc>/atlas.com/` to confirm convention.
3. If it's a lib-resolution issue affecting only some services, that contradicts Task 2's outcome — revisit the empirical test for the failing service and apply Task 3's `-replace` block.
4. If it's a missing `config.yaml`, inspect `services/<svc>/atlas.com/<inner>/config.yaml` and decide whether the service legitimately has no config (in which case the shared Dockerfile needs a guard) or whether the inner-module glob picked the wrong directory.
5. Fix the shared Dockerfile (or, in extreme cases, the offending service tree). Commit the fix with a message like `fix(docker): handle <case> in shared Dockerfile`. Re-run `docker buildx bake all-go-services` until it's green.

- [ ] **Step 2: Spot-check three additional services with quirky inner-module names**

Run:
```bash
docker image inspect atlas-drop-information:local atlas-monster-death:local atlas-families:local --format '{{.RepoTags}} {{.Config.Cmd}}'
```
Expected: three lines, each ending in `[/server]`. These services have inner modules `dis`, `monster`, `family` respectively — the glob-based discovery must handle them. If any failed in Step 1, this is where the glob assumption breaks.

- [ ] **Step 3: Clean up local images (optional)**

Run:
```bash
docker buildx bake --print all-go-services 2>&1 | jq -r '.target | keys[]' | sed 's/$/:local/' | xargs -r docker rmi || true
```
Expected: image cleanup output; "no such image" warnings are fine.

- [ ] **Step 4: No commit required for this task** — verification only. If Step 1 forced a Dockerfile fix, that fix was already committed inline.

---

## Phase 3 — Rewrite the composite docker-build action to wrap bake

### Task 7: Replace `.github/actions/docker-build/action.yml`

**Files:**
- Modify: `.github/actions/docker-build/action.yml` (full replacement)

- [ ] **Step 1: Replace the action file**

Overwrite `.github/actions/docker-build/action.yml` with:

```yaml
name: 'Docker Build (bake)'
description: 'Build (and optionally push) one or more Atlas Go-service images via docker buildx bake against the repo-root docker-bake.hcl.'

inputs:
  targets:
    description: 'JSON array of bake target names (e.g. ["atlas-account","atlas-ban"]). Each name must exist as a target in docker-bake.hcl.'
    required: true
  image-name-map:
    description: 'JSON object mapping target name → fully-qualified image name without tag (e.g. {"atlas-account":"ghcr.io/chronicle20/atlas-account/atlas-account"}).'
    required: true
  tags:
    description: 'Comma-separated tag list applied to every target (e.g. "pr-123-abc" or "latest-amd64,main-abc-amd64").'
    required: true
  platform:
    description: 'Single target platform (linux/amd64 | linux/arm64). For multi-arch, invoke this action twice with different platforms and join with docker manifest.'
    required: false
    default: 'linux/amd64'
  push:
    description: 'Push images to the registry.'
    required: false
    default: 'false'
  registry:
    description: 'Container registry hostname.'
    required: false
    default: 'ghcr.io'
  registry-username:
    description: 'Registry username (required if push=true).'
    required: false
    default: ''
  registry-password:
    description: 'Registry password/token (required if push=true).'
    required: false
    default: ''
  cache-scope:
    description: 'GHA cache scope key (e.g. "atlas-bake-amd64"). Single shared scope across all targets in this invocation.'
    required: true
  provenance:
    description: 'Generate provenance attestation. "false" produces single-platform images (not manifest lists) — required for the create-manifest job.'
    required: false
    default: 'false'
  sbom:
    description: 'Generate SBOM attestation. "false" produces single-platform images (not manifest lists).'
    required: false
    default: 'false'

runs:
  using: 'composite'
  steps:
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Log in to container registry
      if: inputs.push == 'true' && inputs.registry-username != ''
      uses: docker/login-action@v3
      with:
        registry: ${{ inputs.registry }}
        username: ${{ inputs.registry-username }}
        password: ${{ inputs.registry-password }}

    - name: Build per-target --set flags
      id: setflags
      shell: bash
      env:
        TARGETS_JSON: ${{ inputs.targets }}
        IMAGE_MAP_JSON: ${{ inputs.image-name-map }}
        TAGS_CSV: ${{ inputs.tags }}
        PLATFORM: ${{ inputs.platform }}
        PUSH: ${{ inputs.push }}
        CACHE_SCOPE: ${{ inputs.cache-scope }}
        PROVENANCE: ${{ inputs.provenance }}
        SBOM: ${{ inputs.sbom }}
      run: |
        set -euo pipefail

        # Validate JSON inputs and emit the array of target names.
        mapfile -t TARGETS < <(echo "$TARGETS_JSON" | jq -r '.[]')
        if [ "${#TARGETS[@]}" -eq 0 ]; then
          echo "ERROR: 'targets' input must be a non-empty JSON array" >&2
          exit 1
        fi

        # Per-target image name lookup.
        FLAGS=()
        for target in "${TARGETS[@]}"; do
          image=$(echo "$IMAGE_MAP_JSON" | jq -r --arg t "$target" '.[$t] // empty')
          if [ -z "$image" ]; then
            echo "ERROR: 'image-name-map' has no entry for target '$target'" >&2
            exit 1
          fi
          # Build comma-joined "<image>:<tag1>,<image>:<tag2>" list from TAGS_CSV.
          TAG_LIST=""
          IFS=',' read -ra TAG_ARRAY <<< "$TAGS_CSV"
          for tag in "${TAG_ARRAY[@]}"; do
            tag="$(echo "$tag" | xargs)" # trim
            [ -z "$tag" ] && continue
            if [ -n "$TAG_LIST" ]; then
              TAG_LIST="${TAG_LIST},${image}:${tag}"
            else
              TAG_LIST="${image}:${tag}"
            fi
          done
          FLAGS+=("--set" "${target}.tags=${TAG_LIST}")
        done

        # Global --set flags applied to every target in this invocation.
        FLAGS+=("--set" "*.platform=${PLATFORM}")
        FLAGS+=("--set" "*.cache-from=type=gha,scope=${CACHE_SCOPE}")
        FLAGS+=("--set" "*.cache-to=type=gha,mode=max,scope=${CACHE_SCOPE}")
        FLAGS+=("--set" "*.output=type=image,push=${PUSH}")
        FLAGS+=("--set" "*.attest=type=provenance,disabled=$( [ "$PROVENANCE" = "true" ] && echo false || echo true )")
        FLAGS+=("--set" "*.attest=type=sbom,disabled=$( [ "$SBOM" = "true" ] && echo false || echo true )")

        # Persist for the next step (newline-delimited; the next step re-splits).
        {
          for f in "${FLAGS[@]}"; do
            printf '%s\n' "$f"
          done
        } > "${RUNNER_TEMP}/bake-flags.txt"

        # Persist the target list for the next step (one per line).
        printf '%s\n' "${TARGETS[@]}" > "${RUNNER_TEMP}/bake-targets.txt"

        echo "Built ${#FLAGS[@]} flag tokens for ${#TARGETS[@]} target(s); platform=${PLATFORM}, push=${PUSH}, cache-scope=${CACHE_SCOPE}"

    - name: docker buildx bake
      shell: bash
      run: |
        set -euo pipefail
        mapfile -t TARGETS < "${RUNNER_TEMP}/bake-targets.txt"
        mapfile -t FLAGS   < "${RUNNER_TEMP}/bake-flags.txt"
        echo "Running: docker buildx bake ${TARGETS[*]} ${FLAGS[*]}"
        docker buildx bake "${TARGETS[@]}" "${FLAGS[@]}"

    - name: Summary
      shell: bash
      env:
        TARGETS_JSON: ${{ inputs.targets }}
        TAGS_CSV: ${{ inputs.tags }}
        PLATFORM: ${{ inputs.platform }}
        PUSH: ${{ inputs.push }}
        CACHE_SCOPE: ${{ inputs.cache-scope }}
      run: |
        {
          echo "### Docker Build Summary (bake)"
          echo ""
          echo "- **Platform:** ${PLATFORM}"
          echo "- **Pushed:** ${PUSH}"
          echo "- **Cache scope:** ${CACHE_SCOPE}"
          echo "- **Tags applied:** ${TAGS_CSV}"
          echo ""
          echo "| Target |"
          echo "|--------|"
          echo "$TARGETS_JSON" | jq -r '.[] | "| " + . + " |"'
        } >> "$GITHUB_STEP_SUMMARY"
```

- [ ] **Step 2: Lint the action with `act` or `actionlint` if installed (best-effort)**

Run:
```bash
command -v actionlint && actionlint .github/actions/docker-build/action.yml || echo "actionlint not installed; skipping"
```
Expected: clean output or skip line. (CI will catch real problems when the workflow runs.)

- [ ] **Step 3: Commit**

```bash
git add .github/actions/docker-build/action.yml
git commit -m "ci(docker-build): rewrite composite action to wrap buildx bake

Inputs: targets (JSON array), image-name-map (JSON object), tags
(comma-separated, applied to every target), platform, push,
cache-scope, registry creds, provenance, sbom.

Generates --set <target>.tags=<image>:<tag> per target and global
--set *.platform / *.cache-from / *.cache-to / *.output / *.attest
flags, then runs 'docker buildx bake <targets> <flags>'. Per-target
tag injection preserves today's tag-string semantics
(pr-<N>-<sha>, latest-<arch>, main-<sha>-<arch>) verbatim."
```

---

## Phase 4 — Collapse the CI matrix jobs

### Task 8: Rewrite `pr-validation.yml` `build-docker` job to a single bake invocation

**Files:**
- Modify: `.github/workflows/pr-validation.yml` (only the `build-docker` job; keep `update-pr-overlay`, `pr-validation-complete`, and everything else untouched)

- [ ] **Step 1: Replace the `build-docker` job**

Open `.github/workflows/pr-validation.yml`. Locate the `build-docker:` job (currently named `Build Docker - ${{ matrix.service.name }}`, uses `strategy.matrix.service`). Replace the entire job block with:

```yaml
  # ============================================
  # Build Docker Images (bake — single job)
  #
  # Always runs Dockerfile validation; pushes to ghcr only when the PR
  # carries the `deploy-env` label. The full set of services to build
  # is sourced from detect-changes.outputs.docker-services-matrix and
  # collapsed into one `docker buildx bake` invocation; the per-PR tag
  # (pr-<N>-<sha> when push=true, pr-<N>/pr-dispatch otherwise) is
  # computed once and applied to every target.
  # ============================================
  build-docker:
    name: Build Docker (bake)
    needs: [detect-changes, test-go-services, test-go-libraries, test-ui]
    if: |
      always() &&
      needs.detect-changes.outputs.docker-services-matrix != '[]' &&
      (needs.test-go-services.result == 'success' || needs.test-go-services.result == 'skipped') &&
      (needs.test-go-libraries.result == 'success' || needs.test-go-libraries.result == 'skipped') &&
      (needs.test-ui.result == 'success' || needs.test-ui.result == 'skipped')
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Compute short SHA
        id: sha
        run: |
          SHA="${{ github.event.pull_request.head.sha || github.sha }}"
          echo "short=$(git rev-parse --short=7 "$SHA")" >> $GITHUB_OUTPUT

      - name: Compute push flag and tag
        id: pushtag
        env:
          PR_NUMBER: ${{ github.event.pull_request.number }}
          SHORT_SHA: ${{ steps.sha.outputs.short }}
          IS_PR: ${{ github.event_name == 'pull_request' }}
          HAS_LABEL: ${{ github.event_name == 'pull_request' && contains(github.event.pull_request.labels.*.name, 'deploy-env') }}
        run: |
          set -euo pipefail
          if [ "$HAS_LABEL" = "true" ]; then
            echo "push=true" >> "$GITHUB_OUTPUT"
            echo "tag=pr-${PR_NUMBER}-${SHORT_SHA}" >> "$GITHUB_OUTPUT"
          else
            echo "push=false" >> "$GITHUB_OUTPUT"
            if [ "$IS_PR" = "true" ]; then
              echo "tag=pr-${PR_NUMBER}" >> "$GITHUB_OUTPUT"
            else
              echo "tag=pr-dispatch" >> "$GITHUB_OUTPUT"
            fi
          fi

      - name: Derive bake inputs from docker-services-matrix
        id: bake-inputs
        env:
          MATRIX_JSON: ${{ needs.detect-changes.outputs.docker-services-matrix }}
        run: |
          set -euo pipefail
          # targets = [.name, .name, ...]
          TARGETS=$(echo "$MATRIX_JSON" | jq -c '[.[].name]')
          # image-name-map = {name: docker_image}
          IMAGE_MAP=$(echo "$MATRIX_JSON" | jq -c 'map({(.name): .docker_image}) | add')
          echo "targets=$TARGETS"       >> "$GITHUB_OUTPUT"
          echo "image-name-map=$IMAGE_MAP" >> "$GITHUB_OUTPUT"

      - name: Build Docker images (bake)
        uses: ./.github/actions/docker-build
        with:
          targets: ${{ steps.bake-inputs.outputs.targets }}
          image-name-map: ${{ steps.bake-inputs.outputs.image-name-map }}
          tags: ${{ steps.pushtag.outputs.tag }}
          push: ${{ steps.pushtag.outputs.push }}
          platform: linux/amd64
          registry-username: ${{ github.actor }}
          registry-password: ${{ secrets.GHCR_TOKEN }}
          cache-scope: atlas-bake-amd64
```

- [ ] **Step 2: Confirm `update-pr-overlay` and `pr-validation-complete` jobs are untouched**

Run:
```bash
grep -nE 'name: (Resolve PR overlay|PR Validation Complete)' .github/workflows/pr-validation.yml
```
Expected: both lines present, with the surrounding `needs:` and step bodies unchanged.

Specifically, `update-pr-overlay` MUST still have `needs: [detect-changes, build-docker]` and consume `docker-services-matrix` from `detect-changes`. If anything in those two jobs changed, revert the changes — they are explicitly out of scope.

- [ ] **Step 3: Workflow syntax validation (best-effort)**

Run:
```bash
command -v actionlint && actionlint .github/workflows/pr-validation.yml || echo "actionlint not installed; skipping"
```
Expected: clean or skip line.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci(pr-validation): collapse build-docker matrix to single bake job

detect-changes still drives the service set (docker-services-matrix is
consumed unchanged). The matrix's per-service rows are folded into one
'targets' JSON array + 'image-name-map' JSON object passed to the bake
action. Per-PR tag (pr-<N>-<sha> when deploy-env labeled, pr-<N>/
pr-dispatch otherwise) is computed once and applied to every target via
the action's tag injection.

update-pr-overlay's needs/inputs unchanged."
```

### Task 9: Collapse `main-publish.yml` AMD64 + ARM64 matrix jobs

**Files:**
- Modify: `.github/workflows/main-publish.yml` (only `build-amd64` and `build-arm64`; keep `create-manifest` and `update-image-tags` matrix-shaped)

- [ ] **Step 1: Replace `build-amd64`**

Locate the `build-amd64:` job. Replace with:

```yaml
  # ============================================
  # Build and Push AMD64 Images (bake — single job)
  # ============================================
  build-amd64:
    name: Build AMD64 (bake)
    needs: detect-changes
    if: needs.detect-changes.outputs.has-changes == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Derive bake inputs
        id: bake-inputs
        env:
          MATRIX_JSON: ${{ needs.detect-changes.outputs.docker-services-matrix }}
        run: |
          set -euo pipefail
          TARGETS=$(echo "$MATRIX_JSON" | jq -c '[.[].name]')
          IMAGE_MAP=$(echo "$MATRIX_JSON" | jq -c 'map({(.name): .docker_image}) | add')
          echo "targets=$TARGETS"       >> "$GITHUB_OUTPUT"
          echo "image-name-map=$IMAGE_MAP" >> "$GITHUB_OUTPUT"

      - name: Build and push AMD64 images (bake)
        uses: ./.github/actions/docker-build
        with:
          targets: ${{ steps.bake-inputs.outputs.targets }}
          image-name-map: ${{ steps.bake-inputs.outputs.image-name-map }}
          tags: latest-amd64,main-${{ needs.detect-changes.outputs.short-sha }}-amd64
          push: 'true'
          platform: linux/amd64
          registry-username: ${{ github.actor }}
          registry-password: ${{ secrets.GHCR_TOKEN }}
          cache-scope: atlas-bake-amd64
```

- [ ] **Step 2: Replace `build-arm64`**

Locate the `build-arm64:` job. Replace with:

```yaml
  # ============================================
  # Build and Push ARM64 Images (bake — single job)
  # ============================================
  build-arm64:
    name: Build ARM64 (bake)
    needs: detect-changes
    if: needs.detect-changes.outputs.has-changes == 'true'
    runs-on: ubuntu-24.04-arm
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Derive bake inputs
        id: bake-inputs
        env:
          MATRIX_JSON: ${{ needs.detect-changes.outputs.docker-services-matrix }}
        run: |
          set -euo pipefail
          TARGETS=$(echo "$MATRIX_JSON" | jq -c '[.[].name]')
          IMAGE_MAP=$(echo "$MATRIX_JSON" | jq -c 'map({(.name): .docker_image}) | add')
          echo "targets=$TARGETS"       >> "$GITHUB_OUTPUT"
          echo "image-name-map=$IMAGE_MAP" >> "$GITHUB_OUTPUT"

      - name: Build and push ARM64 images (bake)
        uses: ./.github/actions/docker-build
        with:
          targets: ${{ steps.bake-inputs.outputs.targets }}
          image-name-map: ${{ steps.bake-inputs.outputs.image-name-map }}
          tags: latest-arm64,main-${{ needs.detect-changes.outputs.short-sha }}-arm64
          push: 'true'
          platform: linux/arm64
          registry-username: ${{ github.actor }}
          registry-password: ${{ secrets.GHCR_TOKEN }}
          cache-scope: atlas-bake-arm64
```

- [ ] **Step 3: Confirm `create-manifest` and `update-image-tags` are untouched**

Run:
```bash
grep -nE 'name: (Create Manifest|GitOps promotion|Update image tag)' .github/workflows/main-publish.yml
```
Expected: existing per-service-matrix jobs remain. They run after both `build-amd64` and `build-arm64`, iterate `docker-services-matrix` row by row, and call `docker manifest create/push` for each. No build work in these jobs — collapsing them buys nothing and risks subtle changes.

- [ ] **Step 4: Workflow syntax validation**

Run:
```bash
command -v actionlint && actionlint .github/workflows/main-publish.yml || echo "actionlint not installed; skipping"
```
Expected: clean or skip line.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/main-publish.yml
git commit -m "ci(main-publish): collapse build-amd64 and build-arm64 to bake jobs

Each per-platform job becomes one job that consumes
detect-changes.outputs.docker-services-matrix as a 'targets' array and
calls the bake-wrapping docker-build action with platform-appropriate
tags (latest-<arch>, main-<sha>-<arch>) and cache-scope
(atlas-bake-amd64 / atlas-bake-arm64).

create-manifest and update-image-tags remain per-service matrices —
they do no build work, just docker manifest create/push and yq edits."
```

---

## Phase 5 — Update compose files

### Task 10: Rewrite `deploy/compose/docker-compose.core.yml` build blocks

**Files:**
- Modify: `deploy/compose/docker-compose.core.yml`

- [ ] **Step 1: Identify every go-service build block**

Run:
```bash
grep -n 'dockerfile: services/atlas-' deploy/compose/docker-compose.core.yml | wc -l
```
Expected: 52.

- [ ] **Step 2: Rewrite each build block**

For every block matching the pattern:

```yaml
    build:
      context: ../..
      dockerfile: services/atlas-<svc>/Dockerfile
```

rewrite to:

```yaml
    build:
      context: ../..
      dockerfile: Dockerfile
      args:
        SERVICE: atlas-<svc>
```

You can do this via a single `sed` invocation that captures the service name:

```bash
sed -i -E \
  's|^(\s+)dockerfile: services/(atlas-[a-z-]+)/Dockerfile$|\1dockerfile: Dockerfile\n\1args:\n\1  SERVICE: \2|' \
  deploy/compose/docker-compose.core.yml
```

**Verify after sed:**

```bash
grep -c 'dockerfile: services/atlas-' deploy/compose/docker-compose.core.yml
```
Expected: `0` (the `dockerfile: services/atlas-...` lines should all be gone).

```bash
grep -c 'SERVICE: atlas-' deploy/compose/docker-compose.core.yml
```
Expected: `52`.

```bash
grep -c 'dockerfile: Dockerfile$' deploy/compose/docker-compose.core.yml
```
Expected: `52`.

- [ ] **Step 3: Confirm atlas-assets block is untouched**

Run:
```bash
grep -A2 'container_name: atlas-assets' deploy/compose/docker-compose.core.yml | head -10
```
Expected: `build.context: ../../services/atlas-assets` and `build.dockerfile: Dockerfile` (the in-service Dockerfile path, NOT the repo-root one). `atlas-assets` is a static-service; its block stays.

- [ ] **Step 4: Visually confirm one rewritten block**

Run:
```bash
grep -B1 -A5 'SERVICE: atlas-account' deploy/compose/docker-compose.core.yml
```
Expected:
```
    build:
      context: ../..
      dockerfile: Dockerfile
      args:
        SERVICE: atlas-account
    image: atlas-account:${ATLAS_IMAGE_TAG:-local}
```

- [ ] **Step 5: Smoke `docker compose build` against the rewritten file**

Run:
```bash
cd deploy/compose
docker compose -f docker-compose.core.yml build atlas-account
cd -
```
Expected: image build succeeds. (Compose now invokes the shared Dockerfile via `build.args.SERVICE: atlas-account`.)

If your local compose v2 is unwilling to bake-route this (compose builds use buildx by default in recent versions), this confirms end-to-end that the compose-side cutover works.

- [ ] **Step 6: Clean up the test image**

Run:
```bash
docker rmi atlas-account:local 2>/dev/null || true
```

- [ ] **Step 7: Commit**

```bash
git add deploy/compose/docker-compose.core.yml
git commit -m "build(compose): point core services at shared Dockerfile with build.args.SERVICE

Every go-service block in docker-compose.core.yml swaps
'dockerfile: services/atlas-<svc>/Dockerfile' for
'dockerfile: Dockerfile' + 'args: {SERVICE: atlas-<svc>}'.

atlas-assets (static-service, service-local context) untouched."
```

### Task 11: Rewrite `deploy/compose/docker-compose.socket.yml` build blocks

**Files:**
- Modify: `deploy/compose/docker-compose.socket.yml`

- [ ] **Step 1: Confirm 2 go-service build blocks**

Run:
```bash
grep -n 'dockerfile: services/atlas-' deploy/compose/docker-compose.socket.yml
```
Expected: 2 matches — `atlas-login` and `atlas-channel`.

- [ ] **Step 2: Apply the same sed transform**

```bash
sed -i -E \
  's|^(\s+)dockerfile: services/(atlas-[a-z-]+)/Dockerfile$|\1dockerfile: Dockerfile\n\1args:\n\1  SERVICE: \2|' \
  deploy/compose/docker-compose.socket.yml
```

**Verify:**

```bash
grep -c 'dockerfile: services/atlas-' deploy/compose/docker-compose.socket.yml   # → 0
grep -c 'SERVICE: atlas-' deploy/compose/docker-compose.socket.yml               # → 2
grep -c 'dockerfile: Dockerfile$' deploy/compose/docker-compose.socket.yml       # → 2
```

- [ ] **Step 3: Smoke `docker compose build atlas-login`**

Run:
```bash
cd deploy/compose
docker compose -f docker-compose.socket.yml build atlas-login
cd -
```
Expected: success. Clean up: `docker rmi atlas-login:local 2>/dev/null || true`.

- [ ] **Step 4: Commit**

```bash
git add deploy/compose/docker-compose.socket.yml
git commit -m "build(compose): point socket services at shared Dockerfile with build.args.SERVICE

atlas-login and atlas-channel blocks rewritten to use the shared
Dockerfile with build.args.SERVICE."
```

### Task 12: Confirm `deploy/compose/docker-compose.yml` requires no changes

**Files:** none

- [ ] **Step 1: Inspect**

Run:
```bash
grep -n 'dockerfile:' deploy/compose/docker-compose.yml
```
Expected: empty (the file only defines `nginx` from the `nginx:alpine` image, no `build:` block).

If anything matches, re-evaluate — this contradicts the design assumption.

- [ ] **Step 2: No commit** — verification only.

---

## Phase 6 — Delete the legacy per-service Dockerfiles (rip-and-replace cutover)

### Task 13: Delete every in-scope `services/atlas-*/Dockerfile`

**Files:**
- Delete: `services/atlas-<svc>/Dockerfile` for every `<svc>` in §2 of `context.md`.

- [ ] **Step 1: Build the explicit deletion list**

Run:
```bash
jq -r '.services[] | select(.type=="go-service") | .path' .github/config/services.json \
  | sed 's|$|/Dockerfile|' > /tmp/task074-dockerfiles-to-delete.txt
wc -l /tmp/task074-dockerfiles-to-delete.txt
```
Expected: 54.

- [ ] **Step 2: Verify every listed file exists before deletion (sanity)**

Run:
```bash
xargs -a /tmp/task074-dockerfiles-to-delete.txt -I {} test -f {} && echo "all present" || echo "missing files — STOP"
```
Expected: `all present`. If `missing files — STOP`, investigate which path doesn't exist before any deletion.

- [ ] **Step 3: Delete the listed files**

Run:
```bash
xargs -a /tmp/task074-dockerfiles-to-delete.txt rm
```

- [ ] **Step 4: Confirm the three out-of-scope Dockerfiles still exist**

Run:
```bash
ls services/atlas-ui/Dockerfile services/atlas-assets/Dockerfile services/atlas-pr-bootstrap/Dockerfile
```
Expected: three lines printed, no errors.

- [ ] **Step 5: Confirm no go-service Dockerfile remains under services/**

Run:
```bash
find services -maxdepth 2 -name Dockerfile -printf '%p\n' | sort
```
Expected: exactly:
```
services/atlas-assets/Dockerfile
services/atlas-pr-bootstrap/Dockerfile
services/atlas-ui/Dockerfile
```
(One per untouched service.)

- [ ] **Step 6: Commit**

```bash
git add -A services/
git commit -m "build(docker): delete 54 per-service Dockerfiles; shared Dockerfile takes over

Every services/atlas-<svc>/Dockerfile for a go-service in services.json
removed. The repo-root shared Dockerfile (with docker-bake.hcl, the
collapsed CI matrix, and the compose-file updates) is now the only
build path.

Untouched: atlas-ui (Next.js/nginx), atlas-assets (static), and
atlas-pr-bootstrap (alpine+rpk) keep their service-local Dockerfiles."
```

### Task 14: Delete every `Dockerfile.dev` and `Dockerfile.debug`

**Files:**
- Delete: every `services/atlas-*/Dockerfile.dev` and `services/atlas-*/Dockerfile.debug`.

- [ ] **Step 1: Enumerate**

Run:
```bash
find services -maxdepth 2 \( -name 'Dockerfile.dev' -o -name 'Dockerfile.debug' \) -printf '%p\n' | sort > /tmp/task074-dev-debug-to-delete.txt
wc -l /tmp/task074-dev-debug-to-delete.txt
```
Expected: some non-zero number (design says ~86 across both).

- [ ] **Step 2: Confirm none of these files are referenced anywhere in the repo (design §2.6 assertion)**

Run:
```bash
grep -rnE 'Dockerfile\.(dev|debug)' deploy/ tools/ .github/ Makefile docs/ 2>/dev/null || echo "no references"
```
Expected: `no references`. If any reference exists, STOP and investigate — design §2.6 assumed none, and a hit invalidates the "delete outright" decision.

- [ ] **Step 3: Delete**

Run:
```bash
xargs -a /tmp/task074-dev-debug-to-delete.txt rm
```

- [ ] **Step 4: Verify clean removal**

Run:
```bash
find services -maxdepth 2 \( -name 'Dockerfile.dev' -o -name 'Dockerfile.debug' \) -printf '%p\n'
```
Expected: empty output.

- [ ] **Step 5: Commit**

```bash
git add -A services/
git commit -m "build(docker): delete Dockerfile.dev and Dockerfile.debug variants

No consumers anywhere in deploy/, tools/, .github/, Makefile, or docs/.
These are stale artifacts from a pre-monorepo era — their per-service
contexts haven't existed since the deploy-reorg PR. If a future need
for delve-based debugging arises, recreate the debug stage at that
point — speculative now."
```

---

## Phase 7 — Update tools

### Task 15: Delete `tools/inject-dockerfile-replace.sh`

**Files:**
- Delete: `tools/inject-dockerfile-replace.sh`

- [ ] **Step 1: Confirm no other tool sources it**

Run:
```bash
grep -rn 'inject-dockerfile-replace' tools/ .github/ Makefile docs/ 2>/dev/null || echo "no references"
```
Expected: only the file itself appears (or "no references" if grep doesn't reflect its self-reference). Importantly, no `source` / `bash inject-dockerfile-replace.sh` / `tools/inject-dockerfile-replace.sh` callsite from a different script.

- [ ] **Step 2: Delete**

Run:
```bash
rm tools/inject-dockerfile-replace.sh
```

- [ ] **Step 3: Commit**

```bash
git add -A tools/
git commit -m "build(tools): remove inject-dockerfile-replace.sh

The script's purpose was injecting the per-Dockerfile 'go mod edit
-replace' block. With one shared Dockerfile, that surface is gone."
```

### Task 16: Rewrite `tools/build-services.sh` as a bake wrapper

**Files:**
- Modify: `tools/build-services.sh` (full replacement)

- [ ] **Step 1: Replace contents**

Overwrite `tools/build-services.sh` with:

```bash
#!/usr/bin/env bash
# Builds Atlas Go-service images via docker buildx bake against the
# repo-root docker-bake.hcl. Forwards any arguments through to bake so
# callers can target a subset:
#
#   tools/build-services.sh                        # all-go-services
#   tools/build-services.sh atlas-account          # one
#   tools/build-services.sh atlas-account atlas-ban  # subset
#
# Run from the repo root.
set -euo pipefail
exec docker buildx bake "$@"
```

- [ ] **Step 2: Ensure executable bit**

Run:
```bash
chmod +x tools/build-services.sh
```

- [ ] **Step 3: Verify behavior**

Run from the repo root:
```bash
./tools/build-services.sh --print atlas-account 2>&1 | head -20
```
Expected: bake `--print` output for the `atlas-account` target.

- [ ] **Step 4: Commit**

```bash
git add tools/build-services.sh
git commit -m "build(tools): rewrite build-services.sh as a thin bake wrapper

Body is one exec line: 'docker buildx bake \"\$@\"'. Bake handles
parallelism, target selection, and cache reuse; no need for the
per-Dockerfile loop the old script implemented."
```

### Task 17: Update `tools/import-lib.sh` docstring

**Files:**
- Modify: `tools/import-lib.sh` (header comment / docstring only)

- [ ] **Step 1: Read the current top-of-file**

Run:
```bash
head -10 tools/import-lib.sh
```

- [ ] **Step 2: Insert a docstring block after the shebang**

Use Edit to insert this block immediately after the `set -euo pipefail` line:

```bash

# Imports a new Chronicle20/atlas-<name> lib repo into libs/<name>. After
# this script completes, manually:
#   1. Append "    ./libs/<name>" to /go.work's `use (...)` block.
#   2. Append two COPY lines to the repo-root Dockerfile:
#        - one in the mod-only block:
#            COPY libs/<name>/go.mod libs/<name>/go.sum libs/<name>/
#        - one in the source block:
#            COPY libs/<name> libs/<name>
#   3. Run `docker buildx bake atlas-account` to verify resolution.
#
# Adding the lib to /go.work and the shared Dockerfile is the single
# place lib dependencies are declared today (post-task-074 consolidation
# — see CLAUDE.md "Build & Verification").
```

- [ ] **Step 3: Verify the script still runs (-h / usage check)**

Run:
```bash
bash -n tools/import-lib.sh && echo "syntax OK"
```
Expected: `syntax OK`.

- [ ] **Step 4: Commit**

```bash
git add tools/import-lib.sh
git commit -m "docs(tools): update import-lib.sh docstring for shared-Dockerfile workflow

Document the new post-import manual steps (append to /go.work, append
two COPY lines to the repo-root Dockerfile)."
```

### Task 18: Update `tools/import-service.sh` docstring

**Files:**
- Modify: `tools/import-service.sh` (header comment / docstring only)

- [ ] **Step 1: Insert a docstring block after `set -euo pipefail`**

Use Edit to insert:

```bash

# Imports a new Chronicle20/atlas-<name> service repo into
# services/atlas-<name>. After this script completes, manually:
#   1. Append a "{name,type:go-service,path,module_path,docker_image,
#      docker_context}" row to .github/config/services.json under
#      .services[].
#   2. Append "    ./services/atlas-<name>/atlas.com/<inner>" to
#      /go.work's `use (...)` block (where <inner> is the inner module
#      directory the imported repo uses).
#   3. Run `docker buildx bake atlas-<name>` to verify the shared
#      Dockerfile builds it.
#
# No per-service Dockerfile is needed — the shared repo-root Dockerfile
# parameterized by ARG SERVICE handles it. See CLAUDE.md
# "Build & Verification" for the post-task-074 workflow.
```

- [ ] **Step 2: Verify syntax**

Run:
```bash
bash -n tools/import-service.sh && echo "syntax OK"
```
Expected: `syntax OK`.

- [ ] **Step 3: Commit**

```bash
git add tools/import-service.sh
git commit -m "docs(tools): update import-service.sh docstring for shared-Dockerfile workflow

Document that no per-service Dockerfile generation is needed — only
services.json + go.work appends."
```

---

## Phase 8 — Update CLAUDE.md

### Task 19: Rewrite `CLAUDE.md` "Build & Verification" section

**Files:**
- Modify: `CLAUDE.md` (Build & Verification section only)

- [ ] **Step 1: Locate the section**

Run:
```bash
grep -n '^## Build & Verification' CLAUDE.md
```
Expected: one line number (e.g., `21:## Build & Verification`).

- [ ] **Step 2: Replace the section body**

Use Edit to replace the section from `## Build & Verification` through the line `For large refactors expect multiple fix-and-rebuild cycles. Don't shortcut the Docker step.` (inclusive) with:

```markdown
## Build & Verification

Before claiming a branch is "done," "ready for PR," or invoking `superpowers:finishing-a-development-branch`, verify the affected services this way:

1. `go test -race ./...` clean in every changed module.
2. `go vet ./...` clean in every changed module.
3. `go build ./...` clean in every changed service.
4. **`docker buildx bake atlas-<svc>` from the worktree root for every service whose `go.mod` was touched.** This is mandatory, not optional. The shared `Dockerfile` at the repo root is parameterized by `ARG SERVICE`; `docker-bake.hcl` enumerates one target per Go service driven by `.github/config/services.json` (single source of truth). `go build`/`go test` against the workspace `go.work` will NOT catch a missing `COPY libs/...` line in the shared Dockerfile — only `docker buildx bake` will. CI catches it too, but each round-trip wastes a CI cycle and turns "verified" into a lie.

To build everything locally: `docker buildx bake all-go-services` (or `tools/build-services.sh` — a thin wrapper).

Adding a new shared lib requires appending two `COPY` lines to the repo-root `Dockerfile` (one in the mod-only block, one in the source block) and one `./libs/<name>` line to `go.work`. That's it — no per-service edits.

For large refactors expect multiple fix-and-rebuild cycles. Don't shortcut the bake step.
```

- [ ] **Step 3: Sanity-check the rewrite**

Run:
```bash
sed -n '/^## Build & Verification/,/^## /p' CLAUDE.md | head -40
```
Expected: the new section content, followed by the next `## ...` header.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(claude.md): rewrite Build & Verification for shared Dockerfile

Replace the four-place lib-list paragraph with a one-place rule:
edit the repo-root Dockerfile (two COPYs) and go.work (one use line).
Verification command becomes 'docker buildx bake atlas-<svc>' (or
'docker buildx bake all-go-services' for the full fleet)."
```

---

## Phase 9 — Final verification

### Task 20: End-to-end local sanity sweep

**Files:** none (verification only)

- [ ] **Step 1: Confirm tree is clean**

Run: `git status --short`
Expected: empty.

- [ ] **Step 2: Confirm no per-service Go Dockerfile remains**

Run:
```bash
find services -maxdepth 2 -name 'Dockerfile*' -printf '%p\n' | sort
```
Expected:
```
services/atlas-assets/Dockerfile
services/atlas-pr-bootstrap/Dockerfile
services/atlas-ui/Dockerfile
```

- [ ] **Step 3: Confirm the four new/changed top-level files exist**

Run:
```bash
ls Dockerfile docker-bake.hcl
git diff --name-only HEAD~20..HEAD | grep -E '^(Dockerfile|docker-bake\.hcl|\.github/actions/docker-build/action\.yml|\.github/workflows/(pr-validation|main-publish)\.yml|deploy/compose/docker-compose\.(core|socket)\.yml|tools/(build-services\.sh|import-lib\.sh|import-service\.sh)|CLAUDE\.md)$' | sort -u
```
Expected: all the listed paths appear. (Adjust `HEAD~20` if the commit count differs.)

- [ ] **Step 4: Bake the full fleet one more time**

Run:
```bash
docker buildx bake all-go-services 2>&1 | tail -30
```
Expected: every target succeeds. Cache hits should dominate (the lib mod COPYs and `go mod download` step are now shared across all targets).

- [ ] **Step 5: `go test -race ./...` sanity**

CLAUDE.md item 1 is mandatory, even though this task doesn't change Go source. Some Go modules use `go.work.sum` which may have been touched.

Run:
```bash
go test -race ./...
```
Expected: pass.

If `go.work.sum` is dirty after this run, commit the change:
```bash
git status --short
git add go.work.sum 2>/dev/null && git commit -m "chore: refresh go.work.sum" || echo "no go.work.sum changes"
```

- [ ] **Step 6: `go vet ./...` and `go build ./...` sanity**

Run:
```bash
go vet ./...
go build ./...
```
Expected: clean.

- [ ] **Step 7: Confirm post-commit state**

Run:
```bash
git rev-parse --show-toplevel
git branch --show-current
git log --oneline -20
```
Expected: top-level ends with `.worktrees/task-074-docker-build-shared-dockerfile`, branch `task-074-docker-build-shared-dockerfile`, and the last ~15 commits trace Phases 1–8.

If `--show-toplevel` does NOT end with the task worktree path, or branch differs, STOP and investigate — do not push or open a PR until the worktree is correct.

- [ ] **Step 8: Code-review gate**

CLAUDE.md "Code Review Before PR" requires running `superpowers:requesting-code-review` BEFORE opening the PR. Dispatch the appropriate reviewer agents:

- `backend-guidelines-reviewer` — N/A here (no Go source changes); skip.
- `frontend-guidelines-reviewer` — N/A here (no TS/React changes); skip.
- `plan-adherence-reviewer` — YES. The plan has 20 tasks; verify each is implemented as written.

Run the review per `superpowers:requesting-code-review`. Resolve any findings before proceeding to PR.

- [ ] **Step 9: No commit for this task** — verification only. Any fixes from prior steps were committed inline.

---

## Self-Review (writing-plans skill checklist)

**Spec coverage — PRD acceptance criteria (§10) mapping:**

| PRD criterion | Plan task |
|---------------|-----------|
| One shared `Dockerfile` parameterized by `ARG SERVICE`, builds every in-scope service | Tasks 1, 4, 6 |
| All per-service `services/atlas-*/Dockerfile` for in-scope go-services deleted | Task 13 |
| All `Dockerfile.dev` / `Dockerfile.debug` deleted | Task 14 |
| `atlas-ui`, `atlas-assets`, pure-nginx Dockerfiles untouched | Task 13 Step 4; verified Task 20 Step 2 |
| `docker-bake.hcl` at repo root with per-go-service target + `all-go-services` group | Task 5 |
| BuildKit cache mounts on go mod download / go build steps | Task 1 (cache mounts on the `RUN go build` step; `go mod download` is implicit through workspace mode + the same cache mount) |
| `go mod edit -replace` + `go mod tidy` block removed or justified | Tasks 2 & 3 (gate + conditional) + design.md update |
| `pr-validation.yml` `build-docker` is a single bake job; `detect-changes` matrix still drives selection; tag semantics unchanged; `update-pr-overlay` still consumes `docker-services-matrix` | Task 8 |
| `main-publish.yml` equivalent change | Task 9 |
| `docker-build/action.yml` rewritten to wrap bake | Task 7 |
| `deploy/compose/*.yml` use shared Dockerfile with `build.args.SERVICE`; compose build/up work | Tasks 10, 11, 12 |
| `tools/*.sh` audited; obsolete removed, surviving updated | Tasks 15, 16, 17, 18 |
| `CLAUDE.md` Build & Verification rewritten | Task 19 |
| Single-service PR build green | Task 8 + post-merge CI on this PR itself |
| Multi-service change green + measurable wall-time reduction | Task 9 + post-merge measurement (deferred per PRD §5; the bake architecture itself is the deliverable) |
| `update-pr-overlay` still produces correct `bot/pr-<N>-resolved` | Task 8 Step 2 (verify job untouched) |

**Placeholder scan:** no `TBD`, `TODO`, `implement later`, `add appropriate X`, `similar to Task N`, or `write tests for the above` markers remain. The one conditional gate (Task 3) has explicit skip semantics in Task 2's interpretation step.

**Type consistency:** every output field, env var, and JSON shape used across tasks matches:
- `targets`/`image-name-map`/`tags`/`platform`/`push`/`cache-scope`/`registry`/`registry-username`/`registry-password`/`provenance`/`sbom` are consistent between Task 7 (action definition) and Tasks 8 & 9 (callers).
- `docker-services-matrix` row shape (`{name, path, docker_context, docker_image}`) is consumed identically in Tasks 8 and 9 via the same `jq -c '[.[].name]'` / `jq -c 'map({(.name): .docker_image}) | add'` recipe.
- Service name list is sourced exclusively from `.github/config/services.json` (`type == "go-service"`) — bake file (Task 5), deletion list (Task 13), context.md §2, design.md §3.4. No drift surface.
- Inner-module discovery glob (`ls -d services/${SERVICE}/atlas.com/*/ | head -1`) is identical in Task 1 Step 1 and Task 3 Step 1.

---

## Execution Handoff

Plan complete and saved to `docs/tasks/task-074-docker-build-shared-dockerfile/plan.md`. Companion context at `docs/tasks/task-074-docker-build-shared-dockerfile/context.md`.

Per CLAUDE.md `/execute-task` defaults: subagent-driven, in this existing worktree, no confirmation prompt.

Next: `/clear`, then `/execute-task task-074`.
