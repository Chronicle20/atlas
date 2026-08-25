# Go 1.27 Migration — Design

Version: v1
Status: Draft
Created: 2026-08-25
Consumes: `prd.md` (v1), `pin-inventory.md`
Branch: `task-261-go-1-27-migration`

---

## 0. Summary

The migration itself is mechanical: 103 `go` directives, one `go.work`, four
Dockerfile ARGs, two bake variables, six CI pins, one lint pin. The design work is
in three places where the PRD left a decision open or where the spec scan missed a
site:

1. **Where the single source of truth lives** (PRD open question 2) — decided:
   `tools/lint.versions` is renamed to `tools/toolchain.versions` and gains
   `GO_VERSION` and `ALPINE_VERSION`. One file, five mechanical consumer edits.
2. **What the drift guard is and how it is wired** (FR-6) — decided: a pure-bash
   `tools/toolchain-pin-guard.sh` that sources the pin file, enumerates `go.mod`
   files through `git ls-files` (so a module added tomorrow is covered without a
   guard edit), and reports `file:line: expected X, got Y`.
3. **Three findings that change the worklist**, all evidence-backed below:
   - `Dockerfile:94` hardcodes `go 1.26.0` inside the synthesized workspace. Left
     alone it **breaks every service image build** under a 1.27.0 module set. Not
     in `prd.md` and not in `pin-inventory.md`.
   - `go mod tidy` **cannot run standalone** on a service module in this repo
     today. FR-1.4's "run `go mod tidy` for every module" is not executable as
     written; §5 replaces it with an equivalent that is.
   - `renovate.json`'s rule **order** currently defeats any `automerge: false` set
     on the `go` package. AC-14 is not satisfiable by editing the existing
     `go-version` rule in place; §7 gives the ordering that works.

Two PRD open questions are closed by execution rather than deferred: golangci-lint
v2.13.1 (§6) and `reconcile-bump-prs.yml` (§8).

---

## 1. Evidence gathered during design

All commands were run in this worktree at branch point `6c94ee127`.

### 1.1 The local toolchain is already Go 1.27.0

```
$ go version
go version go1.27.0 linux/amd64
$ go env GOTOOLCHAIN
auto
```

Implementation does not need a toolchain download; `GOTOOLCHAIN=auto` also means a
`go 1.27.0` directive is self-servicing for contributors on older toolchains.

### 1.2 golangci-lint v2.13.1 supports Go 1.27 — confirmed by execution

PRD open question 1 and FR-5.2 asked for proof rather than inference from upstream's
`go.mod`. Downloaded the pinned release asset and ran it:

```
$ /tmp/gcl/golangci-lint version
golangci-lint has version 2.13.1 built with go1.27.0 from 6d2288e0 on 2026-08-20T14:28:34Z

$ cd /tmp/gcl/probe && GOWORK=off /tmp/gcl/golangci-lint run ./...   # module declares `go 1.27.0`
0 issues.
GCL_EXIT=0
```

The binary is **built with go1.27.0**, which is stronger than the `go.mod`-based
inference in the PRD: its embedded `go/types` understands the 1.27 language version.
The v2.13.1 pin is safe. Open question 1 is closed; the escalation branch it
described is not needed. FR-5.2's repo-wide `tools/lint.sh` run still happens during
implementation — that proves *this codebase* is clean under v2.13.1, which the probe
above does not.

Release-asset availability is also confirmed (`golangci-lint-2.13.1-checksums.txt`
fetched successfully), so `tools/lint.sh`'s fast download path works for the new pin.

### 1.3 `go mod tidy` fails standalone on a service module — pre-existing

```
$ cd libs/atlas-retry && go mod tidy -diff
EXIT=0

$ cd services/atlas-account/atlas.com/account && go mod tidy -diff
EXIT=1
go: downloading github.com/Chronicle20/atlas/libs/atlas-env v0.0.0
go: atlas-account imports
	github.com/Chronicle20/atlas/libs/atlas-kafka/consumer imports
	github.com/Chronicle20/atlas/libs/atlas-env: reading github.com/Chronicle20/atlas/libs/atlas-env/go.mod at revision libs/atlas-env/v0.0.0: unknown revision libs/atlas-env/v0.0.0
```

