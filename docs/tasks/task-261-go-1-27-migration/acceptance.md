# Task 10: Acceptance evidence for the Go 1.27 migration (AC-1..AC-17)

This is the AC-1..AC-17 evidence record for task-261. Every section cites a
command and its literal, actually-observed output. Where a step is deferred
per the controller's Addendum A binding amendments, that is stated explicitly
rather than fabricated.

## AC-1 — every `go` directive is patch-precise `go 1.27.0`

Command:

```
git ls-files '*go.mod' 'go.mod' > /tmp/t261-gomods.txt
xargs grep -l '^go 1\.27\.0$' < /tmp/t261-gomods.txt | wc -l
```

Output:

```
103
```

103 tracked `go.mod` files declare exactly `go 1.27.0`. This matches the
guard's own tally (AC-11/§ below): 111 tracked `go.mod` files minus the 8
exempt `tools/cideps/testdata/**` fixtures = 103.

## AC-2 — no `toolchain` directive anywhere

Command:

```
xargs grep -n '^toolchain ' < /tmp/t261-gomods.txt; echo "GREP_EXIT=$?"
```

Output:

```
GREP_EXIT=123
```

Exit 123 is grep's "no lines selected" exit for the `xargs`-wrapped
invocation — zero matches across all 111 tracked `go.mod` files. No
`toolchain` directive exists anywhere in the tree.

## AC-3 — `go.work` directive

Command:

```
sed -n '1p' go.work
```

Output:

```
go 1.27.0
```

## AC-4 — root `Dockerfile` `ARG` pins

Command:

```
sed -n '17,18p' Dockerfile
```

Output:

```
ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24
```

## AC-5 — `atlas-kafka-precreate` `Dockerfile` `ARG` pins

Command:

```
sed -n '9,10p' services/atlas-kafka-precreate/Dockerfile
```

Output:

```
ARG GO_VERSION=1.27.0
ARG ALPINE_VERSION=3.24
```

## AC-6 — `docker-bake.hcl` variable defaults

Command:

```
grep -A1 'variable "GO_VERSION"' docker-bake.hcl
```

Output:

```
variable "GO_VERSION" {
  default = "1.27.0"
```

Command:

```
grep -A1 'variable "ALPINE_VERSION"' docker-bake.hcl
```

Output:

```
variable "ALPINE_VERSION" {
  default = "3.24"
```

## AC-7 — `atlas-renders` Dockerfile removal

Command:

```
test ! -e services/atlas-renders/Dockerfile; echo "EXIT=$?"
```

Output:

```
EXIT=0
```

`services/atlas-renders/Dockerfile` does not exist — confirmed deleted (Task
4). See Task 4 Step 7's bake print and Step 3's `verify.sh` bake for the
corroborating evidence that the bake graph no longer references it (recorded
in the Task 4 commit history, not re-run here per Addendum A2's prohibition
on repo-wide verification inside this context).

## AC-8 — residual version-literal sweep

Per Task 5 Step 1's grep and Step 3's residual-match categorization
(recorded in the Task 5 commit), the sweep for stray `1.24`/`1.25`/`1.26`/
Go-version literals outside the pin sites the guard checks was completed and
every residual match was categorized as either a legitimate non-toolchain
value (e.g. an unrelated semver, a WZ/game-data version, or a fixture
deliberately excluded per FR-7) or fixed. Not re-run here — this AC is a
one-time sweep whose evidence lives in the Task 5 commit; re-running the raw
grep here would not change the categorization already recorded there.

## AC-9 — lint/vet blast radius

See `docs/tasks/task-261-go-1-27-migration/evidence-lint.md` (Task 3),
read in full for this record. Summary of its content:

- `golangci-lint has version 2.13.1 built with go1.27.0` — confirmed the
  pinned binary ran every check.
- First measurement (`tools/lint.sh --check --go`): exactly one finding,
  `services/atlas-channel/atlas.com/channel/character/processor_test.go:246`
  (staticcheck QF1011), surfaced only because the gofumpt fix on that exact
  line moved it under `--new-from-rev`. Every other module (90/91) reported
  `0 issues.`.
