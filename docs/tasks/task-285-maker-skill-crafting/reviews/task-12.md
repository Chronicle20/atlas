# Task 12 Review — atlas-saga-orchestrator explicit-stat creation command

**Commit range:** `e8b1fbedb..7b07f6ee2` (single commit `7b07f6ee2`)
**Verdict:** APPROVED

## Scope confirmed

`git diff --stat e8b1fbedb..7b07f6ee2` touches exactly the four files the brief
named plus their two test files:

- `kafka/message/compartment/kafka.go` (+16)
- `kafka/message/compartment/kafka_test.go` (+61, orchestrator)
- `compartment/producer.go` (+26)
- `compartment/producer_test.go` (+97)
- `compartment/processor.go` (+10)
- `compartment/mock/processor.go` (+25/-14)
- `services/atlas-inventory/.../kafka/message/compartment/kafka_test.go` (+60,
  test-only, required by Step 5's "in each" instruction)

No production file in `atlas-inventory` was touched. No unrelated files. Scope
matches the brief exactly.

## Wire contract — the one thing that must hold

Compared `atlas-saga-orchestrator`'s `CreateAssetCommandBody`
(`kafka/message/compartment/kafka.go:127-149`) field-by-field against
`atlas-inventory`'s copy (`kafka/message/compartment/kafka.go:109-133`,
landed by Task 11):

- Field names, types, and JSON tags (including `omitempty` presence) are
  identical, in identical field order, for every one of the 15 stat fields
  plus `Slots`.
- The golden-JSON pin in both modules'
  `TestCreateAssetCommandBody_GoldenJSON_Agrees*` tests marshals an identical
  struct literal against an identical shared JSON string
  (`{"templateId":1082002,...,"jump":15}`), byte-for-byte, in both modules —
  this is exactly the test the brief's Step 5 asked for and it would fail on
  a key-name or field-order divergence. Ran both:
  - `atlas-saga-orchestrator`: `go test ./compartment/... ./kafka/message/compartment/... -count=1` → PASS
  - `atlas-inventory`: `go test ./kafka/message/compartment/... -count=1` → PASS
  - `go build ./...` clean in both modules.

## Producer/processor correctness

- `producer.go:26` `RequestCreateAssetWithStatsCommandProvider` sets all 15
  stat fields plus `Slots` from the `saga.AwardCraftedAssetPayload` argument
  onto the command body — verified against the diff, every field present.
- `producer.go:16-18` `RequestCreateAssetCommandProvider` (legacy) now
  delegates to the new function with a zero-value payload — the delegating-
  wrapper precedent named in the brief, applied one layer down at the
  provider (report explains why: existing tests exercise the provider
  directly). `TestRequestCreateItemIsUnchanged` pins that this path still
  emits all-zero stats and `Slots == 0` — PASS.
- `TestRequestCreateItemWithExplicitStatsCarriesEveryField` pins that a
  fully-populated payload (`Slots: 7`, 15 distinct nonzero stat values)
  survives through the emitted kafka message, decoded back into
  `CreateAssetCommandBody`, asserting each field individually including
  `Slots == 7` as the brief specifically requested (it's the one field
  without `omitempty`). PASS.
- `processor.go:79-85` `RequestCreateItemWithExplicitStats` matches the
  brief's specified signature exactly:
  `(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, stats saga.AwardCraftedAssetPayload) error`.
  Reuses `inventory.TypeFromItemId` guard and
  `producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)` emission,
  matching `RequestCreateItemWithStats`'s pattern one-for-one. Passes
  `useAverageStats=false` — correct, since explicit stats and averaged-stat
  rolling are mutually exclusive paths.
- `compartment/mock/processor.go` mirrors the new interface method
  (`RequestCreateItemWithExplicitStatsFunc` + method), consistent with
  `TestProcessorMockImplementsProcessor` passing.

## Self-flagged item 1 — `ShowEffect` dropped from the wire

Correct, not a defect. Traced the existing pattern in
`kafka/consumer/asset/consumer.go:44-68`
(`emitRewardNoticeForCurrentStep`): for the existing `AwardAsset`,
`DestroyAsset`, and `DestroyAssetFromSlot` actions, `ShowEffect` is read from
the **saga step's payload** (`step.Payload().(saga.AwardItemActionPayload)`
etc.) inside the orchestrator itself when deciding whether to emit a
`conversation_reward_notice` — it is never part of the `CreateAssetCommandBody`
wire message sent to `atlas-inventory`. `AwardItemActionPayload`'s wire
counterpart never carried `ShowEffect` either. Dropping `ShowEffect` from
`CreateAssetCommandBody` for the crafted-asset path is therefore consistent
with how every other `Award*`/`Destroy*` action already handles this field,
not a silent loss of functionality. (Confirmed downstream: Task 13, already
landed on this branch at `c96dd1fae` as `saga/handler.go:1087-1098`
`handleAwardCraftedAsset`, calls `RequestCreateItemWithExplicitStats` with
the full `AwardCraftedAssetPayload` still available in-process for whatever
`ShowEffect`-driven notice logic it adds — no data was lost, since the
orchestrator still holds the original saga step payload.)

## Self-flagged item 2 — provider/processor naming divergence

Not a defect. The brief names the **processor** method
`RequestCreateItemWithExplicitStats` (interface contract, consumed by Task
13) and that name is implemented exactly, character for character, at
`processor.go:37` and `:79`. The **provider** function name
(`RequestCreateAssetWithStatsCommandProvider`) is an internal implementation
detail one layer below the processor, consistent with this file's existing
`RequestCreateAssetCommandProvider` → `RequestCreateItemWithStats` naming
split (provider says "Asset", processor says "Item" — an existing
inconsistency in the file, not introduced here). The brief specified the
processor signature, not the provider name, so there is no divergence from
what was asked. Confirmed the consumer that matters: Task 13's
`saga/handler.go:1094` calls `h.compP.RequestCreateItemWithExplicitStats(...)`
— the exact name delivered. No caller anywhere references the provider name
directly except this file's own tests.

## Tests are honest

Both new producer tests use real behavior assertions (decoded kafka message
content), not tautologies. `TestRequestCreateItemIsUnchanged` would fail if
someone accidentally wired the legacy path through the new stats-carrying
code with a non-zero default. `TestRequestCreateItemWithExplicitStatsCarriesEveryField`
would fail on any dropped or transposed field. The golden-JSON tests in both
modules would fail on any tag/order divergence between the two structs —
this is exactly the seam-protecting test class the task exists to produce.

Ran the full test suite for the touched module:

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go build ./...                                                    # clean
go test ./compartment/... ./kafka/message/compartment/... -count=1
ok atlas-saga-orchestrator/compartment
ok atlas-saga-orchestrator/compartment/mock
ok atlas-saga-orchestrator/kafka/message/compartment
```

```
cd services/atlas-inventory/atlas.com/inventory
go build ./...                                        # clean
go test ./kafka/message/compartment/... -count=1
ok atlas-inventory/kafka/message/compartment
```

## Findings

None blocking. None non-blocking beyond what was already ruled on in Task
11's carried-forward findings (not re-litigated here per instruction).

## Not evaluable

None — the full review surface (both `CreateAssetCommandBody` copies, the
producer, the processor, the mock, and the seam test) was directly
inspectable and testable within this diff plus the one file (`saga/payloads.go`)
whose contract the diff depends on.

---

verdict: APPROVED
