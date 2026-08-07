# Backend Audit — character-rankings LEADERBOARD feature (Go diff only)

- **Service Path:** services/atlas-rankings/atlas.com/rankings
- **Guidelines Source:** backend-dev-guidelines skill (file-responsibilities.md, ai-guidance.md, patterns-multitenancy-context.md, patterns-rest-jsonapi.md, testing-guide.md)
- **Date:** 2026-07-24
- **Git Range:** aacbd6569..HEAD (aba3d894b), Go files only
- **Build:** PASS (`go build ./...` in rankings module)
- **Tests:** targeted `go test ./ranking/... ./character/... ./tasks/... -count=1` — all packages `ok` (full-suite race run already executed upstream per task brief; not repeated here)
- **Overall:** NEEDS-WORK

## Scope

Diff touches: `character/{model,rest,processor_test}.go`, `ranking/{administrator,builder,compute,compute_test,entity,entity_test,leaderboard_rest(new),model,processor,processor_test,provider,provider_test(new),resource,resource_test}.go`, `tasks/recompute_test.go`. 18 files, 447 insertions / 12 deletions, purely additive (new `LeaderboardProvider`/`GET /rankings` leaderboard read path plus `Name`/`Level`/`JobId` display-field threading).

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Immutable model | Private fields + getters + Builder | PASS | `ranking/model.go:9-30` — `name`, `level`, `jobId` are unexported with value-receiver getters (`Name()`, `Level()`, `JobId()`); `ranking/builder.go:10-54` — `Builder.Set*` fluent setters + `Build()` construct the immutable `Model`. |
| Processor shape | Interface+Impl, `NewProcessor(l, ctx, db)` | PASS | `ranking/processor.go:21,46,54` — `type Processor interface`, `type ProcessorImpl struct`, `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `ranking/processor.go:54` — `l logrus.FieldLogger`, not `*logrus.Logger`. |
| DOM-10 | Test DB registers tenant callbacks | PASS | `ranking/entity_test.go:22` — `database.RegisterTenantCallbacks(logrus.New(), db)` inside `testDatabase(t)`, reused by every new test (`provider_test.go`, `resource_test.go`, `processor_test.go`). |
| Tenant scoping — leaderboard read path | Relies on GORM tenant callback via `db.WithContext(ctx)`, no unscoped query | PASS | `ranking/processor.go:87-90` — `LeaderboardProvider` builds `p.db.WithContext(p.ctx)` before calling `byWorldPagedEntityProvider`; `ranking/provider.go:67-77` — the provider only adds `world_id`/`job_category` `Where` clauses, no explicit tenant filter (correctly relies on the callback per patterns-multitenancy-context.md:17-26). Positive proof via test: `ranking/provider_test.go:565-590` `TestByWorldPagedTenantIsolation` seeds two tenants with the same `WorldId: 0` and asserts tenant A's page contains only its own row. |
| Tenant scoping — single-char lookup + RestModel | Unchanged | PASS (unchanged) | `ranking/resource.go:144-170` `handleGetRankingForCharacter` and `ranking/rest.go` are untouched by this diff (confirmed via `git diff` — no hunks in `rest.go`; `resource.go` diff only adds the new `handleGetLeaderboard` function and one route line). |
| DOM-21 | Shared `atlas-constants` types, none reinvented | PASS | `ranking/builder.go:6-7`, `ranking/entity.go:233-234`, `ranking/model.go` all import `github.com/Chronicle20/atlas/libs/atlas-constants/{job,world}` and use `world.Id` / `job.Id` directly for the new `Level`/`JobId` fields — no new byte-classification type or local `job.Id`/`world.Id` reimplementation introduced. |
| Pagination | `paginate.ParseParams` + `server.MarshalPaginatedResponse` | PASS | `ranking/resource.go:62,83` (diff lines) — `paginate.ParseParams(q, paginate.DefaultPageSize, paginate.MaxPageSize)` and `server.MarshalPaginatedResponse[[]LeaderboardRestModel](...)`, mirroring the documented pattern in patterns-rest-jsonapi.md:15. |
| Pagination — 400 on missing/invalid `filter[worldId]` | PASS | `ranking/resource.go:39-49` — empty `rawWorld` → `server.WriteBadRequest(...)`; `strconv.ParseUint` failure → `server.WriteBadRequest(...)`. Verified by test `ranking/resource_test.go:740-749` `TestLeaderboardRequiresWorldId` asserting `http.StatusBadRequest`. |
| Pagination — empty world → 200 empty page | PASS | Test `ranking/resource_test.go:755-779` `TestLeaderboardEmptyWorldReturnsEmptyPage` asserts `http.StatusOK`, zero `data` items, `meta.total == 0` for `filter[worldId]=5` (unseeded world). |
| DOM-09 | Transform errors handled, no swallowed errors | PASS | `ranking/resource.go:75-80` — the `model.SliceMap(...)()` result's `err` is checked before proceeding; every other error branch in the new/touched handler code (`ranking/resource.go:44,54,63,69,76`) is checked, none use `_, _ :=`. |
| DOM-27 | Transient DB errors → `WriteErrorResponse`, not bare 500 | PASS | `ranking/resource.go:69-73` (DB read) and `:76-80` (transform) both call `server.WriteErrorResponse(d.Logger())(w)(err)`, not a raw `w.WriteHeader(http.StatusInternalServerError)`. Classifier registration confirmed pre-existing at `main.go:48-54` (`server.RegisterTransientErrorClassifier` composing `database.IsTransientConnectionError` + `database.CountTransient`), unmodified by this diff. |
| DOM-14/DOM-15 | Handler delegates to processor only, no direct provider/DB calls | PASS | `ranking/resource.go:68` — `handleGetLeaderboard` calls only `NewProcessor(...).LeaderboardProvider(...)()`; no `db.Create`/`db.Save`/provider function called directly from `resource.go`. |
| DOM-18 | JSON:API interface on new REST model | PASS | `ranking/leaderboard_rest.go:314-324` (diff) — `LeaderboardRestModel` implements `GetName()`, `GetID()`, `SetID()`. |
| JSON:API resource correctness — shared type name | Noted, not a defect (plan-mandated per task brief) | INFO | `ranking/rest.go:20-22` `RestModel.GetName()` and `ranking/leaderboard_rest.go` `LeaderboardRestModel.GetName()` both return `"rankings"`. Per the audit brief this collision is explicitly plan-mandated; recorded for visibility, not scored as a finding. |
| No TODO/stub/501 | PASS | `grep -n "TODO\|FIXME\|panic(\"not implemented" ` over the diff's changed files returns no matches; `tasks/recompute_test.go`'s new `LeaderboardProvider` method on `fakeProcessor` (`tasks/recompute_test.go:34-36`) is a genuine test-double interface satisfaction, not a production stub. |
| **FILE-02** | **`RestModel` + `Transform` in `rest.go`** | **FAIL (Important)** | `ranking/leaderboard_rest.go` (new file) defines `LeaderboardRestModel` (a JSON:API `RestModel`-shaped struct with `GetName`/`GetID`/`SetID`) and `func TransformLeaderboard(m Model) (LeaderboardRestModel, error)` — both responsibilities file-responsibilities.md:102-115 assigns to `rest.go` ("Define `RestModel` struct... Implement `Transform(Model) (RestModel, error)`"). The existing `ranking/rest.go` is untouched by this diff. Unlike the explicitly-permitted `processor_<group>.go` split for large Processors, no guideline documents a `*_rest.go`-per-projection split as an allowed exception — see the audit's closed loophole clause: "the ONLY thing that turns a deviation into a non-finding is a guideline that explicitly documents it." Splitting a second REST projection into a sibling file this cleanly (single responsibility, well-commented rationale) is a materially smaller deviation than a collapsed catch-all file, but it is still a FILE-02 placement violation as written. |

## Security Review

Not applicable — atlas-rankings is not an auth/token-handling service; SEC-01..04 skipped.

## Summary

### Blocking (must fix)
- **FILE-02** — `LeaderboardRestModel` and `TransformLeaderboard` live in `services/atlas-rankings/atlas.com/rankings/ranking/leaderboard_rest.go` instead of `ranking/rest.go`. file-responsibilities.md designates `rest.go` as the single home for RestModel struct definitions and `Transform`/`TransformSlice` functions in a package; no guideline documents a per-projection rest-file split as an allowed exception (the documented split exception is `processor_<group>.go` for Processor methods only). Fix: move `LeaderboardRestModel`, its JSON:API methods, and `TransformLeaderboard` into `ranking/rest.go` (or get the guideline amended to explicitly permit multi-projection rest-file splits before merging this pattern again).

### Non-Blocking (should fix)
- `ranking/resource.go`'s leaderboard handler wraps `TransformLeaderboard` in an unnecessary anonymous closure (`model.SliceMap(func(m Model) (LeaderboardRestModel, error) { return TransformLeaderboard(m) })(...)`) where `TransformLeaderboard` already satisfies `model.Transformer[Model, LeaderboardRestModel]` directly — passing it by name (`model.SliceMap(TransformLeaderboard)(...)`) would match the documented composition idiom in ai-guidance.md:236 more directly. Cosmetic; not a guideline violation.
- The two `RestModel` types in this package (`ranking/rest.go`'s `RestModel` and the new `LeaderboardRestModel`) both return `"rankings"` from `GetName()`. Per the task brief this is plan-mandated and not scored as a defect, but flagging again here since it compounds with the FILE-02 split — if the fix folds `LeaderboardRestModel` into `rest.go`, the two types and their shared resource-type string will sit side by side, which is the point at which a future reviewer should confirm api2go route/resource-type disambiguation actually works as intended (both types are query-response-only, never used as PATCH/POST targets, so no observed runtime collision — but worth a second look when consolidating).