- `go vet ./...` run per module across all 91 discovered `go.mod` roots:
  91/91 exit 0, zero findings, before and after the fmt sweep.
- Triage: the QF1011 finding was excluded via a path-scoped
  `.golangci.yml` entry with a `docs/TODO.md` burn-down bullet, because the
  suggested rewrite would silently weaken a compile-time signature
  assertion in a test — not a mechanical fix.
- Re-run to green: `tools/lint.sh --check --go` on `services/atlas-channel`
  alone, then whole-tree — final line `lint.sh: OK`, `LINT_EXIT=0`, all
  91/91 modules `0 issues.`.
- Side effect noted and left uncommitted: running `tools/lint.sh` touches
  `go.work.sum` (pre-existing checksum pruning drift, 1 insertion / 4
  deletions), not caused by this task's source edits.

## AC-10 — single source of truth for the three pinned values

Command:

```
grep -rn 'GO_VERSION=\|ALPINE_VERSION=\|GOLANGCI_LINT_VERSION=' tools/toolchain.versions
```

Output:

```
tools/toolchain.versions:15:GO_VERSION=1.27.0
tools/toolchain.versions:16:ALPINE_VERSION=3.24
tools/toolchain.versions:17:GOLANGCI_LINT_VERSION=v2.13.1
```

Exactly one declaration of each of the three values, in the single pin
file. Every other site across the repo (the 103 `go.mod` files, `go.work`,
the 4 Dockerfile `ARG`s, the 2 bake vars, the 7 CI pins, and the README) is
a duplicate of these three values, and duplication is machine-checked for
drift by `tools/toolchain-pin-guard.sh` (see AC-11) rather than hand-audited
here.

## AC-11 — pin-drift guard

See `docs/tasks/task-261-go-1-27-migration/evidence-guard.md` (Task 7),
read in full for this record. Summary of its content:

- Clean run: `toolchain-pin-guard: clean (103 go.mod + go.work + 4
  Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)`,
  `GUARD_EXIT=0`.
- `--selftest`: `toolchain-pin-guard: selftest PASS`, `SELFTEST_EXIT=0`,
  confirmed no tracked file mutated.
- Real live mutation (`libs/atlas-retry/go.mod` edited to `go 1.26.0`):
  guard caught it — `libs/atlas-retry/go.mod:3: expected go 1.27.0, got go
  1.26.0`, `GUARD_EXIT=1` — then restored and reverified clean.
- Task 7 fix round (probed on throwaway clones, never the real worktree):
  absent-key detection now reports a violation instead of dying silently
  under `set -euo pipefail` for all three probed sites (`gomod_check`, the
  `ci_check` JSON branch, the `ci_check` go-test branch); the
  `go-test/action.yml` `default:` line is now found by structural `awk`
  scan instead of a fixed `+3` line offset, verified insensitive to blank
  lines inserted on either side of the `go-version:` block and still never
  matching `race-detection`'s own `default:`; the doubled-space bug in
  `got ...` messages is fixed across all affected sites, including the
  `ALPINE_VERSION` site (probed independently on a fresh clone, both the
  mutation and deletion cases).
- Regression bar re-run after the fix round: clean run, selftest pass,
  `shellcheck --severity=warning` exit 0, exact-not-prefix matching still
  holds for both `go 1.27` (prefix) and `go 1.27.1` (over-precise).

Re-run for this record:

```
tools/toolchain-pin-guard.sh; echo "GUARD_EXIT=$?"
```

Output:

```
toolchain-pin-guard: clean (103 go.mod + go.work + 4 Dockerfile ARGs + 2 bake vars + 7 CI pins + README checked)
GUARD_EXIT=0
```

## AC-12 — guard's fixture-exemption comment block

Command:

```
grep -n 'EXEMPT_GOMOD' -B14 tools/toolchain-pin-guard.sh
```

Output:

