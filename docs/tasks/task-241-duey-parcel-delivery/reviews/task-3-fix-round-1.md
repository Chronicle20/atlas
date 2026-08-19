# Task 3 — fix round 1 re-review

Scope: commit `7449ce973` only (`fix(parcel): world-scope HasInFlight, map
GetById 404, CAS receive/discard`). The range given, `91dc1cae9..7449ce973`,
also contains `cae523317` (`feat(parcel): JSON:API REST surface for parcels
and parcel-status`, Task 4's own work, reviewed separately) — confirmed via
`git show --stat 7449ce973`, which touches only `administrator.go`,
`administrator_test.go`, `processor.go`, `processor_test.go`, `provider.go`.
`resource.go`/`rest.go` are Task 4 territory and out of scope here.

## Finding 1 — BLOCKING, `HasInFlight` hardcoded `world.Id(0)` — ADDRESSED

`processor.go:97` (now) calls the new `ReceivableByRecipientAnyWorld
(characterId, now)` (`provider.go`, new function) instead of
`ReceivableByRecipient(characterId, world.Id(0), now)`. The provider drops
the `world_id` predicate entirely, so an inbound parcel in any world is now
found.

New test: `TestProcessorHasInFlight/inbound_receivable_in_a_non-zero_world`
(`processor_test.go`), seeds `WorldId(7)`, asserts `HasInFlight(100) ==
true`.

**Verified independently** (not just accepted the implementer's claim): checked
out `91dc1cae9` into a scratch worktree, layered the new `processor_test.go`
on top of the pre-fix `processor.go`/`provider.go`, and ran it:

```
--- FAIL: TestProcessorHasInFlight (0.02s)
    --- FAIL: TestProcessorHasInFlight/inbound_receivable_in_a_non-zero_world (0.00s)
        Error: Should be true
```

Confirms the test is a genuine regression test, not one that would pass
either way. On the fix commit it passes (`go test ./parcel/... -run
TestProcessorHasInFlight -v`, all 6 subtests PASS).

## Finding 2 — BLOCKING, `GetById` never mapped to `ErrNotFound` — ADDRESSED

`processor.go:65-73` now: `GetById` calls `ById(...)()`, and if `errors.Is(err,
gorm.ErrRecordNotFound)`, returns `parcel.ErrNotFound` (`errors.go:12`)
instead of the raw gorm sentinel.

New test: `TestProcessorGetById/missing` (`processor_test.go`), asserts
`errors.Is(err, ErrNotFound)`.

**Verified independently**, same scratch-worktree technique: against the
pre-fix `processor.go`, the new `missing` subtest fails —

```
--- FAIL: TestProcessorGetById/missing (0.00s)
    Error: Target error should be in err chain:
           expected: "parcel not found"
           in chain: "record not found"
```

— and passes on the fix commit.

**Already-routed, not new breakage**: this same fix flips
`resource.go:148`'s `errors.Is(err, gorm.ErrRecordNotFound)` check
(Task 4, `cae523317`) from matching to not matching, breaking
`TestParcelResource/get_by_id_missing` (`resource_test.go:176`) — 404
becomes 500. The report documents this exact regression and explicitly
declines to touch Task 4's files. The working tree currently has an
uncommitted, in-progress fix to `resource.go` (swaps the check to
`errors.Is(err, ErrNotFound)`) confirming a Task 4 fix round is genuinely in
flight, as stated. Not filed as a new Task 3 defect, per instruction.

## Finding 3 — Important (ruled into scope), `resolve` read-then-write — ADDRESSED

`administrator.go:41-50` (new): `UpdateStatusIfPending` issues `db.Model(&Entity{}).
Where("id = ? AND status = ?", id, StatusPending).Updates(...)` and returns
`res.RowsAffected` — the predicate is genuinely in the SQL `WHERE` clause,
not a Go-side re-read. `processor.go:141-174` (`resolve`): after the gate
passes, calls `UpdateStatusIfPending` and treats `rows == 0` as
`ErrNotPending`, i.e. the DB-affected-row count decides the outcome, not a
second `SELECT`. `UpdateStatus` (unpredicated) is left untouched, correctly
reserved for task-23's expiry sweep per its own doc comment and the
report's rationale.

New test: `TestUpdateStatusIfPending` (`administrator_test.go`) — two
subtests: (a) a pending row is updated, 1 row affected; (b) an
already-`StatusReceived` row is left untouched, 0 rows affected, `Status`/
`ResolvedAt` unchanged. This exercises the predicate directly (not via a
goroutine race) — reasonable given the report's documented constraint that
`databasetest.NewInMemoryTenantDB`'s sqlite setup caps `MaxOpenConns` to 1,
which would serialize any two transactions and cannot exhibit an
interleaved race. The test would fail under the old unpredicated
`UpdateStatus` (an unconditional `Updates` would have succeeded and
clobbered the already-resolved row) — confirmed by inspection, not
re-run, since `UpdateStatus`/`UpdateStatusIfPending` are two different
functions and there is no "old" version of `UpdateStatusIfPending` to check
out; the CAS semantics were verified by reading the SQL predicate and
return value directly, per the task's ask ("check that the CAS is real").

Existing `TestProcessorReceive`/`TestProcessorDiscard` subtests
(happy path, wrong recipient, not-yet-receivable, already-received/
-discarded, missing) still pass on the fix commit — the CAS path did not
regress the existing gate-based error precision (`ErrNotPending`,
`ErrNotYetReceivable`, `ErrNotRecipient`, `ErrNotFound` are all still
returned as before; the CAS only adds a second line of defense at the
storage layer).

## New breakage check (fix diff only)

- `go vet ./...` and `go build ./...` clean from
  `services/atlas-parcel/atlas.com/parcel`.
- `go test ./parcel/...` — all relevant subtests pass on the fix commit.
- No other callers of `ReceivableByRecipient` (still used, unmodified,
  by whatever calls it with an explicit world elsewhere — not touched by
  this diff) or `UpdateStatus` were broken; `UpdateStatus` remains exported
  and unused in production code today, consistent with it being reserved
  for task-23 (not yet implemented) — not a defect.
- No breakage beyond the already-known/routed `resource_test.go:176`
  failure, which is Task 4's, not new, and not differently-shaped than
  described in the report.

## Verdict

All three findings genuinely fixed, each with a test independently
confirmed to fail against the pre-fix code (findings 1 and 2 by actual
re-run; finding 3 by direct inspection of the CAS SQL and return-value
wiring, per the task's own ask). No new breakage introduced by
`7449ce973` beyond the already-routed, already-predicted Task 4 fallout.
