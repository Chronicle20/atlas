# Follow-up: `clear_skill` removes the skill but never tells the client

**Raised during:** task-290 Plan C, Task C4 (commit `4278bded9`), by the controller
while adjudicating the implementer's DONE_WITH_CONCERNS.

## What Cosmic does

`iceCave` calls `teachSkill(id, -1, 0, -1)` five times. Per plan-c §0.1, that path
reaches `Character.changeSkillLevel(skill, -1, 0, -1)`, whose `newLevel > -1` branch is
false, so Cosmic does three things:

1. `skills.remove(skill)`
2. `DELETE FROM skills WHERE skillid = ? AND characterid = ?`
3. **sends `updateSkill(id, -1, 0, -1)` to the client**

## What Atlas does after Task C4

Items 1 and 2 are covered. `clear_skill` -> `skill.Processor.RequestDeleteSkill` ->
`REQUEST_DELETE` -> atlas-skills `DeleteForSagaCompensationAndEmit`, which always emits
the skill `DELETED` status event.

Item 3 is **not** covered. Traced by hand:
`services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go:233-245`
(`handleSnapshotSkillDeleted`) discards its `writer.Producer` argument (`_ writer.Producer`)
and only calls `snapshot.GetRegistry().RemoveSkill(...)`. Its own doc comment records why:

> handleSnapshotSkillDeleted is atlas-channel's first consumer of the skill DELETED event
> (saga compensation path; the packet layer never needed it — event-coverage.md §3).

That premise was true before Task C4. Compensation undoes a grant the client never saw, so
no notification was needed. `clear_skill` is the first **user-visible** deletion, which
makes the seam's assumption stale: the skill vanishes server-side and from the channel
snapshot, but the client's skill window keeps showing it until the next relog.

## Why it was not fixed inside Task C4

The opcode is not missing. `CharacterSkillChange`
(`libs/atlas-packet/character/clientbound/skill_change.go`) exists, is registered in
`services/atlas-channel/atlas.com/channel/main.go:793`, and is already announced from the
*same consumer file* on the create/update path (`consumer.go:128`).

The blocker is the encoding of a removal, not the packet's existence. Cosmic sends
level `-1`; Atlas's constructor is
`NewCharacterSkillChange(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool)`
and the writer encodes level as an unsigned 4-byte field. `-1` is not expressible through
a `byte` parameter, so emitting the removal means either widening that signature (which
touches every existing caller) or adding a removal-specific constructor — and either way
it requires deriving what `CWvsContext::OnChangeSkillRecordResult` actually does with a
negative level, against the client binary.

That is packet work under `docs/packets/PROCESS.md`. plan-c's Global Constraints fence it
off explicitly: "Any task that thinks it needs a new opcode has hit an unexpected gap: stop
and report rather than deriving one."

## Recommended disposition

A separate packet task: derive the negative-level read order in
`CWvsContext::OnChangeSkillRecordResult` from the IDB, decide between a widened signature
and a removal constructor, then announce from `handleSnapshotSkillDeleted` and update the
`event-coverage.md` §3 note that currently says the packet layer never needed this event.

**Impact if left as-is:** an `iceCave` entrant has the five Aran tutorial skills removed
server-side and from the channel snapshot — every server-authoritative check is correct —
but their client UI still lists the skills until they relog.
