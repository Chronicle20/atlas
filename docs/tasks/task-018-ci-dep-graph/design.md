# Design — Dep-Graph-Driven CI Matrix Selection

## Problem

CI for this monorepo (56 services × 2 architectures) takes 20–25 minutes whenever any shared library changes. Today, `detect-changes` treats *any* file under `libs/**` as a "library change" and sets `BUILD_ALL=true`, which:

- Rebuilds every service on PRs (AMD64 only, but still 56 Docker builds + 56 Go test jobs).
- Rebuilds every service on main (56 × AMD64 + 56 × ARM64 + 56 manifest joins).
- Saturates GitHub Actions runner concurrency, serializing work that could otherwise parallelize.
- Evicts GHA cache entries frequently — with 56 services × 2 arches = 112 cache scopes, the 10 GB per-repo quota is under constant pressure.

In reality, most library changes affect a small subset of services. A change to `atlas-saga` has no effect on services that don't import `atlas-saga`. The build should reflect that.

## Goal

Replace "any library change → build everything" with "any library change → build exactly the services/libraries whose Go module graph transitively depends on the changed library." Preserve safety by falling back to the current "build all" behavior on any tool failure.

## Non-Goals

- Not changing what a service/library change itself triggers (files under `services/<svc>/**` still trigger that service; files under `libs/<lib>/**` still trigger that library).
- Not changing `go.work`, `.github/**`, or `--force-all` behavior — all three still trigger a full rebuild.
- Not adding content-hash skipping inside Docker builds (was considered; out of scope).
- Not changing Dockerfile structure, multi-arch layout, or cache strategy.
- Not touching atlas-ui or atlas-assets build paths.

## Architecture

### Moving piece: `tools/cideps/`

A small in-repo Go program responsible for computing the affected-module set.

**Why Go:** we're parsing `go.mod` files; `golang.org/x/mod/modfile` is the standard way; the rest of the repo is Go.

**Where it runs:** inside `.github/actions/detect-changes`, after the existing file-diff step.

**What it replaces:** the `HAS_LIBRARY_CHANGES == "true"` branch inside `pr-validation.yml` and `main-publish.yml` that today flips `BUILD_ALL=true`.

### Data flow

```
git diff
  → detect-changes/action.yml (unchanged: emits changed-services, changed-libraries)
  → cideps (new: emits enriched matrices via graph closure)
  → workflow matrix strategies (unchanged consumers)
```

The matrix consumers (`test-go-libraries`, `test-go-services`, `build-docker`, `build-amd64`, `build-arm64`, `create-manifest`) keep their current shape. Only what feeds them narrows.

## The `cideps` tool

### CLI

```
cideps \
  --changed-libs=atlas-kafka,atlas-tenant \
  --changed-services=atlas-account \
  --config=.github/config/services.json \
  [--force-all]
```

### Output (stdout, single JSON object)

```json
{
  "go-services": [
    { "name": "...", "path": "...", "module_path": "...", "docker_image": "..." }
  ],
  "go-libraries": [
    { "name": "...", "path": "...", "module_path": "...", "coverage_threshold": 0 }
  ],
  "docker-services": [
    { "name": "...", "path": "...", "docker_context": "...", "docker_image": "..." }
  ],
  "reason": "lib atlas-kafka changed; 7 services affected via transitive closure"
}
```

The shape of each array element matches what the existing workflows already pass to matrix strategies — this is a drop-in replacement for the `jq` pipelines in `detect-changes`.

### Internals

1. **Graph build.** Walk `libs/*/go.mod` and `services/*/atlas.com/*/go.mod`. For each file:
   - Parse with `golang.org/x/mod/modfile`.
   - Derive the module's **short name** from its directory — `libs/atlas-kafka` → `atlas-kafka`, `services/atlas-account/atlas.com/account` → `atlas-account`. The module path inside `go.mod` is not reliable (services use short names like `module atlas-account`; libs use full paths like `module github.com/Chronicle20/atlas/libs/atlas-kafka`). Directory-based normalization gives us a single identifier scheme.
   - Record outgoing edges: for each `require` entry whose module path starts with `github.com/Chronicle20/atlas/libs/`, add an edge from the current module to that lib's short name. Both direct and indirect `require` blocks are treated identically.