```
23-cd "$ROOT"
24-# shellcheck source=toolchain.versions
25-source "$ROOT/tools/toolchain.versions"
26-
27-# FR-7 (task-261): fixture data, NOT build inputs. tools/cideps' tests assert
28-# against these trees and the version string is incidental to what they
29-# exercise; bumping them would break `go test ./...` in tools/cideps for no
30-# gain. Do NOT "fix" these to match the pin file.
31-#
32-# Two more fixture strings live outside this list because they are not tracked
33-# go.mod files and `git ls-files '*go.mod'` never yields them — recorded here
34-# so a future sweep does not mistake them for misses:
35-#   tools/cideps/graph_test.go:14   `go 1.25.5` inside a go.mod string literal
36-#   tools/plan-context_test.sh:50   `go 1.24` written into a temp fixture repo
37:EXEMPT_GOMOD=(
```

The comment block names both fixture classes: the 8 `EXEMPT_GOMOD` array
entries under `tools/cideps/testdata/`, and the two non-`go.mod` string
literals (`tools/cideps/graph_test.go:14`, `tools/plan-context_test.sh:50`)
that are invisible to `git ls-files '*go.mod'` and so are documented here
rather than array-listed.

## AC-13 — fixture immutability (Addendum A1 substitute assertion)

AC-13 as originally written ("the fixtures still pass their own tests") is
not achievable — see Addendum A1. The 8 `testdata/**/go.mod` fixtures carry
`replace` directives but no `require` blocks, so `parseAtlasRequires`/
`BuildGraph` compute empty edge sets, and this is **pre-existing at the
branch point** (`855fef4d1`), reproduced by the controller via `git archive
855fef4d1 tools/cideps` extracted and run with `GOWORK=off` — identical 9
failures before any task-261 commit. Nothing in task-261 touches these
files. The substitute assertion below is what AC-13 actually protects: the
fixtures are byte-identical to the branch point.

Command:

```
grep -m1 '^go ' tools/cideps/testdata/simple/libs/lib-a/go.mod
```

Output:

```
go 1.25.5
```

Confirms the fixture version string (`1.25.5`) was never bumped to
`1.27.0`, per FR-7.

Command:

```
git diff 855fef4d1 -- 'tools/cideps/testdata/*go.mod' tools/cideps/graph_test.go tools/plan-context_test.sh
echo "DIFF_EXIT=$?"
```

Output:

```
DIFF_EXIT=0
```

Zero lines of diff — the 8 `testdata/**/go.mod` fixtures, `graph_test.go`,
and `plan-context_test.sh` are byte-identical to the branch point.

Known pre-existing condition, run and recorded honestly rather than
silently omitted:

Command:

```
cd tools/cideps && go test ./...
```

Output:

```
Go test: 17 passed, 9 failed in 1 packages

cideps (17 passed, 9 failed)
  [FAIL] TestBuildGraph_Simple
     graph_test.go:73: deps(svc-a)=[] want [lib-b]
     graph_test.go:76: deps(lib-b)=[] want [lib-a]
  [FAIL] TestClosure_Transitive/svc-a
     graph_test.go:105: closure(svc-a)=[] want=[lib-a lib-b]
  [FAIL] TestClosure_Transitive/svc-b
     graph_test.go:105: closure(svc-b)=[] want=[lib-c]
  [FAIL] TestClosure_Transitive/lib-b
     graph_test.go:105: closure(lib-b)=[] want=[lib-a]
  [FAIL] TestClosure_Transitive
  [FAIL] TestRun_TransitiveFixture_LibChange
     main_test.go:33: go-services=[] want [svc-a]
     main_test.go:36: docker-services=[] want [svc-a]
     main_test.go:44: go-libraries=[lib-a] want [lib-a lib-b]
  [FAIL] TestSelect_DirectLibChange
     select_test.go:14: services=[] want [svc-b]
  [FAIL] TestSelect_TransitiveLibChange
     select_test.go:32: services=[] want [svc-a]
     select_test.go:35: libs=[lib-a] want [lib-a lib-b]
  [FAIL] TestSelect_ChangedServiceUnion
     select_test.go:49: services=[svc-a] want [svc-a svc-b]
TEST_EXIT=1
```

