# Summon packet wire truth (from the IDBs, asm-level)

> Authority = the client IDB (asm), NOT Cosmic (takes shortcuts) and NOT Hex-Rays
> pseudocode for the summon functions (flagged `positive sp value detected` /
> inlined — unreliable). Every read order below was confirmed at disassembly level.
> The summon pool is keyed by **owner charId (cid)**; the dispatcher consumes the
> leading `Decode4 cid` and looks the summon up by it, so per-op readers do NOT
> re-read an oid on v83/v87.

## v83 clientbound (CSummonedPool::OnPacket @0x938dd7 dispatch)

Dispatcher: if op==0xAF → spawn (vtable+0x2C). else `cid = Decode4`, pool-lookup by cid, then:
- 0xB0 → remove (sub_7A64EB)
- 0xB1 → OnMove @0x7a6861
- 0xB2 → OnAttack @0x7a6882
- 0xB3 → OnHit @0x7a6e5a   ← **reads 1 byte = SKILL behavior** (name misleading)
- 0xB4 → OnSkill @0x7a6ebe ← **reads damage fields = DAMAGE behavior** (name misleading)

| packet | opcode | v83 wire (after cid consumed by dispatcher) | notes |
|---|---|---|---|
| Move | 0xB1 | `cid` + `CMovePath::Decode`: `short startX, short startY, byte count, count×{move cmds}` | **no oid**. startX/startY are the FIRST fields of the move blob (`CMovePath::Decode@0x68a33c`). |
| Attack | 0xB2 | `cid` + `byte charLevel` + `byte action(hi-bit=left, lo7=action)` + `byte count` + count×{`int mobOid`, if≠0:`byte`,`int dmg`} | **no oid**. No trailing byte (v95 adds one). |
| **SKILL** | **0xB3** | `cid` + `byte (action&0x7F)` | summon plays skill animation. **just 1 byte** — NO summonSkillId int, NO oid. |
| **DAMAGE** | **0xB4** | `cid` + `byte attackIdx` + `int dmg` + if attackIdx>-2:{`int mobTemplateId`,`byte bLeft`} | **no oid**. (attackIdx is Cosmic's "12".) |
| Remove | 0xB0 | TBD (sub_7A64EB) | |
| Spawn | 0xAF | `cid(i4)` + `skillId(i4)` + `charLevel(b)` + `SLV(b)` + Init blob | **CONFIRMED asm** (OnCreated = sub_938F61). **NO oid on v83.** Init blob (sub_7A379B): `x(i2), y(i2), moveAction(b), foothold(i2), moveAbility(b), assistType(b), [if foothold found: enterType(b)]`. The int after cid is the **skillId** (passed to CSummoned ctor sub_7A30A9 → stored at [obj+0B4h] → consumed by `GetSkill@CSkillInfo` in sub_7A379B), NOT an oid. So `skillId` is PRESENT on v83; `oid` is the v95-only addition. v95 OnCreated reads cid, **oid**, skillId, charLevel, SLV before the Init blob. → gate **oid `>=95`**; keep `skillId` unconditional. avatar-look byte stays `>=95`. |

### v83 spawn asm evidence (sub_938F61, the 0xAF vtable+0x2C target)
```
938f7c  Decode4  -> arg_4   = cid       (ctor arg_0 -> [obj+0ACh])
938f86  Decode4  -> var_18  = skillId   (ctor arg_4 -> [obj+0B4h]; sub_7A379B does push [edi+0B4h]; GetSkill)
938f90  Decode1  -> var_14  = charLevel (ctor arg_8 -> [obj+0B8h])
938f9a  Decode1  -> var_10  = SLV       (ctor arg_C -> [obj+0BCh])
939030  call sub_7A379B  (Init blob: x i2, y i2, moveAction b, foothold i2, moveAbility b, assistType b, [enterType b if fh])
```
Only ONE int between cid and the two bytes → v83 has NO oid. v95's OnCreated inserts oid (i4) right after cid.

### Confirmed bugs in current Atlas impl (libs/atlas-packet/summon + templates)
1. **Extra `oid`**: clientbound Move/Attack/Damage write `int oid` right after `cid`. v83/v87 clients DON'T read it (pool is cid-keyed). `oid` is a **v95+ addition** → gate `oid` write/read on `>= 95` (GMS), omit below.
2. **Skill/Damage opcodes SWAPPED** in templates for v83/v84/v87/jms185: skill is the LOWER opcode, damage the HIGHER, in **every** version (incl. v95, which the task-088 6.1 harvest got right by luck; the others it assigned backwards by trusting the misleading OnHit/OnSkill names). v83 must be SKILL=0xB3, DAMAGE=0xB4. v95 (SKILL=0x11A, DAMAGE=0x11B) already correct.
3. **SummonSkill structure wrong**: we write `cid + summonSkillId(int) + newStance(byte)`. Client reads `cid + 1 byte`. Drop the summonSkillId int (all versions — v95 OnSkill also reads a single byte).

## v95 deltas (PDB, names reliable — from prior v95 verify pass)
- Move/Attack/Damage DO carry `oid` (read via CUser::OnSummonedX after the pool cid). → `oid` present `>=95`.
- Spawn carries `skillId` int (read @0x75a9ef) + the avatar-look byte (>=95) — v83 spawn likely omits skillId.
- Attack has a trailing byte (>=95) absent in v83/v87.
- Damage has a trailing `dir<0` byte present **since v87** (gate the trailing byte `>=87`, not `>=95`).
- SKILL=0x11A, DAMAGE=0x11B (skill lower) — correct.

## Serverbound (client→server SEND sites) — CONFIRMED at asm (task-088)

> Authority = the COutPacket SEND sites in each IDB. Identity field = the int
> right after the opcode. **v83/v87 identify the summon by the owner charId
> (cid, CSummoned [obj+0xAC], set from ctor arg_0 = cid). v95 identifies it by
> the server-allocated m_dwSummonedID.** The v83 client has NO oid concept (the
> pool is cid-keyed — see clientbound section), so the channel handler passes
> the wire id through and atlas-summons reconciles via
> `GetByOwner(senderCharacterId)` when the id misses (resolveOwned).

### Identity / opcode matrix

| packet | v83 op | v87 op | v95 op | v83/v87 identity | v95 identity |
|---|---|---|---|---|---|
| Move   | 0xAF | 0xBB | 0xCF (207) | owner cid | m_dwSummonedID |
| Attack | 0xB0 | 0xBC | 0xD0 (208) | owner cid | m_dwSummonedID |
| Damage | 0xB1 | 0xBD | 0xD1 (209) | owner cid | m_dwSummonedID |

(Opcodes are routed by the socket layer to the registered handler; the decoders
consume the body only.)

### Move send — CVecCtrlSummoned::EndUpdateActive (v83 sub_9C84E9, v87 @0xa591da, v95 @0x9a0700)
```
COutPacket(op)
Encode4 summonId        ; v83 ctrl[0x248]=cid (sub_9C84E9 @0x9c853d); v87 ctrl[188]; v95 m_dwSummonedID
CMovePath::Flush(...)    ; opaque move blob (CMovePath::Encode @0x68a563):
                         ;   Encode2 startX, Encode2 startY, Encode1 count,
                         ;   count×{cmd...}, Encode1 keypadLen, keypad..., Encode2 minX/minY/maxX/maxY
```
Identical shape across all versions (only identity semantics + opcode differ).
**Decoder**: `summonId` then `rawMovement = rest`; startX/startY = first 4 bytes
of the blob (for position seeding); rawMovement is rebroadcast byte-faithfully.

### Damage send — CSummoned::SetDamaged (v83 @0x7a607a, v87 @0x7f879a, v95 @0x74b730)
**Byte-identical body across all three versions** (only identity + opcode differ):
```
Encode4 summonId
if (source mob present):
  Encode1 attackIdx                 ; mob attack index
  Encode4 damage
  Encode4 monsterTemplateId         ; mob dwTemplateID (NOT an oid)
  Encode1 (dir < 0)                 ; impact-dir flag — PRESENT in v83 too (@0x7a62f4)
else:
  Encode1 0xFE                      ; sentinel "-2" (no source mob) (@0x7a62a8)
  Encode4 damage
```
**Correction to prior doc**: the trailing dir byte is NOT v95-only — it is in
v83/v87 as well; and the 0xFE no-mob branch exists in all versions. The old
Cosmic-derived decoder (oid + skip1 + dmg + monsterIdFrom, no dir byte, no
0xFE branch) was wrong.

### Attack send — CSummoned::TryDoingAttackManual (v83 sub_7A4D42 @0x7a57dc, v87 @0x7f6666, v95 @0x751240)
**Three structurally distinct layouts** (per-target block identical):
```
v83 header (LEAN — no anti-hack envelope):
  Encode4 summonId(cid), Encode4 updateTime, Encode1 action|left, Encode1 count,
  Encode2 userX,userY, Encode2 summonX,summonY

v87 header (anti-hack envelope, NO repeatSkillPoint):
  Encode4 summonId(cid), Encode4 ~drInfo0, Encode4 ~drInfo1, Encode4 updateTime,
  Encode4 ~drInfo2, Encode4 ~drInfo3, Encode1 action|left, Encode4 dwKey,
  Encode4 crc32, Encode1 count, Encode2 userX,userY,summonX,summonY

v95 header (envelope + repeatSkillPoint):
  ...as v87... then Encode4 repeatSkillPoint (@0x752450)

per target (26 bytes, all versions):
  Encode4 mobOid, Encode4 templateId, Encode1 hitAction, Encode1 foreAction|left,
  Encode1 frameIdx, Encode1 calcDamageStatIndex, Encode2 curX, Encode2 curY,
  Encode2 hitX, Encode2 hitY, Encode2 tDelay, Encode4 damage

trailer: Encode4 skillCRC
```
**Decoder** gates: `IsRegion("GMS") && MajorAtLeast(87)` → envelope present;
`MajorAtLeast(95)` → also repeatSkillPoint. v84 == v83 (lean). drInfo/dwKey/crc32
are read at exact widths (skipped, not validated) so the cursor stays aligned and
the target mobOid/damage fields decode correctly. Server consumes summon
identity + per-target mobOid + damage + delay (+ templateId, surfaced but unused).

### Go reconciliation (channel → summons)
- Decoders expose `SummonId()` (= wire identity), `Targets()` (mobOid/templateId/
  damage/delay), `Damage`/`MonsterIdFrom` (= mob template id).
- Channel handlers pass `p.SummonId()` + `s.CharacterId()` (= owner cid) into the
  SUMMON command bodies unchanged.
- `summon.ProcessorImpl.resolveOwned(id, senderCharacterId, preferPuppet)`:
  tries `Get(id)` (owner-matched, the v95/exact path); else
  `GetByOwner(senderCharacterId)` (the v83/v87 path where the wire id IS the cid).
  preferPuppet=true for Damage, false for Move/Attack. `senderCharacterId` is the
  authoritative session owner, so owner-based resolution is safe.
