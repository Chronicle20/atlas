# Task 015 — Quest Start Reward Notices: Implementation Plan

Last Updated: 2026-04-20
Status: Draft (ready for implementation)
PRD: `prd.md`

---

## 1. Executive Summary

Quest **starts** that grant items today land silently in the inventory: `AddAwardItem` leaves `ShowEffect` at its zero value, and `QuestStartedEventBody` carries no item list, so atlas-channel never writes a `CharacterQuestEffectBody` on start. EXP and meso already render via their own paths.

This task closes the gap symmetrically to task-014's completion work: widen `QuestStartedEventBody` with an `Items` list, thread a `Rewards` field through `StartQuestPayload` → `StartCommandBody` → `Start()` / `StartChained()` as `externalRewards`, change `processStartActions` to return the awarded items, render items on the channel side via the existing `CharacterQuestEffectBody` packet, and add a `suppressAwardAssetByStartQuest` pass in the conversation operation_executor that mirrors `suppressAwardAssetByCompleteQuest`.

No WZ data crosses the orchestrator boundary. All new Kafka fields are `omitempty`/zero-value-safe, no DB migrations, no new topics, no new client packets, no new UI. Scope spans five services/libs.

## 2. Current State Analysis

- `libs/atlas-saga/payloads.go:252` — `StartQuestPayload` has no `Rewards` field. `QuestRewardItem` already exists at line 247 (added for task-014).
- `services/atlas-saga-orchestrator/.../kafka/message/quest/kafka.go:~37` — `StartCommandBody` has no `Rewards`. `CompleteCommandBody` does (task-014).
- `services/atlas-saga-orchestrator/.../saga/handler.go:~1532` — `handleStartQuest` does not extract `payload.Rewards`. `handleCompleteQuest` at lines 1510–1528 shows the exact target pattern.
- `services/atlas-saga-orchestrator/.../quest/producer.go:13` — `StartQuestCommandProvider` has no rewards parameter.
- `services/atlas-saga-orchestrator/.../quest/processor.go` — `RequestStartQuest` has no rewards parameter; `RequestCompleteQuest` does.
- `services/atlas-quest/.../kafka/message/quest/kafka.go:~37` — mirror of the orchestrator `StartCommandBody`; no `Rewards` field.
- `services/atlas-quest/.../quest/processor.go:189` — `Start()` does not accept `externalRewards`; `processStartActions` returns `error` only (not `([]ItemReward, error)` like `processEndActions` at line 796).
- `services/atlas-quest/.../quest/processor.go:327` — `StartChained()` parallel to `Start()`; same gap.
- `services/atlas-quest/.../kafka/message/quest/kafka.go:100` — `QuestStartedEventBody` has no `Items`.
- `services/atlas-quest/.../kafka/producer/quest/producer.go:100` — `EmitQuestStarted` does not accept items.
- `services/atlas-quest/.../quest/processor.go:18` — `EventEmitter` interface signature (and its mock) does not carry items on `EmitQuestStarted`.
- `services/atlas-quest/.../quest/resource.go` — HTTP handler calls `Start(...)`; must pass `nil` for the new `externalRewards` parameter to preserve behavior.
- `services/atlas-channel/.../kafka/message/quest/kafka.go:94` — `QuestStartedEventBody` has no `Items`.
- `services/atlas-channel/.../kafka/consumer/quest/consumer.go:57` — `handleQuestStarted`; line 74 `announceQuestStarted` writes the status-message packet only. `announceQuestCompleted` at line 117 is the pattern for `CharacterQuestEffectBody`.
- `services/atlas-npc-conversations/.../conversation/operation_executor.go:~795` — `createSagaForOperations`; lines 812–836 collect sibling `AwardAsset` items into `CompleteQuest` payloads. Line 844 calls `suppressAwardAssetByCompleteQuest(built)`. Line 874 is `suppressAwardAssetByCompleteQuest` itself.

## 3. Proposed Future State

- `libs/atlas-saga`: `StartQuestPayload` carries `Rewards []QuestRewardItem` (zero-value → WZ fallback).
- `atlas-saga-orchestrator` & `atlas-quest`: shape-compatible `StartCommandBody.Rewards` on both sides; `handleStartQuest` extracts and forwards; `RequestStartQuest` / `StartQuestCommandProvider` accept rewards.
- `atlas-quest`: `Start()` and `StartChained()` accept `externalRewards []questmessage.ItemReward` as final parameter. `processStartActions` returns `([]questmessage.ItemReward, error)` populated from positive-count `AddAwardItem` calls. `EmitQuestStarted` / `EventEmitter` interface / mock / `QuestStartedEventBody` all carry `Items`. `resource.go` passes `nil` for the new parameter.
- `atlas-channel`: `QuestStartedEventBody` carries `Items`. `handleQuestStarted` forwards into `announceQuestStarted`, which writes `CharacterQuestEffectBody("", rewards, 0)` when `len(items) > 0`.
- `atlas-npc-conversations`: `createSagaForOperations` adds a symmetric sibling-rewards collection pass for `StartQuest` steps. `suppressAwardAssetByStartQuest` is added next to `suppressAwardAssetByCompleteQuest` and invoked alongside it. The two suppression passes are independent — each inspects only its own action type.

