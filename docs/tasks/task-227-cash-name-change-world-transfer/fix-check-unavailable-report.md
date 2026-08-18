# Fix: report eligibility check outages as `check_unavailable`

Implements step 3 only of the "Suggested fix order" in
`bug-world-transfer-eligibility-reasons.md`. Steps 1, 2, 5 were already
landed. Step 4 (splitting the gate table, wiring the CHECK-time rejection
path) is explicitly out of scope for this unit and was not touched.

## What changed

### 1. `services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go`

`evaluateTransferEligibility`'s doc comment now explains the distinction
between "fails closed" (unchanged — always refuses the transfer) and "reports
an outage as a finding" (the bug, now fixed).

Every dependency-error branch across the 9 gates that call a remote/local
dependency now rejects with `"check_unavailable"` instead of the gate's
affirmative reason:

- Gate 3 (`worldStatus`) — was `world_unknown`
- Gate 4 (`accountSlots`, and the local `GetForAccountInWorld` count) — was
  `no_character_slot` (both error sites)
- Gate 5 (local `GetForName`) — was `name_taken`
- Gate 6 (`banned`) — was `banned`
- Gate 7 (`guildTitle`) — was `is_guild_master`
- Gate 8 (`inFamily`) — was `in_family`
- Gate 9 (`tradeOpen`) — was `trade_open`
- Gate 10 (`merchantOpen`) — was `merchant_open`
- Gate 11 (`mtsHolding`) — was `mts_listings_open`

Gates 1 (`world_same`) and 2 (`is_gm`) have no dependency call and so no
error path — nothing to change there. The `p.l.WithError(err).Errorf(...)`
log line before every `reject(...)` call is untouched: it still names the
real dependency and the underlying error, at error level, before the
info-level `reject` log records the (now-honest) reason.

This is the "uniform change" the brief asked for: every error branch's
literal was swapped to the same string, rather than restructuring the gate
table (explicitly deferred to step 4).

### 2. `docs/tasks/task-227-cash-name-change-world-transfer/design.md`

§6 "Reason taxonomy" gains `check_unavailable` in the closed-set list, plus a
paragraph explaining it is distinct from the 9 affirmative reasons it sits
alongside and why (design is the contract other code reads, per the brief).

### 3. `libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go`

The CHECK-time reason → arm mapper (`checkTransferWorldPossibleReasonArms`,
`checkTransferWorldPossibleReasonKey`, `CheckTransferWorldPossibleResultRejectedBody`)
now includes `check_unavailable`, routed to `CheckTransferWorldPossibleUnknownError`
(arm 9 / UNKNOWN_ERROR) — it is not one of the arms with independently
confirmed distinct client text, so it collapses the same way an unrecognised
reason does. Doc comments updated to list it alongside the other 9 arms that
already fold to UNKNOWN_ERROR (now framed as 10, not 9).

### 4. `libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result_test.go`

`TestCheckTransferWorldPossibleResultReasonMapping`'s `required` slice gained
`"check_unavailable"`. This is the exhaustiveness test the brief warned would
fail until the arm table was updated — it now passes with `check_unavailable`
asserted to resolve to `UNKNOWN_ERROR` / wire code 9, same as every other
non-`in_family` reason in the table.

### 5. `services/atlas-character/atlas.com/character/pending_change/processor_eligibility_test.go`

Added, per the brief, an error-injection case per gate:

- `TestEligibilityGateErrorsReportCheckUnavailable` — table-driven, covers
  the 8 `gateDeps`-seam dependencies (`worldStatus`, `accountSlots`,
  `banned`, `guildTitle`, `inFamily`, `tradeOpen`, `merchantOpen`,
  `mtsHolding`) each injected with a real `error` value; asserts
  `ok=false, reason="check_unavailable"` for each.
- `TestEligibilityGate4ExistingCharacterCountErrorReportsCheckUnavailable` —
  gate 4's *second* dependency (the local `character.GetForAccountInWorld`
  count) is not part of the `gateDeps` seam, so this test forces the error by
  closing the underlying `sql.DB` pool, exactly matching the existing
  precedent in `character/resource_test.go`'s
  `DatabaseFailureIsNotA200ZeroCharacter`.
