# gms_v87 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `GMSv87_4GB.exe` (v87), port 13338. Harvested 2026-06-13 from the live IDB via
`packet-audit export` → merged absent-keys-only into `docs/packets/ida-exports/gms_v87.json`
(319 → 351 keys). Export resolution: **28/28 roster fnames resolved, 0 unresolved**
(run split into a 26-fname bulk pass + a 2-fname giant pass for `CMob::Update` 0x17C3 and
`CUserLocal::TryDoingBodyAttack` 0x128C with `--descent-depth 1 --ida-timeout 300s`; the
default 60s timeout aborted on those two).

## Dispatcher provenance + registry fname corrections (the v87 three-way un-rotation)

`CMobPool::OnMobPacket` @ 0x6B4EAD. Verified cases (the whole cluster was rotated in the
registry; un-rotated to op-name alignment this version):

| client case | handler | addr |
|---|---|---|
| 0x104 | CMob::OnSuspendReset | 0x6A73CB |
| 0x105 | CMob::OnAffected | 0x6A7540 |
| 0x107 | CMob::OnSpecialEffectBySkill | 0x6A87B3 |
| 0x10B | CMob::OnCatchEffect | 0x6A8585 |
| 0x10C | CMob::OnEffectByItem | 0x6A886E |
| 0x10D | CMob::OnMobSpeaking | 0x6AC31E |
| 0x10E | CMob::OnIncMobChargeCount | 0x6AC230 |
| 0x10F | CMob::OnMobSkillDelay | 0x6AD0E8 |
| 0x110 | CMob::OnMobAttackedByMob | 0x6AC074 |

The dispatcher named cases END at 0x110. **There is no 0x111 case and no
`CMob::OnEscortFullPath` symbol in the v87 IDB** → MOB_ESCORT_FULL_PATH (273/0x111) is
**VERSION-ABSENT in v87** (confirms the registry caveat note). Same +1 Atlas-op-vs-client-case
off-by-one as v83 (MOB_SPEAKING=270/0x10E in Atlas, OnMobSpeaking dispatches at case 0x10D).

Registry edits this version:
- MOB_SPEAKING (270) `CMob::OnIncMobChargeCount` → `CMob::OnMobSpeaking` (manual, 0x6AC31E).
- INC_MOB_CHARGE_COUNT (271) `CMob::OnMobSkillDelay` → `CMob::OnIncMobChargeCount` (manual, 0x6AC230).
- MOB_SKILL_DELAY (272) `CMob::OnMobAttackedByMob` → `CMob::OnMobSkillDelay` (manual, 0x6AD0E8).
- MONSTER_BOOK_COVER (sb, 60) `""` → `CUserLocal::SetMonsterBookCover` (ida-discovered, 0x9E2D06).
- MOB_ESCORT_FULL_PATH (273): left as-is (registry already noted absent); recorded VERSION-ABSENT
  in applicability.md.

## Shared-function sub-op demux (Stage 2 attention)

**CField_MonsterCarnival::OnRequestResult** @ 0x590303 serves MONSTER_CARNIVAL_SUMMON
(309/0x135) and MONSTER_CARNIVAL_MESSAGE (310/0x136), branching on opcode arg `a2`
(IDENTICAL structure to v83):
- `a2 != 0` (SUMMON) → `Decode1`, `Decode1`, `DecodeStr` (name). 3 reads.
- `a2 == 0` (MESSAGE) → `Decode1` (message-type selector 1..6); no further wire reads
  (strings from StringPool 4090..4094).

**CMob::Update** @ 0x6A1C43 backs FIELD_DAMAGE_MOB (203), MOB_DAMAGE_MOB_FRIENDLY (204),
MOB_SKILL_DELAY_END (207) — same shared-tick caveat as v83; derive per-op send payloads
from the COutPacket build sites, not this read body.

**CMobPool::OnMobCrcKeyChanged** backs MOB_CRC_KEY_CHANGED (cb) + MOB_CRC_KEY_CHANGED_REPLY (sb).

