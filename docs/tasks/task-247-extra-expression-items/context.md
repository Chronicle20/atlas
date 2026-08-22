# task-247 — Implementation Context

Companion to `plan.md`. Everything an implementer or reviewer needs that does
not belong inside a task body.

## Key files

| Path | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_expression.go` | The whole gate. 8 lines today, ~55 after. Both slices touch it. |
| `services/atlas-channel/atlas.com/channel/character/expression/{processor,producer}.go` | Single-importer package — `character_expression.go` is the only consumer, so widening `Change` has exactly one call site. |
| `services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go` | `Command` (outbound) and `Event` (inbound). |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go:57-63` | The stale TODO and the `NewCharacterExpression(..., 0)` call. The `int32`→`uint32` narrowing lives here and nowhere else. |
| `services/atlas-expressions/atlas.com/expressions/expression/{processor,producer,task}.go` | The mirror chain. `model.go`/`registry.go` are deliberately untouched. |
| `libs/atlas-packet/character/clientbound/expression.go:41` | `NewCharacterExpression` — the constructor that never exposed `byItemOption`. |
| `libs/atlas-packet/character/serverbound/expression.go:33-41` | `ExpressionRequest` — already decodes `duration int32` / `byItemOption bool` under `GMS && MajorVersion > 87`. No change needed. |
| `libs/atlas-constants/item/constants.go:96` | `ClassificationExpression = Classification(516)` — the existing constant Task 1 derives from. |

## Decisions carried from design.md

- **No `CashSlotItemType(6)` arm.** The client converts a type-6 item use into
  an emote request before any cash-item-use packet exists
  (`CDraggableItem::OnDoubleClicked` @`0x50814b` → `SendEtcCashItemUseRequest`
  @`0x508165` → `case 6:` @`0xa02c86`). An arm would be unreachable. There is no
  practical test for "this file is unchanged"; it is a review check against the
  branch diff, which is why it appears in the plan's post-plan verification.
- **Mapping is arithmetic, not a data lookup.** Mirroring the client's own
  `nItemID % 100 + 8` is higher fidelity than asking `atlas-data` which `516xxxx`
  items exist, and avoids a second service dependency on a cosmetic path.
- **`duration` travels as `int32` between services**, so the JSON payload reads
  `"duration": -1` rather than `4294967295`. The single narrowing to `uint32`
  is at `kafka/consumer/expression/consumer.go`, which is what makes the §5.1
  fallback a three-line change at one site.
- **Constructor widened rather than a `With…` variant.** The package has no
  other `With…` methods, and the compiler enumerating all four call sites is
  the point — three of them are GMS ≤ 87 fixtures whose expected bytes must not
  move, which demonstrates the "Go-surface only" guarantee rather than asserting
  it.
- **Ownership seam returns `(bool, error)`, not the asset.** Nothing downstream
  needs the asset; a narrower seam is a narrower thing for a test to fake.
- **`TransactionId` is in scope** (design §1.4) but isolated in Task 6 so review
  can drop it independently.

## Dependencies and ordering

```
Task 1 (constants) ─────────────────────────┐
Task 2 (packet ctor + channel call-site fix) ┤
Task 3 (atlas-expressions thread)            ├─> Task 4 (atlas-channel thread) ─> Task 5 (gate)
                                             ┘
Task 6 (transactionId)  — after Task 4
Task 7 (docs)           — independent
```

`libs/atlas-packet` is a `replace` dependency of `atlas-channel`
(`services/atlas-channel/atlas.com/channel/go.mod:89`), so **Task 2 must fix
`kafka/consumer/expression/consumer.go:62` in the same commit** or the channel
module stops compiling. Task 2 passes a literal `false` there; Task 4 replaces
it with the real field and deletes the TODO.

Task 3 and Task 4 are separate tasks but land on the same branch, so the JSON
contract is never half-deployed. Design §6's "land both services' halves
together" is satisfied at branch granularity — do not open the PR with only one
of them merged.

## Deliberately large tasks

- **Task 3 (atlas-expressions) touches 7 files**, one over the plan-lint
  threshold. It cannot be split: widening `Processor.Change` breaks
  `kafka/consumer/expression/consumer.go`, `producer.go`, `task.go`, the mock,
  and `processor_test.go` in the same compile. Every edit is the same mechanical
  two-parameter addition, so the batch is uniform rather than broad.
- **Task 2 spans two modules** (`libs/atlas-packet` and `atlas-channel`) for the
  same reason — the `replace` directive makes the constructor change and the
  call-site fix one atomic compile unit.

## Risk carried into Phase 5

Design §5.1: after Task 4, **every** GMS v95 emote broadcast carries
`duration = 0xFFFFFFFF` instead of today's hardcoded `0`, because every emote a
v95 client originates encodes `nDuration = -1`. Whether the receiving client's
`m_tEmotionEnd` expiry predicate tolerates a past stamp was **not established**
in the IDB (`CAvatar::PrepareFaceLayer` @`0x4647d0` sets it; no reader was
located). If live testing shows remote observers see the emote flash and vanish,
the fix is a clamp at the one narrowing site —
`d := e.Duration; if d < 0 { d = 0 }` in
`kafka/consumer/expression/consumer.go`. **Do not add it pre-emptively**: it
would make the `-1 → 0xFFFFFFFF` acceptance criterion unsatisfiable, and
forwarding what the client encoded is the fidelity choice.

## Test-infrastructure notes

- The `handler` package already has everything Task 5 needs: `mustTenant`,
  `newCashItemUseTestSession` (GMS v83), `newCashItemUseTestSessionForVersion`
  (`character_cash_item_use_test.go:23-84`) and `installCapturingProducer`
  (`cash_item_gachapon_test.go:50`). Do not add a `*_testhelpers.go`.
- Asserting "forwarded" through the capturing producer is stronger than the
  design's seam-call-count-only option and costs nothing, since the helper
  already exists in-package. Both are asserted.
- `atlas-expressions` tests need `setupProcessorTest` (miniredis) before any
  registry touch; copy the preamble from `TestProcessor_Change_AddsMessageToBuffer`.
- `message.Buffer.GetAll()` returns `map[string][]kafka.Message`, keyed by topic
  env name — unmarshal `.Value` to inspect the emitted `StatusEvent`.

## Out of scope (recorded so it is not re-litigated)

Server-side emote cooldown and morph block (cosmetic UX gates needing new
per-character state); item consumption or inventory mutation (these items have
no `spec` node and are permanent); cash-shop purchase path for the 516 range;
extending the `atlas-expressions` registry `Model` with the new fields (nothing
reads them back — the revert path uses fixed zeros).
