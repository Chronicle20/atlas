# Review: Task 9 — atlas-channel back-effect REST client and map-enter replay

Commit range: `62b7d981b..be28fe948` (single commit `be28fe948`)
Brief: `.superpowers/sdd/plan/task-9-brief.md`
Report: `.superpowers/sdd/plan/task-9-report.md`

## Scope

Reviewed the full diff: `services/atlas-channel/atlas.com/channel/backeffect/{rest,requests,processor}.go`,
`backeffect/mock/processor.go`, and the additions to
`kafka/consumer/map/consumer.go` and `consumer_test.go`. Cross-checked
against `services/atlas-maps/atlas.com/maps/map/backeffect/rest.go` (server
model, Task 7) and `services/atlas-channel/atlas.com/channel/jukebox/requests.go`
(precedent pattern) as contract dependencies. Ran the new tests, `go build
./...`, and `go vet` on the touched packages myself rather than trusting the
report.

## Findings

### 1. Client `RestModel` vs server `RestModel` — PASS

Diffed both files side by side myself:

Server (`services/atlas-maps/atlas.com/maps/map/backeffect/rest.go:5-11`):
```go
type RestModel struct {
	Id       string `json:"-"`
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
func (m RestModel) GetName() string { return "backEffect" }   // line 19
```

Client (`services/atlas-channel/atlas.com/channel/backeffect/rest.go:6-12,17-19`):
```go
type RestModel struct {
	Id       string `json:"-"`
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
func (m RestModel) GetName() string { return "backEffect" }
```

Field names, order, Go types, and JSON tags are identical field-for-field.
`GetName()` returns `"backEffect"` on both sides. The client additionally
defines `SetToOneReferenceID`/`SetToManyReferenceIDs` no-ops
(`rest.go:29-31`) and an `Extract` free function (`rest.go:33-35`), which are
consumer-side plumbing the server model has no need for (it doesn't decode
responses) — consistent with `events/rest.go`'s equivalent shape.

### 2. Resource path — PASS

`services/atlas-channel/atlas.com/channel/backeffect/requests.go:10-13,15-25`:
```go
mapInstanceResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
mapInstanceBackEffectResource = mapInstanceResource + "/backEffects"
...
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}
func requestBackEffectsInMap(ctx context.Context, f field.Model) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	...
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+mapInstanceBackEffectResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
```
Diffed byte-for-byte against `jukebox/requests.go:12-26` — identical
structure (`mapInstanceResource` base constant, `RootUrlFor(ctx, "MAPS")`,
same argument order), only the suffix (`/backEffects` vs `/jukebox`) and the
collection type (`[]RestModel` vs `RestModel`, matching the `events`
collection-request precedent per the brief) differ. Final resolved path is
exactly `worlds/%d/channels/%d/maps/%d/instances/%s/backEffects`.

### 3. `Duration = 0` replay asserted at the decoded-wire level — PASS

`TestAnnounceActiveBackEffects_ReplaysWithZeroDuration`
(`kafka/consumer/map/consumer_test.go:975-1005`) stubs `doorAnnounce`
(`stubDoorAnnounceForBackEffect`, pre-existing from Task 8,
`consumer_test.go:809-822`) to capture the raw encoded `[]byte` body, then
decodes it through the **real** `fieldcb.SetBackEffect.Decode` codec
(`decodeSetBackEffect`, pre-existing from Task 8, `consumer_test.go:824-835`,
unmodified by this diff) and asserts on the decoded struct's
`Effect()/FieldId()/PageId()/Duration()` accessors. The stub server returns
`duration: 1000` / `500` (`consumer_test.go:983-985`), and the assertions
require `Duration() == 0` for both (`consumer_test.go:996, 1004`) while
`Effect`/`FieldId`/`PageId` are asserted to replay exactly as stored. This is
a genuine wire round-trip check, not a struct-equality check on the pre-encode
value — confirmed by reading the actual codec call, not by trusting the
report's claim.

### 4. Fail-open on lookup error (PRD FR-5) — PASS, test-enforced

`TestAnnounceActiveBackEffects_LookupFailureIsFailOpen`
(`consumer_test.go:1012-1026`) points the stub server at HTTP 404, calls
`announceActiveBackEffects` directly (not through a panic-recovery wrapper —
a genuine panic would fail the test run), and asserts `len(calls) == 0`. The
production code path (`consumer.go:1221-1224`) is:
```go
es, err := backeffect.NewProcessor(l, ctx).GetActive(f)
if err != nil {
	return
}
```
No error is returned to the caller (the function signature is `func(...)`
with no return value), and `announceActiveBackEffects` itself runs inside its
own `routine.Go` block (`consumer.go:396-398`), isolated from the map-entry
`return nil` path. Re-ran the test myself:
```
=== RUN   TestAnnounceActiveBackEffects_LookupFailureIsFailOpen
--- PASS: TestAnnounceActiveBackEffects_LookupFailureIsFailOpen (0.00s)
```

### 5. Seam-mechanism reading (implementer's flagged concern) — ruled: correct-per-convention, not a defect

