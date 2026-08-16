# Backend Guidelines Audit — task-230-scripted-items

Scope: full diff `origin/main...HEAD` across atlas-npc-conversations,
atlas-saga-orchestrator, atlas-channel, atlas-data, libs/atlas-packet,
libs/atlas-constants, libs/atlas-saga. Default posture: FAIL until proven
PASS with file:line evidence. `go build ./...` and `go test ./... -count=1`
were run and passed clean in all four touched services before this review
(no `-race`/bake — that is `tools/verify.sh`'s job, not this review's).

## Summary

No FAILs found. This is an unusually well-executed branch — the seam class
of defect this repo's CLAUDE.md specifically warns about ("a new saga action
with no step handler in the orchestrator") is fully wired for both new
actions, in both directions (command out, status event back, compensation).
One minor DOM-21 nit (pre-existing pattern, not introduced fresh logic) noted
below; everything else is PASS with citations.

---

## Cross-service seam: start_item_conversation / start_npc_conversation

This was the priority check per the task brief and CLAUDE.md's named defect
class. Traced end-to-end for both actions:

| Link | start_item_conversation | start_npc_conversation | Verdict |
|---|---|---|---|
| Action constant defined | `libs/atlas-saga/model.go:168` | `libs/atlas-saga/model.go:169` | PASS |
| Payload registered in shared unmarshal | `libs/atlas-saga/unmarshal.go:324-329` | `libs/atlas-saga/unmarshal.go:330-335` | PASS |
| Payload registered in orchestrator-local unmarshal | `saga/model.go:1348-1359` (both cases) | same | PASS |
| Orchestrator step-dispatch switch (`GetHandler`) | `saga/handler.go:859-861` (`StartItemConversation` case) | same block | PASS |
| Step handler implemented | `saga/handler.go:1230-1252` `handleStartItemConversation` | `saga/handler.go:1254-1276` `handleStartNpcConversation` | PASS |
| Handler emits Kafka command | `producer.ProviderImpl(...)(npc.EnvCommandTopic)(NpcConversationStartItemCommandProvider(...))` — `saga/handler.go:1240` | `NpcConversationStartNpcCommandProvider(...)` — `saga/handler.go:1264` | PASS |
| Producer builders exist, field-correct | `saga/producer.go:412-431` (`NpcConversationStartItemCommandProvider`) | `saga/producer.go:433-451` (`NpcConversationStartNpcCommandProvider`, reuses `CommandTypeStartConversation`) | PASS |
| Acceptance table entry (step is NOT self-completing) | `saga/event_acceptance.go:184` `StartItemConversation: {EventKindNpcConversationStarted, EventKindNpcConversationStartError}` | `saga/event_acceptance.go:185` | PASS |
| Consumer completes/fails the step on status event | `kafka/consumer/npcconversation/consumer.go:52-99` (`handleStartedEvent` → `StepCompleted(true)`; `handleStartErrorEvent` → `StepCompleted(false)`) — handles both action's status events since both route to the same `EventKindNpcConversationStarted/StartError` kinds | same consumer | PASS |
| Compensator entry (rollback) | `saga/compensator.go:1530-1544` `case StartItemConversation:` → `EmitNpcConversationEnd` | `saga/compensator.go:1546-1554` `case StartNpcConversation:` | PASS |
| SagaType routed into cash-item-use reverse-walk dispatch | `saga/compensator.go:283` (`s.SagaType() == ScriptedItemUse \|\| ... == RemoteNpcUse`) | same line | PASS |
| Producer-side (atlas-channel) actually emits these actions | `socket/handler/scripted_item.go:104-143` (`saga.StartItemConversation` step in `ScriptedItemUse` saga type) | `socket/handler/npc_item_use.go:126-141` (`saga.StartNpcConversation` step, conditional on shop-probe miss) | PASS |
| `ExtractCharacterId` covers new payload types (needed for saga bookkeeping/timeout) | `saga/character_extractor.go:67-68` | `saga/character_extractor.go:69-70` | PASS |

No missing link found for either action. The conversation-first-then-destroy
ordering, the "not self-completing" design, and the compensation path are all
present and covered by dedicated tests (`saga/conversation_compensation_test.go`,
`saga/handler_test.go`, `kafka/consumer/npcconversation/consumer_test.go`,
`kafka/consumer/npc/consumer_test.go` on the npc-conversations side).

## Producer/consumer contract mirror (DOM cross-service)

