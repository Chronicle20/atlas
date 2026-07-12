# task-126 AP/SP Reset Items — Implementation Context

Companion to `plan.md`. Everything below was verified against the worktree source during planning (2026-07-02); file:line references are anchors, not guarantees — re-check before editing.

## Inputs

- PRD: `docs/tasks/task-126-ap-sp-reset-items/prd.md`
- Design (authoritative): `docs/tasks/task-126-ap-sp-reset-items/design.md`
- Plan: `docs/tasks/task-126-ap-sp-reset-items/plan.md` (17 tasks)

## Architecture in one paragraph

Channel decodes the new `ItemUsePointReset` sub-body (To then From, trailing updateTime on non-updateTimeFirst layouts — IDA-verified per version), pre-validates every rule except the job pool-minimum, and creates a `point_reset` saga `[destroy_asset, transfer_ap|transfer_sp]`. atlas-character (`TRANSFER_AP`) and atlas-skills (`TRANSFER_SP`) validate authoritatively and emit success (`STAT_CHANGED` / `SP_TRANSFERRED`) or a typed ERROR event; the orchestrator maps those to StepCompleted(true/false), failure triggers a reverse-walk that re-awards the destroyed item and emits saga-failed with the service's error code (detail rides in `Reason`). Channel renders pink text + enable-actions on pre-validation failure directly and on saga failure via a new `point_reset` branch in `handleFailedEvent`.

## Key files (verified)

