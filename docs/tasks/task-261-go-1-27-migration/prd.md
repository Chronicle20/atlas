# Go 1.27 Migration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-25
---

## 1. Overview

Atlas pins its Go toolchain in at least fourteen distinct places, and those places
have drifted apart. Today the repository simultaneously builds against Go 1.24,
1.25, and 1.26 depending on which file you read: 96 production modules declare
`go 1.25.5`, one declares `go 1.26.0`, six declare a 1.24.x or 1.25.0 variant, the
workspace file declares `go 1.26.0`, the shared builder image defaults to
`GO_VERSION=1.26.0`, and two CI workflows still set `GO_VERSION: '1.25.5'`. Nothing
enforces agreement between these, so each Renovate bump lands partially and the
spread widens.

This task moves every Go version pin in the repository to Go 1.27.0 — released and
stable, with `golang:1.27.0-alpine3.24` published on Docker Hub — and collapses the
existing three-way drift in the same change. It also bumps the Alpine base from
3.23 to 3.24 so the builder stage matches the runtime stage (`Dockerfile:148`
already uses `alpine:3.24`), and bumps golangci-lint from v2.12.2 to v2.13.1, the
release whose own `go.mod` declares `go 1.26.0` under the upstream rule that "the
minimum Go version must always be latest-1" — i.e. the release that tracks Go 1.27.

Because Go offers no mechanism for a `go.mod` to read its version from an external
file, the 103 `go` directives are irreducibly duplicated. The defence against
re-drift is therefore not deduplication but enforcement: this task introduces a
single declared source of truth for the toolchain versions plus a `verify.sh` guard
that fails when any pin site disagrees with it, following the repository's existing
`tools/lint.versions` + `tools/<name>-guard.sh` conventions.

This is a version-pin migration. No Go 1.27 language feature or stdlib API is
adopted, and no runtime behavior changes.

## 2. Goals

Primary goals:

- Every non-fixture `go.mod` in the repository declares `go 1.27.0`.
- `go.work` declares `go 1.27.0`.
- Every builder image, bake variable, CI workflow, and composite action resolves to
  Go 1.27.0 and Alpine 3.24.
- golangci-lint is pinned to v2.13.1 and runs clean against Go 1.27 modules.
- A single declared source of truth exists for `GO_VERSION`, `ALPINE_VERSION`, and
  `GOLANGCI_LINT_VERSION`, and a guard in `tools/verify.sh` fails the build when any
  pin site disagrees with it.
- Flagless `tools/verify.sh` exits 0 on the resulting branch.

Non-goals:

- Adopting any Go 1.27 language feature, stdlib API, or new `GOEXPERIMENT`.
- Any behavioral change to any service.
- Dependency upgrades beyond the `go.sum` / `go.work.sum` churn that `go mod tidy`
  produces as a direct consequence of the toolchain bump.
- Adding `toolchain` directives. With a patch-precise `go 1.27.0` directive, a
  `toolchain go1.27.0` line is redundant — it would restate the floor the `go`
  directive already implies. The repository has no `toolchain` directives today and
  that convention is preserved.
- The `services/atlas-ui` Node 24 toolchain.
- Migrating the `tools/cideps` test fixtures' embedded `go 1.25.5` strings for their
  own sake (see FR-7).

## 3. User Stories

- As an Atlas developer, I want every module in the workspace to compile under one
  Go version, so that `go build ./...` from the workspace root does not silently
  select a different language version per module.
- As an Atlas developer, I want CI and my local build to use the same Go version, so
  that a build that passes locally cannot fail in CI for toolchain reasons alone.
- As a release engineer, I want the published container images built with the
  current Go release, so that toolchain-level security fixes reach production.
- As a maintainer, I want a single file to edit for the next Go bump and a guard that
  catches any pin I miss, so that the 1.24/1.25/1.26 drift that exists today cannot
  recur.
- As a reviewer, I want the migration to contain no behavioral change, so that a
  green verify plus a diff of version literals is sufficient evidence of correctness.

## 4. Functional Requirements

### FR-1 — Module version directives

**FR-1.1** Every `go.mod` under `libs/`, `services/`, and `tools/` that is not a test
fixture MUST declare exactly `go 1.27.0`. Current state, measured on `main`:

