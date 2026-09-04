# Task 13 Review — `atlas-saga-orchestrator` AwardCraftedAsset handler wiring

Commit range: `7b07f6ee2..c96dd1f` (single commit `c96dd1f`)
Brief: `.superpowers/sdd/plan/task-13-brief.md`
Report: `.superpowers/sdd/plan/task-13-report.md`

## Scope reviewed

`git diff --stat 7b07f6ee2..c96dd1f`:

```
saga/event_acceptance.go |  1 +
saga/handler.go          | 23 ++++++
saga/handler_test.go     | 93 ++++++++++++++++++++++
saga/model.go             | 10 +++
4 files changed, 127 insertions(+)
```

Matches the brief's file list exactly (model.go, handler.go, event_acceptance.go, handler_test.go). `saga/rest.go` confirmed untouched, per the ruling. No files outside `saga/` in the orchestrator module were touched.

## Findings

### PASS — `saga/model.go` has both required changes

- Type alias: `AwardCraftedAsset = sharedsaga.AwardCraftedAsset` added in a new "Crafting actions" block above the Guild actions block (diff hunk at `const (` block near line 214).
- Payload alias: `AwardCraftedAssetPayload = sharedsaga.AwardCraftedAssetPayload` added to the payload alias block (near line 284).
- **Local** `case AwardCraftedAsset:` arm added inside this package's own `Step[T].UnmarshalJSON` (diff hunk near line 1582), duplicating the same `json.Unmarshal` pattern used for sibling cases (e.g. `ReleaseFromParcel`). This is exactly the change the brief calls out as easy to silently omit.
- Confirmed via `git show 7b07f6ee2:.../model.go | grep -c AwardCraftedAsset` → `0` before the commit, i.e. genuinely new, not a pre-existing stub.

### PASS — `saga/handler.go` wiring is complete

- `handleAwardCraftedAsset(s Saga, st Step[any]) error` added to the `Handler` interface (line ~141).
- `case AwardCraftedAsset: return h.handleAwardCraftedAsset, true` added to `GetHandler` (line ~826).
- Handler body (line 1084-1097) type-asserts the payload, and calls:
  ```go
  h.compP.RequestCreateItemWithExplicitStats(s.TransactionId(), payload.CharacterId, payload.TemplateId, payload.Quantity, time.Time{}, payload)
  ```
  Verified against the actual interface signature in `compartment/processor.go:37`:
  ```go
  RequestCreateItemWithExplicitStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, stats saga.AwardCraftedAssetPayload) error
  ```
  Argument order/types match. The saga's transaction id and the full payload (as the `stats` argument, satisfying "full stat block") are both forwarded correctly.
- `time.Time{}` for `expiration` is correct: confirmed `AwardCraftedAssetPayload` (`libs/atlas-saga/payloads.go:1078-1098`) has no `Expiration` field, so there is nothing else to forward. Consistent with the report's stated reasoning.
- Error path: on failure, logs via `h.logActionError` and returns the error — matches the sibling `handleAwardAsset` pattern.
- `"time"` import added to handler.go's import block (was not previously imported there) — build confirms this is required and non-conflicting.

### PASS — `saga/event_acceptance.go` reuses `AwardAsset`'s exact event kinds

```go
sharedsaga.AwardCraftedAsset: {EventKindAssetCreated, EventKindAssetQuantityChanged},
```
placed immediately above the existing:
```go
sharedsaga.AwardAsset:            {EventKindAssetCreated, EventKindAssetQuantityChanged},
```
Confirmed by direct read (`grep -n "sharedsaga.AwardAsset:"`) — same two constants, not invented. `TestAwardCraftedAssetEventAcceptance` asserts table equality against the live `AwardAsset` entry rather than hardcoding both constants a second time, which is a slightly stronger regression guard than a literal comparison (it would still catch drift if `AwardAsset`'s own entry changed later without doing so here).

### PASS — Test honesty

Verified by direct comparison against `7b07f6ee2` that `AwardCraftedAsset`, `handleAwardCraftedAsset`, and the `event_acceptance.go` entry did not exist pre-commit (`grep -c` → 0 in all three files). The four new tests reference these symbols directly, so they would not have compiled — let alone passed — before this commit. This satisfies the brief's Step 2 intent even though the implementer did not perform the separate "run and watch it fail" step as a discrete recorded action (report is honest about this: it substitutes a `grep` confirming the symbol was undefined beforehand).

`TestStepUnmarshalAwardCraftedAssetLocal` uses the exact same JSON literal as Task 10's `TestAwardCraftedAssetStepUnmarshal` (`libs/atlas-saga/payloads_test.go:145`) — confirmed byte-for-byte identical by direct comparison — and exercises this package's own `Step[any]` type, not the shared library's, so it specifically catches an omitted local `UnmarshalJSON` case.

`TestHandleAwardCraftedAssetRequestsCreationWithStats` mocks `compartment.Processor` via the pre-existing `WithCompartmentProcessor` production method (`saga/handler.go:300`, not a test-only helper — satisfies the repo's Builder/no-testhelpers convention) and asserts on every argument: transaction id, character id, template id, quantity, and the full payload as the stats block.

### PASS — Build, vet, gofmt, and full test suite

```
$ go build ./...        # clean
$ go vet ./...           # clean
$ gofmt -l saga/model.go saga/handler.go saga/event_acceptance.go saga/handler_test.go   # no output
$ go test ./saga/... -count=1 -run AwardCraftedAsset -v
--- PASS: TestGetHandlerResolvesAwardCraftedAsset
--- PASS: TestStepUnmarshalAwardCraftedAssetLocal
--- PASS: TestHandleAwardCraftedAssetRequestsCreationWithStats
--- PASS: TestAwardCraftedAssetEventAcceptance
$ go test ./... -count=1   # all packages ok, no FAIL lines
```

## Not evaluable

- The actual Kafka message shape `RequestCreateItemWithExplicitStats` produces, and its consumption by `atlas-inventory` (Task 12's contract), is out of this task's surface — Task 12 already wired and tested `compartment.Processor`; this task only calls the existing interface method correctly, which is confirmed above by signature match. Full producer-to-consumer trace belongs to Task 12's review, not this one.
- Task 14 (dispatch of the `AwardCraftedAsset` step from the crafting saga itself) is explicitly out of scope for this task and not evaluated here.

## Verdict

APPROVED. All four brief deliverables are present, correctly wired, cross-checked against the exact `AwardAsset` sibling entries (no invented constants), and covered by tests that provably would not compile/pass without the change. Build, vet, gofmt, and full module test suite are all clean.
