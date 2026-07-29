# Backend Audit — atlas-data baseline-restore atomicity fix

Scope: `git diff main...HEAD -- services/atlas-data` (2 files: `baseline/restore.go`,
`baseline/restore_failure_test.go`). Adversarial review, FAIL-until-proven mindset.
File:line references are against the worktree at review time.

## Verdict: PASS

No blocking (FAIL) findings. One MEDIUM (non-blocking) finding on test-coverage depth.
Build/vet/test all clean (see §5).

---

## 1. Correctness of the new pgx transaction handling

`replaceTableBinary`, `services/atlas-data/atlas.com/data/baseline/restore.go:342-388`.

**Is `c.Begin(ctx)` + `c.PgConn().CopyFrom(...)` on the same `*pgx.Conn` actually one
transaction?** — Verified YES, by reading `pgx/v5` source
(`~/go/pkg/mod/github.com/jackc/pgx/v5@v5.10.0/tx.go:96-110`): `Conn.Begin` is
implemented as `dbTx{conn: c}` and issues `BEGIN` via `c.Exec`. A Postgres
transaction is *session state on the wire connection*, not an object the pgx.Tx
wrapper owns — `dbTx.Commit`/`Rollback` just issue `commit`/`rollback` on the same
`*pgx.Conn`. Since `c.PgConn()` (`restore.go:375`) returns the identical underlying
`pgconn.PgConn` for that same `*pgx.Conn`, the `COPY` participates in the
already-open transaction. This is the direct fix for the diagnosed root cause,
confirmed independently: `gorm.DB.DB()` (`~/go/pkg/mod/gorm.io/gorm@v1.31.2/gorm.go:426-445`)
special-cases a `*sql.Tx` connPool and reflects into the **private `db` field**
(the pool `*sql.DB`) rather than returning the tx's own connection — i.e.
`tx.DB().Conn(ctx)` provably never returns the transaction's connection. The old
`copyInBinary` calling `tx.DB()` was therefore guaranteed to check out a second,
unrelated connection. Root cause and fix both verified from source, not asserted.

**Error paths** (`restore.go:358-386`):
| Failure point | Handling | Verdict |
|---|---|---|
| `c.Begin(ctx)` fails (:359) | returns immediately, no goroutine started yet | clean, no leak |
| `tx.Exec` DELETE fails (:362-365) | `tx.Rollback(ctx)` then return; no goroutine started yet | clean, no leak |
| `CopyFrom` fails (:375-381) | `pr.CloseWithError(err)` unblocks the writer goroutine, `<-errc` drains it, then `tx.Rollback(ctx)` | goroutine drained, no leak |
| `rw.Stream` (writer goroutine) fails but `CopyFrom` still returns nil (plain `pw.Close()`, not `CloseWithError`, so the reader sees ordinary EOF) | caught by the **separate** `if err := <-errc; err != nil` check at :382-385, which still rolls back | correct safety net — pre-existing pattern, unchanged by this diff (see note below) |
| `tx.Commit(ctx)` fails (:386) | no explicit `Rollback` call | correct: `dbTx.Commit` (`tx.go:178-197`) itself closes the underlying connection when `TxStatus() != 'I'` after a failed commit, so a manual rollback would be redundant/erroring on an already-closed tx |

