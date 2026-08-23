# Bug — Duey RECEIVE succeeds but the client never gets a confirmation, so CParcelDlg stays locked

- Task: task-241-duey-parcel-delivery
- Branch: task-241-duey-parcel-delivery
- PR: atlas-pr-1434
- Reported: live test

## Reproduced

Reported from live testing. Not re-reproduced in-game in this session — the
defect is a missing producer/consumer pair that is conclusively visible in the
source (see Root cause), so no log evidence was needed to establish it.

## Observed

Player clicks "Receive" on a Duey parcel row. The item **is** granted (the
parcel_receive saga completes). But:

- the received row is never removed from the parcel list,
- no further Receive or Delete works,
- tabs cannot be switched, and the dialog cannot be closed.

The dialog is stuck in its "request in flight" state.

## Expected

On successful receive, the client should receive `PARCEL[PARCEL_REMOVED]`
(operations key `PARCEL_REMOVED`, mode 23 on all gms_v* columns, 24 on
jms_v185) with `kind == ParcelRemovedKindClaimed`, which removes the row,
shows SP_3900_SUCCESSFULLY_CLAIMED, and re-enables the dialog controls.

## Root cause

**Nothing announces anything to the client when a parcel_receive saga
completes.** The path simply ends server-side.

Evidence, in order:

1. `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go:receiveParcel`
   runs the pre-flight and calls `deps.createSaga(...)`. On the success path it
   announces **nothing** — by design, deferring to a saga-completion consumer.
   Its own doc comment on `discardParcel` says so explicitly: "unlike receive,
   whose PARCEL_REMOVED is announced on saga completion, **by a consumer
   outside this task's scope**".