This is on the **unmodified tree** — it is a pre-existing property of the repo, not
something the version bump introduces. `atlas-account` carries 16 `replace`
directives but none for `atlas-env`, which it reaches transitively through
`atlas-kafka`; only workspace mode resolves it. `go mod tidy` deliberately ignores
`go.work`, so there is no flag that fixes this. `tools/lint.sh:11-14` documents the
same property from the other direction ("service go.mod files are not
standalone-consistent, so `GOWORK=off` would fail type-loading").

Consequence for §5: the plan must not contain a per-module `go mod tidy` sweep.

### 1.4 `Dockerfile:94` — an unlisted, build-breaking pin site

```
$ grep -rn "go 1\.2" --include='Dockerfile*' .
services/atlas-renders/Dockerfile:16:RUN echo 'go 1.25.5' > go.work && \
Dockerfile:94:         printf 'go 1.26.0\n\nuse (\n'; \
```

`Dockerfile:88-101` synthesizes a minimal `go.work` inside the builder because the
repo-root `go.work` lists ~50 modules absent from the build context. Its `go`
directive is the literal `1.26.0`.

A workspace's `go` directive must be **greater than or equal to** the maximum `go`
directive across its member modules. After FR-1.1 every member declares `go 1.27.0`,
so a `go 1.26.0` workspace file makes `go build -C "$MOD_DIR"` at `Dockerfile:105`
fail workspace load for **every go-service target**. AC-15 (flagless `verify.sh`,
which bakes) would catch it, but only after a full bake cycle — worth designing out
rather than debugging.

`services/atlas-renders/Dockerfile:16` is the same pattern; FR-3.5 deletes that file,
so it needs no separate handling.

`services/atlas-kafka-precreate/Dockerfile` synthesizes nothing (flat module,
`COPY go.mod go.sum ./` + `go mod download`) — its only pins are the two ARGs
already covered by FR-3.3.

### 1.5 `README.md:79` — an unlisted documentation pin

```
$ sed -n '79p' README.md
| Go | 1.25.5+ | All backend services |
```

The prerequisites table is the contributor-facing statement of the required
toolchain and is two minors stale. It is also the natural home for FR/NFR
"Compatibility"'s required callout about patch-precise directives and
`GOTOOLCHAIN=local`.

### 1.6 CI enumerates guards as individual jobs

`.github/workflows/pr-validation.yml` gives each shell guard its own job
(`npc-shop-contract-mirror-guard` at :207, `npc-conversation-contract-mirror-guard`
at :228, and others). CI does **not** run `tools/verify.sh`. FR-6.5's conditional
therefore resolves to its second branch: the new guard needs its own CI job.

### 1.7 `renovate.json` rule ordering defeats a per-package `automerge: false`

Renovate applies `packageRules` in array order; a later matching rule overrides an
earlier one. The current file has, in order:

- index 0 — `matchManagers: [gomod]`, `matchPackageNames: [go]`, minor/patch,
  `automerge: true`
- index 1 — same, major, `automerge: false`
- index 6 — `matchManagers: [gomod]`, `matchUpdateTypes: [patch, minor]`,
  `automerge: true` — **no package filter**, so it matches the `go` package too and
  wins over anything set at index 0
- index 7 — `matchManagers: [gomod]`, major, `automerge: false`

Setting `automerge: false` on index 0 in place therefore does **not** stop a Go
toolchain minor/patch bump from auto-merging: index 6 re-enables it. This is
mechanically why today's drift exists. §7 fixes it by ordering, not by editing
index 0 alone.

### 1.8 `reconcile-bump-prs.yml` — no change needed

PRD open question 4. The workflow triggers on
`deploy/k8s/overlays/main/kustomization.yaml` pushes and reconciles conflicting
**image-tag** bump PRs opened by `main-publish.yml`. Grep for `go`, `version`,
`golang` returns nothing; it encodes no toolchain assumption in any form. Open
question 4 is closed: out of scope, no edit.

### 1.9 A third fixture the PRD did not list

```
$ grep -rn "go 1\.2[456]" --include='*.sh' .
tools/plan-context_test.sh:50:go 1.24
```

`tools/plan-context_test.sh:44-52` writes a throwaway `go.mod` into a temp git repo
as test fixture data — the same category as FR-7's `tools/cideps` fixtures. It is not
a tracked `go.mod`, so the guard's `git ls-files` enumeration never sees it and no
exemption entry is required. It is recorded here and in the guard's comment block so
a future reader does not "fix" it.

---

## 2. Decision D1 — the pin file

**Decision: rename `tools/lint.versions` → `tools/toolchain.versions` (via
`git mv`) and add `GO_VERSION` and `ALPINE_VERSION` alongside the existing
`GOLANGCI_LINT_VERSION`.**

```sh
# tools/toolchain.versions — the single source of truth for the repo's
# toolchain pins (task-261). Shell-sourceable KEY=value; read by tools/lint.sh,
# .claude/hooks/format-on-write.sh, tools/toolchain-pin-guard.sh, and hashed by
# the CI lint-tools cache key.
#
# Nothing here is read at build time by go.mod, go.work, Dockerfile ARG
# defaults, or docker-bake.hcl — none of those formats can read an external
# file. Those pins are duplicated on purpose and machine-checked against this
# file by tools/toolchain-pin-guard.sh. Bump here, run the guard, fix what it
# names.
GO_VERSION=1.27.0
ALPINE_VERSION=3.24
GOLANGCI_LINT_VERSION=v2.13.1
```

Consumers to update (complete list, from `grep -rn 'lint\.versions'` excluding
`docs/tasks/`):

| Site | Change |
|---|---|
| `tools/lint.sh:19` | `# shellcheck source=toolchain.versions` |
| `tools/lint.sh:20` | `source "$ROOT/tools/toolchain.versions"` |
| `tools/lint.sh:38` | usage text: "Versions are pinned in `tools/toolchain.versions`." |
| `.claude/hooks/format-on-write.sh:28-29` | shellcheck directive + `source` path |
| `.golangci.yml:12` | comment reference |
| `.github/workflows/pr-validation.yml:478` | `hashFiles('tools/toolchain.versions')` |

**Alternatives rejected.**

*Extend `tools/lint.versions` in place.* Zero consumer edits, and FR-6.1's
uniqueness property would hold. Rejected because the filename would then be
actively misleading — a file named `lint.versions` declaring the Go and Alpine
versions for every container image is exactly the kind of thing the next
maintainer greps past. The cost of the alternative is six mechanical, greppable
edits, paid once.

*A sibling `tools/toolchain.versions` that `lint.versions` sources.* Rejected:
two files where one will do, and a nested `source` needs `BASH_SOURCE`-relative
path resolution inside a file that `.claude/hooks/format-on-write.sh:29` sources
with `2>/dev/null || exit 0` — a silent-failure seam for no benefit.

**Accepted side effect.** The CI lint-tools cache key at `pr-validation.yml:478`
now also hashes `GO_VERSION`/`ALPINE_VERSION`, so a future Go bump invalidates the
golangci-lint binary cache once. One extra download on one CI run per Go bump;
not worth a second file to avoid.

**Explicitly not done:** teaching `docker-bake.hcl` or the Dockerfiles to read this
file. Dockerfile `ARG` defaults and `go.mod`/`go.work` directives cannot read an
external file at all, so even if bake HCL could, the guard would still be required
for the majority of sites. One enforcement mechanism beats two half-mechanisms.

---

## 3. Decision D2 — the drift guard

**Decision: `tools/toolchain-pin-guard.sh`, pure bash + `grep`/`sed`, no Go and no
`python3`.**

Shape follows the repo's existing shell guards (`tools/service-name-guard.sh`,
`tools/pr-sparse-mirror-guard.sh`): `#!/usr/bin/env bash`, `set -euo pipefail`,
`ROOT="$(cd "$(dirname "$0")/.." && pwd)"`, accumulate violations, print each as
`file:line: expected X, got Y`, exit 1 if any, exit 0 clean.

