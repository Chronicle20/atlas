# Review — Task 19: W1 `relocate` subcommand

Range: `21c1f2367..8c568d6b7` (single commit `8c568d6b7`, "chore(task-263): add builder relocation to codemod")
Brief: `.superpowers/sdd/plan/task-19-brief.md`
Report: `.superpowers/sdd/plan/task-19-report.md`

## Scope confirmed

`git diff --stat 21c1f2367..8c568d6b7` touches only:
`codemod/main.go` (+2/-2, wiring), `codemod/relocate.go` (new, 629 lines), `codemod/relocate_test.go`
(new, 470 lines), `progress.md`, `agent-ledger.tsv`, and three unrelated review artifacts
(`review-task-18.md`, `review-task-18b-A/B/C.md`) that were already staged/committed by prior
task work landing on the same branch — no source content in them is Task 19's.

No file under `services/` or `libs/` is touched by this commit — confirmed by the diff stat and
independently by `git diff --name-only 21c1f2367..8c568d6b7`. `classify-file05.tsv` is not in the
diff either (`git diff 21c1f2367..8c568d6b7 -- .../classify-file05.tsv` is empty), so the brief's
"read-only input" contract for that file is honored — it was neither regenerated nor edited to fix
the stale-`builderType` bug; the fix lives entirely in `relocate.go`'s `resolveBuilderType`.

**Environmental note, not a Task 19 finding:** at review time `git status` showed unrelated,
uncommitted modifications under `services/atlas-query-aggregator/.../{character,compartment,
inventory,marriage,party,party_quest,quest,skill,validation}/{model.go,builder.go}` — these are
outside the `21c1f2367..8c568d6b7` range, were not present at the start of this review session, and
are consistent with a concurrent Task 20/21 (real, non-dry `relocate` apply) or gate process
running in the same shared worktree. Not evaluated or touched here; flagging so the controller
knows the tree was not idle during this review.

## 1. The seven brief table cases — present and discriminating

`relocate_test.go`'s `TestRelocate` contains all seven brief cases plus one added by the
implementer (`HAS-BUILDER-GO: methods already in builder.go are not a shared receiver`, added to
lock in bug fix #2 below). All eight subtests pass (`GOWORK=off go test ./... -run TestRelocate -v`,
re-run during this review, all PASS).

- `basic move` (`relocate_test.go:10`) — PASS
- `Clone function moves` (`:59`) — PASS
- `imports recomputed` (`:106`) — genuinely discriminating: the fixture puts `time` and
  `example.com/m/id` both in `model.go`'s original import block, `time` used only by the
  non-moved `Model`, `id.ID` used only by the moved builder. The assertions
  (`:172`-`175`) require `builder.go` to gain `id` and drop `time`, and `model.go` the reverse.
  A copy-the-original-import-block implementation (the thing the brief explicitly forbids) would
  put both imports in both files and fail this test.
- `appends to existing builder.go` (`:178`) — PASS, asserts exactly one `package fixture` clause.
- `source would be empty` (`:290`) — PASS, and additionally asserts the source file is
  byte-identical to the pre-relocate input after a skip (`:313`-`316`), matching the brief's "both
  files byte-identical to input" requirement, not just the ledger reason string.
- `shared receiver` (`:319`) — PASS, models the brief's `resource.go`-holds-a-non-builder-method
  shape exactly.
- `CRLF preserved` (`:363`) — genuinely discriminating: the fixture's `model.go` is CRLF-only,
  and the assertions (`:379`-`387`) require CRLF in both outputs and no lone `\r` (i.e. not just
  "contains one `\r\n` substring by luck of a hardcoded literal" but the whole file consistently
  CRLF). A version of `Relocate` that dropped the `crlf := bytes.Contains(...)` /
  `bytes.ReplaceAll` step (`relocate.go:134`, `186`-`189`) would fail this immediately since
  `format.Source` always emits LF.

All eight subtests were re-run in this review (`go test ./... -run TestRelocate -v`) and pass.

## 2. Skip reasons verbatim against the brief's table

