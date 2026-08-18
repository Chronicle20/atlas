# Task 30 review: atlas-buddies consumes `NAME_CHANGED`

Commit reviewed: `4586d78b2` (5 files, +319/-0).

## Verdict: PASS with one non-blocking finding

## 1. Emit correctness end to end — PASS

`list/processor.go:723-724`:

```go
_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyUpdatedStatusEventProvider(character2.Id(u.OwnerId), worldId, character2.Id(characterId), u.Group, name, u.ChannelId, u.InShop))
```

`BuddyUpdatedStatusEventProvider` signature (`kafka/producer/list/producer.go:57`):
`(characterId character.Id, worldId world.Id, buddyId character.Id, group string, buddyName string, channelId int8, inShop bool)`.
The producer body (`producer.go:57-71`) puts `characterId` into the envelope
(`StatusEvent.CharacterId`, i.e. the Kafka key / recipient) and `buddyId` into
`Body.CharacterId` (the buddy being described).

Verified against the sibling call at `processor.go:693` (`UpdateBuddyShopStatus`):
arg1 there is `b.CharacterId` — the **owner** whose list contains the updated
row — and arg3 (`buddyId`) is `tbe.CharacterId`, which by construction equals
the subject who changed. Same shape here: arg1 is `u.OwnerId` (the owner
whose buddy-list row was renamed, resolved in `administrator.go` via
`le.CharacterId` after loading the row's `ListId`), arg3 is `characterId`
(the renamed character). `group`/`channelId`/`inShop` come straight off the
row that was renamed (`u.Group`/`u.ChannelId`/`u.InShop`), which is more
direct than the shop-status sibling's double-list-lookup and carries no
ambiguity.

Cross-checked against the live consumer,
`services/atlas-channel/atlas.com/channel/kafka/consumer/buddylist/consumer.go:137-147`:
`session.NewProcessor(...).IfPresentByCharacterId(sc.Channel())(uint32(c.CharacterId), ...)`
resolves the session to announce to from the **envelope** `CharacterId`
(confirms it must be the owner/recipient), and `updateBuddy(...)` is called
with `c.Body.CharacterId` (confirms the body must describe the buddy). The
argument order in this commit matches both ends of the contract. Getting
owner/buddy swapped would compile fine and push to the wrong session — it
does not do that.

## 2. Idempotency — structurally PASS, but the test doesn't prove the "no second emit" half — NON-BLOCKING FINDING

`administrator.go:145-148`:

```go
for _, row := range rows {
    if row.CharacterName == name {
        continue
    }
    ...
```

Rows already at the target name are skipped — not `Save`d, not added to the
returned `[]buddyNameUpdate` — and `processor.go`'s `UpdateBuddyName` only
`mb.Put`s for entries actually present in that slice. So the gating is real
and structural, matching the brief's `update bool`-equivalent requirement.

However, `TestNameChangedIsIdempotent` (`consumer_test.go:176-190`) only
asserts the final DB value:

```go
handleStatusEventNameChanged(db)(l, ctx, ev)
handleStatusEventNameChanged(db)(l, ctx, ev)

require.Equal(t, "Zulu", buddyName(t, db, ten, 10, 1))
```

It never inspects the outbox table (`outbox.Migration(db)` is set up in
`setupTestDatabase`, so the table exists and is queryable) to confirm a
second `BUDDY_UPDATED` message was **not** enqueued on redelivery. A
regression that dropped the `row.CharacterName == name` skip (e.g. rewritten
to always update, or an emit moved outside the gate) would still pass this
test — the DB value converges to "Zulu" either way; only a message count
would catch it. This matches the concern flagged in the review brief
verbatim: the idempotency claim as tested is weaker than the report states.
Recommend adding an outbox-row-count (or emitted-message) assertion in a
follow-up commit, not blocking this one.

## 3. Producer/consumer contract — PASS

Consumer side, `kafka/message/character/kafka.go:16,53-59`:

```go
StatusEventTypeNameChanged    = "NAME_CHANGED"
...
type NameChangedStatusEventBody struct {
    OldName string `json:"oldName"`
    NewName string `json:"newName"`
}
```

Producer side, `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:235,359-362`:

```go
StatusEventTypeNameChanged       = "NAME_CHANGED"
...
type StatusEventNameChangedBody struct {
    OldName string `json:"oldName"`
    NewName string `json:"newName"`
}
```

Literal type string, field count, field types, and json tags all match. Only
the Go type name differs (`NameChangedStatusEventBody` vs
`StatusEventNameChangedBody`), which is explicitly fine — local convention,
no compile-time link, JSON marshaling unaffected. Confirmed `NameChangedStatusEventBody`
follows this file's own naming convention (`CreatedStatusEventBody`,
`DeletedStatusEventBody`, `LoginStatusEventBody`, `LogoutStatusEventBody`,
`ChannelChangedStatusEventBody`).

