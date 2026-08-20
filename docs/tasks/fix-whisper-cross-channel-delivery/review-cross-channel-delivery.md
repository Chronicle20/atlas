# Review: fix-whisper-cross-channel-delivery (748510af2..489bb3327)

Commits reviewed:
- `30b459905` fix(atlas-channel): deliver whisper to recipient regardless of channel
- `489bb3327` fix(atlas-channel): deliver party/guild/buddy/alliance, messenger, and pink-text chat regardless of channel

Requirements: `docs/tasks/fix-whisper-cross-channel-delivery/bug-whisper-cross-channel-delivery.md`,
`docs/tasks/fix-whisper-cross-channel-delivery/bug-cross-channel-chat-siblings.md`

Files touched: `services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go`,
`.../consumer_test.go` (new), the two bug docs.

## Seam traced: chat_event fan-out topology

The core question is whether "world-scoped gate + per-channel session-registry lookup" actually
delivers exactly once, given handlers are registered per `(tenant, world, channel)` socket
listener (`main.go` `buildListener`, called once per key by the projection apply loop).

Traced the Kafka wiring by hand rather than taking the brief's "exactly-once follows from at
most one session per world" on faith:

- `main.go:182` — `consumerGroupId := consumergroup.Resolve(consumerGroupIdTemplate,
  serviceId.String())`, and `serviceId := uuid.MustParse(os.Getenv("SERVICE_ID"))`. Consumer
  group ID is per-process (keyed on `SERVICE_ID`), not a fixed literal shared by the whole
  fleet.
- `main.go:227` — `message.InitConsumers(l)(cmf)(consumerGroupId)` is called **once** at
  process startup — one Kafka reader per pod for `chat_event`, using that pod's own consumer
  group ID.
- `main.go:480` (`buildListener`) — `message.InitHandlers(fl)(sc)(wp)(rh)` is called once per
  `(tenant, world, channel)` key that this pod's projection state currently holds, each
  registering its own `handleXXXChat` closures against the **same** per-pod consumer via
  `consumer.Manager.RegisterHandler` (`libs/atlas-kafka/consumer/manager.go:203`) — i.e.
  handlers fan out in-process from one reader, not one reader per handler.
- `session/registry.go` — the session registry is a per-process, in-memory singleton
  (`sync.Once`), keyed `tenantId -> sessionId -> Model`. `session.ProcessorImpl.GetByCharacterId`
  / `IfPresentByCharacterId` (`session/processor.go:175-191, 235-239`) filter on
  `(characterId, WorldIdFilter(ch.WorldId()), ChannelIdFilter(ch.Id()))`, i.e. the lookup is
  scoped to *this handler's own* channel (`sc.Channel()`), not the whole registry.

Because each pod has its own consumer group ID (per-pod `SERVICE_ID`), a pod does not share
Kafka partitions with any other pod for this topic — every pod that has a live `chat_event`
consumer sees **every** `chat_event` message (broadcast per pod), and within a pod every
registered channel handler evaluates that message. Delivery to a specific character therefore
resolves to exactly the one `(pod, channel)` combination whose in-memory registry actually
holds that character's session — which is unique as long as (a) a character has at most one
live session in a world (enforced by login flow, not part of this diff) and (b) a given
`(tenant, world, channel)` key is hosted by exactly one live process at a time (enforced by the
projection apply loop's drain-then-add sequencing in `configuration/projection/apply.go`,
also not part of this diff). Both are pre-existing invariants this fix depends on but does not
alter — checked, not merely assumed.

**Current deployment topology note (non-blocking):** `deploy/k8s/base/atlas-channel.yaml` sets
`replicas: 1` and a **hard-coded literal** `SERVICE_ID` (`value:
e7fb1d7e-47b8-46bd-97dc-867d93530000`), not a per-pod-unique value (e.g. `metadata.uid`). With
today's topology there is exactly one atlas-channel process, so the multi-pod question is moot
in production right now. If this Deployment is ever scaled to `replicas > 1` without first
making `SERVICE_ID` unique per pod, every pod would join the *same* Kafka consumer group and
the topic would be partitioned across them — at which point a chat_event could land on a pod
that hosts none of the relevant channels, breaking delivery again (and, more broadly, breaking
every other Kafka-consumed, session-routed feature in this file, not just chat — the session
registry itself is entirely pod-local and has no cross-pod affinity or replication).  This is a
pre-existing scaling constraint of `atlas-channel`, not something this diff introduces or
regresses, and it is out of scope for a two-commit chat-routing fix — flagging as context, not
a blocking finding.

## Checklist

1. **Exactly-once delivery** — verified against the session registry (see above). `PASS`.
2. **Zero/duplicate handlers in multi-pod** — traced; holds under the current
   `replicas: 1` topology and under any topology where `SERVICE_ID` (and hence consumer group)
   is unique per pod. Pre-existing constraint noted above if that assumption is ever violated.
   `PASS` (with the topology caveat noted, non-blocking).
3. **WhisperSendResult emitted exactly once, sender's channel only** —
   `consumer.go:186-191` (`whisperDeliveryPlan`): `sendResult` is only `true` when
   `sc.Is(t, e.WorldId, e.ChannelId)`, i.e. this handler's own channel matches the sender's
   channel — true for exactly one channel handler fleet-wide. Confirmed by
   `TestHandleWhisperChat_SenderChannel_DeliversResultOnly` (sender receives, recipient on a
   different channel receives nothing) and `TestHandleWhisperChat_SameChannel_DeliversBothOnce`
   (both packets exactly once, `expectNoPacket` checked for a *second* write). `PASS`.