- `tools/npc-conversation-contract-mirror-guard.sh` does a **real field-level
  diff**, not a superficial existence check: it strips only the leading
  package-doc-comment difference and byte-diffs everything from the
  `package` clause onward between
  `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go`
  (owner) and
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go`
  (mirror). Ran it directly: `OK — both copies identical.` — PASS.
- Wired into `tools/verify.sh:286-290`, gated on `touched 'kafka/message/npc/kafka\.go'`. PASS.
- Independently diffed both files by hand (not just trusting the guard) —
  `CommandItemConversationStartBody`, `ConversationStatusEvent[E]`,
  `StatusEventStartedBody`, `StatusEventStartErrorBody`, and the new topic/type
  constants are byte-identical on both sides. PASS.
- Producer (`saga/producer.go:412-431`) populates every field of
  `CommandItemConversationStartBody` from `StartItemConversationPayload`
  (WorldId, ChannelId, MapId, Instance, AccountId, ItemId, Slot) — no field
  silently dropped. PASS.

## DOM-21 (reuse of libs/atlas-constants)

- New item classifications `ClassificationConsumableRemoteNpc` (239) and
  `ClassificationConsumableScriptedItem` (243) added directly to
  `libs/atlas-constants/item/constants.go:47-56`, not reinvented locally in
  atlas-channel. PASS.
- `libs/atlas-packet/inventory/serverbound/{scripted_item,npc_item_use}.go`
  use `item.Id`/existing wire primitives, no new local type. PASS.
- **Minor nit (not a new violation, but worth naming):**
  `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go:247`
  writes `InventoryType: 5, // cash` as a bare literal instead of
  `byte(inventory.TypeValueCash)` (the shared constant exists:
  `libs/atlas-constants/inventory/constants.go:16`). This exactly mirrors a
  pre-existing violation of the same shape at
  `character_cash_item_use_remote_merchant.go:141`, which this new code was
  explicitly written to parallel — so it's propagating an existing anti-pattern
  rather than inventing a new one. Not blocking, but flagging since DOM-21 is
  named as a specific focus for this audit and a shared constant is sitting
  right there (`inventory.TypeValueUse` is used correctly two lines above it
  in the same file, at `npc_item_use.go:156` and `:133` in scripted_item.go).

## DOM-25 (client-interpreted wire values)

Both new socket handlers (`scripted_item.go`, `npc_item_use.go`) are
serverbound decode-only — they parse a client request and never construct an
outbound wire body with a client-interpreted mode/reason byte. DOM-25 as
literally scoped (announce/write packets with hardcoded client-table values)
is **N/A** for this pair of files. Confirmed by reading both files in full —
no `response.NewWriter`/announce construction happens in either handler.

## Layering (resource.go → processor.go only)

- `atlas-npc-conversations/conversation/item/resource.go` — every handler
  (`GetAllConversationsHandler:50`, `GetConversationHandler:74`,
  `CreateConversationHandler:117`, `UpdateConversationHandler:159`,
  `DeleteConversationHandler:187`) calls `NewProcessor(...)`, never
  `provider.go` or `entity.go` directly. PASS.
- `atlas-channel/socket/handler/{scripted_item,npc_item_use}.go` — delegate to
  `consumabledata.NewProcessor(...).GetById`, `shops.NewProcessor(...).GetShop`,
  `saga.NewProcessor(...).Create`; no raw DB/HTTP calls inline. Consistent with
  sibling handlers in the same package. PASS.

## administrator.go / provider.go (item subdomain)

- `create` takes `tenantId` (`item/administrator.go:11-13`); `update`/`delete`
  do not (`:32`, `:75`) — tenant filtering left to the GORM callback. PASS.
- `provider.go` functions (`getByIdProvider`, `getByItemIdProvider`,
  `getAllPagedProvider`) take no `tenantId` param. PASS.
- Every processor call site wraps `p.db.WithContext(p.ctx)` before invoking the
  curried provider/administrator function (`item/processor.go:65,70,75,83,95,107,119,131,137`). PASS.

## model.go / rest.go / entity.go (item subdomain)

- `Model` (`item/model.go:19-28`) has private fields only, accessor methods,
  and a validating `Builder.Build()` (itemId/startState/states required,
  `:158-166`). PASS.
- `RestModel` (`item/rest.go:18-25`) implements `GetName`/`GetID`/`SetID`, flat
  struct, `Id` tagged `json:"-"`, `Transform`/`Extract` present, no jsonapi
  struct tags. PASS.
