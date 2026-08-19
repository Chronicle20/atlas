# Review — Task 18: atlas-channel receive, discard and close arms

Commit range: `3b8de6cc0..4d75fe7ed` (`4d75fe7ed`)
Brief: `.superpowers/sdd/plan/task-18-brief.md`
Report: `.superpowers/sdd/plan/task-18-report.md`
Design: `docs/tasks/task-241-duey-parcel-delivery/design.md` §4.3, §4.4, §5.3, §7.2, §7.3

## Priority 1 — the wire-id → row-id mapping (`resolveParcelByWireId` / `wireIdOf`)

Code: `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go:95-115`

**(a) Single named definition, but in the wrong package — this is the blocking part.**

`wireIdOf` (`duey_action_receive.go:97-99`) is the one and only place the
uuid→uint32 projection is computed; it is reused correctly within this
task (resolution at line 110, and the discard announce at line 250). So
far so good.

But it lives in package `handler` (`services/atlas-channel/.../socket/handler`),
and `handler` imports the `parcel` package (`dueyparcel "atlas-channel/parcel"`,
`duey_action_receive.go:5`, also `duey_action_send.go:7`). Task 21's own plan
entry places the consuming code, `parcel.Model.ToPacket() packetparcel.Parcel`,
in a **new file inside the `parcel` package**
(`docs/tasks/task-241-duey-parcel-delivery/plan.md:2176-2178,2195`:
`services/atlas-channel/atlas.com/channel/parcel/model.go`).

Since `handler` → `parcel` is an existing import edge, `parcel` cannot import
`handler` back without creating an import cycle. `parcel.Model.ToPacket()`
therefore **cannot call `handler.wireIdOf`** — it is not just unexported, it
is unreachable from the package where Task 21 is required to write it. Task
21's implementer has exactly two options neither of which the current code
supports: (1) reimplement the same 4-byte truncation independently in the
`parcel` package (silent duplication, real drift risk if anyone touches one
copy and not the other), or (2) this task's code needs to move first. As
landed, "single named, reusable definition" is not true across the boundary
that matters — the boundary Task 21 will actually hit.

**Fix the reviewer should require before this is safe to build on:** move
`wireIdOf`/the projection into the `parcel` package (e.g.
`func (m Model) WireId() uint32` or a free function in `parcel`), exported,
and have `handler` call the `parcel`-package version instead of defining its
own. That is the one symbol Task 21 must reuse; as committed, no such symbol
exists in `parcel`.