Checked the actual precedent myself:
- `announceActiveVisuals` (`consumer.go:802-803`) calls
  `events.NewProcessor(l, ctx).ActiveVisualsInMap(f)` directly — no
  package-level `Processor` seam, no mock injection. Its own tests stand up a
  real `httptest.Server`.
- `grep -rn "jukebox/mock\|events/mock\|backeffect/mock"
  services/atlas-channel/atlas.com/channel` (excluding the mock packages'
  own definitions) returns **no matches** — neither `jukebox/mock` nor
  `events/mock` is referenced anywhere in the module today, confirming they
  are unused scaffolding, same status as the new `backeffect/mock`.
- The brief's own Step 4 code snippet (task-9-brief.md:98-99) calls
  `backeffect.NewProcessor(l, ctx).GetActive(f)` directly — the same direct-
  call shape as `announceActiveVisuals`, not a mock-injected seam.

The implementer's reading is correct: the brief's Step 1 prose
("`backeffectmock.ProcessorMock`... whatever `announceActiveVisuals` already
does") describes a seam that does not exist in the codebase; the brief's own
Step 4 code and the actual `announceActiveVisuals`/`announceActiveJukebox`
precedent both use direct processor calls plus the `doorAnnounce` write-seam.
Following the literal Step 4 code was the correct disambiguation, and the
unwired `backeffect/mock/processor.go` matches (does not diverge from) the
existing `jukebox/mock` / `events/mock` precedent exactly. This is a defect
in the brief's prose, not in the implementation — non-blocking, correctly
self-reported.

### 6. Unchecked error returns (errcheck class) — PASS

Manually swept every new line for error-returning calls:
- `requests.go:19-21`: `getBaseRequest` error checked (`if err != nil`).
- `consumer.go:1221-1224`: `GetActive` error checked.
- `consumer.go:1226`: `_ = doorAnnounce(...)` — explicit discard, matching
  the identical pattern in `announceActiveJukebox`/`announceActiveVisuals`
  (existing convention; `doorAnnounce`'s error is intentionally not
  propagated at any of the three call sites).
- `consumer_test.go` new `backEffectServer` helper: `_, _ =
  w.Write([]byte(body))` (`consumer_test.go:919` in diff) — explicit
  discard, matching `jukeboxServer`'s identical pattern (this is the same
  class of finding a prior gate flagged in a different file; here it is
  handled correctly).
- `mock/processor.go`: no error-returning calls.
- `decodeSetBackEffect` (the only line in the new tests that calls a
  Decode-family function) is **pre-existing** from Task 8, not part of this
  diff — confirmed via `git diff` on `consumer_test.go`, which shows the new
  hunk starting after line 945, well past the pre-existing helper's
  definition. Not this task's responsibility.

No unchecked error returns in code introduced by this commit.

## Additional checks

- Dispatch placement: `routine.Go(l, ctx, func(_ context.Context) {
  announceActiveBackEffects(l, ctx, wp, f, s) })` sits immediately after the
  `announceActiveJukebox` `routine.Go` block and before `return nil`
  (`consumer.go:396-401`), matching the brief's Step 4 placement instruction
  exactly.
- `go build ./...` (module root `services/atlas-channel/atlas.com/channel`):
  clean, no errors.
- `go vet ./backeffect/... ./kafka/consumer/map/...`: clean, no output.
- `go test ./kafka/consumer/map/... -run AnnounceActiveBackEffects -v`: all
  three new tests pass (re-ran myself, not trusting the report's pasted
  output):
  ```
  --- PASS: TestAnnounceActiveBackEffects_ReplaysWithZeroDuration (0.02s)
  --- PASS: TestAnnounceActiveBackEffects_EmptyAnnouncesNothing (0.00s)
  --- PASS: TestAnnounceActiveBackEffects_LookupFailureIsFailOpen (0.00s)
  PASS
  ```
- Order preservation: the two-call test (finding 3) asserts `calls[0]` is
  `PageId:1` and `calls[1]` is `PageId:2` in that order, matching the
  server's returned slice order iterated by a plain `for _, e := range es`
  loop (`consumer.go:1225`) with no re-sorting.
- Empty-set test (`TestAnnounceActiveBackEffects_EmptyAnnouncesNothing`)
  correctly distinguishes "zero results, no error" from "lookup error" —
  both produce zero announces, but only the latter is the FR-5 fail-open
  case; the empty case just exercises the `for` loop over a zero-length
  slice. Both are separately tested.

## Not evaluable

None. The full review surface (client model, request builder, processor,
mock package, consumer wiring, and tests) was read and independently
verified against its stated contract dependencies (server `RestModel`,
jukebox request pattern, `announceActiveVisuals` precedent).

## Verdict rationale

All six review questions resolve to PASS or a correct, non-blocking
disposition. No defect was found in the diff itself. Item 5 identifies a
prose inconsistency in the *brief*, not the implementation, and the
implementer both flagged and correctly resolved it, documenting the
resolution for review — this is exactly the kind of self-caught ambiguity
that merits a note but not a blocking finding.
