# Review: Task 11 — atlas-portal-actions REST handler scaffolding alias

Range under review: `61b0bd614..b0834c97f` (56e927c1e conversion, b0834c97f id-parser delegation)

Brief: `.superpowers/sdd/plan/task-11-brief.md`
Report: `.superpowers/sdd/plan/task-11-report.md`
Reference (approved): atlas-map-actions, `bb5f74d2d + 61b0bd614`

## Scope confirmed

Diff touches exactly the two files named in the brief:
- `services/atlas-portal-actions/atlas.com/portal/rest/handler.go`
- `services/atlas-portal-actions/atlas.com/portal/script/resource.go`

`git diff --stat 61b0bd614..b0834c97f -- services/atlas-portal-actions` confirms no other file in the module changed. No test file, `main.go`, or `libs/atlas-rest/` touched.

## Checks

### 1. All `d.DB()` sites removed
`grep -rn 'd\.DB()' services/atlas-portal-actions/atlas.com/portal` → zero hits. All 6 original sites (in `GetAllScriptsHandler`, `GetScriptHandler`, `GetScriptsByPortalHandler`, `CreateScriptHandler`, `UpdateScriptHandler`, `DeleteScriptHandler`) converted. PASS.

### 2. Shape A applied consistently to all six handlers
Diff of `56e927c1e` vs `61b0bd614` for `resource.go` shows each handler converted to `func XxxHandler(db *gorm.DB) rest.GetHandler` or `rest.InputHandler[RestModel]`, closing over `db`, e.g.:

```go
func GetAllScriptsHandler(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc { ... }
}
func UpdateScriptHandler(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc { ... }
}
```

`InitResource` drops the `(db)` curry from `registerHandler`/`registerInputHandler` construction (`rest.RegisterHandler(l)(db)(si)` → `rest.RegisterHandler(l)(si)`, `resource.go:19-20`) and passes `(db)` at each of the 6 registration call sites (`resource.go:24-29`). Matches the approved atlas-map-actions structure. PASS.

### 3. `rest/handler.go` alias form
`b0834c97f:services/atlas-portal-actions/atlas.com/portal/rest/handler.go` confirmed:
- `HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]` are type aliases to `server.*`.
- `RegisterHandler = server.RegisterHandler`.
- `RegisterInputHandler[M]` thin wrapper delegating to `server.RegisterInputHandler[M](l)`.
- Local `ParseInput` dropped.
- Dead imports (`context`, `io`, `gorm.io/gorm`, `mux` after step 5) pruned; final import list is `net/http`, `uuid`, `jsonapi`, `logrus`, `server` — matches the brief's target shape and the approved reference. PASS.

### 4. Id-parser delegation
Final state of `rest/handler.go`:
```go
func ParseScriptId(l logrus.FieldLogger, next func(scriptId uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "scriptId", next)
}
func ParsePortalId(l logrus.FieldLogger, next func(portalId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "portalId", next)
}
```
`ScriptIdHandler` / `PortalIdHandler` named types deleted. `grep -rn 'ScriptIdHandler\|PortalIdHandler' services/atlas-portal-actions/atlas.com/portal` → zero hits, confirming no surviving reference. `mux` import correctly dropped from `handler.go` (still legitimately imported in `resource.go` for `*mux.Router`, confirmed via build success and `mux.` usage at `resource.go:21`). PASS.

**Unreachability of the narrowed `ParsePortalId` check** (controller-accepted, verified per instruction): route registration at `resource.go:28` is `router.HandleFunc("/portals/{portalId}/scripts", ...)`. Gorilla mux does not match an empty path segment against `{portalId}`, so the original emptiness-check branch was already unreachable through the router in this service too — same shape as the approved `ParseParcelId` precedent. Non-blocking, confirmed not to diverge.

### 5. Commit split is real
`git diff --stat 56e927c1e..b0834c97f -- services/atlas-portal-actions/atlas.com/portal/script/resource.go` → empty output. The id-parser-delegation commit touches only `rest/handler.go` (`+4/-26`, confirmed via `git diff --stat 56e927c1e..b0834c97f -- services/atlas-portal-actions`). PASS.

### 6. Test file / main.go / libs untouched
- `git diff --stat 61b0bd614..b0834c97f -- .../script/resource_pagination_test.go` → empty.
- `git diff --stat 61b0bd614..b0834c97f -- .../main.go libs/atlas-rest` → empty.
- `go test ./...` from the module root: all packages pass, including `atlas-portal-actions/script` (0.169s) which houses the pagination test. PASS.

### 7. Build / test / fmt (self-verified)
From `services/atlas-portal-actions/atlas.com/portal`:
- `gofmt -l .` → no output (clean).
- `go build ./...` → exit 0.
- `go test ./...` → all packages `ok` or `no test files`; no failures.

PASS.

## Findings

None blocking. No non-blocking findings beyond the already-adjudicated `ParsePortalId` narrowing note recorded above for completeness (controller ruling: not a finding, verified unreachable for this service's actual route shape).

## Not evaluable

None — full diff surface, its direct dependency (`libs/atlas-rest/server` contract for `ParseStringId`/`ParseUUIDId`/`RegisterHandler`/`RegisterInputHandler`, already established as correct in prior approved conversions of the same pattern), build, and test were all directly exercised.

## Verdict

APPROVED