**Why bash, not a Go analyzer.** The four `tools/*guard` analyzers exist because
their invariants need type information. This one compares string literals in
`go.mod`, YAML, HCL, and Dockerfiles — several of which are not Go at all. Bash also
means the CI job (§4) is `checkout` + `run`, with no `setup-go` step, which matters
because this guard's whole purpose is to be the thing that still works when the Go
pin is wrong.

### 3.1 Checked set

| Class | How the guard finds the sites |
|---|---|
| Module directives | `git ls-files '*go.mod' 'go.mod'`, minus the exemptions in §3.2; each must match `^go ${GO_VERSION}$` exactly |
| Workspace | `go.work` line 1 must match `^go ${GO_VERSION}$` |
| Dockerfile ARGs | `Dockerfile` and `services/atlas-kafka-precreate/Dockerfile`: `ARG GO_VERSION=` and `ARG ALPINE_VERSION=` defaults |
| Bake variables | `docker-bake.hcl`: `default = "..."` inside the `GO_VERSION` and `ALPINE_VERSION` variable blocks |
| CI pins | the six sites in the FR-4 table, by explicit `file:pattern` pairs |
| Lint pin | `tools/toolchain.versions`'s own `GOLANGCI_LINT_VERSION` cross-checked against `tools/lint.sh`'s expectation (presence check only — the pin file *is* the source) |
| README | `README.md` prerequisites row: `\| Go \| ${GO_VERSION}+ \|` |

