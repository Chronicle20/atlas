# TOUCH_MONSTER_ATTACK derivation (task-092 Stage 4 — Item 1)

> **OUTCOME (2026-06-14): TOUCH_MONSTER_ATTACK is ALREADY IMPLEMENTED — do not add a
> standalone codec.** It is one of FOUR serverbound attack ops decoded by the shared
> `libs/atlas-packet/model/AttackInfo` codec and routed in production:
> CLOSE_RANGE_ATTACK (0x2C, TryDoingMeleeAttack), RANGED_ATTACK (0x2D,
> TryDoingShootAttack), MAGIC_ATTACK (0x2E, TryDoingMagicAttack), TOUCH_MONSTER_ATTACK
> (0x2F, TryDoingBodyAttack) — each wired via CharacterMelee/Ranged/Magic/TouchAttack
> handlers that call `AttackInfo.Decode` + `processAttack`. **All four show ❌ in the
> matrix for the SAME reason**: `AttackInfo` carries no `packet-audit:verify` markers
> and the registry op-names don't link to it. This is a family-wide *verification*
> gap on a shared codec, NOT a missing codec — verifying it (link `AttackInfo` to all
> four registry ops with markers/evidence/reports) is its own follow-up task, broader
> than task-092's MOB scope. A standalone touch codec was prototyped during this
> stage and REMOVED as a duplicate (AttackInfo is more correct — e.g. it gates the
> dr-block at GMS major >= 84, handles keyDown/mask/special-skill fields, etc.).
> The derivation below is retained as REFERENCE for that future verification task.

Serverbound `CUserLocal::TryDoingBodyAttack` — the body-attack packet the client
sends when the player's avatar touches/melees a mob. IDA-verified across all four
GMS/JMS clients. (Reference only — the live codec is `model.AttackInfo`.)

> **Correction (owner, 2026-06-14):** an earlier pass decompiled a function it
> believed was v83 `TryDoingBodyAttack` and recorded a flat `updateTime + two
> branches` layout sending `0x30` — that function was actually `SetDamaged`. The
> v83 IDB was renamed; the REAL `TryDoingBodyAttack` is @0x95f135 and sends `0x2F`.
> The layout below is the corrected, verified one.

## Opcodes (owner-confirmed, IDA-verified at each COutPacket ctor)
| version | port | function | opcode |
|---|---|---|---|
| gms_v83 | 13342 | @0x95f135 | **0x2F** (47) |
| gms_v87 | 13341 | @0x9e17dc | **0x31** (49) |
| gms_v95 | 13340 | @0x930710 | **0x32** (50) |
| jms_v185 | 13343 (clean DEVM build) | @0xa2ac53 | **0x26** (38) |

(The obfuscated jms retail dump @13339 is SMC-protected and unusable; the DEVM
build @13343 decompiles cleanly.)

## Unified layout
All four share ONE structure with version-gated sections. Field order:

```
u8    fieldKey                       // *(get_field()+0x134 v83 / +0x148 v87+)
== crypto block (v87/v95/jms only — major >= 87) ==
u32   ~dr0                            // _DR_INFO masked, inverted
u32   ~dr1
u8    countByte                       // high nibble = mobCount, low nibble = hitsPerMob
                                      //   (v83: countByte = mobCount<<4 | 1)
== crypto block cont. (v87/v95/jms) ==
u32   ~dr2
u32   ~dr3
[jms only, runtime-flag-gated] sub_AAA158 info appender   // see "jms extras"
u32   skillId
== crypto block cont. (v87/v95/jms) ==
u32   rand                            // Random() % dr0 (or raw Random() if dr0==0)
u32   crc32                           // CCrc32::GetCrc32(drInfo,4,rand)
u32   skillLevelCrc1                  // SKILLLEVELDATA::GetCrc, 0 if no skill
u32   skillLevelCrc2                  // GMS ONLY (v83/v87/v95). JMS emits only crc1.
u8    0                               // filler
u16   action                          // action&0x7FFF | (left<<15)
u8    attackActionType
u8    combatOrders                    // 0 in v83
u32   attackTime                      // get_update_time()
u32   dwId                            // v95/jms ONLY (major >= 95). v83/v87 omit.
== per-mob loop, mobCount iterations ==
  u32  mobId
  u8   hitAction
  u8   foreAction&0x7F | (isLeft<<7)
  u8   frameIdx
  u8   calcDamageStat&0x7F | (templateChanged<<7)
  u16  mobX
  u16  mobY
  u16  mob2X
  u16  mob2Y
  u16  delay
  u32 × hitsPerMob   damage[]          // v83: hitsPerMob==1 → exactly one damage
  u32  mobCrc
u16   playerX
u16   playerY
[jms only, runtime-flag-gated] trailing per-mob block      // see "jms extras"
```

### Version flags (how the codec branches)
- `hasCrypto` = `MajorVersion() >= 87` → emits dr0..dr3, rand, crc32. v83/v84 omit
  them entirely (the old flat shape is exactly the layout minus these fields).
- `hasDwId` = `MajorVersion() >= 95` → emits the `dwId` field (v95 + jms[185]).
- `singleSkillCrc` = region `JMS` → emits one skill-level CRC; GMS emits two.

These three flags reproduce every observed v83/v87/v95/jms divergence (verified by
two independent decompile passes: v87 has 2 CRCs + no dwId; v95 has 2 CRCs + dwId;
jms has 1 CRC + dwId).

### jms extras — runtime-flag-gated, MODELED OFF (documented limitation)
jms wraps two extra wire blocks in `if (sub_4AE4F1(1))`:
1. an info appender (`sub_AAA158`) between `~dr3` and `skillId`, and
2. a trailing per-mob block after `playerY` (count byte + per-mob optional sub-records).

`sub_4AE4F1(1)` is a **runtime hash-map config lookup** (key=1) — `sub_4AE542` is an
lrotr-hashed bucket walk over a global map populated from server-pushed config, NOT
a compile-time constant. So whether these blocks are on the wire is determined at
runtime, not statically. The codec models the **flag-off (no-extras)** canonical
case (and its golden test pins that). If a jms tenant runs with the feature on, the
trailing blocks would need an explicit mode — tracked as a follow-up; the base
layout (everything above) is byte-exact for the common case.

## Decode note
The packet has no separate body-attack/no-mob discriminator on the wire (the old
mis-derived "two branches" came from SetDamaged). `mobCount`/`hitsPerMob` come from
`countByte`, which fully drives the loops — so Decode is a straight mirror of Encode
keyed on the three version flags. The crypto/CRC/dr fields are opaque u32s to the
server (it reads the widths; it does not recompute them) — correct for a decode+log
handler.