| Current directive | Count | Notes |
|---|---|---|
| `go 1.25.5` | 96 | bulk of `libs/` + `services/` |
| `go 1.24.4` | 4 | `libs/atlas-constants`, `libs/atlas-constants/gen`, `libs/atlas-script-core`, `libs/atlas-retry` |
| `go 1.26.0` | 1 | `services/atlas-data/atlas.com/data` |
| `go 1.25.0` | 1 | `tools/catalog-lint` |
| `go 1.24` | 1 | `tools/packet-audit` (minor-only form) |
| **Total** | **103** | |

**FR-1.2** The form MUST be patch-precise (`go 1.27.0`), not minor-only (`go 1.27`).
`tools/packet-audit`'s minor-only `go 1.24` is normalized to the patch-precise form.

**FR-1.3** No `toolchain` directive is added to any module.

**FR-1.4** `go mod tidy` MUST be run for every module after the directive change, and
the resulting `go.sum` / `go.work.sum` churn committed. `go.work.sum` is already
modified in the working tree on `main`; the branch starts from a clean `main` and the
only `go.work.sum` delta in the final diff must be attributable to this migration.

### FR-2 — Workspace

**FR-2.1** `go.work:1` MUST declare `go 1.27.0` (currently `go 1.26.0`).

**FR-2.2** The 95 `use` entries in `go.work` are unchanged. The 8 modules outside the
workspace that are guard tooling (`tools/atlasguards`, `tools/buffdurationguard`,
`tools/envguard`, `tools/goroutineguard`, `tools/outboxguard`,
`tools/producerseamguard`, `tools/rediskeyguard`, `tools/scopeguard`) are in scope for
FR-1.1 even though they are not workspace members.

### FR-3 — Container builds

**FR-3.1** `Dockerfile:17` — `ARG GO_VERSION` MUST default to `1.27.0` (from `1.26.0`).

**FR-3.2** `Dockerfile:18` — `ARG ALPINE_VERSION` MUST default to `3.24` (from `3.23`),
matching the runtime stage's existing `FROM alpine:3.24` at `Dockerfile:148`. The tag
`golang:1.27.0-alpine3.24` is published.

**FR-3.3** `services/atlas-kafka-precreate/Dockerfile:9-10` MUST carry the same
`GO_VERSION=1.27.0` / `ALPINE_VERSION=3.24` defaults. This Dockerfile is live — the
`atlas-kafka-precreate` bake target at `docker-bake.hcl:143` uses
`context = "services/atlas-kafka-precreate"` with `dockerfile = "Dockerfile"`, which
resolves to this file, not the repo root one.

**FR-3.4** `docker-bake.hcl:26` (`GO_VERSION`) MUST default to `1.27.0` and
`docker-bake.hcl:30` (`ALPINE_VERSION`) to `3.24`.

**FR-3.5** `services/atlas-renders/Dockerfile` MUST be deleted. It hardcodes
`golang:1.25.5-alpine3.21` at line 1 and synthesizes a `go.work` containing a literal
`go 1.25.5` at line 17, and it is **dead code**: `atlas-renders` is a `go-service` in
`docker-bake.hcl:93` and is therefore built by the matrix `go-service` target
(`docker-bake.hcl:106-120`) from the repo-root `Dockerfile`. The repository already
documents this file as a known leftover — `.claude/skills/backend-dev-guidelines/resources/scaffolding-checklist.md:169`
states "the file is dead, not an exception to the rule." Deleting it removes two pin
sites rather than leaving two stale ones that the FR-6 guard would have to either
police or exempt.

Rationale note: this is the one item in the task that is a deletion rather than a
version edit. It is in scope because leaving it would require the guard to carry a
permanent exemption for a file nobody builds.

### FR-4 — CI workflows and composite actions

Every `go-version` input MUST resolve to `1.27.0`:

| File | Line | Current | Required |
|---|---|---|---|
| `.github/workflows/pr-validation.yml` | 41 | `GO_VERSION: '1.26.0'` | `'1.27.0'` |
| `.github/workflows/main-publish.yml` | 33 | `GO_VERSION: '1.26.0'` | `'1.27.0'` |
| `.github/workflows/catalog-lint.yml` | 17 | `GO_VERSION: '1.25.5'` | `'1.27.0'` |
| `.github/workflows/packet-matrix.yml` | 13 | `GO_VERSION: '1.25.5'` | `'1.27.0'` |
| `.github/actions/go-test/action.yml` | 11 | `default: '1.25.5'` | `'1.27.0'` |
| `.github/actions/detect-changes/action.yml` | 280 | `go-version: '1.26.0'` | `'1.27.0'` |

