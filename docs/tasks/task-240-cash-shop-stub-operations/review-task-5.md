# Review: Task 5 — Purchase records (`eb8563e85..e1a6d590a`, commit `e1a6d590a`)

## Scope

Single commit adding `purchaserecord` package to `services/atlas-cashshop/atlas.com/cashshop`,
wiring `Record` into `cashshop/processor.go`'s `Purchase` transaction, registering the migration
in `main.go`, and updating the `purchaseTestDatabase` test helper. Matches the brief's file list
exactly (`purchaserecord/{entity,administrator,processor,administrator_test}.go`,
`cashshop/processor.go`, `cashshop/processor_test.go`, `main.go`). `scope_confirmed`: reviewed
diff matches the brief's described unit of work.

## Findings

### 1. Interface signatures — PASS

`purchaserecord/entity.go:10` and `purchaserecord/administrator.go:17,42` — signatures are
character-for-character identical to the brief:

```go
func Migration(db *gorm.DB) error
func Record(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) error
func Get(db *gorm.DB, tenantId uuid.UUID, accountId uint32, serialNumber uint32) (uint32, error)
```

`Get` returns `(0, nil)` on `gorm.ErrRecordNotFound` (`administrator.go:52-54`), not an error —
confirmed by `purchaserecord/administrator_test.go:88-95` (`different account`), `:97-104`
(`different tenant`), and `:106-112` (`miss is not an error`), all asserting `err == nil`.

### 2. `Record` runs inside the purchase transaction — PASS

`cashshop/processor.go:229` calls `purchaserecord.Record(tx, p.t.Id(), c.AccountId(), serialNumber)`
where `tx` is the `*gorm.DB` handle bound by `database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {...})`
(`processor.go:110`), and the call sits after the asset-create success check and before the
`mb.Put(...PurchaseStatusEventProvider...)` — exactly where the brief specified (Step 6). On
error it sets `rejectEmit` to the same `ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", transactionId)`
shape used by every other in-transaction failure on this path and returns the error, which rolls
back the transaction (aborting the just-created wallet debit / pet / asset rows along with the
would-be purchase record). No separate db handle, no separate transaction.

### 3. Tenant scoping — PASS

`Record`'s upsert `Columns` are `tenant_id, account_id, serial_number` (`administrator.go:29-33`)
and `Get`'s `Where` clause is `tenant_id = ? AND account_id = ? AND serial_number = ?`
(`administrator.go:49`). The unique index `idx_purchase_record_unique` spans all three columns
(`entity.go:20-23`), and the test's `different tenant is separate` subtest
(`administrator_test.go:79-86`) proves cross-tenant isolation rather than merely asserting it —
same account/serial, different tenant, count `0`.

### 4. Migration registration — PASS

`main.go:64` — `purchaserecord.Migration` is in the `database.SetMigrations(...)` list, placed
after `opening.Migration` and before `coupon.Migration` per the brief.

### 5. `Processor` wrapper (`purchaserecord/processor.go`) — conformance PASS, non-blocking note

The wrapper (`Processor` interface, `ProcessorImpl{l, ctx, db, t}`, `NewProcessor`,
`var _ Processor = (*ProcessorImpl)(nil)`) is structurally identical to the established
package-level processor shape used elsewhere in this module — compared directly against
`wishlist/processor.go:19-49`, same field set and same `NewProcessor` construction via
`tenant.MustFromContext(ctx)`. This is the repo's normal Atlas shape, not ad hoc scaffolding.
It is currently unreferenced by any non-test caller (`grep -rn "purchaserecord.NewProcessor"`
returns nothing outside the package itself), which is expected since Task 7 (GET_PURCHASE_RECORD)
is the consumer and hasn't landed yet. Non-blocking: this is forward-looking dead code by design,
consistent with the brief's Interfaces block being satisfied by the package-level funcs alone; no
action required for Task 5 itself.

### 6. `purchaseTestDatabase` helper edit — PASS, minimal

`cashshop/processor_test.go:104` — only `purchaserecord.Migration` was appended to the existing
migration list; no assertions in `TestPurchaseTransactionId*` / `TestPurchaseZeroTransactionIdAccepted`
were touched. This is the minimal fix required once `Purchase` started writing to
`cash_purchase_records` inside the same in-memory DB — without it those pre-existing tests would
fail with `no such table: cash_purchase_records`. Confirmed no other lines in the diff for this
file changed.

### 7. Test honesty — PASS

`administrator_test.go`'s `TestRecordUpsertsAndCounts` follows the brief's table exactly
(first/second purchase, different serial, different account, different tenant, miss-is-not-error).
The `different serial is separate` subtest re-checks the *original* serial's count is still `2`
after a second serial is recorded (`administrator_test.go:70-76`), which would fail under a
column-scoping bug (e.g. an index that didn't include `serial_number`). These are genuine
behavior-pinning assertions, not tests that would pass unconditionally.

### 8. Build/test verification (run independently, not trusted from report)

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./...   # clean
go clean -testcache && go test ./purchaserecord/... ./cashshop/... -v
```

All PASS: `TestRecordUpsertsAndCounts` and its 6 subtests, `TestPurchaseTransactionIdSurvivesToSuccessEvent`,
`TestPurchaseTransactionIdSurvivesToErrorEvent`, `TestPurchaseTransactionIdDistinguishesConcurrentPurchases`,
`TestPurchaseInsufficientFundsReachesConsumer`, `TestPurchaseZeroTransactionIdAccepted`. Full
`go test ./...` from the module root is also clean (no FAIL). `go vet ./...` and `gofmt -l` on the
touched files report nothing.

## Not evaluable

- Task 7 (GET_PURCHASE_RECORD) consumer of `purchaserecord.Get`/`Processor` has not landed yet, so
  whether the `Processor` wrapper's shape actually fits that future caller cannot be checked from
  this unit alone — noted as a forward-looking risk, not a defect of this task.

## Verdict

APPROVED. No blocking findings.
