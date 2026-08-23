# Review — bug-duey-receive-no-completion-confirmation (commit e071e5708)

## Scope

Reviewed the single commit `e071e5708` (`fix(parcel): announce PARCEL_RECEIVED
so RECEIVE unlocks the Duey dialog`), diffed against
`bug-duey-receive-no-completion-confirmation.md`'s Fix section and the
implementer's fix report. Files touched:

- `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`
- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/received_test.go` (new)

`scope_confirmed`: the diff matches the bug's prescribed fix exactly — same
files, same event name, same emission point, same consumer shape. No scope
mismatch.

## Findings

### 1. Field-for-field JSON tag identity — PASS

`StatusEventParcelReceivedBody` in both modules is:

```go
type StatusEventParcelReceivedBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}
```

atlas-parcel: `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go:67-69`
atlas-channel: `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go:91-93`

Both share the same `json:"characterId"` / `json:"type"` / `json:"body"`
envelope (`StatusEvent[E]`, kafka.go:43-47 parcel-side, kafka.go:65-69
channel-side) — unchanged by this commit, and already consistent. The new
type is a straight generic-typed mirror; `parcelmsg.StatusEvent[StatusEventParcelReceivedBody]`
round-trips correctly.

### 2. Handler registered on the status-event topic — PASS

`services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go:64-68`
adds a third `rf(t, ...)` registration reusing the same `t` variable that was
resolved from `parcelmsg.EnvStatusEventTopic` at line 51 (shared with
`handleParcelArrivedEvent` and `handleParcelSentEvent`), and appends its
handle to the same `handles` slice returned to the caller. Confirmed by
reading the full `InitHandlers` body — no separate topic resolution, no
accidental registration on `EnvCommandTopic`.

### 3. `ErrAlreadyReleased` replay emits neither event — PASS

`services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go:179-186`:

```go
err := buffer.Emit(p)(func(mb *buffer.Buffer) error {
    m, rerr := processor(l, ctx, db).ReleaseCustody(b.ParcelId, b.RecipientId)
    if rerr != nil {
        return rerr
    }
    if perr := mb.Put(custody.EnvStatusTopic, custodyproducer.ReleasedStatusEventProvider(...)); perr != nil {
        return perr
    }
    return mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelReceivedStatusEventProvider(b.RecipientId, m.Id()))
})
```

`ReleaseCustody` returns `parcel.ErrAlreadyReleased` as `rerr`, which
short-circuits the closure before either `mb.Put` call runs. Both the
custody RELEASED ack and the new PARCEL_RECEIVED emit are unreachable on
replay. `consumer_test.go`'s "release replay" case
(`consumer_test.go:265-267`) asserts `len(receivedEvents) == 1` (the count
from the prior successful "release" subtest, not incremented by the replay)
with an explicit "replay must not re-emit PARCEL_RECEIVED" message — this is
a real assertion, not a tautology, since the replay case reuses `rp.events`
accumulated across both subtests in the table run. Verified `go test
./kafka/...` passes for atlas-parcel.

### 4. Announced packet: `ParcelRemovedKindClaimed` + `WireId` 4-byte BE — PASS

`services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go:266`:

```go
session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(
    parcelcb.ParcelRemovedBody(dueyparcel.WireId(e.Body.ParcelId), parcelcb.ParcelRemovedKindClaimed))