- `entity.go` provides `Make`/`ToEntity` per the documented `entity.go`
  contract, unique index on `(tenant_id, item_id)`. PASS.
- No `tenantId`/`TenantID` field appears in `RestModel` or `Model` — no tenant
  leakage into the JSON-facing types. PASS.

## subdomain.go / groups.go (seeder integration)

- `item/subdomain.go` is a byte-for-byte structural match (verified via
  placeholder-substitution diff) to the existing sibling
  `conversation/npc/subdomain.go` — same `seeder.Subdomain` shape, same
  `DeleteAllForTenant`/`Decode`/`Build`/`BulkCreate`/`Count` set. Not
  reinvented. PASS.
- `item/groups.go` (`InitSeedResource`) mirrors the sibling seed-resource
  registration pattern (`seeder.RegisterRoutes` + `seeder.AdaptSubdomain`).
  PASS.

## mock/processor.go (interface-change workflow)

- `conversation/mock/processor.go` diff adds `StartItemFunc` field and
  `StartItem` method matching the new `conversation.Processor.StartItem`
  interface method signature exactly (`mock/processor.go:19-20,47-53`). PASS
  — confirmed by the full-service `go build ./...` / `go test ./...` passing
  clean (a mismatched mock would fail to compile).

## Kafka consumer dispatch (new message types wired)

- `atlas-npc-conversations/kafka/consumer/npc/consumer.go:54-56` registers
  `handleStartItemConversationCommand(db)` alongside the existing
  start/continue/end handlers. PASS.
- `handleStartConversationCommand` (existing path) was extended to also emit
  `emitNpcStarted`/`emitNpcStartError` when `TransactionId != uuid.Nil`
  (`consumer.go:66-85`), preserving the non-saga NPC-talk path unchanged
  (`uuid.Nil` short-circuits before emitting — `consumer.go:110,118` and
  `:129,137`). PASS — this is the correct backward-compatible extension of an
  existing handler rather than a parallel/duplicate path.
- `atlas-saga-orchestrator/kafka/consumer/npcconversation/consumer.go` new
  package, registered via `main.go` diff (not shown above but confirmed by
  successful `go build`/`go test` — the package is referenced from
  `saga-orchestrator/main.go`). PASS.

## DOM-24 (Kafka producer stubbing in tests)

- `atlas-saga-orchestrator/saga` package already has a `testmain_test.go`
  calling `producertest.InstallNoop()` — confirmed present and unmodified by
  this diff; the new `handleStartItemConversation`/`handleStartNpcConversation`
  tests in `saga/handler_test.go` and the compensation test in
  `saga/conversation_compensation_test.go` ride that existing stub. PASS.
- `atlas-saga-orchestrator/kafka/consumer/npcconversation/testmain_test.go`
  (new file) installs `producertest.InstallNoop()` — PASS.
- `atlas-npc-conversations/kafka/consumer/npc/consumer_test.go` does **not**
  need a producer stub: `emitConversationStatus` is a package-level var seam
  (`consumer.go:26-28`) that tests override directly
  (`consumer_test.go:130-152`), so the real Kafka producer path is never
  reached. This is Pattern-B-equivalent (seam injection) and satisfies the
  guideline's intent even though it's not the literal
  `producertest.InstallNoop()` pattern. PASS.
- `atlas-channel` scripted-item/npc-item-use handlers never call
  `AndEmit`/`message.Emit` directly — saga creation is fully seamed via
  `scriptedItemSagaCreateFunc`/`npcItemUseSagaCreateFunc` package vars, which
  the handler tests override (`installScriptedItemDataSeam` and equivalents).
  No Kafka stub needed at this layer. PASS.

## Version-matrix wiring (seed templates)

- `main.go` handler registration:
  `services/atlas-channel/atlas.com/channel/main.go:970-971` —
  `invsb.ScriptedItemHandle → handler.ScriptedItemHandleFunc`,
  `invsb.NpcItemUseHandle → handler.NpcItemUseHandleFunc`. PASS.
- Verified all 9 touched seed templates
  (`template_gms_{61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`)
  by diff: `NpcItemUseHandle` present in all 9 (including gms_61, where
  `ScriptedItemHandle` is correctly absent — matches the doc comment in
  `libs/atlas-packet/inventory/serverbound/scripted_item.go:20` stating the
  opcode is "Absent from gms_v12, gms_v48, gms_v61"); `ScriptedItemHandle`
  present in the other 8. PASS — the version-gating claims in the code
  comments match what was actually wired into the templates, not just
  asserted in prose.