The six `go-version: ${{ env.GO_VERSION }}` / `${{ inputs.go-version }}` call sites
(`catalog-lint.yml:29`, `pr-validation.yml:149,424,460,668,694`, `packet-matrix.yml:25`,
`go-test/action.yml:40`) are indirections and need no edit.

### FR-5 — Lint toolchain

**FR-5.1** `tools/lint.versions` MUST set `GOLANGCI_LINT_VERSION=v2.13.1` (from
`v2.12.2`).

**FR-5.2** The bump MUST be proven, not assumed: `tools/lint.sh` must run to
completion against at least one Go 1.27.0 module and exit 0, with the output captured
in the task folder. The supporting evidence for choosing v2.13.1 is that its `go.mod`
declares `go 1.26.0` under the upstream comment "The minimum Go version must always be
latest-1", whereas v2.12.2's declares `go 1.25.0`. This is inference from the upstream
convention, not a quoted support statement — the design phase must confirm it by
execution.

**FR-5.3** If v2.13.1 introduces new diagnostics that fail existing code, those
findings are fixed in this task if trivial and mechanical; if a finding requires a
behavioral judgement, it is recorded and excluded per the escape-hatch rule in
`.golangci.yml` (an exclusion must carry a comment naming a `docs/TODO.md` burn-down
follow-up). No lint failure may be left unaddressed.

**FR-5.4** `.golangci.yml` requires no version edit; it names no Go version. The
`local-prefixes` and `atlas-tenant` alias notes at `.golangci.yml:23-33` are unaffected.

### FR-6 — Single source of truth and drift guard

**FR-6.1** A declared pin file MUST hold `GO_VERSION`, `ALPINE_VERSION`, and
`GOLANGCI_LINT_VERSION`. It follows the established `tools/lint.versions` shape — a
shell-sourceable `KEY=value` file — so existing consumers (`tools/lint.sh:20`,
`.claude/hooks/format-on-write.sh:29`) keep working. Whether this extends
`tools/lint.versions` or adds a sibling file is a design-phase decision; the
requirement is that exactly one file declares each value.

**FR-6.2** A new guard script MUST fail when any pin site disagrees with the declared
values. Its checked set is exactly the sites in FR-1.1, FR-2.1, FR-3.1–FR-3.4, FR-4,
and FR-5.1. It follows the repo's guard convention: `tools/<name>-guard.sh`, exit 0
clean / non-zero on violation, wired into `tools/verify.sh` via a `step` line
alongside the existing guards at `tools/verify.sh:342-453`.

**FR-6.3** The guard MUST be path-gated in `verify.sh` consistent with the surrounding
guards — it runs when a `go.mod`, `go.work`, `Dockerfile`, `docker-bake.hcl`, workflow,
composite action, the pin file, or the guard's own source changed.

**FR-6.4** The guard MUST explicitly exempt the `tools/cideps` test fixtures (FR-7).
The exemption is a named list or a path predicate, not a silent omission, and carries
a comment saying why.

**FR-6.5** The guard MUST be wired into CI. Given `verify.sh` is the gate, satisfying
FR-6.2 is sufficient if CI runs the same script; if CI enumerates guards separately,
the new guard is added there too.

### FR-7 — Test fixtures are explicitly excluded

**FR-7.1** The eight `go.mod` files under `tools/cideps/testdata/` (`simple/` and
`transitive/` trees) declare `go 1.25.5` as *fixture data*, not as build inputs. They
MUST NOT be bumped as part of this migration, because `tools/cideps` tests assert
against them and the version string is incidental to what they exercise.

**FR-7.2** `tools/cideps/graph_test.go:14` embeds a literal `go 1.25.5` inside a
`go.mod` string built by `TestParseAtlasRequires_DirectAndIndirect`. It is likewise
fixture data and MUST NOT be bumped.

**FR-7.3** Both exclusions MUST be recorded as comments at the guard's exemption list
so a future reader does not "fix" them.

## 5. API Surface

