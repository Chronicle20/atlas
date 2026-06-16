# gms_v84 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `GMS_v84.1_U_DEVM.exe` (v84_1), port 13341. Harvested 2026-06-13.

## TL;DR — v84 IDB family NAMED 2026-06-13; mostly v83-identical, with one divergence

task-083 established v84 ≡ v83 byte-wise (and v84 takes the v83 codec path via the
`MajorAtLeast(87)` gate — v84 < 87). The Stage-1 harvest originally resolved **only 1 of 25**
in-scope roster fnames because the v84 IDB symbolized the family as unnamed `sub_XXXX`.

**This pass renamed the in-scope MOB/MONSTER family in the v84 IDB by LAYOUT MATCH to its
v83 named twins** (recipe in `structures/naming-recipe.md`), then re-harvested:

```
export: 22 resolved, 3 descended-helper, 0 unresolved
```

Merged absent-keys-only (resolved-only) into `docs/packets/ida-exports/gms_v84.json`
(440 → 464 keys). The 22 newly-named functions + their 3 descended display/string helpers
now carry real byte layouts.

### One genuine v84 ≠ v83 divergence (IMPORTANT)

**v84 HAS `CMob::OnMobSkillDelay` (clientbound); v83 does NOT.** The v84 mob dispatcher
`CMobPool::OnMobPacket` @0x68FEF7 has **case 261 → sub_688524 = OnMobSkillDelay**
(`Decode4` time-adjusted via the v84 get_update_time-equivalent, then `Decode4`×3 =
tSkillDelayTime/nSkillID/nSLV/nOption), whereas the v83 dispatcher's switch ends at
`OnMobAttackedByMob`/0xFF with no skill-delay case. So for MOB_SKILL_DELAY (cb), v84 sits
with v87 (which also has it), NOT with v83 (which is version-absent). Every OTHER in-scope
op IS byte-identical to v83.

The escort/next-attack family is still ABSENT in v84 (the dispatcher cases stop at the
OnMobAttackedByMob equivalent; no OnEscort*/OnNextAttack), matching v83/v87.

## Dispatcher evidence

### CMobPool::OnMobPacket @ 0x68FEF7 (clientbound mob cluster)

`switch(a2)`. Renamed targets confirmed by Decode-layout match to the v83 twin:

| v84 case | v84 target (was) | v83 twin | layout match |
|---|---|---|---|
| 245 | CMob::OnMove | OnMove | (already named) |
| 246 | CMob::OnCtrlAck | OnCtrlAck | (already named) |
| 248 | CMob::OnStatSet | OnStatSet | (already named) |
| 249 | CMob::OnStatReset | OnStatReset | (already named) |
| 250 | CMob::OnSuspendReset | OnSuspendReset | (named earlier) |
| 251 | `sub_682977` → **OnAffected** | OnAffected | Decode4, Decode2 |
| 252 | CMob::OnDamaged | OnDamaged | (already named) |
| 253 | `sub_683BE9` → **OnSpecialEffectBySkill** | OnSpecialEffectBySkill | Decode4 + delegates |
| 256 | CMob::OnHPIndicator | OnHPIndicator | (already named) |
| 257 | `sub_6839BB` → **OnCatchEffect** | OnCatchEffect | Decode1 |
| 258 | `sub_683C9F` → **OnEffectByItem** | OnEffectByItem | Decode4, Decode1 |
| 259 | `sub_687743` → **OnMobSpeaking** | OnMobSpeaking | Decode4, Decode4 → TrySpeaking(helper) |
| 260 | `sub_687655` → **OnIncMobChargeCount** | OnIncMobChargeCount | Decode4→this[34], Decode4→this[35] |
| 261 | `sub_688524` → **OnMobSkillDelay** | (v83 ABSENT; matches v87) | Decode4(+time)×, Decode4×3 |
| 262 | `sub_68749A` → **OnMobAttackedByMob** | OnMobAttackedByMob | Decode1,Decode4,Decode4,Decode1 + display |

OnMobSpeaking vs OnIncMobChargeCount (both Decode4,Decode4 in v83) were disambiguated by
positional dispatch order AND store fingerprint: OnIncMobChargeCount writes this[34]/this[35]
directly (v83+v84 identical), OnMobSpeaking forwards both ints to a TrySpeaking-helper.

### CWvsContext::OnPacket @ 0xa51cd0 (book/taming/bridle sub-dispatch)

The v84 CWvsContext sub-table is **shifted +2** vs v83 for the monster-book/bridle handlers
(v84 inserted two `CNpcPool::OnPacket` forwarders at the old 0x53/0x54 slots). Targets matched
by Decode-layout + distinctive callee/StringPool fingerprint:

