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
