# Item-Expiration Extenders (task-222) — Implementation Context

Companion to [`plan.md`](./plan.md). Inputs: [`prd.md`](./prd.md), [`design.md`](./design.md).

---

## 1. Decisions already made

Three design questions were open in design.md §7. All are signed off; the plan
implements them and does not re-litigate them.

| # | Question | Decision | Consequence |
|---|---|---|---|
| D1 | Over-cap use: reject, or clamp-and-consume? | **Reject** | The PRD's FR-3.4 clamp is superseded. `evaluateExpirationExtension` returns a rejection when `proposed > cap`; no `min()` anywhere. Matches the client, which shows a notice and sends nothing. |
| D2 | Where does atlas-inventory get `maxDays` for re-validation? | **Carry `ExtenderTemplateId`** | The saga payload and the Kafka command body both carry the sandglass's template id; atlas-inventory does its own `GET /data/cash/items/{id}`. Costs one extra service call on a cold path and buys a real trust boundary. |
| D3 | Naming the reused codec | **Rename to `ItemUseTargetSlot`** | `ItemUseItemTag` disappears. One wire layout, one type, named for the layout rather than for either of its two callers. |

Two further PRD requirements were changed by the client research and are
already folded into the plan:

- **FR-3.2(1) is dropped.** There is no inventory type on the wire — the
  client hard-codes EQUIP. Nothing to gate.
- **A gate the PRD does not have was added (G5).** The client reads
  `info/notExtend` off the *target equip* and refuses when it is set. Nothing
  in the tree parsed that field before this task.

## 2. Task dependency order

Tasks 1–5 are independent of each other and can be done in any order. From
Task 6 on, the chain is real:

```
1 (atlas-data cash)        ─┐
2 (atlas-data equipment)   ─┼─→ 13 (channel data mirrors) ─┐
3 (constants)              ─┼───────────────────────────────┼─→ 14 (channel arm)
4 (codec rename)           ─┘                               │
5 (libs/atlas-saga)  ─→ 6 (inventory contract + cash client)│
                           ├─→ 7 (asset)  ─→ 8 (compartment) ─→ 9 (consumer)
                     5 ─→ 10 (orch contract) ─→ 11 (handler) ─→ 12 (timer/compensator)
                                                              14 ←┘ (saga aliases)
15 (verification) — last, needs everything
```

Task 14 is the only one that needs the whole stack in place; Task 15 is the
gate before the PR.

## 3. Key source anchors

Read these before touching the corresponding task — they are the patterns the
plan copies, not merely related code.

| Concern | File:line | Why |
|---|---|---|
| The structural template for the whole feature | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:264-331` | The Sealing Lock arm: same sub-body decode → gates → cash-data lookup → two-step saga shape. |
| The Item Tag arm (shares this task's codec) | same file, `:202-259` | The other caller of `ItemUseTargetSlot`. Its negative-slot handling is the precedent. |
| Version-resolver precedent | same file, `:704-713` (`viciousHammerCashSlotItemType`) and `:260-263` (the `sealTimed` switch) | Exactly the shape Task 14's resolver copies, including the doc comment explaining *why* it must stay version-scoped. |
| The classifier's 550 branch | same file, `:1034-1040` | Already returns 62 at GMS ≥ 95, 61 below. Task 3 only replaces the bare `550` literal; the body is untouched. |
| The trap the feature must not fall into | `services/atlas-inventory/atlas.com/inventory/asset/processor.go:329-341` | `ApplyLock` adds `FlagLock` unconditionally *and* rejects `!Locked() && !Expiration().IsZero()` — this feature's only valid target. |
| Flag-preserving write path | `services/atlas-inventory/atlas.com/inventory/asset/administrator.go:65` (`updateFlagAndExpiration`) | Reused with the asset's existing flag passed straight through. One write path; the invariant is a unit-test assertion. |
| Compartment method shape | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1045-1076` (`ApplyAssetLock*`) | Lock registry → `GetByCharacterAndType` → `GetBySlot` → asset method; `*AndEmit` wraps it in `ExecuteTransaction` + `message.Emit`. |
| Test-seam precedent | same file, `:142` (`WithAssetProcessor`) | Task 8's `WithCashProcessor` copies it. Read it — it returns a shallow copy, not a mutated receiver. |
| Data-client precedent (atlas-inventory) | `services/atlas-inventory/atlas.com/inventory/data/consumable/` | The `requests.go` / `rest.go` / `model.go` / `processor.go` / `mock/` five-file shape Task 6 replicates for `data/cash`. |
| Saga registration sites | `libs/atlas-saga/{model.go,payloads.go,unmarshal.go}` | Type, action, payload, decode arm. All four are needed. |
| Orchestrator registration sites | `.../saga/{model.go,handler.go,event_acceptance.go,timer.go,compensator.go}` | **Eight** distinct edits across five files — see §4. |
| The refund path (no change needed) | `.../saga/compensator.go:1428-1480` (`DispatchCashItemUseRollbacks`) | Its `DestroyAsset` → `RequestCreateItem` arm already refunds the sandglass by `TemplateId`. |
| Client feedback (no change needed) | `services/atlas-channel/.../kafka/consumer/asset/consumer.go:269-296` | `handleAssetUpdatedEvent` turns the asset `UPDATED` event into an `INVENTORY_OPERATION` add-entry, which rewrites the slot and refreshes the tooltip. This is why FR-5.1 needs no new packet. |

