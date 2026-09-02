# Review: Task 3 — the pre-reservation consume gate

Range: `8baa153fc..61d74c97b` (commit `61d74c97b feat(consumables): gate standard consumes on STOP_PORTION before reservation`)
Module: `services/atlas-consumables/atlas.com/consumables`
Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Scope

Two files changed, both under `services/atlas-consumables/atlas.com/consumables/consumable/`:

- `processor.go` (+47) — `bp` field, `NewProcessor` wiring, `resolvePotionLocked`, the gate in `RequestItemConsume`, `rejectPotionLocked`.
- `processor_potion_lock_test.go` (+216, new file) — five test functions.

`git diff --name-only 8baa153fc..61d74c97b` confirms exactly these two files. No file under `services/atlas-channel/` is touched — the channel-side lock check (Task 4) is genuinely out of scope here, satisfying the binding constraint.

## Findings

### PASS — Gate lives only in atlas-consumables; no channel-side edits

`git diff --name-only` lists only the two files above. No channel package touched.

### PASS — Gate positioned correctly (after `TypeFromItemId`, before `RequestReserve`/`RegisterHandler`); no `CancelItemReservation`

`processor.go:94-108` (diff line numbers): the gate sits directly after the `it, ok := inventory2.TypeFromItemId(itemId)` validity check and before the `var itemConsumer ItemConsumer` declaration. `RegisterHandler` (~line 121 in the diff hunk) and `RequestReserve` (~line 123) both come later in the function body — confirmed by reading the diff's second hunk in full. `rejectPotionLocked` (`processor.go:138-145` in the new code) only calls `producer.ProviderImpl(...)` to emit the ERROR event and returns `ErrPotionLocked`; it never calls `p.cpp.CancelItemReservation` or any `ConsumeError`-style path. FR-5 satisfied.

### PASS — Buff-read failure fails open at Warn

`resolvePotionLocked` (`processor.go:69-76` new code): on `bp.GetByCharacterId` error, logs `l.WithError(err).Warnf(...)` and returns `false` (unlocked), letting the consume proceed. Verified live: `TestRequestItemConsume_BuffReadErrorFailsOpen` asserts `err == nil` and `reserveCalled == true` and passes.

### PASS — Out-of-scope items issue no buffs read (FR-2)

Gate condition is `usesStandardConsumer(itemId) && resolvePotionLocked(p.l, p.bp, characterId)` (`processor.go:105`) — short-circuit evaluation guarantees `resolvePotionLocked` (and therefore `bp.GetByCharacterId`) is never called for an item `usesStandardConsumer` rejects. `usesStandardConsumer` (`processor.go:122-129`) matches only classifications 200, 201, 202, 205, and `ClassificationConsumableTransformation` (221) — town warp (203), pet food (212), and monster card (238) all fall through to `return false`. `TestRequestItemConsume_OutOfScopeIssuesNoBuffRead` drives ids 2030000/2120000/2380000 and asserts `read == false` in each subtest; test passes.

### PASS — Magnitude of STOP_PORTION never consulted (FR-3)

`buff.IsPotionLocked` (`character/buff/model.go:87-99`, consumed not modified by this diff) only checks `c.Type == charconst.TemporaryStatTypeStopPortion`; `Amount` is never read. Consistent with the fixture in the new test (`stat.Model{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}` — the `Amount: 1` value is never asserted on or varied).

### PASS — Locked-consume log line is Debug, not Error (FR-5.4)

`rejectPotionLocked`: `p.l.Debugf("Character [%d] attempted to consume item [%d] while potion use is locked. Rejecting.", ...)` (`processor.go:139` new code). The only `Errorf` in the function guards the unrelated Kafka-emit failure path, not the locked-consume event itself.

### PASS — `ApplyConsumableEffect`/`CancelConsumableEffect`/`ApplyItemEffects` still construct their own `buff.Processor`

`ApplyItemEffects` (`processor.go:294`, pre-existing, unchanged): `bp := buff.NewProcessor(l, ctx)` — local variable, not `p.bp`. `CancelConsumableEffect` (`processor.go:1169`, unchanged): `bp := buff.NewProcessor(p.l, p.ctx)`. `ApplyConsumableEffect` (line 1154) delegates to `ApplyItemEffects`, which builds its own. `p.bp` is used only inside `resolvePotionLocked` via the new gate call at `processor.go:105`. Confirmed by `grep -n "buff.NewProcessor\|ApplyConsumableEffect\|CancelConsumableEffect\|ApplyItemEffects" consumable/processor.go`.

