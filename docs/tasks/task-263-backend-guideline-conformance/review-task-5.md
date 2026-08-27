# Review — Task 5: disambiguate `atlas-character/character`'s two builders

Commit under review: `c861e29b4d4414cba995ff6a1a9a84ec29aaa712`
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Implementer report: `docs/tasks/task-263-backend-guideline-conformance/task-5-report.md`

## Scope

`git show --stat c861e29b4` — 19 files changed, all under `services/atlas-character`, 90
insertions / 88 deletions. Matches the brief's expected shape (88 call-site renames + 2
comment lines added). Scope confirmed: this is exactly the disambiguation sweep described
in the brief, nothing more.

## Checks

1. **Leftover `NewModelBuilder` references** —
   `grep -rn --include='*.go' 'NewModelBuilder' services/atlas-character` → empty. PASS.

2. **`builder.go` untouched** —
   `git show c861e29b4 -- services/atlas-character/atlas.com/character/character/builder.go`
   produces only the commit header, no diff hunks; the file is not in the commit's changed-file
   list at all. `NewBuilder` (builder.go:91) and `type Builder` (builder.go:30) remain as
   declared before the change. PASS.

3. **Nothing outside `services/atlas-character` staged** —
   `git show --name-only c861e29b4` lists 19 files, all under
   `services/atlas-character/atlas.com/character/...`. PASS.

4. **Doc comment on the new declaration is accurate** —
   `model.go:242-244` (post-commit):
   ```go
   // NewEmptyBuilder creates a zero-valued builder for reconstructing a Model.
   // The creation-flow builder is NewBuilder in builder.go; the two are distinct.
   func NewEmptyBuilder() *modelBuilder {
   	return &modelBuilder{}
   }
   ```
   Confirmed accurate: `NewBuilder` does exist in `builder.go:91` as a genuinely distinct
   character-creation constructor (`func NewBuilder(c BuilderConfiguration, accountId uint32,
   worldId world.Id, name string, skinColor byte, gender byte, hair uint32, face uint32)
   *Builder`), and `NewEmptyBuilder` here returns a bare zero-valued `&modelBuilder{}` used for
   model reconstruction. The comment does not overstate or misdescribe either builder. PASS.

5. **FR-13 exemptions (`type modelBuilder`, `CloneModel`) left unchanged, per task instruction** —
   `model.go:211` still declares `type modelBuilder struct`, `model.go:248` still declares
   `func CloneModel(m Model) *modelBuilder`. Not flagged as misses, per the task's explicit
   ruling that these are FR-13 exemptions to be recorded in `exemptions.md` by a later task
   (Task 25).

6. **Build sanity** — `go build ./character/... ./kafka/consumer/character/... ./pending_change/...`
   from `services/atlas-character/atlas.com/character` exits clean, no output. Confirms the
   rename didn't break compilation in the touched packages (spot-check of the report's
   Step 5 claim, not a full `verify.sh` run — that gate is separate per repo convention).

## Findings

None. No blocking or non-blocking issues found within the reviewed surface.

## Not evaluable

- Full `go test ./...` and `go vet ./...` across the whole module were not re-run by this
  review (only a scoped `go build` on the touched packages was executed); the report's Step 5
  transcript claiming a full green run is taken as reported evidence, not independently
  reverified end-to-end.
- `exemptions.md` entries for `type modelBuilder`/`CloneModel` are explicitly out of scope for
  this task (deferred to Task 25) and were not checked.

## Verdict

APPROVED — every checked item passes with cited evidence; no scope creep, no leftover
references, no unintended file touches, and the new doc comment is factually accurate.
