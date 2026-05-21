# Docker Build Optimization via Shared Parameterized Dockerfile + buildx bake — Design

Version: v1
Status: Draft
Created: 2026-05-21
Companion to: `prd.md`
---

## 1. Scope recap

PRD §1 establishes the problem (≈50 near-identical `services/atlas-*/Dockerfile` files, four-place lib-list drift hazard, multi-service CI runs that pay per-runner setup × N) and the chosen direction (one parameterized Dockerfile + `docker buildx bake` + collapsed CI matrix). This design picks the concrete shape of each artifact, justifies the choices, and records the open-question decisions from PRD §9.

In-scope artifacts:

- `Dockerfile` (repo root, parameterized via `ARG SERVICE`).
- `docker-bake.hcl` (repo root, JSON-driven matrix from `.github/config/services.json`).
- `.github/actions/docker-build/action.yml` (rewritten to wrap bake).
- `.github/workflows/pr-validation.yml` `build-docker` job (collapsed matrix → single job; per-PR tag/push semantics preserved).
- `.github/workflows/main-publish.yml` `build-amd64` / `build-arm64` / `create-manifest` jobs (each collapsed; same multi-arch manifest semantics).
- All `services/atlas-*/Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug` for Go services (deleted).
- `deploy/compose/docker-compose.{core,socket}.yml` (build blocks rewritten to shared Dockerfile + `build.args.SERVICE`).
- `tools/build-services.sh`, `tools/inject-dockerfile-replace.sh`, `tools/import-service.sh`, `tools/import-lib.sh` (audited; some deleted, some rewritten).
- `CLAUDE.md` Build & Verification section (rewritten one-place rule).

Out-of-scope (unchanged):

- `services/atlas-ui/Dockerfile` (Next.js / nginx, separate template).
- `services/atlas-assets/Dockerfile` (pure nginx).
- `services/atlas-pr-bootstrap/Dockerfile` (alpine + rpk + tools; spot-checked — pure-nginx-shaped, not Go-template-shaped).
- `services/atlas-wz-extractor/Dockerfile` is **folded in** (it is a Go service following the same template; the runtime image is multi-stage with a Go binary, identical pattern).
- Cache backend switch (`type=gha` → `type=registry`), distroless/scratch runtime base, CGO toggles, numeric SLOs — explicit non-goals per PRD §2.
- Sharding bake across multiple runners — explicit non-goal per PRD §2.

## 2. Existing state — relevant facts

These shape the design and constrain the alternatives.

### 2.1 The 50 Dockerfiles are byte-identical modulo four substitutions

A spot check across `services/atlas-account/Dockerfile`, `services/atlas-channel/Dockerfile`, and `services/atlas-wz-extractor/Dockerfile` confirms the only per-service variation in the build stage is:

1. The service directory name (`atlas-account` vs `atlas-channel`).
2. The inner module path under `services/<svc>/atlas.com/<inner-name>/`.
3. The list of libs `COPY`ed in (subset of the 17-lib union).
4. The list of `-replace=` lines in the final `go mod edit` block.

Substitutions (2)–(4) collapse to "use all 17 libs and the conventional inner-module location" — every service consumes a *subset* of the libs, and copying the full set in every image costs only build-time bytes (not runtime — the lib code never appears in the final stage).

### 2.2 A repo-root `go.work` already exists and is authoritative

`go.work` at the repo root enumerates all 17 libs and all 55 service modules. Today's per-service Dockerfile *re-synthesizes* a smaller `go.work` inline because the build context is the repo root but only one service builds. With a single shared Dockerfile, **`COPY go.work go.work` directly** — no synthesis. The same file `go build` uses locally is the one the container uses. This eliminates the entire `RUN echo ... > go.work` block and one of the four drift hazards.

### 2.3 The `go mod edit -replace` + `go mod tidy` block is load-bearing today

`tools/inject-dockerfile-replace.sh` documents *why*: "cold compose builds resolve Chronicle20/atlas-* modules from the in-repo libs/ rather than github.com (which 404s without auth)." Each service's `go.mod` lists `github.com/Chronicle20/atlas/libs/atlas-*` as a normal dependency, which would attempt a network fetch.

`go.work use(...)` does override `require` directives with local replacements — but only when the build is invoked *from* the workspace root (`go build` finds the `go.work` by walking up from cwd). The Dockerfile invokes `go build -C services/<svc>/atlas.com/<name> -o /server`. With `-C`, the workspace is found at `/app/go.work`, so `go build` *should* use workspace mode.