## 4. Implementation Phases

Phases ordered to minimize cross-service breakage. Each ends with a build verification.

### Phase A — Foundational saga type (libs)
Add `Rewards` to `StartQuestPayload`. Builder/unmarshal round-trip.

### Phase B — atlas-quest internals
Thread `externalRewards` through `Start()` / `StartChained()`; change `processStartActions` return type; widen `EmitQuestStarted` + `EventEmitter` interface + mock; widen `QuestStartedEventBody`; widen `StartCommandBody`; update resource.go.

### Phase C — atlas-saga-orchestrator threading
Widen `StartCommandBody`; thread `Rewards` through `handleStartQuest` → `RequestStartQuest` → `StartQuestCommandProvider`. The orchestrator must stay wire-compatible with the atlas-quest side from Phase B.

### Phase D — atlas-channel rendering
Widen `QuestStartedEventBody`; `handleQuestStarted` forwards items; `announceQuestStarted` writes `CharacterQuestEffectBody` when items present.

### Phase E — atlas-npc-conversations planner
Add sibling-rewards collection pass for `StartQuest` payloads. Add `suppressAwardAssetByStartQuest` and wire it alongside the completion suppressor.

### Phase F — Build sweep & acceptance walk
`go build ./...` and `go test ./...` across all five affected modules; verify Docker builds for services consuming `libs/atlas-saga`. Walk PRD §10.

## 5. Detailed Tasks

Effort: **S** ≈ <½ day, **M** ≈ ½–1 day, **L** ≈ 1–2 days.

### Phase A — Foundational saga type

**A1. Add `Rewards` to `StartQuestPayload`** — Effort: S
- In `libs/atlas-saga/payloads.go` around line 252, add `Rewards []QuestRewardItem \`json:"rewards,omitempty"\``. Reuse the existing `QuestRewardItem` type.
- Update any `unmarshal.go` / builder coverage as needed (zero-value when absent).
- Acceptance: `go build ./...` and existing saga tests pass; JSON round-trip preserves field.
- Dependencies: none.

### Phase B — atlas-quest internals

**B1. Widen `StartCommandBody` (atlas-quest side)** — Effort: S
- Add `Rewards []ItemReward \`json:"rewards,omitempty"\`` to `services/atlas-quest/atlas.com/quest/kafka/message/quest/kafka.go` at ~line 37.
- Acceptance: package builds.
- Dependencies: none.

**B2. Widen `QuestStartedEventBody` (atlas-quest side) + `EmitQuestStarted`** — Effort: M
- Add `Items []ItemReward \`json:"items,omitempty"\`` to `QuestStartedEventBody` at ~line 100.
- Update `kafka/producer/quest/producer.go:100` `EmitQuestStarted` to accept `items []questmessage.ItemReward` and write it to the body.
- Update `EventEmitter` interface at `quest/processor.go:18` signature.
- Update the mock event emitter to capture `items`.
- Acceptance: package compiles; mock exposes captured items for assertion.
- Dependencies: none.

**B3. Change `processStartActions` return type** — Effort: M
- `processStartActions` currently returns `error`. Change to `([]questmessage.ItemReward, error)`.
- For each positive-count `AddAwardItem` (both the randomly-selected pool winner and unconditional positives), append an `ItemReward{ItemId, Amount: int32(Count)}`.
- Do **not** append for `Count < 0` entries — those are consumed start-requirement items, not rewards.
- Mirror `processEndActions` (processor.go:796), which already does this.
- Acceptance: existing unit tests updated for the new return; items list equals awarded items on representative fixtures.
- Dependencies: B2 (ItemReward symbol availability not an issue; both types already exist).

**B4. Add `externalRewards` to `Start()` and `StartChained()`** — Effort: M
- New final parameter: `externalRewards []questmessage.ItemReward`.
- After the `processStartActions` call in each function, apply the override: `reportedItems := awardedItems; if len(externalRewards) > 0 { reportedItems = externalRewards }`, then `EmitQuestStarted(..., reportedItems)`.
- Update internal callers: `resource.go` passes `nil`; the Kafka start command handler passes `c.Body.Rewards`; any other in-process callers pass `nil`.
- Acceptance: all atlas-quest callers compile; non-conversation paths emit with `Items` equal to WZ-awarded items; conversation path with non-empty rewards emits the override.
- Dependencies: B1, B2, B3.