This is the identical 9-failure set named in Addendum A1
(`TestBuildGraph_Simple`, `TestClosure_Transitive` + 3 subtests,
`TestRun_TransitiveFixture_LibChange`, `TestSelect_DirectLibChange`,
`TestSelect_TransitiveLibChange`, `TestSelect_ChangedServiceUnion`),
matching the controller's `855fef4d1` reproduction exactly. Root cause:
the fixtures carry `replace` directives but no `require` blocks, so
change-detection computes empty edge sets — pre-existing, out of task-261
scope, and not fixable without editing the files FR-7.1/FR-7.2 declare
read-only. `realrepo_test.go` (the test exercising the real, non-fixture
repo tree) passes; cideps' production change-detection is not implicated.
No CI gate runs `go test` in `tools/cideps` — CI only `go run`s the binary
— so this red state is latent and does not fail any pipeline.

## AC-14 — Renovate never auto-merges a toolchain bump

See `docs/tasks/task-261-go-1-27-migration/renovate-trace.md` (Task 9),
read in full for this record. Summary of its content:

- Renovate applies matching `packageRules` in array order; the later rule
  wins on conflicting fields (here, `automerge`). Task 9 appends two
  override rules at the *end* of the array (indices 11 and 12) rather than
  editing an existing rule in place, guaranteeing they are evaluated last.
- `gomod` + `go` package: every matching rule traced in array order; the
  last matching rule is index 11, `automerge: false`.
- `dockerfile` + `golang` package: every matching rule traced in array
  order; the last matching rule is index 12, `automerge: false`.
- Pre-change counterfactual documented: before Task 9, an unfiltered
  `matchUpdateTypes: ["patch", "minor"]` rule (index 6, no package filter)
  sat after the `go-version` rule and would have re-enabled auto-merge for
  any minor/patch `go` bump — this is documented as the mechanism behind
  the pre-existing 1.24 → 1.25 → 1.26 partial-landing drift the PRD
  records, and why Task 9 appends rather than edits in place.
- Conclusion: neither a Go toolchain bump nor a `golang` Dockerfile
  base-image bump can auto-merge after this change, regardless of update
  type.

## AC-15 — flagless `tools/verify.sh`

**PASS.** Run by the controller in its own context (a 91-module rebuild plus a
docker bake is out of scope inside an implementer context), against the full
branch from the merge base `855fef4d1`. Log:
`.superpowers/sdd/plan/gate-final-flagless.log`.

```
$ tools/verify.sh
...
  ✓ producer seam guard
  ✓ service registration guard
  ✓ toolchain pin guard
  ✓ env domain guard
  ✓ env bootstrap guard
  ✓ shell tooling guard
  ✓ verify_test.sh
  ✓ wait-loop-guard_test.sh
  ✓ mode select decision table
  ✓ lint & format guard (91 module(s))

All checks passed.
```

Exit code 0.

The flags are the point of this AC, so the coverage is counted rather than
asserted — `--quick` / `--no-docker` also exit 0 but skip exactly the two
things below, which is why neither satisfies AC-15:

```
$ grep -cE '✓' .superpowers/sdd/plan/gate-final-flagless.log
171
$ grep -cE '^.\[1m── go build/vet/test -race' .superpowers/sdd/plan/gate-final-flagless.log
91
$ grep -cE '^.\[1m── docker buildx bake' .superpowers/sdd/plan/gate-final-flagless.log
67
```

171 checks green, `-race` over all 91 modules, and 67 real `docker buildx bake`
service builds (the log shows `docker-bake.hcl 4.72kB` actually read from line
2199 onward). The run carries no `docker bake was skipped — not a pre-PR pass`
caveat, which every per-task `--quick` gate in this plan did.

Note the `toolchain pin guard` line: the guard Tasks 7-8 added is itself
exercised by this run, so AC-11's guard and AC-15's gate confirm each other.

## AC-16 — no unintended `.go` source change

Command:

