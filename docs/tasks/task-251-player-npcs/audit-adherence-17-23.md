# Plan Audit — task-251-player-npcs (Tasks 17-23)

**Plan Path:** docs/tasks/task-251-player-npcs/plan.md
**Audit Date:** 2026-08-22
**Branch:** task-251-player-npcs
**Base Branch:** main (merge base `5f299e4bb`)
**Scope:** Tasks 17-22 of plan.md, plus Task 23a/23b/23c — the operator-ruled Kafka-contract
extension (FR-8.3/FR-6.3) recorded in `progress.md` (Session 9), which supersedes the plan.md
text for FR-6.3/FR-8.3 delivery mechanism.

## Executive Summary

All seven items in this range (Tasks 17-22 plus the 23a/23b/23c split) are DONE with direct
diff evidence: new files, matching test functions, and wiring at the cited call sites. The
operator's ruling to extend `COMMAND_TOPIC_PLAYER_NPC`/`EVENT_TOPIC_PLAYER_NPC_STATUS` rather
than narrow the PRD is fully implemented across all three services (`atlas-player-npcs`,
`atlas-saga-orchestrator`, `atlas-messages`), including the two Task 22 non-blocking findings
folded into 23b (`mock.go`→`processor.go` rename with compile-time assertion; the corrected
doc comment). All seven affected Go modules build clean and test clean (`go build ./...` and
`go test ./... -count=1`, 0 FAIL across all modules).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 17 | Kafka — messages, producer, consumers (`atlas-player-npcs`) | DONE | `kafka/message/playernpc/kafka.go`, `kafka/message/character/kafka.go`, `kafka/consumer/playernpc/consumer.go` (+`consumer_test.go`, `TestPlayerNpcCommandConsumer`), `kafka/consumer/character/consumer.go` (+`consumer_test.go`, `TestLevelChangedConsumer`), `playernpc/producer.go`; wired in `main.go:9-10,96-98` (`npcconsumer.InitConsumers`, `charconsumer.InitConsumers`) |
| 18 | `atlas-channel` — `playernpc/` read client | DONE | `playernpc/{model,builder,rest,requests,processor}.go` + `rest_test.go` with `TestRestModel_Unmarshal`, `TestExtract_EquipmentOrder`, `TestForEachInMap_RequestsByMapAndWorld` |
| 19 | `atlas-channel` — spawn, broadcast, controller exclusion | DONE | `kafka/consumer/map/player_npc.go` + `player_npc_test.go` (`TestSpawnPlayerNpcForSession`, `TestExitProducesNoControllerHandoffForPlayerNpc`); `kafka/consumer/playernpc/{consumer,kafka}.go` + `consumer_test.go` (`TestPlayerNpcStatusConsumer`); `main.go:55,252,560` consumer registration; `main.go:683-684` writer registration (`npccb.NpcImitatedDataWriter`, `npccb.NpcRemoveWriter`) |
| 20 | `atlas-tenants` — `player-npcs` configuration resource | DONE | `configuration/{rest,processor,provider,kafka,resource}.go` extended; `configuration/mock/processor.go`; `player_npc_config_handler_test.go` (`TestPlayerNpcConfigHandlerWireRoundTrip`) |
| 21 | `atlas-messages` — GM commands | DONE | `command/playernpc/commands.go` (+`commands_test.go`, `TestPlayerNpcCommands` with all 8 plan sub-cases incl. "failure reported back"), `kafka/message/playernpc/kafka.go`, `main.go` registry adds. Note: the "failure reported back" case as originally landed asserted the publish-layer error only (per progress.md's documented Task 21 blocking finding); Task 23c superseded this with a real status-consumer path (see below) — original test still passes and was not removed. |
| 22 | Conversation-engine condition and operation | DONE | `libs/atlas-saga/{model,payloads,unmarshal,validation}.go` (condition + action constants, `DeployPlayerNpcPayload`); `query-aggregator/validation/{model,rest,processor,context}.go` + `model_test.go` (`TestCanSpawnPlayerNpcCondition`, 4/4 cases); `query-aggregator/playernpc/{processor,requests,rest}.go`; `saga-orchestrator/saga/{handler,model,event_acceptance,character_extractor,producer}.go` + `handler_test.go` (`TestDeployPlayerNpcAction`); docs updated in `query-aggregator/docs/{rest,domain}.md` and `docs/npc_conversation_conversion_spec.md` |
| 23a | `atlas-player-npcs` — wire contract, `CodeFor`, outcome emission | DONE | Commit `2ae390710`. `kafka/message/playernpc/kafka.go` extended with `TransactionId`/`Requester` envelope fields and `COMMAND_SUCCEEDED`/`COMMAND_FAILED` event types; `playernpc/errors.go` (+`errors_test.go`) adds `CodeFor`; `kafka/consumer/playernpc/consumer.go` emits outcome on all three consumer arms (`+120` lines), asserted by `TestPlayerNpcCommandConsumerOutcomeEmission` |
| 23b | `atlas-saga-orchestrator` — mirror, event-driven `deploy_player_npc` | DONE | Commit `d92561b85`. `kafka/message/playernpc/kafka.go` mirror; `kafka/consumer/playernpc/consumer.go` (+`consumer_test.go`, `errorcode_result_test.go`) new status consumer; `saga/handler.go`/`producer.go` thread `TransactionId`; `saga/event_acceptance.go:139-144,225-231,326,493-497` moves `DeployPlayerNpc` out of the fire-and-forget table into `{EventKindPlayerNpcCommandSucceeded, EventKindPlayerNpcCommandFailed}`, with `errorCode` result carrier verified by `TestHandleCommandFailedEventThreadsErrorCode`-style coverage in `errorcode_result_test.go`. Both Task 22 non-blocking findings folded in: `playernpc/mock/mock.go`→`playernpc/mock/processor.go` (rename confirmed via `git diff --find-renames`) with `var _ playernpc.Processor = (*ProcessorMock)(nil)` at `processor.go:16` and a new `processor_test.go`; the doc comment at `kafka/message/playernpc/kafka.go:1-11` now scopes to "the subset ... that this service produces or consumes" rather than the prior overstated "field for field" claim |
| 23c | `atlas-messages` — mirror, `requester`, pink-text on outcome | DONE | Commit `ee5640d83`. `kafka/message/playernpc/kafka.go` mirror (+75 lines); `kafka/consumer/playernpc/consumer.go` (+171 lines, new) + `consumer_test.go` (+150 lines, new) — first status-event consumer in this service; `command/playernpc/commands.go` sets `requester` on both GM commands; `main.go` wires the new consumer |

**Completion Rate:** 9/9 items in range (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. No task was left unchecked without evidence, and no task was silently
narrowed. FR-8.3/FR-6.3's originally-plan.md'd mechanism (Task 17's fire-and-forget consumer,
Task 21's publish-layer-only pink text) was deliberately superseded by an operator ruling
recorded in `progress.md` ("Operator ruling — FR-8.3 / FR-6.3: extend the Kafka contract"),
which chose option (A), extend the contract, over option (B), narrow the PRD. That ruling is
treated as the authoritative spec for this range per the dispatch instructions, and it is fully
implemented (23a/23b/23c above) — not merely documented as a gap.

## Build & Test Results

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| `atlas-player-npcs` | PASS | PASS | 0 FAIL; `kafka/consumer/{character,playernpc}` both green |
| `atlas-channel` | PASS | PASS | 0 FAIL |
| `atlas-tenants` | PASS | PASS | 0 FAIL |
| `atlas-messages` | PASS | PASS | 0 FAIL |
| `atlas-saga-orchestrator` | PASS | PASS | 0 FAIL; `saga` and `saga/mock` green |
| `atlas-query-aggregator` | PASS | PASS | 0 FAIL; `validation` and `validation/mock` green |
| `libs/atlas-saga` | PASS | PASS | 0 FAIL |

(Module-local `go build ./... && go test ./... -count=1` per module, matching the plan's own
per-task verify commands. This shard did not run the repo-wide flagless `tools/verify.sh` —
that is the controller's Final-gate responsibility per plan.md and is already recorded as
Gate 35 PASS at `d92561b85` in progress.md.)

## Overall Assessment

- **Plan Adherence:** FULL (for Tasks 17-23a/23b/23c)
- **Recommendation:** READY_TO_MERGE (for this range; overall branch readiness depends on the
  other shards' findings and the Final-gate sequence)

## Action Items

None required for Tasks 17-23. Carry-forward items already tracked in `progress.md` (nested-
`equipment` JSON:API decode gap from Task 18 review, Task 19 not-evaluable multi-pod broadcast
and object-id-collision items, two operator hand-backs from Task 8, and the live-broker
end-to-end check for the new outcome path from Task 23) are pre-existing and outside this
shard's task range; they are noted here for the controller's awareness but are not new findings
of this audit.
