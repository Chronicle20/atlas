# Review: Task 6 — `rename` subcommand (commit 27c5fe431)

## Scope

Reviewed commit `27c5fe43122bd06ad1cf1c1ce22940530336213b` ("chore(task-263): add
type-aware builder rename to codemod"):

- `docs/tasks/task-263-backend-guideline-conformance/codemod/rename.go` (new, 458 lines)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/rename_test.go` (new, 365 lines)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` (+3/-2, wiring only)

Cross-checked against `.superpowers/sdd/plan/task-6-brief.md` and
`docs/tasks/task-263-backend-guideline-conformance/task-6-report.md`. Also inspected
`load.go`, `classify.go`, `report.go` only as far as needed to confirm reuse
(`LoadModules`, `moduleRootFor`, `readLines`, `splitEntry`, `Ledger`) is correct, per
"code the diff calls" scope.

Ran `GOWORK=off go build ./... && go vet ./... && go test ./... -run TestRename -v`
inside the codemod module: build clean, vet clean, all 8 `TestRename` subtests pass.

## Blocking findings

### 1. The worktree is dirty with ~83 files of uncommitted, undisclosed modifications across `services/atlas-channel` that could not have been produced by the reviewed code, and the task-6 report does not mention them

`git status --short` (run against this worktree at HEAD = `27c5fe431`) shows 81
modified files under `services/atlas-channel/atlas.com/channel/**` plus an untracked
`docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv`
(19 `APPLIED` lines, no `SKIPPED`, TSV format matching `Ledger.WriteTo`'s own output —
strongly suggesting a real, non-dry-run invocation of `rename -only
services/atlas-channel -ledger ledger-rename-channel.tsv`, timestamped one minute
after the commit). Nothing is staged (`git diff --cached` is empty), so the literal
ask ("confirm nothing outside the codemod module was staged") is satisfied — but the
brief's Step 5 calls for a **dry-run only**, and the task-6 report's "Dry-run against
the real tree" section reports only the whole-tree dry run (`-dry-run`, "exit 0, no
output"). It never mentions running the tool for real against `atlas-channel`, nor
does it mention reverting it.

Worse: the file contents in this uncommitted diff are **inconsistent with the
committed `rename.go`**. Example,
`services/atlas-channel/atlas.com/channel/character/builder_test.go`:

```
-func TestNewModelBuilder(t *testing.T) {
+func TestNewBuilder(t *testing.T) {
```
```
-func TestModelBuilder_SetMaxMp(t *testing.T) {
+func TestBuilder_SetMaxMp(t *testing.T) {
```

`TestNewModelBuilder` and `TestModelBuilder_SetMaxMp` are distinct, unrelated
top-level test functions — their `*types.Func` objects can never be `==` to the
`NewModelBuilder`/`ModelBuilder` objects `renameImpl` resolves (`rename.go:76-90`),
and `applyObjectRenames` (`rename.go:186-217`) only ever assigns `ident.Name = newName`
for a whole identifier whose `Uses`/`Defs` object is a map key — it never does
substring rewriting inside an unrelated, longer identifier. This pattern repeats
identically across every one of the ~19 `APPLIED` packages in
`ledger-rename-channel.tsv` (confirmed via `git diff -- services/atlas-channel | grep
'^[-+]func Test'`), meaning the modifications sitting in this worktree were **not**
produced by the code in this commit. Whatever tool actually produced them (an earlier
prototype, a hand script, or work from a different, uncommitted attempt) left the tree
in a state inconsistent with the reviewed deliverable, and this was neither disclosed
nor cleaned up.

This is a hygiene and provenance problem the next task (`Tasks 7-9`, which apply this
very tool in bulk) will inherit blind: if this stray, unexplained diff is committed
alongside a future task's real output, it silently smuggles an out-of-scope rename
(test function names, never named by FR-12/13/14) into the history under an unrelated
commit message. Per repo convention ("the tree must be clean after \[an agent\]
runs"), this must be investigated and either reverted or explained/committed on its
own merits before Task 7 proceeds. — `services/atlas-channel/atlas.com/channel/character/builder_test.go:13,52`,
`docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv` (untracked)

## Non-blocking findings

### 2. Report's causal claim about why `Tests: true` was needed for the `atlas-quest/quest` SKIP is not supported by the evidence it cites

The report states the `quest` collision was "triggered by a `_test.go` local var" and
that without `Tests: true` "the quest collision ... would have gone undetected." In
fact the colliding local var is declared twice in **production** code —
`services/atlas-quest/atlas.com/quest/quest/processor.go:851` and `:921`
(`builder := sagaproducer.NewBuilder(...)`) — inside the base (non-test) package,
which the collision guard (`rename.go:65-73`, `hasObjectNamed` at `rename.go:100-108`)
already scans without needing `Tests: true`. The `builder_test.go:14` local var the
report cites lives in the external `quest_test` package, whose own
`pkg.Types.Scope().Lookup("modelBuilder")` is nil (unexported name, invisible from
outside the package), so that variant's `Defs` are never what trips the guard's
`&&` condition. The design decision to add `Tests: true` is still sound for a
different, real reason — it is needed to rewrite a `_test.go` call site of the
renamed constructor itself (confirmed for real: `builder_test.go:14,41,79,122,...`
call `quest.NewModelBuilder()`, which would break after a production-only rename) —
but the report's specific evidenced example for *why the SKIP was caught* overstates
what `Tests: true` contributed here. Not blocking (the guard is correct; the
narrative is just imprecise), but worth correcting before this reasoning is reused
to justify future decisions.

### 3. `loadModuleForRename`'s `Tests: true` path is exercised only against the real tree, never by an automated `TestRename` case

Every subtest in `rename_test.go` calls the shared `LoadModules` (no `Tests: true`),
never `loadModuleForRename`. The brief's test table (8 rows) doesn't ask for a
`_test.go`-call-site fixture either, so this matches the brief's letter — but it means
the one behavior that motivated writing a second, duplicated loader (reaching a
`_test.go` call site) has zero regression coverage; it was validated only by the
real-tree dry run finding the (differently-explained, see finding 2) `quest` SKIP.
A future refactor of `loadModuleForRename` could silently regress `Tests: true` and no
test would catch it. Worth a follow-up fixture (a package + its own `_test.go` calling
the renamed constructor) in a later task, not blocking for this one.

### 4. Collision guard covers only the `modelBuilder`/`builder` pair, matching the brief precisely, but leaves the `Builder`/`NewBuilder`/`CloneBuilder` collision case unchecked

Per the brief's R2, this is intentional and in-scope: only the unexported pair is
guarded. Flagging for completeness since it is a place a package could still fail to
compile after a real (non-dry-run) apply if, e.g., a package already declared an
unrelated `type Builder struct` or `func NewBuilder(...)` before this codemod runs
(this exact shape exists for `channel/asset` and `character`, per Tasks 4-5's hand
disambiguation, though in both of those cases the pre-existing name was `Builder`,
not something colliding with a fresh `NewModelBuilder`→`NewBuilder`/`ModelBuilder`→
`Builder` target, so no actual collision was hit in the real dry run). Not evaluable
against the given brief (brief doesn't ask for this), so not a scope defect —
recorded as a residual risk for Tasks 7-9 to watch for.

## Passed checks (with evidence)

- **`Target`/`Rename` signature matches the brief exactly.** `type Target struct {
  PkgDir string; From, To string }` and `func Rename(pkgs []*packages.Package, targets
  []Target) (*Ledger, error)` — `rename.go:38-46, 59-61`.
- **Fixed FR-12/13/14 rename table matches the brief's table verbatim**, including
  `CloneModel` correctly excluded — `rename.go:48-54`; confirmed by the
  `CloneModel_untouched` subtest (`rename_test.go:171-208`), which passes.
- **Object-identity, not name-matching, drives the rewrite.** `applyObjectRenames`
  (`rename.go:186-217`) keys off `info.Uses[ident]`/`info.Defs[ident]` against a
  `map[types.Object]string`; it never compares `ident.Name` to a target string except
  to decide whether a change is a no-op. Confirmed by the `cross-package call site`
  subtest passing (`rename_test.go:210-252`), which rewrites both the qualified type
  reference `*fixture.ModelBuilder` and the call `fixture.NewModelBuilder()` in a
  separate importing package.
- **`_test.go` call sites are reached in principle.** `loadModuleForRename` sets
  `Tests: true` (`rename.go:275-301`) and `findPackages` matches every package variant
  sharing a directory by trailing-path-segment suffix (`rename.go:117-131`), so a
  `p_test [p.test]` package's `Uses` map is included in the walk. Confirmed for real:
  `services/atlas-quest/atlas.com/quest/quest/builder_test.go`'s calls to
  `quest.NewModelBuilder()` were reached by the collision-guard scan that produced the
  real `SKIPPED` entry (see finding 2 for the caveat on the specific causal claim).
- **String literals and struct tags are never touched (R4/FR-17).** The rewrite only
  visits `*ast.Ident` (`rename.go:191-211`); confirmed by the `string literal
  untouched` subtest passing (`rename_test.go:297-317`), which asserts `json:
  "modelBuilder"` and `const s = "NewModelBuilder"` survive unchanged in a file whose
  `ModelBuilder`/`NewModelBuilder` type and constructor *are* renamed.
- **Collision guard (R2) matches the brief's algorithm.** Checks
  `pkg.Types.Scope().Lookup("modelBuilder")` and scans `TypesInfo.Defs` (covering both
  package- and function-scope objects) for an existing `builder` — `rename.go:65-73,
  100-108`. On a hit, the whole package's four renames are withheld (not applied
  partially) and the file is left byte-identical — confirmed by the `collision guard`
  subtest (`rename_test.go:254-295`), which asserts the ledger records `SKIPPED` with
  the exact reason string and that the file is unchanged.
- **Ledger fidelity.** `renameImpl` calls `ledger.Verify()` before any file write
  (`rename.go:94-96`), and the real dry run's ledger line count (59) equals the
  inventory's line count (59 = `wc -l inventory-dom01-newmodelbuilder.txt`), matching
  the `APPLIED + SKIPPED == input` invariant. Confirmed by the `ledger completeness`
  subtest passing (`rename_test.go:319-364`).
- **CRLF preservation and file-permission preservation** — `writeRenamedFile`
  (`rename.go:242-262`) reads original bytes first, checks for `\r\n`, and reuses
  `info.Mode().Perm()`. Not exercised by an automated test (no CRLF fixture in the
  table — matches the brief, which doesn't ask for one), so this is unverified by
  the test suite, only by code reading. Recorded under Not evaluable.
- **No `load.go`/`report.go` modification** — confirmed via `git show --stat
  27c5fe431`; only `rename.go`, `rename_test.go`, `main.go` changed. The duplicated
  rename-local loader (`loadModuleForRename` vs. `LoadModules`) is a deliberate,
  documented tradeoff (comment at `rename.go:266-274`, echoed in the report's
  "Concerns" section) to avoid risking `classify.go`'s per-root `byDir` assumption.
  Judged as the right call for this task: `classify.go` and `rename.go` have opposite
  requirements (one package variant vs. every test variant), and forcing one loader to
  serve both would either regress `classify.go` or complicate it with a mode flag the
  brief didn't ask for. It is a **maintenance hazard** worth flagging (a third
  subcommand needing test-awareness should prompt promoting this loader, as the report
  itself notes), but not a defect in this task.
- **Ledger's 59/58/1 vs. the 51 remaining-declaration count is correctly explained,
  not silently mismatched.** The report explicitly derives why 59 (input) ≠ 51
  (current live count) and does not paper over the gap — confirmed the underlying
  facts: `grep -c '^func NewModelBuilder(' services libs --include='*.go' | grep -v
  _test.go` = 51 (report's own quoted number, re-verifiable), inventory line count = 59
  (`wc -l`), ledger = 58 APPLIED + 1 SKIPPED = 59. This is ledger-honest bookkeeping,
  not a discrepancy the tool needs to fix.
- **`main.go` wiring is minimal** — 3 lines: subcommand map entry + comment update,
  no other change — `git show 27c5fe431 -- .../main.go`.
- **Test/build hygiene.** `GOWORK=off go build ./...`, `go vet ./...`, and `go test
  ./... -run TestRename -v` all pass cleanly, re-run directly for this review (not
  taken from the report).

## Not evaluable

- CRLF round-trip and file-permission preservation (`writeRenamedFile`) — no fixture
  exercises either path; verified by code reading only.
- Whether `findPackages`'s suffix-match (`strings.HasSuffix(filepath.Clean(pkg.Dir),
  clean)`) could ever over-match two unrelated directories that happen to share a
  path-segment suffix somewhere else in the real tree — plausible in the abstract, not
  something I could rule out or reproduce against the full 14-service tree within this
  unit's scope; the real dry run's 58/1 split gives no evidence of it having happened,
  but that is not a proof of absence.
- The full-tree dry run itself is not independently reproducible within review scope
  (a `go run` over 14+ services is expensive); I relied on the report's quoted ledger
  output and cross-checked it against the persisted inventory file's line count, which
  is consistent.

## Scope confirmation

The reviewed commit's diff is confined to the codemod module, as claimed, and nothing
outside it is staged. However, the worktree this review ran in is **not** in the state
the report describes it in — see Blocking Finding 1. `scope_confirmed` below reflects
what I evaluated (the commit's diff plus the working-tree discrepancy I found while
confirming "nothing staged outside the codemod module"), not a broader sweep of
`atlas-channel`.