- `TestEligibilityGate5NameTakenCheckErrorReportsCheckUnavailable` — same
  DB-close technique for gate 5's local `character.GetForName` call.

Together these cover all 9 reasons the diagnosis named (`world_unknown`,
`no_character_slot`, `name_taken`, `banned`, `is_guild_master`, `in_family`,
`trade_open`, `merchant_open`, `mts_listings_open`) plus the untouched-by-
design gates 1/2 which have no error path to test.

### 6. `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation_reason_test.go` (new)

Cross-service seam tests, traced by hand per CLAUDE.md's rule that the gate
cannot see this boundary:

- `TestWorldTransferRejectionReasonPassesCheckUnavailableThrough` — asserts
  `worldTransferRejectionReason` forwards `"check_unavailable"` verbatim
  (it already does, via `re.Reason`; this pins that the pass-through is not
  accidentally narrowed later).
- `TestCashShopTransferWorldFailedBodyResolvesCheckUnavailableAsConfigured` —
  end-to-end through `CashShopTransferWorldFailedBody`: builds an `errors`
  options table with distinct codes for `check_unavailable` (231) and
  `unknown_error` (99), and asserts the encoded `errorCode` byte is 231 —
  i.e. the raw reason, not a folded-down `unknown_error`, is what reaches
  `ResolveCode`. This is the NEW contract; the old behavior (silent fold to
  `unknown_error`) would have failed this assertion.
- `TestCheckTransferWorldPossibleReasonKeyRoutesCheckUnavailableToUnknownError` —
  end-to-end through `CheckTransferWorldPossibleResultRejectedBody` with a
  tenant-bearing context (`mustTenant`, required because
  `CheckTransferWorldPossibleResult.Encode` reads `tenant.MustFromContext`);
  asserts the resolved result byte is UNKNOWN_ERROR's configured code (9).

## Not done (explicitly out of scope for this unit)

- Splitting the gate table into destination-independent/dependent halves and
  wiring the CHECK-time rejection path (step 4) — left untouched, per the
  brief.
- Re-seeding or changing the BUY-time `errors`-table templates — step 5
  already seeded `check_unavailable -> PLEASE_TRY_AGAIN` into all applicable
  templates; not re-verified against a live client here, only that the wire
  path resolves whatever code the template configures.

## Verification

Module-local build + test, three modules, all commands `-count=1` (no
cached-result ambiguity) except the first (initial `go build` sanity check):

```
$ cd services/atlas-character/atlas.com/character && go build ./... && go test ./pending_change/...
ok  	atlas-character/pending_change	153.459s
```

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go test -count=1 ./...
ok  	atlas-channel/socket/handler	0.907s   (and 100+ other ok lines, no FAIL)
```

```
$ cd libs/atlas-packet && go build ./... && go test -count=1 ./...
(80 "ok" package results, no FAIL)
$ go test -count=1 ./cash/...
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound	0.034s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound	0.035s
```

All output pristine — no warnings, no skips.

## Self-review

- Every one of the 9 dependency-backed gates verified individually changed
  (grep for `check_unavailable` in `processor_eligibility.go` shows exactly
  9 occurrences in `reject(...)` calls plus the doc comment).
- Log lines (`p.l.WithError(err).Errorf(...)`) were left untouched — they
  still name the real dependency and error, satisfying "the log must still
  name the real dependency and the underlying error."
- Did not touch the gate table's shape/split, `RequestCheckTransferEligibility`
  wiring, or any CHECK-time handler — step 4 is untouched.
- Did not re-seed or alter any `template_*.json` files — step 5's seeding
  stands as landed.
- Cross-service seam traced by hand and asserted with a NEW-contract test
  (not just re-running old tests), per CLAUDE.md's explicit requirement for
  changes crossing a service boundary.

## Issues or concerns

None. `tools/verify.sh` was not run, per the implementer contract — that is
the controller's `atlas-verifier` dispatch to run.
