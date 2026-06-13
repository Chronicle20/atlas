# gms_v95 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `GMS_v95.0_U_DEVM.exe` (v95_0), port 13339. Harvested 2026-06-13 → merged
absent-keys-only into `docs/packets/ida-exports/gms_v95.json` (354 → 395 keys).
**38/38 in-scope roster fnames resolved, 0 unresolved at the root** (giants
`CMob::Update` 0x654300 and `CUserLocal::TryDoingBodyAttack` 0x930710 harvested with
`--descent-depth 1 --ida-timeout 300s`; one descended-helper inside OnSuspendReset is an
indirect-call `Unresolved` marker — the root resolved). v95 has the BEST symbol coverage
of all five IDBs: the entire escort family + next-attack + attacked-by-mob are named.

## Registry state (1 edit needed)

- MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY: already correct (csv-import),
  IDA-verified — `CMob::OnMobSpeaking` @0x650000, `CMob::OnIncMobChargeCount` @0x63D500,
  `CMob::OnMobSkillDelay` @0x63D560. No change.
- Escort family already correctly fname'd (csv-import), all IDA-verified:
  - MOB_ESCORT_FULL_PATH (cb 304) `CMob::OnEscortFullPath` @0x643D90
  - MOB_ESCORT_STOP (cb 305) `CMob::OnEscortStopEndPermmision` @0x63B9C0
  - MOB_ESCORT_STOP_SAY (cb 306) `CMob::OnEscortStopSay` @0x64C500
  - MOB_ESCORT_RETURN_BEFORE (cb 307) `CMob::OnEscortReturnBefore` @0x649410
  - MOB_NEXT_ATTACK (cb 308) `CMob::OnNextAttack` @0x6528A0
  - MOB_ATTACKED_BY_MOB (cb 309) `CMob::OnMobAttackedByMob` @0x6436A0
  - MOB_ESCORT_COLLISION (sb 236) `CMob::SendCollisionEscort` @0x641150
  - MOB_REQUEST_ESCORT_INFO (sb 237) `CMob::SendRequestEscortPath` @0x6411F0
  - MOB_ESCORT_STOP_END_REQUEST (sb 238) `CMob::SendEscortStopEndRequest` @0x641290
  - **Naming reconciliation:** the plan's `MOB_ESCORT_RETURN_STOP`/`_STOP_SAY` ARE the
    registry's `MOB_ESCORT_STOP`/`_STOP_SAY` (OnEscortStopEndPermmision / OnEscortStopSay).
    No new rows; the existing v95 rows are canonical. Stage-2 codec struct names should
    align to MOB_ESCORT_STOP / MOB_ESCORT_STOP_SAY.
- MONSTER_BOOK_COVER (sb, 62): `""` → `CUserLocal::SetMonsterBookCover` (ida-discovered, 0x908DD0).

## Notable v95 deltas + sub-op demux (Stage 2 attention)

- **CMob::OnMobSkillDelay** reads 4×Decode4 (mob oid is the outer dispatcher; payload =
  4 ints) — present and named in v95/v87, ABSENT/unnamed in v83.
- **CMob::OnEscortStopEndPermmision** (MOB_ESCORT_STOP) is `QAEXXZ` — takes NO CInPacket,
  reads NOTHING. Wire payload = empty (just the opcode + mob oid from the pool dispatcher).
- **CMob::OnEscortFullPath** reads 8×Decode4 + Decode1 + Decode4 + Decode1 (path waypoints).
- **CMob::OnNextAttack** / **OnEscortReturnBefore** each read a single Decode4.
- **CField_MonsterCarnival::OnRequestResult** @0x55A890 demux on `bResult` arg
  (= the opcode): `!=0` SUMMON → Decode1, Decode1, DecodeStr; `==0` MESSAGE → Decode1
  (selector switch 1..5, strings from StringPool 0x101B..0x101F). Same as v83/v87.
- **CMob::Update** @0x654300 backs FIELD_DAMAGE_MOB (230), MOB_DAMAGE_MOB_FRIENDLY (231),
  MOB_SKILL_DELAY_END (234) — shared tick; per-op serverbound payloads come from the
  COutPacket build sites, not this read body.
- **CMobPool::OnMobCrcKeyChanged** backs MOB_CRC_KEY_CHANGED (cb) + REPLY (sb).

---

## Byte layouts

### CField_MonsterCarnival::OnEnter
- **address:** 0x55a6c0
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
- **address:** 0x55a2a0
- **calls (2):**
  - `Decode2` 
  - `Decode2` 

### CField_MonsterCarnival::OnProcessForDeath
- **address:** 0x55ab90
- **calls (3):**
  - `Decode1` 
  - `DecodeStr` 
  - `Decode1` 

