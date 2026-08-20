# bug: a whisper is never delivered when the recipient is on a different channel than the sender

**Reproduced:** tenant `625de849-e34f-45c8-95e6-b8e794774422`, region GMS,
ms.version 83.1, namespace `atlas-pr-1407` (task-238 ephemeral env, but the
defect is present on `main` — none of task-238's diff touches the delivery
path). Character `Atlas` (id 1) logged in on world 0 / channel 0, character
`Chronicle` (id 2) logged in on world 0 / channel 1. `Atlas` whispers `yo` to
`Chronicle`.

**Observed:** the whisper is accepted, published, and consumed — and then
silently dropped. From `kubectl logs -n atlas-pr-1407
atlas-channel-5758d674df-bkw7l`:

```
11:19:32.499Z channel.id:0 "[CharacterChatWhisperHandle] read [mode [6] updateTime [0] targetName [Chronicle] msg [yo]]"
11:19:32.682Z "Message received {\"worldId\":0,\"channelId\":0,\"mapId\":240000001,\"instance\":\"00000000-0000-0000-0000-000000000000\",\"actorId\":1,\"message\":\"yo\",\"type\":\"WHISPER\",\"body\":{\"recipient\":2}}." originator:EVENT_TOPIC_CHARACTER_CHAT-ac3e
11:19:32.683Z "Issuing [GET] request to [.../api/characters/1]."   -> name "Atlas"
11:19:32.685Z "Issuing [GET] request to [.../api/characters/2]."   -> name "Chronicle"
```

Nothing follows. No announce, no error. `Chronicle` (whose session logs under
`channel.id:1` in the same pod, session `40352b59-a927-4d3a-b7cc-e43a25c08748`)
receives no whisper packet.

The event body itself is correct end to end: `type: WHISPER`, `recipient: 2`.
The Kafka contract is not at fault.

**Expected:** `Chronicle` receives the `CharacterChatWhisper` packet
(`WhisperReceive`, mode `0x12`, sender name, sender's channel id) regardless of
which channel in the same world holds their session. Same-world cross-channel
whisper is the normal case in a multi-channel world; whisper is world-scoped in
the client, not channel-scoped — which is exactly why `WhisperReceive` carries
the sender's channel id as a field.

**Root cause:** the chat-event consumer scopes *recipient* delivery to the
*sender's* channel, twice over.

`services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go:154`
— `handleWhisperChat`:

1. Line ~160: `if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) { return }`.
   `e.ChannelId` is the *sender's* channel (set by
   `services/atlas-messages/atlas.com/messages/message/producer.go:43`
   `whisperChatEventProvider` from the sender's `field.Model`). Handlers are
   registered once per `(tenant, world, channel)` socket listener
   (`services/atlas-channel/atlas.com/channel/main.go:425` — `sc := h.ServerModel`,
   then `message.InitHandlers(fl)(sc)(wp)(rh)`), so only the handler bound to
   the sender's channel survives this gate. The handler bound to channel 1 —
   the only one whose session registry holds `Chronicle` — returns immediately.

2. Line ~178: even inside the surviving handler, delivery uses
   `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.Recipient, ...)`.
   `sc.Channel()` is the sender's channel, so the lookup misses. And
   `IfPresentByCharacterId` (`session/processor.go:183-191`) returns `nil` when
   the session is not found, so the miss produces no error log — which is why
   the trace ends with no diagnostic at all.

Same-channel whisper works; that is why this was not caught earlier.

Note the sender's own "message sent" confirmation (`WhisperSendResult`, mode
`0x0A`, `success: true`) is correctly scoped — it *must* go only to the sender's
channel — and must stay gated on `sc.Is(...)`.

## Fix

