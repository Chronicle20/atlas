# Bug: meso award notified in chat, and party-owned drops are unpickable for 15s

Task: task-248-party-meso-split · PR #1446 · env `atlas-pr-1446`
Tenant: GMS v83.1 (`region":"GMS","majorVersion":83,"minorVersion":1`, from the
`atlas-drops` tenant config message in the PR-1446 log).

Two distinct defects, reported together from live testing of PR #1446.

---

## Bug 1 — meso award renders as a chat line, not the drop-pickup notification

### Reproduced

Live, in `atlas-pr-1446`. Characters 1 and 2 in party `1000000000`, same map.
`atlas-drops` log:

```
"Splitting [5000] meso from drop [1000001] among [2] recipient(s)."
"Awarding [2500] meso from drop [1000001] to character [1]."
"Awarding [2500] meso from drop [1000001] to character [2]."
```

### Observed

Each recipient's share is announced as a chat line. Path:
`AwardPickedUpMeso` emits `MESO_CHANGED` with `ShowEffect: true`
(`services/atlas-character/atlas.com/character/character/processor.go:~955`)
→ `handleStatusEventMesoChanged`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:452`)
→ `CharacterStatusMessageOperationIncreaseMesoBody`.

Separately, the picker also receives
`CharacterStatusMessageOperationDropPickUpMesoBody(false, e.Body.Meso, 0)` from
`handleStatusEventPickedUp`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:182`),
where `e.Body.Meso` is the **full drop amount** (`pickedUpEventStatusProvider`
sets `Meso: d.Meso()`, `services/atlas-drops/atlas.com/drops/drop/producer.go:93`).
So the picker sees "+5000" from the pickup message and a chat line for 2500,
and non-pickers see only the chat line.

### Expected

Every recipient sees their own share via the drop-pickup meso mechanism
(`CharacterStatusMessage` DropPickUpMeso), and no chat line for a drop-sourced
meso award. The picker's pickup message must show their share (2500), not the
full drop (5000).

### Root cause

The award reuses the generic `MESO_CHANGED` → `IncreaseMeso` chat path, and the
existing `PICKED_UP` → `DropPickUpMeso` message was never made share-aware. The
per-recipient `MESO_AWARDED` event exists but no channel consumer renders it.

### Fix

- `services/atlas-character/atlas.com/character/character/processor.go` —
  `AwardPickedUpMeso`: emit `MESO_CHANGED` with `showEffect: false` (last arg of
  `mesoChangedStatusEventProvider`). The event still drives stat/saga consumers;
  the channel handler already treats `!ShowEffect` as "no chat line". Keep the
  `statChanged` emit unchanged — it is what unlocks the client's exclusive
  request.
