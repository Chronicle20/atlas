# gms_v83 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `MapleStory_dump.exe` (v83_Me), port 13337. Harvested 2026-06-13 from the live
IDB via `packet-audit export` (targeted roster) → merged absent-keys-only into
`docs/packets/ida-exports/gms_v83.json`. Each `### Class::Method` below IS the byte
layout Stage 2 transcribes: the ordered `Decode*`/`Encode*` calls are the wire fields
in order. Comments are sparse in the targeted export; decompile addresses are given so
Stage 2 can re-open any handler.

## Dispatcher provenance (registry fname corrections)

`CMobPool::OnMobPacket` @ 0x67936D dispatches the clientbound mob cluster. Verified cases:

| client case | handler |
|---|---|
| 0xF4 | CMob::OnSuspendReset |
| 0xF5 | CMob::OnAffected |
| 0xF7 | CMob::OnSpecialEffectBySkill |
| 0xFB | CMob::OnCatchEffect |
| 0xFC | CMob::OnEffectByItem |
| 0xFD | CMob::OnMobSpeaking (0x6711EA) |
| 0xFE | CMob::OnIncMobChargeCount (0x6710FC) |
| 0xFF | CMob::OnMobAttackedByMob (0x670F41) |

**Atlas op-enum vs client-case off-by-one:** the Atlas registry opcodes for the
0xFD–0xFF cluster sit +1 above the client case label (MOB_SPEAKING=254/0xFE in Atlas,
but the client dispatches OnMobSpeaking at case 0xFD). This is the known v83 mob-cluster
off-by-one (also flagged on the SHOW_MAGNET 0xFD row). Registry fnames were assigned by
**op name** (MOB_SPEAKING→OnMobSpeaking, INC_MOB_CHARGE_COUNT→OnIncMobChargeCount),
not by the raw client case. `MOB_SKILL_DELAY` (256/0x100) keeps its csv-import fname
`CMob::OnMobSkillDelay`, but **that symbol does not exist in the v83 IDB** — the
dispatcher's named cases stop at 0xFF (OnMobAttackedByMob). See Blockers below.

Registry edits this version:
- MOB_SPEAKING (254) fname `CMob::OnIncMobChargeCount` → `CMob::OnMobSpeaking` (manual, 0x6711EA).
- INC_MOB_CHARGE_COUNT (255) fname `CMob::OnMobAttackedByMob` → `CMob::OnIncMobChargeCount` (manual, 0x6710FC).
- MONSTER_BOOK_COVER (sb, 57) fname `""` → `CUserLocal::SetMonsterBookCover` (ida-discovered, 0x95FB3E).

## Shared-function sub-op demux (Stage 2 attention)

**CField_MonsterCarnival::OnRequestResult** @ 0x56557D serves BOTH
MONSTER_CARNIVAL_SUMMON (292/0x124) and MONSTER_CARNIVAL_MESSAGE (293/0x125). It branches
on its opcode arg `a2`:
- `a2 != 0` (SUMMON) → `Decode1` (idx0), `Decode1` (idx1), `DecodeStr` (name). 3 reads.
- `a2 == 0` (MESSAGE) → `Decode1` (message-type selector 1..6), then NO further wire
  reads — the displayed strings come from StringPool (SP_4082..SP_4086), not the packet.
  So MONSTER_CARNIVAL_MESSAGE wire payload = a single Decode1.

**CMob::Update** @ 0x6675A8 (serverbound) backs FIELD_DAMAGE_MOB (191), MOB_DAMAGE_MOB_FRIENDLY (192),
and MOB_SKILL_DELAY_END (195). It is a 0x14FE-byte virtual tick method; the three ops are
demuxed elsewhere (these serverbound rows point at it as the conceptual owner). Stage 2 must
derive the three distinct send-payloads from the client's COutPacket build sites, NOT from
this read-side Update body. The targeted export captured Update's internal Decode calls,
which are NOT the per-op serverbound layouts — treat with care.