**Enumerating `go.mod` via `git ls-files` rather than a checked-in list is the
central design choice.** A hardcoded 103-path list is itself a drift site: add a
service next month, forget the guard, and the guard silently passes on 103 of 104
modules. Enumeration means a new module is covered the moment it is tracked, and a
deleted one stops being checked without a guard edit.

The six CI sites are enumerated explicitly (not by pattern) because
`.github/**` contains many unrelated version strings — an over-broad pattern there
produces false positives that get "fixed" by loosening the guard.

Exact-match, not prefix-match: `^go 1\.27\.0$` rejects both `go 1.27` (FR-1.2's
minor-only form) and `go 1.27.1`.

### 3.2 Exemptions

Encoded as a literal array in the guard with the rationale inline:

```sh
# FR-7 (task-261): fixture data, NOT build inputs. tools/cideps' tests assert
# against these trees and the version string is incidental to what they
# exercise; bumping them would break `go test ./...` in tools/cideps for no
# gain. Do NOT "fix" these to match the pin file.
#
# Two more fixture strings live outside this list because they are not tracked
# go.mod files and `git ls-files '*go.mod'` never yields them — recorded here
# so a future sweep does not mistake them for misses:
#   tools/cideps/graph_test.go:14   `go 1.25.5` inside a go.mod string literal
#   tools/plan-context_test.sh:50   `go 1.24` written into a temp fixture repo
EXEMPT_GOMOD=(
    tools/cideps/testdata/  # both simple/ and transitive/ trees (8 files)
)
```

A path-prefix predicate on `tools/cideps/testdata/` rather than eight literal paths:
the eight files are one fixture corpus, and a ninth added to the same corpus should
inherit the exemption.

### 3.3 Self-verification (AC-11)

AC-11 requires the guard be **proven to fail**. Two layers:

1. A `--selftest` flag that copies the tree's checked files into a temp dir, mutates
   one `go.mod` to `go 1.26.0`, runs the checks against the copy, and asserts a
   non-zero exit and a matching violation line. `tools/redis-key-guard.sh` already
   carries a `SELFTEST` knob, so the convention exists. `--selftest` mutates nothing
   under `$ROOT`.
2. A recorded manual demonstration during implementation: revert one real `go.mod`
   to `go 1.26.0`, run the guard, paste the failing output into the task folder,
   restore. This is what AC-11 literally asks for; the `--selftest` flag is what
   keeps it true after the branch merges.

---

## 4. Decision D3 — wiring

### 4.1 `tools/verify.sh`

New `step` in the path-gated block (after the `service name guard` at `:432`, before
the template guards at `:438`), matching the surrounding style:

