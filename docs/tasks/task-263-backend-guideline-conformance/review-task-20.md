# Review — Task 20: W1 `relocate` apply, first half

Commit range: `8c568d6b7..c9582f0ec`
Commits reviewed: `a433d6513` (codemod fix), `2301a7f71` (query-aggregator),
`e25c5b5f3` (login), `af27afccb` (pets), `7f58dc448` (maps), `0a573f178`
(consumables), `767c65d1a` (npc-shops), `c9582f0ec` (script-core).

Brief: `.superpowers/sdd/plan/task-20-brief.md`. Report:
`.superpowers/sdd/plan/task-20-report.md`.

## Verdict

APPROVED_WITH_FINDINGS

## 1. Step 2 asymmetry check — run independently, not taken from the report

For each of the seven service commits, ran the brief's own command
(`git diff -M <parent> <commit> | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE
<import/package/blank filters> | sed 's/^[+-]//' | sort | uniq -c | awk
'$1%2!=0'`) directly against the commit range, not against a working-tree
staged diff.

Result for all seven: **the only unpaired lines are the appended rows of
`ledger-relocate-a.tsv`** (an additive-only file — each row necessarily
appears once as `+` with no `-` counterpart, which is expected and not an
asymmetry in moved code). Confirmed for:

- `2301a7f71` (query-aggregator) — 10 unpaired lines, all `ledger-relocate-a.tsv` rows.
- `e25c5b5f3` (login) — 6, all ledger rows.
- `af27afccb` (pets) — 4, all ledger rows (incl. the SKIPPED row).
- `7f58dc448` (maps) — 4 ledger rows + blank-line noise (my filter, unlike the
  report's, didn't also strip bare `-` blank lines; re-ran with `-$` excluded
  too and it drops to the same 4 ledger rows) + one `-\ttenant
  "github.com/Chronicle20/atlas/libs/atlas-tenant"` false positive from an
  incomplete grep filter (aliased import on the removal side wasn't excluded).
  Verified by hand: the alias is dropped because the package's own declared
  name (`package tenant` in `libs/atlas-tenant/tenant.go`, `package _map` in
  `libs/atlas-constants/map/model.go`) already matches the old explicit
  alias, so the import is semantically identical unaliased. Not a defect.
- `0a573f178` (consumables) — 4, all ledger rows.
- `767c65d1a` (npc-shops) — 3, all ledger rows.
- `c9582f0ec` (script-core) — 3, all ledger rows.

No signature, field name, comment, or body text differs between the `-` and
`+` sides in any of the seven commits. Step 2 passes on all seven, verified
directly, not via the report's word.

## 2. Codemod fix (`a433d6513`) precedes all seven, and no service commit used pre-fix output

`git log --oneline --reverse 8c568d6b7..c9582f0ec` confirms `a433d6513` is the
first commit in the range; all seven service commits follow it. `git show
a433d6513 --stat` touches only
`docs/tasks/task-263-backend-guideline-conformance/codemod/{relocate.go,relocate_test.go}`
— no service file — so there is no ordering ambiguity to resolve.

Checked the two defects' actual blast radius in the committed output, not
just the commit order:

- **Multi-group overwrite** (`atlas-query-aggregator/validation`): both
  `ValidationContextBuilder` (from `context.go`) and `ConditionBuilder` (from
  `model.go`) are present in the final `validation/builder.go`, including
  `NewValidationContextBuilderWithLogger` — the exact function the report says
  disappeared on the pre-fix run
  (`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/builder.go:69`).
  Confirmed by `grep -n func` on the file: both builders' full method sets are
  present, nothing missing.
- **Comment loss** (`atlas-query-aggregator/character`): the two comments the
  report says were lost live on `Sp()`/`RemainingSp()`
  (`character/model.go:180,195`), which are getters, not builder methods —
  they never moved to `builder.go` in the committed diff, so this file was
  never at risk of losing them in the actual applied output. The moved
  `Builder`/`Clone` declarations in `character/builder.go` carry no body
  comments in source, so there was nothing to lose or preserve there either;
  the asymmetry check in §1 independently confirms no comment or body text
  went missing.

Built `atlas-query-aggregator`, `atlas-pets`, and `libs/atlas-script-core`
directly (`go build ./...`) as a spot check; all three build clean at HEAD.

## 3. CRLF check — not spot-checked by the implementer, done here

Enumerated all `model.go`/`builder.go`/`context.go` files touched anywhere in
`8c568d6b7..c9582f0ec` (65 paths). For every path that already existed at the
range's base commit (`8c568d6b7`) — i.e. every `model.go`/`context.go`, since
all `builder.go` files are new — fetched `git show 8c568d6b7:<path>` and
counted `\r\n` occurrences. **Zero of the 65 files had any `\r\n` in the base
revision.** There is no CRLF file among the seven services touched in this
task, so there is no CRLF round-trip to break and nothing to regress. This
confirms (rather than merely trusts) the implementer's own uncertainty note in
the report — the risk they flagged does not exist in this specific range.

## 4. Ledger (`ledger-relocate-a.tsv`) — 33 APPLIED / 1 SKIPPED

- 34 rows total confirmed by direct file read: 33 `APPLIED`, 1 `SKIPPED`
  (`services/atlas-pets/atlas.com/pets/data/pet/model.go`, reason "multiple
  builders in file").
- The reason string traces to `staticSkipReason` in
  `docs/tasks/task-263-backend-guideline-conformance/codemod/relocate.go:90-109`,
  a pre-existing function **not touched by `a433d6513`** (confirmed: `git show
  a433d6513 -- .../relocate.go` has no `staticSkipReason` hunk) — so this skip
  category predates Task 20 and was not invented for this task. It is not
  verbatim one of the three prose bullets in design.md:400-403 ("a builder
  method whose receiver is also used by a non-builder declaration," "a builder
  declared inside a file whose remaining content would become empty,"
  `reference_data.go`), but it is a `distinct builder-type count > 1` static
  check that predates this branch's work and is recorded, not silently
  dropped — satisfying design §10's "every SKIPPED package must appear either
  in a hand-work commit or in exemptions.md" by being on the ledger for
  Task 25's hand-work pass, per the report. Non-blocking: the design doc's
  §4.2 skip-condition prose could be tightened to name this fourth condition
  explicitly, but that's a documentation gap in an earlier task's artifact,
  not a defect in this range.
- APPLIED count reconciles against the brief's per-service declaration counts:
  query-aggregator 10 (character, compartment, inventory, marriage, party,
  party_quest, quest, skill, validation/context, validation/model — the last
  two are the two builders sharing one output file), login 6, pets 3 applied
  + 2 declarations folded into the single SKIPPED row (`ModelBuilder` +
  `SkillModelBuilder`) = 5, maps 4, consumables 4, npc-shops 3, script-core 3.
  10+6+5+4+4+3+3 = 35 declarations; 33 single-declaration rows + 1 two-declaration
  skipped row = 34 ledger rows, 35 declarations. Matches the brief's stated
  per-service counts exactly.

## 5. `classify-file05.tsv` untouched

`git diff --stat 8c568d6b7 c9582f0ec -- .../classify-file05.tsv` is empty.
Confirmed read-only, as declared.

## Task 19's open FILE-05 count thread (64+9=73 vs. design's 72)

Nothing in this range's ledger or diffs bears on that reconciliation — Task 20
operates on a fixed 34-row subset of the already-classified population and
does not re-run or alter the dry-run classification counts. `progress.md:2789`
and `:2828` already show this is explicitly assigned to Task 21-C for
resolution; no new evidence here changes that assignment.

## Not evaluable

- `go test ./...` was not re-run by this review (only `go build ./...` spot
  checks on 3 of 7 modules) — a concurrent `tools/verify.sh` gate is running
  against this same worktree per instructions, and re-running full test
  suites risked interfering with or duplicating that gate. The report's own
  per-module PASS table (Step 3) was not independently re-executed for test
  behavior, only for build correctness on 3/7 modules.
- The four remaining modules (login, maps, consumables, npc-shops) were not
  independently built or tested in this review; the asymmetry check (§1) and
  the ledger/count reconciliation (§4) cover them, but a `go build`/`go
  vet`/`go test` re-run was only done for query-aggregator, pets, and
  script-core.

## Findings summary

No blocking defects found. All seven commits are pure relocations by the
brief's own asymmetry test, run independently against the actual commit
range. The two codemod defects are fixed before any service commit and their
specific failure signatures (missing
`NewValidationContextBuilderWithLogger`, dropped `Sp()`/`RemainingSp()`
comments) are absent from / not applicable to the committed output. CRLF risk
does not materialize in this range (no CRLF files touched). Ledger counts
reconcile exactly against the brief's declared per-service declaration
counts. `classify-file05.tsv` is untouched.

Non-blocking: design.md's §4.2 skip-condition prose (3 bullets) does not
literally enumerate the "multiple builders in file" / distinct-builder-type
check that `staticSkipReason` actually implements as a fourth condition —
worth a doc fix in a later task, not a defect in this range.
