# Seam review — quest / area-info / exploration (shard 3 of 3)

Range: `9613e7259..d058fd34a`. Scope: cross-service producer/consumer contracts for
tasks C1 (`set_quest_progress`/`start_quest`), C20/C21 (area-info persistence +
`areaInfo` condition), C22/C23 (`explorer_quest` + `explorationPoint`). Plan
adherence and backend-guidelines checklists already ran on this branch (0 and 28/28
closed respectively) and are not repeated here.

Ground truth pinned by the dispatcher and not re-litigated: quest status ports 1:1
with no offset (`StateNotStarted=0`, `StateStarted=1`, `StateCompleted=2`,
`services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go`);
`map-rien.json`'s `questStatus 21101 == "2"` is correct.

## C1 — `set_quest_progress` / `start_quest` (map-actions → saga → atlas-quest)

**Producer:** `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:612-657`
builds `saga.SetQuestProgressPayload{CharacterId, WorldId, QuestId uint32, InfoNumber
uint32, Progress string}` and `saga.StartQuestPayload{CharacterId, WorldId, QuestId
uint32, NpcId uint32}` from string params, `progress` passed through verbatim (not
parsed as int, per plan).

**Consumer 1 (orchestrator):** `saga/handler.go:2251` (`handleSetQuestProgress`) and
`saga/handler.go:2165` (`handleStartQuest`) are pre-existing/unchanged in this diff
(confirmed no hunks touch these lines) — they already call
`questP.RequestUpdateProgress(... InfoNumber, Progress)` and
`questP.RequestStartQuest(... NpcId, rewards)`. Types agree field-for-field with the
new payload structs (`InfoNumber uint32`, `Progress string`, `NpcId uint32`).

**Consumer 2 (atlas-quest kafka commands):**
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/quest/kafka.go:54-58`
`UpdateProgressCommandBody{QuestId, InfoNumber uint32, Progress string}` — unchanged,
matches `services/atlas-quest/atlas.com/quest/kafka/message/quest/kafka.go:64` and is
consumed by `handleUpdateProgressCommand` →
`quest.NewProcessor(...).SetProgress(..., c.Body.InfoNumber, c.Body.Progress)`
(`services/atlas-quest/atlas.com/quest/kafka/consumer/quest/consumer.go:117-126`),
also unchanged in this diff (`git diff --stat` shows zero touched files under
`services/atlas-quest/atlas.com/quest/quest/` or `kafka/`). No "step" vs
"infoNumber" key mismatch found — the field is named `InfoNumber` end to end and its
`uint32` type is preserved at every hop.

**Test:** `TestExecuteSetQuestProgress`/`TestExecuteStartQuest` and their param-
validation tables in `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go:345-505`
assert the new payload against the wire-visible fields.

**Verdict: PASS.** This task only added the map-actions executor arm; the rest of the
chain was already end-to-end and is unmodified by this branch.

## C20/C21 — area-info persistence and the `areaInfo` condition

**Write path.** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2999-3024`
(`handleUpdateAreaInfo`) calls `h.areaInfoP.Put(payload.CharacterId, payload.Area,
payload.Info)` **before** `h.systemMessageP.UpdateAreaInfo(...)`, matching the plan's
"persist before announce" requirement. `area_info/requests.go:24` PUTs to
`characters/%d/area-info/%d`. Consumer:
`services/atlas-character/atlas.com/character/area_info/resource.go:22-26` registers
exactly `PathPrefix("/characters/{characterId}/area-info")` + `"/{area}"` PUT — paths
agree. Persistence is a full replace, not a merge:
`services/atlas-character/atlas.com/character/area_info/administrator.go:9-21` uses
`clause.OnConflict{... DoUpdates: clause.AssignmentColumns([]string{"info"})}` on a
unique `(tenant_id, character_id, area)` index — matches Cosmic's
`updateAreaInfo` full-string replace, confirmed by
`TestUpsertAreaInfoReplacesWholeString` (plan step 2, present in
`area_info/processor_test.go`).