```sh
# task-261: the repo pins its Go/Alpine/golangci-lint versions in ~110 places
# that no format can cross-reference (go.mod and go.work directives, Dockerfile
# ARG defaults, bake variables, workflow env). Before this guard, a partial
# Renovate bump left the tree building against three different Go versions at
# once with nothing failing. tools/toolchain.versions is the source of truth;
# this asserts every pin site agrees with it.
if touched '(^|/)go\.mod$|^go\.work$|^Dockerfile$|/Dockerfile$|^docker-bake\.hcl$|^\.github/|^tools/toolchain\.versions$|^tools/toolchain-pin-guard\.sh$|^README\.md$'; then
    step "toolchain pin guard" ./tools/toolchain-pin-guard.sh
else
    skip "toolchain pin guard (no pin site changed)"
fi
```

The predicate includes the guard's own source and the pin file, mirroring the
self-inclusion at `verify.sh:394` (scope guard) and `:411` (producer seam guard).
`README.md` is included because it is now a checked site.

### 4.2 CI (FR-6.5)

Per §1.6, add a job to `.github/workflows/pr-validation.yml` modeled on
`npc-shop-contract-mirror-guard` (:207-218):

```yaml
  toolchain-pin-guard:
    name: Toolchain Pin Guard
    needs: detect-changes
    if: needs.detect-changes.outputs.deploy-only != 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: toolchain pin guard
        run: ./tools/toolchain-pin-guard.sh
```

No `setup-go`: pure shell, by design (§3). Gated only on `deploy-only`, like its
two siblings — a `deploy/`-only change set cannot move a Go pin, and everything
else can.

### 4.3 `docs/verification.md`