## 4. The registration sites that fail silently

Design §9 flags these as the highest-risk part of the change. Each is its own
plan step with its own file:line, because none of them fails a build when
missed — they fail at runtime, quietly.

| Site | File | Miss consequence |
|---|---|---|
| `reverseWalkSagaTypes` | `saga/timer.go:176` | A timed-out saga gets no reverse walk: the sandglass stays consumed, the target never extended. |
| `allSagaTypes` | `saga/timer.go:205` | `TestEverySagaTypeIsClassified` fails — the intended safety net, and the reason the type must appear in both lists. |
| `dispatchTimeoutRollbacks` switch | `saga/timer.go:237` | Same as the first, on the timeout path specifically. |
| `CompensateFailedStep` branch | `saga/compensator.go:267` | A *failed* (not timed-out) extension step never refunds the sandglass. |
| `acceptanceTable` | `saga/event_acceptance.go:126` | Unknown actions default-deny in `StepAcceptsEvent`, so the step never completes and the saga sits until timeout. `event_acceptance_test.go` has a coverage test that catches this. |
| Action dispatch switch | `saga/handler.go:950` | The step has no handler and never fires. |
| Local unmarshal switch | `saga/model.go:1609` | The payload decodes to the wrong type; the handler's type assertion fails with "invalid payload". |
| Shared unmarshal switch | `libs/atlas-saga/unmarshal.go:587` | Same, one layer up. |

## 5. Cross-module contracts that can drift

Two places where the same wire shape is written twice in separate Go modules,
so a one-sided edit compiles cleanly and fails at runtime:

1. **The compartment command body.** `atlas-inventory`'s
   `kafka/message/compartment/kafka.go` and `atlas-saga-orchestrator`'s copy
   of the same path. Task 10 pins the json tags with a marshalling test for
   exactly this reason. (This is the same failure mode
   `tools/trade-contract-mirror-guard.sh` exists to catch for the trade
   contract; there is no guard for the compartment contract.)
2. **The atlas-data REST models.** `addTime`/`maxDays` are defined in
   atlas-data's cash `RestModel` and consumed by *two* independent client
   models — atlas-channel's `data/cash` and the new atlas-inventory
   `data/cash`. `notExtend` similarly spans atlas-data's equipment
   `RestModel` and atlas-channel's `data/equipment`.

## 6. Version scope

| Version | Family present | Slot-item-type | Status |
|---|---|---|---|
| gms_v48 | no | — | Arm unreachable; `SendConsumeCashItemUseRequest` switch covers types 12–47. |
| gms_v61 | no | — | Arm unreachable; switch covers 12–52. |
| gms_v72 | yes | 61 | IDA-verified `@0x49FB33`. |
| gms_v79 | yes | 61 | IDA-verified `@0x47EC3E`. |
| gms_v83 | yes | 61 | IDA-verified `@0x48645B`. |
| gms_v84 | assumed yes | 61 (**unverified**) | Task 15 Step 1 settles it. |
| gms_v87 | yes | 61 | IDA-verified `@0x473D96`. |
| gms_v92 | assumed yes | 61 (**unverified**) | Task 15 Step 1 settles it. |
| gms_v95 | yes | 62 | IDA-verified `@0x488C70`. |
| jms_v185 | yes | 39 on the wire | Atlas *derives* the type from the item id rather than reading it off the packet, so it computes 61 and the resolver matches. The wire value never enters dispatch. |