```

- `dueyparcel.WireId` (`services/atlas-channel/atlas.com/channel/parcel/model.go:72`,
  reachable via grep) is `binary.BigEndian.Uint32(id[:4])` — the same
  function `Model.ToPacket()` uses for the OPEN list and the same one
  `duey_action_receive.go`'s RECEIVE resolution reads back against, per its
  doc comment.
- `parcelcb.ParcelRemovedKindClaimed = byte(0)`
  (`libs/atlas-packet/parcel/clientbound/parcel.go:592`), matching the
  bug doc's `kind == ParcelRemovedKindClaimed` requirement and the new
  test's `kind == 0` expectation.
- `received_test.go`'s happy-path case builds the expected byte sequence as
  mode byte + 4 little-endian bytes of `WireId(parcelId)` + kind byte, which
  is how `ParcelRemovedBody` (`libs/atlas-packet/parcel/clientbound/parcel_body.go:183`)
  actually encodes a `uint32` onto the wire — confirmed against the existing
  fixture in `parcel_notify_test.go:37-38` (`ParcelRemovedBody(7, Claimed)` →
  `{0x17, 0x07, 0x00, 0x00, 0x00, Claimed}`).

### 5. No `RECV_ENABLE_ACTIONS` announce added — PASS

Grepped the diff (`git show e071e5708 | grep -i "RECV_ENABLE\|EnableActions"`):
the only matches are prose references inside the fix-report markdown
documenting the *decision not to* add one. No code hunk in
`consumer.go`/`kafka.go` introduces a second announce or a distinct mode-20
packet. `handleParcelReceivedEvent` issues exactly one
`session.Announce` call.

### 6. Correctness / error paths

- Producer side: `ParcelReceivedStatusEventProvider` keys by `characterId`
  (`producer.go:52`), matching the sibling providers' per-character
  ordering; `b.RecipientId` is passed at the call site, which is correct —
  it is the recipient, not the sender.
- If the first `mb.Put` (custody RELEASED) fails, the closure returns before
  the second `Put`, so `ParcelReceivedStatusEventProvider` is never invoked
  on a partial custody-ack failure either — consistent, no split-brain state.
- Consumer side: `IfPresentByCharacterId` guard for absent sessions,
  `t.Is(sc.Tenant())` tenant guard, and event-type guard are all present and
  ordered the same as `handleParcelSentEvent`'s established shape
  (`consumer.go:239` region, confirmed by reading the new handler in full).
- Error from `Announce` is logged (`l.WithError(err).Errorf(...)`), not
  swallowed silently and not propagated as a retry — matches sibling
  handlers' posture.

### 7. Test honesty

- `received_test.go`'s four subtests (online recipient, offline recipient,
  wrong tenant, wrong event type) each assert a distinct guard: the "online
  recipient" case is the only one that positively asserts announced bytes,
  and it checks exact byte content (mode, wire parcelId, kind), which would
  fail under the pre-fix state (no handler existed at all, so this file and
  the function under test are wholly new — the test cannot pass without the
  fix by construction).
- `consumer_test.go`'s "release" case asserts `receivedEvents[0].characterId
  == 100` (the recipient) — this genuinely could not pass before the fix,
  since `parcelmsg.StatusEventParcelReceived` did not exist prior to this
  commit.
- `consumer_test.go`'s "release replay" case pins the replay non-emission,
  which is exactly the seam called out in the review brief.

Ran `go build ./...` and `go test ./kafka/...` in both
`services/atlas-parcel/atlas.com/parcel` and
`services/atlas-channel/atlas.com/channel` — all green.

## Not evaluable

- Whether `handleReleaseFromParcel`'s pre-existing `ReleaseCustody` call and
  its surrounding saga plumbing are themselves correct is out of this unit's
  scope (untouched by the diff); only the two new `mb.Put` lines were
  reviewed for correctness.
- Live in-game confirmation (the bug file's own "Not yet answered" section)
  was not performed by this review — that is explicitly deferred, and the
  bug file's Outcome section is still "(pending)". This does not block
  approval of the code change itself, since the fix is directly evidenced by
  IDB decompilation cited in the bug doc, not by a live repro.
- The compensation-window trade-off (client already removed the row before
  a downstream `accept_to_character` failure triggers `handleRestoreParcel`)
  is explicitly accepted and documented as a known trade-off in the bug
  file's "Not yet answered" section, not a defect of this commit.

## Verdict

All five focus areas from the review brief check out with cited evidence.
No blocking defects found.