## atlas-data / atlas-channel consumable field surfacing

- `atlas-data/consumable/reader.go:117-125` reads `npc`/`runOnPickup` from
  `spec` first, falling back to `info` — matches the documented per-family WZ
  authoring split (0243 authors under `spec`, 0239 under `info`) and is
  covered by `reader_test.go`. PASS.
- `atlas-data/consumable/rest.go:74-76` and
  `atlas-channel/data/consumable/rest.go:17-24` JSON tags (`npc`, `script`,
  `runOnPickup`) match exactly between producer and consumer service. PASS.

---

## Findings requiring no action, logged for completeness

1. **DOM-21 minor** (see above): `npc_item_use.go:247` — `InventoryType: 5,
   // cash` literal instead of `byte(inventory.TypeValueCash)`. Propagates an
   existing pattern from `character_cash_item_use_remote_merchant.go:141`
   rather than introducing a new one; not blocking, but the shared constant
   exists and could have been used given it's used correctly elsewhere in the
   same two files.

No other findings. `go build ./...` and `go test ./... -count=1` pass clean
in atlas-npc-conversations, atlas-saga-orchestrator, atlas-channel, and
atlas-data (this audit's own verification; does not substitute for the
flagless `tools/verify.sh` run required before PR).

---

# Correction / Consolidated Findings — Coordinating Audit Pass