Note on the `rw.Stream` error path: the writer goroutine closes `pw` with a plain
`Close()` rather than `CloseWithError(err)` on a `Stream` failure
(`restore.go:368-371`, the `defer pw.Close()` / `errc <- rw.Stream(...)` pair), so a
mid-stream failure surfaces to the reader as ordinary EOF, not as a read error. This
is **not a regression** — it is byte-for-byte the pre-existing pattern from the old
`copyInBinary` (see the diff: the goroutine-launch and pipe-close code is
unchanged); it works because binary-COPY format has a defined trailer and any
truncation before it either makes Postgres reject the COPY (surfacing via
`CopyFrom`'s own error) or is caught by the always-checked `<-errc` after a
"successful" `CopyFrom`. Flagging as informational, not a new defect.

**Is the connection left clean for the pool after every exit path?** — Verified via
`pgx/v5/stdlib` source (`~/go/pkg/mod/github.com/jackc/pgx/v5@v5.10.0/stdlib/sql.go:553-582`):
`Conn.ResetSession` — the `driver.SessionResetter` hook `database/sql` calls before
handing a pooled connection to the next caller — checks
`c.conn.PgConn().TxStatus() != 'I'` and `c.conn.IsClosed()`, returning
`driver.ErrBadConn` (which causes eviction, not reuse) if either trips. This matters
because `sql.Conn.Raw`'s own contract is *not* protective here: per the stdlib doc,
a non-`driver.ErrBadConn` error returned from the `Raw` callback leaves the
connection "usable" from `database/sql`'s point of view — it does **not**
auto-discard on a generic error. So the safety net against handing back a
mid-transaction or dead connection is entirely pgx's `ResetSession`, not this code.
Confirmed that safety net is actually wired in production: `atlas-database`'s
`retryConnector.Connect` (`libs/atlas-database/connector.go:38-61`) returns the
`*stdlib.Conn` from `stdlib.GetConnector` **unwrapped**, so `ResetSession` and the
`*stdlib.Conn` type assertion at `restore.go:353-356` both see the real pgx type.
`defer conn.Close()` (`restore.go:351`) always runs exactly once regardless of which
branch returns, releasing the `*sql.Conn` back through this machinery. PASS.

**Connection-count note (improvement, not a new risk):** the old code held *two*
pooled connections concurrently per table (the gorm tx's connection for DELETE, plus
a second one from `tx.DB().Conn(ctx)` for COPY). The new code holds exactly one.
Combined with the per-tenant advisory-lock connection held for the whole `Restore()`
call (`restore.go:266-292`), worst case is now 2 connections in use per in-flight
restore instead of 3 — a strict improvement against `DB_MAX_OPEN_CONNS` exhaustion,
not a regression.

## 2. Invariants relied on by Restore()/runRestoreTables/cleanupAfterFailure

Read `restore.go:60-239` in full. Findings:

- Per-table atomicity granularity is **unchanged**: the old `db.Transaction(...)`
  wrap was already scoped to one `restoreOneTable` call (i.e. per table, not per
  whole restore); `replaceTableBinary` preserves exactly that scope — one
  BEGIN/COMMIT per table, called once per `tar.Reader` entry from
  `runRestoreTables` (:64-85). No cross-table atomicity was ever claimed or
  removed.
- The deferred `StatusComplete` UPSERT (`restore.go:229-236`) still runs only after
  the full `runRestoreTables` loop **and** every `ANALYZE` succeed
  (`restore.go:215-227`) — untouched by this diff, and independently pinned by the
  pre-existing `TestRestoreDeferredMarkerStructure` test.
- `cleanupAfterFailure` (`restore.go:108-116`) DELETEs `DumpTables` rows in its own
  best-effort loop under a fresh `context.Background()`-derived timeout, entirely
  independent of `replaceTableBinary`'s transaction — unaffected by this diff.
- `markRestoring`/advisory-lock sequencing (`restore.go:139-147`, `203-205`)
  unaffected.

No invariant the surrounding flow depends on is broken by removing the
`db.Transaction` wrapper.

## 3. Resource / goroutine leaks vs. the original `copyInBinary`

Covered in §1's error-path table. Every exit path that starts the writer goroutine
also drains `errc` exactly once before returning (`restore.go:377-378` and
`:382-385`); paths that return before the goroutine is started (`Begin` failure,
DELETE failure) never leak it. `routine.Go` (`libs/atlas-routine/routine.go:15-26`)
recovers panics so a `rw.Stream` panic cannot crash the process — unchanged,
pre-existing. `conn.Close()` is a single unconditional `defer`
(`restore.go:351`) — reached on every return path including a panic unwind from
deeper in the call, though nothing in this function is expected to panic.
Context cancellation (the 30-minute `restoreOpTimeout`, `restore.go:49`) is honored
by every pgx call in the new code (`Begin(ctx)`, `Exec(ctx,...)`, `CopyFrom(ctx,...)`,
`Commit(ctx)`) — a mid-restore timeout surfaces as an error on whichever call is in
flight and is handled by the same rollback/return paths above. No new leak found.

## 4. Is the source-structure test (`TestRestoreTableSwapIsAtomic`) a reasonable guard?

`restore_failure_test.go:104-126`. It follows the exact same idiom as the file's
four pre-existing tests (`TestRestoreDeferredMarkerStructure`,
`TestRestoreIntentPrecedesTables`, `TestRestoreAcquiresTenantLockFirst`,
`TestRestoreContextDetached`) — all lexical/string-position guards over
`restore.go`'s source text, chosen because the pgx binary-COPY path only executes
against real Postgres and this service has **no** real-Postgres integration-test
harness (`grep -rl "+build integration\|testcontainers"` under
`services/atlas-data/atlas.com/data` returns nothing but a `go.sum` hit). So it is
consistent with established practice in this file, not an ad hoc weak pattern.

**MEDIUM, non-blocking finding:** it is a regression *pin*, not a correctness
*proof*. It asserts the literal tokens `c.Begin(ctx)`, `DELETE FROM `, `CopyFrom(`,
`tx.Commit(ctx)` appear inside `replaceTableBinary` and that `db.Transaction(` does
not — it does not check ordering beyond that, does not verify the same connection
variable (`c`) is threaded through all four operations (currently true, but a future
edit could introduce a second `pgxConn.Conn()`/`db.Begin` call using a different
variable name and this test would not catch it), and cannot detect the actual
runtime defect class this task exists to fix. That defect class is exactly the
danger here: the original bug (`tx.DB().Conn(ctx)` silently returning a pool
connection instead of the transaction's own) *read* as correct on a source-text
skim — gorm's method name (`DB()`) gives no lexical signal that it bypasses the
transaction. A lexical guard over the fix is therefore necessarily weaker evidence
than the lexical guard over the original bug would have been (i.e., none). This is
not a reason to block — there's no existing real-Postgres test infra to build on in
this service, and adding one is a bigger undertaking than this bug fix — but it
means the atomicity property itself (a concurrent reader never observes the gap) is
currently asserted by design reasoning (§1, cross-checked against pgx/gorm source)
and not exercised by any automated test. Recommend surfacing this as a tracked
follow-up if/when Postgres-backed integration tests are added to atlas-data,
rather than treating this PR as having empirically closed the loop.

## 5. Build / vet / test

Run from `services/atlas-data/atlas.com/data`:

```
$ go build ./...
(no output — clean)

$ go vet ./...
(no output — clean)

$ go test -race -count=1 ./baseline/...
ok  	atlas-data/baseline	1.036s
```

All three clean as required.

---

## Summary

| # | Area | Severity | Result |
|---|---|---|---|
| 1 | pgx tx/COPY mechanics, all error paths, connection hygiene | — | PASS (verified against pgx/v5 and gorm source, not assumed) |
| 2 | Surrounding Restore()/runRestoreTables/cleanupAfterFailure invariants | — | PASS, unaffected |
| 3 | Goroutine/resource leaks vs. original | — | PASS, no new leak; connection-count strictly improved |
| 4 | Source-structure test adequacy | MEDIUM (non-blocking) | Reasonable regression pin, consistent with file's existing convention; does not and cannot prove the atomicity property at runtime — no real-Postgres test infra exists in this service to do better right now |
| 5 | build/vet/test | — | PASS, all clean |

No FAIL findings. The single MEDIUM finding is a coverage-depth observation, not a
defect in the shipped code.
