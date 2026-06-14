# jms_v185 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `MapleStory_dump_SCY.exe` (JMS v185), port 13340. Harvested 2026-06-13 → merged
absent-keys-only into `docs/packets/ida-exports/gms_jms_185.json` (302 → 335 keys).
**26/26 bulk roster fnames resolved + CMob::ShowCatchEffect (the CATCH_MONSTER target),
0 unresolved.** jms has good symbol coverage; the giants (CMob::Update, TryDoingBodyAttack)
were not rostered (see blockers).

## jms-specific structure

- **MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY are VERSION-ABSENT in jms** —
  no registry rows, and the jms `CMobPool::OnMobPacket` dispatcher has no such cases
  (the GMS 0xFD-0x100 cluster does not exist in JMS v185). Correct per context.md §2; no
  rows added.
- The GMS-only clientbound escort/next-attack family (MOB_ESCORT_FULL_PATH, MOB_NEXT_ATTACK,
  MOB_ATTACKED_BY_MOB, MOB_ESCORT_RETURN_BEFORE, MOB_ESCORT_STOP/_SAY) is ABSENT in jms.
  jms DOES carry the serverbound escort sends (MOB_ESCORT_COLLISION 203,
  MOB_REQUEST_ESCORT_INFO 204, MOB_ESCORT_STOP_END_REQUEST 205) — all named + resolved.

## Registry state (1 edit needed)

- MONSTER_BOOK_COVER (sb, 49): `""` → `CUserLocal::SetMonsterBookCover` (ida-discovered, 0xA2C930).
- CATCH_MONSTER (cb, 268): fname stays `CMob::OnCatchEffect` (unnamed in jms IDB) with
  `fname_alts: [sub_6EAE5F]` — **the real dispatch target is unnamed `sub_6EAE5F` @0x6EAE5F**,
  which `Decode1`s then calls `CMob::ShowCatchEffect` @0x6E5F77. For Stage-2 evidence, pin
  against `CMob::ShowCatchEffect` (now in the export) or the sub address; the wire layout is
  a single Decode1 (catch result byte). No registry change beyond the existing alt.

## Sub-op demux / shared functions (Stage 2 attention)

- **CField_MonsterCarnival::OnRequestResult** @0x5B0332 — SUMMON (316/0x13C) vs
  MESSAGE (317/0x13D), branch on `bResult` arg: `!=0` SUMMON → Decode1, Decode1, DecodeStr;
  `==0` MESSAGE → Decode1 (selector 1..6, StringPool 0x101C..0x1020). Same as GMS.
- **CMob::Update** backs FIELD_DAMAGE_MOB (197), MOB_DAMAGE_MOB_FRIENDLY (198),
  MOB_SKILL_DELAY_END (201) — shared tick; NOT rostered here (giant). Derive per-op
  serverbound payloads from the COutPacket build sites.
- **CMobPool::OnMobCrcKeyChanged** @0x6F8BCB backs MOB_CRC_KEY_CHANGED (cb) + REPLY (sb).
- **sub_6EAE5F** (CATCH_MONSTER): `Decode1` → `CMob::ShowCatchEffect`.

---

## Byte layouts

