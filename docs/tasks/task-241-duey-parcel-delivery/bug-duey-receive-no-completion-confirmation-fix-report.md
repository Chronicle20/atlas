# Fix report — Duey RECEIVE succeeds but the client never gets a completion confirmation

- Task: task-241-duey-parcel-delivery
- Bug: bug-duey-receive-no-completion-confirmation.md

## What was implemented

Added the missing `PARCEL_RECEIVED` event and its channel-side announcer,
exactly per the brief's `## Fix` section. Two rulings honored without
revisiting:

- The event is emitted from `handleReleaseFromParcel` in atlas-parcel (not a
  saga-completion hook) — the release IS the completion, the same posture as
  `PARCEL_SENT`'s "the row create IS the completion".
- `PARCEL_REMOVED` alone unlocks the dialog. No `RECV_ENABLE_ACTIONS`
  announce was added.

### atlas-parcel

- `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go` —
  added `StatusEventParcelReceived = "PARCEL_RECEIVED"` and
  `StatusEventParcelReceivedBody{ ParcelId uuid.UUID `json:"parcelId"` }`,
  doc comments documenting the compensation-window trade-off.
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go`
  — added `ParcelReceivedStatusEventProvider(characterId uint32, parcelId
  uuid.UUID)`, keyed by `characterId` like its two siblings.
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go`
  — in `handleReleaseFromParcel`'s `buffer.Emit` closure, after the existing
  custody `RELEASED` put, now also puts
  `parcelproducer.ParcelReceivedStatusEventProvider(b.RecipientId, m.Id())`
  onto `parcelmsg.EnvStatusEventTopic`. The `ErrAlreadyReleased` replay
  branch is unaffected — it still returns before the emit lands, so replay
  emits neither RELEASED nor RECEIVED.

### atlas-channel

- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go`
  — mirrored `StatusEventParcelReceived` + `StatusEventParcelReceivedBody`
  field-for-field (same JSON tag `parcelId`).
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
  — added `handleParcelReceivedEvent`, registered as the fourth handler on
  `EnvStatusEventTopic` in `InitHandlers` (after `handleParcelSentEvent`).
  Same guards/posture as `handleParcelSentEvent`: event-type guard,
  `t.Is(sc.Tenant())`, then `IfPresentByCharacterId` (silent no-op if the
  recipient's session isn't present on this channel). Announces
  `parcelcb.ParcelRemovedBody(dueyparcel.WireId(e.Body.ParcelId),
  parcelcb.ParcelRemovedKindClaimed)`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/received_test.go`
  (new) — modeled on `sent_test.go`: wrong event type ignored, wrong tenant
  ignored, absent session no-op, and a happy path asserting the announced
  bytes are `[0x17, wireId LE32, ParcelRemovedKindClaimed]` — the
  tenant-resolved PARCEL_REMOVED mode (0x17 per
  `parcel_notify_test.go`/`docs/packets/dispatchers/parcel.yaml`), the wire
  parcelId (`dueyparcel.WireId`), and `kind == 0` (Claimed).

### Producer-side test

- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`
  — extended the existing `"release"` case to assert `PARCEL_RECEIVED` is
  emitted alongside the custody `RELEASED` ack, addressed to `RecipientId`
  (100). Extended the existing `"release replay"` case to assert the replay
  (`ErrAlreadyReleased`) path re-emits neither `RELEASED` nor
  `PARCEL_RECEIVED`.

No new test-infrastructure was needed: the existing `recordedEvent`/
`eventsOfType` helpers already decode any envelope carrying
`characterId`/`type`, which the `PARCEL_RECEIVED` event's `StatusEvent[E]`
envelope satisfies unchanged.

## What was tested

```sh
cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...
```
Result:
```
?   	atlas-parcel	[no test files]
?   	atlas-parcel/kafka/consumer	[no test files]
ok  	atlas-parcel/kafka/consumer/custody	0.024s
?   	atlas-parcel/kafka/message	[no test files]
?   	atlas-parcel/kafka/message/custody	[no test files]
?   	atlas-parcel/kafka/message/parcel	[no test files]
?   	atlas-parcel/kafka/producer/custody	[no test files]
?   	atlas-parcel/kafka/producer/parcel	[no test files]
ok  	atlas-parcel/parcel	0.085s
?   	atlas-parcel/rest	[no test files]
```

```sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Result: all packages `ok` (or `[no test files]`), no `FAIL` lines — full
output was ~90 lines; `go test ./... | grep -v '^ok\|no test files'` returned
nothing.

Targeted run of the new handler's tests:
```sh
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/parcel/... -run TestParcelReceivedEvent -v
```
Result:
```
=== RUN   TestParcelReceivedEvent
=== RUN   TestParcelReceivedEvent/online_recipient
=== RUN   TestParcelReceivedEvent/offline_recipient
=== RUN   TestParcelReceivedEvent/wrong_tenant
=== RUN   TestParcelReceivedEvent/wrong_event_type
--- PASS: TestParcelReceivedEvent (0.00s)
    --- PASS: TestParcelReceivedEvent/online_recipient (0.00s)
    --- PASS: TestParcelReceivedEvent/offline_recipient (0.00s)
    --- PASS: TestParcelReceivedEvent/wrong_tenant (0.00s)
    --- PASS: TestParcelReceivedEvent/wrong_event_type (0.00s)
PASS
ok  	atlas-channel/kafka/consumer/parcel	0.008s
```

## Files changed

- `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go`
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go`
- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/received_test.go` (new)

## Self-review findings

- Verified `ParcelRemovedBody`'s wire layout against
  `libs/atlas-packet/parcel/clientbound/parcel_notify_test.go`'s fixture
  (`[mode, parcelId little-endian uint32, kind]`) rather than guessing the
  byte order; the new test asserts against that layout, not an invented one.
- Confirmed `ParcelOperationParcelRemoved` resolves to mode `0x17` (23) on
  gms_v83 via the existing fixture — matches the brief's "mode 23 on all
  gms_v* columns" claim; did not re-verify jms_v185's mode 24 (out of scope
  — that's the tenant operations table, not this task's code).
- Did not add a `RECV_ENABLE_ACTIONS` announce, per the ruling.
- Did not build a re-add packet for the `handleRestoreParcel` compensation
  path — documented the trade-off in the emitter's doc comment instead, per
  the brief.
- `ErrAlreadyReleased` replay branch confirmed still a no-op: it returns
  before `buffer.Emit`'s closure result is used, so neither the custody
  RELEASED nor the new PARCEL_RECEIVED put ever executes on replay (new test
  assertion covers this).

## Issues or concerns

None. The "Not yet answered" section in the bug file (the compensation
window, and live re-confirmation) is intentionally left open per the brief —
not this task's remit to close.

## Outcome

Implementation complete, module-local builds and tests green in both
`atlas-parcel` and `atlas-channel`. Live re-confirmation (updating the bug
file's own `## Outcome` section) is still pending, as the bug file's "Not
yet answered" section already notes.
