# Backend Audit Recheck — task-290-cosmic-map-action-parity (shard 1 of 2)

- **Service Path(s):** services/atlas-saga-orchestrator, libs/atlas-saga, services/atlas-quest, services/atlas-character
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-03
- **Prior audit:** `docs/tasks/task-290-cosmic-map-action-parity/backend-audit-shard1/audit.md` (range `3f37fe6da..9c2b9be13`, NEEDS-WORK)
- **This recheck's range:** `9c2b9be13..HEAD` (fix commits `4f790da4d`, `0213dda9f`, `ff153af74`, `b4a76759b`, `1a45308c3`, `7a5989ed2`, `dd88cd13b`, in shard-1 scope; `a5948b5fc`/`655ae8e63`/`722e87d97`/`d26497798`/`ffba7aded` are FIX-BE2A/2B, out of shard-1 scope — confirmed by `git show --stat` against none of this shard's four module paths)
- **Build:** PASS (all four modules)
- **Tests:** PASS (all four modules, `go test ./... -count=1`; `atlas-character/pending_change` 354.994s, `atlas-character/kafka/consumer/character` 54.491s, rest fast; no `FAIL` lines anywhere)
- **Overall:** NEEDS-WORK (one prior blocking finding remains open; nothing new)

## Build & Test Results

```
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator : go build OK; go test ./... -count=1 OK (grep for "FAIL" across full output: 0 matches)
libs/atlas-saga                                               : go build OK; go test ./... -count=1 OK
services/atlas-quest/atlas.com/quest                          : go build OK; go test ./... -count=1 OK
services/atlas-character/atlas.com/character                  : go build OK; go test ./... -count=1 OK (pending_change 354.994s, kafka/consumer/character 54.491s, rest fast)
```

## Per-finding disposition

### 1. DOM-01 — `area_info/builder.go` non-validating `Build()`

**CLOSED.** `services/atlas-character/atlas.com/character/area_info/builder.go:41-49`:
```go
func (b *builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	return Model{...}, nil
}
```
`Build()` itself now returns `(Model, error)` and enforces the `characterId` invariant, matching DOM-01's literal bar ("a `Build()` that enforces invariants" — `file-responsibilities.md:190`). All three in-package callers (`processor_test.go:71,79,135,142,178`) already handle the two-value return; build/test confirmed green.

### 2. DOM-01 — `medal_map/builder.go` non-validating `Build()`

**STILL OPEN.** `services/atlas-quest/atlas.com/quest/medal_map/builder.go:44-49` — `Build()` is **unchanged** from the prior audit: it still returns a bare `Model` with no invariant check. The fix instead added a *new*, separate method:
```go
// builder.go:54-61
func (b *builder) BuildWithValidation() (Model, error) {
	if b.characterId == 0 { return Model{}, ErrMissingCharacterId }
	if b.questId == 0 { return Model{}, ErrMissingQuestId }
	return b.Build(), nil
}
```
DOM-01's rule text is specifically about `Build()` ("a `Build()` that enforces invariants" — `file-responsibilities.md:190`), not about the package having *some* validating method under *some* name. `Build()` does not enforce invariants; `BuildWithValidation()` does, but it is dead code — confirmed by grep: its only caller in the package is `entity.go:31`'s `Make()`, which itself has zero callers anywhere in the package (`grep -rn "medal_map\.Make\|[^.]Make(" medal_map/*.go` excluding the declaration returns nothing). The package's actual write path, `administrator.go:14` (`recordIfAbsent`), constructs an `entity{...}` literal directly and never touches the builder at all, so no code path — live or dead — actually gets invariant enforcement in practice. This is a **partial, workaround fix**, not a closure: the flagged method is unchanged, and the new method that *would* satisfy the rule's intent is unreachable.

Severity unchanged from prior audit: **Important (blocking)**, per Mindset ("structural / File-Responsibilities violation defaults to Important").

### 3. DOM-02/DOM-03 — `area_info/entity.go` missing `ToEntity()`/`Make()`

**CLOSED.** `services/atlas-character/atlas.com/character/area_info/entity.go:26-32` (`Make(e entity) (Model, error)`, calling `Build()` which now validates) and `:35-42` (`func (m Model) ToEntity() entity`). Both used: `provider.go:11,15,24` call `Make`; `administrator.go:20` calls `Make`; `administrator.go:9` calls `m.ToEntity()`.