**The empirical test (PRD §4.2):** drop the `RUN go mod edit -replace ... && go mod tidy` block, run a single-service `docker buildx bake atlas-account`, observe whether resolution succeeds against `/app/go.work` alone. If yes, drop the block permanently (one fewer place to drift). If no, keep `-replace` (drop `tidy` independently — `tidy` writes to `go.mod`, which the workspace setup makes unnecessary).

**Expected outcome:** `go.work use(...)` is sufficient. Justification: `go build -C <dir>` with `GOFLAGS` defaulting to workspace mode discovers `/app/go.work` and treats every `use`-listed module as the local source for any matching `require`. This is the documented Go ≥1.18 behavior. The historical reason the `replace` block was added was that the *old* per-service Dockerfile may have predated the workspace pattern, or the inner module's `go.work` discovery was being defeated by something specific. We test, we adapt.

**Fallback if the test fails:** keep `go mod edit -replace` (parameterized to inject the union of all libs for any service) but still drop `go mod tidy` — `tidy` is a normalization step the workspace makes irrelevant for build correctness. Either way, this becomes a generated-once line in the shared Dockerfile, not a per-service hand edit.

### 2.4 The "four-place rule" collapses

Today's drift surfaces:

1. `COPY libs/X/go.mod libs/X/go.sum libs/X/` (build-stage mod layer).
2. Synthesized `go.work use(./libs/X)` line.
3. `COPY libs/X libs/X` (build-stage source layer).
4. `-replace=github.com/Chronicle20/.../X=/app/libs/X` (final mod-edit line).

Post-design surfaces:

1. **One place**: the shared Dockerfile's `COPY libs ...` blocks. Both mod-only and source-tree copies are statically enumerated for **all 17 libs** in the shared file. Adding a new lib means appending two lines (one to the mod-COPY block, one to the source-COPY block). `go.work` is COPY'd as-is from the repo, so adding a lib to `go.work` is the same edit a developer makes locally.

The PRD's stated goal ("disappear or shrink to a one-place rule that is hard to get wrong") is met.

### 2.5 `services.json` is the existing single source of truth for service metadata

CI matrix generation (`.github/actions/detect-changes`) reads `.github/config/services.json` and emits `{name, path, module_path, docker_context, docker_image}` rows. The bake file must agree with `services.json` on which services exist. **Solution:** `docker-bake.hcl` uses HCL's `jsondecode(file("..."))` to read `services.json` and expand a matrix target per `type == "go-service"` entry. Single source of truth, no drift.

### 2.6 No current consumer of Dockerfile.dev / Dockerfile.debug

