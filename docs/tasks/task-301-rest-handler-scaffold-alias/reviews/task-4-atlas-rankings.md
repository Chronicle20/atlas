# Review: Task 4 — atlas-rankings REST handler scaffold alias

Commit range: `e023ad2ee..1531c1a9a`
Commits: `45e9564cf` (conversion), `1531c1a9a` (FR-1.3 delegation)

## Scope

`git diff --stat e023ad2ee..1531c1a9a`:

```
services/atlas-rankings/atlas.com/rankings/ranking/resource.go | 172 +++++++++++----------
services/atlas-rankings/atlas.com/rankings/rest/handler.go     | 101 +-----------
2 files changed, 95 insertions(+), 178 deletions(-)
```

Matches the brief's file list exactly (`rest/handler.go`, `ranking/resource.go`). `ranking/resource_test.go` diff is empty — confirmed untouched (`git diff e023ad2ee..1531c1a9a -- .../resource_test.go` produced no output).

## Findings

### PASS — Step 1: scaffolding alias (`rest/handler.go`)

`services/atlas-rankings/atlas.com/rankings/rest/handler.go:1-21` now reads:

```go
type HandlerDependency = server.HandlerDependency
type HandlerContext = server.HandlerContext
type GetHandler = server.GetHandler
var RegisterHandler = server.RegisterHandler
```

exactly matching the brief's Step 1 template. `InputHandler`/`ParseInput`/`RegisterInputHandler` were omitted (brief explicitly required this — service has zero `rest.RegisterInputHandler` callers). Imports pruned to `net/http`, `logrus`, `server` (commit 1); `context`/`io`/`jsonapi`/`gorm.io/gorm` removed. Verified via `git diff` hunk — only these four import lines survive plus `ParseCharacterId`'s remaining needs.

### PASS — Step 2: `ranking/resource.go` Shape A/C conversion

All 3 `d.DB()` sites converted to Shape A (`handleGetLeaderboard(db *gorm.DB) rest.GetHandler`, `handleGetRankingsForCharacters(db *gorm.DB) rest.GetHandler`, `handleGetRankingForCharacter(db *gorm.DB) rest.GetHandler`), each closing over `db` in place of `d.DB()`. `handleMissingIds` correctly left as Shape C (bare `rest.GetHandler`, no DB).

`InitResource` (`ranking/resource.go:21-31`) drops the `(db)` currying level: `rest.RegisterHandler(l)(si)` (was `(l)(db)(si)`), and each Shape-A handler receives `(db)` at its registration site: `handleGetLeaderboard(db)`, `handleGetRankingsForCharacters(db)`, `handleGetRankingForCharacter(db)`.

Diffed the handler bodies with scaffolding noise filtered out (indentation-only changes, `func handle…`/`return func(d…`/`(db)` call sites excluded) — every remaining `+`/`-` line pairs 1:1 as pure reindentation or the `d.DB()` → `db` substitution. No logic, error-handling, or response-shape changes beyond the mechanical conversion.

### PASS — Step 5/FR-1.3: `ParseCharacterId` delegation (commit `1531c1a9a`)

`rest/handler.go:19-21`:

```go
func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}
```

matches the brief verbatim. `CharacterIdHandler` type deleted; `strconv`/`mux` imports pruned (confirmed absent from post-image). `server.ParseIntId` (`libs/atlas-rest/server/id_parser.go:16-26`) is behaviorally identical to the deleted hand-rolled body: `strconv.Atoi(mux.Vars(r)[varName])`, log + 400 on error, `next(T(value))(w, r)` on success. No caller of `ParseCharacterId` needed to change — its exported signature (`func(logrus.FieldLogger, func(uint32) http.HandlerFunc) http.HandlerFunc`) is preserved; only the internal `CharacterIdHandler` named-type indirection was removed, which is transparent to `ranking/resource.go:158` (`rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {...})`).

### PASS — Commit separation and Global Constraints

Two commits as required: `45e9564cf` touches `rest/handler.go` + `ranking/resource.go` (conversion); `1531c1a9a` touches only `rest/handler.go` (FR-1.3 delegation), confirmed via `git show --stat` on each SHA. No `main.go` touched (`git diff --stat -- '**/main.go'` empty). No `libs/` files touched (`git diff --stat -- libs/` empty).

### PASS — Build and test

```
cd services/atlas-rankings/atlas.com/rankings && go build ./...   # exit 0, no output
go vet ./...                                                       # exit 0, no output
go test ./... -count=1
ok  	atlas-rankings          0.024s
ok  	atlas-rankings/character 0.037s
ok  	atlas-rankings/configuration 0.029s
ok  	atlas-rankings/ranking  0.070s
?   	atlas-rankings/rest     [no test files]
ok  	atlas-rankings/tasks    0.010s
ok  	atlas-rankings/tenant   0.023s
```

Ran with `-count=1` (bypassing cache) to confirm a genuine rerun, not a stale cached pass. `ranking/resource_test.go` is unedited and green.

## Not evaluable

None. The full diff surface (2 files, 373 raw diff lines) was read in full; no file exceeded the slice-first threshold in a way that required partial reading, and both touched files were read completely.

## Verdict rationale

The two commits are a clean, mechanical application of the established alias-scaffolding + Shape A/C recipe validated on prior services in this plan. Scaffolding types are aliased 1:1 to `libs/atlas-rest/server`, `db` is correctly closed over per-handler rather than threaded through `HandlerDependency`, `InitResource` registration call sites match the new arity, `ParseCharacterId` delegates to `server.ParseIntId` with an equivalent error contract, and the untouched `resource_test.go` passes on a forced rerun. No deviation from the brief found.

---

verdict: APPROVED
artifact: docs/tasks/task-301-rest-handler-scaffold-alias/reviews/task-4-atlas-rankings.md
scope_confirmed: reviewed commits 45e9564cf and 1531c1a9a in full (rest/handler.go, ranking/resource.go); confirmed resource_test.go untouched; ran go build/vet/test -count=1 in services/atlas-rankings/atlas.com/rankings
blocking: 0
non_blocking: 0
not_evaluable: 0
