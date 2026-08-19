# Task 26 review — world-transfer gate 12 `parcel_pending`

Commit: `65d9c2704` "feat(character): world-transfer gate 12 parcel_pending"
Brief: `.superpowers/sdd/plan/task-26-brief.md`
Module: `services/atlas-character/atlas.com/character`

## Scope

Reviewed the full diff of `65d9c2704` (5 files, +282/-8):
`pending_change/processor_eligibility.go`, `processor_eligibility_test.go`,
`requests.go`, `requests_test.go`, `rest.go`. Cross-checked the one seam file
(`rest.go`'s `parcelStatusRestModel`) against the real upstream producer,
`services/atlas-parcel/atlas.com/parcel/parcel/rest.go`. Ran the package's
tests and `go vet`/`gofmt` against the committed tree (not the report's
narrative).

`scope_confirmed`: the committed diff matches the brief's Files list plus one
unbriefed-but-necessary type (`parcelStatusRestModel` in `rest.go`, needed to
decode the REST response `requests.go` calls — a genuine implementation
requirement, not scope creep).

## Findings

### 1. Both-functions invariant — PASS

`processor_eligibility.go:211` (`evaluateTransferEligibility`) and `:242`
(`evaluateTransferEligibilityIndependent`) both append
`func() (string, bool, error) { return p.checkParcelPending(c) }` as the last
gate, immediately after `checkMtsHolding(c)` (gate 11), with identical
comment text. Verified by direct read, not the report's claim — same relative
position in both slices.

### 2. Fail-closed, correct reason — PASS

- `runGates` (`processor_eligibility.go:139-150`): any gate error short-circuits
  to `reject("check_unavailable")`, never the gate's own reason.
- `checkParcelPending` (`processor_eligibility.go:405-416`): on
  `p.gates.parcelPending` error, logs and returns `("", false, err)` —
  matches the fail-closed contract; the affirmative reason `"parcel_pending"`
  is only returned in the `err == nil, pending == true` branch.
- The two `dependency error` subtests
  (`processor_eligibility_test.go:551-579`) assert
  `reason != "check_unavailable"` explicitly, not merely `ok == false` — they
  would fail if `checkParcelPending` (or `runGates`) leaked `"parcel_pending"`
  on an error path. Confirmed by reading the assertion lines, not just their
  names.

### 3. The unbriefed file (`rest.go`) — PASS

`rest.go:218-225`:
```go
type parcelStatusRestModel struct {
	Id       string `json:"-"`
	InFlight bool   `json:"inFlight"`
}
func (r parcelStatusRestModel) GetName() string        { return "parcel-statuses" }
func (r parcelStatusRestModel) GetID() string          { return r.Id }
func (r *parcelStatusRestModel) SetID(id string) error { r.Id = id; return nil }
```
Cross-checked directly against the real producer,
`services/atlas-parcel/atlas.com/parcel/parcel/rest.go` (`parcelStatusRestModel`,
`GetName() → "parcel-statuses"`, `transformParcelStatus` setting
`Id: strconv.FormatUint(uint64(characterId), 10)`, `InFlight: inFlight`).
Resource type, attribute name/tag, and id encoding all agree. This seam is
correct.

### 4. REST client's error path (`requests.go`) — PASS

`parcelPending` (`requests.go:208-214`):
```go
func parcelPending(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error) {
	rm, err := requestParcelStatus(characterId)(l, ctx)
	if err != nil {
		return false, err
	}
	return rm.InFlight, nil
}
```
Never returns `(false, nil)` on a request error — matches `mtsHoldingOpen`'s
shape. The `service down` subtest (`requests_test.go:140-156`) drives a real
503 through `httptest.NewServer` and asserts `err == nil` is a failure
(`t.Fatal` if so) — this is a genuine assertion on the error path, not just
`pending == false`.

### 5. Test-table completeness — PASS

All 7 `processor_eligibility_test.go` subtests from the brief's table exist
(`blocks buy-time`, `blocks check-time`, `passes buy-time`, `passes
check-time`, `dependency error buy-time`, `dependency error check-time`,
`runs after mts`) and assert exactly the reason/ok pairs the table specifies.
`runs after mts` (`processor_eligibility_test.go:581-611`) stubs both
`mtsHolding` and `parcelPending` to reject and asserts
`reason == "mts_listings_open"` on both entry points — this pins gate 11
preceding gate 12 and would fail if the order in either evaluate function
were reversed.

All 3 `requests_test.go` subtests exist (`in flight`, `not in flight`,
`service down`) and match the table.

`passingGateDeps()` (`processor_eligibility_test.go:44-46`) was updated to
include a passing `parcelPending` stub, so the pre-existing gate 1-11 tests
that reuse it were not silently broken by the new field.

Ran the tests against the committed tree directly (not the report's
narrative):
```
go test ./pending_change/... -run "Parcel" -v   → all subtests PASS
go test ./pending_change/...                    → ok (full package, no regressions)
go vet ./pending_change/...                      → clean
gofmt -l pending_change/*.go                     → no output (formatted)
```

### 6. `checkParcelPending` vs `checkMtsHolding` shape — PASS

`checkMtsHolding` (`processor_eligibility.go:389-398`) and `checkParcelPending`
(`processor_eligibility.go:405-416`) are structurally identical: same
dependency-call → error-log (`p.l.WithError(err).Errorf("Unable to check
... for character [%d].", c.Id())`) → `return "", false, err` → affirmative
check → `return "<reason>", true, nil` → `return "", false, nil` shape. Not a
divergent variant.

## Non-blocking

- `processor_eligibility.go:152-169` and `:216-223` — the doc comments above
  `evaluateTransferEligibility` and `evaluateTransferEligibilityIndependent`
  still say "Gates 2, 6-11 are destination-INDEPENDENT" / "gates (2, 6-11 of
  evaluateTransferEligibility's table...)" without updating to "2, 6-12" or
  similar. The code is correct (gate 12 does run in both, as verified above);
  only the enumerated-range prose in these two doc comments is now stale by
  one gate. Cosmetic, not a functional defect — flagging so it doesn't
  compound the next time a gate is added.

## Not evaluable

- None. The full diff, its one unbriefed file, and the one live cross-service
  seam (`rest.go` vs atlas-parcel's producer) were all inspected directly
  against source, and the tests were executed against the committed tree.

## Verdict rationale

Every point the controller flagged for explicit verification checks out
against the actual committed code (not the implementer's report), with tests
that demonstrably assert the load-bearing behavior (reason strings, not just
booleans; a genuine error rather than swallowed nil). The one blemish is a
stale doc-comment range that doesn't affect behavior.