### 4. DOM-02/DOM-03 — `medal_map/entity.go` missing `ToEntity()`/`Make()`

**CLOSED as far as the literal ask goes**, with a caveat already triaged as non-regression per this task's brief: `entity.go:31-38` adds `Make(e entity) (Model, error)`, `entity.go:41-48` adds `ToEntity()`. Neither has an in-package caller (`Make` calls `BuildWithValidation` per #2 above; `ToEntity` is unused — the write path builds `entity{...}` literals directly in `administrator.go:14-20`). Per the task's own pre-triage ("quest/medal_map's new Make/ToEntity have no in-package caller — added to satisfy the audit's literal DOM-02/03 ask... out of scope to refactor"), this is not re-reported as a new finding, but it is the same root cause as the still-open DOM-01 item above: the builder/Make/ToEntity round-trip exists on paper but the real write path never uses it.

### 5. FILE-05 — `area_info/administrator.go` readers misplaced

**CLOSED.** `services/atlas-character/atlas.com/character/area_info/administrator.go` now holds only the write, `upsert` (`administrator.go:9-21`). The two readers moved to `provider.go:8-15` (`getByCharacterIdAndArea`) and `provider.go:17-30` (`getAllByCharacterId`).

### 6. FILE-05 — `medal_map/administrator.go` reader misplaced, no `provider.go`

**CLOSED.** `services/atlas-quest/atlas.com/quest/medal_map/administrator.go` now holds only the write, `recordIfAbsent` (`administrator.go:14-28`). A `provider.go` now exists (`provider.go:9-16`, `countByCharacterAndQuest`), holding the one reader that was previously in `administrator.go`.

### 7. FILE-02/FILE-03/FILE-06 — `saga-orchestrator/field/processor.go` collapse

**CLOSED.** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/field/` now has three files: `processor.go` (Processor/ProcessorImpl/NewProcessor/ResetField only), `rest.go` (`ResetFieldInputRestModel` + `GetName`/`GetID`), `requests.go` (`getBaseRequest`/`requestResetField`). Verified by reading all three files in full — no responsibility bleeds across files.

### 8. FILE-02/FILE-03/FILE-06 — `saga-orchestrator/reactor/processor.go` collapse (growing)

**CLOSED.** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/` now has `processor.go` (Processor/ProcessorImpl/NewProcessor/HitReactorByName/ResetReactors/ShuffleReactors/produceHitCommand/Command types only — no RestModel or request-builder left in it), `rest.go` (`ReactorRestModel`, `ResetReactorsInputRestModel`, `ShuffleReactorsInputRestModel`), `requests.go` (`getReactorsBaseRequest`, `requestReactorsByName`, `requestResetReactors`, `requestShuffleReactors`). Verified by reading all three files in full.

### 9. EXT-01 — `saga-orchestrator/area_info/rest.go` missing relationship stubs