Add a row to the "Path-gated" section's guard inventory naming the invariant and
the silent failure it prevents ("a partial toolchain bump leaves the tree building
against two Go versions with nothing failing"), matching the existing entries at
`docs/verification.md:181-186` and `:206-215`.

---

## 5. Decision D4 — how the 103 directives are edited

**Decision: `go mod edit -go=1.27.0` per module; no per-module `go mod tidy`;
`go work sync` once at the root; `go.work` line 1 edited via `go work edit -go=1.27.0`.**

`go mod edit -go=` writes the canonical patch-precise form and normalizes
`tools/packet-audit`'s minor-only `go 1.24` (FR-1.2) without a regex. It also
touches nothing else in the file — which is what AC-16's "no other change" wants.

**FR-1.4 is amended.** Its instruction ("`go mod tidy` MUST be run for every module")
is not executable: §1.3 shows it fails on service modules on the *unmodified* tree,
for reasons that predate this task. The requirement's *intent* — that the dependency
metadata is consistent after the bump and any resulting churn is committed — is
satisfied by:

1. `go mod edit -go=1.27.0` in each of the 103 modules.
2. `go work edit -go=1.27.0` at the root.
3. `go work sync` once, committing whatever `go.work.sum` delta results.
4. `go build ./...` and `go test ./...` through `tools/verify.sh` as the consistency
   proof.

Expected `go.sum` churn: **none**. The `go` directive selects the language version
and the module-graph pruning mode; pruning behavior has been constant since 1.17, so
a 1.24/1.25/1.26 → 1.27 move does not change the module graph. If a `go.sum` does
change, that is a signal to stop and explain it, not to commit it blind — it would
mean something other than the directive moved, which AC-16 treats as a scope
violation.

`go.work.sum` on `main` currently carries `github.com/golangci/golangci-lint/v2
v2.12.2` entries (`go.work.sum:273-274`) although no tracked `go.mod` requires
golangci-lint — sum-file residue from a past `go install`. `go work sync` may drop
them. That is benign and in-scope churn; call it out in the PR description rather
than reverting it.

**Ordering:** all 103 `go.mod` edits land before the `go.work` edit is validated,
because Go rejects a workspace whose directive is below a member's. Doing `go.work`
first would leave the tree transiently unbuildable — irrelevant for a squashed
branch, but it makes any mid-sweep `go build` misleading.

---

## 6. Decision D5 — `Dockerfile:94` becomes derived, not pinned

**Decision: re-declare `ARG GO_VERSION` inside the `build-env` stage and interpolate
it into the synthesized workspace, removing the literal entirely.**

```dockerfile
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env

# Re-declare after FROM: a global ARG is not visible inside a build stage.
# Consumed by the synthesized go.work below so the workspace directive can
# never drift from the builder image (task-261).
ARG GO_VERSION
```

and at :94:

```dockerfile
         printf 'go %s\n\nuse (\n' "${GO_VERSION}"; \
```

`ARG` declared before the first `FROM` is global and must be re-declared inside a
stage to be usable there — hence the extra line rather than relying on the
already-declared global.

**Why derive instead of bumping the literal to `1.27.0`.** A pin site that cannot
drift is better than a pin site the guard has to police. This one is uniquely
dangerous: the guard would need to parse a `printf` format string inside a
line-continued `RUN`, and getting that regex subtly wrong yields a guard that passes
while the bake fails. Deriving removes both the site and the parsing problem. The
guard consequently does **not** check `Dockerfile:94`; a comment there says why.

**Trade-off accepted.** The synthesized workspace directive is now exactly the
builder image's Go version. If someone builds with
`--build-arg GO_VERSION=1.26.0` while modules declare `1.27.0`, the build fails at
workspace load with a clear Go error instead of silently compiling at the wrong
language version. That failure mode is strictly better than today's.

`services/atlas-renders/Dockerfile` gets the same treatment by deletion (FR-3.5).
Deleting it is verified by AC-7's `docker buildx bake atlas-renders`, which
exercises the claim that the root Dockerfile is what actually builds it.

---

## 7. Decision D6 — `renovate.json`

**Decision: append two override rules at the END of `packageRules`, and leave the
existing grouping rules in place.**

Per §1.7, editing the index-0 `go-version` rule in place does not work — the
unfiltered gomod rule at index 6 wins. The minimal correct change is a
last-position override, since Renovate's later-rule-wins semantics make position the
enforcement mechanism:

```jsonc
{
  "description": "task-261: Go toolchain bumps never auto-merge. The `go` directive is duplicated across ~110 pin sites (go.mod x103, go.work, Dockerfile ARGs, bake vars, CI env) that Renovate's gomod manager cannot all see, so a merged bump lands PARTIALLY and the tree builds against two Go versions at once — that is how the 1.24/1.25/1.26 spread this task collapses came to exist. MUST stay last in packageRules: the unfiltered gomod patch/minor rule above sets automerge:true and later rules win.",
  "matchManagers": ["gomod"],
  "matchPackageNames": ["go"],
  "automerge": false
},
{
  "description": "task-261: golang base-image bumps never auto-merge — same partial-landing reason as the `go` rule above. tools/toolchain-pin-guard.sh must pass before a Go/Alpine bump merges, and it cannot pass until every pin site moves together.",
  "matchManagers": ["dockerfile"],
  "matchPackageNames": ["golang"],
  "automerge": false
}
```

The existing `groupName` assignments (`go-version`, `dockerfile-golang`) are kept, so
bumps still arrive as one PR per group — grouping is desirable; auto-merging is not.
No `matchUpdateTypes` on either override: every update type, including patch, is
manual.

**AC-14 verification** is by reasoning over the resulting rule array, as the AC
specifies (no live PR is producible on a branch). The argument is exactly §1.7: the
`go` package is matched by rules at index 0, 6 (unfiltered), and the new final rule;
last match wins; the last match sets `automerge: false`. The same trace is written
into the task folder so a reviewer can check it without re-deriving Renovate
semantics.

**No `customManagers` regex manager for `tools/toolchain.versions`** (PRD open
question 5). Rejected as YAGNI, and slightly worse than nothing: Renovate would open
a PR editing the pin file alone, which the guard would immediately fail because the
103 `go.mod` files had not moved. That produces a recurring red PR whose only
resolution is the manual sweep this task's guard already forces through the existing
`gomod`/`dockerfile` managers. The pin file stays human-edited; Renovate keeps
proposing bumps through the managers it already has, and the guard converts a partial
landing from silent to red. Revisit only if the manual sweep proves burdensome in
practice.

---

## 8. Scope delta against the PRD

Everything below is an addition to or amendment of `prd.md` v1 / `pin-inventory.md`,
justified in §1.

| # | Item | Kind | Evidence |
|---|---|---|---|
| 1 | `Dockerfile:94` synthesized `go.work` directive → derived from `ARG GO_VERSION` | **Addition** — missing pin site, breaks every image build if left | §1.4, §6 |
| 2 | `README.md:79` prerequisites row `1.25.5+` → `1.27.0+`, plus the NFR-Compatibility `GOTOOLCHAIN` callout | **Addition** — missing pin site | §1.5 |
| 3 | FR-1.4's per-module `go mod tidy` replaced by `go mod edit` + `go work sync` + verify | **Amendment** — the requirement as written is not executable | §1.3, §5 |
| 4 | `renovate.json` fixed by appending override rules, not by editing the existing rule | **Amendment** — the obvious edit does not achieve AC-14 | §1.7, §7 |
| 5 | `tools/plan-context_test.sh:50` recorded as a third fixture string | **Addition** — documentation only, no guard entry needed | §1.9 |
| 6 | `tools/lint.versions` renamed; 6 consumers updated | **Resolution** of PRD open question 2 | §2 |
| 7 | golangci-lint v2.13.1 confirmed by execution | **Resolution** of PRD open question 1 | §1.2 |
| 8 | `reconcile-bump-prs.yml` — no change | **Resolution** of PRD open question 4 | §1.8 |
| 9 | No Renovate `customManagers` | **Resolution** of PRD open question 5 | §7 |
| 10 | New CI job in `pr-validation.yml` | **Resolution** of FR-6.5's conditional | §1.6, §4.2 |

PRD open question 3 (blast radius of new Go 1.27 `go vet` and golangci-lint v2.13.1
diagnostics) **remains open by construction** — it is only answerable by running the
sweep. It is the one item that can materially change the size of the task, and it is
handled as a plan-phase risk (§9), not resolved here.

---

## 9. Risks

| Risk | Likelihood | Handling |
|---|---|---|
| Go 1.27 `go vet` or golangci-lint v2.13.1 reports new findings across 103 modules | Unknown — PRD open question 3 | Measure first: run `tools/lint.sh --check` and `verify.sh --quick` immediately after the directive sweep, before any other work, so the volume is known while the branch is still small. FR-5.3's policy applies: trivial and mechanical findings fixed here; anything needing behavioral judgement gets a commented `.golangci.yml` exclusion naming a `docs/TODO.md` burn-down entry. If the volume is large enough to dominate the task, that is a scope conversation, not a silent expansion. |
| The bake fails on the synthesized-workspace change | Low | §6 removes the literal; AC-15's flagless `verify.sh` bakes every target, and AC-7 bakes `atlas-renders` specifically |
| `go work sync` produces surprising `go.work.sum` churn | Low | §5: expected delta is the stale golangci-lint residue; anything else stops the sweep for explanation rather than being committed |
| Guard passes vacuously (e.g. its `git ls-files` glob matches nothing) | Low, high impact | §3.3: `--selftest` plus the AC-11 recorded live mutation. Additionally the guard asserts a non-zero module count before reporting success — a guard that checked zero files must fail, not pass |
| Renaming `tools/lint.versions` breaks the format-on-write hook silently | Low | The hook's `source ... 2>/dev/null || exit 0` swallows a bad path. Verified during implementation by touching a `.go` file and confirming the hook still formats — a green `verify.sh` would not catch this |
| A worktree or open PR still references `tools/lint.versions` | Low | Repo-wide grep after the rename; only `docs/tasks/task-171-*` historical artifacts should remain, and those are correct as history and must not be rewritten |

---

## 10. Out of scope

Restating `prd.md` §2 with the design-phase additions: no Go 1.27 language feature
or stdlib API adoption; no `toolchain` directives; no behavioral change to any
service; no dependency upgrades beyond `go work sync` churn; no `services/atlas-ui`
Node work; no `tools/cideps` fixture bumps; no Renovate custom manager; no
teaching Dockerfile/bake to read the pin file; no rewriting of historical
`docs/tasks/` artifacts that mention `tools/lint.versions` or older Go versions.