| v84 case | v84 target (was) | v83 case | v83 twin | match evidence |
|---|---|---|---|---|
| 0x30 | `sub_A748D8` → **OnSetTamingMobInfo** | 0x30 | OnSetTamingMobInfo | Decode4×4, Decode1 |
| 0x51 | `sub_A522FC` → **OnBridleMobCatchFail** | 0x4F | OnBridleMobCatchFail | Decode1,Decode4,Decode4 + GetBridleItem (sub_A52427) + SP 4353/4354 |
| 0x55 | `sub_A524A6` → **OnMonsterBookSetCard** | 0x53 | OnMonsterBookSetCard | Decode1,Decode4,Decode4 + SetMonsterCardCount (sub_99E951) + SP 2597/2598 |
| 0x56 | `sub_A525C3` → **OnMonsterBookSetCover** | 0x54 | OnMonsterBookSetCover | Decode4 + SetMonsterBookCover (sub_99E8EE) + CUIMonsterBook::UpdateUI |

### CField_MonsterCarnival::OnPacket @ 0x571FF5 (carnival cluster)

Same structure as v83 (8 cases + CField::OnPacket default; OnRequestResult serves SUMMON+MESSAGE).
The v84 carnival clientbound opcodes are 296–303 (vs v83 0x121–0x128). All matched by Decode-layout:

| v84 case | v84 target (was) | v83 twin | layout match |
|---|---|---|---|
| 296 | `sub_57209E` → **OnEnter** | OnEnter | Decode1, Decode2-burst, Decode1 |
| 297 | `sub_572215` → **OnPersonalCP** | OnPersonalCP | Decode2, Decode2 |
| 298 | `sub_572245` → **OnTeamCP** | OnTeamCP | Decode1, Decode2, Decode2 |
| 299/300 | `sub_572284` → **OnRequestResult** | OnRequestResult | SUMMON: Decode1,Decode1,DecodeStr / MESSAGE: Decode1 + SP 4085-4089 |
| 301 | `sub_5724EE` → **OnProcessForDeath** | OnProcessForDeath | Decode1, DecodeStr, Decode1 |
| 302 | `sub_572669` → **OnShowMemberOutMsg** | OnShowMemberOutMsg | Decode1, Decode1, DecodeStr |
| 303 | `sub_5727E4` → **OnShowGameResult** | OnShowGameResult | Decode1 |

`CUIMonsterCarnival::RequestSend` (serverbound carnival CP request) = `sub_89BDDA` (0x99 bytes,
matches v83 0x99): `CanSendExclRequest(500,0)` → `COutPacket(0xE0)` → `Encode1`, `Encode4(x-1)`
→ SendPacket → sets the excl-request cooldown. v84 opcode 0xE0 (v83 was 0xDA; +6 shift).

## Serverbound senders

| op | v84 fn | status |
|---|---|---|
| CMob::Update (FIELD_DAMAGE_MOB / friendly / skill-delay-end shared tick) | 0x67d4ea | **NAMED** (was custom label `CMob__Update_ctrl_send_0xC4_0xC5friendlyDmg_0xC8`; 0x16ac virtual). Harvested. |
| CMob::SetDamagedByMob | 0x6871bc | **NAMED** (was custom label `CMob__SetDamagedByMob_send_0xC7_mobDmgMob`; 0x2de ≈ v83 0x2df). Harvested. |
| CMob::TryFirstSelfDestruction | — | UNRESOLVED — unnamed CMob sub, no anchor symbol in v84 IDB. Inherit v83 layout. |
| CMob::SendDropPickUpRequest | — | UNRESOLVED — unnamed CMob sub, no anchor. Inherit v83. |
| CMob::UpdateTimeBomb | — | UNRESOLVED — inlined into Update@CMob in this build (as in v83/v87). Inherit v83. |
| CUserLocal::TryDoingBodyAttack | — | UNRESOLVED — the entire CUserLocal cluster is unnamed in v84 (no Update@CUserLocal / TryDoing* anchors); not confidently locatable without building the namespace from scratch. Inherit v83. |
| CUserLocal::SendBanMapByMobRequest | — | UNRESOLVED — same (CUserLocal cluster unnamed). Inherit v83. |
| CUserLocal::SetMonsterBookCover (MONSTER_BOOK_COVER sb) | — | UNRESOLVED — send-site unnamed; CUserLocal cluster unnamed. fname stays `""`. Inherit v83. |

These remaining senders are serverbound (harvest as "0 calls" even when named) and the no-guess
rule forbids labelling them without a reliable anchor. They are v83-equivalent (v84 ≡ v83 wire),
so Stage 2 inherits the v83 send layouts.

---

## Byte layouts (in-scope, from the merged export)