- `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go` —
  add `StatusEventTypeMesoAwarded = "MESO_AWARDED"` and
  `MesoAwardedStatusEventBody{CharacterId uint32, Amount uint32, Picker bool}`,
  mirroring `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go`
  (`StatusEventMesoAwardedBody`) and the copy already in
  `services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go` —
  register a `handleStatusEventMesoAwarded` handler in `InitHandlers` alongside
  the four existing ones; it announces
  `charpkt.CharacterStatusMessageOperationDropPickUpMesoBody(false, e.Body.Amount, 0)`
  to `e.Body.CharacterId` via `session.NewProcessor(...).IfPresentByCharacterId(sc.Channel())`
  after the usual `sc.Is(tenant, e.WorldId, e.ChannelId)` guard. Skip when
  `Amount == 0` (a zero share is emitted only to complete the picker's pickup).
- Same file, `handleStatusEventPickedUp` — drop the `e.Body.Meso > 0` branch
  that writes the full-amount `DropPickUpMeso`; `MESO_AWARDED` now owns the meso
  notification and carries the correct per-recipient amount. The item branches,
  the monster-book card branch, and the `DropDestroy` announce are unchanged.
  Every meso pickup (including pet pickup, `petSlot >= 0`) routes through
  `ReserveAndEmit`, which always emits `MESO_AWARDED` for the picker — verify
  this before removing the branch.

---

## Bug 2 — a party-owned monster drop is unpickable by anyone for 15 seconds

### Reproduced

Live, in `atlas-pr-1446`. Character 1 (in party `1000000000`) killed monsters;
`atlas-drops` received:

```
"type":"SPAWN","body":{"itemId":0,...,"mesos":471,"dropType":0,
 "ownerId":1,"ownerPartyId":1000000000,...}
```

`atlas-channel` shows character 1 picking up only drop `1000033`; drops
`1000023`, `1000032`, `1000034` were picked up by character 2. Character 1's
client sent **no** pickup request for those (no `DropPickUpHandle` log line for
character 1 on those drop ids), i.e. the refusal is client-side, not a server
reservation failure (`atlas-drops` logged no "Failed reserving").

### Observed

`atlas-channel` announces `DROP_ENTER_FIELD` with
`owner = d.Owner()` — which returns the **party id** when `ownerPartyId != 0`
(`services/atlas-channel/atlas.com/channel/drop/model.go:89`) — paired with
`dropType = d.Type()`, which is `0` for every monster drop because
`atlas-monster-death` hard-codes it (`dropType := byte(0)  // TODO determine
type of drop`, `services/atlas-monster-death/atlas.com/monster/monster/processor.go:21-22`).

Client evidence (GMS v83 `MapleStory_dump.exe.i64`, session `754107bf`):

`CDropPool::OnDropEnterField` @0x505900 field map (DWORD indices as used in
`TryPickUpDrop`): `+0x20`=dropId, `+0x24`=**dwOwner**, `+0x28`=dropperId,
`+0x2C`=**nOwnType**, `+0x30`=isMeso, `+0x34`=amount/itemId, `+0x40`=drop time.

`CDropPool::TryPickUpDrop` @0x50463c gate before `SendDropPickUpRequest`:

```c
if ( (v23 - v12[16] >= 15000            // >=15s since the drop landed
   || !v12[10]                          // dropperId == 0 (player drop)
   || ((v13 = v12[11]) != 0 || v12[9] == <my character id>)   // ownType 0 -> owner must be me
     && (v13 != 1 || v12[9] == <my party id>))                // ownType 1 -> owner must be my party
  && ... )
```

With `ownType = 0` and `owner = 1000000000` (a party id), the comparison is
against the character id and fails for **every** player, so no client sends a
pickup request until the 15-second ownership window expires. Whoever happens to
still be standing on the drop after 15s picks it up — which is what the reporter
saw as "only my party member could pick it up". Note `ownType >= 2` bypasses the
owner check entirely, so FFA/explosive drop types are unaffected.

### Expected

The killer (and their party) can pick the drop up immediately. `owner` and
`ownType` must agree: `ownType = 1` when `owner` carries a party id, `0` when it
carries a character id.

### Root cause

`Model.Owner()` substitutes the party id for the character id, but nothing
substitutes the matching `ownType`. Pre-existing defect (not introduced by
task-248); the party-meso feature is what made party-owned drops routine enough
to notice.

### Fix

- `services/atlas-channel/atlas.com/channel/drop/model.go` — add an
  `OwnType() byte` accessor that keeps the pairing decision next to `Owner()`:
  return `m.dropType` when it is `>= 2` (FFA/explosive — the client skips the
  owner check), else `1` when `m.ownerPartyId != 0`, else `0`. Document the
  v83 `TryPickUpDrop` @0x50463c evidence in a comment.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:108`
  and `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:733`
  — pass `d.OwnType()` instead of `d.Type()` into `droppkt.NewDropSpawn`. These
  are the only two `NewDropSpawn` call sites.
- Unit-test `OwnType()` over the four cases (character-owned, party-owned,
  type 2, type 3) and assert the announced byte in whichever drop-spawn test
  already covers the consumer.

No server-side reservation change is needed: `Model.CanBeReservedBy`
(`services/atlas-drops/atlas.com/drops/drop/model.go:98`) already admits both the
owner and any party member.

---

## Not yet answered

- Whether `DropPickUpMeso`'s `partial` flag should be set to `true` for a split
  share. The client semantics of that byte are unverified; the fix keeps
  `false`. Do not change it without IDB evidence.
- `atlas-monster-death`'s `dropType` TODO (FFA / explosive-reward monsters) is
  left in place; the channel-side pairing fix makes the current hard-coded `0`
  correct for both owner cases, but real FFA loot still never reaches the client
  as type 2.
- A picker whose own share is 0 (drop meso < party size) gets a `MESO_AWARDED`
  with `Amount: 0` and no `STAT_CHANGED` (the credit transaction is skipped when
  `meso == 0`). Whether that leaves the client's exclusive-request lock hanging
  was not tested.

## Resolution

- Fix commits: `35462cc0b` (both bugs) and `19f19f229` (review fix-up: guard the
  meso-only pickup from a spurious `DropPickUpStackableItem(0, 0)` packet).
- Review: `reviews/review-bug-meso-notify-and-ownership.md` — CHANGES_REQUIRED,
  one blocking finding, fixed by `19f19f229`.
- Gate: `tools/verify.sh --quick --base 3b055a922` exit 0 after `19f19f229`
  (12 changed paths, 2 Go modules, all checks passed).
- Live re-test: pending — re-test in the PR-1446 environment after the rollout
  picks up these commits. Confirm (a) each party member sees their own share as
  a meso pickup notification and no chat line, and (b) the killer can pick a
  party-owned drop up immediately rather than after 15 seconds.
