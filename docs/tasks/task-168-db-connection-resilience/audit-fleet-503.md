# Backend Audit — Fleet-wide 503 Adoption (task-168), DOM-26 focus

- **Scope:** commits 3f3e3b07b6..1a99a5a066 (32 commits, 120 files), diff
  `.superpowers/sdd/review-3f3e3b07b6..1a99a5a066.diff`
- **Guidelines:** `patterns-resilience.md` (DOM-26), backend-dev-guidelines
- **Date:** 2026-07-12
- **Overall:** CONDITIONAL PASS — one Important fall-through double-write in atlas-saga-orchestrator; everything else is a faithful, correct migration.

## DOM-26 scope completeness — PASS

- 32 classifiers registered in `main.go` files (HEAD): 31 in this diff + atlas-inventory (Task 11 reference, pre-existing). All compose `IsTransientConnectionError` + `CountTransient` per the reference (`libs/atlas-rest/server/error.go:31,48`).
- DB-backed set (services calling `database.Connect`) = exactly {32 adopted} + atlas-fame. Cross-check: every service still containing a bare `w.WriteHeader(http.StatusInternalServerError)` (atlas-world, atlas-parties, atlas-messengers, atlas-monsters, atlas-reactors, atlas-transports, atlas-summons, atlas-buffs, atlas-doors, atlas-drops, atlas-invites, atlas-chairs, atlas-chalkboards, atlas-portals, atlas-rates, atlas-effective-stats, atlas-query-aggregator) does NOT call `database.Connect` → correctly out of DOM-26 scope.
- atlas-fame: DB-backed but zero 500-writing handlers (absent from bare-500 breakdown) → no classifier, as documented. Correct — a classifier with no `WriteErrorResponse` consumer is dead code.
- 443 `WriteErrorResponse` call sites total across services.

## Mechanical correctness — PASS (except saga, below)

- **err variable is the actual in-scope branched error** at every site: 437 pass `err`; 4 correctly pass the handler's locally-named error (`putErr` at data/wzinput/handler.go:116, `qerr` at data/npc/resource.go:178,229, `terr` at data/monster/resource.go:188, `txErr` at mts/testsupport/resource.go:349); 1 synthesized (npc-conversations, below); plus the maps refactor. Matching local names rather than blindly substituting `err` indicates a careful, non-blind migration.
- **No non-500 branch altered:** 0 removed `WriteHeader(BadRequest|NotFound|Conflict|Forbidden|Unauthorized)` lines in the diff. Only removed non-500 error write is baseline restore's `http.Error(w, err.Error(), code)` (handled correctly, below). 404/400/409/403 branches preserved (e.g. npc/resource.go:6116, mts/listing switch, families switch).
- **Log lines preserved:** existing `d.Logger().WithError(err).Errorf(...)` retained before the new call (WriteErrorResponse also logs Warn — extra log, not a bug).

## Four non-mechanical sites — all CORRECT

1. **atlas-maps** `character/location/resource.go` — `changeCharacterLocation` refactored to `(int, error)`; caller routes only `status == 500` through `WriteErrorResponse(err)` and returns, else `w.WriteHeader(status)`. All 3 internal-error returns carry non-nil err; 404/400/204 return nil err. 5 unit-test call sites updated with correct err assertions (resource_test.go:4373–4469). CORRECT.
2. **atlas-data** `baseline/handler.go` — restore's ambiguous `code` var replaced by explicit 422 branch (`http.Error(...StatusUnprocessableEntity); return`) + `WriteErrorResponse(err)` for the true 500; publish/list `http.Error(...,500)` → `WriteErrorResponse`. Pre-existing 503/403 `http.Error` guards untouched. Removed now-unused `fmt` import. CORRECT.
   - **`map/resource.go:358` nil-err inert case** (handleGetMapFootholdBelow, `fh == nil` branch): `err` is provably nil here (passed the earlier `if err != nil`). `WriteErrorResponse(nil)` → `IsTransientConnectionError(nil)` returns false (`libs/atlas-database/transient.go:29`), so status stays 500 with a JSON:API body. Panic-safe and status-preserving. CORRECT (though passing a live nil is slightly sloppy; behavior is correct).
3. **atlas-tenants** `configuration/resource.go` — 4 handlers' `w.WriteHeader(500)` + `json.NewEncoder(w).Encode(map...)` double-write collapsed to a single `WriteErrorResponse(err)` (lines ~506, 529, 552, 743 in diff). Success-path `json.NewEncoder(w).Encode(result)` retained → json import still used. CORRECT.
4. **atlas-npc-conversations** `ReindexRecipesHandler` — type-assert guard now `WriteErrorResponse(errors.New("processor is not *ProcessorImpl"))` (classifier returns false → stays 500 with body); the `err`-path double-write (`WriteHeader(500)` + `json.Encode`) collapsed to `WriteErrorResponse(err)`. `errors`/`json` imports both still used (json at ValidateConversationHandler:244). CORRECT.

## Findings

### Important

- **atlas-saga-orchestrator `saga/resource.go` — `getAllSagasHandler` and `getSagaByIdHandler` fall through to a second body write.**
  In both GET handlers each `if err != nil { d.Logger()...; server.WriteErrorResponse(d.Logger())(w)(err) }` block has **no `return`** (getAll: lines ~28–38, diff 7902–7913; getById: lines ~52–62, diff 7928–7939). On a processor or transform error the handler writes the JSON:API error document and then falls through to `server.MarshalResponse[...](...)(rms)`, emitting a second JSON document — a malformed `{"errors":[...]}` + `{"data":...}` concatenated body.
  The missing `return` is **pre-existing** (the removed `-` lines show the original `w.WriteHeader(500)` also lacked a return), but the migration **worsened** it: before, the trailing `MarshalResponse` only produced a superfluous `WriteHeader(200)` no-op plus one (empty) body; now it produces a full second JSON document after the error document. The sibling `createSagaHandler` in the same file correctly returns after each `WriteErrorResponse` (diff 7962/7971/7980), confirming the two GET handlers are the outliers.
  **Fix (on-branch):** add `return` after each of the 4 `WriteErrorResponse` calls in `getAllSagasHandler` and `getSagaByIdHandler`. The fleet-503 transform contract itself states a following write after `WriteErrorResponse` is a bug.

### Non-blocking

- `map/resource.go:358` passes a live nil `err` to `WriteErrorResponse` (works, but a synthesized `errors.New("no foothold below position")` would read cleaner and log a meaningful cause). Behavior is correct; cosmetic.

## Verified-clean checks

- No other fall-through: an exhaustive scan for `if`-blocks containing `WriteErrorResponse` without a `return` flagged only saga (genuine), the two terminal DELETE handlers in configurations (templates:146, tenants:136 — error branch is the last statement, no fall-through), inventory:45 (return present after a trailing log), and families:132 (return present one brace-level up). Only saga is a real defect.
- No handler double-writes header+body outside saga.
