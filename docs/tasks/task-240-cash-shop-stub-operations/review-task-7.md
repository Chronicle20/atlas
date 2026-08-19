# Review: Task 7 — GET_PURCHASE_RECORD (mode 40/44)

Commit range: `cb9eeb9..4871c8581` (single commit `4871c8581`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git diff --stat cb9eeb9..4871c8581` — 10 files, 301 insertions, 1 deletion, across
`services/atlas-cashshop/atlas.com/cashshop` (main.go, `purchaserecord/resource.go`,
`purchaserecord/rest.go`, `rest/handler.go`) and
`services/atlas-channel/atlas.com/channel` (new `cashshop/purchaserecord/` package —
model.go, rest.go, requests.go, processor.go — plus the `cash_shop_operation.go`
handler edit and a new test file). Matches the brief's file list exactly. No
churn outside this surface; the concurrent Task 6 backfill fix (`purchaserecord/backfill.go`)
is untouched by this commit, confirmed by `git show --stat`.

## Findings

### 1. Miss is 200/Purchased=false end to end — PASS

- `purchaserecord/administrator.go:44-54` (Task 5/6, read-through target): `Get`
  returns `(0, nil)` on `gorm.ErrRecordNotFound` — never an error for a miss.
- `purchaserecord/resource.go:23-34`: on `err == nil`, builds
  `RestModel{Purchased: count > 0, Count: count}` and marshals `200` unconditionally.
  Only a genuine non-`ErrRecordNotFound` DB error goes through `WriteErrorResponse`.
- Channel client `cashshop/purchaserecord/rest.go` `Extract` passes `Purchased`
  straight through from the REST body; no channel-side reinterpretation.
- Channel handler `cash_shop_operation.go:203-223`: on a successful `GetForAccount`
  (miss or hit, both `err == nil`) it announces `CashShopPurchaseRecordDoneBody`
  with `purchaseRecordFlag(m.Count())` — `0` for a miss, `1` for any count > 0.
  Confirmed end to end: a miss never surfaces as a 404, a dropped packet, or a
  silent no-response.

### 2. No Kafka — PASS

`git show 4871c8581 | grep -in "kafka\|ErrorEventBody\|consumer"` returns nothing.
No command type, no `ErrorEventBody` arm, no consumer arm added. Pure synchronous
REST read, as directed.

### 3. No stub left behind; every branch answers — PASS

`cash_shop_operation.go:203-223` fully replaces the log-only arm (previously just
`l.Infof(...)` then `return`). Every branch now answers the client:
- REST read succeeds → `CashShopPurchaseRecordDoneBody` announced.
- REST read errors (cashshop down, timeout, any non-nil `err` from
  `GetForAccount`) → logged, then `CashShopPurchaseRecordFailedBody("unknown_error")`
  announced, then `return`. No silent drop — this satisfies the brief's FR-X-2
  instruction and matches the sibling `CashShopLoadWishFailedBody` pattern at
  `cash_shop_operation.go:188` (same file, same shape, pre-existing).
- The announce call itself can fail (network layer); on that failure it is
  logged (`l.WithError(err).Errorf(...)`) and the handler returns — this mirrors
  every other announce site in the file (e.g. lines 81, 198, 424, 444, 483) and
  is not a task-7-introduced gap.

### 4. REST-call-error path is defensible — PASS

Traced: cashshop down/timeout → `requests.GetRequest` returns a non-nil `error`
→ `ProcessorImpl.GetForAccount` (channel `processor.go:38-40`) propagates it →
handler's `err != nil` branch fires → client gets `PURCHASE_RECORD_FAILED`
("unknown_error") rather than hanging or silently dropping. This is the same
contract the brief specified (Step 5) and is a reasonable definitive answer for
an infra failure, distinct from the "miss" case (which is always 200/false).

### 5. Version gating — PASS (no gate needed, and none invented)

No `MajorAtLeast`/raw `MajorVersion()` gate was added for this arm. Verified this
is defensible, not an omission: `PurchaseRecordDone.Encode` (`shop_operation_result_misc.go:164-171`)
writes a fixed `mode + int32 + byte` layout regardless of version; the only
version-varying piece is the mode byte, and that is resolved at runtime via
`WithResolvedCode("operations", CashShopOperationPurchaseRecordDone, ...)` against
the tenant's per-version operations table — the same mechanism every other arm
in this file uses, none of which carry a `MajorAtLeast` gate either. Grepped
`cash_shop_operation.go` for `MajorAtLeast`/`MajorVersion`/`IsRegion`: only one
unrelated hit (line 481, world-transfer-failure logging, pre-existing). No raw
`MajorVersion() >= N` idiom introduced.

### 6. Pattern-copy requirement — PASS

- Cashshop side: `purchaserecord/resource.go` mirrors `wishlist/resource.go`'s
  shape exactly — `InitResource(si) -> func(db) -> RouteInitializer`, subrouter +
  `registerGet`, `rest.ParseXId` chaining, `NewProcessor(...).<verb>`,
  `MarshalResponse[RestModel]`. `rest.go`'s `RestModel`/`GetName`/`GetID`/`SetID`
  matches the convention used elsewhere in the package (e.g. wishlist's own
  RestModel identity methods).
- Channel side: `cashshop/purchaserecord/{model,rest,requests,processor}.go` is
  file-for-file against `cashshop/wishlist/`'s four files, with the expected
  single-object-vs-list differences called out honestly in the report (no
  `DrainProvider`/pagination since this is a single GET, correctly following
  `wallet/requests.go`'s single-object shape instead — a reasonable adaptation,
  not an invented shape).

### 7. Tenant scoping — PASS

`purchaserecord/processor.go` (Task 5, now wired in): `ProcessorImpl.t` is set
via `tenant.MustFromContext(ctx)` in `NewProcessor`, and `Get` calls the
package-level `Get(p.db.WithContext(p.ctx), p.t.Id(), accountId, serialNumber)`
— tenant id threads into the WHERE clause. The REST handler passes `d.Context()`
straight through, so the request-scoped tenant travels correctly. Channel-side
`requests.GetRequest`/`RootUrlFor(ctx, "CASHSHOP")` is the standard atlas-rest
mechanism that also carries tenant headers — same as every other cashshop client
package in this codebase (wishlist, wallet, etc.), not something this commit
had to reinvent.

### 8. Carry-over question 1 — Task 5's `Processor` wrapper — RESOLVED, used

`purchaserecord/resource.go:23`: `NewProcessor(d.Logger(), d.Context(), db).Get(accountId, serialNumber)`.
This is exactly the Task-5 `purchaserecord/processor.go` `Processor`/`ProcessorImpl`
type flagged as unreferenced scaffolding in the Task 5 review. It is now the sole
call site inside `handleGetPurchaseRecord`. Confirmed by reading `processor.go`
directly (`Get` delegates to the tenant-scoped package `Get` function) and by
`go build`/`go test` passing with it wired in. The wrapper is not dead code —
it has exactly one consumer, added by this commit, and its `Record` method
(unused by this task) is presumably the future write-side hook for whichever
task performs the purchase-record write on an actual buy — reasonable to leave
unconsumed for now since it's an interface method, not a standalone unreferenced
symbol.

### 9. Carry-over question 2 — the `CashShopOperationPurchaseRecord` correction — VERIFIED

Independently grepped `libs/atlas-packet/cash/clientbound/shop_operation_body.go`:
- `CashShopOperationPurchaseRecord` (without "Done") does not exist anywhere in
  that file — confirmed by an exact-match grep returning zero hits.
- `CashShopOperationPurchaseRecordDone = "PURCHASE_RECORD"` exists at line 72.
- `CashShopPurchaseRecordDoneBody` (line 640) resolves against
  `WithResolvedCode("operations", CashShopOperationPurchaseRecordDone, ...)`
  (line 641) — the exact identifier the implementer's test and the handler use.

The brief's literal text was wrong; the implementer's correction is accurate and
no invented constant was introduced. The test (`cash_shop_purchase_record_test.go:20`)
and the handler (`cash_shop_operation.go`, via `CashShopPurchaseRecordDoneBody`)
both use the real identifier.

### 10. Wire-layout test fidelity — PASS

`TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag` (`cash_shop_purchase_record_test.go:17-51`)
asserts against `PurchaseRecordDone.Encode` (`shop_operation_result_misc.go:164-171`):
mode byte, then `WriteInt32` (little-endian, 4 bytes), then the purchased byte —
byte-for-byte match, table-driven over purchased/not-purchased. `TestPurchaseRecordFlag`
covers `0→0, 1→1, 7→1` exactly as the brief's table specifies.

## Build/test verification (run independently, both modules)

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Build clean; all packages `ok`, including `atlas-channel/socket/handler` (0.836s,
includes the new test).

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
Build clean; all packages `ok`, including `atlas-cashshop/purchaserecord`.

## TDD-sequencing note (non-blocking)

The report is honest that Step 1/Step 2 (write the test, watch it fail, then
implement) were not run as literal separate tool-call batches — the test and
`purchaseRecordFlag` were authored together since both were fully specified by
the brief's table. This is disclosed, not concealed, and does not affect the
correctness of the delivered code or test. Noted, not blocking.

## Not evaluable

- None. The full unit (both module sides, the channel handler, and the two
  carry-over questions) was traceable within this commit's diff plus the
  directly-called `purchaserecord` package files (Task 5/6 code the diff calls
  into, in scope per the review-surface rule).

## Verdict

APPROVED. No blocking findings. All brief requirements (miss=200/false, no
Kafka, no stub, error-path answers the client, `MajorAtLeast` idiom not needed
and not misused, pattern-copy fidelity, tenant scoping) are satisfied and
verified against `file:line` evidence. Both carry-over questions are closed:
the Task 5 `Processor` wrapper is now the sole call site in the new resource
handler (not dead code), and the brief's constant-name correction is verified
accurate with no invented identifier.