Per-version `addTime`/`maxDays` are **data**, read at runtime from atlas-data
per tenant. No code branches on them, so no per-version value verification
gates this change. The v83 values in prd.md §4 serve as the reader fixture.

**No template edits expected.** `CASH_ITEM_USE` is an already-registered
handler and this task adds no opcode. Task 15 Step 3 confirms rather than
assumes.

## 7. Two collisions the version resolver prevents

The resolver is required for correctness, not style. `CashSlotItemType` values
are reused across classifications at different version boundaries:

- **61 at GMS ≥ 95** is the megaphone arm (`ClassificationMegaphones`,
  `otherCategory == 7`, `character_cash_item_use.go:829`). A bare
  `CashSlotItemType(61)` comparison would swallow v95 megaphones into the
  sandglass arm.
- **62 at GMS < 95** is classification 551 (`:1041`). A bare
  `CashSlotItemType(62)` comparison would swallow that family on older
  versions.

The version-scoped resolver mirrors the classifier's own condition
(`Region() == "GMS" && MajorVersion() >= 95`) exactly, which is what
guarantees the two can never disagree. Task 14 has a test asserting that
agreement across every GMS version in scope.

## 8. Test strategy

The handler arm has no existing per-arm test harness — the only seam in
`character_cash_item_use.go` is the `cashItemInSlotFunc` package var, and
testing the arm end-to-end would need four more. Rather than invent them, the
plan extracts the gates and the formula into a **pure** function
(`evaluateExpirationExtension`) in a sibling file, following the
`character_cash_item_use_point_reset.go` split precedent. Every gate except
G1 (empty slot, which is an error from `GetItemInSlot`) is then table-tested
directly.

Coverage by layer:

| Layer | What is asserted |
|---|---|
| atlas-data readers | All five v83 `addTime`/`maxDays` pairs plus the absent-field default; `notExtend` present-true / present-false / absent. |
| Codec | Round-trip across every version variant for negative and positive slots; v83 golden bytes; equipped-slot golden bytes. |
| Pure evaluator | Under cap, exactly at cap, over cap, already past cap, `maxDays == 0`, the 99-day-vs-30-day case, and each of the four target-state gates. |
| Resolver | Version-scoped value per version; agreement with `GetCashSlotItemType`. |
| atlas-inventory asset | Flag preservation (the load-bearing invariant), locked/permanent rejection, redelivery idempotency including the still-emit-UPDATED requirement. |
| atlas-inventory compartment | Forged over-cap clamped server-side; in-bounds request honoured verbatim; zero-`maxDays` rejected with the asset unchanged. |
| Orchestrator | Step handler issues the command with the right arguments; acceptance-table entry present; both timer lists carry the type; the refund fires. |
| Cross-module | The command body's json tags pinned by a marshalling test. |

Live verification (Task 15 Step 7) is not inferrable and is called out
explicitly in the acceptance criteria: the tooltip must refresh without a
relog.

## 9. Where the placeholder helper names appear

Several test snippets in the plan use placeholder names for harness helpers
that already exist under other names in the target files:
`testDatabase`, `testContext`, `seedAsset`, `seedEquipCompartmentWithAsset`,
and `NewCompensatorWith`. Each occurrence says so in prose. Read the
neighbouring tests in the same file and substitute the real helper rather than
adding a parallel harness — the project's Test Helper Pattern rule forbids
`*_testhelpers.go` files with test-only constructors, and the Builder pattern
is the intended setup mechanism.

## 10. Build gates

Per CLAUDE.md, before this branch is "done":

- `go test -race ./...` and `go vet ./...` clean in all seven changed modules
  (libs/atlas-constants, libs/atlas-packet, libs/atlas-saga, atlas-data,
  atlas-channel, atlas-inventory, atlas-saga-orchestrator)
- `docker buildx bake` for the four changed services
- `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
  `tools/goroutine-guard.sh`, `tools/skill-job-id-guard.sh`,
  `tools/buff-duration-guard.sh` all clean from the repo root
- template guards only if Task 15 Step 3 produced a template edit
- `superpowers:requesting-code-review` run before the PR is opened