2. **Closure.** For each service and library, compute the set of libraries it transitively requires (`libClosure(m) = ∪ deps(m) ∪ ⋃ libClosure(d) for d in deps(m)`).

3. **Selection.**
   - A **service** is affected if its name is in `--changed-services` OR `libClosure(service) ∩ --changed-libs ≠ ∅`.
   - A **library** is affected if its name is in `--changed-libs` OR `libClosure(lib) ∩ --changed-libs ≠ ∅`.
   - With `--force-all`, every service and every library is affected.

4. **Enrichment.** Join each affected name with the corresponding entry in `services.json` to produce matrix rows with `path`, `module_path`, `docker_image`, `docker_context`, `coverage_threshold`.

### Exit codes

- `0` — graph computed and matrices emitted.
- Non-zero — any parse error, walk failure, or malformed `services.json`. stderr carries the reason. No partial output.

## `detect-changes` integration

The action gains a new step after the existing file-diff step:

```yaml
- name: Compute affected modules
  id: affected
  shell: bash
  run: |
    set -eo pipefail
    CHANGED_LIBS=$(echo '${{ steps.detect.outputs.libraries }}' | jq -r 'join(",")')
    CHANGED_SERVICES=$(echo '${{ steps.detect.outputs.services }}' | jq -r 'join(",")')

    FORCE_FLAGS=""
    if [ "${{ steps.detect.outputs.has_go_workspace_changes }}" = "true" ] \
      || [ "${{ steps.detect.outputs.has_workflow_changes }}" = "true" ] \
      || [ "${{ inputs.force-all }}" = "true" ]; then
      FORCE_FLAGS="--force-all"
    fi

    if OUT=$(go run ./tools/cideps \
        --changed-libs="$CHANGED_LIBS" \
        --changed-services="$CHANGED_SERVICES" \
        --config=.github/config/services.json \
        $FORCE_FLAGS 2>cideps.err); then
      echo "$OUT" > affected.json
      echo "go-services=$(jq -c '."go-services"' affected.json)" >> $GITHUB_OUTPUT
      echo "go-libraries=$(jq -c '."go-libraries"' affected.json)" >> $GITHUB_OUTPUT
      echo "docker-services=$(jq -c '."docker-services"' affected.json)" >> $GITHUB_OUTPUT
      echo "reason=$(jq -r '.reason' affected.json)" >> $GITHUB_OUTPUT
    else
      echo "::warning::cideps failed, falling back to build-all. See cideps.err below."
      cat cideps.err
      # Fallback: emit full matrices directly from services.json — the same
      # jq pipelines that live today in pr-validation.yml / main-publish.yml
      # under the BUILD_ALL branch. Copy-pasted verbatim to preserve shape.
      echo "go-services=$(jq -c '[.services[] | select(.type == "go-service") | {name, path, module_path, docker_image}]' .github/config/services.json)" >> $GITHUB_OUTPUT
      echo "go-libraries=$(jq -c '[.libraries[] | {name, path, module_path, coverage_threshold: (.coverage_threshold // 0)}]' .github/config/services.json)" >> $GITHUB_OUTPUT
      echo "docker-services=$(jq -c '[.services[] | select(.docker_image != null) | {name, path, docker_context: (.docker_context // .path), docker_image}]' .github/config/services.json)" >> $GITHUB_OUTPUT
      echo "reason=cideps failed; fell back to build-all" >> $GITHUB_OUTPUT
    fi
```

The action's outputs gain `go-services-matrix`, `go-libraries-matrix`, `docker-services-matrix` (already present in the workflows today — moved into the composite action so both workflows consume a single source of truth). The `jq`-based matrix construction currently living in `pr-validation.yml` and `main-publish.yml` deletes.

## Error handling