**Read path.** `services/atlas-query-aggregator/atlas.com/query-aggregator/area_info/requests.go:16-22`
GETs the same `characters/%d/area-info/%d` path. `validation/context.go:374-390`
(`GetAreaInfo`) feeds `validation/model.go:575-585`'s `EvaluateWithContext` arm:

```go
case AreaInfoCondition:
    stored := ctx.GetAreaInfo(uint16(c.referenceId))
    if strings.Contains(stored, c.valueString) {
        actualValue = 1
    } else {
        actualValue = 0
    }
```

— a literal substring test, not a `k=v;` pair parse, exactly as the plan specifies and
as Cosmic's `containsAreaInfo` does. `c.valueString` is populated from
`ValidationConditionInput.ValueString` (`builder.go:319-321`), and both
`FromInput`/`Validate()`/`rest.go` reject `referenceId==0 || valueString==""` with
the specified message (`builder.go:389-390,482-483`; `rest.go:284-285`).

**Wire widening.** `libs/atlas-saga/validation.go` adds `ValueString
string \`json:"valueString,omitempty"\`` — additive, `omitempty`, confirmed carried
end to end: `services/atlas-map-actions/atlas.com/map-actions/validation/model.go:19`
(map-actions' own client-side `ConditionInput`) and
`script/evaluator.go:110` (`ValueString: cond.ValueString()`) both use the identical
JSON key `valueString`.

**Seed.** `deploy/seed/gms/83_1/map-actions/onUserEnter/map-rien.json`'s `areaInfo`
condition carries both `value: "1"` (compared via `Equals` against `actualValue`,
which is `1` when the substring is found) and `valueString:
"miss=o;arr=o;helper=clear"` (the actual substring operand) — the aggregator arm
reads `valueString`, not `value`, for the substring comparison, and `value` only
supplies the expected boolean-as-int outcome. This is the comparison the seed
intends.

**Test asserting the exact comparison:** `TestAreaInfoCondition`
(`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go:2541+`)
is table-driven and includes the load-bearing row — `"rienArrow guard after rienArrow
ran"`: stored `miss=o;arr=o;helper=clear`, `valueString` `miss=o;helper=clear` →
`false` — the case that would trip a naive "parse the k=v pairs and compare sets"
implementation. Confirmed present at `model_test.go:2561-2568`.

**Verdict: PASS.** Both directions of the seam agree in path, field name, type, and
comparison semantics, and a test pins the one case (row 5) that distinguishes a
correct substring test from a wrong pair-parse.

## C22/C23 — `explorer_quest` and `explorationPoint`

**Producer:** `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:699-738`
(`executeExplorerQuest`) builds
`saga.ExplorerQuestPayload{CharacterId, WorldId, ChannelId, QuestId uint32, MapId
_map.Id, AreaName string}`, `MapId` from `f.MapId()` (never a param), matching the
plan's requirement that Cosmic credits the map the character is standing in.

**Orchestrator composition:**
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2207-2248`
(`handleExplorerQuest`) calls `questP.RequestExplorerQuest`, and on
`!result.NewlyRecorded` returns early without writing progress — mirrors Cosmic's
`if (!qs.addMedalMap(...)) return;`. On success it writes
`RequestUpdateProgress(..., result.InfoNumber, strconv.Itoa(int(result.Count)))`.
`RequestExplorerQuest`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/processor.go:88-131`)
force-starts via npc `9000066`, POSTs the medal-map record, and on
`NewlyRecorded` resolves `infoNumber`/`infoEx[0]` from atlas-data.