None. This task adds, removes, and modifies no HTTP endpoint, no Kafka topic, no
message schema, and no JSON:API resource. The only new interface is the guard script's
CLI contract (exit 0 clean, non-zero with a per-violation `file:line: expected X, got Y`
report), matching the convention of the existing `tools/*-guard.sh` scripts.

## 6. Data Model

None. No entity, field, relationship, constraint, or database migration is introduced
or altered. No `tenant_id` scoping question arises.

The only persistent-artifact churn is dependency-graph metadata: `go.sum` files across
103 modules and the root `go.work.sum`, regenerated by `go mod tidy` under the new
directive.

## 7. Service Impact

All 14+ Go services are affected identically and mechanically — their `go.mod`
directive changes and their image is built by a newer toolchain. No service's source
changes.

| Area | Change |
|---|---|
| All Go services (`services/atlas-*/atlas.com/*`) | `go.mod` directive → `1.27.0`; `go.sum` retidied; image rebuilt on `golang:1.27.0-alpine3.24` |
| All `libs/atlas-*` | `go.mod` directive → `1.27.0` (includes the four 1.24.4 stragglers) |
| `tools/*` (guards, `catalog-lint`, `packet-audit`, `cideps`) | `go.mod` directive → `1.27.0`; `packet-audit` also normalized from minor-only to patch-precise |
| `services/atlas-renders` | Dead `Dockerfile` deleted (FR-3.5); no source change; still built from the root Dockerfile |
| `services/atlas-kafka-precreate` | Own Dockerfile's ARGs bumped (FR-3.3) |
| `services/atlas-ui` | **Not affected.** Node 24 / nginx; no Go build stage |
| `services/atlas-pr-bootstrap` | **Not affected.** `FROM alpine:3.24` only; no Go build stage |

Cross-service seam risk is low by construction — this task changes no event, no
contract, and no shared type. The realistic failure modes are toolchain-level: a
stricter `go vet` in 1.27, a stdlib behavior change surfacing in a test, or
golangci-lint v2.13.1 reporting new diagnostics.

## 8. Non-Functional Requirements

**Performance.** No performance requirement is asserted, and no benchmark target is
set. Toolchain-level throughput or GC changes in 1.27 are accepted as-is. If a
pre-existing benchmark regresses materially, it is reported, not silently accepted.

**Security.** Moving to the current Go release and to Alpine 3.24 is the security
motivation: both carry upstream fixes absent from 1.24/1.25/1.26 and Alpine 3.21/3.23.
No new secret, credential, or network surface is introduced. `secret-scan.yml` is
unaffected.

**Observability.** Unchanged. No log line, metric, or trace attribute is added,
removed, or renamed.

**Multi-tenancy.** Not applicable. No tenant-scoped data path is touched.

**Reproducibility.** After this task, the next Go bump is a single edit to the pin
file plus a guard-driven sweep. The guard is the durable deliverable; the version
numbers are perishable.

**Compatibility.** A patch-precise `go 1.27.0` directive means any contributor on an
older toolchain triggers an automatic toolchain download, or receives a hard error if
`GOTOOLCHAIN=local` is set. This is a deliberate accepted consequence of the
patch-precise choice and MUST be called out in whatever contributor-facing doc the
design phase identifies as the right home.

## 9. Open Questions

1. **golangci-lint v2.13.1 Go 1.27 support is inferred, not quoted.** The evidence is
   upstream's own `go.mod` (`go 1.26.0` at v2.13.1 vs `go 1.25.0` at v2.12.2) under the
   comment "The minimum Go version must always be latest-1." An attempt to read the
   v2.13.0/v2.13.1 release notes for an explicit "Go 1.27" statement was rate-limited
   by the GitHub API and did not complete. FR-5.2 resolves this by execution rather
   than by citation. If v2.13.1 turns out not to support 1.27, the fallback decision
   (wait for a newer release vs. proceed with a lint version behind the toolchain) is
   escalated rather than assumed.

2. **Where the pin file lives.** FR-6.1 fixes the shape and the uniqueness property but
   not the filename. Extending `tools/lint.versions` reuses two existing consumers but
   overloads a file whose name says "lint"; a sibling `tools/toolchain.versions` is
   cleaner but adds a second file to source. Design-phase call.

