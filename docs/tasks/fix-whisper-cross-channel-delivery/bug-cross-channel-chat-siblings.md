# bug: party/buddy/guild/alliance, messenger, and pink-text chat are all dropped for recipients on another channel

**Reproduced:** not reproduced live. This is the *same defect* as
`bug-whisper-cross-channel-delivery.md`, which WAS reproduced live (tenant
`625de849-e34f-45c8-95e6-b8e794774422`, GMS 83.1, namespace `atlas-pr-1407`,
sender on channel 0, recipient on channel 1, packet never delivered, no error
logged). Three sibling handlers in the same file carry the identical pattern
verbatim. Treat the whisper reproduction as the evidence; the mechanism is
character-for-character the same and is established by code reading, not
guessed.

**Observed (by inspection):** in
`services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`,
each of these three handlers gates on the *sender's* channel and then looks
recipients up in the *sender's* channel session registry:

- `handleMultiChat` — buddy / party / guild / alliance chat
- `handleMessengerChat` — messenger chat
- `handlePinkChat` — pink-text broadcast to a recipient list

Each does:

```go
if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) { return }
...
for _, cid := range e.Body.Recipients {
    err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cid, ...)
}
```

`e.ChannelId` is the sender's channel. Handlers are registered once per
`(tenant, world, channel)` socket listener
(`services/atlas-channel/atlas.com/channel/main.go:425`), so only the
sender's-channel handler runs at all, and `sc.Channel()` then confines the
recipient lookup to that one channel. `IfPresentByCharacterId`
(`session/processor.go:183-191`) returns `nil` on a miss, so every dropped
recipient is silent.

Net effect: a party/guild/alliance/buddy message, a messenger message, and a
pink-text notice reach only those recipients who happen to be on the sender's
own channel. Party and guild members are cross-channel by nature, so this is
the common case, not an edge case.

**Expected:** every recipient in `e.Body.Recipients` who is online anywhere in
the same world receives the packet, exactly once.

**Root cause:** identical to the whisper bug — recipient delivery is scoped to
the sender's channel instead of to the world. Fixed for `handleWhisperChat` in
commit `30b459905` on this branch; these three were deliberately left out of
that commit's scope and are now in scope.

## Fix

Read `handleWhisperChat` and the pure helper `whisperDeliveryPlan` in
`services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
as they stand after `30b459905` **first** — that is the established shape for
this fix; follow it rather than inventing a second idiom.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
  — `handleMultiChat`, `handleMessengerChat`, `handlePinkChat`:
  - Replace each `if !sc.Is(t, e.WorldId, e.ChannelId) { return }` with the
    world-scoped `if !sc.IsWorld(t, e.WorldId) { return }`. Use the existing
    `server.Model.IsWorld`; add no new predicate.
  - Keep the per-handler `e.Type` guard exactly as it is.
  - Recipient delivery already goes through
    `IfPresentByCharacterId(sc.Channel())(cid, ...)`, which becomes correct
    once the gate is world-scoped: the one channel holding each recipient
    delivers, every other channel is a silent no-op. Exactly-once delivery
    follows from a character having at most one session in a world.
  - **REST fan-out**, same concern as the whisper fix: `handleMultiChat` and
    `handlePinkChat` fetch the sender character
    (`character.NewProcessor(l, ctx).GetById()(e.ActorId)`) *before* the
    recipient loop, so world-scoping would multiply that call by the channel
    count for every message. Resolve the sender character lazily — only once
    this handler has established that at least one recipient's session is
    present on `sc.Channel()`, e.g. by pre-filtering the recipient list with
    `session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())` and
    returning early when nothing matches. `handleMessengerChat` needs no
    character lookup at all; leave it lookup-free.
  - `handlePinkChat` tolerates a failed sender lookup today (it falls back to
    an empty `characterName`). Preserve that behaviour.
  - Do NOT change `handleGeneralChat` or `handlePetChat`. Both are
    map-scoped, not recipient-list-scoped: general chat goes to
    `ForSessionsInMap` on the sender's own field, and pet chat's owner is by
    construction on the channel the event originated from. Their `sc.Is(...)`
    gate is correct.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer_test.go`
  (extend the file added in `30b459905`) — for each of the three handlers, a
  case that must fail before and pass after:
  - recipient on a channel other than the sender's, handler bound to the
    recipient's channel → recipient receives the packet.
  - two recipients on two different channels → each channel's handler
    delivers to its own recipient and to no one else (proves exactly-once
    across the fan-out).
  - handler bound to a different world → nothing emitted.
  Reuse the `net.Pipe` + background-reader session fixtures and the
  `httptest` character fixture already established in that file. Do not add a
  `*_testhelpers.go` file.

## Not yet answered

- `WhisperSendResult.success` is out of scope here and is being handled on a
  separate branch stacked on task-238 (it needs that task's presence record).
  Do not touch it.
- If any of the three handlers turns out to have a *second*, distinct reason
  for being channel-scoped that this file has missed, stop and report it
  rather than forcing the uniform change.

## Resolution

Fixed in `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
by replacing `sc.Is(t, e.WorldId, e.ChannelId)` with `sc.IsWorld(t, e.WorldId)` in
`handleMultiChat`, `handleMessengerChat`, and `handlePinkChat`, matching the shape
`handleWhisperChat`/`whisperDeliveryPlan` established in commit `30b459905`.

- `handleMessengerChat` needed no other change: it does no character lookup, so
  the recipient loop through `IfPresentByCharacterId(sc.Channel())` becomes
  correct as soon as the gate is world-scoped.
- `handleMultiChat` and `handlePinkChat` both fetch the sender character before
  the recipient loop. To avoid multiplying that REST call by the channel count,
  a new helper `presentRecipients` pre-filters `e.Body.Recipients` down to those
  with a session on `sc.Channel()` (via `session.NewProcessor(l,
  ctx).GetByCharacterId(sc.Channel())`) and the handler returns early — before
  the character fetch — when none are present. This mirrors the "test presence
  before the character fetch" idiom already used in `handleWhisperChat`'s
  `sendReceive` branch.
- `handlePinkChat`'s tolerance of a failed sender lookup (falls back to an
  empty `characterName`) is unchanged.
- `handleGeneralChat` and `handlePetChat` were left untouched — both are
  map-scoped rather than recipient-list-scoped, so their `sc.Is(...)` gate is
  correct as-is, per the brief.

Regression coverage added to `consumer_test.go` (extending the fixtures added in
`30b459905`, no new `*_testhelpers.go` file): for each of the three handlers,
one test proves delivery to a recipient on a channel other than the sender's,
one proves two recipients on two different channels each receive delivery
from their own channel's handler only (exactly-once across the fan-out), and
one proves a handler bound to a different world emits nothing. All nine new
tests were confirmed to fail against the pre-fix `sc.Is(...)` gate (verified
by stashing the `consumer.go` change and re-running the three
`*_RecipientChannel_Delivers` cases, which timed out waiting for the expected
packet) and pass after restoring the fix.

`go build ./...` and `go test ./...` from
`services/atlas-channel/atlas.com/channel` both pass.