**Medal-map REST contract.**
Producer (saga-orchestrator) POSTs `characters/%d/quests/%d/medal-maps` with body
`{"mapId": <id>}` (`quest/medal_map_requests.go:23-25`) and reads back
`{"count": uint32, "newlyRecorded": bool}` (`quest/medal_map_rest.go:8-31`).
Consumer (atlas-quest) registers exactly that path
(`services/atlas-quest/atlas.com/quest/medal_map/resource.go:22-24`) and its
`RestModel` uses the identical field names/types
(`services/atlas-quest/atlas.com/quest/medal_map/rest.go:30-33`; the JSON:API stub
`RestModel` fields `count`/`newlyRecorded` match, not `threshold`/`completed` as the
plan's Interfaces section sketched — that is a documented, self-consistent deviation:
threshold/infoNumber are instead resolved separately from atlas-data by the
orchestrator, per `quest/processor.go`'s doc comments and `ExplorerQuestResult`).
Status code: `201` on `NewlyRecorded`, default `200` otherwise
(`medal_map/resource.go:44-46`) — matches plan.

**atlas-data quest-definition contract.** `quest/quest_data_rest.go:7-19`
(`questDataRestModel{EndRequirements questRequirementsRestModel{InfoNumber uint32,
InfoEx []string}}`) against the path `data/quests/%d`
(`quest/quest_data_requests.go:14-22`) matches atlas-data's actual route
(`services/atlas-data/atlas.com/data/quest/resource.go:26`) and REST model field
names verbatim (`services/atlas-data/atlas.com/data/quest/rest.go:27,83-84`:
`endRequirements`, `infoNumber`, `infoEx`).

**explorationPoint ordering hazard — checked and correctly resolved.** The engine is
first-match-wins: `script/processor.go:144-172` iterates `ms.Rules()` in document
order and `return`s on the first matched rule; rule order is guaranteed stable
because the whole document is stored as a single `jsonb` column
(`script/entity.go:20`) and deserialized preserving JSON array order
(`script/entity.go:78`, `155-165`), not re-queried from a per-row table. Commit
`fc3dbd313`'s message documents exactly this finding and the resulting restructure:
`deploy/seed/gms/83_1/map-actions/onUserEnter/map-explorationPoint.json`'s first rule
is `map_104000000` (both `explorer_quest` 29005/"Beginner Explorer" and
`field_effect maplemap/enter/104000000`), placed **before** `beginner_range`
(`>=100000000, <105040300`), which also numerically covers 104000000. Verified by
direct inspection of the ten rules in document order — `map_104000000` is rule 1 of
10, so a character entering map 104000000 gets both operations from a single rule
and never reaches `beginner_range`. All eleven roots are byte-identical
(`diff -q` against all ten non-canonical roots: no drift).

**Tests:**
- `TestExecuteExplorerQuest`/`TestExecuteExplorerQuestParamValidation`
  (`script/executor_test.go:507-560+`) — new contract.
- `TestRequestExplorerQuest_NewlyRecordedResolvesThreshold` /
  `_AlreadyRecordedSkipsProgressWrite` (`quest/processor_test.go:82,163`).
- `TestHandleExplorerQuest_Routing` / `_WritesProgressWhenNewlyRecorded` /
  `_SkipsProgressWhenNotNewlyRecorded` (`saga/handler_test.go:1910,1969,2032`) — the
  ordering/gating contract (force-start → medal-map → conditional progress write) is
  pinned by mock-return-driven table cases, not just a happy path.
- `TestRecordMedalMapDeduplicates`/`_CountsDistinctMaps`/`_ArePerQuest`/`_ArePerCharacterAndTenant`
  (`medal_map/processor_test.go:58,113,142,170`).
- `TestUnmarshalExplorerQuestStep` (`libs/atlas-saga/unmarshal_test.go:2325`).

**Verdict: PASS.** Every hop (map-actions → saga-orchestrator → atlas-quest
medal-map REST → atlas-data quest-definition REST) agrees in path, field name, and
type, and the one genuine cross-cutting hazard (rule-ordering vs. engine semantics)
was identified and correctly resolved with a documented rationale.

## Not evaluable

None. All five task seams in scope were traced hop-by-hop from producer through
every consumer, and each has at least one test asserting the new contract (not just
the old one).

## Summary

No blocking or non-blocking findings. All seams in scope (C1, C20/C21, C22/C23) are
internally consistent end to end, and the tests genuinely pin the new contracts
(including the one row per seam — areaInfo's substring guard, the explorationPoint
rule-ordering restructure — that a weaker implementation would get wrong).