2. That consumer does not exist. `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
   registers exactly three handlers: `handleShowParcelCommand`,
   `handleParcelArrivedEvent` (`PARCEL_ARRIVED`), and `handleParcelSentEvent`
   (`PARCEL_SENT`). There is no receive-completion handler.
3. Nor does the event exist. `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go`
   defines only `PARCEL_ARRIVED` and `PARCEL_SENT`; `kafka/producer/parcel/producer.go`
   has only `ParcelArrivedStatusEventProvider` and `ParcelSentStatusEventProvider`.
4. Repo-wide, `ParcelRemovedBody` has exactly one non-test caller:
   `duey_action_receive.go:299`, the **discard** path.

Why that leaves the UI locked (not merely stale): `CParcelDlg` disables its own
controls when it sends `CTabReceive::ReceiveParcel` and only re-enables them
when a `PARCEL` result packet arrives. Verified by decompiling
`CParcelDlg::OnPacket` @0x6f56ea on the v83 IDB (session `754107bf`): every
explicit case and the default arm call `CParcelDlg::SetCtrlEnabled(v3, 1)`.
With no packet at all, `SetCtrlEnabled(1)` is never reached — hence no
close, no tab switch, no further action.

Note (corrects a plausible earlier reading): case 23 (`PARCEL_REMOVED`) **does**
call `SetCtrlEnabled(a1, 1)` itself, immediately after `CParcelDlg::RemoveParcel(a1, v18)`
and the SP_3899/SP_3900 notice. So `PARCEL_REMOVED` alone both removes the row
and unlocks the dialog — a separate `RECV_ENABLE_ACTIONS` (mode 20) packet is
**not** needed and must not be added.

The discard path is unaffected: it announces `PARCEL_REMOVED` synchronously
already, so it unlocks correctly.

## Fix

Add the missing PARCEL_RECEIVED event and its channel-side announcer.

Ruling on **where** the event is emitted: from `handleReleaseFromParcel` in
atlas-parcel — the moment the parcel row transitions to `received`. This is
the direct analogue of `PARCEL_SENT`, which is emitted from
`handleAcceptToParcel` because "the row create IS the completion". Accepted
trade-off, mirroring the existing posture: if the downstream
`accept_to_character` fails and `handleRestoreParcel` compensates, the client
will already have removed the row and will only see it again on reopening the
dialog. Document that in the emitter's doc comment; do not build a re-add
packet for it.

Files:

- `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go`
  — add `StatusEventParcelReceived = "PARCEL_RECEIVED"` and
  `StatusEventParcelReceivedBody{ ParcelId uuid.UUID \`json:"parcelId"\` }`.
  The envelope's `CharacterId` is the recipient.
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go`
  — add `ParcelReceivedStatusEventProvider(characterId uint32, parcelId uuid.UUID)`,
  keyed by characterId like its two siblings.
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go`
  — in `handleReleaseFromParcel`, inside the existing `buffer.Emit` closure,
  `mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelReceivedStatusEventProvider(b.RecipientId, m.Id()))`
  alongside the existing custody RELEASED put. The `ErrAlreadyReleased` replay
  branch must stay a no-op (it already returns before the emit lands).
- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go`
  — mirror `StatusEventParcelReceived` + `StatusEventParcelReceivedBody`
  field-for-field (separate Go modules; JSON tags must match exactly).
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
  — add `handleParcelReceivedEvent`, registered as a fourth handler on
  `EnvStatusEventTopic` in `InitHandlers`. Same guards and posture as
  `handleParcelSentEvent` (event-type guard, `t.Is(sc.Tenant())`, then
  `IfPresentByCharacterId` so an absent session is a silent no-op). It
  announces
  `parcelcb.ParcelRemovedBody(dueyparcel.WireId(e.Body.ParcelId), parcelcb.ParcelRemovedKindClaimed)`.
  `WireId` is `services/atlas-channel/atlas.com/channel/parcel/model.go:72` —
  the same 4-byte big-endian projection the OPEN list and RECEIVE resolution
  already agree on, so the client's row id matches.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/received_test.go`
  (new) — model it on the existing `sent_test.go`: wrong type ignored, wrong
  tenant ignored, absent session no-op, and a happy path asserting the
  announced bytes carry the tenant-resolved `PARCEL_REMOVED` mode, the wire
  parcelId, and `kind == 0` (Claimed).
- Producer-side test alongside `services/atlas-parcel/.../kafka/consumer/custody/`'s
  existing release tests — assert the release emits the PARCEL_RECEIVED
  message in addition to the custody RELEASED ack, and that the replay
  (`ErrAlreadyReleased`) path emits neither.

## Not yet answered

- The compensation window described above is accepted, not solved. If live
  testing later shows `accept_to_character` failing with any regularity, the
  right follow-up is a re-add packet from `handleRestoreParcel`
  (`PARCEL[PARCEL_ARRIVED]`, mode 24, which calls `AddNewParcel`), not a
  change to where PARCEL_RECEIVED is emitted.
- Not re-tested in-game yet; live confirmation is required before this file's
  outcome section can be closed.

## Outcome

Fixed in **e071e5708** — "fix(parcel): announce PARCEL_RECEIVED so RECEIVE
unlocks the Duey dialog". Implemented exactly as the `## Fix` inventory above,
both rulings honoured (emitted from `handleReleaseFromParcel`; no
`RECV_ENABLE_ACTIONS` announce added).

Gates:

- `tools/verify.sh --quick --base e071e5708^` — exit 0, all checks passed.
  Docker bake and `-race` were skipped (`--quick`); the flagless run is still
  owed before the PR is called ready.
- `atlas-reviewer` over `e071e5708^..e071e5708` — **APPROVED**, 0 blocking,
  0 non-blocking. Artifact:
  `docs/tasks/task-241-duey-parcel-delivery/reviews/bug-duey-receive-no-completion-confirmation-review.md`.
  It independently confirmed the cross-module JSON-tag identity of
  `StatusEventParcelReceivedBody`, the handler's registration on
  `EnvStatusEventTopic`, that the `ErrAlreadyReleased` replay emits neither
  message, and the `WireId` / `ParcelRemovedKindClaimed` projection.

**Live confirmation: still pending.** Not re-tested in-game. Re-test is:
receive a parcel from Duey and confirm the row disappears, SP_3900
"successfully claimed" shows, and the dialog accepts further actions / tab
switches / close.
