# Backend Audit — atlas-database (task-119-db-transaction-coverage)

- **Scope:** `libs/atlas-database/` Go changes on branch `task-119-db-transaction-coverage`
  - `libs/atlas-database/transaction.go` (isTransaction fix)
  - `libs/atlas-database/transaction_test.go` (new)
  - `libs/atlas-database/databasetest/failwrites.go` (new)
  - `libs/atlas-database/databasetest/failwrites_test.go` (new)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-12
- **Build:** PASS (`go build ./...` clean)
- **Tests:** PASS (`go test ./... -count=1`, `go test -race ./...`, `go vet ./...` all clean)
- **Overall:** PASS — nothing blocks a PR of these three commits.

## Build & Test Results

```
cd libs/atlas-database
go build ./...            -> clean
go vet ./...              -> clean
go test ./... -count=1    -> ok  database 0.013s;  ok databasetest 0.006s
go test -race ./...       -> ok  database 1.046s;  ok databasetest 1.022s
```

Only 4 Go files changed in the range (`git diff --name-only 61dcb3aeb9..5ce27de633 -- '*.go'`); no other module is touched. This is a shared-library/test-support change, not a domain package, so the `model.go`/`resource.go`-scoped DOM-01..DOM-20 items are Not Applicable by construction. Applicable checks were run and are recorded below.

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GATE | Build & tests (Phase 1) | PASS | vet/race/`-count=1` clean, see above |
| CORR-TX | Transaction detection is correct | PASS | `libs/atlas-database/transaction.go:20-25` uses `db.Statement.ConnPool.(gorm.TxCommitter)` + `ok && committer != nil` — byte-for-byte the guard GORM itself uses at `gorm.io/gorm@v1.31.2/finisher_api.go:640`. Interface defined at `interfaces.go:57-58`. |
| CORR-TX2 | Nil-safety of the guard | PASS | `transaction.go:21-23` returns false when `Statement == nil` or `ConnPool == nil` before the assertion; false → `db.Transaction(fn)` opens a real tx (safe default). |
| REGRESSION | Tests actually guard the fixed bug | PASS | `transaction_test.go:24-40` (`RollsBackOnError`) fails under the old `isTransaction` (which returned true for a root handle → fn ran with no tx → count would be 1). Under the fix a real tx opens and rolls back → count 0. Same guard via `failwrites_test.go:55-78`. |
| COVERAGE | Commit + nested + tenant-callback paths tested | PASS | `transaction_test.go:42-53` commit; `:55-70` nested-joins-outer; `:72-84` tenant create-callback stamps inside the tx. |
| DOM-10 | Test DB registers tenant callbacks | PASS | `databasetest/testdb.go:27` `database.RegisterTenantCallbacks(l, db)` inside `NewInMemoryTenantDB`, used by every new test. |
| DOM-21 | No duplication of atlas-constants types | PASS | New types `WriteVerb` (`failwrites.go:11`), `txEntity`/`fwEntity` (test entities) have no atlas-constants equivalent. |
| DOM-24 | Kafka producer stubbed in emitting tests | N/A | No test in these packages emits Kafka (`AndEmit`/`message.Emit`/`producer.Produce` absent). |
| TEST-HELPER | No `*_testhelpers.go` test-only constructor in a production package | PASS | `FailWritesOn` lives in the established `databasetest` support package (sibling to `testdb.go`, which already imports `testing`/`testify`). `grep` confirms only `_test.go` files import `databasetest` — `testing`/`testify` never enter a production build. |
| IMPORTS | Import conventions | PASS | `failwrites.go:3-8`, `transaction_test.go:3-11` — stdlib then third-party grouped; aliased `database "github.com/Chronicle20/atlas/libs/atlas-database"` per repo convention. |
| DOM-20 | Table-driven tests | N/A / informational | atlas-database is a shared-lib, not a domain package. The scenario tests are appropriately discrete (each asserts a distinct tx invariant); forcing a `[]struct` table would reduce clarity. Not a violation. |

## Adversarial Correctness Review — transaction fix

- The `committer != nil`-without-`reflect.IsNil` shape matches GORM's own begin-guard (`finisher_api.go:640`). GORM only adds the extra `reflect.ValueOf(committer).IsNil()` on the commit/rollback paths (`:717`), not on the begin decision. The fix reproduces the correct analog. No defect.
- `*sql.Tx` implements `TxCommitter` (Commit/Rollback); `*sql.DB` and `*sql.Conn` do not — so root pools and `db.WithContext(...)` session handles correctly resolve to "not in a tx" and get wrapped. Verified against GORM source.
- Nested behavior is a deliberate no-savepoint join (inner failure propagates to outer), documented and pinned by `TestExecuteTransaction_NestedJoinsOuterTransaction`. This differs from `db.Transaction`'s savepoint nesting, but that is the intended atlas contract and the test asserts it.

## Adversarial Correctness Review — FailWritesOn helper

- Table isolation and verb isolation are asserted (`failwrites_test.go:34-42`); the `Statement.Table == table` predicate (`failwrites.go:31`) is exercised for create/update/delete.
- The raw-`.Exec` bypass is documented honestly (`failwrites.go:22-23`).
- Callbacks are never unregistered, which is safe only because `NewInMemoryTenantDB` builds a fresh `*gorm.DB` per test (`testdb.go:24`). No cross-test leakage. Non-blocking observation, not a defect.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (observations)
- **Blast radius (out of this diff's scope):** the fix makes all 175 `ExecuteTransaction` call sites genuinely transactional (previously a no-op). Correctness of downstream callers under real atomicity is the subject of the Task-3 write-path survey (`audit.md`), not of this library diff. No breaking caller was found, and no evidence of one exists in the changed files.
- **DOM-20 table-driven** is N/A for a shared-lib; discrete scenario tests are appropriate here.
- **FailWritesOn** offers no de-registration; relies on fresh-per-test DBs. Documented by construction; acceptable.
