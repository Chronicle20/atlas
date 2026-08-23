# Review — bug-duey-receive-item-type-never-persisted

- **Unit**: commit `d41d70d39` (range `76a8961ef..d41d70d39`)
- **Brief**: `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-item-type-never-persisted.md`
- **Reviewer**: atlas-reviewer (Sonnet 5)

## Scope

`git diff --stat 76a8961ef..d41d70d39` — 12 files, +214/-0. Excluding the bug
doc itself (119 lines), the change touches exactly the seven files enumerated
in the bug's fix table plus two new/extended test files:

- `libs/atlas-saga/payloads.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/{saga/processor.go, saga/handler.go, parcel/processor.go, parcel/producer.go, kafka/message/parcel/custody/kafka.go, saga/parcel_expansion_test.go}`
- `services/atlas-parcel/atlas.com/parcel/{kafka/message/custody/kafka.go, kafka/consumer/custody/consumer.go, parcel/processor_custody.go, parcel/processor_custody_test.go}`

No channel-side files are touched (confirmed via `git diff --stat | grep -i channel` → empty), consistent with the bug report's claim that the channel side is already correct.

## Requirement-by-requirement trace

### 1. `libs/atlas-saga/payloads.go` — new field on `AcceptToParcelPayload`

`ItemType byte \`json:"itemType"\`` added in the "Item snapshot" block, directly above `TemplateId` (`libs/atlas-saga/payloads.go:1039`), documented as "the source inventory type... Zero when HasItem is false." Matches the brief's placement and semantics exactly. PASS.

### 2. `expandTransferToParcel` — item branch sets `ItemType`, meso branch stays zero