**CLOSED.** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/area_info/rest.go:26-30` — `SetToOneReferenceID`/`SetToManyReferenceIDs` added on `*RestModel`.

### 10. EXT-01 — `saga-orchestrator/quest/medal_map_rest.go` and `quest_data_rest.go` missing relationship stubs

**CLOSED.** `medal_map_rest.go:50-54` and `quest_data_rest.go:36-40` — both `medalMapRestModel` and `questDataRestModel` gain `SetToOneReferenceID`/`SetToManyReferenceIDs`.

### 11. EXT-01 — `saga-orchestrator/npc_spawn/rest.go` missing relationship stubs

**CLOSED** — and on closer inspection this appears to have been a **false positive in the prior audit**, not a fix that landed in this recheck's range: `git blame` on `npc_spawn/rest.go:55-60` (`SetToOneReferenceID`/`SetToManyReferenceIDs`) attributes those lines to `c35e9e446`, which is an ancestor of `9c2b9be13` — i.e. the stubs were already present at the commit the prior audit reviewed. `git show 9c2b9be13:.../npc_spawn/rest.go` confirms both methods exist at that exact commit. Current state is unambiguously CLOSED either way; noted here only because "FIX-BE1C's commit message says 'npc_spawn already carried the stubs on this branch'" corroborates the same conclusion independently.

### 12. EXT-02 — no test file in `area_info`, `quest` (top level), `field`, `npc_spawn`, `reactor`

**CLOSED for all five.** Each package now has a `*_test.go` at top level, and each uses `httptest.NewServer` (not a mock) to exercise the real HTTP-calling implementation:
- `area_info/processor_test.go:31,86` — `TestPutIssuesPutWithInfo`, `TestPutPropagatesUpstreamFailure`, 2 `httptest.NewServer` call sites.
- `quest/processor_test.go:82,163` — `TestRequestExplorerQuest_NewlyRecordedResolvesThreshold`, `TestRequestExplorerQuest_AlreadyRecordedSkipsProgressWrite`, 4 `httptest.NewServer` call sites (via `TestMain`/shared harness).
- `field/processor_test.go:37,100` — `TestResetFieldIssuesPostWithDifficulty`, `TestResetFieldPropagatesUpstreamFailure`, 2 `httptest.NewServer` call sites.
- `npc_spawn/processor_test.go:43,116` — `TestSpawnNpcCarriesSpawnIfAbsent`, `TestSpawnNpcPropagatesUpstreamFailure`, 2 `httptest.NewServer` call sites.
- `reactor/processor_test.go:50,104,134,189` — `TestHitReactorByNameResolvesAndProducesHit`, `TestHitReactorByNameNoMatchIsAnError`, `TestResetReactorsCarriesMinState`, `TestShuffleReactorsIssuesPost`, 4 `httptest.NewServer` call sites.

All packages report `ok` under `go test ./... -count=1`.

### 13. No-rule-ID — Handler struct-copy bug (`WithNpcSpawnProcessor`/`WithDropsProcessor`/`WithFieldProcessor`)

**CLOSED**, and more thoroughly than the task brief's framing suggested. The task brief said the sweep "fixed 28 methods, not the 3 named" — actual count confirmed by grep: **all 34** `func (h *HandlerImpl) With*` methods in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` now use the shallow-copy form (`c := *h; c.<field> = <val>; return &c`), including the three originally named:
- `handler.go:607-611` `WithNpcSpawnProcessor`
- `handler.go:615-619` `WithDropsProcessor`
- `handler.go:623-627` `WithFieldProcessor`

`grep -c "c := \*h"` against the file returns 34, matching the 34 `With*` method signatures — no `&HandlerImpl{...}` struct-literal construction remains anywhere in the file (confirmed by a `&HandlerImpl{` grep across all `With*` bodies: zero matches). This is also the item the task brief flagged as "tracked open item" (`WithNpcSpawnProcessor`) — it is not open; it is fixed, same as its two siblings.

A new regression test, `TestHandlerImpl_WithProcessorMethods_PreserveOtherFields` (`saga/handler_test.go:2092-2171+`), exercises all 34 `With*` methods (34-case table) asserting each preserves an unrelated anchor field (`areaInfoP` or `charP`) set by a prior builder call — this is exactly the defect class the bug represented, now regression-guarded. Test passes (`go test ./saga/...` green, no FAIL).

### 14. No-rule-ID — Stale exhaustiveness test (`allActions` missing 9 of 10 new Actions)