**B5. Handler wiring for `StartCommandBody.Rewards`** — Effort: S
- In the atlas-quest Kafka command consumer that handles `StartCommandBody`, pass `c.Body.Rewards` into `Start(...)` / `StartChained(...)` as the new final parameter.
- Acceptance: command body is threaded end-to-end; unit test exercises the happy path.
- Dependencies: B4.

### Phase C — atlas-saga-orchestrator threading

**C1. Widen orchestrator `StartCommandBody`** — Effort: S
- Add `Rewards []ItemReward \`json:"rewards,omitempty"\`` to `services/atlas-saga-orchestrator/.../kafka/message/quest/kafka.go` at ~line 37. Stay shape-compatible with atlas-quest (B1).
- Acceptance: builds.
- Dependencies: B1 (for wire compat).

**C2. `StartQuestCommandProvider` accepts rewards** — Effort: S
- Add `rewards []quest.ItemReward` parameter to `quest/producer.go:13` `StartQuestCommandProvider` and set `Body.Rewards`.
- Mirror `CompleteQuestCommandProvider`'s shape.
- Acceptance: builds; existing callers updated.
- Dependencies: C1.

**C3. `RequestStartQuest` forwards rewards** — Effort: S
- Add `rewards []quest.ItemReward` parameter to `quest/processor.go`'s `RequestStartQuest` and forward to the provider.
- Mirror `RequestCompleteQuest`.
- Acceptance: builds; mocks updated.
- Dependencies: C2.

**C4. `handleStartQuest` extracts and forwards `payload.Rewards`** — Effort: M
- In `saga/handler.go:~1532`, extract `payload.Rewards`, convert to `[]questmessage.ItemReward` (same conversion used in `handleCompleteQuest` at 1510–1528), and pass to `RequestStartQuest(..., rewards)`.
- Acceptance: unit test asserts forwarded `StartCommandBody.Rewards` matches payload on non-empty input; empty payload forwards zero-length (or nil) slice.
- Dependencies: A1, C3.

### Phase D — atlas-channel rendering

**D1. Widen channel-side `QuestStartedEventBody`** — Effort: S
- Add `Items []ItemReward \`json:"items,omitempty"\`` to `services/atlas-channel/.../kafka/message/quest/kafka.go:94`. Stay shape-compatible with B2.
- Acceptance: builds.
- Dependencies: B2 (wire compat).

**D2. `handleQuestStarted` threads items to `announceQuestStarted`** — Effort: M
- In `kafka/consumer/quest/consumer.go:57` `handleQuestStarted`, extract `e.Body.Items` and pass into `announceQuestStarted` (line 74), which gains an `items []quest.ItemReward` parameter.
- `announceQuestStarted` writes the existing status-message packet unchanged, then — if `len(items) > 0` — additionally writes `CharacterQuestEffectBody("", rewards, 0)` using the same call shape as `announceQuestCompleted` at line 117. Convert `items` to the `[]QuestReward` argument the packet writer expects (same conversion used on completion).
- Do not introduce a foreign-broadcast variant; start has no analog of the completion foreign-effect packet.
- Acceptance: session-connected test confirms both packets are written when items present; with empty items only the status-message packet fires.
- Dependencies: D1.

### Phase E — atlas-npc-conversations planner

**E1. Collect sibling rewards into `StartQuest` payloads** — Effort: M
- In `conversation/operation_executor.go:~795` `createSagaForOperations`, immediately after the existing `CompleteQuest` rewards-collection pass (lines 812–836), add a symmetric pass that writes the same `[]saga.QuestRewardItem` list into any `StartQuest` step payload's new `Rewards` field.
- Acceptance: unit test constructs a conversation batch with sibling `award_item` + `start_quest` and asserts the built `StartQuestPayload.Rewards` contains the sibling items; without siblings `Rewards` is nil/empty.
- Dependencies: A1.

**E2. Add `suppressAwardAssetByStartQuest`** — Effort: L
- Add the function next to `suppressAwardAssetByCompleteQuest` at line 874. For each `StartQuest` step, inspect `payload.Rewards`; for each **preceding** `AwardAsset` step whose `(itemId, quantity)` is fully covered by a remaining reward entry, flip `ShowEffect` to `false` and subtract the covered quantity.
- Post-`StartQuest` `AwardAsset` steps and over-quantity `AwardAsset` steps remain visible.
- The two suppression helpers must be independent — a batch with both `start_quest` and `complete_quest` gets each pass looking only at its own action type.
- Call `suppressAwardAssetByStartQuest(built)` at line 844 alongside `suppressAwardAssetByCompleteQuest(built)`.
- Table-driven tests mirror task-014's B4 coverage: no `start_quest` → no suppression; matching sibling → suppression; quantity-mismatch sibling → no suppression; `award_item` following `start_quest` → no suppression; batch with both `start_quest` and `complete_quest` → independent suppression.
- Acceptance: tests pass; observable step-list shape change.
- Dependencies: A1, E1.

