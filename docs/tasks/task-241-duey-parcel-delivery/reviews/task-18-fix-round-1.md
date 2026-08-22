# Review — Task 18 fix round 1 (re-review)

Commit range: `a476d638f..0124fa9d8` (`0124fa9d8` — "fix(channel): task-18
review fixes — shared parcel.WireId, RECEIVE coverage, collision guard").

Original review: `docs/tasks/task-241-duey-parcel-delivery/reviews/task-18.md`
Fix brief: `.superpowers/sdd/plan/task-18-brief-cont.md`
Implementer report: `.superpowers/sdd/plan/task-18-report.md` ("Fix round 1"
section)

Scope: `git diff --stat a476d638f..0124fa9d8` — exactly three files, all in
`services/atlas-channel/atlas.com/channel`:

```
parcel/processor.go                          | 17 ++++++++
socket/handler/duey_action_receive.go        | 35 +++++++++------
socket/handler/duey_action_receive_test.go   | 50 +++++++++++++++++++---
```

This matches the fix brief's `### Files` list exactly. No scope drift.

## BLOCKING 1 — projection moved packages: FIXED

`parcel/processor.go:105-121` (new) defines:

```go
func WireId(id uuid.UUID) uint32 {
	return binary.BigEndian.Uint32(id[:4])
}
```

- **(a) Genuinely in `parcel`, exported.** Confirmed — `package parcel`,
  capitalized `WireId`.
- **(b) No import cycle.** `grep` across `services/atlas-channel/atlas.com/channel/parcel/*.go`
  for any import of `atlas-channel/socket/handler` returns nothing; `parcel`
  does not import `handler`. `handler` continues to import `dueyparcel
  "atlas-channel/parcel"` (`duey_action_receive.go:5`, pre-existing edge).
  Task 21's `parcel/model.go` can therefore call `parcel.WireId` with no
  cycle — the whole point of the move.
- **(c) Both former call sites now go through it, no leftover truncation.**
  `resolveParcelByWireId` (`duey_action_receive.go:109`):
  `dueyparcel.WireId(m.Id()) == wireId`. `discardParcel`'s
  `ParcelRemovedBody` call (`duey_action_receive.go:257`):
  `dueyparcel.WireId(p.Id())`. `wireIdOf` and its now-unused
  `encoding/binary` import are gone from `handler` (confirmed by diff —
  both removed). No open-coded 4-byte truncation remains anywhere in the
  diff.
- **(d) Doc comment traveled and was extended.** `parcel/processor.go:107-119`
  carries the full projection-scheme explanation plus an explicit new line:
  "Every caller that projects a parcel id onto the wire — DUEY_ACTION
  RECEIVE/DISCARD's resolution and Model.ToPacket()'s emission of the OPEN
  list body alike — MUST use this function rather than re-deriving the
  projection, or the two directions will silently disagree." This is
  strictly more explicit about Task 21's obligation than the original
  comment was, not thinned.

**Verdict: fixed, no new defect.**

## BLOCKING 2 — three new RECEIVE subtests: FIXED

`TestDueyActionReceive` now has 8 subtests (confirmed by `grep -c 'name: "'`
= 8): the 5 original plus `not addressed to me` (`duey_action_receive_test.go:215`),
`already received` (`:226`), `getParcel error` (`:234`).

- All three set `getParcelErr` (`errParcelNotResolved` for the first two,
  a distinct `errors.New("atlas-parcel unavailable")` for the third) and
  `want: expect{reason: parcelcb.ParcelOperationIncorrectRequest}`
  (`sagaLen` defaults to the zero value, i.e. 0).
- The shared harness (`duey_action_receive_test.go:253-260`) now makes
  `getParcel` return `(dueyparcel.Model{}, tc.getParcelErr)` when
  `tc.getParcelErr != nil`, instead of unconditionally `(p, nil)` as
  before the fix. This genuinely routes through `receiveParcel`'s
  `err != nil` branch (`duey_action_receive.go:145-150`) rather than
  through the happy path.
- The shared assertion block (`:282-339`) checks, for every subtest
  including the three new ones: `len(sagas) == tc.want.sagaLen` (0 for
  these three), exactly one `announced` entry with
  `writer == parcelcb.ParcelWriter` and `reason == tc.want.reason`, and
  `*closes == 0`. "Stays pending" is structurally guaranteed — this test
  never PATCHes atlas-parcel, matching the report's stated rationale and
  the original review's acceptance of that same structural argument for
  the five pre-existing RECEIVE subtests.
- Ran the test directly: `go test ./socket/handler/... -run TestDueyAction -v`
  — `TestDueyActionReceive` passes with all 8 subtests including
  `getParcel_error`, `already_received`, `not_addressed_to_me` (verified
  by this review, not just taken from the report).

**The specific concern from the original review — that three subtests could
all funnel through the same earlier guard and leave the not-found branch
uncovered — is resolved**, because the fix routes the *dependency injection*
(`getParcel`'s return value) through the new branch, not merely the subtest
*name*. Before this fix, `getParcel` in the test harness always returned
`(p, nil)` unconditionally regardless of subtest, so `receiveParcel`'s
`err != nil` branch was unreachable by any subtest; after the fix it is the
only way these three subtests can reach a non-happy-path result. A revert of
just the test-harness `getParcel` closure (keeping the fixture/case-table
additions) would make these three subtests fail, since they'd fall through
to the happy path and assert the wrong `announced`/`sagas` outcome — this
is a genuine behavior-pinning test, not a vacuous one.

## Provenance of the new subtests' expected values

`.superpowers/sdd/plan/task-18-brief-cont.md`'s BLOCKING 2 section supplies
the exact table the implementer was told to use:

| subtest | setup | expect |
|---|---|---|
| `not addressed to me` | parcel addressed to character 999 | `INCORRECT_REQUEST`, no saga |
| `already received` | parcel `status` `received` | `INCORRECT_REQUEST`, no saga |
| `getParcel error` | `getParcel` returns a real error | `INCORRECT_REQUEST`, no saga |

The landed subtests match this table verbatim (`ParcelOperationIncorrectRequest`,
`sagaLen` unset i.e. 0, `getParcelErr` set as directed). Because the brief
itself pins the expected values (not left to the implementer's inference),
there is no room here for "read the value back off the implementation" —
the brief's own table is the independent source of truth, and the landed
code matches it. No provenance concern.

## NON-BLOCKING 1 — collision guard: FIXED, with test coverage note

`resolveParcelByWireId` (`duey_action_receive.go:101-122`) now iterates all
of `GetForRecipient`'s results, tracks `found`/`match`, and on a **second**
match logs a warning and returns `dueyparcel.Model{}, errParcelNotResolved`
immediately (`:110-113`) rather than continuing to silently prefer the
first hit. This is a real, non-nil error return — a caller cannot misread
it as "not found vs. found" ambiguity, since both the no-match and
collision paths return the same sentinel `errParcelNotResolved`, and both
correctly propagate through `receiveParcel`/`discardParcel`'s existing
`err != nil` → `INCORRECT_REQUEST` handling.

**Test coverage: none.** No subtest in this diff constructs a
same-mailbox wire-id collision (two pending parcels whose first 4 UUID
bytes collide) to exercise the `found` branch at `:110-113`. The fix brief
did not ask for one (NON-BLOCKING 1 only asked for the guard, not a test),
so this is not a broken promise, but it means the collision-rejection path
itself is unverified by any test — worth flagging for completeness, not
blocking.

## NON-BLOCKING 2 — idiom: FIXED

`duey_action_receive.go:155`: `it := inventory.Type(p.ItemType())`, dropping
the previous `int8(...)` round-trip. `ItemType()` returns `byte`
(`parcel/processor.go:101`) and `inventory.Type` is defined as `int8`
(`libs/atlas-constants/inventory`); a direct `byte → int8` numeric
conversion is legal Go and now matches the send-side idiom
(`duey_action_send.go:183`: `inventory.Type(sp.InventoryType())`).

## Approved-list — confirmed untouched

`git diff --stat a476d638f..0124fa9d8 -- services/atlas-parcel libs/atlas-saga
services/atlas-saga-orchestrator services/atlas-channel/atlas.com/channel/socket/handler/duey_action.go`
returns empty — none of these paths appear in the fix commit. Specifically
confirmed not touched:

- atlas-parcel's `PATCH /parcels/{parcelId}` route (no atlas-parcel files in
  the diff at all).
- §7.2 pre-flight ordering — `receiveParcel`'s check sequence (resolve →
  free-slot → unique-item → `ReceivableAt` → build+create saga) is
  byte-for-byte unchanged in the diff except for the cast at line 155 (NON-BLOCKING 2).
- DISCARD-not-a-saga — `discardParcel`'s body is structurally unchanged
  apart from the `WireId` call-site update.
- Routing in `duey_action.go` — file not present in the diff.
- `libs/atlas-saga/` and `services/atlas-saga-orchestrator/` — not present
  in the diff (and also outside this range's ownership per the task
  instructions, since Task 19 landed those separately in `a476d638f`).

## Verification performed by this review

```
cd services/atlas-channel/atlas.com/channel
go build ./...                                          # clean
go test ./socket/handler/... -run TestDueyAction -v      # all PASS, 8 RECEIVE subtests
```

(`tools/verify.sh` intentionally not run per the task's instructions — a
repo-wide gate is running concurrently.)

## Not evaluable

- Whether `tools/lint.sh --go services/atlas-channel/atlas.com/channel` is
  currently clean — not re-run by this review (a repo-wide gate is running
  concurrently); the implementer's report states it was run after the last
  edit of this round and returned "0 issues," closing the previous round's
  open concern, but this review did not independently re-verify.
- The collision-guard's rejection path (`duey_action_receive.go:110-113`)
  is unverified by any test in this diff — noted under NON-BLOCKING 1
  above, not independently exercised by this review either (would require
  constructing a UUID collision, outside a re-review's scope).

## Summary

All four original findings are genuinely fixed, with evidence:

1. **BLOCKING 1** (package placement) — fixed. `parcel.WireId` is exported,
   reachable from `parcel` with no import cycle, both call sites updated,
   doc comment carried forward and extended for Task 21.
2. **BLOCKING 2** (missing RECEIVE coverage) — fixed. Three new subtests
   genuinely route through `receiveParcel`'s previously-uncovered
   not-found branch via a real `getParcel` error injection, not merely a
   new subtest name funneling through the same guard. Expected values
   traced to the fix brief's own table, not read back from the
   implementation.
3. **NON-BLOCKING 1** (collision guard) — fixed as a real reject, not a
   silent nil; no dedicated test exercises it (not requested by the brief;
   noted as an unfilled gap, not a defect).
4. **NON-BLOCKING 2** (idiom) — fixed, now matches the send side exactly.

No new defects introduced. Diff scope matches the fix brief's `### Files`
list exactly, and the approved-do-not-touch list is confirmed untouched.
