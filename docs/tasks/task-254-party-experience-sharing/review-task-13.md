# Review — Task 13: `atlas-monster-death` processor tests with mocks

Commit range: `21df4197c..22b8b603c`
Brief: `.superpowers/sdd/plan/task-13-brief.md`
Implementer report: `.superpowers/sdd/plan/task-13-report.md`

## Scope

`git diff --stat 21df4197c..22b8b603c` shows exactly one changed file:

```
.../monster/monster/processor_experience_test.go | 868 +++++++++++++++++++++
1 file changed, 868 insertions(+)
```

No production code changed. This matches the task's stated shape (an orchestration
test pass, new file only). `git diff --name-only` confirms the single file; a
`find . -name '*_testhelpers.go'` in the module found none.

Read in full: `processor_experience_test.go` (868 lines, 15 test functions), and the
production code it exercises — `monster/processor.go::DistributeExperience` and
`monster/experience.go::planDistribution` — to check assertions against real branch
behaviour rather than trusting the test's own framing.

## 1. Coverage against the brief's test table

The brief's table (lines 74-88) enumerates 15 rows. The file has exactly 15
`TestDistributeExperience_*` functions (confirmed via `grep -c`), one per row, same
names. Checked each row's assertion against the actual test body:

| Row | Test | Verdict |
|---|---|---|
| `OnePartyLookupPerParty` | `getByMemberIdCalls == 1` (line 109) | PASS — matches FR-2.4/NFR-1 |
| `OneRateLookupPerRecipient` | `getForCharacterCalls == 2` and `recordedIds == [1,2]` (lines 163-168) | PASS — matches FR-8.4 |
| `PartyLookupErrorFallsBackToSolo` | 1 award, `party == 0`, no panic (lines 214-219) | PASS — matches FR-2.3 |
| `PartyRecipientsCarryNonZeroParty` | both awards `party > 0` and `party == uint32(0.10*amount)` (lines 265-275) | PASS — matches FR-8.5 (0.05 × 2 members = 0.10, consistent with `PartyBonusPerMember`) |
| `SoloRecipientCarriesZeroParty` | 1 award, `party == 0`, `amount == 1000` (lines 321-329) | PASS |
| `ZeroDamageAwardsNothing` | 0 award, 0 hint, `err == nil` (lines 362-367) | PASS — matches FR-3.3, exercises the `totalDamage == 0` early return at `processor.go:217-220` |
| `OutOfFieldDamagerReceivesNothing` | 1 award (character 1 only), `getByMemberIdCalls == 1` (lines 418-426) | PASS — matches D12 |
| `ExcludedMemberGetsExactlyOneHint` | 1 hint, `characterId == 2`, `width/height == 0/0`, hint text == `levelGateHintText(...)` (lines 474-486) | PASS — matches FR-6.7/6.8 |
| `HintFailureDoesNotAbortAwards` | 3 hint attempts, in-range contributor still awarded, `err == nil` (lines 545-553) | PASS — matches FR-6.10 |
| `AwardFailureDoesNotAbortOthers` | 3 award attempts, `err == nil` (lines 603-605) | PASS — matches FR-9.2 |
| `HintIsThrottledAcrossKills` | 1 hint after +30s, 2nd hint after +61s more (lines 658-669) | PASS — matches D10 |
| `GateDisabledEmitsNoHint` | 0 hint, both members awarded (lines 727-732) | PASS — matches FR-6.5 acceptance |
| `InformationErrorReturnsError` | error returned, 0 awards (lines 767-772) | PASS |
| `FieldErrorReturnsError` | error returned, 0 awards (lines 807-812) | PASS |
| `AwardOrderIsAscendingCharacterId` | award sequence exactly `[1,2,3]` from an out-of-order party (`3,1,2`) (lines 859-867) | PASS — matches FR-9.1 |

No row is silently dropped, weakened, or asserting something looser than specified.

Tenant/span propagation: the brief says this is covered structurally by reading both
bodies rather than a broker-dependent test. Verified independently:
`character/processor.go:33-34` and `system_message/processor.go:37-38` both call
`producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(...)` — same shape, confirming
the equivalence claim (the report only re-verified the character side from memory;
I independently read `system_message/processor.go` to close that gap).

## 2. Task 12's deferred not-evaluable item — the four orchestration behaviours

- **`routine.Go` concurrency fan-out** (`processor.go:230-239`, the concurrent
  `information.GetById` / `map.CharacterIdsInField` fetch joined by `wg.Wait()`):
  exercised by essentially every test in the file (they all reach this stage) and
  confirmed race-clean: `go test ./monster/ -run TestDistributeExperience -v -race`
  → all 15 pass, no race reports.
- **Degrade-to-solo paths**: both distinct triggers are covered — party-lookup
  *error* (`PartyLookupErrorFallsBackToSolo`, exercises `processor.go:273-280`) and
  party-lookup returning a zero-value `party.Model{}` i.e. `pt.Id() == 0`
  (`SoloRecipientCarriesZeroParty` and `OutOfFieldDamagerReceivesNothing`, exercises
  `processor.go:281-284`). Both are asserted, not merely reached.