### Phase F — Build sweep & acceptance walk

**F1. Cross-service build & test sweep** — Effort: M
- `go build ./...` and `go test ./...` in: `libs/atlas-saga`, `services/atlas-saga-orchestrator`, `services/atlas-quest`, `services/atlas-channel`, `services/atlas-npc-conversations`.
- Verify Docker builds for services consuming changed `libs/atlas-saga` per CLAUDE.md: atlas-saga-orchestrator, atlas-quest, atlas-channel, atlas-npc-conversations (plus any other consumer surfaced by grep).
- Acceptance: all builds and tests green.
- Dependencies: A1–E2.

**F2. Acceptance-criteria walkthrough** — Effort: M
- Walk each PRD §10 bullet against the implementation; record evidence (test names, log references, manual verification notes) in `tasks.md`.
- Acceptance: every checkbox in PRD §10 is verified or has an explicit follow-up.
- Dependencies: F1.

## 6. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `StartCommandBody` wire schemas drift between orchestrator and atlas-quest | Low | High | Keep `ItemReward` field shape and JSON tags byte-identical across B1/C1; single field with same `omitempty` tag. |
| `QuestStartedEventBody` wire schemas drift between atlas-quest and atlas-channel | Low | High | Same. B2 and D1 gate each other. |
| `processStartActions` signature change ripples to untracked internal callers | Medium | Medium | Grep all callers in atlas-quest before B3; update each. Expect `resource.go` plus the chained-start path. |
| Double-rendering on sibling `award_item` + `start_quest` if E2 misorders walks | Medium | Medium | Table-driven tests including "preceding-only" and "batch with both start and complete" cases; assert shape. |
| Non-conversation callers of `Start()` / `StartChained()` break on signature change | Low | High | Every internal caller compiles only if updated; `nil` default for `externalRewards` preserves behavior on the wire (no `Items` emitted → channel takes the WZ-driven path already). |
| Chain follow-ups (complete → auto-start next) render confusingly back-to-back | Low | Low | PRD §3 states both effects firing is correct behavior. Document in acceptance test. |
| Mock event emitter divergence breaks existing tests | Medium | Low | Update mock in B2 and fix assertion signatures in the same PR. |
| Cross-service refactor cycles longer than estimated | Medium | Medium | Phase ordering minimizes cross-breakage; build gate per phase. Expect 2–3 fix-and-rebuild cycles per CLAUDE.md. |

## 7. Success Metrics

- All PRD §10 acceptance criteria pass.
- End-to-end manual verification: a job-advancement quest that starts via NPC and grants a starter weapon renders the item-gain effect on a v83 client.
- Zero regressions in existing test suites across the five affected modules.
- Suppression unit tests cover all five enumerated cases in E2.
- Wire fields are `omitempty` and absent from emitted JSON when unused (verify with one focused test against the JSON marshal output).

## 8. Required Resources & Dependencies

- Local Go toolchain across all affected services.
- Task-014 already landed on `main` — its `suppressAwardAssetByCompleteQuest` pattern, `CharacterQuestEffectBody` render on completion, `QuestRewardItem` type, and `CompleteCommandBody.Rewards` serve as the reference implementation. Re-read those sites before starting each phase.
- Familiarity with: `libs/atlas-saga` payload conventions, the atlas-quest processor pattern (`NewProcessor(l, ctx)`, `Method(mb)` / `MethodAndEmit()`), atlas-channel's `CharacterEffectWriter`, the conversation operation_executor planner.
- No infra, no Kafka topic provisioning, no DB migration, no client packet research — all paths reuse existing plumbing.

## 9. Timeline Estimate

- **Phase A:** 0.5 day (S).
- **Phase B:** 1.5–2 days (S+M+M+M+S). B3 and B4 sequential; B1/B2 can parallel with B3.
- **Phase C:** 1 day (S+S+S+M). Sequential within phase; gated on A1 + B1.
- **Phase D:** 0.5–1 day (S+M). Gated on B2.
- **Phase E:** 1–1.5 days (M+L). Gated on A1.
- **Phase F:** 0.5–1 day (M+M).

**Total:** ~5–6 working days for one engineer; ~3–4 with parallelization between atlas-quest (Phase B) and atlas-npc-conversations (Phase E).

## 10. Open Questions

None outstanding. PRD §9 explicitly notes the design-phase open items are all resolved in §4.1–§4.9. Any implementation-time questions will be appended here as they arise.
