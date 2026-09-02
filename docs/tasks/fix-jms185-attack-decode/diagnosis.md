# JMS 185.1 serverbound attack decode is 28 bytes misaligned

## Symptom

On the `atlas-main` k3s environment (tenant `abedf3b4-1d7c-4b3b-bc52-70f62ab09418`,
region `JMS`, version `185.1`) a melee attack disconnects the client. The channel
log shows:

    Character [40] attempting to attack with skill [671081756] which they do not own.

`processAttack` (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`)
treats an unowned skill as a cheat signal and calls `session.Destroy`, so the
client's socket is closed mid-attack — observed by the player as a crash.

`671081756` = `0x27FFE51C`, which is wire bytes `1c e5 ff 27` at offset 4 of the
captured packet. The skill id was read from the wrong offset: the decode is
misaligned, not the skill check.

## Root cause

Every damage-randomizer / anti-hack gate in `libs/atlas-packet/model/attack_info.go`
is written as `Region() == "GMS" && MajorVersion() >= N`, so the JMS 185.1 tenant
takes the *legacy pre-84 GMS* path. The JMS 185.1 client does write those fields.
Total delta on a melee attack: **28 bytes** in the head/trailer plus **4 bytes per
attacked mob** (`DamageInfo`'s per-mob CRC, gated `GMS >= 61`).

## Evidence — two live captures

Both captured from `atlas-channel` `[PKT IN ]` debug dumps, `op=0x0023`
(`CLOSE_RANGE_ATTACK`, opcode 35 in `docs/packets/registry/jms_v185.yaml`), same
tenant/version, 2026-09-02. The reader starts after the 2-byte opcode.

### Capture A — swing hitting nothing (len 53, session `898eb528`, 10:13:55.697Z)

    23 00 01 8f 1c e5 ff 27  0c 32 ff 01 ff ff ff ff
    f3 ff 62 81 00 00 00 00  5b a3 0a 00 3d ec 6c 8e
    00 00 00 00 00 11 00 01  05 86 8c fe 0d 00 00 00
    00 a3 ff d5 01

### Capture B — swing connecting with one mob (len 79, session `81e50c2e`, 10:24:21.503Z)

    23 00 00 27 0c 32 ff ff  ff ff ff 11 77 b6 e5 8c
    c3 3f 65 81 00 00 00 00  30 43 97 00 6a 92 48 13
    00 00 00 00 00 05 00 01  05 6d 1c 08 0e 00 00 00
    00 42 42 0f 00 07 81 01  00 35 01 9b 00 35 01 9b
    00 a5 01 02 00 00 00 66  1d a1 07 07 01 9b 00

### Derived layout (capture B, offsets from packet start)

| off | bytes | field |
|---|---|---|
| 02 | `00` | fieldKey |
| 03 | `27 0c 32 ff` | dr0 |
| 07 | `ff ff ff ff` | dr1 |
| 0b | `11` | numAttacked/damage mask → hits=1, mobs=1 |
| 0c | `77 b6 e5 8c` | dr2 |
| 10 | `c3 3f 65 81` | dr3 |
| 14 | `00 00 00 00` | skillId = 0 (plain swing) |
| 18 | `30 43 97 00` | randomDr |
| 1c | `6a 92 48 13` | crc32 |
| 20 | `00 00 00 00` | skillDataCrc — **one** CRC, not two |
| 24 | `00` | mask1 |
| 25 | `05 00` | mask2 → nAction 5, bLeft false |
| 27 | `01` | attackActionType |
| 28 | `05` | attackSpeed |
| 29 | `6d 1c 08 0e` | attackTime tick |
| 2d | `00 00 00 00` | melee trailing word (the GMS v95 "battle mage" slot) |
| 31 | `42 42 0f 00` | DamageInfo[0].monsterId = 1000002 |
| 35 | `07 81 01 00` | hitAction / forceAction / frameIdx / calcDamageStatIndex |
| 39 | `35 01` `9b 00` | hitPositionX = 309, hitPositionY = 155 |
| 3d | `35 01` `9b 00` | previousPositionX = 309, previousPositionY = 155 |
| 41 | `a5 01` | delay = 421 |
| 43 | `02 00 00 00` | damage[0] = 2 |
| 47 | `66 1d a1 07` | per-mob CRC — **present on JMS** |
| 4b | `07 01` `9b 00` | characterX = 263, characterY = 155 |

79 bytes, fully consumed, every field lands on a semantically sensible value
(skillId 0 for an unskilled swing, skillDataCrc 0 to match, nAction 5, attack
speed 5, a monotonic tick, a level-1 hit for 2 damage, and a caster Y equal to
the mob's Y). Capture A is independently consistent under the same layout
(53 bytes, 0 mobs, nAction 17, caster at −93,469).

## Why this could not be derived statically

The jms IDB available here is the retail SCY dump
(`E:\Programs\Nexon\IDBs_v9\JMS\v185\MapleStory_dump_SCY.exe.i64`); there is no
`*_U_DEVM` build for v185. `CUserLocal::TryDoingNormalAttack` @ `0xa122be`
decompiles, but its packet-encode tail is code-flow-virtualized: an instruction
scan of the function finds its last real call at `0xa16b74`
(`DR_check(_DR_INFO*, ulong*, HINSTANCE*)`) and no reachable code between there
and the function end at `0xa18dda`. This is the same limitation already recorded
for this op in `docs/packets/audits/jms_v185/CharacterAttackMeleeRequest.md`
(task-150 note), whose generated wire-diff table reads every client field as an
untyped `byte`.

The `DR_check` call is positive corroboration that this client computes the
damage-randomizer words the captures show; the field *order* comes from the
captures.

Per `VERIFYING_A_PACKET.md` ("Producible prerequisite vs genuine blocker"), an
SMC/control-flow-virtualized binary is a genuine blocker for decompile-derived
bytes. Live wire capture is the higher-ranked evidence available (CLAUDE.md:
"Repo source, WZ data, IDA, and live output outrank remembered MapleStory
knowledge") and is what the fixture is pinned to.

## Why the cell was already ✅

`status.json` had `CLOSE_RANGE_ATTACK / character/serverbound/CharacterAttackMeleeRequest`
at `verified` for `jms_v185`, carried by `TestAttackMeleeRequest`, which is a
`pt.RoundTrip` over the wrapper. A round-trip is symmetric: it passes for any
self-consistent layout, including a wrong one. That is the false positive this
change closes — the jms cell now carries a byte fixture pinned to observed wire.

## Scope of the fix

`libs/atlas-packet/model/attack_info.go` and `model/damage_info.go`:

1. primary dr-block (dr0/dr1, dr2/dr3, randomDr/crc32) — JMS included
2. skill-data CRC — JMS writes ONE, not two
3. melee trailing word after attackTime — JMS included
4. `DamageInfo` per-mob CRC — JMS included

## Not covered

- **JMS magic and ranged attacks.** They share the head, so items 1–2 apply, but
  the magic secondary dr-block, the magic trailing word, and the ranged
  bullet/exJablin fields have no jms capture. Left on their existing GMS-only
  gates and still unverified for jms; needs a live magic + ranged capture.
- **JMS movement decode.** The same session logged
  `Code [185] not configured for use in movement (element 1 of 164, reader at
  offset 9)` — a separate misalignment in the movement codec, not addressed here.

---

# Part 2 — JMS 185.1 MOVE_PLAYER decode is 28 bytes misaligned

## Symptom

Same session, same tenant:

    Code [93] not configured for use in movement (element 1 of 121, reader at
    offset 9). This is a decode misalignment, not an unknown fragment type.

`121` is not an element count; it is `0x79`, a byte of the anti-cheat header
being read as one.

## Root cause

`libs/atlas-packet/character/serverbound/move.go` gated the self-move anti-cheat
header — `dr0`, `dr1`, `dr2`, `dr3`, the move `crc`, `dwKey`, `crc32` — on
`IsRegion("GMS")`. For JMS the decoder therefore read a single `fieldKey` byte
and went straight into `CMovePath`, 28 bytes early.

The same shape as Part 1, and not a coincidence: this client wraps its
cheat-sensitive serverbound ops (attack, move) in the same damage-randomizer
header, and every gate for that header was written GMS-only.

## Evidence — nine live captures

All `MOVE_PLAYER` (`op=0x0020`, opcode 32) `[PKT IN ]` frames logged in a 90
minute window, 2026-09-02, same tenant. Wire lengths 90, 98, 108, 108, 116,
126, 134, 144, 188. Every one is consistent with a 29-byte header
(`dr0`,`dr1`,`fieldKey`,`dr2`,`dr3`,`crc`,`dwKey`,`crc32`) placing the
`CMovePath` blob at frame offset `0x1f`:

| wire len | origin x,y at 0x1f | count at 0x23 |
|---|---|---|
| 90 | −77, 215 | 2 |
| 98 | −71, 215 | 3 |
| 108 | −93, 440 | 3 |
| 116 | 55, 215 | 4 |
| 126 | −7, 153 | 4 |
| 134 | 119, 191 | 5 |
| 144 | 181, 198 | 5 |
| 188 | 231, 206 | 8 |

Element count rises monotonically with frame length; `dr0` (`e1ffffff`) and
`dr1` (`c3795d8d`) are constant across all nine, `dr2`/`dr3` are per-session
constants, and the `dwKey` slot carries a small counter. 9/9, not a spot-check.

## The earlier decompile conclusion was applied to the wrong function

`move.go` carried this comment:

> IDA JMS v185 CVecCtrlUser::EndUpdateActive@0xaaa076: encodes
> Encode1(detectFlag) then if active: Encode1(fieldKey)+Encode4(crc)+
> CMovePath::Flush — NO dr0/dr1/dr2/dr3/dwKey/crc32 fields. The || JMS clause on
> dr-field gates was incorrect; JMS uses GMS v83-style layout (no dr fields).

The decompile of `0xaaa076` is accurate — that function really does write
`bActive`, `fieldKey`, `crc`, `Flush`. It is simply not the sender behind any
observed frame:

- its tail is a 6-byte `bActive`/`fieldKey`/`crc` run, but the six bytes
  before the blob on the wire are `00 00` followed by a 4-byte word;
- its whole body is inside `if (bActive)`, so a frame with `0x00` at that
  offset would be 6 bytes long — yet two frames differing only in that byte
  (`0x01` vs `0x00`) are both 108 bytes.

The registry's primary fname for this op is `CUserLocal::OnKey`, which the
retail SCY dump does not decompile — the same virtualization that hides the
attack senders' encode tails. `EndUpdateActive` is a second, unused send path.
The lesson is the one CLAUDE.md already states: a decompile of *a* function is
not evidence about *the* function until the opcode-construction site is tied to
it.

## The 18-byte tail — modelled, and why it lives in the serverbound codec

An earlier pass of this write-up called the tail unmodelable. That was wrong;
it is fully decompile-derived. `CMovePath::Flush` @`0x70ba2c` delegates all
encoding to `CMovePath::Encode` @`0x70b6c4`, whose tail, after the element
loop, is:

    @0x70b8ec  Encode1(len(m_aKeyPadState))     entry COUNT, not a byte count
    @0x70b8f3  loop i += 2, Encode1(nType):
               nType = state[i] & 0xF, and for every i but the last
               nType |= state[i+1] << 4         two entries packed per byte
    @0x70b942  Encode2(m_rcMove.left)
    @0x70b950  Encode2(m_rcMove.top)
    @0x70b95e  Encode2(m_rcMove.right)
    @0x70b96c  Encode2(m_rcMove.bottom)

So the tail is `1 + ceil(count/2) + 8`. Every captured frame carries
`count = 0x11 = 17`, giving `1 + 9 + 8 = 18` — exactly the bytes that were
being left unread — and `m_rcMove` is the path bounding box, maintained as the
encoder walks the elements (@`0x70b81c`-`0x70b842`). It decodes to the real
bounds of each walk: the 90-byte frame yields left −77, top 215, right −71,
bottom 215, matching a move from x=−77 to x=−71 at y=215.

**It is serverbound-only, and that is why it must NOT go into
`model.Movement`.** That model is shared between the serverbound decode and the
clientbound encode. The client's read side, `CMovePath::Decode` @`0x70b3ce`,
consumes the keypad block and the rect only under `if (bPassive)`, and the
clientbound entry point `CUserRemote::OnMove` @`0xa443ee` calls
`CMovePath::OnMovePacket(..., iPacket, 0)` — `bPassive = 0`. Adding the fields
to `model.Movement` would make every clientbound movement broadcast emit 18
bytes the client never reads. They live on `character/serverbound.Move`
instead, behind `moveKeyPadTail`.

`TestCharacterMoveBytesJMS185` now requires all three frames to decode with
zero bytes remaining and to re-encode byte-identically to the full captured
body.

Gated to JMS because jms is the client whose `CMovePath::Encode` was read. The
tail is unconditional in that encoder, so the GMS senders very likely append it
too — and the existing GMS `Move` fixtures pin Atlas's own output rather than
captured client wire, so they would not have caught it. Each GMS version's
`CMovePath::Encode` must be read before extending the gate; do not assume it.

The same tail applies to the other serverbound movement ops (monster, pet,
summon, npc, dragon), which share `CMovePath::Encode` client-side. Only
`character/serverbound.Move` was changed here.

## Still open after both parts

JMS magic and ranged attacks. Their senders — `CUserLocal::TryDoingMagicAttack`
@`0xa1d280` and `CUserLocal::TryDoingShootAttack` @`0xa19266` — were checked in
the IDB this session and are virtualized exactly like melee: the last reachable
call in each is `DR_check` (@`0xa205e5` and @`0xa1c4a0`), with no encode tail.
They inherit the head fix from Part 1, but the magic secondary dr-block, the
magic trailing word, and the ranged bullet/`exJablin` fields cannot be settled
from the binary. They need one live magic capture and one live ranged capture,
after which they pin exactly like melee did.
