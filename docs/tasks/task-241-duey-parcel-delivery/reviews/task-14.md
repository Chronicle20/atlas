# Review — Task 14: Orchestrator handler dispatch, event acceptance, compensation

Commit range: `1ccfb5e20..964196ee0` (single commit `964196ee0`)
Brief: `.superpowers/sdd/plan/task-14-brief.md`
Report: `.superpowers/sdd/plan/task-14-report.md`

## Scope

`git diff --stat 1ccfb5e20..964196ee0`:

```
saga/compensator.go               | 328 ++++
saga/handler.go                   | 108 +-
saga/model.go                     |   4 +
saga/parcel_compensation_test.go  | 428 ++++ (new)
```

Matches the brief's file list, plus one addition outside it (`model.go`
aliases — evaluated in Finding 2 below). `saga/event_acceptance.go` has zero
hunks in this diff, as the report claims. Scope confirmed.

## 1. `event_acceptance.go` "no changes needed" claim

Verified by reading the current file and by `git log -- saga/event_acceptance.go`:
all four checks pass.

- `EventKindParcelCustodyAccepted/Released/Error` constants present
  (`saga/event_acceptance.go:93-95`). Only three kinds exist (no `...Moved`,
  unlike MTS's four) — that's correct: parcel custody has no analogue to an
  MTS "move to holding" ack, and the brief's own "Interfaces produced" list
  only specifies three (Accepted/Released/Error). The brief prose's "four"
  appears to be an error in the reviewer's framing, not a brief requirement —
  the brief's own interface list and the Steps section both only ever
  reference three kinds.
- Action→event-kind entries present: `TransferToParcel: {}`,
  `WithdrawFromParcel: {}` (composite, correctly empty),
  `AcceptToParcel: {EventKindParcelCustodyAccepted, EventKindParcelCustodyError}`,
  `ReleaseFromParcel: {EventKindParcelCustodyReleased, EventKindParcelCustodyError}`
  (`saga/event_acceptance.go:257-260`).
- Outcome entries present: `EventKindParcelCustodyAccepted/Released → OutcomeSuccess`
  (`saga/event_acceptance.go:454-455`).
- `git log --oneline -- saga/event_acceptance.go` shows the file was last
  touched by `47fdbf191` — "feat(orchestrator): parcel custody command
  dispatch and status consumer" — i.e. task 13, one commit before this
  range's base. The claim is accurate: these entries pre-date this diff.

No blocking finding here.

## 2. `ParcelSend`/`ParcelReceive` saga-type aliases in `model.go`

`saga/model.go:61-63` adds:
```go
// Duey parcel-delivery saga types (task-241)
ParcelSend    = sharedsaga.ParcelSend
ParcelReceive = sharedsaga.ParcelReceive
```

Verified:
- `libs/atlas-saga/model.go:74-79` defines the actual `Type` constants
  (`ParcelSend Type = "parcel_send"`, `ParcelReceive Type = "parcel_receive"`)
  — the type itself lives in the shared library, not redefined in the
  service. No duplicate definition.
- The added lines are re-export aliases, not new types — every other line in
  that same const block (`InventoryTransaction`, `MtsOperation`,
  `WorldTransfer`, `ScriptedItemUse`, etc.) follows the identical
  `X = sharedsaga.X` pattern going back to the file's original design
  (`saga/model.go:20-29` comment: "Type, Action, and Status are re-exported
  from the shared saga library"). The addition is placement-correct and
  necessary — `CompensateFailedStep`'s new `s.SagaType() == ParcelSend` /
  `== ParcelReceive` routing arms (`saga/compensator.go:459,466`) would not
  compile without it, and no earlier task (11-13) added these aliases
  (confirmed: they are absent from the pre-range file).

No blocking finding. This is a legitimate small gap-fill, correctly placed,
following the existing convention — not a scope violation and not a
constants-reuse violation.

## 3. world.Id marshalling-boundary tag checks

This diff (`handler.go`, `compensator.go`, `model.go`, the new test file)
introduces **no new struct definitions** that carry `world.Id` across a
marshalling boundary. Every `world.Id` reference in this diff is a read of an
existing field (`payload.WorldId` passed straight through in
`handleAcceptToParcel`, `AwardMesosPayload.WorldId` read in the reverse-walk
switch to build a `channel.Model`) — no new JSON-tagged struct is added or
touched. `AcceptToParcelPayload` (the struct that does carry the tag) was
added in task 11 (`libs/atlas-saga/payloads.go:1016`); `parcel.AcceptToParcelParams`
/ `parcelCustody.AcceptToParcelCommandBody` (task 13, `parcel/processor.go`,
`parcel/producer.go`) are the structs where a genuine tag-for-tag check would
apply, and neither is touched by this commit. Not evaluable in this task's
scope — correctly out of scope for a task-14 review, not a defect of this
unit. (Flagging for the controller: if this is one of the branch's four
found instances, it was introduced in task 11 or 13, not here.)

## 4. Compensation correctness vs. the MTS custody arms

Compared side-by-side against `DispatchMtsOperationRollbacks`
(`saga/compensator.go:2401-2470`), `compensateMtsOperation`
(`:2340-2377`), the MTS entries in `lateCompensableActions`
(`:2986-2988`), and the corresponding `dispatchLateInverse` MTS cases
(`:3282-3298`).

- **`compensateParcelSend`/`compensateParcelReceive`** (`:2475-2530`,
  `:2589-2644`) are structurally identical to `compensateMtsOperation`:
  dispatch rollbacks → try-transition Compensating→Failed → cancel timer →
  evict cache → `EmitSagaFailed` with a step-scoped reason string. No drift.
- **`DispatchParcelSendRollbacks`** (`:2539-2588`): locates the sibling
  `AcceptToParcel` snapshot the same way MTS locates its `AcceptToMtsListing`
  snapshot; reverse-walks `Completed` steps; `AwardMesos → AwardMesosAndEmit(-Amount)`
  matches the MTS `AwardCurrency → AwardCurrencyAndEmit(-Amount)` inverse
  direction exactly (send debits with negative `Amount`, so `-Amount`
  re-credits positively — confirmed by the passing test asserting
  `awardCalls[0] == mesoAmount+feePaid`, `saga/parcel_compensation_test.go:270`).
  `ReleaseFromCharacter → RequestAcceptAsset` mirrors the MTS arm field-for-field,
  correctly adds the brief-mandated `HasItem` guard (RISK-2) with a `continue`
  when the snapshot is nil or `HasItem == false` (`:2560-2568`) — the MTS twin
  has no such guard because MTS listings always carry an item.
- **`DispatchParcelReceiveRollbacks`** (`:2657-2699`): `AcceptToCharacter →
  RequestDestroyItem` field-for-field matches the trade-settlement
  `AcceptToCharacter` arm at `:2818-2825` (including the `qty==0 → 1` fallback
  and the `false` no-force-delete flag) — confirmed by reading both blocks.
  `ReleaseFromParcel → RestoreParcelAndEmit` and `AwardMesos → -Amount`
  re-debit are correctly directioned.
- **`assetDataFromParcelSnapshot`** (`:2913-2936`) vs.
  `assetDataFromMtsListingSnapshot` (`:2882-2905`): field-for-field identical
  mapping (both omit `RingId`/`ViciousCount`, matching the MTS twin's
  omission — not a parcel-specific regression, since neither snapshot struct
  is misread relative to its sibling).
- **`lateCompensableActions`** (`:2999-3000`): `AcceptToParcel: {}`,
  `ReleaseFromParcel: {}` added to the *global* map (correct — unlike
  `tradeLateCompensableActions`, which is deliberately scoped per the comment
  at `:3002-3005` because it would leak into unrelated saga types; parcel
  custody actions are only ever steps of `parcel_send`/`parcel_receive`
  sagas, so global registration is safe and matches the MTS custody entries'
  placement).
- **`dispatchLateInverse`** (`:3300-3316`): `AcceptToParcel → RemoveParcelAndEmit`,
  `ReleaseFromParcel → RestoreParcelAndEmit` — correct direction, matches the
  design comments and the reverse-walk's own semantics (a late accept after
  compensation already re-granted the item must remove the duplicate parcel;
  a late release after compensation already undid the grant must restore the
  row to custody).

No copy-paste drift found — the 328 new lines track the MTS/trade patterns
faithfully, including the one place they correctly diverge (the `HasItem`
guard, which MTS has no equivalent of).

## 5. Idempotency comments

`grep -i idempot` against the diff (not the whole file) returns **zero
matches** — no idempotency language was added by this commit at all, bare or
otherwise. The RISK-2 guard is stated as a guard ("must never produce an
award"), not framed as an idempotency claim. Not a defect.

## 6. Test quality

`saga/parcel_compensation_test.go` (428 lines, new):

- Uses `NewBuilder()` (`saga/parcel_compensation_test.go:152` etc.) — the
  repo's Saga builder pattern — not a bespoke `*_testhelpers.go` constructor.
- All five brief-mandated scenarios are present and each assertion targets
  the NEW contract, not just a happy path:
  - `send fails at accept`: asserts `RequestAcceptAsset` called **exactly
    once** with `Strength: 5` and `Owner: "Alice"` preserved from the
    snapshot (design §10's stats-survive case) — a test that would fail
    under a naive "just re-run AwardAsset from template" compensation.
  - `send meso-only fails at accept`: asserts the meso re-credit fires and
    `acceptCalls == 0` — this is the RISK-2 regression test; without the
    `HasItem` guard the reverse walk would either wrongly re-grant a
    phantom item or panic on a nil snapshot dereference.
  - `send late accept`: drives `CompensateLateStep` directly, asserts
    exactly one `RemoveParcelAndEmit` with the matching parcel id.
  - `receive fails at accept_to_character`: asserts exactly one
    `RestoreParcelAndEmit`.
  - `receive fails at award_mesos`: asserts both `RestoreParcelAndEmit` AND
    `RequestDestroyItem` fire with the correct character/template/quantity —
    directly tests the "item not left in recipient's inventory" invariant.
- `parcelTestMock` is a genuine full-interface mock with a compile-time
  `var _ parcelsvc.Processor = (*parcelTestMock)(nil)` assertion
  (`:92`), consistent with the repo's existing mock style
  (`mts_dupe_safety_test.go`).
- Ran `go test ./saga/... -run TestParcel -v`: all 5 subtests PASS. Ran
  `go build ./...` and `go test ./...` for the module: clean, no
  regressions.

No non-honest tests found (none of the five assert something that would also
pass under the pre-fix code path — each targets a specific field, call
count, or absence).

## Verification performed

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go build ./...                                  # clean
go test ./saga/... -run 'TestParcel' -v         # 5/5 PASS
go test ./...                                   # all packages ok
```

## Not evaluable

- Item 3 (world.Id tag-for-tag) — the marshalling-boundary structs that
  would need checking (`AcceptToParcelPayload`, `parcel.AcceptToParcelParams`,
  `parcelCustody.AcceptToParcelCommandBody`) were all added in tasks 11/13,
  not touched by this diff. Out of this unit's review surface.

## Verdict rationale

All six priority checks pass with cited evidence. The one out-of-brief
addition (`model.go` aliases) is a correctly-placed, non-duplicative,
necessary gap-fill, and the implementer's report honestly flags it as such
rather than hiding it. No blocking or non-blocking defects found in the
reviewed surface.