**(b) Scoping — correct.** `resolveParcelByWireId` (`duey_action_receive.go:104-115`)
calls `GetForRecipient(characterId, worldId)` where `characterId` is
`s.CharacterId()` from the caller's own session (`duey_action_receive.go:60,
80`) — never taken from the wire. `GetForRecipient` issues
`GET /parcels?filter[recipientId]=%d&filter[worldId]=%d&filter[status]=pending`
(`parcel/requests.go:19-38`), and atlas-parcel's `ByRecipient` query
(`services/atlas-parcel/atlas.com/parcel/parcel/processor.go:80-82`) is
scoped by recipientId + worldId + status=pending at the DB layer, with
tenant scoping coming from the existing `db.WithContext(ctx)` tenant-gorm
convention used unchanged from Task 6. A crafted `parcelId` can only ever
match a row inside the caller's own pending mailbox; it cannot reach another
character's or another tenant's row. No cross-character/cross-tenant finding.

**(c) Collisions — no detection, first match silently wins.** The loop at
`duey_action_receive.go:109-113` returns the first row (in `GetForRecipient`'s
result order — unordered by anything the query specifies) whose first 4
bytes equal `wireId`; it does not check for a second match. If two of the
caller's own ≤10 pending parcels (`design §6.2 MailboxCapacity`) happen to
collide on their first 4 UUID bytes, `RECEIVE`/`DISCARD` silently resolves
to whichever one the slice happened to return first — potentially the
*wrong* parcel of the *same* caller's own mailbox (never another
character's, per (b)). Actual behavior, not probability: this is exactly a
same-caller wrong-row substitution, not a crash and not a rejection. Given
the mailbox cap of 10 and a full 32-bit truncation space, the odds are low
but not the point — the code has no guard, and it is silent. Non-blocking
given the odds, but worth a one-line fix (reject with `errParcelNotResolved`
on a second match rather than best-effort-first) before this ships broadly.

**(d) Documentation — visible today, not where Task 21 will look.** The
scheme is documented at `handleDueyActionReceive`'s doc comment
(`duey_action_receive.go:40-56`), which is thorough and correctly flags the
open question. But per (a), that comment lives in `handler`, a package
Task 21's `parcel/model.go` cannot import. Recommend the controller either
have this task's fix relocate `wireIdOf` (and its doc comment) into
`parcel`, or explicitly amend Task 21's brief to point at
`duey_action_receive.go`'s comment before that task starts, so the
implementer is not left to independently invent (and likely diverge on) the
same projection.

**Verdict on Priority 1: blocking.** (a) is the substantive defect — not
"an unreviewed engineering call" in the abstract (that part is fine and
well-flagged) but a concrete unreachable-from-Task-21 placement that, left
as-is, will cause Task 21 to reimplement the mapping independently with no
compiler-enforced guarantee the two projections agree. (c) is a real but
low-probability latent bug, reported as non-blocking.

## Priority 2 — the atlas-parcel `PATCH /parcels/{parcelId}` route

Files: `services/atlas-parcel/atlas.com/parcel/parcel/{resource.go,rest.go,resource_test.go}`

- **Tenant scoping**: `handleDiscardParcel` (`resource.go:169-198`) calls
  `NewProcessor(...).Discard(parcelId, input.RecipientId)`, which goes
  through `ProcessorImpl.resolve` (`processor.go:164-192`) — `db.WithContext(p.ctx)`
  is the same tenant-scoped-gorm convention every other route in this file
  already uses (`GetById`, `GetForRecipient`, etc., all pre-existing from
  earlier tasks). No new tenant-scoping code was needed or added; nothing
  here weakens it.
- **Authorization**: `ProcessorImpl.Discard`/`resolve` re-reads the row
  inside a transaction and rejects `m.RecipientId() != recipientId` with
  `ErrNotRecipient` → mapped to 409 (`resource.go:190-192`; `processor.go:172-174`).
  A caller cannot discard someone else's parcel — the recipient id is
  validated server-side against the persisted row, not merely trusted from
  the request body. Covered by the "discard not the recipient" subtest
  (`resource_test.go`, asserts 409 and the row stays `pending`).
- **JSON:API shape**: `DiscardRestModel` (`rest.go:87-109`) follows the
  same GetName/GetID/SetID/relationship-stub pattern as every other
  RestModel in this file (`RestModel`, `parcelStatusRestModel`). Consistent
  with repo convention.
- **Idempotency on an already-discarded row**: `resolve`'s in-memory gate
  (`processor.go:144-149`, `Status() != StatusPending` → `ErrNotPending`)
  plus the DB-level `UpdateStatusIfPending` compare-and-swap
  (`processor.go:178-184`) both reject a second discard with `ErrNotPending`
  → mapped to 409 (`resource.go:193-195`). A double-discard is a clean 409,
  not a silent no-op and not a crash — correct and race-safe (the DB-level
  check closes the concurrent-caller race the in-memory gate alone would
  miss).
- **Test coverage**: `resource_test.go`'s three new subtests
  (`discard`/`discard not the recipient`/`discard missing`) cover the happy
  path (200, row becomes `discarded`), the authorization rejection (409, row
  stays `pending`), and the not-found path (404) — not merely the happy
  path. No dedicated "discard an already-discarded row" subtest exists, but
  that path shares the exact same `ErrNotPending`/409 code path already
  exercised (implicitly) by the recipient-mismatch case's assertions on the
  gate; not flagging as a gap given `ProcessorImpl.Discard`/`resolve` was
  already tested pre-existing (Task 6) and this task only adds the REST
  wrapper around it.

`handleDiscardParcel`, `ProcessorImpl.Discard`, and the underlying
`resolve` helper are all pre-existing from Task 6
(`91dc1cae9`, cited in the report) — the only genuinely new code in
atlas-parcel here is the route registration, the REST model, and the
handler function wiring the two together. This is a legitimate,
self-contained "finish producible work" prerequisite per CLAUDE.md, and the
implementation is correct.

**Verdict on Priority 2: approved, no findings.**

## General pass

### Brief's subtest table (10 rows) vs. what the tests actually assert

The brief's Step-1 table lists 10 subtests. The landed test file has 8:

| brief row | present? | evidence |
|---|---|---|
| receive happy path | yes | `duey_action_receive_test.go:172-177` |
| receive meso only | yes | `:179-184` |
| no free slot | yes | `:186-192` |
| unique conflict | yes | `:194-200` |
| not receivable yet | yes | `:202-207` |
| not addressed to me | **missing** | — |
| already received | **missing** | — |
| discard happy path | yes | `:323-326` |
| discard not mine | yes | `:328-331` |
| close | yes | `TestDueyActionClose`, `:388-403` |

The report's rationale (§"What I implemented") is that resolution is scoped
to the caller's own pending mailbox, so "not addressed to me" and "already
received" collapse into the same not-found path as an unknown id — true as
an implementation fact. But that claim is **untested for `receiveParcel`
specifically**. Every `TestDueyActionReceive` case injects
`getParcel: func(...) (dueyparcel.Model, error) { return p, nil }` — never
an error (`duey_action_receive_test.go:222`). The `err != nil` branch inside
`receiveParcel` (`duey_action_receive.go:138-143`, the reject-with-
`INCORRECT_REQUEST` path) is only exercised by `TestDueyActionDiscard`'s
"discard not mine" case, which calls `discardParcel`, a different function
with its own (near-identical, but separately written) reject call
(`duey_action_receive.go:237-242`). `receiveParcel`'s own not-found branch
has **zero test coverage** despite being one of the three ways the brief's
table says `RECEIVE` must reach `INCORRECT_REQUEST`.

This is a real gap, not merely a documentation nit: the brief explicitly
names these two subtests, and the code path they'd exercise inside
`receiveParcel` is untested. **Blocking** — add the two missing subtests
(or a single one covering `getParcel` returning `errParcelNotResolved` for
`receiveParcel`) before this task is considered complete against its own
brief.

### Design §7.2 pre-flight ordering

`receiveParcel` (`duey_action_receive.go:133-177`): resolve →
free-slot (`:155-159`) → unique-item (`:160-164`) → parcel-state / `ReceivableAt`
(`:167-171`) → build+create saga. Matches §7.2's stated order (free-slot,
then unique-item, then parcel-state) exactly. The parcel *identity*
resolution necessarily precedes all three since none of the three checks
are meaningful without a resolved row; this is not a fourth ordered check
in the design's table, it's a prerequisite, and placing it first is correct.

### Every rejection subtest asserts parcel-stays-pending / session-not-closed

- Receive: none of the five subtests PATCH atlas-parcel at all (the deps
  struct's `discardParcel` field is never populated in `TestDueyActionReceive`,
  and no receive-side write endpoint exists), so "stays pending" is
  structurally guaranteed rather than independently asserted — acceptable,
  matches the report's own comment at `:296-298`. Session-not-closed is
  explicitly asserted every subtest (`:299-301`, via `closeCountingConn`).
- Discard: "discard not mine" asserts `patchedIds == 0`
  (`duey_action_receive_test.go:360-366`), i.e. the PATCH (and therefore any
  status change) never happens on rejection. Session-not-closed asserted at
  `:381-383`.

### DISCARD is genuinely not a saga

Confirmed: `discardParcel` (`duey_action_receive.go:232-251`) calls
`deps.discardParcel` (a direct PATCH wrapper, `parcel/processor.go:186-192`)
and announces synchronously — no `saga.Saga`, no `createSaga` call anywhere
on the discard path. Matches design §4.4/§7.3.

### Test-value provenance

Checked whether the numeric assertions were pinned off the implementation
rather than derived from the brief/design. `5000` (meso amount) is fixed by
the brief's own baseline ("holding one equip... and 5,000 meso") and
asserted against `saga.AwardMesosPayload.Amount` (`:275-277`) — not
implementation-derived. `ParcelRemovedKindDiscarded`/`ParcelOperationRecvNoFreeSlots`
etc. are pre-existing named constants from the packet library (predates this
task, confirmed at `libs/atlas-packet/parcel/clientbound/parcel.go:566-576`),
not invented by this diff. No test in this diff appears to assert a value
read back off the code under test.

### Routing (`duey_action.go`)

`RECEIVE`/`DISCARD`/`CLOSE` are wired mirroring `SEND`'s existing shape
exactly (`duey_action.go:44-61`); each decodes its own wire struct before
dispatch. No issues found.

### Minor / non-blocking

- `p.ItemType()` (a `byte`) is cast `inventory.Type(int8(p.ItemType()))`
  (`duey_action_receive.go:148`) rather than the direct
  `inventory.Type(sp.InventoryType())` pattern used on the send side
  (`duey_action_send.go:183`). Both compile and both pass under the test's
  fixture values (`byte(inventory.TypeValueEquip)` = 1), so this is not a
  demonstrated defect, just an inconsistency in the conversion idiom between
  the two files. Flagging for awareness, not requiring a fix.
- `tools/lint.sh` was not re-run after the final unused-field removal per
  the report; `go build`/`go test` are confirmed clean and the removal
  (a struct field with no other references) has no plausible path to a
  new lint failure. Not independently re-verified by this review (lint was
  not re-run here either, per the task instructions) — noted, not treated
  as a defect.

## Not evaluable

- Whether `tools/lint.sh` is currently clean on
  `services/atlas-channel/atlas.com/channel` — not re-run per this review's
  instructions (a repo-wide gate is running concurrently).
- The correctness of `atlas-saga`'s upstream `WithdrawFromParcel` /
  `WithdrawFromParcelPayload` definitions consumed via re-export
  (`saga/model.go`) — outside this diff's surface; only confirmed the
  re-export compiles (`go build` clean per the report) and is referenced
  consistently.

## Summary

Two blocking findings:

1. `wireIdOf`'s projection lives in `handler`, a package Task 21's
   `parcel.Model.ToPacket()` (per the plan, in the `parcel` package) cannot
   import back without a cycle — the "single reusable definition" claim
   does not hold across the boundary that matters. Move the projection into
   `parcel` (exported) before Task 21 starts.
2. The brief's two "not addressed to me" / "already received" `RECEIVE`
   subtests are missing, and `receiveParcel`'s own not-found rejection
   branch (`duey_action_receive.go:138-143`) has zero test coverage as a
   result — only `discardParcel`'s separate reject call is exercised by an
   analogous case.

Everything else reviewed — the atlas-parcel PATCH route (Priority 2), the
§7.2 pre-flight ordering, session-not-closed/parcel-stays-pending
assertions, DISCARD-not-a-saga, routing, and test-value provenance — is
correct with cited evidence.
