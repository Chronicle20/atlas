# Review: Task 10 — the shared command ledger

Commit range: `795f370fe..fb2c946f8` (`27602462f` lint fix, `fb2c946f8` ledger feature)
Brief: `.superpowers/sdd/plan/task-10-brief.md` (Controller corrections C1–C6 are authoritative and override the original prose)
Report: `.superpowers/sdd/plan/task-10-report.md`

## Scope

`git diff --stat 795f370fe..fb2c946f8`:

```
services/atlas-cashshop/atlas.com/cashshop/ledger/ledger.go            |  68 +++
services/atlas-cashshop/atlas.com/cashshop/ledger/ledger_test.go       | 134 +++
services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill.go       |   2 +-
services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill_test.go  |   2 +-
```

Exactly four files, matching the report's "Files changed" list. Reviewed both commits in full plus the read-only dependency `libs/atlas-database/idempotency.go` that the ledger package's correctness depends on.

## C1–C6 compliance (the authoritative design ruling)

- **No new table / no `entity.go` / no `duplicate.go`.** `ledger.go` defines no `gorm` model of its own; it constructs `database.IdempotencyEntity` directly (`ledger.go:54-59`) and calls `.Create(&claim)` against it. Confirmed by grep: no `TableName()`, no `AutoMigrate`, no `cash_command_ledger` string anywhere in the diff. PASS.
- **No migration.** No `Migration` function defined or referenced in the diff. PASS.
- **No `main.go` change.** `git diff 795f370fe..fb2c946f8 -- .../main.go` is empty. Independently confirmed `main.go:64` already registers `database.IdempotencyMigration` and `main.go:78` already starts the sweeper (pre-existing, untouched). PASS.
- **Correct key derivation.** `ledger.go:56` sets `Key: transactionId.String()`, not `database.Key(...)`. This is what makes `TestClaim_ReplayUnderDifferentCommandTypeIsStillRejected` (`ledger_test.go:42-55`) pass — the brief's C2 explicitly calls out `database.Key()` as the wrong choice because it would hash the operation in and let a replay through under a different command type. Verified by reading `database.Key()` at `idempotency.go:47-61`: it does hash `operation` and `payload` into the digest, so using it here would indeed have broken subtest 3. The implementation avoids that trap. PASS.
- **`uuid.Nil` rejected before any DB write, and not reported as `ErrAlreadyProcessed`.** `ledger.go:45-47`. `TestClaim_TheZeroTransactionIdIsRejectedOutright` asserts `require.Error` + `require.False(errors.Is(err, ErrAlreadyProcessed))` (`ledger_test.go:81-89`). PASS.
- **Tenant taken from context, not parameter.** `ledger.go:49` — `tenant.FromContext(ctx)()`, matching `Once`'s own idiom at `idempotency.go:74`. PASS.
- **`CharacterId` not persisted.** Confirmed — `database.IdempotencyEntity` (`idempotency.go:28-35`) has no such field, and `ledger.go`'s `claim` literal (`:54-59`) does not set one. Kept only in the signature and the `uuid.Nil` error message; documented rationale in `ledger.go:41-43` and the report. This matches C2's explicit "your call, say which and why" — done. PASS.

## Independent verification (not trusting the report)

**1. Ledger rides on the real shared entity, no parallel table.**
`ledger.go:18` imports `database "github.com/Chronicle20/atlas/libs/atlas-database"`; `ledger.go:54` constructs `database.IdempotencyEntity{...}`. `ledger_test.go:120-134` (`TestClaim_LandsInTheSharedIdempotencyTable`) independently queries `var row database.IdempotencyEntity` against the same DB and asserts the row matches. No new struct, table name, or migration is defined anywhere in the diff — grep confirms `TableName()` appears nowhere in `ledger.go`. PASS.

**2. Duplicate-rejection guard is load-bearing (reproduced myself, not trusted from the report).**
Removed the `if res.RowsAffected == 0 { return ErrAlreadyProcessed }` branch from a working copy of `ledger.go` and ran `go test ./ledger/... -run TestClaim_ReplayIsRejected -v`:

```
--- FAIL: TestClaim_ReplayIsRejected (0.00s)
    Error: Expected error with "command already processed for this transaction" in chain but got nil.
FAIL
```

Restored the file (`git diff --stat` empty afterward) and re-ran the same test:

```
--- PASS: TestClaim_ReplayIsRejected (0.00s)
```

This matches the report's RED/GREEN claim exactly. PASS — replay protection is genuinely enforced by the test, not tautological.

**3. Tenant isolation.**
`TestClaim_TheSameTransactionUnderADifferentTenantSucceeds` (`ledger_test.go:69-79`) claims transaction `X` under tenant A, then claims the same `X` under a fresh tenant and asserts `NoError`. Correct because `IdempotencyEntity`'s primary key is the composite `(TenantId, Key)` (`idempotency.go:29-33`), so two tenants sharing the same `Key` value do not collide. Verified against the real primary-key definition, not assumed. PASS.

**4. Step 0 lint fix (`27602462f`) is Task 6's files, pure QF1008 cleanup.**
Diff is exactly:
```
-	if db.Dialector.Name() == "sqlite" {
+	if db.Name() == "sqlite" {
```
in `backfill.go:70`, and the identical substitution in `backfill_test.go:236` inside an assertion that only changed the way it *reads* the dialect name (`db.Dialector.Name()` → `db.Name()`), not what it asserts against (`"sqlite"` unchanged, still `t.Fatalf` on mismatch). `gorm.DB` embeds `*Config`, which embeds `Dialector`, so `db.Name()` promotes through to the same call — behaviourally identical. Ran `tools/lint.sh --check --go services/atlas-cashshop` myself:
```
0 issues.
lint.sh: OK
```
PASS — no behavioural change, no assertion altered, lint genuinely exits 0.

## Additional checks

- `go build ./...` and `go test ./...` from `services/atlas-cashshop/atlas.com/cashshop` — all packages pass, including `ok atlas-cashshop/ledger`.
- `git status --porcelain services/atlas-cashshop` — clean after review's own experiment was reverted.
- Full brief Step-1 test table (6 subtests) plus C4's two mandatory additions (`TestClaim_JoinsCallersTransaction`, `TestClaim_LandsInTheSharedIdempotencyTable`) are all present and all pass.
- `TestClaim_JoinsCallersTransaction` (`ledger_test.go:96-114`) correctly proves `Claim` joins the caller's transaction handle rather than opening its own (rollback removes the claim), guarding against the exact "silently opens its own tx" defect class called out in the brief.
- Scope: no changes to `libs/`, `main.go`, or any service other than atlas-cashshop; `git add` used named paths per the commit history (two atomic commits, not `-A`).

## Not evaluable

None. The full review surface (ledger package, its test file, the two lint-fix files, and the read-only `idempotency.go` contract it depends on) was within reach and was verified directly.

## Verdict

APPROVED. All of C1–C6 are honored, the replay-guard test is genuinely load-bearing (reproduced RED/GREEN independently), tenant isolation is correct against the real composite-key definition, and the Step 0 lint fix is a pure, behaviour-preserving QF1008 cleanup with a clean `lint.sh --check` run.