Note: `CUserLocal::TryDoingBodyAttack` (TOUCH_MONSTER_ATTACK) and
`CUserLocal::SendBanMapByMobRequest` (MOB_BANISH_PLAYER) ARE named symbols in v87
(0x9E17DC / 0x9DF571), unlike v83 where they were unnamed.

---

## Byte layouts

### CField_MonsterCarnival::OnEnter
- **address:** 0x59011d
- **direction:** 
- **calls (8):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode1` 

### CField_MonsterCarnival::OnPersonalCP
- **address:** 0x590294
- **direction:** 
- **calls (2):**
  - `Decode2` 
  - `Decode2` 

### CField_MonsterCarnival::OnProcessForDeath
- **address:** 0x590568
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `DecodeStr` 
  - `Decode1` 

### CField_MonsterCarnival::OnRequestResult
- **address:** 0x590303
- **direction:** 
- **calls (4):**
  - `Decode1` 
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnShowGameResult
- **address:** 0x59085e
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CField_MonsterCarnival::OnShowMemberOutMsg
- **address:** 0x5906e3
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnTeamCP
- **address:** 0x5902c4
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 

### CMob::OnAffected
- **address:** 0x6a7540
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode2` 

### CMob::OnCatchEffect
- **address:** 0x6a8585
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CMob::OnEffectByItem
- **address:** 0x6a886e
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode1` 

### CMob::OnIncMobChargeCount
- **address:** 0x6ac230
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnMobSkillDelay
- **address:** 0x6ad0e8
- **direction:** 
- **calls (4):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 

### CMob::OnMobSpeaking
- **address:** 0x6ac31e
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnSpecialEffectBySkill
- **address:** 0x6a87b3
- **direction:** 
- **calls (4):**
  - `Decode4` 
  - `Delegate` 
  - `Delegate` 
  - `Delegate` 

### CMob::OnSuspendReset
- **address:** 0x6a73cb
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CMob::SendDropPickUpRequest
- **address:** 0x6a98ae
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::SetDamagedByMob
- **address:** 0x6abd95
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::TryFirstSelfDestruction
- **address:** 0x6a95bd
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::Update
- **address:** 0x6a1c43
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x6b5399
- **direction:** 
- **calls (1):**
  - `Decode4` 

### CUIMonsterCarnival::RequestSend
- **address:** 0x8d93c3
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CUserLocal::SendBanMapByMobRequest
- **address:** 0x9df571
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CUserLocal::SetMonsterBookCover
- **address:** 0x9e2d06
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CUserLocal::TryDoingBodyAttack
- **address:** 0x9e17dc
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CWvsContext::OnBridleMobCatchFail
- **address:** 0xa9d692
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCard
- **address:** 0xa9d83c
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCover
- **address:** 0xa9d959
- **direction:** 
- **calls (1):**
  - `Decode4` 

### CWvsContext::OnSetTamingMobInfo
- **address:** 0xac0d8b
- **direction:** 
- **calls (7):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 
  - `Delegate` 
  - `Delegate` 



---

## Stage-2 blockers (fname not resolvable in v87 IDB)

| op | dir | opcode | registry fname | status |
|---|---|---|---|---|
| MOB_TIME_BOMB_END | sb | 208 | CMob::UpdateTimeBomb | UNRESOLVED — no `UpdateTimeBomb`/`TimeBomb` symbol in v87 IDB |
| MOB_ESCORT_COLLISION | sb | 209 | CMob::SendCollisionEscort | UNRESOLVED — no `Escort`/`CollisionEscort` symbol in v87 IDB |
| MOB_ESCORT_FULL_PATH | cb | 273 | CMob::OnEscortFullPath | VERSION-ABSENT — 0x111 not dispatched; no symbol (see dispatcher above) |

TIME_BOMB_END / ESCORT_COLLISION are real ops with unnamed send-sites; ESCORT_FULL_PATH is
genuinely version-absent in v87. Flagged, not fabricated.