### CField_MonsterCarnival::OnEnter
- **address:** 0x57209e
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
- **address:** 0x572215
- **calls (2):**
  - `Decode2`
  - `Decode2`

### CField_MonsterCarnival::OnProcessForDeath
- **address:** 0x5724ee
- **calls (5):**
  - `Decode1`
  - `DecodeStr`
  - `Decode1`
  - `Delegate` -> sub_447019
  - `Delegate` -> sub_447019

### CField_MonsterCarnival::OnRequestResult
- **address:** 0x572284
- **calls (4):**
  - `Decode1`
  - `Decode1`
  - `Decode1`
  - `DecodeStr`

### CField_MonsterCarnival::OnShowGameResult
- **address:** 0x5727e4
- **calls (1):**
  - `Decode1`

### CField_MonsterCarnival::OnShowMemberOutMsg
- **address:** 0x572669
- **calls (4):**
  - `Decode1`
  - `Decode1`
  - `DecodeStr`
  - `Delegate` -> sub_447019

### CField_MonsterCarnival::OnTeamCP
- **address:** 0x572245
- **calls (3):**
  - `Decode1`
  - `Decode2`
  - `Decode2`

### CMob::OnAffected
- **address:** 0x682977
- **calls (2):**
  - `Decode4`
  - `Decode2`

### CMob::OnCatchEffect
- **address:** 0x6839bb
- **calls (1):**
  - `Decode1`

### CMob::OnEffectByItem
- **address:** 0x683c9f
- **calls (2):**
  - `Decode4`
  - `Decode1`

### CMob::OnIncMobChargeCount
- **address:** 0x687655
- **calls (2):**
  - `Decode4`
  - `Decode4`

### CMob::OnMobAttackedByMob
- **address:** 0x68749a
- **calls (4):**
  - `Decode1`
  - `Decode4`
  - `Decode4`
  - `Decode1`

### CMob::OnMobSkillDelay
- **address:** 0x688524
- **calls (4):**
  - `Decode4`
  - `Decode4`
  - `Decode4`
  - `Decode4`

### CMob::OnMobSpeaking
- **address:** 0x687743
- **calls (2):**
  - `Decode4`
  - `Decode4`

### CMob::OnSpecialEffectBySkill
- **address:** 0x683be9
- **calls (3):**
  - `Decode4`
  - `Delegate` -> sub_411344
  - `Delegate` -> sub_402C9B

### CMob::SetDamagedByMob
- **address:** 0x6871bc
- **calls (0):**
  - (no Encode calls captured — serverbound; COutPacket build below descent boundary. Layout = v83 twin.)

### CMob::Update
- **address:** 0x67d4ea
- **calls (0):**
  - (no Encode calls captured — serverbound shared tick; per-op send payloads derive from COutPacket build sites, NOT this read body. Layout = v83 twin.)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x690354
- **calls (1):**
  - `Decode4`

### CUIMonsterCarnival::RequestSend
- **address:** 0x89bdda
- **calls (0):**
  - (no Encode calls captured — serverbound; COutPacket(0xE0), Encode1(nType), Encode4(idx-1). Layout = v83 twin.)

### CWvsContext::OnBridleMobCatchFail
- **address:** 0xa522fc
- **calls (3):**
  - `Decode1`
  - `Decode4`
  - `Decode4`

### CWvsContext::OnMonsterBookSetCard
- **address:** 0xa524a6
- **calls (3):**
  - `Decode1`
  - `Decode4`
  - `Decode4`

### CWvsContext::OnMonsterBookSetCover
- **address:** 0xa525c3
- **calls (1):**
  - `Decode4`

### CWvsContext::OnSetTamingMobInfo
- **address:** 0xa748d8
- **calls (5):**
  - `Decode4`
  - `Decode4`
  - `Decode4`
  - `Decode4`
  - `Decode1`

---

## Stage-2 blockers (v84 IDB naming — residual)

| scope | status |
|---|---|
| 8 CMob clientbound handlers + 4 CWvsContext + 7 carnival + RequestSend | **RESOLVED 2026-06-13** (named + harvested, 0 unresolved). |
| CMob::Update, CMob::SetDamagedByMob | **RESOLVED** (custom labels → canonical mangled names + harvested). |
| CMob::TryFirstSelfDestruction, CMob::SendDropPickUpRequest, CMob::UpdateTimeBomb | UNRESOLVED — unnamed CMob subs / inlined; no anchor. v83-equivalent. |
| CUserLocal::TryDoingBodyAttack, SendBanMapByMobRequest, SetMonsterBookCover | UNRESOLVED — CUserLocal cluster fully unnamed in v84 (no anchor symbols). v83-equivalent. |
| MOB_SKILL_DELAY (cb) | NOTE: present in v84 (case 261), unlike v83. Now named + harvested. |
