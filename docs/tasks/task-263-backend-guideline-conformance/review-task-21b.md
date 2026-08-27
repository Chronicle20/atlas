# Review: Task 21-B — W1 `relocate` codemod, batch B (full range, all 3 segments)

Range reviewed: `59afb53af..b7f3e8d90` (13 commits). Baseline for behavior-preservation: `59afb53af`.

## Scope confirmed

All 21 changed files fall inside batch B's declared surface: the codemod (`relocate.go`,
`relocate_test.go`), the batch ledger (`ledger-relocate-b.tsv`), and exactly the nine named
modules' `model.go`/`builder.go` (or `message.go`/`builder.go` for marriages). No file outside
that list appears in `git diff --stat 59afb53af..b7f3e8d90 --name-only`. `go.work.sum` does not
appear in any commit in the range. Matches the brief and both continuation briefs. No scope
mismatch.

## 1. `ac3fa65a3` — codemod fix for free-floating section comments (highest scrutiny item)

**Root cause as described is plausible and independently reproduced.** `renderFile`
(`docs/tasks/task-263-backend-guideline-conformance/codemod/relocate.go:548-602`) now classifies
each decl's comment groups into `declDoc` (left to the printer), interior comments (left to the
printer), and leading/trailing floating groups (written verbatim via the new `writeCommentGroup`
helper) using position tests `cg.End() < ds.decl.Pos()` / `cg.Pos() > ds.decl.End()`.

**RED→GREEN claim verified directly, not taken on faith.** Checked out `relocate.go` at
`ac3fa65a3^` (pre-fix) with the new test file present and ran
`TestRelocatePreservesFloatingSectionComments`: it fails, dropping both floating comments — output
matches the report's pasted failure text verbatim. Re-checked out the fixed `relocate.go`: the same
test passes. This is a real regression test, not a decoration.

**Test-coverage judgment: the committed test is narrower than what the brief asked this commit to
be judged against, but manual probing shows the fix itself generalizes correctly.** The committed
`TestRelocatePreservesFloatingSectionComments` only covers "floating comment before a decl that
stays" (twice) — it does not exercise: an EOF-trailing comment after the last decl, two consecutive
floating comment groups before one decl, a floating comment before a decl that *moves*, or
interleaved line/block comments. Given this codemod's track record (seven defects across Tasks
19-21, all found by real-tree application, none by unit tests), a test that covers only the exact
families shape is weak evidence on its own.

To close that gap I built a standalone probe (not committed, removed after use) combining all four
of the untested shapes — EOF trailing comment, two consecutive floating comments before a decl
that stays, a line comment immediately before the trailing EOF comment, a block comment mixed with
line comments, and a floating comment before a decl that *moves* into `builder.go` — against the
fixed `relocate.go`. All six distinct comments in that fixture appeared exactly once each, in the
correct file, correctly heading their declarations (including the floating comment ahead of a
moving decl landing correctly in `builder.go`, and the EOF comment surviving with no following
decl to attach to). So while the regression test's own scope is narrow, the fix's actual behavior
holds up against every scenario the task asked me to check. This is a non-blocking finding: the
committed test under-specifies the contract it's supposed to pin, even though the underlying fix is
sound by direct probing.

No other codemod behavior was touched — the diff is confined to `renderFile`'s per-decl comment
loop plus two new unexported helpers, consistent with the report's self-review.

## 2. `d21d54298` — atlas-families re-application

Both previously-dropped comments are present in the current tree, each exactly once
(`grep -n "Business logic methods\|Pure functions for business logic validation"` on
`services/atlas-families/atlas.com/family/family/model.go` → lines 75 and 159, no other hits in
`builder.go`), and each correctly heads its original declaration (`HasSenior` at line 78,
`ValidateCharacterId` at line 161). The genuinely-attached `// Builder forward declaration...` doc
comment moved with the `Builder` struct into `builder.go`, unchanged.

Diff against baseline `59afb53af` for both files, filtered per the brief's Step-2 recipe, is
symmetric: 14 `+` / 14 `-`, every line pairs (struct decl + its 10 fields + closing brace).
`go build && go vet` clean for the module; `tools/lint.sh --check --go
services/atlas-families/atlas.com/family/family` → `lint.sh: OK`.

## 3. Lint-gate fix commits (the only behavior-touching commits in range)

**`f1faf17aa` (atlas-messages, dead `stance` field).** Confirmed pre-existing at base:
`git show 59afb53af:services/atlas-messages/atlas.com/messages/character/model.go` has the field
declared once, and `git grep -n '\.stance\b|SetStance' 59afb53af -- services/atlas-messages` finds
no reads or writes anywhere in the service at the base commit. Removal is inert.