3. **Blast radius of new 1.27 diagnostics is unknown until measured.** `go vet` ships
   inside the toolchain, so a 1.27 `vet` may report findings across 103 modules that
   1.26 did not. FR-5.3 states the policy; the volume is unknown at spec time and could
   materially change the size of the task.

4. **Whether `reconcile-bump-prs.yml` needs a change.** It was not inspected in the
   spec scan and contains no `go-version` line, but it exists to reconcile Renovate bump
   PRs and may encode version assumptions in another form.

5. **Renovate behavior after the pin file exists.** FR-6 introduces a value Renovate's
   `gomod` and `dockerfile` managers cannot see. §"Renovate" below sets the policy for
   the managers; whether Renovate should additionally track the new pin file via a
   `customManagers`/regex manager is a design-phase question.

### Renovate

Per the scoping decision, `renovate.json` is in scope. The current config auto-merges
grouped Go minor/patch bumps for both `gomod` and `dockerfile` managers
(`renovate.json:18-42`), and that auto-merge is the mechanism that produced today's
partial, drifted state — a grouped bump can land for some modules and not others with
nothing failing. The requirement is that Go toolchain bumps stop auto-merging and
become a deliberate, guard-verified change. The precise rule shape (dropping
`automerge` from the `go-version` and `dockerfile-golang` groups, versus a broader
restructuring) is a design-phase decision; the acceptance criterion is behavioral, not
textual — see AC-14.

## 10. Acceptance Criteria

- [ ] **AC-1** Every non-fixture `go.mod` (103 files) declares exactly `go 1.27.0`.
      Verified by a repo-wide sweep whose output is pasted, not summarized.
- [ ] **AC-2** No `go.mod` in the repository contains a `toolchain` directive.
- [ ] **AC-3** `go.work:1` reads `go 1.27.0`.
- [ ] **AC-4** `Dockerfile` declares `ARG GO_VERSION=1.27.0` and `ARG ALPINE_VERSION=3.24`.
- [ ] **AC-5** `services/atlas-kafka-precreate/Dockerfile` declares the same two values.
- [ ] **AC-6** `docker-bake.hcl` defaults `GO_VERSION = "1.27.0"` and `ALPINE_VERSION = "3.24"`.
- [ ] **AC-7** `services/atlas-renders/Dockerfile` no longer exists, and
      `docker buildx bake atlas-renders` still succeeds.
- [ ] **AC-8** All six CI pin sites in the FR-4 table read `1.27.0`; a repo-wide grep
      for `1\.2[456]` under `.github/` returns no Go version pin.
- [ ] **AC-9** `tools/lint.versions` (or its successor) pins `GOLANGCI_LINT_VERSION=v2.13.1`,
      and `tools/lint.sh` exits 0 across the repo with the run output recorded in the
      task folder.
- [ ] **AC-10** The pin file exists and is the only place each of `GO_VERSION`,
      `ALPINE_VERSION`, `GOLANGCI_LINT_VERSION` is declared as a source of truth.
- [ ] **AC-11** The drift guard exists, is wired into `tools/verify.sh`, exits 0 on the
      branch, and **is proven to fail**: a deliberate single-pin mutation (e.g. reverting
      one `go.mod` to `go 1.26.0`) makes it exit non-zero, and the failing output is
      recorded. A guard that has only ever passed is not verified.
- [ ] **AC-12** The guard's exemption list names the `tools/cideps` fixtures and
      `tools/cideps/graph_test.go` with a comment explaining why.
- [ ] **AC-13** `tools/cideps/testdata/**/go.mod` and `tools/cideps/graph_test.go:14`
      still read `go 1.25.5`, and `cd tools/cideps && go test ./...` passes.
- [ ] **AC-14** After the `renovate.json` change, a Go toolchain bump PR does not
      auto-merge. Demonstrated by reasoning over the resulting `packageRules` against
      the Renovate schema, since a live PR cannot be produced on the branch.
- [ ] **AC-15** Flagless `tools/verify.sh` exits 0, including the Docker bake and
      `go test -race`. `--quick` / `--no-docker` runs do not satisfy this criterion.
- [ ] **AC-16** The full branch diff contains no change to any `.go` file other than
      (a) lint-driven fixes required by FR-5.3 and (b) the new guard's own source. Any
      other `.go` change is a scope violation and must be justified explicitly.
- [ ] **AC-17** Code review has run per `docs/review-protocol.md` before the PR opens.