| Failure | Behavior |
|---|---|
| `go.mod` fails to parse | cideps exits non-zero with the offending path on stderr → workflow falls back to build-all with a visible step-summary warning |
| `services.json` missing or malformed | Same as above |
| Module appears in the graph but has no `services.json` entry | cideps emits a warning on stderr, skips that entry in the output matrices, continues. Prevents one misconfigured service from killing CI for all others. |
| `--changed-libs` contains a lib name unknown to the graph | Ignored silently (can happen if a lib was deleted in the same PR) |
| `--changed-services` contains a service name unknown to the graph | Same |

## Testing

Unit tests live under `tools/cideps/*_test.go`. Fixtures live under `tools/cideps/testdata/`.

**Fixture-based unit tests.** Each test case sets up a synthetic `libs/` + `services/` tree in a temp dir with hand-crafted `go.mod` files, then runs the graph+selection logic.

Cases:
1. **Direct dep.** Lib A changed → only services directly requiring A are affected.
2. **Transitive dep.** Lib A changed, lib B requires A, service S requires B → S affected.
3. **No-op.** No lib changes, no service changes → empty matrices.
4. **Force-all.** `--force-all` → every service and every library returned.
5. **Union with changed-services.** Changed service X and lib Y where Y affects services {X, Z}; output contains {X, Z} exactly once.
6. **Disconnected lib.** Lib with no dependents changed → no services affected, library itself still flagged.
7. **Indirect require only.** Service S has lib A only in its `// indirect` block; A changed → S affected.
8. **Unknown lib in `--changed-libs`.** Ignored; no error.
9. **Missing `services.json` entry.** Module in graph but not in config → warning to stderr, other entries still emitted.
10. **Malformed `go.mod`.** Non-zero exit, no output.

**Real-repo sanity test.** `TestRealRepoGraph` runs the graph builder against the actual repository tree and asserts a handful of known-true edges:
- `atlas-saga → atlas-constants`
- `atlas-account` requires `atlas-kafka`, `atlas-tenant`, `atlas-rest`
- `atlas-channel` is not empty (sanity check that services are picked up)

This catches breakage when a new Atlas lib naming pattern is introduced.

**No workflow-level integration test.** The composite action is exercised on every PR to this change.

## Edge cases flagged for the reader

1. **Indirect-only deps.** `go.mod` marks some Atlas libs as `// indirect`. Closure must include them; parser treats direct and indirect `require` blocks identically.
2. **Service's `go.mod` itself changing.** Files under `services/atlas-X/**` already count as a service change, so the service rebuilds. Its new dep set is picked up fresh on the next run (graph is recomputed every time; nothing cached).
3. **`go.work` changes.** Keep current behavior (build all). Adding/removing a module from the workspace is rare and usually accompanies service/lib file changes already captured by path-based detection.
4. **Dockerfile-declared libs vs `go.mod`-declared libs can drift.** This design makes `go.mod` the source of truth. A lib COPY'd by a Dockerfile but not required by `go.mod` will *not* trigger a rebuild when that lib changes. Correct behavior (unused copies don't affect the binary), but worth noting. A follow-up lint could flag `COPY libs/X` without a matching `require`; out of scope here.
5. **atlas-ui.** Node-only, not in the Go graph. Its workflow path (`has-ui-changes`) is untouched.
6. **atlas-assets.** `static-service` type, no `go.mod`. Excluded from the graph. Its Docker build is only triggered when its own path changes (current behavior preserved).

## Expected impact

With a typical lib change (e.g., a targeted fix in `atlas-saga`, `atlas-kafka`, or `atlas-tenant`):
- Today: 56 Docker builds × 2 arches on main; 56 Docker builds on PR; 56 Go test jobs.
- With this change: build only the transitive dependents — typically in the range of 5–20 services for most libs, based on a spot-check of `go.mod` files.

Changes that still build everything (unchanged behavior):
- `go.work` modifications
- `.github/**` modifications
- Manual `--force-all`
- A change to a lib that genuinely is required by every service (e.g., `atlas-kafka` or `atlas-tenant` are close to this — the graph will reflect that accurately).

## Open items

None — all resolved during design.
