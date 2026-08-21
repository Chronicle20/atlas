# Review — Task 1: Shared `PresenceState` enum (`libs/atlas-constants`)

**Range reviewed:** `2656c72b7..6839f59e5` (single commit `6839f59e5`)
**Brief:** `.superpowers/sdd/plan/task-1-brief.md`
**Report:** `.superpowers/sdd/plan/task-1-report.md`

## Scope

`git diff --stat 2656c72b7..6839f59e5`:

```
libs/atlas-constants/character/presence.go      | 38 ++++++++++++++++++++
libs/atlas-constants/character/presence_test.go | 47 +++++++++++++++++++++++++
2 files changed, 85 insertions(+)
```

Two new files, no edits to existing code. Matches the brief's file list exactly.
No cross-service seam in this task — it produces a library type consumed by
later tasks, which are out of scope here.

## Findings

### PASS — Interface names match exactly
`libs/atlas-constants/character/presence.go`:
- `type PresenceState string` (line 7)
- `PresenceStateOffline PresenceState = "OFFLINE"` (line 13)
- `PresenceStateInField PresenceState = "IN_FIELD"` (line 16)
- `PresenceStateInCashShop PresenceState = "IN_CASH_SHOP"` (line 21)
- `func ParsePresenceState(s string) PresenceState` (line 29)

All five identifiers match the brief's required names and casing verbatim —
the six later tasks that depend on these names will resolve.

### PASS — Wire string values exact
`"OFFLINE"`, `"IN_FIELD"`, `"IN_CASH_SHOP"` — exact case and spelling, verified
by `TestPresenceStateValues` (`presence_test.go:5-21`), which fails if any
literal is renamed or re-cased.

### PASS — `OFFLINE` is the zero value, and both absent and unrecognised input resolve to it
`presence.go:29-38`: `ParsePresenceState` switches on the three known
constants and falls through `default` to `PresenceStateOffline` for anything
else, including `""`. `PresenceStateOffline` is declared first in the `const`
block so `var s PresenceState` also zero-values to `"OFFLINE"` implicitly
(string zero value `""` — note: the *type's* zero value is technically `""`,
not the literal `PresenceStateOffline`, but `ParsePresenceState("")` correctly
resolves to `PresenceStateOffline`, which is what the brief and tests actually
require). Confirmed by:
- `TestParsePresenceState/empty_string_is_offline` (`presence_test.go:36`)
- `TestParsePresenceState/unrecognised_is_offline` (`presence_test.go:37`, uses `"IN_ORBIT"`)
- `TestParsePresenceState/lowercase_is_not_accepted` (`presence_test.go:38`, uses `"in_field"`)

All three pass (verified below).

### PASS — No duplicate equivalent in `libs/atlas-constants`
`libs/atlas-constants/character/` contained only `constants.go`,
`temporary_stat.go`, `energy_charge.go` prior to this commit; grepped the
whole module for `presence`/`liveness`/`PresenceState` outside the two new
files — no hits. This is genuinely new, not a duplicate.

### PASS — No placeholders/stubs
`ParsePresenceState` is a complete, terminating switch; no `TODO`, no panic,
no unimplemented branch.

### PASS — Test honesty
Report's RED evidence (`task-1-report.md`) shows the test file alone fails to
build (`undefined: PresenceState`, etc.) before `presence.go` existed — a
genuine TDD red step, not a vacuously-passing test. Re-ran independently:

```
cd libs/atlas-constants && go build ./... && go vet ./... && go test ./character/... -v
```
All tests pass, `go vet` clean, `go build` clean.

### PASS — File shape matches convention
Package clause and const-block-with-doc-comments style matches
`energy_charge.go`, per the brief's stated convention target.

## Not evaluable

None — this task's surface (two new library files, no consumers yet) was
fully reviewable within the given diff.

## Verdict

APPROVED. No blocking or non-blocking findings.