### CField_MonsterCarnival::OnRequestResult
- **address:** 0x55a890
- **calls (4):**
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 
  - `Decode1` 

### CField_MonsterCarnival::OnShowGameResult
- **address:** 0x55af80
- **calls (1):**
  - `Decode1` 

### CField_MonsterCarnival::OnShowMemberOutMsg
- **address:** 0x55ad80
- **calls (3):**
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 

### CField_MonsterCarnival::OnTeamCP
- **address:** 0x55a2d0
- **calls (3):**
  - `Decode1` 
  - `Decode2` 
  - `Decode2` 

### CMob::OnAffected
- **address:** 0x644400
- **calls (2):**
  - `Decode4` 
  - `Decode2` 

### CMob::OnCatchEffect
- **address:** 0x63cd00
- **calls (2):**
  - `Decode1` 
  - `Decode1` 

### CMob::OnEffectByItem
- **address:** 0x63cd40
- **calls (2):**
  - `Decode4` 
  - `Decode1` 

### CMob::OnEscortFullPath
- **address:** 0x643d90
- **calls (11):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 
  - `Decode4` 
  - `Decode1` 

### CMob::OnEscortReturnBefore
- **address:** 0x649410
- **calls (1):**
  - `Decode4` 

### CMob::OnEscortStopEndPermmision
- **address:** 0x63b9c0
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::OnEscortStopSay
- **address:** 0x64c500
- **calls (7):**
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 
  - `Decode1` 
  - `DecodeStr` 
  - `Decode4` 
  - `Delegate`  -> _bstr_t::_bstr_t

### CMob::OnIncMobChargeCount
- **address:** 0x63d500
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnMobAttackedByMob
- **address:** 0x6436a0
- **calls (4):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 

### CMob::OnMobSkillDelay
- **address:** 0x63d560
- **calls (4):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 

### CMob::OnMobSpeaking
- **address:** 0x650000
- **calls (2):**
  - `Decode4` 
  - `Decode4` 

### CMob::OnNextAttack
- **address:** 0x6528a0
- **calls (1):**
  - `Decode4` 

### CMob::OnSpecialEffectBySkill
- **address:** 0x6540b0
- **calls (5):**
  - `Decode4` 
  - `Decode4` 
  - `Decode2` 
  - `Delegate`  -> SKILLENTRY::GetSpecialUOL
  - `Delegate`  -> _bstr_t::Data_t::Release

### CMob::OnSuspendReset
- **address:** 0x64acb0
- **calls (3):**
  - `Decode1` 
  - `Delegate`  -> IWzGr2DLayer::Getalpha
  - `Unresolved` packet var passed to unresolved/indirect call; hand-trace

### CMob::SendCollisionEscort
- **address:** 0x641150
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendDropPickUpRequest
- **address:** 0x644450
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendEscortStopEndRequest
- **address:** 0x641290
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SendRequestEscortPath
- **address:** 0x6411f0
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::SetDamagedByMob
- **address:** 0x64b260
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::TryFirstSelfDestruction
- **address:** 0x640ee0
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::Update
- **address:** 0x654300
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMob::UpdateTimeBomb
- **address:** 0x643c30
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x657230
- **calls (1):**
  - `Decode4` 

### CUIMonsterCarnival::RequestSend
- **address:** 0x80b4a0
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::SendBanMapByMobRequest
- **address:** 0x908d50
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::SetMonsterBookCover
- **address:** 0x908dd0
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CUserLocal::TryDoingBodyAttack
- **address:** 0x930710
- **calls (0):**
  - (no Decode/Encode calls captured — void handler or reads via descended helper)

### CWvsContext::OnBridleMobCatchFail
- **address:** 0x9d9a80
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCard
- **address:** 0x9ddcb0
- **calls (3):**
  - `Decode1` 
  - `Decode4` 
  - `Decode4` 

### CWvsContext::OnMonsterBookSetCover
- **address:** 0x9cfa70
- **calls (1):**
  - `Decode4` 

### CWvsContext::OnSetTamingMobInfo
- **address:** 0x9f7280
- **calls (5):**
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode4` 
  - `Decode1` 



---

## Stage-2 blockers (v95)

None — every in-scope v95 fname resolved in the IDB and merged into the export. The escort
family (v95-only) is fully named. The only caveats are the shared-function demuxes above
(OnRequestResult, CMob::Update), which Stage 2 must split per-op, and the serverbound
send-only functions (SendCollisionEscort/SendRequestEscortPath/SendEscortStopEndRequest/
SendDropPickUpRequest/etc.) whose Encode order may need hand-tracing (the Decode-focused
harvester captured few/no calls for Encode-side functions).
