# Review: Task 2 — atlas-mounts (Shape A pilot)

Range reviewed: `9d5b7b8a0..cce9a5ca4` (commits `dabb2eabe`, `cce9a5ca4`)
Files touched: `services/atlas-mounts/atlas.com/mounts/rest/handler.go`,
`services/atlas-mounts/atlas.com/mounts/mount/resource.go`

## Scope confirmation

`git diff --stat 9d5b7b8a0..cce9a5ca4` shows exactly these two files, two commits,
matching the brief's Step 1–6 plan and the "commit per concern" constraint (conversion
in `dabb2eabe`, FR-1.3 parser delegation in `cce9a5ca4`, independently revertable).
No scope creep found (`main.go`, `libs/atlas-rest/`, processor/provider/entity files
all untouched — verified via `git log -p --name-only` over the range, filtered to
non-atlas-mounts paths: empty).

## Findings

### PASS — Commit 1 (`dabb2eabe`): alias block matches brief and library contract

`rest/handler.go` after commit 1 replaces the hand-rolled `HandlerDependency`,
`HandlerContext`, `GetHandler`, `RegisterHandler` with aliases over
`server.HandlerDependency` / `server.HandlerContext` / `server.GetHandler` /
`server.RegisterHandler`, matching the brief's target snippet verbatim
(`services/atlas-mounts/atlas.com/mounts/rest/handler.go:1-19` at `dabb2eabe`).
`CharacterIdHandler`/`ParseCharacterId` bodies are byte-identical to before
(deferred to commit 2 per Step 1), confirmed via `git show dabb2eabe -- .../handler.go`.

Cross-checked against the actual library contract (not just the brief's snippet,
since correctness depends on it):
- `libs/atlas-rest/server/register.go:11` — `RegisterHandler(l) func(si) func(name, handler) http.HandlerFunc` — matches the alias call site `rest.RegisterHandler(l)(si)` in `resource.go:19`. No `db` curry level in the library signature, consistent with the brief's registration-line transform.
- `libs/atlas-rest/server/context.go:14,31,43` — `HandlerDependency`, `HandlerContext`, `GetHandler` shapes match usage; `HandlerDependency` has no `DB()` method in the library version, which is fine because Shape A removes the `d.DB()` call from the handler body entirely and closes over `db` instead — confirmed the converted handler body no longer references `d.DB()` (`mount/resource.go:29`).
- `libs/atlas-rest/server/register.go:11-24` chain is `RetrieveSpan → ParseEnvironment → ParseTenant`, and the log fields are `logrus.Fields{"originator": handlerName, "type": "rest_handler"}` (`register.go:15`) — identical text to the deleted hand-rolled chain's field set, confirmed by diff (the removed lines in `handler.go` carried the same literal). Tracing span name (`handlerName` passed straight through) also unperturbed. This is the FR-4.1 intended behavior change (`ParseEnvironment` newly present) called out by the global constraints; nothing to flag since it's an accepted, documented deviation, not silently introduced.

### PASS — Shape A transform on `handleGetMountForCharacter`

`mount/resource.go:26-48` (post-conversion): `handleGetMountForCharacter(db *gorm.DB) rest.GetHandler` wraps the original body unchanged except `d.DB()` → `db` at the `NewProcessor` call (`resource.go:29`). This is exactly Shape A per `global-constraints.md:27-41`, using the DB only directly in the body (no closure-owned sub-closure to check separately). Selection rule satisfied: `d.DB()` did appear in the body, so currying was correct (not a Shape C candidate).

### PASS — Curry drop at `InitResource`

`resource.go:16-24`: `registerGet := rest.RegisterHandler(l)(si)` (dropped `(db)`), and the handler arg gained `(db)`: `handleGetMountForCharacter(db)` at `resource.go:20`. `InitResource`'s own signature `func(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer` is untouched — confirmed unchanged pre/post diff (no hunk touches that line).

### PASS — No local `db` collision

No local variable named `db` existed in the handler body before conversion (grep of the pre-image body shows only `d.DB()` calls, no `db :=` declaration) — no rename-to-`tx` was required, and none was done. Correct per the "leave alone unless colliding" rule.

