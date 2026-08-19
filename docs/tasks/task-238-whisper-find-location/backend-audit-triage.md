# Controller triage of `backend-audit.md` (verdict CHANGES_REQUIRED, 4 blocking)

Reviewed at `1104053da`, merge base `6fb58bdc5`. Every finding was checked
against the merge base rather than accepted on the reviewer's word. Two of the
four are pre-existing conditions the branch did not introduce; one is genuinely
introduced; one is real-in-effect but rooted in pre-existing production code.

## 1. FILE-02 — `RestModel` in `requests.go` instead of `rest.go` — PRE-EXISTING

`git show 6fb58bdc5:services/atlas-channel/atlas.com/channel/maps/location/requests.go`
already contains `RestModel` and its full JSON:API method set (`GetName`,
`GetID`, `SetID`, `SetToOneReferenceID`, `SetToManyReferenceIDs`) at the merge
base. The package has never had a `rest.go` — its only files are `requests.go`,
`requests_test.go`, `resolve.go`.

This branch's contribution is one added field (`State`) on an already-misplaced
struct. **Not introduced here.** Fixing it means splitting a pre-existing file.

## 2. FILE-05 — new domain `Model` in `requests.go` — GENUINELY INTRODUCED

`Model`, `Get`, and `NewModelForTest` (`requests.go:88-139`) are new in this
branch (Tasks 6 and 8). The package has no `model.go`, so the new domain type
went into `requests.go` alongside the REST plumbing.

**This is the one unambiguously branch-introduced structural finding.** The
implementer followed the package's existing (non-conforming) local convention
rather than the guideline. Cheap to fix: split `model.go` / `rest.go` out of
`requests.go`; it is a pure file move plus imports, no behaviour change.

## 3. DOM-10 — GORM test bootstrap without `RegisterTenantCallbacks` — SPLIT

Repo-wide convention confirmed: **93 of 117** `*_test.go` files that call
`sqlite.Open` also call `RegisterTenantCallbacks`. So the guideline reflects the
dominant practice, not a dead rule.

- `character/location/processor_test.go:102-112` — `newTestDB` is **byte-identical
  to the merge base** (`git show 6fb58bdc5:...processor_test.go` lines 100-110).
  The branch's 137 added lines are all new test funcs; the helper is untouched.
  **Pre-existing, not this branch's.**
- `kafka/consumer/cashshop/consumer_test.go:23-33` — **new file** (Task 5), which
  copied the package-local helper shape. **Genuinely introduced.**

## 4. DOM-24 — tests reach a live, unstubbed Kafka producer — REAL, PRE-EXISTING ROOT

Confirmed in `kafka/consumer/cashshop/consumer.go:55,75`:
`p := _map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)`.

In `git diff 6fb58bdc5..HEAD` these two lines appear as **unchanged context
lines** — the branch refactored the handlers into `...Func(db)` closures around
them but never touched the producer construction. The producer is built inline
from a package function, so it is not injectable and cannot be stubbed from a
test.

**The branch did not cause this; the branch's new tests are the first thing to
execute it.** That is what produces the reviewer's observed 98.985s / 59.521s
package runtimes — each handler call attempts a real Kafka connection and waits
out its timeout.

The effect is real and recurring (every CI run pays it), but the fix is a
production-code change to a pre-existing consumer: thread the producer provider
in as a parameter the way `db` now is. That is a genuine scope expansion beyond
FR-1…FR-7, which is why it is being surfaced rather than silently taken or
silently dropped.

## Ruling

Findings 2 and the `cashshop/consumer_test.go` half of finding 3 are this
branch's and should be fixed on this branch. Findings 1 and 4, and the
`processor_test.go` half of 3, are pre-existing defects this branch made
visible. Decision on how far to widen is the user's — recorded in the section
below once made.