Brief table (six reasons) vs. `relocate.go`'s emission sites, checked verbatim:

| brief reason | emission site | verbatim? |
|---|---|---|
| `receiver shared with non-builder decl` | `relocate.go:155` | yes |
| `source file would become empty` | `relocate.go:163` | yes |
| `reference_data.go excluded` | `relocate.go:99` | yes |
| `entity builder` | `relocate.go:92` | yes |
| `excluded tree` | `relocate.go:94` | yes |
| `multiple builders in file` | `relocate.go:106` | yes |

All six match the brief's table character-for-character. This is required for the ledger's
design-§10 resolvability contract and it holds.

One additional skip reason not in the brief's table, `no %s declarations found in %s`
(`relocate.go:149`, `:160`), was introduced by the implementer as part of bug fix #1 below. The
report is explicit that this is "a genuine data-integrity finding (stale TSV column), not a
planned disposition" and documents it in `progress.md:2736`-2741 rather than silently reusing an
existing reason string or inventing a plausible-looking one from the brief's table — correct
handling of an out-of-brief case.

## 3. The two real-tree bugs

**Bug 2 (shared-receiver guard too strict for HAS-BUILDER-GO) — correct, and tested.**
`sharesReceiverElsewhere` (`relocate.go:215`-235) now exempts both `srcPath` and `builderGoPath`
from the cross-file receiver scan; only a method in a genuine third file trips the guard. The new
`TestRelocate/HAS-BUILDER-GO:...` case (`relocate_test.go:234`-288) reproduces exactly the shape
described (struct in `model.go`, all methods already in `builder.go`) and would fail against the
pre-fix version (which excluded only `srcPath`). PASS, and it is a real regression test for this
bug — good.

**Bug 1 (stale `builderType` column post-W2-rename) — fix is logically sound, but has zero test
coverage, contrary to the report's characterization.** `resolveBuilderType` (`relocate.go:243`-257)
tries the TSV name first, then falls back to `Builder`/`builder` when the TSV name is the exact
pre-rename `ModelBuilder`/`modelBuilder`. This is a reasonable, narrowly-scoped fix. However:

- No fixture in `relocate_test.go` declares a struct named `Builder` (the renamed name) while
  passing `builderType="ModelBuilder"` (the stale TSV name) to `Relocate` — the exact scenario
  `resolveBuilderType`'s fallback branch exists to handle. Every one of the eight `TestRelocate`
  subtests passes a `builderType` argument that is also the struct's actual declared name
  (`ModelBuilder`), so the fallback branch (`relocate.go:246`-249) is never exercised by the test
  suite (confirmed by inspection — `grep -n '"Builder"' relocate_test.go` returns only
  `mustNotContain(..., "Builder")` assertions, no fixture struct named bare `Builder`).
- The `*skipError("no %s declarations found in %s")` path this same fix introduces
  (`relocate.go:149`, `:160`) is likewise never triggered by any test.
- The report's own words: "Fixed with `resolveBuilderType`... **and made "no declarations found" a
  `*skipError`**... Two corrections found only by running the dry run (both now fixed, **both
  covered by tests where practical**)." Only bug 2 has a dedicated regression test; bug 1's fix —
  the one the report itself calls "stale for most rows" (i.e. the common case in the real tree) —
  has none. "Covered by tests where practical" is not qualified per-bug in the report, and reads as
  covering both.

This is not a correctness defect I can point to (the fallback's two-name mapping is simple and its
correctness is corroborated empirically by the 64-applied dry run, most of which are exactly this
already-renamed shape per the report), but it is a real gap between the report's claim and the
diff's content — flagged as non-blocking since the tool is task-local throwaway tooling whose
behavior against the real tree was independently exercised, but it should not have been reported
as tested.

## 4. The 64/9 vs. design §5's ≈65/≈7 gap

Design `docs/tasks/task-263-backend-guideline-conformance/design.md:456` states "W1 | ~65 packages
| ~7 | reference_data.go split (FR-11), skip conditions", and §2.5 (`design.md:141`-147) states
FILE-05 covers exactly 72 distinct packages. 65+7=72: **design's own W1 estimate is already computed
at package granularity**, not raw declaration/row granularity (§2.5 separately reports 100
declarations across those 72 packages).

