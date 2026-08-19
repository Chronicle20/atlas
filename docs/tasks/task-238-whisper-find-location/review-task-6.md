# Review: Task 6 — atlas-channel state-bearing location client

Commit range: `296ae8b40..e09eee916` (single commit `e09eee916`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

Diff touches exactly two files, matching the brief's file list precisely:

- `services/atlas-channel/atlas.com/channel/maps/location/requests.go` (+49)
- `services/atlas-channel/atlas.com/channel/maps/location/requests_test.go` (+114)

`git diff --stat 296ae8b40..e09eee916 -- .../resolve.go` returns empty — `resolve.go`
was not touched, confirming `ResolveMapId` is unmodified. No files outside the
declared scope are present in this commit.

## Checklist

1. **`Get` returns `ErrNotFound` on HTTP 404, mirroring `GetField`.**
   PASS. `requests.go:117-133` — the `errors.Is(err, requests.ErrNotFound)` branch
   is byte-for-byte the same pattern as `GetField` (`requests.go:77-86`).
   `TestGet_NotFoundIsErrNotFound` (`requests_test.go`) stands up a stub
   returning `http.StatusNotFound` and asserts
   `errors.Is(err, ErrNotFound)`. `TestGet_InfrastructureErrorIsNotErrNotFound`
   asserts a 5xx does *not* satisfy `errors.Is(..., ErrNotFound)` — the negative
   case is also covered. Ran `go test ./maps/location/ -v`: both pass.

2. **Empty/absent/unrecognised `state` all resolve to `PresenceStateOffline`,
   including the absent-key case specifically.**
   PASS. `characterconst.ParsePresenceState` (`libs/atlas-constants/character/presence.go:29-38`)
   defaults anything not matching `IN_FIELD`/`IN_CASH_SHOP` to `PresenceStateOffline`,
   which covers `""` (zero value of `string`) and any garbage value.
   `TestGet_AbsentStateIsOffline` (`requests_test.go`) calls `serveLocation` with
   an `attrs` block that **omits the `"state"` key entirely** (not merely an
   empty string) — confirmed by reading the literal test fixture, which has no
   `"state":` entry at all. Go's JSON decoder leaves `RestModel.State` at its
   zero value `""` in that case, which `ParsePresenceState` narrows to
   `PresenceStateOffline`. This is a genuine absent-key test, not just an
   empty-string test. `TestGet_UnrecognisedStateIsOffline` separately covers an
   unrecognised value (`"IN_ORBIT"`).
   Mutation check: if `Get` assigned `state: characterconst.PresenceState(rm.State)`
   directly instead of routing through `ParsePresenceState`, both
   `TestGet_AbsentStateIsOffline` and `TestGet_UnrecognisedStateIsOffline` would
   fail (state would be `""` / `"IN_ORBIT"`, not `PresenceStateOffline`) — the
   test suite is not vacuous on this path.

3. **JSON attribute key is exactly `"state"` (lowercase), matching atlas-maps.**
   PASS. `requests.go:39` — `State string \`json:"state"\`` on the channel side.
   `services/atlas-maps/atlas.com/maps/character/location/rest.go:21` —
   `State characterconst.PresenceState \`json:"state"\`` on the producer side.
   Both use the identical lowercase key; the wire contract is consistent.

4. **`GetField` and `resolve.go`/`ResolveMapId` unmodified.**
   PASS. Diff confirms `GetField` (`requests.go:77-86`) is untouched — the new
   code is appended below it. `git diff --stat` for `resolve.go` is empty, so
   `ResolveMapId` is byte-identical to the pre-task version.

5. **`NewModelForTest` was NOT added.**
   PASS. `grep -rn "NewModelForTest" .../maps/location/` returns nothing.
   Correctly deferred to Task 8 per the brief.

6. **Tests are not vacuous.**
   PASS — see item 2's mutation analysis. Additionally,
   `TestGet_DecodesInField`'s comment explains the channel-7 fixture choice
   deliberately avoids 0/1 so a hard-coded-channel regression would be caught;
   this is a real regression guard rather than a copy-paste filler test.

## Build/test verification (module-local, read-only)

```
cd services/atlas-channel/atlas.com/channel && go test ./maps/location/ -v
```
All 9 tests pass (3 pre-existing `GetField` tests + 6 new `Get` tests).

```
cd services/atlas-channel/atlas.com/channel && go build ./...
```
Builds cleanly.

## Findings

None blocking. The diff matches the brief's Step 3 code verbatim, the fail-safe
narrowing is genuinely exercised (including the absent-key case), the wire key
matches the producer, and the two "must not touch" files (`GetField`,
`resolve.go`) are confirmed unmodified.

## Not evaluable

- Task 8 (the `/find` handler that will consume this `Get`/`Model`) is out of
  scope for this range and not reviewed here.