- **Award-loop continue-on-error** (`processor.go:320-324`): `AwardFailureDoesNotAbortOthers`
  makes the *first* award fail and asserts all three attempts still happen and the
  overall call returns `nil`. This is a real assertion on loop behaviour, not a
  vacuous pass.
- **Hint throttling** (`processor.go:330-337`, `system_message.Throttle`):
  `HintIsThrottledAcrossKills` uses a fake clock advanced by concrete deltas (30s,
  then +61s) straddling the 60s window and asserts the call count changes exactly
  when expected. This is a real clock-dependent assertion, not a static counter.

All four are genuinely exercised by asserting tests, closing Task 12's deferred item.

## 3. Test soundness — vacuous-assertion check

Checked every assertion for the "counter >= 0" / "mock never reached" / "throttle
clock never advances" failure modes called out in the brief:

- No test asserts a count with `>=`; all use `==` against an exact expected value.
- Traced the Kafka-dial trap the report self-reports (tests that omit
  `WithCharacterProcessor`/`WithSystemMessageProcessor` and then reach the real,
  non-mocked `character.NewProcessor`/`system_message.NewProcessor`, which would
  dial a real broker). Cross-checked every one of the 15 tests against
  `plan.Exclusions` reachability (`experience.go:283-284`, only populated inside the
  per-party loop, never for solo recipients) and confirmed:
  - Tests with only solo recipients (`PartyLookupErrorFallsBackToSolo`,
    `SoloRecipientCarriesZeroParty`, `OutOfFieldDamagerReceivesNothing`) correctly
    omit `WithSystemMessageProcessor` because `plan.Exclusions` is provably empty on
    those paths — `p.smp` is never touched, so the omission is safe, not an
    oversight.
  - All four tests the report calls out as previously reaching the real producer
    (`OnePartyLookupPerParty`, `OneRateLookupPerRecipient`,
    `ExcludedMemberGetsExactlyOneHint`, `HintIsThrottledAcrossKills`) now carry an
    explicit `charactermock.ProcessorMock{}`/`systemmessagemock.ProcessorMock{}`
    override in the committed file (verified by reading, e.g. lines 87-88, 145-146,
    458 and 465, 637).
  - Tests that never reach the award/hint phase at all (`InformationErrorReturnsError`,
    `FieldErrorReturnsError`) return before `p.pp`/`p.cp`/`p.smp` are invoked
    (`processor.go:241-248`), so a missing mock there is inert.
  - Ran the full suite with `-race` myself (see §2) — no test hung or timed out,
    consistent with the report's claim that the fix is in and the trap is closed.
- No test relies on the two known-benign `processor.go` quirks (the `CreateDrops`
  WARN and the unreachable `pt.Id() == 0` branch around lines 132-143) — those live
  in `CreateDrops`, which this file does not touch or test at all.

## 4. Determinism

- The `routine.Go` fan-out is joined by `wg.Wait()` before any of its results are
  used (`processor.go:239`); no assertion in the test file depends on goroutine
  scheduling order.
- Every one of the 15 tests constructs its own `clock := time.Now()` and passes
  `WithHintThrottle(newTestThrottle(&clock))` (verified by `grep`, present in all 15
  test bodies) — the process-wide `system_message.GetHintThrottle()` singleton is
  never touched by this file, so no throttle state leaks between tests or from a
  prior/parallel test run.
- `AwardOrderIsAscendingCharacterId` explicitly feeds party members in a
  scrambled order (`3, 1, 2`) and asserts the award callback fires in ascending
  order — this proves the *production* sort (`experience.go:331`), not an
  incidental map-iteration order in the test.

## Verification performed

```
go build ./...                                                    # clean
go vet ./monster/...                                               # clean
go test ./monster/ -run TestDistributeExperience -v -race         # 15/15 PASS, no race reports
git diff --stat / --name-only 21df4197c..22b8b603c                # test file only
find . -name '*_testhelpers.go'                                    # none
```

## Findings

No blocking findings.

Non-blocking:
- The implementer's report notes (and I independently confirmed) that
  `NewProcessor(...).With(...)` leaves any un-overridden collaborator wired to the
  real, Kafka-producer-backed implementation — a latent footgun for anyone adding a
  test to this file later without overriding `WithCharacterProcessor`/
  `WithSystemMessageProcessor` once the code path reaches AWARD or HINT. Not a
  defect in this commit (already fixed here), but worth carrying forward as a note
  for future contributors to this file.

## Not evaluable

None — the full scope (test file coverage, the four deferred orchestration
behaviours, vacuous-assertion risk, determinism) was directly checkable from the
diff plus `processor.go`/`experience.go`, both already in scope as the production
code the tests exercise.
