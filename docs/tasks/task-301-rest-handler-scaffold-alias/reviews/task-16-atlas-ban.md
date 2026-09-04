# Review: Task 16 — atlas-ban REST handler scaffold alias

**Range:** `aa33395f4..4db1055c7`
**Commits:**
- `9f56518` refactor(atlas-ban): alias rest scaffolding, close db over handler constructors
- `4db1055` refactor(atlas-ban): delegate id parsers to shared server parsers

**Module:** `services/atlas-ban/atlas.com/ban`

## Scope

Files touched (confirmed via `git diff --name-only aa33395f4..4db1055c7`):
- `services/atlas-ban/atlas.com/ban/ban/resource.go`
- `services/atlas-ban/atlas.com/ban/history/resource.go`
- `services/atlas-ban/atlas.com/ban/report/resource.go`
- `services/atlas-ban/atlas.com/ban/rest/handler.go`

No other files changed. This matches the brief's stated scope exactly (`main.go`, `libs/atlas-rest/`, and all `_test.go` files are absent from the diff).

## 1. Zero `d.DB()` remaining, all 15 sites converted

Pre-image `d.DB()` call-site count, verified by direct grep of the pre-image blobs:
- `ban/resource.go`: 7 sites (lines 38, 84, 110, 133, 157, 171, 202)
- `report/resource.go`: 4 sites (lines 43, 45, 69, 103)
- `history/resource.go`: 4 sites (lines 43, 45, 72, 101)

Total = 15, matching the brief.

Post-image: `grep -rn "\.DB()" services/atlas-ban/atlas.com/ban` returns exactly one hit — `report/resource_test.go:279: sqlDB, err := db.DB()` — which is the unrelated gorm `*gorm.DB.DB()` sql.DB accessor in test setup, not `d.HandlerDependency.DB()`. **PASS.**

## 2. Shape A applied consistently

All 15 sites, across all three resource files, are of the form `func handleX(db *gorm.DB) rest.GetHandler` / `rest.InputHandler[RestModel]` that closes over `db` and calls `NewProcessor(d.Logger(), d.Context(), db)...` instead of `d.DB()`. Verified by full read of `ban/resource.go`, `report/resource.go`, `history/resource.go` post-image.

`InitResource` in all three files drops `(db)` from the `RegisterHandler`/`RegisterInputHandler` chain (`register := rest.RegisterHandler(l)(si)`) and instead threads `db` per-route as `register("handler_name", handleX(db))`. Confirmed this is consistent with `server.HandlerDependency` exposing only `Logger()`/`Context()` (no `DB()`), per the known brief defect noted in the task. **PASS.**

## 3. Commit split is genuinely revertable

`git diff --stat 9f56518..4db1055c7 -- ban/resource.go report/resource.go history/resource.go` (paths relative to module root) returns empty — the second commit touches only `rest/handler.go`. **PASS.**

## 4. `rest/handler.go` matches landed precedent (Task 15, atlas-npc-shops)

Post-image `rest/handler.go`:
```go
type HandlerDependency = server.HandlerDependency
type HandlerContext = server.HandlerContext
type GetHandler = server.GetHandler
type InputHandler[M any] = server.InputHandler[M]
var RegisterHandler = server.RegisterHandler
func RegisterInputHandler[M any](l logrus.FieldLogger) ... { return server.RegisterInputHandler[M](l) }
func ParseBanId(...) { return server.ParseIntId[uint32](l, "banId", next) }
func ParseAccountId(...) { return server.ParseIntId[uint32](l, "accountId", next) }
func ParseReportId(...) { return server.ParseUUIDId(l, "reportId", next) }
```
This is structurally identical to `services/atlas-npc-shops/atlas.com/npc/rest/handler.go` (aliases, `var RegisterHandler`, `RegisterInputHandler` wrapper, `ParseIntId`/`ParseUUIDId` delegation). Import block is fully live: `net/http`, `uuid` (used in `ParseReportId` signature), `jsonapi` (used in `RegisterInputHandler` signature), `logrus`, `server`. No unused imports; `mux`, `strconv`, `io`, `context` from the pre-image are correctly dropped since their only uses (the deleted named `*Handler` types and the old `HandlerDependency`/`RegisterHandler` bodies) are gone.

Minor, non-blocking: unlike npc-shops' handler.go, atlas-ban's version lacks the package-level doc comment explaining the DB-parameterized variant. Not required by the brief; noted as a style parity gap only. **PASS** (non-blocking note).

## 5. Parser delegation ruling

Pre-image (`git show aa33395f4:.../rest/handler.go`) bodies:
- `ParseBanId`: `mux.Vars(r)["banId"]` → `strconv.Atoi` → `uint32(value)`, bare. → **should delegate to `server.ParseIntId[uint32]`.** Delegated correctly; `BanIdHandler` type deleted.
- `ParseAccountId`: `mux.Vars(r)["accountId"]` → `strconv.Atoi` → `uint32(value)`, bare, identical shape. → **should delegate to `server.ParseIntId[uint32]`.** Delegated correctly; `AccountIdHandler` type deleted.
- `ParseReportId`: `mux.Vars(r)["reportId"]` → `uuid.Parse`, bare. → **should delegate to `server.ParseUUIDId`.** Delegated correctly; `ReportIdHandler` type deleted.

All three parsers were bare lookups with no extra business logic (no extra validation, no alternate error branching beyond the standard "log + 400"), so full delegation to the shared `server.Parse*Id` helpers is correct for all three — no case where the implementer should have left a body byte-identical instead. **PASS.**

## 6. Deleted `*Handler` type names — zero remaining references

`grep -rn "BanIdHandler\|AccountIdHandler\|ReportIdHandler" services/atlas-ban/atlas.com/ban` (code and comments) returns no hits. **PASS** — no repeat of the Task 13 stale-doc-comment defect.

## 7. Untouched surfaces

- `main.go`: not in the diff. Verified `InitResource(GetServer())(db)` call sites for `ban`, `history`, `report` at lines 86-88 are unchanged and still compatible — `InitResource`'s external signature (`func(si) func(db) server.RouteInitializer`) was not altered by this refactor, only its internals.
- `libs/atlas-rest/`: not in the diff.
- Test files: not in the diff (`git diff --name-only` shows only the 4 non-test files). `go test ./...` (see below) confirms they still pass unmodified.

**PASS.**

## 8. Build / test / format verification (run independently)

```
$ go build ./...        # clean, no output/errors
$ gofmt -l .             # clean, no output
$ go test ./...
ok  	atlas-ban	0.028s
ok  	atlas-ban/ban	0.092s
ok  	atlas-ban/character	0.051s
ok  	atlas-ban/chat	0.063s
ok  	atlas-ban/history	0.084s
ok  	atlas-ban/kafka/consumer/report	0.056s
ok  	atlas-ban/report	0.077s
(others: no test files)
```
All pass. **PASS.**

## Route sanity check (ParseAccountId reuse)

`ParseAccountId` is called from exactly one site, `history/resource.go:96`, wired to the route `router.PathPrefix("/history").Subrouter()` + `/accounts/{accountId}` (`history/resource.go:23`). The mux var name matches. No other route in the module references `accountId`, so the "may be reached by more than one route" caution in the brief does not apply here — single call site confirmed by grep.

## Findings

None blocking. One non-blocking style note (item 4: missing package doc comment relative to the npc-shops precedent — not required by brief, cosmetic only).

## Not evaluable

None — full review surface (4 changed files, both commits, build/test/gofmt) was independently exercised.
