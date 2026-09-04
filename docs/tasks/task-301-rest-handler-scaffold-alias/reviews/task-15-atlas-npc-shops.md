# Task 15 Review — atlas-npc-shops REST handler scaffold alias

**Commit range:** `c63ded980..aa33395f4` (2 commits: `b076b2b2c`, `aa33395f4`)
**Module:** `services/atlas-npc-shops/atlas.com/npc`
**Files in scope:** `rest/handler.go`, `commodities/resource.go`, `shops/resource.go`

## Summary

`git diff --stat`:

```
commodities/resource.go |  82 ++--
rest/handler.go         | 116 +-----
shops/resource.go       | 446 +++++++++++----------
```

## Checks

### 1. Zero `d.DB()` remaining
PASS. `grep -rn "\.DB()" services/atlas-npc-shops/atlas.com/npc/` returns only `test/database.go:47: sqlDB, err := db.DB()`, which is the unrelated gorm `*gorm.DB.DB()` (returns `*sql.DB`), not `HandlerDependency.DB()`. No handler is double-converted.

### 2. Shape A applied consistently to all 12 sites
PASS — counted 12 handler-constructor functions: `commodities/resource.go:31 handleGetCommoditiesByItem`, and 11 in `shops/resource.go` (`handleGetShop`, `handleAddCommodity`, `handleUpdateCommodity`, `handleRemoveCommodity`, `handleDeleteAllCommodities`, `handleDeleteAllShops`, `handleGetShopCharacters`, `handleGetAllShops`, `handleCreateShop`, `handleUpdateShop`, plus `decoratorsFromInclude` which is a plain helper, not a registered handler — confirmed 10 registered handlers in shops + the helper). All follow `func XxxHandler(db *gorm.DB) rest.GetHandler` / `rest.InputHandler[M]`, closing over `db` in the returned closure. `InitResource` in both files calls each constructor with `(db)` per call site (`shops/resource.go:26-39`, `commodities/resource.go:22`), not chained through the registrar. Confirmed `server.HandlerDependency` (`libs/atlas-rest/server/context.go:14-17`) has no `DB()` method — Shape A is the only viable pattern here.

### 3. Commit split is genuinely revertable
PASS. `git diff --stat b076b2b2c..aa33395f4 -- commodities/resource.go shops/resource.go` is EMPTY — commit 2 (`aa33395f4`) touches only `rest/handler.go` (the parser delegation), confirming no parser change leaked into commit 1 and no non-parser change leaked into commit 2.

### 4. `rest/handler.go` matches landed precedent
PASS. Compared byte-for-byte structure against `services/atlas-party-quests/atlas.com/party-quests/rest/handler.go` (Task 14): identical alias block (`HandlerDependency`, `HandlerContext`, `GetHandler`, `InputHandler[M]`, `RegisterHandler` var, `RegisterInputHandler` wrapper) and identical `ParseXxxId` delegation pattern to `server.ParseIntId`/`server.ParseUUIDId`. Import block (`net/http`, `uuid`, `jsonapi`, `logrus`, `server`) is fully live — no unused imports, no dead aliases. The npc-shops file additionally carries a doc-comment banner (lines 1-11) explaining the DB-parameterized variant, present in both the pre- and post-image, unchanged by this diff — not a defect.

### 5. Parser delegation ruling
PASS. Pre-image (`git show c63ded980:.../rest/handler.go`) shows:
- `ParseNpcId`/`NpcIdHandler`: bare `mux.Vars` + `strconv.Atoi` body, no extra logic beyond error-wrapping and cast.
- `ParseCommodityId`/`CommodityIdHandler`: bare `mux.Vars` + `uuid.Parse` body.

Both are exactly the "bare lookup" shape design.md §4.3 says should delegate. Post-image correctly delegates both to `server.ParseIntId[uint32](l, "npcId", next)` and `server.ParseUUIDId(l, "commodityId", next)`, and deletes the named `NpcIdHandler`/`CommodityIdHandler` types. No parser was left that should have been deleted, and no parser doing extra work was wrongly collapsed (there were none of the latter kind in this file).

### 6. Deleted `*Handler` type names have zero remaining references
PASS. `grep -rn "NpcIdHandler\|CommodityIdHandler" services/atlas-npc-shops/` returns nothing — no code or comment reference survives.

### 7. Untouched: main.go, atlas-rest libs, all `_test.go`
PASS. `git diff --stat c63ded980..aa33395f4 -- .../main.go 'libs/atlas-rest/**' 'services/atlas-npc-shops/**/*_test.go'` is empty. `commodities/resource_test.go` and `shops/resource_paginate_test.go` are untouched and green (see below). `main.go:103-104` confirms `InitResource(GetServer())(db)` call shape for both `shops` and `commodities` is unchanged by this refactor — consistent with Shape A (only the inner handler constructors changed).

### 8. Build / test / gofmt
PASS, run from module root (`services/atlas-npc-shops/atlas.com/npc`):
- `go build ./...` — clean, no output.
- `go test ./...` — all packages `ok`, including `atlas-npc/commodities` (0.125s) and `atlas-npc/shops` (0.759s).
- `gofmt -l .` — no output (all files formatted).

## Findings

None. All 8 checks pass with direct evidence.

## Not evaluable

None — the review surface (the two commits' diff plus the referenced `libs/atlas-rest/server/context.go` and `services/atlas-party-quests` precedent) was fully evaluable.

## Verdict

APPROVED.