**CLOSED.** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go:14-64` (`allActions`) now includes all 11 new/changed Action constants from the prior audit's list — confirmed present by direct read: `WarpToMap` (line 19), `ClearSkill` (line 23), `ExplorerQuest` (line 24), `PlaySound` (line 28), `ChangeMusic`/`BoatEffect` (line 30), `SpawnNpc` (line 31), `ClearDrops`/`ResetReactors`/`ShuffleReactors`/`ResetField` (line 32). `TestAcceptanceTable_EveryActionRepresented` passes.

### 15. DOM-20 — table-driven test conversions

**CLOSED across the board**, verified individually:
- `area_info/processor_test.go` (character): all 4 funcs (`TestUpsertAreaInfoReplacesWholeString` :58, `TestGetByAreaMissingReturnsEmpty` :98, `TestAreaInfoIsPerCharacter` :122, `TestAreaInfoIsTenantScoped` :165) now use `tests := []struct{...}` + `t.Run`.
- `medal_map/processor_test.go` (quest): all 4 funcs (`TestRecordMedalMapDeduplicates` :58, `TestRecordMedalMapCountsDistinctMaps` :90, `TestMedalMapsArePerQuest` :119, `TestMedalMapsArePerCharacterAndTenant` :147) converted.
- `saga-orchestrator/drops/drops_test.go`: both funcs (`TestClearDropsIssuesDeleteToTheRightField` :24, `TestClearDropsPropagatesUpstreamFailure` :71) converted.
- `saga-orchestrator/saga/handler_test.go`: all 4 originally-flagged funcs (`TestHandleUpdateAreaInfo` :1842, `TestHandleExplorerQuest_Routing` :1910, `TestHandleExplorerQuest_WritesProgressWhenNewlyRecorded` :1969, `TestHandleExplorerQuest_SkipsProgressWhenNotNewlyRecorded` :2032) converted — each wraps a single-case `tests := []struct{ name string }{...}` + `t.Run`, satisfying the literal rule text (`tests := []struct{...}` + `t.Run` — `audit-checklist.md:113`) though the table has only one row (no data variation); the rule as written grades the pattern, not the row count.
- `libs/atlas-saga/unmarshal_test.go`: verified programmatically — all 54 `func Test...` bodies now contain both `tests := []struct` and `t.Run(`, up from 1/53 at the prior audit.

## New blocking violations introduced by the fix commits themselves

None found. The two highest-risk surfaces named in the brief were checked directly:
- **field/reactor file splits** (`ff153af74`): read all six resulting files (`field/{processor,rest,requests}.go`, `reactor/{processor,rest,requests}.go`) in full — clean one-responsibility-per-file split, no new collapse, no orphaned/duplicated code.
- **28+ `With*` builder sweep** (`4f790da4d`): read every `With*` method signature (34 total) and confirmed each uses `c := *h; c.<field> = val; return &c`; no method reverted to or introduced a `&HandlerImpl{...}` literal.

`go build ./...` and `go test ./... -count=1` are green on all four modules with zero `FAIL` lines, so no compile-time or test regression was introduced either.

## Applicability

Unchanged from the prior audit (this recheck evaluates only the fix commits layered on top of the same four modules; no new package classification was needed). See prior audit's Applicability table for the full disposition.

## Not evaluable from the diff

- Whether `RequestExplorerQuest`'s `strconv.Atoi(questData.EndRequirements.InfoEx[0])` silent-fallback-to-0 (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/processor.go`) matches atlas-data's actual `infoEx` wire format — unchanged since the prior audit; the fix commits in this recheck's range did not touch atlas-data, and confirming this would require reading atlas-data's quest REST model outside this shard's scope.
- Whether any caller outside this shard's four modules chains `With*Processor` calls in a way that would have triggered the (now-fixed) struct-copy bug historically — moot now that the bug is closed repo-scope-independently, but was not itself re-checked by a repo-wide grep in this recheck.

## Summary

### Blocking (must fix)

- DOM-01: `services/atlas-quest/atlas.com/quest/medal_map/builder.go:44-49` — `Build()` itself still has no invariant validation; the new `BuildWithValidation()` (`builder.go:54-61`) is unreached dead code, and the package's real write path (`administrator.go:14-28`) never goes through the builder at all.

### Non-Blocking (should fix)

None carried forward as newly non-blocking; the prior audit's one non-blocking item ("zero test coverage for 10 of 12 new saga handler methods") was not in scope for this recheck (the brief's finding list did not include it) and was not re-verified here.

### Closed (verified against current HEAD)

DOM-01 (area_info), DOM-02/DOM-03 (area_info, medal_map), FILE-05 (area_info, medal_map), FILE-02/03/06 (field, reactor), EXT-01 (area_info, quest medal_map+quest_data, npc_spawn), EXT-02 (area_info, quest, field, npc_spawn, reactor), Handler struct-copy bug (all 34 `With*` methods), stale `allActions` exhaustiveness gap, DOM-20 (area_info, medal_map, drops, saga/handler_test.go, libs/atlas-saga/unmarshal_test.go).
