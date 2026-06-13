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
| Spawn | 0xAF | TBD (vtable+0x2C / OnCreated) | likely no skillId int on v83 (added v95). |

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

## Serverbound (client→server send sites) — TBD
- Move: CVecCtrlSummoned::EndUpdateActive
- Attack: CSummoned::TryDoingAttackManual (anti-hack envelope: drInfo/CRC/positions — our Cosmic-derived decoder does NOT match; needs faithful per-version port)
- Damage: CSummoned::SetDamaged