**`da3ece073` (atlas-marriages, discarded `Put` error).** Confirmed pre-existing and unchecked at
base: `git show 59afb53af:.../message.go` has both `bb.buffer.Put(topic, provider)` call sites
(lines 108, 115) with the `error` return value already ignored, identical to the post-move
`builder.go`. `Buffer.Put` can return a non-nil error (from the provider function `p()` failing),
but that error was silently dropped in both `AddMessage` and `AddConditionalMessage` before this
task touched the file — the fix makes the discard explicit (`_ = ...`) without changing which
errors are observed or where. Genuinely inert; not masking a newly-introduced failure path.

Both fixes are separate, clearly-labeled commits landed after their respective pure-relocation
commits, exactly as the brief required.

## 4. The other seven relocation commits

Ran the brief's own Step-2 asymmetry filter against each commit's diff, isolated to the module's
own paths:

- `d4cba2f98` atlas-notes — symmetric on decl content; one unmatched line is a new aliased import
  (`sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"`) added to `builder.go`, which the
  filter regex doesn't recognize as an import line (it only matches `^\+\t"..."`, not
  `^\+\talias "..."`). This is the same known filter gap flagged in batch A's report, not a
  defect — the import is legitimately duplicated because `builder.go` is a new compilation unit
  that needs its own import block; `model.go`'s copy of the same import is untouched.
- `98fe7db6c` atlas-messages — fully symmetric, no unmatched lines.
- `b29063a95` atlas-marriages — fully symmetric, no unmatched lines.
- `d7979fec1` atlas-inventory — fully symmetric, no unmatched lines.
- `59adbb47d` atlas-drops — same aliased-import filter artifact as atlas-notes
  (`tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`), not a defect.
- `54eec0dab` atlas-doors — same aliased-import filter artifact
  (`_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`), not a defect.
- `219387e44` atlas-character-factory — fully symmetric, no unmatched lines.
- `b7f3e8d90` atlas-character — fully symmetric, 211 `+` / 211 `-`, no unmatched lines.

No receiver/signature drift, no dropped struct field, no visibility change, and no build-tag or
`init()` ordering change observed in any of the eight relocation diffs (nine including families,
covered separately above).

Spot-checked build/vet directly (via `go build ./... && go vet ./...` from each module root under
the workspace, not `GOWORK=off`) for the two highest-risk modules, families and character — both
clean — and re-ran `tools/lint.sh --check --go` for families, character, messages, and marriages —
all `lint.sh: OK`.

## 5. Ledger accuracy

`wc -l docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-b.tsv` → 21.
`cut -f2 ledger-relocate-b.tsv | sort | uniq -c` → 17 `APPLIED`, 4 `SKIPPED`. Matches the
continuation report's final claimed state exactly. `services/atlas-families/atlas.com/family/family/model.go`
row is `APPLIED` in the current file (flipped from the earlier `SKIPPED (defect...)` recorded by
`b2a35ab8c`), consistent with `d21d54298`'s re-application. All nine batch-B modules have exactly
one `APPLIED` row each; the remaining 4 `SKIPPED` rows are all `entity builder` rows inherited from
batch A, none touched by batch B. Ledger is internally consistent with what the commits actually
did.

## 6. Scope

Confirmed via `git diff --stat 59afb53af..b7f3e8d90 --name-only`: exactly 21 files, all inside the
codemod dir, the ledger, and the nine named modules. `go.work.sum` does not appear in
`git log 59afb53af..b7f3e8d90 --stat --name-only`. `progress.md` and `agent-ledger.tsv` are not
part of this commit range (they are separately dirty in the working tree per the task's own note)
and are correctly excluded from every commit.

## Not evaluable

- Batch C's modules are out of scope for this review and were not examined; the report's own
  caveat ("budget time to re-verify rather than assume [the fix] holds for batch C") is a fair
  flag for that unit's reviewer, not a finding against this one.
- Full-suite `go test ./...` for every one of the nine modules was not re-run in this review
  (the report's own per-module test-count table was taken as evidence for that layer); `go
  build`/`go vet` were spot-checked directly for two of the nine.

## Verdict rationale

No blocking defect found. The single non-blocking finding is that `ac3fa65a3`'s committed
regression test is narrower than the shape space the task asked this commit to be judged against
(EOF comment, two consecutive floating comments, floating comment before a moving decl, and
interleaved line/block comments are all untested by the committed test) — but direct probing against
the fixed code confirms the fix itself handles every one of those shapes correctly, so this is a
test-quality gap, not a live defect.
