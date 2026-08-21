# Review: Task 4 — atlas-maps character-status presence transitions

## Scope

Reviewed commit `9254bd6b5` (range `a56c5b15d..9254bd6b5`), the single commit in
`.superpowers/sdd/plan/review-a56c5b15d..9254bd6b5.diff`. Two files touched:

- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`
- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go`

This matches the brief's declared file list exactly. `git status --porcelain`
on the touched directory is clean — the diff package reflects the actual
committed state.

To evaluate correctness of the two `SetState`/`SetStateIfOnline` calls this
unit depends on, I also read `character/location/processor.go` and
`character/location/administrator.go` (Task 2's work, not part of this diff,
but the contract this unit's calls rely on).

## Requirement-by-requirement

1. **LOGIN → `IN_FIELD`.** `consumer.go` (diff hunk in `handleStatusEventLoginFunc`):
   `lp.SetState(event.CharacterId, characterconst.PresenceStateInField)` is
   called after `lp.Set(event.CharacterId, f)`. Matches. PASS.

2. **LOGOUT → `OFFLINE`.** `handleStatusEventLogoutFunc`:
   `lp.SetState(event.CharacterId, characterconst.PresenceStateOffline)` is
   called after `lp.Set(event.CharacterId, resolved)` and before
   `p.ExitAndEmit(...)`. Matches. PASS.

3. **CHANNEL_CHANGED → `IN_FIELD`.** `handleStatusEventChannelChangedFunc`:
   `lp.SetState(event.CharacterId, characterconst.PresenceStateInField)` is
   called after `lp.Set(event.CharacterId, newField)` and before the
   map-time-limit timer hooks. Matches. PASS.

4. **Unconditional `SetState`, not `SetStateIfOnline`.** All three call sites
   use `lp.SetState(...)`. Confirmed against `processor.go:94-97` —
   `SetState` calls `setLocationState(...)(state, false)` (unconditional,
   `conditional=false`), distinct from `SetStateIfOnline` at
   `processor.go:102-105` (`conditional=true`). This is the deliberate
   asymmetry called out by the reviewer brief; not flagged. PASS.

5. **`CREATED` and `CHANGE_MAP` untouched.** The diff's only modified handler
   bodies are `handleStatusEventLoginFunc`, `handleStatusEventLogoutFunc`, and
   `handleStatusEventChannelChangedFunc`. `handleStatusEventCreatedFunc`,
   `handleStatusEventDeletedFunc`, and `handleChangeMapFunc` do not appear in
   the diff hunks at all. PASS.

6. **No TTL / sweeper / invented timeout.** No new constant, ticker, or
   sweeper goroutine appears anywhere in the diff. PASS.

7. **Wire values.** `libs/atlas-constants/character/presence.go:14,17,23`
   confirm `PresenceStateOffline = "OFFLINE"`, `PresenceStateInField =
   "IN_FIELD"`, `PresenceStateInCashShop = "IN_CASH_SHOP"` — matches the
   values this task's transitions reference (`InField`, `Offline`). PASS.

## Error handling

Each new `SetState` call is wrapped exactly like the pre-existing `Set` calls
in the same functions:

```go
if err := lp.SetState(event.CharacterId, characterconst.PresenceStateInField); err != nil {
    l.WithError(err).Warnf("location.SetState on LOGIN failed for character [%d].", event.CharacterId)
}
```

- Logged via `l.WithError(err).Warnf(...)` — not swallowed silently.
- No `return` after the `Warnf` — execution falls through to the rest of the
  handler in all three cases (LOGIN: nothing follows; LOGOUT: falls through to
  `p.ExitAndEmit(...)`; CHANNEL_CHANGED: falls through to the timer hooks
  `tp.ForceReturnIfTracked(...)`). A `SetState` failure does not abort
  unrelated event handling. PASS — matches the existing convention in this
  consumer (the `Set` calls immediately above use the identical
  log-and-continue pattern).

## Location write vs. state write atomicity

`Set` (→ `upsertLocation`) and `SetState` (→ `setLocationState`) are two
separate `*gorm.DB` calls, not wrapped in a single transaction — this is
Task 2's existing design, unchanged by this diff. Each call is individually
atomic (a single `Create`+`OnConflict`/`First` pair for `upsertLocation`, a
single `Update` for `setLocationState` — `character/location/administrator.go:20-65`).
The concern in scope for this task is whether a `SetState` failure can leave
the position half-written: it cannot — the position row was already committed
by the preceding `Set` call before `SetState` runs, and a `SetState` failure
only leaves the state column at its prior value (which for LOGIN/CHANNEL_CHANGED
starting from a fresh or offline row is `OFFLINE`, a safe fail-closed default,
and for LOGOUT is whatever it already was — the character is logged out
Kafka-side regardless). No new half-applied-write risk is introduced by this
task's edits. Not flagged.

## Tests

The three new tests (`TestLoginHandler_SetsInField`,
`TestLogoutHandler_SetsOfflineAndPreservesPosition`,
`TestChannelChangedHandler_SetsInFieldOnNewChannel`) match the brief's Step 1
code verbatim (diff lines 152-260 against brief lines 22-131 — identical byte
for byte). The implementer's report shows RED (`state = "OFFLINE", want
IN_FIELD` / `state = "IN_FIELD", want OFFLINE`) before the fix and GREEN
after, which is genuine evidence the tests fail without the change — not a
test that passes either way. Per task instructions I did not re-run the
suite; the report's RED/GREEN transcript is treated as evidence per the
review's scoping instruction.

Test setup reuses `newTestDB`/`newTestCtx` from the existing file and
`field.NewBuilder(...)` / `location.NewProcessor(...)` — the project's
existing Builder/Processor pattern. No new `*_testhelpers.go` file was added
(`git diff --stat` for the range shows only the two files listed above).

## Scope check

`git diff --stat a56c5b15d..9254bd6b5` shows exactly the two files the brief
named, 125 insertions / 2 deletions, one commit. The diff content matches the
brief's Step 3 code blocks character-for-character. No scope drift.

## Not evaluable

None — the full change is within a single, small, directly-readable diff, and
its one out-of-diff dependency (`location.Processor.SetState`/
`SetStateIfOnline`, from Task 2) was read to confirm the contract this task's
calls depend on.

## Verdict rationale

All five brief requirements verified against the diff with line-level
evidence. Error handling matches the file's existing convention. No scope
creep into `CREATED`/`CHANGE_MAP`. No invented TTL. No test-helper file. Tests
are genuine (RED before, GREEN after, per the implementer's transcript).
No blocking findings.