### CField_MonsterCarnival::OnEnter
- **address:** 0x5b014c
- **calls (10):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode2` 
  - `Decode1` 
  - `Delegate`  -> sub_42A9E9
  - `Delegate`  -> sub_4067E3

### CField_MonsterCarnival::OnPersonalCP
- **address:** 0x5b02c3
- **calls (2):**
  - `Decode2` 
  - `Decode2` 

### CField_MonsterCarnival::OnProcessForDeath
- **address:** 0x5b0597
- **calls (4):**
  - `Decode1` 
  - `DecodeStr` 
  - `Decode1` 
  - `Delegate`  -> sub_4067E3

### CField_MonsterCarnival::OnRequestResult
- **address:** 0x5b0332
- **calls (4):**
  - `Decode1` 
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnShowGameResult
- **address:** 0x5b088a
- **calls (1):**
  - `Decode1` 

### CField_MonsterCarnival::OnShowMemberOutMsg
- **address:** 0x5b070f
- **calls (4):**
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 
  - `Delegate`  -> sub_4067E3

### CField_MonsterCarnival::OnTeamCP
- **address:** 0x5b02f3
- **calls (3):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 

### CMob::OnAffected
- **address:** 0x6e9df6
- **calls (2):**
  - `Decode4` 
  - `Decode2` 

### CMob::OnEffectByItem
- **address:** 0x6eb148
- **calls (2):**
  - `Decode4` 
  - `Decode1` 

### CMob::OnSpecialEffectBySkill
- **address:** 0x6eb08d
- **calls (4):**
  - `Decode4` 
  - `Delegate`  -> sub_411A4F
  - `Delegate`  -> sub_402D06
  - `Delegate`  -> sub_402EA5

### CMob::OnSuspendReset
- **address:** 0x6e9c8d
- **calls (1):**
  - `Decode1` 

### CMob::SendCollisionEscort
- **address:** 0x6efeb7
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendDropPickUpRequest
- **address:** 0x6ec289
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendEscortStopEndRequest
- **address:** 0x6effcd
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendRequestEscortPath
- **address:** 0x6eff57
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SetDamagedByMob
- **address:** 0x6edce8
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::ShowCatchEffect
- **address:** 0x6e5f77
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::TryFirstSelfDestruction
- **address:** 0x6ebf98
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::UpdateTimeBomb
- **address:** 0x6ef8f8
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x6f8bcb
- **calls (1):**
  - `Decode4` 

### CUIMonsterCarnival::RequestSend
- **address:** 0x903e24
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::SendBanMapByMobRequest
- **address:** 0xa28621
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::SetMonsterBookCover
- **address:** 0xa2c930
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::TryDoingBodyAttack
- **address:** 0xa2ab71
- **direction:** sb (TOUCH_MONSTER_ATTACK)
- **calls (0):**
  - (NAMED 2026-06-13 — was `sub_A2AB71`; renamed to the v95-canonical mangled symbol.
    Hex-Rays FAILS to auto-decompile this 0x1a3a-byte jms function, so the export harvester
    records it `unresolved: "decompilation failed; hand-trace"` and it was NOT merged into
    the export JSON. The symbol now exists in the IDB and resolves by address for manual
    evidence. Identity is byte-confirmed by: called twice from `Update@CUserLocal` immediately
    after `FindBodyAttackMob@CMobPool` (0xA0B539→0xA0B614/0xA0B69B, the left/right facing
    body-attack sweep — exactly the v95 pattern); callee fingerprint GetRandomHitAction@CMob +
    GetHitPoint@CMob + AddDamageInfo@CMob + GetCurrentAction/FrameIndex@CMob + IsVulnerableTo;
    and a stack frame with body-attack locals `rcAttack`/`nAttackIdx`/`bMoveLeft`/
    `nMoveEndingPosX`/`nMoveType`/`nSkillID`/`bLeft`/`bChase`. Wire layout = v83/v95-equivalent
    TOUCH_MONSTER_ATTACK; transcribe the v83 send-site (opcode + Encode order) until the jms
    COutPacket build site is hand-traced from disasm.)

### CWvsContext::OnBridleMobCatchFail
- **address:** 0xaec5ed
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCard
- **address:** 0xaec797
- **calls (5):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 
  - `Delegate`  -> sub_4067E3
  - `Delegate`  -> sub_4067E3

### CWvsContext::OnMonsterBookSetCover
- **address:** 0xaec8b5
- **calls (1):**
  - `Decode4` 

### CWvsContext::OnSetTamingMobInfo
- **address:** 0xb103a1
- **calls (7):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 
  - `Delegate`  -> sub_40265E
  - `Delegate`  -> sub_40265E



---

## Stage-2 blockers (jms)

| op | dir | opcode | registry fname | status |
|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK | sb | 38 | CUserLocal::TryDoingBodyAttack | **NAMED 2026-06-13** (was `sub_A2AB71` @0xA2AB71; rename saved). Symbol now resolves by address. **NOT auto-harvestable**: jms Hex-Rays fails to decompile it, so the export carries no captured layout — hand-trace the COutPacket build site or inherit the v83/v95 layout for the byte spec. See byte-layout entry above for the identity evidence. |
| CATCH_MONSTER | cb | 268 | CMob::OnCatchEffect | RESOLVED via alt — fname unnamed; pin against `CMob::ShowCatchEffect` @0x6E5F77 (in export) or `sub_6EAE5F` @0x6EAE5F. Layout = 1×Decode1. |

MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY and the GMS clientbound escort/next
family are VERSION-ABSENT in jms (no rows, no dispatch) — recorded in applicability.md, not
blockers.
