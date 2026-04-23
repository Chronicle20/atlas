# Context — Task 018: Dep-Graph-Driven CI Matrix Selection

## Why this exists

`detect-changes` today flips `BUILD_ALL=true` whenever any file under `libs/**` changes, which rebuilds all 56 services × 2 architectures. Most lib changes affect only a handful of services. See `design.md` for the full rationale.

## Repository shape (key bits)

- Monorepo root: `<home>/source/atlas-ms/atlas`
- Go services: `services/atlas-*/atlas.com/<leaf>/go.mod`
  - Service `go.mod` uses **short** module names, e.g. `module atlas-account`
  - "Service short name" = the `services/atlas-*` directory name, e.g. `atlas-account`
- Go libraries: `libs/atlas-*/go.mod`
  - Library `go.mod` uses **full** module paths, e.g. `module github.com/Chronicle20/atlas/libs/atlas-kafka`
  - "Library short name" = the `libs/atlas-*` directory name, e.g. `atlas-kafka`
- Go workspace: `go.work` at repo root lists every lib and service module
- Service catalog: `.github/config/services.json` — name, path, module_path, docker_image, docker_context, coverage_threshold, type (`go-service` | `static-service`)

## Cross-lib deps inside go.mod

Libs can depend on other libs via `require github.com/Chronicle20/atlas/libs/atlas-X` + `replace github.com/Chronicle20/atlas/libs/atlas-X => ../atlas-X`. Example: `libs/atlas-saga/go.mod` requires `libs/atlas-constants`.

Services list atlas libs in both the direct `require (...)` block and the indirect `require (...)` block, e.g. `libs/atlas-retry` in `services/atlas-account/atlas.com/account/go.mod` is indirect.

Both blocks must be walked — the graph closure needs to include indirect deps.

## The workflows we're touching

- `.github/workflows/pr-validation.yml` — PR CI, builds AMD64 only (no push)
- `.github/workflows/main-publish.yml` — main-branch CI, builds AMD64 + ARM64, pushes, stitches multi-arch manifest
- `.github/actions/detect-changes/action.yml` — composite action that both workflows call to compute changed file sets

Both workflows today contain a `Build matrices` step with inline `jq` pipelines over `services.json`. This task moves that pipeline into the composite action (so there is one implementation, not two) and narrows the matrix via dep-graph closure when possible.

## The new tool — `tools/cideps`

A small Go program that:
1. Walks `libs/*/go.mod` and `services/*/atlas.com/*/go.mod`
2. Parses them with `golang.org/x/mod/modfile`
3. Builds a graph keyed on **short names** (directory-derived) with edges for every `require` of a module under `github.com/Chronicle20/atlas/libs/`
4. Accepts changed libs/services on the CLI, computes the affected set via transitive reverse closure
5. Joins against `services.json` and emits a single JSON object with three matrix arrays on stdout

Module path: `github.com/Chronicle20/atlas/tools/cideps`. Added to `go.work`.

## Invariants the plan preserves

- Matrix row shape (`{name, path, module_path, docker_image}` etc.) stays identical so consumer steps don't change.
- `go.work`, `.github/**`, and `--force-all` still trigger build-all.
- Tool failure ⇒ workflow falls back to the full build-all matrices (no new failure mode).
- atlas-ui (Node) and atlas-assets (`static-service`) are untouched.

## Go module dependency the tool needs

`golang.org/x/mod/modfile` — standard Go modfile parser. Already a transitive dep in the ecosystem; will be added to `tools/cideps/go.mod`.

## How to run the tool locally

From repo root, after `go.work` includes the new module:

```
go run ./tools/cideps \
  --changed-libs=atlas-kafka \
  --changed-services= \
  --config=.github/config/services.json
```

Or from inside `tools/cideps/`:

```
go run . --changed-libs=atlas-kafka --config=../../.github/config/services.json
```

## Expected fixture/test layout

- `tools/cideps/testdata/simple/` — minimal synthetic tree with 2 libs + 1 service
- `tools/cideps/testdata/transitive/` — 3 libs chained + 2 services with disjoint closures
- Unit tests use `t.TempDir()` or the fixtures directly
- One sanity test runs against the real repo root

## Things that are NOT in scope

- No content-hash image skipping.
- No Dockerfile changes, cache strategy changes, or multi-arch restructuring.
- No lint to verify Dockerfile `COPY libs/X` matches `go.mod` requires (noted as follow-up in the design).
- No changes to the Go test matrix shape, only to which rows it contains.