- Item branch (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:2263`): `ItemType: byte(payload.SourceInventoryType),` inside the `HasItem: true` `AcceptToParcelPayload` literal. `payload` here is `TransferToParcelPayload`, and `SourceInventoryType` is the field the bug traced back to `duey_action_send.go:277` — this is the sender's actual compartment type, not a re-derivation from `TemplateId`/`ItemId`. PASS.
- Meso-only branch (`saga/processor.go:2159-2192`, guarded by `payload.AssetId == 0`): the `AcceptToParcelPayload{...HasItem: false}` literal at line 2191 has no `ItemType` field set — it defaults to the zero value. Confirmed no `foundAsset`/inventory lookup occurs on this path (`comp`/`compartment.RequestCompartment` call is below the early return). PASS.

### 3. saga-orchestrator handler → params → producer → wire body

- `handler.go:2560` (`handleAcceptToParcel`): `ItemType: payload.ItemType,` copied into `parcel.AcceptToParcelParams`. PASS.
- `parcel/processor.go:41` (saga-orchestrator): `ItemType byte` added to `AcceptToParcelParams` struct, in the same block as `TemplateId`. PASS.
- `parcel/producer.go:36`: `ItemType: params.ItemType,` copied into the `AcceptToParcelCommandBody` wire literal. PASS.
- `kafka/message/parcel/custody/kafka.go:63` (saga-orchestrator's mirror): `ItemType byte \`json:"itemType"\`` added to `AcceptToParcelCommandBody`. PASS.

### 4. Wire-contract JSON-tag agreement between the two mirrors

- saga-orchestrator: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go:63` → `ItemType byte \`json:"itemType"\``
- atlas-parcel: `services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go:68` → `ItemType byte \`json:"itemType"\``

Same field name, same Go type (`byte`), same JSON tag (`itemType`). PASS — the two mirrors agree.

### 5. atlas-parcel consumer → `AcceptParams` → `AcceptCustody`

- `kafka/consumer/custody/consumer.go:100`: `ItemType: b.ItemType,` copied from the deserialized command body into `parcel.AcceptParams`. PASS.
- `parcel/processor_custody.go:42`: `ItemType byte` added to `AcceptParams`. PASS.
- `parcel/processor_custody.go:104-107`: `b.SetItemType(params.ItemType)` is called **only inside the `if params.HasItem {` branch** (line 103), alongside `SetItemId`/`SetQuantity`/`SetItemSnapshot`. The meso-only path (the `else`/fallthrough where `HasItem` is false) never calls `SetItemType`, so the builder default (zero) is preserved on that row. PASS — matches the "leave the meso-only branch at zero" requirement end-to-end (both at expansion time in the orchestrator and at persistence time in atlas-parcel).

### 6. Value provenance — not re-derived

The only place `ItemType`/`SourceInventoryType` is written is `saga/processor.go:2263`, sourced from `payload.SourceInventoryType` (itself set upstream at `duey_action_send.go:277`, outside this diff). No hop in the chain re-derives the type from `TemplateId`/`ItemId` (e.g. no call to an `inventory.TypeFromItemId`-style helper appears anywhere in the diff). PASS — matches the brief's explicit "no new inventory-type derivation from the template id anywhere."

## Test coverage — per hop

| Hop | Test | Asserts NEW contract? |
|---|---|---|
| `expandTransferToParcel` (item branch) | `saga/parcel_expansion_test.go:95` — `require.Equal(t, byte(1), acc.ItemType)` against `SourceInventoryType: 1` | Yes |
| `expandTransferToParcel` (meso branch) | `saga/parcel_expansion_test.go:131` — `require.Zero(t, acc.ItemType)` | Yes |
| `handler.go:handleAcceptToParcel` (payload → `AcceptToParcelParams`) | none found (`grep -rn "handleAcceptToParcel" saga/*_test.go` only turns up mock plumbing in `parcel_compensation_test.go`, not an assertion on the handler's field mapping) | **No** — but the bug's own test list makes this one explicitly conditional ("if an existing test covers that mapping"); none does, so no new test was strictly required by the brief. |
| `parcel/producer.go:AcceptToParcelProvider` (params → wire body) | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor_test.go:31` (`TestParcelProcessorDispatch`) exercises this exact function and unmarshals the produced `AcceptToParcelCommandBody`, but **never references `ItemType`/`cmd.Body.ItemType`** anywhere in the file (`grep -n ItemType processor_test.go` → no hits) | **No.** This test would still pass verbatim if `producer.go:36`'s `ItemType: params.ItemType,` line were deleted — it is the same "sender's compartment type gets dropped on a hop" shape as the original bug, now silently unguarded at this specific hop. |
| `kafka/consumer/custody/consumer.go:handleAcceptToParcel` (wire body → `AcceptParams`) | `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`, `TestCustodyCommands` "accept with item" (line ~128) drives the real DB round-trip via `newAcceptCommand` | **No** — two independent gaps: (1) `newAcceptCommand` (`consumer_test.go:88-117`) does not set `Body.ItemType` at all, so the value flowing through this test is always the zero value; (2) the test's assertions (`m.ItemId()`, `m.ItemSnapshot()...`, `m.RecipientName()`) never check `m.ItemType()`. This is the exact consumer-side hop the bug traced ("the write path is missing"), and it is exercised end-to-end by this test but the new field is neither populated in the fixture nor asserted on the result. |
| `AcceptCustody` (`AcceptParams` → persisted row) | `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody_test.go` (new, `TestProcessorAcceptCustody`) — `ItemType: 2` in, asserts `m.ItemType() == 2` and re-read `== 2`; meso-only case asserts `assert.Zero(t, m.ItemType())` | Yes |

## Findings

### Non-blocking

1. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor_test.go:31-70` (`TestParcelProcessorDispatch`) exercises `AcceptToParcelProvider` end-to-end through JSON serialization but does not assert `cmd.Body.ItemType`. A regression that deletes `parcel/producer.go:36`'s `ItemType: params.ItemType,` line would not be caught by any existing or new test in this diff.
2. `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go:88-160` (`newAcceptCommand` + "accept with item" case) exercises `handleAcceptToParcel` through a real DB round-trip but (a) the fixture never sets `Body.ItemType`, and (b) the assertions never check `m.ItemType()`. A regression that deletes `consumer.go:100`'s `ItemType: b.ItemType,` line would not be caught here either — the same shape of gap the bug itself was caused by, now reproduced one layer over at the consumer hop.
3. The handler-level hop (`saga/handler.go:2560`) has no direct unit test at all (pre-existing gap, not introduced by this change) — the brief itself made a new test here conditional on existing coverage, and none exists, so this is consistent with the brief rather than a violation of it. Noted for completeness only.

Both non-blocking findings are coverage gaps, not correctness defects: I traced every hop by hand (source-level, not test-level) and confirmed the field is copied correctly at each of the 8 mapping sites, the two wire mirrors agree on `json:"itemType"`, the value is the sender's `SourceInventoryType` and never re-derived, and the meso-only path is zero at both places it could be set (expansion and persistence). `go build ./... && go test ./...` passes clean in both `services/atlas-parcel/atlas.com/parcel` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

## Not evaluable

None — the full diff surface was read, and every referenced type/struct the diff calls into (`AcceptCustody`, `SetItemType`, `TransferToParcelPayload.SourceInventoryType`, the two wire-body structs) was inspected directly.

## Disposition

The fix itself is correct and complete against the brief: every file in the brief's table was changed exactly as specified, the value threaded is the sender's source inventory type (not re-derived), both wire-contract mirrors agree, and the meso-only branch stays zero at both places it's set. The gap is test coverage at two of the five intermediate hops (producer.go and consumer.go), where an existing/new test exercises the code path but does not assert the new field — meaning a future revert of either one-line mapping would regress silently. This does not block the fix from working today, so it is filed as non-blocking findings rather than a blocking defect.
