# Review: Task 8 — atlas-parcel (REST handler scaffolding alias sweep)

Commit range: `b96fa96b6..97bad8d53` (77ec4e5e9 conversion, 97bad8d53 id-parser delegation)
Brief: `.superpowers/sdd/plan/task-8-brief.md`
Report: `.superpowers/sdd/plan/task-8-report.md`

## Verdict

APPROVED

## Checks

### 1. All `d.DB()` sites removed

`grep -rn 'd\.DB()' services/atlas-parcel/atlas.com/parcel` returns zero matches (exit 1,
no output). All 6 sites named in the brief were converted; no handler missed, no double
conversion (spot-checked `resource.go` in full — 5 handler constructors each take
`db *gorm.DB` closed over from `InitResource`, and `writeParcels`, the one plain helper
that never touched `d.DB()`, is unchanged, matching the report's claim).

### 2. `rest/handler.go` matches the brief's alias form

`services/atlas-parcel/atlas.com/parcel/rest/handler.go` (final state) is byte-for-byte
the alias/delegation shape specified in the brief's Step 1 code block, plus
`ParseCharacterId`/`ParseParcelId` delegated per Step 5:

- `HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` are type
  aliases to `server.*`.
- `RegisterHandler` is a `var` alias; `RegisterInputHandler[M]` delegates to
  `server.RegisterInputHandler[M](l)`.
- `InputHandler[M]`/`RegisterInputHandler[M]` correctly PRESERVED — atlas-parcel has 2
  genuine input-handler call sites (`registerDiscardPatch`, `registerNotifyPatch` in
  `resource.go:46-47`, wired to `handleDiscardParcel`/`handleNotifyParcel`).
- Local `ParseInput` dropped — confirmed absent from the file.
- Imports pruned to exactly `net/http`, `github.com/jsonapi`, `logrus`,
  `server` in the scaffolding half; `strconv` and `gorilla/mux` additionally pruned in
  commit 2 once `ParseCharacterId`/`ParseParcelId` stopped needing them directly
  (`server.ParseIntId`/`server.ParseStringId` now own those lookups). `context`, `io`,
  `gorm.io/gorm` pruned per brief (no longer referenced in `rest/handler.go`).

### 3. Commit split is real and revertable

- `git diff --stat 77ec4e5e9..97bad8d53` shows only `rest/handler.go` changed in commit 2
  (28 lines, 2 insertions/26 deletions) — no `resource.go` parser-body change leaked into
  the delegation commit.
- `git diff --stat 77ec4e5e9..97bad8d53 -- .../resource.go` is empty, confirming zero
  diff on the non-parser file between the two commits.
- Commit 1 (`77ec4e5e9`) alone carries the full scaffolding + Shape A/C conversion
  (`resource.go` 370 lines touched, `handler.go` 84 lines touched for the scaffolding
  half only — `ParseCharacterId`/`ParseParcelId` bodies untouched there).

### 4. `parcel/resource_test.go` untouched

`git diff --stat b96fa96b6..97bad8d53 -- '**/resource_test.go'` is empty — no test file
in the diff. `go test ./...` run locally passes (see below), confirming it stayed green
unedited.

### 5. `libs/atlas-rest/` and `main.go` untouched

`git diff b96fa96b6..97bad8d53 -- libs/atlas-rest/ services/atlas-parcel/atlas.com/parcel/main.go`
is empty (0 lines).

### 6. Build / test / fmt (reviewer-run, not report-trusted)

```
$ cd services/atlas-parcel/atlas.com/parcel && go build ./...
(exit 0)
$ go test ./...
Go test: 81 passed in 10 packages
$ gofmt -l .
(no output — clean)
```

## Non-blocking notes

None.

## Not evaluable

None — full review surface (both commits, the two changed files, and the id-parser
narrowing already ruled on by the controller) was directly inspectable and independently
rebuilt/retested.

## Controller-settled items (not re-litigated)

- Two-commit split: mandated, present, correct.
- `server.ParseStringId` empty-vs-absent narrowing for `ParseParcelId`: spec-recorded,
  accepted, unreachable through gorilla/mux routing for a required `{parcelId}` segment.
- No new tests added: per plan, not a finding.