`grep -rn 'Dockerfile.debug\|Dockerfile.dev' deploy/ tools/ .github/ Makefile` returns nothing. `deploy/compose/docker-compose.*.yml`, all CI workflows, all tool scripts reference only `Dockerfile`. The dev/debug variants are stale artifacts from a pre-monorepo era (their `ADD ./atlas.com/<name>` paths reference a per-service-root context that hasn't existed since the deploy-reorg PR).

**Decision:** delete all `Dockerfile.dev` and `Dockerfile.debug` files outright. Do NOT recreate them as `--target dev` / `--target debug` stages in the shared Dockerfile. If a future need for delve-based debugging arises, recreate the debug stage at that point — speculative now.

### 2.7 atlas-pr-bootstrap spot-check

`services/atlas-pr-bootstrap/Dockerfile` is alpine + rpk + tools, **not** Go-template-shaped. It builds no Go binary, doesn't `COPY libs/`, has no multi-stage build, and is irrelevant to the consolidation. Leave it alone.

## 3. Chosen design

### 3.1 Shared Dockerfile location: `./Dockerfile` at repo root

**Decision:** `./Dockerfile`, not `./build/Dockerfile`.

**Why:** the build context is already the repo root (`context: ../..` from `deploy/compose/`, and `context: .` in CI). Putting the Dockerfile at the root keeps `docker build .` working as the obvious one-liner, matches developer muscle memory, and removes one `-f build/Dockerfile` flag from every invocation. There is no clutter argument — the repo root currently has `go.work`, `go.work.sum`, `README.md`, `CLAUDE.md`, etc.; one `Dockerfile` is in line with that.

**Counter-argument considered:** a `build/` directory would group the Dockerfile with `docker-bake.hcl` and any future Dockerfiles. Rejected because (a) `docker-bake.hcl` belongs at the root for the same reasons (its default discovery path), and (b) any future Dockerfiles for non-Go services are already in their service directories and stay there.

### 3.2 Shared Dockerfile shape

`# syntax=docker/dockerfile:1.24` (matches `services/atlas-pr-bootstrap/Dockerfile`'s pinned syntax). Two stages:

**Stage 1: `build-env`**

```dockerfile
# syntax=docker/dockerfile:1.24
ARG GO_VERSION=1.25.5
ARG ALPINE_VERSION=3.21

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env

ARG SERVICE
RUN test -n "${SERVICE}" || (echo "ERROR: build arg SERVICE is required (e.g., atlas-account)" >&2 && exit 1)

RUN apk add --no-cache git

WORKDIR /app

# Layer 1 — repo go.work (cheap to copy, invalidates when libs or services are added/removed)
COPY go.work go.work.sum ./

# Layer 2 — all 17 atlas libs' go.mod/go.sum (lib-mod-only layer; shared across every service target)
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

# Layer 3 — this service's go.mod (per-target layer, invalidates only when this service's mod changes)
COPY services/${SERVICE}/atlas.com/ services/${SERVICE}/atlas.com/
# (Yes — this brings in this service's go.mod AND its source in one COPY. See §3.3 for why
# we don't try to split mod and source for this leaf.)

# Layer 4 — all 17 atlas libs' source trees (shared across every service target; invalidates
# only when any lib source changes — the same invalidation behavior every service has today)
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

# Layer 5 — discover inner module dir (services/${SERVICE}/atlas.com/<inner>) and build.
# All atlas services follow services/<svc>/atlas.com/<inner>/ with exactly one <inner> dir;
# we glob it rather than hard-coding so a future rename inside one service doesn't require
# touching the shared Dockerfile.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && test -n "$MOD_DIR" || (echo "ERROR: no module dir under services/${SERVICE}/atlas.com/" >&2 && exit 1) \
    && test -f "$MOD_DIR/go.mod" || (echo "ERROR: $MOD_DIR has no go.mod" >&2 && exit 1) \
    && go build -C "$MOD_DIR" -o /server

# Layer 6 — capture config.yaml path (used by the runtime stage COPY below).
# This `RUN` cooperates with the runtime stage's `COPY --from=build-env /app/config.yaml /`.
RUN MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && cp "$MOD_DIR/config.yaml" /app/config.yaml
```

**Stage 2: runtime**

```dockerfile
FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /
COPY --from=build-env /app/config.yaml /

CMD ["/server"]
```

**Notes on the design choices in this stage:**

- **No `go mod download` step.** Today's per-service Dockerfile has `RUN go mod download -C services/<svc>/.../ || true` *before* the source COPY. With workspace mode and a BuildKit cache mount on `/go/pkg/mod`, the download happens implicitly during `go build` and is cache-mounted across runs. The `|| true` in today's version is suspicious anyway (it hides resolution failures). Dropping it means the build fails loudly if a module can't resolve, which is what we want.
- **No `go mod edit -replace` block.** Empirically tested per §2.3; expected to be unnecessary because `go.work` covers the local resolution.
  - **Implementation contract:** the first concrete `docker buildx bake atlas-account` invocation during execution is the gate. If it fails with `cannot find module providing package github.com/Chronicle20/atlas/libs/...`, the block is reinstated as a parameterized `RUN go mod edit -replace=... -replace=... && cd "$MOD_DIR"` step that injects all 17 libs unconditionally (it's a no-op if the service doesn't depend on a given lib — `go mod edit -replace` for an unrequested module is harmless). `go mod tidy` is dropped either way.
  - **Recorded outcome:** [filled in during execution; the plan must include this as an explicit verification task].
- **Single-`COPY` for the service tree.** `COPY services/${SERVICE}/atlas.com/` brings in the inner-module go.mod and source in one layer. The per-service layer split (mod-first for caching) is *not* worth it: the only thing that changes within a single service's tree is its own source, and the lib-mod-download cache (layer 2) has already been amortized across all services via shared GHA cache. The cost of one re-`go build` when *that service's* go.mod changes is the same regardless of layer split, because `go build` re-runs whenever any file in the module changes.
- **Inner-module discovery via glob.** `ls -d services/${SERVICE}/atlas.com/*/ | head -1`. Atlas's naming pattern is "exactly one inner directory per service, named after the service short-name." Globbing keeps the Dockerfile from needing per-service knowledge. A `test -f $MOD_DIR/go.mod` guard fails fast if the convention is violated.
- **Cache mounts.** `--mount=type=cache,target=/go/pkg/mod` and `--mount=type=cache,target=/root/.cache/go-build` on the build step. BuildKit caches are scoped per buildx instance; in CI they ride alongside the `type=gha` cache (which handles *layer* caching). Cache mount semantics: the contents persist across builds on the same builder but do **not** propagate via layer cache export. This is fine — the cache mount accelerates incremental rebuilds on the same runner (rare for ephemeral GHA runners but valuable for local dev and bake's intra-run parallelism, where 5 services bake simultaneously and share `/go/pkg/mod`).

### 3.3 `go.work` strategy: COPY the repo file as-is

**Decision:** `COPY go.work go.work.sum ./`.

**Why:** the repo-root `go.work` is already the source of truth for local development. The current per-service Dockerfile re-synthesizes a slimmed version because each Dockerfile built a single service and didn't want all services in the workspace. With one shared Dockerfile, the slimming buys nothing — the COPY is bytes-cheap, the workspace contains modules whose source isn't COPY'd (those service modules don't exist in the build context for builds of other services, but `go.work` `use` directives for missing modules generate `go: warning: directory ... does not exist` and continue; they don't error).

**Risk:** if a `use (./services/atlas-X/...)` directive points at a directory that *should* be COPY'd but isn't, `go build` for service `Y` would warn but succeed. If it would *fail* (because `Y` somehow depends on `X`'s source via a workspace replacement), that's a real issue. Today's pattern doesn't expose this risk because each Dockerfile copies only the libs that service needs.

**Mitigation:** during execution, after building atlas-account against the unsliced `go.work`, run `docker buildx bake all-go-services` (the everything build) and confirm no service errors on a missing workspace module. If anything fails, slim `go.work` inline as a build-time step:

```dockerfile
RUN sed -i "/use (/,/)/c\\
use (\n$(ls -d libs/atlas-* | sed 's|^|    ./|')\n    ./services/${SERVICE}/atlas.com/$(ls services/${SERVICE}/atlas.com/ | head -1)\n)" go.work
```

Defer that complexity until measured.

### 3.4 `docker-bake.hcl` — JSON-driven matrix

`./docker-bake.hcl`:

```hcl
# Atlas docker bake file. Drives all Go-service image builds.
#
# Single source of truth for the service list: .github/config/services.json.
# Selecting which targets to build is the caller's job:
#   docker buildx bake all-go-services        # everything
#   docker buildx bake atlas-account          # one
#   docker buildx bake atlas-account atlas-ban  # subset
#
# Tag injection from CI: --set "<target>.tags=<image>:<tag>" per service.

variable "SERVICES_CONFIG" {
  default = "./.github/config/services.json"
}

variable "ATLAS_IMAGE_TAG" {
  # Used by local builds (matches the deploy/compose pattern).
  default = "local"
}

locals {
  config      = jsondecode(file("${SERVICES_CONFIG}"))
  go_services = [for s in local.config.services : s if s.type == "go-service"]
}

# One target per Go service, expanded from the JSON. `name` becomes the
# target's identifier; `args.SERVICE` is the build arg the shared Dockerfile
# consumes.
target "go-service" {
  matrix = {
    svc = [for s in local.go_services : s.name]
  }
  name       = svc
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    SERVICE = svc
  }
  # Default local tag. CI overrides per-target via --set.
  tags = ["${svc}:${ATLAS_IMAGE_TAG}"]
  # Cache mounts are encoded in the Dockerfile (RUN --mount=type=cache,...).
  # GHA layer cache is configured per-call by CI via --set "*.cache-from=..."
  # / "*.cache-to=...".
}

group "all-go-services" {
  targets = [for s in local.go_services : s.name]
}

# Default group: same as all-go-services. Lets `docker buildx bake` with
# no args build everything.
group "default" {
  targets = ["all-go-services"]
}
```

**Why HCL not JSON:** HCL supports `jsondecode(file(...))` and list comprehensions; JSON bake files would force us to enumerate every service by hand (and re-introduce drift with `services.json`).

**Why `matrix` not enumerated targets:** the matrix expands at parse time into N targets named `atlas-account`, `atlas-ban`, etc. CI selects via `docker buildx bake atlas-account atlas-ban`. Same shape as today's per-service matrix, just driven by a different mechanism.

**Cache scope:** a single shared `type=gha` scope per platform — `atlas-bake-amd64` and `atlas-bake-arm64`. Collapses the 50 per-service scopes to two. This loses per-service cache isolation (a poison in one cache affects all subsequent builds) but gains massive cache hit improvements on shared layers (the 17 lib mod COPYs and `go mod download` are the same SHA across every service, so the first service's build populates a cache every other service hits). Trade-off acceptable; cache invalidation on this layer requires either a workflow-level cache bust input or simply waiting for the GHA cache TTL.

### 3.5 `.github/actions/docker-build/action.yml` — replaced

Replace the existing per-image composite action with a bake-wrapping composite action. New inputs:

```yaml
inputs:
  targets:           # JSON array of target names (matches docker-services-matrix entries by .name)
    required: true
  image-name-map:    # JSON object: {"atlas-account": "ghcr.io/.../atlas-account", ...}
    required: true
  tags:              # comma-separated tag list (e.g., "pr-123-abc,latest" or single "main-abc-amd64")
    required: true
  platform:          # linux/amd64 | linux/arm64 (single platform per call; multi-arch is two calls + manifest)
    required: true
  push:
    required: false
    default: 'false'
  registry-username: ...
  registry-password: ...
  cache-scope:       # e.g., "atlas-bake-amd64"
    required: true
```

Action body:

1. `docker buildx setup`.
2. Login to registry if `push == true`.
3. Generate `--set` flags per target:
   - For each `target` in `targets`:
     - For each `tag` in `tags`:
       - Append `--set "${target}.tags=${image-name-map[target]}:${tag}"`.
   - Append global `--set "*.platform=${platform}"`, `--set "*.cache-from=type=gha,scope=${cache-scope}"`, `--set "*.cache-to=type=gha,mode=max,scope=${cache-scope}"`, `--set "*.output=type=image,push=${push}"`.
4. Run `docker buildx bake ${targets} ${set-flags}`.
5. Emit a summary table.

This keeps the action a thin shell-script over `docker buildx bake`. Tag string handling (comma-split, trim) is unchanged from today's action — the conversion logic copies cleanly.

### 3.6 `.github/workflows/pr-validation.yml` — collapsed `build-docker`

Replace the `build-docker` job's `strategy.matrix.service` with a single job that:

1. Receives `docker-services-matrix` (unchanged from `detect-changes`).
2. Computes `pushtag` once (same logic as today; output is one tag string used for all built services).
3. Builds an `image-name-map` JSON from the matrix: `{name: docker_image}` for each row.
4. Builds a `targets` JSON list: `[name, name, ...]`.
5. Invokes the new `docker-build` action with both, using `cache-scope: atlas-bake-amd64` and `platform: linux/amd64`.

**Job dependency graph unchanged.** `update-pr-overlay` still depends on `build-docker.result == 'success'` and still consumes `detect-changes.outputs.docker-services-matrix`. The matrix's shape (`{name, path, docker_context, docker_image}`) is unchanged — the bake job consumes the same JSON; only its iteration form differs.

**Conditional skip preserved.** `if: ... docker-services-matrix != '[]'` still applies; an empty matrix skips the whole job.

**One bake log instead of N tiles:** acceptable per PRD §8.3. Bake's default log format prefixes lines with `#<step> [<target>]` so failures are unambiguous. The job summary step writes a per-target results table.

### 3.7 `.github/workflows/main-publish.yml` — collapsed AMD64 + ARM64 + manifest

Three jobs collapse:

- `build-amd64`: matrix → single job, same shape as `pr-validation.yml`'s collapsed `build-docker` but with `platform: linux/amd64`, `push: true`, `tags: "latest-amd64,main-<sha>-amd64"`, `cache-scope: atlas-bake-amd64`.
- `build-arm64`: matrix → single job on `runs-on: ubuntu-24.04-arm`, `platform: linux/arm64`, `cache-scope: atlas-bake-arm64`.
- `create-manifest`: still a matrix (or a single job that loops over `docker-services-matrix`), because each service has its own multi-arch manifest. The work is just `docker manifest create` + `docker manifest push` per service, no build. Collapsing it costs nothing but reads the matrix once.

`update-image-tags` job is unchanged — it iterates `docker-services-matrix` independently.

### 3.8 Old per-service Dockerfiles — deleted

Delete the following files in this PR:

- `services/atlas-account/Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug`
- ... (the same triplet for every `services/atlas-*` that is `type == "go-service"` in `services.json`)
- `services/atlas-wz-extractor/Dockerfile`, `Dockerfile.dev`, `Dockerfile.debug` (if dev/debug exist there)

Preserve (untouched):

- `services/atlas-ui/Dockerfile`
- `services/atlas-assets/Dockerfile`
- `services/atlas-pr-bootstrap/Dockerfile`

The plan task that does this deletion lists every path explicitly; the file generation step does not glob-delete (defensive — avoids accidentally taking out non-go-service Dockerfiles).

### 3.9 `deploy/compose/*.yml` — rewritten build blocks

For every `build:` block today that reads:

```yaml
    build:
      context: ../..
      dockerfile: services/atlas-<svc>/Dockerfile
    image: atlas-<svc>:${ATLAS_IMAGE_TAG:-local}
```

Rewrite to:

```yaml
    build:
      context: ../..
      dockerfile: Dockerfile
      args:
        SERVICE: atlas-<svc>
    image: atlas-<svc>:${ATLAS_IMAGE_TAG:-local}
```

`docker compose build <svc>` continues to work, `docker compose up` continues to work. `atlas-assets`'s non-standard block (`context: ../../services/atlas-assets`) is left alone.

This change is mechanical — a `sed`-style edit per service block — and applies to `docker-compose.yml` (only contains nginx, no change), `docker-compose.core.yml` (most services), and `docker-compose.socket.yml` (atlas-login, atlas-channel).

### 3.10 `tools/*.sh` — audit and replace

- `tools/inject-dockerfile-replace.sh` — **delete**. The script's purpose was injecting the four-place rule's third surface into each per-service Dockerfile. With one Dockerfile, that surface is gone. The script has no other use.
- `tools/import-service.sh` — **update**. Today this script imports a new service from an external repo. After consolidation, importing a service requires (1) appending to `services.json`, (2) optionally adding to `go.work`'s `use(...)` block. It does **not** require generating a `Dockerfile`. Update the script's docstring + remove any Dockerfile-template generation.
- `tools/import-lib.sh` — **update**. Importing a new lib requires (1) appending to `go.work`, (2) appending two `COPY` lines to the shared `Dockerfile` (mod + source). Update the script's docstring; optionally automate the Dockerfile edit via `sed -i` insertion at marked anchor comments. Defer the automation if the manual edit is documented clearly.
- `tools/build-services.sh` — **rewrite as a one-liner wrapper**. Body becomes:

  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  exec docker buildx bake all-go-services "$@"
  ```

  Drop the bespoke per-service loop; bake handles parallelism and selection.

### 3.11 `CLAUDE.md` — rewritten Build & Verification

Replace today's four-place-rule paragraph with:

> 4. **`docker buildx bake atlas-<svc>` from the worktree root for every service whose `go.mod` was touched.** This is mandatory, not optional. The shared `Dockerfile` at the repo root parameterizes by `ARG SERVICE`; `docker-bake.hcl` enumerates one target per Go service driven by `.github/config/services.json`. `go build`/`go test` against the workspace `go.work` will NOT catch a missing `COPY libs/...` line in the shared Dockerfile — only `docker buildx bake` will. CI catches it too, but each round-trip wastes a CI cycle and turns "verified" into a lie.
>
> To build everything locally: `docker buildx bake all-go-services`.
>
> Adding a new shared lib requires appending two lines to the shared Dockerfile (one `COPY` to the mod-only block, one `COPY` to the source block) and adding the lib to `go.work`. That's it — no per-service edits.

This shrinks "four places" to "one place" (the shared Dockerfile's lib block, which appears twice within one file — mod-only and source — so it's really two adjacent edits). The hand-edited go.work is unchanged from today's developer workflow (it's the same file `go work use` would edit).

## 4. Build correctness — equivalence proof

PRD §8.2 demands behavioral equivalence with today's images. The post-design image differs from today's in exactly the following ways:

| Aspect | Today | Post-design | Risk |
|--------|-------|-------------|------|
| Build context | `.` (repo root) for all services since deploy-reorg | Same | None |
| Builder base | `golang:1.25.5-alpine3.21` | Same | None |
| Lib mod COPY set | Per-service subset of libs | Union of all 17 libs | Build-stage size grows ~20%; runtime image unchanged because libs are not COPY'd into stage 2 |
| `go.work` content | Synthesized per-service subset | Full repo `go.work` | See §3.3 risk + mitigation |
| `go mod download` | Explicit `RUN`, swallowed errors | Implicit during `go build`, errors propagate | Strictly safer; matches modern Go practice |
| `go mod edit -replace` | Per-service block | Removed (pending §2.3 empirical test) | If test fails, parameterized fallback documented |
| `go build` invocation | `go build -C <hardcoded-path> -o /server` | `go build -C $(glob) -o /server` | Glob resolves to the same path; explicit guard fails fast otherwise |
| Runtime base | `alpine:3.23` | Same | None |
| `EXPOSE` | `8080` | Same | None |
| `CMD` | `["/server"]` | Same | None |
| `config.yaml` location | `/config.yaml` | Same | None |
| `apk add libc6-compat` | Yes | Yes | None |

The output image is behaviorally identical. The `/server` binary may not be *byte*-identical (build cache, link-order differences from cache-mounted `/root/.cache/go-build`), but it is functionally identical given the same source.

**Smoke test plan** (executed during plan execution, not part of the design but recorded here):

1. Pre-change: `docker build -f services/atlas-account/Dockerfile -t atlas-account:before .` and `docker run --rm atlas-account:before /server --help` (or equivalent quick exec).
2. Post-change: `docker buildx bake atlas-account` and same exec.
3. Compare `docker inspect` output: same `Cmd`, `ExposedPorts`, `WorkingDir`.

This is a per-service spot check, run for `atlas-account` (REST + DB pattern) and `atlas-channel` (socket pattern) at minimum.

## 5. Performance — qualitative reasoning

PRD §8.1 explicitly forbids numeric SLOs; the goal is "the maintainer can observe a real reduction in wall time on lib-bump-style PRs." The expected gains:

**Single-service PR (status quo today):**

| Step | Today | Post-design |
|------|-------|-------------|
| Runner provisioning + checkout | ~25s | ~25s |
| Buildx setup | ~5s | ~5s |
| GHA cache restore (per-service scope) | ~10–30s | ~10–30s (shared scope cache; warmer because hit by every service) |
| `apk add git` | ~3s | ~3s |
| Layer reuse for cached steps | ~2s | ~2s |
| Build (uncached path) | full | full (one extra `COPY` for unused libs ≈ 1–2s) |
| **Total uncached** | baseline | **~1–3s slower** (extra lib COPYs) |
| **Total cached** | baseline | **~5–15s faster** (warm shared cache from concurrent builds) |

Single-service is roughly a wash, leaning slightly faster on warm cache.

**Lib-bump PR (touches 30 services):**

| Step | Today | Post-design |
|------|-------|-------------|
| Runner cost | 30 × (25s + 5s + 10s) = ~1200s wall-clock spread across runners | 1 × (25s + 5s + 30s) = ~60s |
| `apk add git` | 30 × 3s = ~90s | 1 × 3s = 3s |
| Lib mod-download (uncached due to lib bump) | 30 × ~15s = ~450s | 1 × ~15s = 15s (shared first-target step; subsequent targets hit `/go/pkg/mod` cache mount) |
| Per-service build | 30 × ~30s = ~900s wall-clock; gated by GHA concurrent-runner quota | Bake runs all 30 in parallel within one runner's CPU budget; ~30s × (30 / parallelism) — buildx defaults to one builder, parallelism ~4–6 → ~150–225s |
| **Wall time** | bounded by quota (often 6 concurrent → 5 batches × ~120s = ~600s) | **~250–300s** |

Expected wins on multi-service PRs: roughly 2–3× faster wall time, with the bigger gain being eliminating the per-runner overhead × N tax. The exact number depends on GHA runner contention and cache state and will be measured post-merge, not predicted.

**Risks:**

- Single-job timeout cap (GHA: 6 hours by default — not a concern).
- Single-builder parallelism cap. `docker buildx bake` defaults to running targets in parallel up to the builder's `parallelism` setting. For an ubuntu-latest runner (2 vCPU), 4–6 parallel `go build` invocations is healthy; beyond that, contention degrades wall-time. If the consolidated bake job becomes slow on large fan-outs, the easy mitigation is `--parallelism 4` or sharding — but that's the deferred non-goal per PRD §2.

## 6. Migration plan — rip-and-replace

PRD §9.8 defaults to rip-and-replace. This design confirms that choice:

- The PR is one atomic change. CI on the PR validates that bake builds every Go service successfully and that compose builds work locally.
- The old per-service Dockerfiles are deleted in the same commit as the new shared Dockerfile + bake file + action + workflow edits.
- No feature flag, no transitional period. If the PR's CI is green for every service and the spot-check smoke tests pass, the change merges as one unit.
- Rollback if needed: revert the PR. The 50 deleted Dockerfiles return; the shared Dockerfile vanishes. Clean revert.

No staged rollout because the alternative (running both Dockerfiles in parallel for a window) doubles the maintenance surface during the window without buying meaningful safety — the change is *all-or-nothing* by nature (CI either uses the per-service matrix or the bake job, not both).

## 7. Open-question decisions (PRD §9)

| Q# | Question | Decision | Rationale (link to section) |
|----|----------|----------|------------------------------|
| 1 | Dockerfile location | `./Dockerfile` at repo root | §3.1 |
| 2 | `go.work` strategy | COPY repo file as-is | §3.3 |
| 3 | Dockerfile.dev / Dockerfile.debug | Delete entirely | §2.6 |
| 4 | atlas-pr-bootstrap shape | Pure-nginx-shaped — leave alone | §2.7 |
| 5 | Keep `go mod edit -replace`? | Drop pending empirical test; fallback documented | §2.3 |
| 6 | GHA cache scope key | Two shared scopes (`atlas-bake-amd64`, `atlas-bake-arm64`) | §3.4 |
| 7 | `tools/*.sh` audit | Delete `inject-dockerfile-replace.sh`; update `import-{lib,service}.sh`; rewrite `build-services.sh` as bake wrapper | §3.10 |
| 8 | Migration plan | Rip-and-replace, one atomic PR | §6 |

## 8. Acceptance criteria mapping (PRD §10)

| PRD criterion | Met by |
|---------------|--------|
| One shared `Dockerfile` (parameterized by `ARG SERVICE`) at agreed location, builds every in-scope Go service to a behaviorally equivalent image | §3.1, §3.2, §4 |
| All per-service `services/atlas-*/Dockerfile` for in-scope Go services deleted | §3.8 |
| All `services/atlas-*/Dockerfile.dev` and `.debug` deleted | §2.6, §3.8 |
| `services/atlas-ui/Dockerfile`, `services/atlas-assets/Dockerfile`, pure-nginx Dockerfiles untouched | §1, §3.8 |
| `docker-bake.hcl` at repo root with per-Go-service target + `all-go-services` group | §3.4 |
| BuildKit cache mounts present on `go mod download` and `go build` steps | §3.2 |
| `go mod edit -replace` + `go mod tidy` block removed or justified | §2.3 |
| `pr-validation.yml` `build-docker` is single bake invocation; `detect-changes` still drives matrix; tagging unchanged; `update-pr-overlay` still gets `docker-services-matrix` | §3.6 |
| `main-publish.yml` equivalent change | §3.7 |
| `docker-build/action.yml` rewritten to wrap bake | §3.5 |
| `deploy/compose/*.yml` build Go services via shared Dockerfile with `build.args.SERVICE`; `docker compose build/up` work | §3.9 |
| `tools/*.sh` audited; obsolete removed; surviving updated | §3.10 |
| `CLAUDE.md` Build & Verification rewritten | §3.11 |
| Representative single-service PR build green | §4 smoke test |
| Representative multi-service change green + measurable wall-time reduction | §5 (qualitative) |
| `update-pr-overlay` still produces correct `bot/pr-<N>-resolved` | §3.6 (unchanged dependency) |

## 9. Risks and mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `go.work` covers `-replace` resolution | High (expected) | If not: parameterized `-replace` block, §2.3 fallback |
| Full repo `go.work` causes warnings/errors for services not in build context | Low | Slim `go.work` inline if observed, §3.3 mitigation |
| Inner-module glob picks wrong dir | Very low (atlas convention is consistent) | Explicit `test -f $MOD_DIR/go.mod` guard fails fast |
| Single shared GHA cache scope gets poisoned | Low | Cache-bust input on workflow; GHA cache TTL eventually clears |
| Buildx parallelism saturates 2-vCPU runner on big fan-outs | Medium | `--parallelism N` flag; sharding deferred (non-goal) |
| Bake log format hides per-target failure context | Low | Bake prefixes lines with target name; summary step prints per-target result table |
| Cache mount cache-miss on cold GHA runners | Inherent to cache mounts | Layer cache (via `type=gha`) handles the cold path; cache mount accelerates intra-run parallelism only |
| Compose `build.args` syntax differs across compose v1/v2 | Very low (project uses v2 exclusively) | Spot-tested during plan phase |

## 10. What this design explicitly does NOT do

- Does not switch GHA cache backend (`type=gha` → `type=registry`). PRD §2 non-goal.
- Does not change the runtime base image (`alpine:3.23` stays). PRD §2 non-goal.
- Does not change `EXPOSE`, `CMD`, or `CGO_ENABLED`. PRD §2 non-goal.
- Does not introduce numeric performance SLOs. PRD §8.1.
- Does not shard the consolidated bake job. PRD §2 non-goal.
- Does not preserve `Dockerfile.dev` / `Dockerfile.debug` as `--target` stages. §2.6.
- Does not generate `docker-bake.hcl` from `services.json` at build time — the HCL `jsondecode(file(...))` reads it at parse time, which is the right granularity.
- Does not modify the `update-pr-overlay`, `update-image-tags`, or `create-manifest` jobs' core logic — only their input shape (one bake job result instead of N matrix results).