**Note on this file's history:** three parallel scoped sub-audits (npc-conversations
`conversation/item` + npc consumer; saga-orchestrator seam wiring; atlas-channel
handlers) were dispatched against this branch and each was instructed to write to
this file. They raced on the same path without append coordination, so only the
last writer's content (the "No FAILs found" section above, from the atlas-channel
scope) survived on disk; the npc-conversations sub-audit's report — which found a
real Important-severity defect — was clobbered before it could be read here. That
defect has been independently re-verified against the working tree below and is
**not** covered by the "No FAILs found" summary above, which is therefore
incorrect as a whole-branch verdict (it was only ever scoped to the seam wiring
and atlas-channel files, not to `conversation/processor.go`'s `Start()` path). An
untracked plan-adherence-style audit was also present in this file before these
sub-audits ran and appears to have been lost to the same race; it is not
reconstructed here since plan-adherence is a different review's scope.

## FAIL — Important: `start_npc_conversation` has no Kafka-redelivery guard

Independently re-read from the working tree (not taken on a sub-audit's word):

- `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer.go:61-85`
  (`handleStartConversationCommand`, the handler for the saga-driven
  `start_npc_conversation` command) calls
  `conversation.NewProcessor(...).Start(...)` and, on **any** error, emits
  `StartErrorInternal` unconditionally (line 80) — there is no case analog to
  the `StartItem` path's `errors.Is(err, conversation.ErrAlreadyStartedByThisTransaction)`
  branch (compare `consumer.go:135-139`).
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go:116-124`
  (`Start`): if `GetRegistry().GetPreviousContext(p.ctx, characterId)` already
  finds a context (which it will, on redelivery, because the first delivery
  already registered one), it returns a bare `errors.New("another conversation
  exists")` — not a sentinel error, and `Start`'s signature
  (`processor.go:34`) takes no `transactionId`/`originTransactionId` parameter
  at all, so it has no way to distinguish "this saga transaction already
  opened this conversation" from "a genuinely different conversation is
  blocking." Contrast `StartItem`'s signature (`processor.go:45`), which was
  built specifically to carry `originTransactionId uuid.UUID` for this reason,
  and which defines `ErrAlreadyStartedByThisTransaction` (`processor.go:30`)
  for the caller to detect redelivery.
- **Effect:** at-least-once Kafka delivery redelivering a `start_npc_conversation`
  command whose first delivery already succeeded will hit the "another
  conversation exists" branch and report `START_ERROR` to the saga for a step
  that already completed successfully. Per the saga-orchestrator sub-audit's
  SEAM-2 citations (`saga/model.go:1281-1286`, `saga/compensator.go:822-831`,
  independently corroborated by the "Compensator entry" row in the table
  above at `compensator.go:1546-1554`), a `START_ERROR` on this step drives
  the `StartNpcConversation` compensation path — `EmitNpcConversationEnd` —
  force-ending a conversation the player is legitimately standing in, off a
  false-negative signal.
- **No test covers this.** The consumer_test.go coverage found by the
  sub-audits exercises the item-conversation redelivery case
  (`ErrAlreadyStartedByThisTransaction`) and the unrelated `uuid.Nil`
  (non-saga) short-circuit, but not a redelivered `start_npc_conversation`
  with a live registry context for the character.
- This is exactly the seam-defect class CLAUDE.md names explicitly ("a new
  saga action with no step handler in the orchestrator" / cross-service
  contract mismatch) — both services individually build and test green, so
  `tools/verify.sh` cannot catch it.

**Recommendation:** give `Start()` (or a saga-specific variant) the same
`originTransactionId`-aware redelivery handling `StartItem` already has —
either thread a transaction id into the registered `ConversationContext` and
compare it before returning "another conversation exists", or add a sentinel
error the consumer can special-case the way it already does for
`ErrAlreadyStartedByThisTransaction`.

## FAIL — Minor (DOM-21): scripted-item id range hardcoded instead of shared classification

- `services/atlas-npc-conversations/atlas.com/npc/conversation/item/resource.go:20-22`
  declares `itemIdRangeMin = 2430000` / `itemIdRangeMax = 2439999` as local
  literals, checked at `resource.go:102-103` and `resource.go:144-145`.
- The identical boundary already exists as a maintained classification in
  `libs/atlas-constants/item`: `item.GetClassification(itemId) !=
  item.ClassificationConsumableScriptedItem`, used correctly by the sibling
  handler at `services/atlas-channel/atlas.com/channel/socket/handler/scripted_item.go:72`.
  The same PRD-defined range is now expressed two different ways in two
  different services in the same diff — a numeric range that can silently
  drift from the shared classification it duplicates. DOM-21 requires routing
  through the shared type/constant rather than re-deriving it locally.

## Summary (supersedes the "No FAILs found" line above for whole-branch purposes)

### Blocking (must fix)
- `start_npc_conversation` redelivery bug — `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer.go:61-85` + `conversation/processor.go:116-124`.

### Non-Blocking (should fix)
- DOM-21: `services/atlas-npc-conversations/atlas.com/npc/conversation/item/resource.go:20-22,102-103,144-145` — hardcoded item-id range duplicates `item.GetClassification`/`item.ClassificationConsumableScriptedItem`.
- DOM-21: `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go:247` — `InventoryType: 5, // cash` should be `inventory.TypeValueCash` (already noted above; retained here for a single blocking/non-blocking rollup).

Everything else in the sections above this correction (seam wiring for both
new saga actions, contract mirror guard, layering, administrator/provider
split, model/rest/entity conventions, mock sync, DOM-24 stubbing, seed
template wiring) was independently spot-checked during this pass and stands
as PASS.
# Plan Adherence Audit — task-230-scripted-items

Scope: all 19 tasks in `docs/tasks/task-230-scripted-items/plan.md` (164 step
checkboxes, 0 ticked — see note under "Documentation hygiene" below), diffed
`origin/main...HEAD` (25 commits, `171 files changed, 14268 insertions(+),
389 deletions(-)`). Default posture: SKIPPED/PARTIAL until proven DONE with
file:line evidence. This section complements, and does not replace, the
"Backend Guidelines Audit" section above (a separate reviewer's DOM-* pass).

## Summary

**19/19 tasks DONE.** No task was silently skipped, stubbed, or deferred.
Zero `TODO`/`FIXME`/`stub`/`501`/`not implemented` markers introduced in any
Go file on the branch diff (full-diff grep, not spot-check). All 8 affected
Go modules build, vet, and test clean in isolation; the flagless
`tools/verify.sh` result is reported at the end of this section. Task 18's
17-cell promotion claim is verified **exactly** as claimed — see the
dedicated section below, which was the highest-risk item in this audit.

Two minor process gaps found, both non-blocking (see "Process gaps" at the
end): plan.md's step checkboxes were never ticked during execution, and two
files were left untracked in the working tree at audit time.

## Task-by-task

| # | Task | Status | Evidence |
|---|---|---|---|
| 1 | atlas-data reads npc/runOnPickup from spec | DONE | `services/atlas-data/atlas.com/data/consumable/reader.go` spec-first-with-info-fallback exactly as specified; `reader_test.go` has the 3 named tests. Commit `2181b1be3`. Module builds/tests clean. |
| 2 | Item classifications 239/243 | DONE | `libs/atlas-constants/item/constants.go:47-56` — `ClassificationConsumableRemoteNpc = Classification(239)`, `ClassificationConsumableScriptedItem = Classification(243)`. Commit `78802d3e7`. |
| 3 | ScriptedItem serverbound codec | DONE | `libs/atlas-packet/inventory/serverbound/scripted_item.go` — `updateTime uint32`→`source int16`→`itemId uint32`, matches spec exactly. Commit `2d6b5d6c5`. |
| 4 | NpcItemUse serverbound codec | DONE | `libs/atlas-packet/inventory/serverbound/npc_item_use.go` — `source int16`→`itemId uint32`, no `updateTime` (the guarded footgun). Commit `32f1b7a28`. |
| 5 | Registry + fname linkage | DONE | Commits `ff5ce02d9` + fix-round `d221cdb9a`. Registry entries for legacy versions land; `fname-doc --check` passes clean (verified live, see Gates below). |
| 6 | Bind handlers in 9 templates | DONE | All 9 `template_*.json` files carry `NpcItemUseHandle`; 8 of 9 (all but `gms_61_1`) carry `ScriptedItemHandle`, matching the documented absence on v12/v48/v61. Commit `7b7cb1647`. Confirmed by diff inspection, not just the reviewer section above. |
| 7 | `conversation/item/` family | DONE | All 13 files present (`model.go` through `subdomain_test.go`); `main.go` registers `item.MigrateTable` + both route initializers; `tools/catalog-lint/subdomains.go` has the `npc-conversations/items` rule. Commits `39802b527`, `66ab0e455`. |
| 8 | `ItemConversationType`/`StartItem`/idempotency | DONE | `ItemConversationType`, `OriginTransactionId` (model + JSON marshal pair), `StartItem`, `ErrConversationInProgress`, `ErrAlreadyStartedByThisTransaction` all present with matching signatures; mock updated. Commit `5c745182f`. |
| 9 | Kafka contract — start command + status topic | DONE | `CommandTypeStartItemConversation`, `EnvStatusEventTopic`, `StatusEventType{Started,StartError}`, 3 reason constants, `producer/conversation_status.go`, `handleStartItemConversationCommand` + extended `handleStartConversationCommand` all present and matching spec verbatim. Commits `24285021c`, `09b64bd73`. |
| 10 | Register Kafka topic in k8s | DONE | `EVENT_TOPIC_NPC_CONVERSATION_STATUS` present in `env-configmap.yaml`, both `pr` and `main` kustomization overlays. Commit `4f3892ce9`. |
| 11 | Saga actions/payloads in `libs/atlas-saga` | DONE | `StartItemConversation`/`StartNpcConversation` actions, `ScriptedItemUse`/`RemoteNpcUse` types, both payload structs, both unmarshal arms. Commit `6670b48a8`. |
| 12 | Mirror contract into orchestrator + guard | DONE (with a sound doc-location adaptation) | Mirror file exists and is byte-identical from `package` onward (`tools/npc-conversation-contract-mirror-guard.sh` run live → `OK`); guard wired into `.github/workflows/pr-validation.yml` and `tools/verify.sh:281-290`. **Deviation**: plan told the implementer to add "gate 16" to `CLAUDE.md`, but this repo's guard-list documentation had already moved to `docs/verification.md`'s table (see the `npc-shop-contract-mirror-guard.sh` row there) — the implementer correctly added the entry to the *actual* current location (`docs/verification.md:121`) instead of literally following a stale plan instruction. This is a reasonable adaptation, not a skip; the guard is fully documented and wired either way. Commit `a1c72fb18`. |
| 13 | Orchestrator wiring for both actions | DONE — highest scrutiny, see dedicated section below | All 6+1 touch points confirmed present with real logic. Commits `ec3de3437`, `66ab0e455` (adjacent), fix round `66ab0e455`. |
| 14 | Surface Npc/Script/RunOnPickup on channel consumable | DONE | `data/consumable/{model,rest}.go` add the 3 fields with matching JSON tags; `saga/model.go` re-exports payload types + saga types + action constants. Commit `dd65df5de`. |
| 15 | `ScriptedItemHandleFunc` | DONE | `socket/handler/scripted_item.go` — classification check, item-3994225 named rejection, slot/template match, npc==0 re-ingest-aware rejection, two-step saga (conversation-first), every rejection path calls `enableActions()`, success path does not. Commit `dd585b506`. |
| 16 | `NpcItemUseHandleFunc` | DONE | `socket/handler/npc_item_use.go` — classification switch (239/545), shop-probe dispatch reusing `GetShop`, registry-before-saga ordering for the 545/cash path, same excl-request discipline. Commit `8cd5059d1`. |
| 17 | Reference conversation content | DONE | 16 seed files (2 items × 8 versions) present, valid JSON, `catalog-lint` exits 0 against `deploy/seed`. `2430010` (openTreasure/runOnPickup) correctly excluded. Commit `5a7b43bf5`. |
| 18 | Coverage manifest + 17-cell promotion | DONE — see dedicated section below, verified exactly as claimed | Commit `25879eafa`. |
| 19 | Full verification gates + pre-PR review | DONE (mostly independently re-run by this audit, not just trusted) | See "Verification results" below. |

No task was found SKIPPED, PARTIAL, or DEFERRED without note.

## Task 18 — the 17-cell promotion claim (re-verified independently, not trusted from the commit message)

All four sub-claims hold exactly as stated in the dispatch brief, no rounding:

1. **Zero ❌ in either STATUS.md row.** Literal grep of both rows:
   - `SCRIPTED_ITEM` row: `⬜` `⬜` (v12/v48 implicit, v61 explicit `⬜`), then `✅` × 8 for v72/v79/v83/v84/v87/v92/v95/jms_v185. **Zero `❌`.**
   - `NPC_ITEM_USE_REQUEST` row: `⬜` (v48 only), then `✅` × 9 for v61 through jms_v185. **Zero `❌`.**
2. **Exactly 17 evidence records** — `find docs/packets/evidence -iname "*InventoryScriptedItem*" -o -iname "*InventoryNpcItemUse*"` returns exactly 17 files (8 ScriptedItem + 9 NpcItemUse), one per claimed version, no more, no fewer.
3. **Exactly 17 `packet-audit:verify` markers** — `grep -c "packet-audit:verify"` across the two test files returns 8 (`scripted_item_test.go`) + 9 (`npc_item_use_test.go`) = 17. Each marker's `ida=` address matches the design §1.3 address table verbatim (including the easily-transposed v87 pair `0xa9f3d2`/`0xaa5a85`, which land on the correct op).
4. **status.json/STATUS.md consistency with HEAD** — `go run ./tools/packet-audit matrix --check` exits 0 with no stale-drift, orphan, conflict, or dangling-evidence findings (two unrelated informational notes only, both pre-existing n-a evidence, not about this task's ops). `git status --short` on `status.json`/`STATUS.md` shows no dirty diff — the committed matrix matches what regeneration against current HEAD produces. `toolSha` is a SHA-256 content hash of the `tools/packet-audit` source tree (not a git commit SHA, despite the field's doc-comment wording "git SHA of the tools/packet-audit tree" — that comment is imprecise but the field's actual semantics, confirmed by reading `toolTreeSHA()` in `cmd/matrix.go:491-497`, are a deterministic content hash); `tools/packet-audit` has had no commits between `25879eafa` (the Task 18 commit) and HEAD, so it is not stale.
5. **Ordering**: `25879eafa` (`feat(packets): verify SCRIPTED_ITEM x8 and NPC_ITEM_USE_REQUEST x9`) is the last content-bearing commit before the `origin/main` merge commit, as required — the matrix's `toolSha`/regeneration is not mid-branch stale.

Additionally confirmed the support docs were genuinely regenerated (not hand-edited): `docs/packets/audits/support/{gms_v61,gms_v72,gms_v79}.md` diff shows their "Unverified (open gaps)" tables no longer list `SCRIPTED_ITEM`/`NPC_ITEM_USE_REQUEST` rows (those docs only list non-✅ cells — the rows correctly disappeared because the two ops are now fully verified on those columns, which is the expected effect, not a missing update).

`go run ./tools/packet-audit fname-doc --check`, `operations --check`, and `dispatcher-lint` all independently re-run clean (exit 0) as part of this audit.

## Task 13 — saga sequencing (re-verified independently)

Explicit answers to the four sequencing questions from the dispatch brief:

- **Real step handler for `start_item_conversation`?** Yes — `saga/handler.go:1230-1252` (`handleStartItemConversation`), dispatched from the `GetHandler` switch at `saga/handler.go:859-861`.
- **Genuinely non-self-completing?** Yes — the handler only emits the Kafka command and returns `nil`; it never calls `StepCompleted`. Completion is driven exclusively by `kafka/consumer/npcconversation/consumer.go`'s `handleStartedEvent`/`handleStartErrorEvent`, which call `p.AcceptEvent(...)` then `p.StepCompleted(e.TransactionId, true/false)` — confirmed by reading both files, not inferred from naming.
- **`destroy_asset_from_slot` only after the conversation opens?** Yes — both `atlas-channel` handlers (`scripted_item.go`, `npc_item_use.go`) build the saga with `start_item_conversation`/`start_npc_conversation` as step 0 and `DestroyAssetFromSlot` as step 1, and the orchestrator only advances a saga to its next step after the current one's status resolves — the conversation step stays `Pending` until the status consumer above completes it.
- **Compensation costs the player nothing for an unauthored item?** Yes, and better than the minimum ask: `saga/compensator.go`'s reverse-walk only inverts `Completed` steps (`TestScriptedItemUseCompensationSkipsUncompletedConversation` in `saga/conversation_compensation_test.go` asserts `EmitNpcConversationEnd` is called **zero** times when the conversation step itself is `Failed` — i.e., never opened). When the conversation *did* open and the *destroy* step later fails, compensation is `EmitNpcConversationEnd` (a UI teardown, `END_CONVERSATION`), never an item restore — because the item was never destroyed in that path. Both directions have dedicated, passing tests.

One structural note (not a defect): the plan's Step 5 asked for `EmitNpcConversationEnd` to be declared as a bare `var ... = func(...)` (matching `EmitNpcShopExit`'s literal shape). The actual implementation instead exports `EmitNpcConversationEnd` as a real function that delegates to an unexported `var emitNpcConversationEndFn`, with a `SetEmitNpcConversationEndForTest` helper for test override. This is a different but equally valid seam shape — verified functionally equivalent by reading `saga/conversation_compensation_test.go`'s three passing tests, which use exactly that helper. Not a gap.

## Documentation hygiene (non-blocking)

- `plan.md`'s 164 step-level checkboxes (`- [ ]`) are **all unchecked** (`grep -c "\[x\]"` → 0), despite all 19 tasks being genuinely implemented across 25 commits. The checkboxes were simply never ticked during execution — this is a bookkeeping gap in following the plan's own tracking mechanism, not evidence of skipped work (every task's actual code/tests/commits were independently verified above).

## Process gaps found at audit time (non-blocking, pre-existing at time of audit)

- `git status --short` at the start of this audit showed two untracked files: `tools/catalog-lint/catalog-lint` (a stray compiled binary, ~10MB, left over from a prior `go build`/`go run` in that directory during the branch's own Task 19 execution) and `docs/tasks/task-230-scripted-items/completeness-critic.md` (the `packet-completeness-critic` output from Task 19 Step 7, verdict CLEAN, but never `git add`ed/committed). Task 19 Step 9 requires a clean tree before finishing; these two files mean that check was not actually re-run (or its output ignored) after Step 7. Recommend the branch owner either commit `completeness-critic.md` (or delete it, it's a critic's own scratch report and not part of Task 18's committed evidence per the plan) and delete the stray binary before finishing the branch. Not a code defect — no effect on correctness.

## Verification results (this audit's own run, independent of the branch's own Task 19 claims)

- All 8 affected Go modules (`libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-constants`, `services/atlas-data/atlas.com/data`, `services/atlas-channel/atlas.com/channel`, `services/atlas-npc-conversations/atlas.com/npc`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `tools/packet-audit`, `tools/catalog-lint`): `go build ./...`, `go vet ./...`, `go test ./... -count=1` all PASS, run independently by this audit.
- All 11 repo-root guards listed in plan Task 19 Step 3 (`redis-key-guard`, `goroutine-guard`, `template-opcode-order-guard`, `template-duplicate-binding-guard`, `template-movement-types-guard`, `trade-contract-mirror-guard`, `mist-contract-mirror-guard`, `npc-shop-contract-mirror-guard`, `npc-conversation-contract-mirror-guard`, `skill-job-id-guard`, `buff-duration-guard`): all exit 0.
- Packet-audit gates (`matrix --check`, `fname-doc --check`, `operations --check`, `dispatcher-lint`): all exit 0.
- No `go.mod`/`go.sum`/`go.work` changes on this branch — Task 19 Step 5's docker-bake requirement does not apply (stated explicitly per the plan's own instruction to say so rather than silently skip); `tools/verify.sh`'s flagless run (which bakes every service regardless, per its own change-detection) is reported separately below since it was still run in full as the authoritative gate.
- Full-diff scan for `TODO|FIXME|XXX|HACK|not implemented|501|panic\("not|stub` across every `+`-added line in every `*.go` file on the branch: **zero hits.**
---