## 4. Handler registration — PASS

`kafka/consumer/character/consumer.go:47-49` registers
`handleStatusEventNameChanged(db)` inside the existing `InitHandlers` body,
beside the other five handlers, same `rf(t, message.AdaptHandler(message.PersistentConfig(...)))`
shape. The handler itself (`consumer.go:118-127`) has the type guard
(`if event.Type != character.StatusEventTypeNameChanged { return }`).
`main.go` is untouched — confirmed via `git log -1 -- .../main.go` showing
the last touching commit is an unrelated pre-existing one
(`e0321f319`, lint/format baseline), not this commit.

## 5. Wrong-column test — PASS

`consumer_test.go:158-172` (`TestNameChangedUpdatesEveryOwnersCopyOfTheBuddyName`)
seeds owner 10 holding character 1 as a buddy ("Yankee") and, separately,
character 1's own list holding character 99 as a buddy ("Whoever") — exactly
the brief's controller-addenda replacement scenario (schema-seedable, unlike
the plan's original two-owners-same-buddy draft, which is blocked by
`buddy.Entity`'s `character_id` being the sole PK). After renaming character
1 to "Zulu", it asserts **both**:

```go
require.Equal(t, "Zulu", buddyName(t, db, ten, 10, 1))
require.Equal(t, "Whoever", buddyName(t, db, ten, 1, 99))
```

An implementation that matched on the owner (`characterId`) instead of the
buddy (`targetId`) would instead rewrite character 1's own list entry for
buddy 99, failing the second assertion while still passing a same-owner
DB probe. This is a real regression-catching test, not just a happy-path
check.

## 6. Pre-settled items — confirmed, not re-raised

- No literal `tenant_id` predicate in `updateBuddyName`'s query
  (`administrator.go:139`, `db.Where("character_id = ?", targetId).Find(&rows)`)
  and in the follow-up `db.Where("id = ?", row.ListId).First(&le)` — both run
  against `tx`, which is derived from `p.db.WithContext(p.ctx)` inside
  `database.ExecuteTransaction`, so the tenant context is present and the
  automatic scope applies. Not flagged.
- `buddy.Entity`'s sole PK being `character_id` (`buddy/entity.go:24`) is
  unchanged by this commit — confirmed via `git show 4586d78b2 -- buddy/entity.go`
  showing no diff to that file. Documented inline in `administrator.go`'s new
  doc comment and in the task report; not re-filed as a new finding.

## 7. DOM-* conformance — PASS

- **Layering**: consumer (`kafka/consumer/character/consumer.go`) → processor
  (`list.NewProcessor(...).UpdateBuddyNameAndEmit`) → administrator
  (`updateBuddyName`, package-private). No layer skipped or reached into
  directly.
- **Consistency with existing admin functions**: `updateBuddyName`'s
  mutate-then-`db.Save` shape matches `updateBuddyChannel` (`administrator.go:66-88`)
  and `updateBuddyShopStatus` (`administrator.go:90-112`) — same idiom used
  throughout this file for GORM entity updates, so no new pattern introduced.
- **No test-only constructors / `*_testhelpers.go`**: `consumer_test.go` is
  the only new test file; its helpers (`setupTestDatabase`, `seedBuddyList`,
  `buddyName`, `nameChangedEvent`) live directly in the `_test.go` file, per
  convention — no separate testhelpers file.
- **No reinvented constants**: no new domain/item/job/etc. constant defined;
  `StatusEventTypeNameChanged` is a local Kafka event-type string mirroring
  the producer's, not a candidate for `libs/atlas-constants`.
- **Immutability**: `buddy.Entity` is a GORM entity, mutated via `db.Save`
  consistent with the rest of this file (GORM entities are not part of the
  immutable-model layer in this codebase's convention); `buddyNameUpdate` is
  a plain value struct returned by value, not mutated after construction.

## Summary

Blocking findings: none.
Non-blocking findings: 1 — `TestNameChangedIsIdempotent` proves the DB
converges correctly on redelivery but does not assert on the outbox/emitted
message count, so it would not catch a regression that broke the "no second
emit" half of the idempotency claim even though the underlying gating
(`row.CharacterName == name` skip) is genuinely structural today.
