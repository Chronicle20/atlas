# Review: Task 3 — delete unreferenced rest/handler.go (atlas-fame, atlas-events)

Range reviewed: `cce9a5ca4..e023ad2ee`
Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Findings

### PASS — Zero-consumer claim independently verified
- `git grep -n '"atlas-fame/rest"' <base|HEAD> -- services/atlas-fame/**/*.go` → no hits (exit 1) at both `cce9a5ca4` and current HEAD.
- `git grep -n '"atlas-events/rest"' cce9a5ca4 -- services/atlas-events/**/*.go` → no hits.
- Confirmed module names match the grepped import paths: `services/atlas-fame/atlas.com/fame/go.mod` → `module atlas-fame`; `services/atlas-events/atlas.com/events/go.mod` → `module atlas-events`.
- Checked both `main.go` files at base commit directly for any `rest` import/reference — `atlas-fame/main.go` and `atlas-events/main.go` only reference `libs/atlas-rest/server`, never the local `rest` package.
- Independently confirmed both `rest/` directories at base commit contained exactly `handler.go` (`git ls-tree -r cce9a5ca4 --name-only`), matching the brief's expectation that the whole directory should go.

### PASS — Diff scope matches exactly what was authorized
`git diff --name-status cce9a5ca4..e023ad2ee`:
```
D	services/atlas-events/atlas.com/events/rest/handler.go
D	services/atlas-fame/atlas.com/fame/rest/handler.go
```
No other files touched. `libs/atlas-rest/`, any `main.go`, and `services/atlas-mts/atlas.com/mts/testsupport/resource.go` are absent from the diff (`git diff --name-only ... | grep -E 'libs/atlas-rest|main\.go|testsupport/resource.go'` → no matches). No `go.mod`/`go.sum` changes in either module (deleted package had no unique imports to prune).

### PASS — `event/definition/resource.go` read-only, already uses the direct registration shape
Confirmed unmodified in this diff and, at base commit, line ~32 (`InitResource`) does call `server.RegisterHandler(l)(si)` directly, matching the brief's stated rationale for why `atlas-events`'s handler is dead code rather than something needing conversion.

### PASS — Build and test, both modules
- `atlas-fame`: `go build ./...` clean; `go test ./...` — all packages `ok` (10 packages, matches report's "37 passed").
- `atlas-events`: `go build ./...` clean; `go test ./...` — all packages `ok`, including `atlas-events/event/definition` and `atlas-events/event/occurrence` (the two resource tests the brief called out) — both green, matches report's "132 passed."
- No test file was edited to make this pass (`git diff --name-status` shows only the two handler.go deletions — no test touched).

### BLOCKING — Single commit spans two Go modules, violating the binding global constraint "Commit per service"
`global-constraints.md:18`: *"Commit per service. Each service is its own Go module."* The range under review is exactly one commit (`e023ad2ee`, `refactor(atlas-fame,atlas-events): delete unreferenced rest handler packages`) that deletes files in both the `atlas-fame` module and the `atlas-events` module. `git log --oneline cce9a5ca4..e023ad2ee` shows only this one commit.

This is a real defect, not a nitpick: the stated purpose of "commit per service" is revertability without touching an unrelated module (see the parser-delegation clause immediately following in the same constraint line). As written, `git revert e023ad2ee` cannot undo the `atlas-fame` deletion without also reverting the unrelated `atlas-events` deletion, and vice versa.

Root cause: the task-3 brief itself (`task-3-brief.md`, Step 4) instructs a single `git commit` after `git rm -r`-ing both services' `rest/` directories in one step. The implementer followed the brief literally and did not flag the conflict with the binding global constraint, even though CLAUDE.md and the global-constraints doc both state the global constraints override task-specific instructions. This should have been caught and reported as a plan defect rather than executed as written; splitting into two commits (`git rm -r services/atlas-fame/... && git commit`, `git rm -r services/atlas-events/... && git commit`) would have cost nothing and satisfied both the brief's intent and the constraint.

- `services/atlas-fame/atlas.com/fame/rest/handler.go:1` (module: atlas-fame)
- `services/atlas-events/atlas.com/events/rest/handler.go:1` (module: atlas-events)
— both deleted together in the single commit `e023ad2ee`.

## Not evaluable
None — the full review surface (both deleted files, their historical consumers, the referenced read-only file, and both modules' build/test) was covered directly.

## Verdicts

- **Spec compliance (against task-3-brief.md as literally written):** met. Every checklist step (fresh grep gate, deletion, build/test, commit) was executed and the report's claims are independently verified true.
- **Task quality (against the binding global constraints that override the brief):** CHANGES_REQUIRED — the single cross-module commit violates the explicit "commit per service" global constraint. The fix is mechanical (split the existing commit into two, one per module) and does not require redoing any of the verified deletion work.
