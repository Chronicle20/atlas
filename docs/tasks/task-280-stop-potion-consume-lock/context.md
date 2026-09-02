# task-280 — Planning Context

Companion to `plan.md`. Everything here was read out of the branch at plan time;
nothing is remembered or inferred.

## Key files and what they currently do

| File | Current state |
|---|---|
| `services/atlas-consumables/atlas.com/consumables/character/buff/model.go` | Immutable `buff.Model` + `IsZombified` (lines 63-77), the slice-level UNDEAD predicate task-256 pinned. `Expired()` honours `noExpiry`. |
| `.../character/buff/processor.go` | `Processor` interface with `GetByCharacterId(uint32) ([]Model, error)`; drains every page and normalizes 404 → empty slice. |
| `.../character/buff/mock/processor.go` | `ProcessorMock` with `GetByCharacterIdFunc`. Already exists; no new mock needed. |
| `.../compartment/mock/processor.go` | `ProcessorMock` with `RequestReserveFunc` / `CancelItemReservationFunc`. |
| `.../consumable/processor.go` (1617 lines) | `ProcessorImpl` (l, ctx, cp, ip, cpp, cdp); `usesStandardConsumer` (lines ~112-123, classifications 200/201/202/205/221); `resolveZombified` (176-183); `RequestItemConsume` (300-370); `ConsumeError` (419-433); `consumeErrorType` (442-450); sentinel block (63-66). |
| `.../consumable/producer.go:18` | `ErrorEventProvider(characterId character.Id, error string)` — the existing ERROR event provider. |
| `.../kafka/message/consumable/kafka.go:118-122` | `ErrorType*` group: PET_CANNOT_CONSUME, PET_CANNOT_LEARN, INVENTORY_FULL, VEGA_INVALID, CONSUME_FAILED. |
| `services/atlas-channel/.../kafka/message/consumable/kafka.go:90-92` | Channel's mirrored subset: PET_CANNOT_CONSUME, INVENTORY_FULL, VEGA_INVALID. **No CONSUME_FAILED constant here.** |
| `services/atlas-channel/.../kafka/consumer/consumable/consumer.go:94-152` | `handleErrorConsumableEvent` — a three-branch if-chain with inline effects plus a catch-all `StatChanged` unstick. No test file in this package today. |

## Decisions carried from design.md

- **Gate placement** is after the `inventory2.TypeFromItemId` validity check and
  before `RequestReserve`/`RegisterHandler`. That placement is what removes any
  need to call `CancelItemReservation` on the rejection path (FR-5).
- **Short-circuit `usesStandardConsumer(itemId) && resolvePotionLocked(...)`** is
  the FR-2 "no buffs read for out-of-scope items" guarantee, and is asserted
  directly by a mock that fails the test if it fires.
- **`bp buff.Processor` as a struct field**, not a `potionGateDeps` struct. The
  gate has one collaborator; `morphCouponDeps` earns its shape from five.
- **`IsPotionLocked` is a new predicate, not a generalised `HasStat`.** Rewriting
  `IsZombified` (pinned by task-256 with its own tests) to buy nothing at two
  call sites was rejected; revisit if a third predicate (`STOP_MOTION`) appears.
- **No shared kafka contract package.** The two `kafka.go` files stay
  hand-mirrored, matching how every other `ErrorType*` in them is maintained.
- **Fail open on a buffs read error** (FR-4). Recorded as a deliberate hole:
  during an `atlas-buffs` outage the lock is unenforced.
- **Two buff reads per in-scope consume** (gate + `resolveZombified` in
  `ApplyItemEffects`) is accepted. The two are in different process invocations
  separated by a Kafka round trip; there is no call site where both are in
  scope, so nothing can be shared. Caching was rejected — it would open a
  staleness window on exactly the authority boundary this task closes.

## Corrections to design.md found while planning

- **design.md §6 says the Kafka emission "errors" in the gate test because no
  broker is configured.** It does not. `consumable/testmain_test.go` installs
  `producertest.InstallCapturing()` for the whole package, so the emission
  succeeds and the message is inspectable via the package-level `emitted`
  capture. The plan therefore asserts the ERROR payload from the real gate path
  as well as from the provider directly, which is strictly stronger than the
  design's fallback.
- **`consumer.GetManager().RegisterHandler` needs no broker.** It returns
  `"no consumer found for topic"` for an unregistered topic
  (`libs/atlas-kafka/consumer/manager.go:206-221`) and `RequestItemConsume`
  already discards that error, so the unlocked-path test runs clean.
- **The channel-side classifier gets four actions, not five.** design.md §6
  listed `actionUnstick, actionPetCashFoodError, actionInventoryFull,
  actionVegaInvalid`; `POTION_LOCKED` maps to `actionUnstick` with an explicit
  `case` rather than a fifth action whose body would be identical. Recognition
  — the actual FR-7 assertion — is proven by the explicit case, not by a
  distinct enum value.

## Dependencies between tasks

```
Task 1 (IsPotionLocked)  ─┐
Task 2 (sentinel + wire)  ├─► Task 3 (the gate)
                          │
Task 4 (channel routing) ─┘ independent of 1-3 except for agreeing on the
                            "POTION_LOCKED" spelling (pinned by a test on both sides)

Task 5 (tools/verify.sh) depends on all four.
```

Tasks 1, 2 and 4 can run in any order. Task 3 needs 1 and 2 landed.

## Sizing notes

No task is deliberately oversized. Largest is Task 3 at two edited files plus
one new test file, all in one module. Task 2 and Task 4 were split along the
service boundary specifically so no single task touches both
`atlas-consumables` and `atlas-channel` — the constant is added on each side by
that side's own task, with a wire-value assertion in both.

## Verification

- Module-local per task: `go build ./... && go test ./...` from
  `services/atlas-consumables/atlas.com/consumables` (Tasks 1-3) or
  `services/atlas-channel/atlas.com/channel` (Task 4).
- Branch gate: flagless `tools/verify.sh` must exit 0 (Task 5). `--quick` and
  `--no-docker` skip the bake and `-race` and do not count.
- Regression bar: the existing zombify cases in `consumable/processor_test.go`
  and `consumable/morph_coupon_test.go` must pass untouched.