The report's and `progress.md:2760`-2763's explanation for the 64/9 (=73 groups) vs. 65/7 (=72
groups) gap is: "the small gap is the file-grouping collapsing multi-builder-row files (e.g.
reference_data.go's 7 rows) into one ledger entry each, changing the unit being counted from 'row'
to '(pkgDir, file)'." This reasoning does not hold up against the actual arithmetic:

- If the true "row" count (100 declarations) were being compared against design's ≈72
  package-level estimate, collapsing multi-row files would reduce the *group* count relative to a
  row count — but the report is not comparing against a row count; it is comparing against
  design's own package-level 72, which is already row-collapsed. A collapsing effect predicts the
  actual group total to land *at or below* 72, not above it.
- The actual total is 73 (64+9), one *more* than design's 72, not fewer. The report's explanation
  gives no account of why the total went up rather than down.
- A more evidence-grounded candidate explanation is visible in the report's own skip breakdown:
  `multiple builders in file` fires for three files (`conversation/model.go`'s 20,
  `conversation/quest/model.go`'s 2, `pets/data/pet/model.go`'s 2), but design's evidence base
  (`design.md` §2.3, which discusses the DOM-01 rename collision set) only calls out
  `conversation/model.go`'s 20 explicitly. `conversation/quest/model.go` and
  `pets/data/pet/model.go` each having a second builder is new information the dry run surfaced,
  not something design's ≈7 estimate necessarily priced in — that is a materially different, and
  more defensible, explanation than "grouping unit" for why hand-work landed at 9 rather than ≈7.
  The report does not raise this alternative or reconcile the sign of the discrepancy.

The magnitude (64 vs ≈65, 9 vs ≈7) is small and not itself a defect — design's own "≈" hedges
against exactly this kind of drift, and it is recorded transparently rather than silently forced to
match. But the specific mechanism offered to explain it is a rationalization rather than a
demonstrated cause, and Task 20/21 will inherit this ledger's exact counts as their starting point,
so the imprecision is worth a second look before those tasks proceed on the assumption that "grouping
unit" is the whole story.

## Other checks

- `go build ./...`, `go vet ./...`, `go test ./... -v` all re-run clean in this review
  (`GOWORK=off`, from `codemod/`).
- `main.go` wiring (`git diff -- main.go`) is a minimal, correct two-line addition
  (`"relocate": runRelocate` plus a stale doc-comment removal).
- `writeRelocatedFiles` (`relocate.go:616`-629) is only reached when `!dryRun`
  (`relocate.go:594`-598); the dry run the report ran (`-dry-run`) exercises the full decision path
  and ledger-writing without ever calling it — consistent with the report's "`git status` clean
  afterward" claim, which this review did not independently re-run against the real tree (out of
  scope: doing so would write to `services/`/`libs/`, and a concurrent process was already
  mutating those trees at review time — see the environmental note above).

## Not evaluable

- The real-tree dry run's exact 64/9 output was not independently reproduced in this review — the
  worktree had concurrent, unrelated modifications under `services/atlas-query-aggregator/` at
  review time (see environmental note), making a clean re-run of `relocate -dry-run` against the
  live tree unsafe to attempt without risking cross-contamination with that other process's
  in-flight state. The count is taken on the report's and `progress.md`'s word, corroborated only
  by their mutual consistency and by the arithmetic check in finding 4 above.

## Verdict rationale

Both `TestRelocate`'s seven-plus-one cases and the six skip-reason strings are correct and
discriminating — the core deliverable does what the brief asked, cleanly, and the tooling-only
constraint (no `services/`/`libs/` file touched, `classify-file05.tsv` untouched) is honored. The
two findings above are about the report's own claims outrunning the diff (untested fix,
underexplained gap) rather than about `Relocate`'s behavior being wrong — hence
`APPROVED_WITH_FINDINGS` rather than `CHANGES_REQUIRED`.