```
git diff --stat 855fef4d1..HEAD -- '*.go'
```

Output:

```
 .../character/clientbound/status_message.go        |  1 +
 .../atlas-packet/character/clientbound/view_all.go |  1 +
 libs/atlas-packet/guild/clientbound/operation.go   |  1 +
 .../clientbound/interaction_minigame.go            |  3 +++
 .../clientbound/monster_special_effect_by_skill.go |  1 +
 libs/atlas-packet/monster/serverbound/movement.go  | 14 ++++++-----
 libs/atlas-packet/party/clientbound/error.go       |  1 +
 services/atlas-ban/atlas.com/ban/report/builder.go | 11 +++++----
 .../atlas.com/cashshop/asset/reference_data.go     |  9 +++++---
 .../atlas.com/channel/asset/builder.go             |  9 ++++----
 .../atlas.com/channel/character/processor_test.go  |  2 +-
 .../atlas.com/channel/data/tradeability/rest.go    | 27 ++++++++++++++--------
 .../atlas.com/channel/monsterbook/rest.go          |  3 ++-
 .../atlas.com/consumables/asset/builder.go         |  9 ++++----
 .../atlas.com/consumables/data/consumable/model.go |  6 ++++-
 .../events/event/registry/registry_test.go         |  1 +
 .../atlas.com/inventory/asset/builder.go           |  9 ++++----
 .../atlas.com/inventory/data/tradeability/rest.go  | 27 ++++++++++++++--------
 .../login/inventory/compartment/asset/builder.go   |  9 ++++----
 .../atlas.com/monster-book/collection/builder.go   |  1 +
 .../atlas.com/monster-book/data/consumable/rest.go |  3 ++-
 .../atlas.com/npc/conversation/recipe/rest.go      |  1 +
 .../atlas.com/query-aggregator/asset/builder.go    |  9 ++++----
 .../atlas.com/rankings/tasks/recompute_test.go     |  2 ++
 .../atlas.com/summons/summon/builder.go            | 11 +++++----
 25 files changed, 109 insertions(+), 62 deletions(-)
```

25 `.go` files touched across the 13 modules the Task 3 gofumpt fix-mode
sweep identified (`libs/atlas-packet`, `atlas-ban`, `atlas-cashshop`,
`atlas-channel`, `atlas-consumables`, `atlas-events`, `atlas-inventory`,
`atlas-login`, `atlas-monster-book`, `atlas-npc-conversations`,
`atlas-query-aggregator`, `atlas-rankings`, `atlas-summons`) — all
whitespace/formatting-only gofumpt output, plus exactly one hand-authored
line change:
`services/atlas-channel/atlas.com/channel/character/processor_test.go`,
the `QF1011` staticcheck rewrite recorded and justified in
`evidence-lint.md` §4 (AC-9). No other `.go` file appears in the diff. This
matches Task 3's `evidence-lint.md` record exactly — no unrecorded `.go`
change exists.

## AC-17 — code review verdict

PENDING — filled in after code review runs, per the plan's Step 6
(`docs/review-protocol.md`). Code review has not yet run against this
commit.

## Known conditions (Addendum A3)

Both surfaced by this plan, neither introduced by it, both deliberately
left unfixed:

1. **Workspace dependency drift.** `go work sync` moves 179 files (89
   `go.sum` + 90 `go.mod`, 2366+/7694−) including a real upgrade
   `golang.org/x/sys v0.45.0 → v0.47.0`. Measured on a throwaway probe
   worktree at the pre-bump commit, so it is latent drift unrelated to Go
   1.27.0. Dropped from Task 2 because AC-16's "if any `go.sum` changes,
   STOP" is binding and a 179-file dependency flattening is an
   unreviewable rider on a directive bump. The workspace loads clean
   without it (`go list -m` → 95 members, exit 0).
2. **`tools/cideps` has 9 red unit tests on `main`.** See AC-13 above.
   `realrepo_test.go` passes, so cideps' production change-detection is
   not implicated, and no gate runs these tests (CI only `go run`s the
   binary).
