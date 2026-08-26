# task-262 — execution status

Durable record of where Phase 4 execution stopped and why. The full per-unit
ledger lives in the git-ignored SDD workspace (`.superpowers/sdd/plan/progress.md`);
this file carries the part a fresh session needs.

## Where the branch stands

All implementation is complete.

**Landed and reviewed:** plan Tasks 1, 2, 3, 4, 8, 9, 10, and 14. Tasks 6, 7,
11, 12, 13, and 15 are **WITHDRAWN in place** by the mid-execution re-scope
(commit `0b3c8b21c`, "oracle withdrawal" — Task 5 found the supplied
HaRepacker reference dump was never exported from the supplied `$WZ_ARCHIVE`;
all 21 apparent divergences are `INPUT-MISMATCH`, not parser defects). The
added Tasks R1 (the re-scope itself), R2 (whole-archive size-accounting
self-check), R3 (post-review fixups), and R4 (DOM-20 disposition) all landed
and were reviewed.

## Pre-PR review is complete

- **Plan-adherence audit**, sharded: shard 1-10 is **APPROVED**
  (`docs/tasks/task-262-wz-property-reader-divergence/audit-plan-1-10.md`);
  shard 11-15+R is **APPROVED_WITH_FINDINGS, 0 blocking**
  (`docs/tasks/task-262-wz-property-reader-divergence/audit-plan-11-R.md`).
- **`backend-guidelines-reviewer`** returned **CHANGES_REQUIRED** with 10
  DOM-20 findings (flat `func Test...` bodies where a table-driven shape was
  preferred). All 10 were answered by Task R4 — see
  `docs/tasks/task-262-wz-property-reader-divergence/dom20-dispositions.md`
  for the per-finding disposition (5 converted, 5 kept flat with a stated
  reason) and the basis for treating DOM-20 as a preference rather than a
  MUST for the kept-flat cases (repo precedent:
  `docs/tasks/task-085-packet-audit-coverage-matrix/audit-backend.md:39`).

## The branch-end gate: RESOLVED and GREEN

Earlier sessions recorded an open blocker here — "flagless `tools/verify.sh`
cannot exit 0 because golangci-lint is built against go1.26 while the repo
toolchain is go1.27.0." That diagnosis was directionally right but incomplete,
and **it is now closed**. The record:

- Rebuilding golangci-lint v2.12.2 from source under go1.27 removed the stdlib
  typecheck errors but did **not** fix the guard: the gate still failed only
  the lint & format guard, with 433 `goanalysis` panics.
- The real cause was `honnef.co/go/tools@v0.7.0` (vendored in golangci-lint
  v2.12.2), whose IR builder cannot parse go1.27 stdlib source —
  `buildir: unexpected expr: *ast.KeyValueExpr` while building package `poll`.
  Every dependent analyzer (`fact_purity`, `nilness`, `typedness`, `SA5012`)
  then panicked on a nil `*buildir.IR`. Reproduced on `libs/atlas-model`, a
  module this branch never touched — so it was never branch-caused.
- **Main fixed it.** `c9e533e6c` (task-261, #1497) migrated the monorepo to
  Go 1.27.0 and bumped `GOLANGCI_LINT_VERSION` to **v2.13.1** (renaming
  `tools/lint.versions` to `tools/toolchain.versions`). After rebasing onto
  `origin/main`, no local workaround is needed. Do **not** set `GOTOOLCHAIN`.

Once the linter actually ran, it reported 8 real `errcheck` findings in
`libs/atlas-wz/wzdiff` that the panic had been masking. Task R6 (`6c6e81622`)
cleared them, plus 6 more in `wzdiff/selfcheck.go` — a file this branch added
after the gate base, which the rev-gated linter flags too. Reviewed
**APPROVED_WITH_FINDINGS, 0 blocking** (`reviews/review-R6.md`).

**Result:** flagless `tools/verify.sh` on the rebased tree exits **0** —
"All checks passed", 91 modules, zero failing checks. CLAUDE.md's
"done means verified" bar is met.

### One flaky failure, not this branch's

An earlier gate hit a data race in `atlas-channel`:
`TestApplyLoop_AddBodyReceivesAContextCanceledByDrain` —
`listener.(*Registry).Drain()` (`registry.go:273`) reads what
`(*Registry).Add()` (`registry.go:158`) writes. It did not reproduce on the
next two runs, so it is flaky rather than deterministic. This branch touches
**zero** `atlas-channel` files (`registry.go` last changed on main in
`848eee43f`, #1441). It needs its own task against main.

## Housekeeping (resolved)

- `go.work.sum`: restored with `git checkout --`; clean.
- `docs/tasks/.../reviews/`: committed (`8975ec275`, `05aa00483`).
- The shared `~/.cache/go-build` corruption did not reproduce; the gate ran
  green with the default cache. No `go clean -cache` was needed.

## Resuming

Nothing left. The branch is rebased on `origin/main`, verified green, and
pushed for PR.
