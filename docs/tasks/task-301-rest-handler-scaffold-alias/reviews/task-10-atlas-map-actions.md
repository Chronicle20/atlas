# Review: Task 10 — atlas-map-actions REST handler scaffolding alias

Commit range: `6fd988e97..61b0bd614` (bb5f74d2d conversion, 61b0bd614 ParseScriptId delegation)

## Scope

`git diff --stat 6fd988e97..61b0bd614` touches exactly two files:

```
services/atlas-map-actions/atlas.com/map-actions/rest/handler.go   | 99 +-------
services/atlas-map-actions/atlas.com/map-actions/script/resource.go | 252 +++++++++----
```

This matches the brief's file list exactly. Scope confirmed: no other file in the module or repo changed in this range.

## Checklist results

### 1. All `d.DB()` sites gone
`grep -rn 'd\.DB()' services/atlas-map-actions/atlas.com/map-actions` → no matches (exit 1). PASS.

### 2. `rest/handler.go` alias form
Current file (`services/atlas-map-actions/atlas.com/map-actions/rest/handler.go`):
- `HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` are type aliases to `server.*`.
- `RegisterHandler` is a `var` alias to `server.RegisterHandler`.
- `RegisterInputHandler[M]` is a thin wrapper delegating to `server.RegisterInputHandler[M](l)` (generic functions cannot be aliased with `var =`, so the wrapper is the correct idiom — matches the pattern in other converted services, e.g. `atlas-quest`).
- Local `ParseInput` dropped.
- Dead imports (`context`, `io`, `gorm.io/gorm`) pruned in commit `bb5f74d2d`; `uuid`, `mux`, `jsonapi`, `logrus`, `net/http`, `server` all remain referenced by `ParseScriptId`/`ParseScriptName`/type signatures.
PASS.

### 3. `ParseScriptName` kept verbatim
`services/atlas-map-actions/atlas.com/map-actions/rest/handler.go:32-43` — `ScriptNameHandler` type and `ParseScriptName` body are byte-identical to the pre-conversion source (compared against `git show 6fd988e97:.../rest/handler.go`). Not delegated, as required (it does more than a bare lookup: empty-string 400 with a custom log message, no `mux`/`uuid` bare-parse-and-forward). PASS.

### 4. `ParseScriptId` delegation + `ScriptIdHandler` deletion
`rest/handler.go:29-31` (post `61b0bd614`):
```go
func ParseScriptId(l logrus.FieldLogger, next func(scriptId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}
```
Matches `server.ParseUUIDId(l logrus.FieldLogger, varName string, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc` (`libs/atlas-rest/server/id_parser.go:28`) exactly — varName `"scriptId"` matches the original `mux.Vars(r)["scriptId"]` key.
`ScriptIdHandler` type deleted in the same commit. `grep -rn 'ScriptIdHandler' services/atlas-map-actions/atlas.com/map-actions` → zero hits anywhere in the module (only `ScriptNameHandler` remains, which is a different, still-live type). PASS — deletion is genuinely safe.

### 5. Commit split is real
`git diff --stat bb5f74d2d..61b0bd614 -- services/atlas-map-actions/atlas.com/map-actions/script/resource.go` → empty output (exit 0, no diff). `script/resource.go` is entirely contained in commit `bb5f74d2d`; commit `61b0bd614` touches only `rest/handler.go` (`git show 61b0bd614 --stat` confirms single file, 2 insertions/13 deletions). PASS.

### 6. Test file / main.go / libs/atlas-rest untouched
`git diff 6fd988e97..61b0bd614 -- services/atlas-map-actions/atlas.com/map-actions/script/resource_test.go services/atlas-map-actions/atlas.com/map-actions/main.go libs/atlas-rest/` → 0 lines of output. `go test ./...` (run in-tree by this review, not just trusted from the implementer report) → `ok atlas-map-actions/script 0.061s`, full module `go test ./...` all green, no skips. PASS.

### 7. Build / test / gofmt (run independently by this review)
```
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && gofmt -l .
```
- `go build ./...` → clean, no output.
- `go test ./...` → all packages `ok` or `[no test files]`, no failures.
- `gofmt -l .` → empty output (no unformatted files).
PASS.

### 8. Shape A consistency across all six handlers, matching the reported deviation
Confirmed independently: `server.HandlerDependency` (`libs/atlas-rest/server/context.go:14-17`) holds only `l`/`ctx`, exposes `Logger()`/`Context()` — no `DB()` method. The implementer's stated deviation from the brief's literal Step 1/Step 2 prose is real and correctly diagnosed, not an excuse for an incorrect landing.

Verified the diff (`git show bb5f74d2d -- .../script/resource.go`) applies Shape A identically at all six sites:
- `GetAllScriptsHandler(db *gorm.DB) rest.GetHandler` — wraps, `db` used in `NewProcessor(...).AllProvider`.
- `GetScriptHandler(db *gorm.DB) rest.GetHandler` — wraps around the existing `rest.ParseScriptId(...)` curry, `db` threaded through correctly (wrapping order is `db-closure(d,c) -> ParseScriptId(...) -> func(w,r)`, preserving the original inner nesting).
- `GetScriptsByNameHandler(db *gorm.DB) rest.GetHandler` — same pattern around `rest.ParseScriptName`.
- `CreateScriptHandler(db *gorm.DB) rest.InputHandler[RestModel]` — signature correctly changed from `(d, c, rm)` bare handler to a constructor returning `InputHandler[RestModel]`.
- `UpdateScriptHandler(db *gorm.DB) rest.InputHandler[RestModel]` — same, with `ParseScriptId` curry preserved inside.
- `DeleteScriptHandler(db *gorm.DB) rest.GetHandler` — same.

`InitResource` (`script/resource.go:19-37`) registration lines: `rest.RegisterHandler(l)(db)(si)` → `rest.RegisterHandler(l)(si)`; `rest.RegisterInputHandler[RestModel](l)(db)(si)` → `rest.RegisterInputHandler[RestModel](l)(si)`; every `register*("name", XxxHandler)` call site gained `(db)`: `GetAllScriptsHandler(db)`, `CreateScriptHandler(db)`, `GetScriptHandler(db)`, `UpdateScriptHandler(db)`, `DeleteScriptHandler(db)`, `GetScriptsByNameHandler(db)`. No identifier renamed, all six remain exported under identical names (repo convention: "Do not rename or unexport handler functions" — respected).

No unrelated body changes: line-by-line diff of each handler body shows only `d.DB()` → `db` substitutions and the wrapper/closure restructuring required by the shape change; no logic, status code, or error-handling behavior altered. PASS — the landed shape is correct and is safe for Tasks 11/12 to copy mechanically.

## Findings

None blocking. None non-blocking beyond what is already documented and ruled on by the controller (two-commit split, Step1/Step5 resolution, no-new-tests policy — all pre-ruled, not re-litigated here).

## Not evaluable

None — the full review surface (both commits' diffs, `rest/handler.go`, `script/resource.go`, `server.HandlerDependency`, `server.ParseUUIDId`, `server.RegisterHandler`/`RegisterInputHandler` contracts, build, test, gofmt) was directly inspected and independently re-run rather than trusted from the implementer report.

## Verdict

APPROVED. This diff is the correct reference pattern for Tasks 11 (atlas-portal-actions) and 12 (atlas-reactor-actions) to copy mechanically.
