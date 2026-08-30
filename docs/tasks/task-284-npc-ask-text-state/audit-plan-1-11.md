# Plan Audit — task-284-npc-ask-text-state (Tasks 1-11)

**Plan Path:** docs/tasks/task-284-npc-ask-text-state/plan.md
**Audit Date:** 2026-08-29
**Branch:** task-284-npc-ask-text-state
**Base Branch:** main (commit range 9cd1ec5af..e6a1540cb)
**Scope:** Plan Tasks 1 through 11 only (the critical-path chain). Tasks 12-21 are out of scope for this audit and are not discussed.

## Executive Summary

All 11 tasks in scope are fully implemented and match the plan's file lists, patterns, and validation rules. Every named commit (`bb142468f` through `728d1099d`, plus the mirror-guard/test-only touch-ups) lands exactly the artifact the corresponding plan task describes: the `Text` field on the continue-conversation contract, the `AskTextType`/`AskTextModel`/`AskTextMatchModel` domain types and JSONB codec, the `StateBuilder` setter and two model builders, REST transform/extract in both REST layers, validator rules including the matches[] circular-reference walk, the outbound `TEXT` command emission, the inbound `Continue` branch evaluation, the atlas-channel decode-and-forward fix, the atlas-channel `TEXT` consumer arm, the quest-progress REST client, and the `local:get_quest_progress` operation. `go build ./...` and `go test ./... -count=1` pass clean for all three affected modules (atlas-npc-conversations, atlas-saga-orchestrator, atlas-channel), and both project-specific guards (`npc-conversation-contract-mirror-guard.sh`, `operator-cancel-path-guard.sh`) pass. The three explicitly-approved out-of-plan commits (`7bc963455`, `bbf4c4782`, `e6a1540cb`) are also correctly implemented: the genericAction AND-semantics fix replaces a stubbed `TODO` with a real full-condition loop, and the askText re-prompt fix updates the Task 7 test table in lockstep with the behavior change so no test asserts stale semantics.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Add `Text` to continue-conversation command contract (3 copies) | DONE | `bb142468f`; `Text string \`json:"text"\`` appended identically to `atlas-npc-conversations/.../kafka/message/npc/kafka.go`, `atlas-saga-orchestrator/.../kafka/message/npc/kafka.go`, and `atlas-channel/.../kafka/message/npc/kafka.go`. `tools/npc-conversation-contract-mirror-guard.sh` prints `OK — both copies identical.` |
| 2 | `AskTextType`, `AskTextModel`, `AskTextMatchModel`, JSONB codec | DONE | `e4b9fbc05`; `model.go` adds the const, `StateModel.askText`/`AskText()`, both immutable models with accessors only. `model_json.go` adds Marshal/Unmarshal for both models and the `StateModel` envelope at all 4 sites (marshal struct, unmarshal struct, assembly, tagged read). `model_json_test.go` (222 lines) added with round-trip tests. |
| 3 | `StateBuilder.SetAskText`, `Build` arm, `AskTextBuilder`/`AskTextMatchBuilder` | DONE | `9acf07b8e`; `askText` field added to `StateBuilder`; `b.askText = nil` present in exactly 12 sites (11 sibling setters + `SetAskText` itself, matching the plan's 12-site count); `Build()` arm returns error on nil `askText`; `AskTextBuilder`/`AskTextMatchBuilder` implement all validation rules from the plan's table (`text`, `maxLength>0`, `min<=max`, `contextKey` default `"answer"`, `nextState` required; match `Build()` enforces exactly-one-of `value`/`valueFromContext`). `builder_test.go` (424 lines) added. |
| 4 | REST transform/extract in both REST layers | DONE | `87a356967`; `RestAskTextModel`/`RestAskTextMatchModel` added to `conversation/rest.go` with the tag set the plan specifies (`text`,`defaultText`,`minLength`,`maxLength`,`contextKey,omitempty`,`matches,omitempty`,`nextState,omitempty`); `TransformAskText`/`ExtractAskText` added with switch arms in `TransformState`/`ExtractState`; identical six sites mirrored into `conversation/quest/rest.go`. Round-trip tests added to both `rest_transform_test.go` and a new `conversation/quest/rest_transform_test.go` (`TestTransformRoundTrip_AskTextQuestLayer`), satisfying the plan's requirement for quest-layer coverage even though it landed in a dedicated file per the plan's own fallback instruction. |
| 5 | Validator rules for `askText` | DONE | `6651e3867` + `c3903c39e`; dispatch arm added to `validateState`; `validateAskText` checks `text`/`contextKey`/`maxLength`/`min<=max`/`nextState` required+reference, and per-match exactly-one-of plus `{context.…}` syntax via `ExtractContextValue` (no second regexp introduced) plus reference check. Circular-reference visitor and next-state collector both walk `matches[].nextState` in addition to the fallback `nextState`, matching the plan's explicit requirement. `validator_asktext_test.go` (326 lines) added; follow-up commit clarifies (not weakens) the circular-reference fixture. |
| 6 | Outbound `CommandTextBody`, `SendText`, `processAskTextState` | DONE | `e3b83983b`; `CommandTextBody{DefaultValue, MinLength, MaxLength}` added beside `CommandNumberBody` and mirrored into atlas-saga-orchestrator (guard re-verified: `OK — both copies identical.`); `textConversationProvider` and `SendText` added following the `numberConversationProvider`/`SendNumber` shape; `processAskTextState` added with the `AskTextType` dispatch arm in `ProcessState`. Test-observability seam (`npcSenderProcessorFactory` package var) is a reasonable substitute for the plan's suggested "thread the dependency" approach — achieves the same emission-observability goal without a test-only constructor. |
| 7 | Thread `text` through `Continue`, add `AskTextType` arm | DONE | `fe7e4f96f` (plus the later approved `bbf4c4782` refinement); `Continue` signature gains `text string`; the mock (`conversation/mock/processor.go`) and the single production caller (`kafka/consumer/npc/consumer.go:102`, passing `c.Body.Text`) are both updated. The `AskTextType` arm trims once, then checks bounds, stores into `choiceContext[askText.ContextKey()]`, and walks `Matches()` in declaration order comparing literal or context-resolved values, falling back to `askText.NextState()` — matching the plan's 5-step "order of operations" exactly. The originally-planned "return an error, stay parked on length violation" behavior was superseded by the user-approved `bbf4c4782` re-prompt fix; the corresponding test table in `processor_asktext_test.go` was updated in the same commit (`wantErr`→`wantReprompt`, error-level log assertion → warn-level log assertion) so no stale assertion remains. |
| 8 | atlas-channel — stop discarding the decoded reply | DONE | `4215c3a64`; the commented-out `returnText := ""` and the "TODO set return text" placeholder comment are both deleted (not revived), replaced with `sp.Text()` passed through `ContinueConversation`/`ContinueConversationCommandProvider` into `ContinueConversationCommandBody.Text`. `bodySelection`/`bodyNone` arms correctly pass `""`. New test seam `npcProcessorFunc` added, mirroring an existing precedent (`newFactoryProcessorFunc`) rather than inventing a test-only constructor. `operator-cancel-path-guard.sh` passes. |
| 9 | atlas-channel `TEXT` consumer arm and `announceTextConversation` | DONE | `911212f43`; `CommandTextBody` added to the message package; `handleTextConversationCommand` registered as the fifth `InitHandlers` entry, guarding on `c.Type != conversation2.CommandTypeText`; `announceTextConversation` passes `npcpkt.NpcConversationMessageTypeAskText` directly (not through `getNPCTalkType`), matching the `announceSlideMenuConversation` precedent the plan cites; `getNPCTalkType` gains the `"TEXT"` case regardless. New `consumer_test.go` (113 lines) covers the announced packet model, the type-guard, and `getNPCTalkType("TEXT")`. |
| 10 | Quest-progress client for atlas-npc-conversations | DONE | `d2b16ba49`; new `conversation/quest/progress/{rest,requests,processor}.go` added as a sibling of `quest/status/`, using the same `RootUrlFor(ctx, "QUEST")` convention (no env/deploy change). `RestModel.GetName()` returns `"progress"`, verified byte-for-byte against the server's `atlas-quest/.../quest/progress/rest.go` `GetName()`. `Progress` stays an unparsed string end-to-end. A new `progress.ErrNotFound` sentinel wraps `requests.ErrNotFound` (which does exist in `libs/atlas-rest/requests/get.go:18`), distinguishing 404 from transport error as the plan requires. `rest_test.go` added. |
| 11 | `local:get_quest_progress` operation | DONE | `728d1099d`; `questProgressP` field added to `OperationExecutorImpl` and wired in `NewOperationExecutor`; the `case "get_quest_progress":` arm lives inside `executeLocalOperation`, whose switch operates on `operationType := strings.TrimPrefix(operation.Type(), "local:")` — confirmed the full operation type authors must use is `local:get_quest_progress`, matching the plan's correction. Required-param extraction for `questId`/`contextKey`, `evaluateContextValue` applied to both `questId` and `infoNumber`/`step`, `step`-as-alias-losing-to-`infoNumber` precedence, the 404→store-empty-string→return-nil branch, and the hard-failure branch for any other error are all present exactly as specified. `operation_executor_test.go` gains 222 lines of table-driven cases. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 11 tasks in scope have direct, verifiable evidence in the branch diff.

## Out-of-Plan Commits (approved mid-execution, judged for correctness only)

| Commit | Description | Assessment |
|---|---|---|
| `7bc963455` | genericAction outcome conditions now evaluate with full AND semantics instead of only `Conditions()[0]` | Correctly implemented: replaces a `// TODO` stub with a loop over `outcome.Conditions()` that short-circuits on the first `false` (since evaluation can make remote calls) and reports the specific failing condition. `processor_condition_and_test.go` (254 lines) added. |
| `bbf4c4782` | askText out-of-range input now re-prompts (same state, `Warnf`, no context write) instead of returning a discarded error | Correctly implemented: the `AskTextType` arm in `Continue` now sets `nextStateId = state.Id()` and leaves `choiceContext` unset on a length violation rather than returning an error the sole caller discards. The pre-existing Task 7 test table (`processor_asktext_test.go`) was updated in the same commit to assert the new behavior (`wantReprompt`, `Warnf`, `sendTextCalled`, absent context key) rather than left asserting the superseded error-return behavior. A new `processor_asktext_reprompt_test.go` (216 lines) adds direct coverage. |
| `e6a1540cb` | Documentation-only fix round closing Task 21 review findings (schema `required` array, re-prompt behavior doc, `local:` prefix sweep in prd/design) | Docs-only; no code in scope for this audit. Changes are consistent with what tasks 1-11's actual code does (e.g., `contextKey` genuinely defaults to `"answer"` per `NewAskTextBuilder()`, matching the schema correction). |

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/npc`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok` or `[no test files]`, none failing. |
| atlas-saga-orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok` or `[no test files]`. |
| atlas-channel (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` — all packages `ok`. `tools/operator-cancel-path-guard.sh` also passes. |

`tools/npc-conversation-contract-mirror-guard.sh` passes at HEAD (`OK — both copies identical.`), covering both the Task 1 field addition and the Task 6 `CommandTextBody` addition.

## Overall Assessment

- **Plan Adherence:** FULL (for Tasks 1-11)
- **Recommendation:** READY_TO_MERGE (with respect to this task range; final merge readiness also depends on the Tasks 12-21 shard's findings, which this audit does not cover)

## Action Items

None. No fixes required for Tasks 1-11.