4. **REST fan-out is lazy** —
   - `handleWhisperChat` (`consumer.go:205-233`): recipient-character GET only inside
     `if sendResult` (one channel); sender-character GET only after
     `session.GetByCharacterId(sc.Channel())(e.Body.Recipient)` succeeds (one channel). At most
     2 GETs total across an N-channel world, not 2N.
   - `handleMultiChat` / `handlePinkChat` (`consumer.go:129-138, 288-298`): new helper
     `presentRecipients` does an in-memory session-registry filter (`GetByCharacterId`, no
     REST) *before* the sender-character GET, and returns early when no recipient is present on
     this channel — so the REST call only fires on channels that actually deliver.
   - `handleMessengerChat` does no character lookup at all (unchanged, correctly so — no REST
     call to make lazy).
   No path reintroduces a per-channel GET for every message. `PASS`.
5. **handleGeneralChat / handlePetChat left channel-gated — correct, not an oversight.**
   `handleGeneralChat` (`consumer.go:83-104`) fans out via `_map.ForSessionsInMap(sc.Field(...))`
   — `sc.Field` builds a field from *this handler's own* channel
   (`server/model.go:75-77`), and a MapleStory map instance is inherently per-channel, so
   world-scoping this handler would deliver the message into the wrong channel's copy of the
   map. `handlePetChat` (`consumer.go:258-276`) is gated on the event's own channel because
   `e.Body.OwnerId` is the character who triggered the pet action from that same channel by
   construction (`services/atlas-messages/.../producer.go:73` `petChatEventProvider` derives
   `e.ChannelId` from the acting player's own `field.Model`) — the owner cannot be on a
   different channel than the event that they themselves produced. `PASS`.
6. **New tests assert the NEW contract, not the old one.** Verified by mechanically reverting
   `consumer.go` to the pre-fix version (`git show 748510af2:...consumer.go`) and re-running:

   ```
   go test ./kafka/consumer/message/... -run TestHandle -v
   ```

   7 of 13 new tests fail against pre-fix code (timeout waiting for the expected packet):
   `TestHandleWhisperChat_RecipientChannel_DeliversReceiveOnly`,
   `TestHandleMultiChat_RecipientChannel_Delivers`,
   `TestHandleMultiChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`,
   `TestHandleMessengerChat_RecipientChannel_Delivers`,
   `TestHandleMessengerChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`,
   `TestHandlePinkChat_RecipientChannel_Delivers`,
   `TestHandlePinkChat_TwoRecipientsTwoChannels_EachDeliversOwnOnly`. All 13 pass against the
   fixed code (`go test ./kafka/consumer/message/... -v -count=1`, confirmed locally). File was
   restored to the fixed version afterward and `git diff --stat` on it shows no residual change.
   `PASS`.

## Other checks

- `server.Model.Is` / `IsWorld` (`server/model.go:50-68`): `Is` = `IsWorld` (tenant match +
  world match) + channel match, exactly as the brief specifies; `IsWorld` alone rejects a
  cross-tenant event via `t.Is(m.Tenant())`, so world-scoping does not leak across tenants.
  `PASS`.
- `session/registry_test_helper.go` (`ClearRegistryForTenant`) pre-dates this range (introduced
  in `51e08d17b`); the new tests reuse it rather than adding a new `*_testhelpers.go` file, per
  repo convention. `PASS`.
- No other service consumes the `chat_event` topic — `grep -rl "ChatEvent\["` across `services/`
  finds only `atlas-channel` (consumer) and `atlas-messages` (producer, unchanged by this diff).
  No cross-service seam beyond the one traced above. `PASS`.
- `go build ./...` and `go build ./... && go test ./...` at
  `services/atlas-channel/atlas.com/channel` both pass; `go vet
  ./kafka/consumer/message/...` clean.
- Scope match: both commits' diffs correspond exactly to what the two bug docs describe (whisper
  fix, then the three sibling handlers). `handleGeneralChat`/`handlePetChat` are untouched, as
  both docs require. No drift between the stated brief and the actual diff.

## Not evaluable

- Full `tools/verify.sh` (bake, `-race`, repo-wide gates) was not run by this review — reviewers
  run module-local build/test, not the full verification gate (that is a separate gate per
  `docs/review-protocol.md`).
- Live re-test against the ephemeral reproduction environment
  (`atlas-pr-1407`/tenant `625de849-e34f-45c8-95e6-b8e794774422`) was not performed by this
  review; the whisper bug doc itself already flags this as "still pending" at the branch level.

## Verdict rationale

No blocking defects found. The delivery-topology reasoning in the two bug docs (world-scoped
gate + per-channel session-registry lookup ⇒ exactly-once) was independently traced against the
Kafka consumer-group wiring and the session registry rather than taken on faith, and holds. Test
honesty was verified empirically (reverted the production file, confirmed 7/13 new tests fail
pre-fix). REST-laziness and the sender-confirmation exactly-once property were verified by
reading the code and cross-checked against passing tests. One non-blocking topology note is
recorded above for awareness if/when `atlas-channel` is ever scaled past `replicas: 1`.
