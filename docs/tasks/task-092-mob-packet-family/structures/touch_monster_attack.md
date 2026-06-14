# TOUCH_MONSTER_ATTACK derivation (task-092 Stage 4 — Item 1)

Serverbound `CUserLocal::TryDoingBodyAttack`. This is the most complex packet in
the MOB family: a heavily-conditional attack/hit packet, version-divergent. The
codec is **not landed** — this doc records the IDA-derived structure so a focused
follow-up can implement it without re-deriving. (A knowingly-incomplete codec is
worse than none; per CLAUDE.md the cell stays ❌ until a faithful codec exists.)

## Opcode correction (IDA-verified)
The client builds `COutPacket(0x30)` in v83 — the registry/STATUS opcode 0x2F/47
is an **off-by-one** (same class as the MOB_SPEAKING cluster). Real opcodes must
be read from each `COutPacket(...)` ctor, not the csv:
- **v83**: `0x30` (48) — verified @0x9581a9 (`COutPacket::COutPacket(&v186, 0x30)`).
- v95 / jms: read from their `TryDoingBodyAttack` ctor (not yet derived).

## v83 wire layout — CUserLocal::TryDoingBodyAttack @0x9581a9
Built only when `a11` (report flag) is set. Two top-level branches on whether a
mob target (`a6`) is present.

```
Encode4  updateTime            // get_update_time()
IF mob present (a6 != 0):
  Encode1  attackInfoIndex     // a7
  Encode1  action              // v170[0]  (= AttackInfo+32, the mob attack action)
  Encode4  damage              // v131 (sub_95FA8B(action, computedDamage))
  Encode4  mobTemplateId       // fused a6[98]+12
  Encode4  mobId               // fused (a6+95), a6[97]   (mob crc/object id)
  Encode1  dirFlag             // v175 ? v174[0] : (a8 < 0)
  Encode1  mpBurn              // v185[0]  (Energy/Combo MP-burn dmg, Power Guard etc.)
  Encode1  reflectKind         // !v177 ? 0 : (v171 + 1)   (power-guard / manaReflect)
  IF (v171 || mpBurn != 0):    // reflect/extra-damage sub-block
    Encode1  reflectActive     // (*v185 != 0 && a9 != 0)
    Encode4  mobId             // fused (a6+95), a6[97]  (repeated)
    Encode1  hitAction         // v184[0]  (GetRandomHitAction)
    Encode2  hitX              // v190[0]  (CMob::GetHitPoint .x)
    Encode2  hitY              // v190[2]  (.y)
    Encode2  bodyX             // avatar body-rect point .x
    Encode2  bodyY             // .y
ELSE (no mob — fall/obstacle damage):
  Encode1  kind                // (a10 == 0) - 3
  Encode1  0                   // literal 0
  Encode4  damage              // v131
  Encode2  a2                  // the __int16 arg (obstacle id / diridx)
Encode1  trailingFlag          // BYTE4(v165) — v178[0] (deadly-attack/guard byte)
SendPacket
```

Notes:
- `damage` (v131) is `sub_95FA8B(action, arg0)` where arg0 is the full client-side
  damage calc (skill/passive/PG/combo). For a faithful codec the field is just a
  u32 on the wire; the calc is client-only.
- The sub-block predicate is `(v171 || *v185)` where `v171` = power-guard active,
  `*v185` = reflected/extra damage. Model as an explicit bool the codec branches on.

## v95 / jms — NOT yet derived (the hard part)
The v95 `TryDoingBodyAttack` is a materially different, larger shape:
field-key + crypto-masked `_DR_INFO` fields, a `GetCrc32` checksum, `SKILLLEVELDATA`,
and an `ATTACKINFO[15]` hit loop. jms `TryDoingBodyAttack` Hex-Rays previously
FAILED to decompile (applicability.md fn9) — resolve by address. A faithful codec
must model the variable hit-loop + the masked fields and prove byte-exactness per
mode, which is task-sized work in its own right.

## Implementation outline (for the follow-up)
- `character/serverbound/touch_monster_attack.go` (CUserLocal owner → character/sb).
- Single immutable model with version branches on `t.MajorAtLeast(...)` /
  `t.Region()`; nested conditional fields modeled as explicit bools the
  Encode/Decode branch on; v95 hit-loop as a `[]HitInfo` sub-slice.
- Golden-byte tests per branch per version (mob-present + sub-block; no-mob; v95
  boss/crc paths), citing the COutPacket line per field.
- Handler decode+log + template routes at the IDA-verified opcodes (v83 0x30, …),
  NOT the csv opcodes. Then markers + evidence + reports like the other ops.
