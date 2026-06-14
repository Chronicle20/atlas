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

## v95 — FULLY DERIVED (CUserLocal::TryDoingBodyAttack @0x930710)
**Opcode `0x32` (50)** — `COutPacket(&oPacket, 50)`. Radically different from v83:
a field-key byte, four inverted `_DR_INFO` crypto fields, nibble-packed counts,
two CRC families, and a per-mob ATTACKINFO loop with an inner per-hit damage loop.

Prefix (in order):
```
Encode1  fieldKey            // CField::GetFieldKey()
Encode4  ~dr0                 // _DR_INFO masked (inverted)
Encode4  ~dr1
Encode1  countByte           // low nibble = nDamagePerMob (hits), high nibble = nRange (mob count)
Encode4  ~dr2
Encode4  ~dr3
Encode4  skillId             // 0 if none
Encode1  combatOrders
Encode4  rand                // get_rand(dr0,0)
Encode4  crc32               // CCrc32::GetCrc32(pData,4,rand)
Encode4  skillLevelCrc1      // SKILLLEVELDATA::GetCrc (0 if no skill)
Encode4  skillLevelCrc2      // 0 if no skill
Encode1  0
Encode2  action|(bLeft<<15)  // action low 15 bits, left flag bit15
Encode4  actionCrc           // GETCRC32Svr<long> per-action
Encode1  attackActionType
Encode1  0                    // v188
Encode4  attackTime          // get_update_time
Encode4  dwId                // v221 if skill 32121003, else dwID
```
Per-mob loop `for nOrder in 0..nRange`:
```
Encode4  mobId               // CMob::GetMobID
Encode1  hitAction
Encode1  foreAction&0x7F | (isLeft<<7)
Encode1  frameIdx
Encode1  calcDmgStatIdx&0x7F | (templateChanged<<7)
Encode2  mobX
Encode2  mobY
Encode2  mob2X
Encode2  mob2Y
Encode2  delay
  for j in 0..nDamagePerMob:  Encode4  damage[j]   // inner per-hit loop
Encode4  mobCrc              // CMob::GetCrc
```
Trailer: `Encode2 playerX; Encode2 playerY;` then SendPacket.

For a serverbound decode+log codec the crypto/CRC values are opaque wire ints
(the server reads the widths, it does not recompute them) — model the STRUCTURE:
read `countByte`, split the nibbles, then loop `nRange` mobs × `nDamagePerMob`
inner damages.

## jms — STATICALLY UNRECOVERABLE (SMC / control-flow virtualization)
jms `TryDoingBodyAttack` @0xa2ab71: Hex-Rays **fails** ("Decompilation failed").
The packet-build region is **encrypted at rest** — the function jumps via a
self-modifying-code trampoline (`loc_DF70C0`→`loc_D21897`: `pusha;…;lock cmpxchg;
lodsb`) and the `COutPacket`/`Encode*` instructions only materialize at runtime.
An xref scan of `COutPacket::COutPacket`/`SendPacket` finds NO plaintext call
sites in the function; even the opcode is hidden behind the trampoline. The whole
attack-send family (incl. `TryDoingMeleeAttack`) is SMC-protected in this build.
**This cannot be derived from the static IDB** — it needs a runtime/dynamic dump
(debugger breakpoint on `??0COutPacket@@QAE@J@Z` while triggering a body attack,
or a post-decrypt memory snapshot). Genuine stop-and-escalate; jms TOUCH stays ❌
until a dynamic capture exists.

## Demux note (affects decode modeling)
v83 has no clean on-wire discriminator between the body-attack and no-mob
(fall/obstacle) branches — the client picks the branch from runtime state, and
both start `u8,u8,u32`. A faithful Decode needs an explicit demux design
(length-based, or model only the dominant body-attack path + document the variant).
This, plus the v95 nibble-driven double loop and the jms SMC wall, is why TOUCH is
a designed follow-up rather than a transcription.

## Implementation outline (for the follow-up)
- `character/serverbound/touch_monster_attack.go` (CUserLocal owner → character/sb).
- Single immutable model with version branches on `t.MajorAtLeast(...)` /
  `t.Region()`; nested conditional fields modeled as explicit bools the
  Encode/Decode branch on; v95 hit-loop as a `[]HitInfo` sub-slice.
- Golden-byte tests per branch per version (mob-present + sub-block; no-mob; v95
  boss/crc paths), citing the COutPacket line per field.
- Handler decode+log + template routes at the IDA-verified opcodes (v83 0x30, …),
  NOT the csv opcodes. Then markers + evidence + reports like the other ops.
