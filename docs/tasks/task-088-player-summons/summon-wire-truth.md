# Summon packet wire truth (from the IDBs, asm-level)

> Authority = the client IDB (asm), NOT Cosmic (takes shortcuts) and NOT Hex-Rays
> pseudocode for the summon functions (flagged `positive sp value detected` /
> inlined — unreliable). Every read order below was confirmed at disassembly level.
> The summon pool is keyed by **owner charId (cid)**; the dispatcher consumes the
> leading `Decode4 cid` and looks the summon up by it, so per-op readers do NOT
> re-read an oid on v83/v87.

## v83 clientbound (CSummonedPool::OnPacket @0x938dd7 dispatch)

> **CORRECTION (oid is present on EVERY summon clientbound packet, v83 included).**
> The cid is read UPSTREAM in `CUserPool::OnUserCommonPacket@0x972401` (`Decode4
> characterId` for the whole `0xAF–0xB4` band) — NOT in `CSummonedPool::OnPacket`.
> So in `CSummonedPool::OnPacket@0x938dd7`: spawn (`0xAF`) calls `OnCreated` with no
> further Decode4 (→ oid, skillId); every non-spawn op does ONE `Decode4` = the
> **oid** (the pool lookup key) before its handler. The table below was written
> against `CSummonedPool::OnPacket` alone and mislabeled that per-op `Decode4` as
> the "cid" — it is the **oid**, and the real cid sits in front of it (upstream).
> Net wire for ALL ops: **`cid + oid + body`**. Atlas now writes the oid
> unconditionally in every `summon/clientbound/*.go`. v83 live-confirmed (x32dbg);
> v84/v87/v95/jms inherit by the same dispatcher logic + Cosmic — matrix cells need
> re-verification against the cid-pre-reading dispatcher (`0x938dd7`/`0x972401`),
> not the old per-handler addrs.

### Cross-version dispatcher confirmation (oid is universal — IDB-verified)
The classic-family `CSummonedPool::OnPacket` is byte-for-byte the same shape on
every GMS/JMS version: the spawn opcode calls the spawn reader via **vtable+0x30**
(v83 uses +0x2C) with **NO** in-pool `Decode4` (cid consumed upstream by
`OnUserCommonPacket`), and every NON-spawn opcode does exactly **one `Decode4`**
(the oid) before its handler. Confirmed by decompile:

| version | dispatcher | spawn op (no pre-read) | non-spawn `Decode4` oid |
|---|---|---|---|
| v83 GMS  | `0x938dd7` | `0xAF` | yes (+ **live x32dbg**: read offset already past cid at OnCreated's 1st Decode4) |
| v84 GMS  | `0x970201` | `179` (`0xB3`) | yes |
| v87 GMS  | `0x9b35bf` | `0xBC` | yes |
| jms185   | `0x9f7f6e` | `0xB5` | yes |
| v95 GMS  | `0x75ac70` (via `CField::OnPacket@0x546d50`) | flat switch; `Decode4 charId` once up front for **every** op incl. spawn | n/a (cid up front → oid present; oid was already written pre-fix, unchanged) |

So writing the oid unconditionally is correct on all versions. The remaining work
is a formal packet-verifier matrix pass to re-pin byte fixtures against these
dispatchers (the old per-handler `ida=` markers analyzed the layer below the cid
read); the wire contract itself is settled.

### Serverbound is NOT affected by this seam (one id, confirmed)
The clientbound seam is specific to the client's RECEIVE path (two layers:
`OnUserCommonPacket` cid + per-op oid). The client's SEND path has no such wrapper
— each send site builds the whole packet with a single leading id. v83 move send
`CVecCtrlSummoned::EndUpdateActive@0x9c84e9` = `COutPacket(0xAF)` + **one**
`Encode4(ctrl[0x248])` + `CMovePath::Flush` (no second id); attack/damage sends are
the same one-id shape (per-field ASM in §Serverbound). The Go serverbound decoders
read one `summonId` + body and reconcile via `resolveOwned` (`Get(id)` else
`GetByOwner(senderCharacterId)`), so cid-vs-oid is moot. **No serverbound change
needed.**

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
| Spawn | 0xAF | `cid(i4)` + **`oid(i4)`** + `skillId(i4)` + `charLevel(b)` + `SLV(b)` + Init blob | **CORRECTED — oid IS present on v83.** The earlier "no oid" reading analyzed `sub_938F61`, an **INACTIVE** OnCreated whose dispatcher does NOT pre-read cid. The **ACTIVE** field path is **OnCreated `@0x95ADEC`**, dispatched by a CSummonedPool::OnPacket that DOES `Decode4 cid` before the call. So the live read order is: dispatcher `cid`, then OnCreated `Decode4 oid` (→ ctor arg → [obj+0ACh]), `Decode4 skillId` (→ [obj+0B4h] → `GetSkill`), `Decode1 charLevel`, `Decode1 SLV`, then Init blob (`sub_7A379B`: `x i2, y i2, moveAction b, foothold i2, moveAbility b, assistType b, [Decode1 if GetSkill≠0]`). Wire = **cid, oid, skillId** (matches Cosmic spawnSummon). Write `oid` unconditionally; avatar-look byte stays `>=95` (GMS) / `>=185` (JMS). |

### v83 spawn — LIVE x32dbg evidence (the authoritative correction)
Breakpoint at OnCreated `@0x95AE07` (its first `Decode4`): `[ecx+0x14]` (CInPacket read offset) = **`0xA`** = header(4) + opcode(2) + **cid(4)** already consumed by the dispatcher. So the first `Decode4` reads the int AFTER cid; stepping it returned `EAX = 0x2F785D` = **3111005 (the skillId)** — proving that with no oid, the client consumes the skillId into the cid slot and then starves at the foothold `Decode2` (`@0x7A37CF`), closing the client.
```
dispatcher  Decode4 -> cid        (consumed before OnCreated; [ecx+14]=0xA on entry)
95ae07      Decode4 -> oid        (ctor arg_0 -> [obj+0ACh])   <-- this is the missing int
95ae10      Decode4 -> skillId    (ctor arg_4 -> [obj+0B4h]; GetSkill)
95ae1a      Decode1 -> charLevel
95ae24      Decode1 -> SLV
            CSummoned::Init(sub_7A379B)  Init blob
```
> NOTE: `sub_938F61` (no-oid) and `0x95ADEC` (oid) have identical *bodies*; they differ only in whether their dispatcher pre-reads cid. The active GMS-field path is `0x95ADEC`. v84/v87/jms185 inherit this correction by the same dispatcher logic + Cosmic, but were NOT re-confirmed live — their coverage-matrix cells need re-verification against the cid-pre-reading dispatcher.

### Confirmed bugs in current Atlas impl (libs/atlas-packet/summon + templates)
1. **`oid` gating was WRONG for Spawn (FIXED) — re-check Move/Attack/Damage.** Spawn
   *does* carry the oid on v83 (live-confirmed above; oid is now written
   unconditionally in `clientbound/spawn.go`). The original "no oid pre-95" reading
   came from the inactive `sub_938F61` dispatcher path. The Move/Attack/Damage
   clientbound packets were gated the same way (`oid >= 95`) on the same (now
   suspect) reasoning — they very likely ALSO need the oid on v83, but the owner's
   own client renders move/attack locally so a solo test never exercises them
   (they broadcast to OTHER sessions only). **Action: live-verify Move/Attack/Damage
   (and Remove/Skill) against the cid-pre-reading dispatcher with a second character
   in the map before trusting their `>=95` gate.**
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