- `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
  — `handleWhisperChat`: split the one channel-scoped gate into two gates.
  - Replace the single `sc.Is(t, e.WorldId, e.ChannelId)` early return with a
    world-scoped early return: `if !sc.IsWorld(t, e.WorldId) { return }`.
    `Model.IsWorld` already exists at
    `services/atlas-channel/atlas.com/channel/server/model.go` (tenant + world
    match, no channel comparison) — do not add a new predicate.
  - Sender confirmation (`WhisperSendResult`) stays inside
    `if sc.Is(t, e.WorldId, e.ChannelId) { ... }` so it is emitted exactly once,
    by the sender's own channel handler.
  - Recipient delivery (`WhisperReceive`) runs on every channel handler in the
    world; `IfPresentByCharacterId(sc.Channel())(e.Body.Recipient, ...)` then
    resolves on exactly the one channel that holds the recipient's session and
    is a no-op everywhere else.
  - Avoid the now-redundant REST fan-out: the two `character.NewProcessor(...).GetById()`
    calls currently run before either announce. Fetch the *recipient* character
    (needed for the send-result target name) only inside the sender-channel
    branch, and fetch the *sender* character (needed for `WhisperReceive`'s
    from-name and gm flag) only after the recipient's session is confirmed
    present on this channel — otherwise every channel in the world issues a
    `GET /api/characters/{id}` per whisper. Use
    `session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.Recipient)`
    to test presence before the character fetch, then announce to that session.
  - Keep `byte(e.ChannelId)` as the channel field of `WhisperReceive` — that is
    the sender's channel and is what the client renders.
  - Keep the existing mode bytes (`0x0A` send result, `0x12` receive) exactly as
    they are; they are out of scope for this fix.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer_test.go`
  (new file) — table test over `handleWhisperChat` that must fail before and
  pass after:
  - sender on channel 0, recipient on channel 1, handler bound to channel 1 →
    recipient receives `WhisperReceive`; sender receives nothing from this
    handler.
  - sender on channel 0, recipient on channel 1, handler bound to channel 0 →
    sender receives `WhisperSendResult`; recipient receives nothing from this
    handler.
  - sender and recipient both on channel 0, handler bound to channel 0 → both
    packets emitted, exactly once each.
  - handler bound to a *different world* → nothing emitted.
  - Use the project Builder pattern for session/server fixtures; do not add a
    `*_testhelpers.go` file. If the existing seams make the announce path
    untestable without a new constructor, prefer extracting the delivery
    decision into a small pure function in the same package and testing that,
    over widening a production constructor for tests.

## Not yet answered

- **Sibling handlers have the identical defect and are deliberately NOT in
  scope for this branch.** `handleMultiChat` (buddy/party/guild/alliance),
  `handleMessengerChat`, and `handlePinkChat` in the same file all gate on
  `sc.Is(t, e.WorldId, e.ChannelId)` and then look recipients up with
  `IfPresentByCharacterId(sc.Channel())`, so every one of them drops recipients
  who are on a channel other than the sender's. The user scoped this branch to
  whisper. Do not widen it — leave those three untouched and let the report
  surface them for a follow-up decision.
- **`WhisperSendResult.success` is always `true`** even when the recipient is
  offline; `atlas-messages` only errors when the recipient *character* does not
  exist (`services/atlas-messages/atlas.com/messages/message/processor.go`
  `HandleWhisper`). Reporting "not online" needs a world-wide session lookup
  that does not exist today. Out of scope — do not attempt it here; leave the
  literal `true` as is.
- If `Model.IsWorld` turns out not to be exported/usable from the consumer
  package, add nothing new: use it via the existing `server.Model` receiver.
  Do not reintroduce a channel comparison.

## Resolution

Fixed in `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`
(`handleWhisperChat`). The single channel-scoped gate (`sc.Is(t, e.WorldId,
e.ChannelId)`) is now two: a world-scoped gate (`sc.IsWorld(t, e.WorldId)`)
that every channel handler in the event's world passes, and the original
channel-scoped `sc.Is(...)` check retained only to decide whether *this*
handler is the sender's own channel (and therefore owns the one-time
`WhisperSendResult` confirmation). The decision is factored into a small pure
function, `whisperDeliveryPlan`, so the cross-channel gating logic is
unit-testable without a live session-registry/socket fixture.

Recipient delivery (`WhisperReceive`) now runs unconditionally on every
channel handler in the world; `session.NewProcessor(...).GetByCharacterId(sc.Channel())`
resolves on exactly the one channel holding the recipient's session, so
delivery is a no-op everywhere else. The `GET /api/characters/{id}` REST
fan-out is also fixed: the recipient's character is fetched only inside the
sender-channel branch (needed for `WhisperSendResult`'s target name), and the
sender's character is fetched only after the recipient's session is
confirmed present on this channel (needed for `WhisperReceive`'s from-name
and gm flag) — so a whisper in an N-channel world no longer issues 2N
REST calls, only the (at most) 2 that correspond to real deliveries.

`handleMultiChat`, `handleMessengerChat`, and `handlePinkChat` share the
identical channel-scoping defect and were deliberately left untouched per
the brief's scope — see "Not yet answered" above for the follow-up.

Verified: `go build ./... && go test ./...` from
`services/atlas-channel/atlas.com/channel` (full module suite, all packages
passing). A new table test,
`services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer_test.go`,
reproduces the exact bug scenario (recipient on a different channel than the
sender) against the pre-fix code — confirmed to fail there and pass against
the fix — plus three companion cases (sender-channel-only delivery,
same-channel double delivery, and a different-world handler emitting
nothing).

**Live re-test against the reported reproduction (tenant
`625de849-e34f-45c8-95e6-b8e794774422`, `atlas-pr-1407`, `Atlas`→`Chronicle`
across channels 0/1) is still pending** — this fix has only been verified by
module-local unit tests, not against a live ephemeral environment.