**CMobPool::OnMobCrcKeyChanged** @ 0x6797BE backs both MOB_CRC_KEY_CHANGED (cb) and
MOB_CRC_KEY_CHANGED_REPLY (sb).

---

## Byte layouts

### CField_MonsterCarnival::OnEnter
- **address:** 0x565397
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
- **address:** 0x56550e
- **direction:** 
- **calls (2):**
  - `Decode2` 
  - `Decode2` 

### CField_MonsterCarnival::OnProcessForDeath
- **address:** 0x5657e7
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `DecodeStr` 
  - `Decode1` 

### CField_MonsterCarnival::OnRequestResult
- **address:** 0x56557d
- **direction:** 
- **calls (4):**
  - `Decode1` 
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnShowGameResult
- **address:** 0x565add
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CField_MonsterCarnival::OnShowMemberOutMsg
- **address:** 0x565962
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnTeamCP
- **address:** 0x56553e
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 

### CMob::OnAffected
- **address:** 0x66c675
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode2` 

### CMob::OnCatchEffect
- **address:** 0x66d6b9
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CMob::OnEffectByItem
- **address:** 0x66d997
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode1` 

### CMob::OnIncMobChargeCount
- **address:** 0x6710fc
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnMobSpeaking
- **address:** 0x6711ea
- **direction:** 
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnSpecialEffectBySkill
- **address:** 0x66d8e7
- **direction:** 
- **calls (4):**
  - `Decode4` 
  - `Delegate` 
  - `Delegate` 
  - `Delegate` 

### CMob::OnSuspendReset
- **address:** 0x66c500
- **direction:** 
- **calls (1):**
  - `Decode1` 

### CMob::SendDropPickUpRequest
- **address:** 0x66e91f
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::SetDamagedByMob
- **address:** 0x670c63
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::TryFirstSelfDestruction
- **address:** 0x66e636
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMob::Update
- **address:** 0x6675a8
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x6797be
- **direction:** 
- **calls (1):**
  - `Decode4` 

### CUIMonsterCarnival::RequestSend
- **address:** 0x8706d3
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CUserLocal::SetMonsterBookCover
- **address:** 0x95fb3e
- **direction:** 
- **calls (0):**
  - (no Decode/Encode calls captured — handler reads nothing, or all reads via descended helper)

### CWvsContext::OnBridleMobCatchFail
- **address:** 0xa0800e
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCard
- **address:** 0xa081b8
- **direction:** 
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCover
- **address:** 0xa082d5
- **direction:** 
- **calls (1):**
  - `Decode4` 

### CWvsContext::OnSetTamingMobInfo
- **address:** 0xa29115
- **direction:** 
- **calls (5):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 



---

## Stage-2 blockers (fname not resolvable in v83 IDB)

These ops have real Atlas opcodes and real registry rows, but their registry fname is a
csv-import conceptual name with **no matching named symbol in the v83 IDB**, so the
export cannot carry them and `evidence pin` will fail until the real send-site is derived:

| op | dir | opcode | registry fname | status |
|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK | sb | 47 | CUserLocal::TryDoingBodyAttack | UNRESOLVED — no `TryDoingBodyAttack`/`BodyAttack@CUserLocal` symbol in IDB |
| MOB_BANISH_PLAYER | sb | 56 | CUserLocal::SendBanMapByMobRequest | UNRESOLVED — no `SendBanMap*`/`BanMapByMob` symbol in IDB |
| MOB_TIME_BOMB_END | sb | 196 | CMob::UpdateTimeBomb | UNRESOLVED — no `UpdateTimeBomb`/`TimeBomb@CMob` symbol in IDB |
| MOB_SKILL_DELAY | cb | 256 | CMob::OnMobSkillDelay | UNRESOLVED — symbol absent; dispatcher named cases stop at 0xFF |

These are NOT version-absent (the ops exist); they are IDB-naming gaps. Stage 2 must
derive the real send/handler sites (likely unnamed `sub_XXXX` reachable from the CP-enum
encode sites) before pinning. Flagged, not fabricated.