### PASS — Wire value derived via `consumeErrorType(ErrPotionLocked)` (FR-6)

`rejectPotionLocked`: `ErrorEventProvider(ts.Id(characterId), consumeErrorType(ErrPotionLocked))` (`processor.go:140` new code) — single binding point, no separately-typed literal string.

### PASS — Pre-existing zombify/morph-coupon suites pass untouched

Ran `go test ./consumable/... -run 'PotionLock|RequestItemConsume_|Zombif|MorphCoupon' -v`: all pre-existing tests (`TestComputeEffectPlan_Zombify` and subtests, `TestComputeEffectPlan_ZombifyLeavesNonHpFieldsIdentical`, `TestResolveZombified` and subtests, the full `morph_coupon_test.go` suite) pass alongside the five new tests. `go test ./...` at the module root: all packages `ok`, no failures.

### PASS — Test honesty: the new tests genuinely pin the new behavior

Mechanically removed the gate block (`if usesStandardConsumer(itemId) && resolvePotionLocked(...) { return p.rejectPotionLocked(...) }`) from `processor.go` and re-ran the suite:

```
--- FAIL: TestRequestItemConsume_LockedInScopeRejects
    --- FAIL: TestRequestItemConsume_LockedInScopeRejects/standard_potion
    --- FAIL: TestRequestItemConsume_LockedInScopeRejects/transformation_potion
--- FAIL: TestRequestItemConsume_BuffReadErrorFailsOpen
```

Both fail without the change; `TestRequestItemConsume_OutOfScopeIssuesNoBuffRead` and `TestRequestItemConsume_UnlockedInScopeReserves` correctly continue to pass (they exercise paths the gate doesn't touch or that were already correct), which is expected — not every new test needs to fail pre-change, only the ones asserting the new behavior. File restored afterward (`git diff --stat` clean, `go build ./...` clean).

### PASS — Controller ruling 1 (nil `cp`/`ip`/`cdb` in FR-2 test)

`TestRequestItemConsume_OutOfScopeIssuesNoBuffRead` constructs `&ProcessorImpl{l: logrus.New(), ctx: context.Background(), bp: bp, cpp: cpp}` (`processor_potion_lock_test.go:280`) — `cp`/`ip`/`cdp` left nil, no panic (confirmed by the passing test run above), and the assertion is exactly `assert.False(t, read, ...)` (`processor_potion_lock_test.go:284`) — nothing else asserted, matching the ruling.

### PASS — Controller ruling 2 (hoisted `emitted` assertion doesn't weaken coverage)

Both in-scope subtests inside the loop (`processor_potion_lock_test.go:210-233`) independently assert `errors.Is(err, ErrPotionLocked)` (line 230) and `assert.False(t, reserved, ...)` (line 231) using per-subtest fresh mocks and a per-subtest `emitted.Reset()` (line 212). The single post-loop block (lines 238-257) does an additional isolated call with its own fresh `emitted.Reset()` purely to assert `len(msgs) == 1`, `Type == "ERROR"`, `Body.Error == "POTION_LOCKED"`. This matches the brief's instruction to assert the emitted-message shape once, without weakening the per-row return-value/reserve assertions — both in-scope rows still independently prove the sentinel and the non-reservation.

## Not evaluable

- **`buff.IsPotionLocked`, `ErrPotionLocked`, `consumeErrorType`, `ErrorTypePotionLocked` correctness** — these are Task 1/Task 2 interfaces this unit only consumes. I read them (`character/buff/model.go:87-99`) to confirm the FR-3 magnitude claim but did not re-review their own test coverage; that belongs to the Task 1/Task 2 review units.
- **Channel-side behavior when the client receives the `POTION_LOCKED` ERROR event** (releasing the item-use exclusive request) — this is Task 4 and lives entirely outside this diff's file list; not evaluated here per the binding scope constraint.

## Verdict rationale

Every binding constraint in the task brief is satisfied and independently verified (build, full test run, mechanical revert to prove test honesty, and direct reads of the touched call sites). No blocking findings.