| Concern | File | Notes |
|---|---|---|
| Cash-item handler | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` | Handler lines 25-114; `updateTimeFirst = GMS && major>=95` (line 32); FieldEffect arm (lines 60-108) is the saga-assembly template; `GetCashSlotItemType` PointReset branch lines 141-149 (5050001→24, 5050000/2/3/4→23); fall-through warn line 110; `writer.Producer` param currently discarded |
| Codec pattern | `libs/atlas-packet/cash/serverbound/item_use_field_effect.go` | Encode value receiver / Decode pointer receiver; `NewX(updateTimeFirst)` |
| Fixture style | `libs/atlas-packet/cash/serverbound/shop_operation_buy_test.go` | `// packet-audit:verify packet=... version=... ida=0x...` markers; `pt.Variants`, `pt.CreateContext`, `pt.RoundTrip`; hex-literal exact-bytes tests |
| candidatesFromFName | `tools/packet-audit/cmd/run.go:280` (func), CWvsContext sender cases ~1831-1841 | Add case for `CWvsContext::SendConsumeCashItemUseRequest` + `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` (registry primary fname) |
| Saga lib | `libs/atlas-saga/model.go` (Types 9-27, Actions 38-161), `payloads.go` (RebalanceAP family 197-223), `unmarshal.go` (default 498-505) | Second UnmarshalJSON switch in orchestrator `saga/model.go` — BOTH need cases |
| Orchestrator dispatch | `.../saga-orchestrator/saga/handler.go:701-864` GetHandler; `handleResetStats` 2220-2234 is the transfer template | Async actions return nil; completion via status events |
| Acceptance table | `.../saga-orchestrator/saga/event_acceptance.go:96-201` + `event_acceptance_test.go` allActions (12-41) | Coverage tests enforce table + unmarshal entries |
| Compensation | `.../saga-orchestrator/saga/compensator.go:170-197` dispatch; PetEvolution reverse-walk 1053-1140 | `DestroyAsset → compP.RequestCreateItem(txId, charId, templateId, qty, time.Time{})`, qty 0→1 |
| Step result | orchestrator `saga/model.go:707` `Step.Result() map[string]any`; `StepCompletedWithResult` processor.go:337-358 | ErrorCode threading channel: error handlers store `{errorCode, errorDetail}`; compensator reads it |
| Error-code precedent | orchestrator `transport/processor.go:14-124`, handler.go:2280-2316 | `FailedStatusEventProvider(..., errorCode, reason, stepId)` |
| Character entity | `services/atlas-character/.../character/entity.go:49` `HpMpUsed int` col `hpmp_used` | NO migration needed |
| Build() bug | `character/model.go:400-430` omits `hpMpUsed` (CloneModel line 257 keeps it; `SetHpMpUsed` 432-435) | Must fix first — corrupts the FR-6 gate |
| Distribute-AP arms | `character/processor.go:865-886` | Already increment HpMpUsed; growth helpers cap at MaxHp>=30000 or HpMpUsed>9999 (941-1005) — NOT used by the reset path |
| RebalanceAP (template) | `character/processor.go:1872-1921` | Canonical buffered-emit + `ExecuteTransaction` + `WithTransaction(tx).GetById()` + `dynamicUpdate(tx)(mods...)(c)` idiom |
| statChangedProvider | `character/producer.go:249-264` | Hardcodes `ExclRequestSent: true`; meso error provider 203-216 is the ERROR template |
| Character REST | `character/rest.go` | `hpMpUsed` already exposed (line 28/105/139) — no REST change |
| Skills processor | `services/atlas-skills/.../skill/processor.go` | `WithTransaction` exists on Impl (86-93) but NOT on the interface; `Update` 148-178 re-reads then `dynamicUpdate`; `ByIdProvider(characterId, id)` |
| Macro package | `.../skills/macro/` | NO WithTransaction (add); `Update(mb)(txId, worldId, charId, []Model)` = delete-then-recreate in one tx; `SkillId1/2/3` typed `skill.Id`; topic `STATUS_EVENT_TOPIC_SKILL_MACRO` |
| Skill events | `.../skills/kafka/message/skill/kafka.go` | UPDATED carries level+masterLevel+expiration; NO ERROR type exists today; envelope has SkillId |
| Channel saga creation | `.../channel/saga/processor.go:31-33` | Kafka `COMMAND_TOPIC_SAGA`, not REST; alias block `saga/model.go:9-71` |
| Channel failed-event | `.../channel/kafka/consumer/saga/consumer.go:78-142` | Only storage branch exists; `kafka/message/saga/kafka.go` has SagaType/ErrorCode consts |
| Pink text | `writer.WorldMessagePinkTextBody("", "", msg)` + `chatpkt.WorldMessageWriter` | Working example `kafka/consumer/party_quest/consumer.go:117` |
| Enable actions | `statpkt.NewStatChanged(make([]statpkt.Update,0), true)` | Example `kafka/consumer/consumable/consumer.go:76` |
| Macro login push | `.../channel/kafka/consumer/session/consumer.go:320-339` | `charpkt.CharacterSkillMacroWriter` + `packetmodel.NewMacro(...)`; channel has NO macro status consumer today |
| Channel character model | `.../channel/character/model.go` | `hpMpUsed` field line 42 but NO accessor (add); `skills []skill.Model` line 57, `SkillById` 259-266; load via `cp.GetById(cp.SkillModelDecorator)(id)` (processor.go:65-70, 149-155) |
| Data skill client | `.../channel/data/skill/` | `GetById(uint32)` → `Model.Effects()`; max level = `len(Effects())`; resource `data/skills/%d` |
| Seed templates | `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | `CharacterCashItemUseHandle` present ONLY in 83 (line ~409, 0x4F) + 84; absent from 87/95/jms |
| Registry opcodes | `docs/packets/registry/gms_v87.yaml:2321` (0x52), `gms_v95.yaml:2539` (0x55), `jms_v185.yaml:2316` (0x47) | v92: no registry row, no IDB → parked |
| constants: job | `libs/atlas-constants/job/model.go` | `Is` 41-45, `IsBeginner` 65-67, `IdFromSkillId` 52-55, `IsFourthJob` 57-63 (EXISTS); `Advancement` is NEW (Task 1); `job.Id` uint16; Cygnus/Aran/Evan stage consts constants.go:1145-1228 |
| constants: skill | `libs/atlas-constants/skill/` | `Id` uint32; `IsPointResetExcluded` is NEW (Task 2) |
| constants: item | `libs/atlas-constants/item/constants.go:73,127` | `ClassificationPointReset = 505`, `GetClassification` |

## Load-bearing decisions (frozen at plan time)

1. **Saga shape B** (design §3): `[destroy_asset, transfer]`, reverse-walk compensation. Double-use race is safe (second destroy fails first); no inverse-transfer math anywhere.
2. **`database.ExecuteTransaction` is a no-op** (`libs/atlas-database/transaction.go:9-18` — `isTransaction` is true even for a root DB). atlas-skills `TransferSp` therefore wraps in gorm-native `p.db.Transaction(...)`. atlas-character `TransferAP` keeps the RebalanceAP idiom (single `dynamicUpdate` = one SQL UPDATE, atomic regardless).
3. **Error threading**: service ERROR events carry `{Error, Detail}` → orchestrator error handlers `StepCompletedWithResult(false, {"errorCode","errorDetail"})` → `compensatePointReset` emits `EmitSagaFailed(errorCode, reason=errorDetail)` → channel `pointreset.ErrorMessage(ErrorCode, Reason)`. `ErrorMessage` falls back to generic text when the detail isn't an ability name.
4. **Validate-then-apply** in TRANSFER_AP: running values, source applied before target validation — avoids Cosmic's leaked-source-decrement bug and handles From==To naturally (From==To HP→HP is deliberately NOT net-zero: −take +gain).
5. **Naming**: `TransferAP`/`TransferSP` (lib convention, cf. `RebalanceAP`), action strings `transfer_ap`/`transfer_sp`, saga type `point_reset`.
6. **SP target row may not exist** → treat as level 0 / masterLevel 0; create at level 1 on success; 4th-job target with masterLevel 0 rejects `SKILL_AT_CAP`.
7. **Evan rejected** (`job.Advancement` returns −1 for 2200–2218 → `WRONG_TIER`); tier-0 skills excluded; policy tables live ONLY in atlas-character (`character/point_reset.go`); channel checks floor/gate/caps but never the min-pool tables.
8. **Dead check**: enable-actions only, no pink text (Cosmic parity).
9. **Pool cap check** is reject-at-cap (`>= 30000` rejects) then clamp the gain to 30000.

## Task dependency order

1→2→3 independent; 4 after 3 (may amend the codec); 5 independent; 6→7→8 (character chain); 9→10 (skills chain, needs 1+2); 11 after 5+8+10 (message shapes); 12 independent; 13 after 1+2+12; 14 after 5+13; 15 after 3+5+12+13+14; 16 independent; 17 last.

## Risks / escalation triggers

- **Task 4 (IDA)**: the To/From order and trailing updateTime are Cosmic hypotheses. IDA instance set rotates — `list_instances` and match binary names first; use checked-in exports for versions without a live IDB. Unresolvable fname = STOP AND ASK (never substitute/fake). jms audit dir is `docs/packets/audits/jms_v185` — pass `--audit-dir` explicitly.
- **Task 16 Step 2**: if StatChanged/WorldMessage/CharacterSkillChange/CharacterSkillMacro writers are missing from a v87/95/jms template, STOP — a new writer opcode needs its own IDA verification.
- **Mock processors**: adding `WithTransaction` to the skill/macro Processor interfaces breaks any mock implementations — grep and update in the same commit.
- **Live tenants**: new handler opcodes require the config PATCH + atlas-channel restart (Task 16 deployment.md); seed templates only affect new tenants.
- Signatures marked "adjust to actual" in plan code (channel `SkillById` return shape, macro builder methods, `macro.NewProcessor` params, stat.Type constant names) must be read from source before use — the plan flags each site.

## Verification gates (Task 17)

Per-module `go test -race` / `go vet` / `go build` (constants, packet, saga, character, skills, channel, orchestrator, packet-audit); `go run ./tools/packet-audit matrix --check`; `tools/redis-key-guard.sh`; `docker buildx bake atlas-character atlas-skills atlas-channel atlas-saga-orchestrator atlas-configurations`.