### PASS — Import pruning, both commits

Commit 1: `context`, `github.com/jtumidanski/api2go/jsonapi`, `gorm.io/gorm` dropped from `rest/handler.go` (all became unused after the struct/func bodies were replaced by aliases) — confirmed via `git show dabb2eabe -- handler.go` diff; `go build ./...` for the module exits 0 (Go treats unused imports as compile errors, so a clean build is direct proof).
`resource.go` keeps `gorm.io/gorm` since `handleGetMountForCharacter(db *gorm.DB)` and `InitResource(...)(db *gorm.DB)` both still reference it.

### PASS — Commit 2 (`cce9a5ca4`): FR-1.3 parser delegation is behavior-preserving

`ParseCharacterId` now delegates to `server.ParseIntId[uint32](l, "characterId", next)`.
Library implementation (`libs/atlas-rest/server/id_parser.go:16-26`) does
`strconv.Atoi(mux.Vars(r)[varName])`, logs `l.WithError(err).Errorf("Error parsing %s as integer", varName)` on failure, writes a bare `w.WriteHeader(http.StatusBadRequest)`, and calls `next(T(value))(w, r)` on success — functionally identical to the deleted hand-rolled body (same `mux.Vars` lookup, same `strconv.Atoi`, same bare 400, same success path), modulo the log message text differing ("Error parsing %s as integer" vs. "Unable to properly parse characterId from path.") which is expected and not a status/shape change. `CharacterIdHandler` named type deleted; grep confirms zero remaining references anywhere in the service before deletion (matches the report's claim, re-verified independently — `grep -rn "CharacterIdHandler" services/atlas-mounts` in the post-image returns nothing outside historical commits). `strconv` and `github.com/gorilla/mux` correctly pruned from `handler.go`'s imports in the same commit.

### PASS — No consumer breakage

`main.go:100` — `AddRouteInitializer(mount.InitResource(GetServer())(db))` — is unchanged across the range (not in the diff, confirmed via `git diff --stat` showing only the two rest/mount files) and remains valid since `InitResource`'s external signature didn't change.

### PASS — Build and test gate

`cd services/atlas-mounts/atlas.com/mounts && go build ./...` exits 0; `go test ./...` — all packages `ok` (`atlas-mounts`, `.../mount`, kafka consumer packages), none touching the REST layer since there's no resource test for this service (confirmed by brief and independently: `atlas-mounts/rest` and `atlas-mounts/tasks` report "no test files"). Consistent with the brief's stated limitation — `go build` is the only gate that can catch a signature-shape defect here, and it passed. This is a real (if weak) verification: an unused import or a mismatched curry level would fail `go build`, and both commits' `go build` runs are independently reproducible from the committed diffs above.

## Not evaluable

- **Runtime/HTTP-level behavior of the FR-4.1 `ParseEnvironment` insertion** (e.g., does a request with an unknown `ENVIRONMENT` header actually 400 for this specific route) cannot be exercised without a resource test or a running instance; the library's own `handler_test.go`/`context_test.go` own that contract per the global constraints, and this task correctly does not add a new test. Flagged as not evaluable rather than assumed passing — the pilot's only executable evidence is the type-checker (`go build`), which cannot observe response bodies or status codes.

## Task quality

The pilot does what it needed to do: it validates the Shape-A recipe end to end (bare
`d.DB()` handler → curried constructor, `InitResource` curry drop, alias block,
optional FR-1.3 delegation) with a clean two-commit history that a later `git revert`
can peel independently. The implementer's report is accurate and does not overstate
what was verified — it explicitly names "no resource test exists" as the gate's limit
rather than papering over it. No shortcuts, no edits to files outside the declared
scope, no test edited to force a pass (none existed to edit).

## Verdict

APPROVED — both commits match the brief line-for-line, the transform is applied
correctly per the Shape A selection rule, no forbidden files were touched, the
alias/delegation targets were cross-checked against the actual `libs/atlas-rest`
contracts (not just the brief's snippets) and match, and build/test evidence is
consistent with what the report claims.
