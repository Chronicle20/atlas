# GMS v79 — packet delta vs v83 (Stage A)

> **Source-of-truth delta doc for the v79 pass.** Stage B (registry), Stage C
> (template), and Stage E (verification campaign) consume this. Anchor =
> `gms_v83`. Every opcode/mode/structure claim below cites the v79 IDB
> (function name + address, switch-case) or explicit v83-anchor evidence.

## IDA pre-flight (re-confirmed by binary name)

| Role | Port | Binary (confirmed via `list_instances`) |
|---|---|---|
| v79 target | 13340 | `GMS_v79_1_DEVM.exe` |
| v83 anchor | 13342 | `MapleStory_dump.exe` |
| v95 tie-breaker | 13341 | `GMS_v95.0_U_DEVM.exe` |

All three present and reachable. v79 IDB uses **mangled MSVC symbols** (e.g.
`?OnPacket@CWvsContext@@QAEXJAAVCInPacket@@@Z`); names below are the demangled
forms. Addresses are v79 unless prefixed `v83:`.

## ⚠️ Critical input finding — the ClientBound CSV `GMS v79` column is empty

The brief states the ClientBound CSV "HAS a `GMS v79` column → the v79
clientbound opcode map is partially given." **It does not.** All **586/586**
rows carry the literal placeholder `0x000` in the `GMS v79` hex column
(`docs/packets/MapleStory Ops - ClientBound.csv`, column index 11). The decimal
column that follows it mirrors the **v83** value, not v79. There is **no real
v79 opcode data in the CSV** to reconcile against. The ServerBound CSV likewise
has no v79 column. **Every v79 opcode in this document is derived from the v79
IDB switch tables.** Stage B must treat the CSV v79 column as absent (seed v79
from this doc / IDB, not from the CSV).

---

## Top-level routing (the shim and the dispatcher windows)

`CClientSocket::ProcessPacket` @ `0x48e209` (`CInPacket::Decode2` scrutinee):

```
case 0x10: CClientSocket::OnMigrateCommand     case 0x13: OnAuthenMessage
case 0x11: OnAliveReq                          case 0x14: CSecurityClient::OnPacket
case 0x12: OnAuthenCodeChanged
default: if (op in [0x1A,0x75]) -> CWvsContext::OnPacket(op)
         else -> current-stage vtable+8 (CLogin / CStage subclass)::OnPacket(op)
```

The CWvsContext window is **`[0x1A,0x75]` (26–117)** in v79 vs **`[0x1D,0x7C]`
(29–124)** in v83 (v83 `ProcessPacket`, registry note "0x1D–0x7C range"). Lower
bound shifts −3, upper bound −7 → ops were inserted in v83 inside this window.
This non-uniform shift recurs throughout: **there is no single global offset; the
delta grows as opcode rises.** Each dispatcher below is mapped fname-by-fname.

---

## (e) usesPin (OQ-2) — **false** for v79

- Anchor: `template_gms_83_1.json` line 5 → `"usesPin": false`.
- v79 evidence: `CLogin::OnPacket` @ `0x5cd229` keeps `OnCheckPinCodeResult`
  (case 6) and `OnUpdatePinCodeResult` (case 7) **identically to v83**, and it
  **lacks** the second-password cases v83 added — v83 case `0x17`
  `OnEnableSPWResult` (LOGIN_AUTH) and case `0x1C` `OnCheckSPWResult`
  (CHECK_SPW_RESULT) have **no case in v79's switch**. v79 is therefore no more
  PIN/SPW-dependent than v83. **`usesPin` carries as `false`.**

---

## (f) Login-flow divergence (OQ-3) — biggest connect risk

`CLogin::OnPacket` v79 @ `0x5cd229` vs v83 @ `0x5f80ff` (both decompiled). The
login opcode layout **diverges structurally** — this is not a uniform shift.

| op | v79 handler (addr) | v83 handler | Δ / note |
|---|---|---|---|
| 0 | `OnCheckPasswordResult` | `OnCheckPasswordResult` (LOGIN_STATUS) | same |
| 1 | `OnGuestIDLoginResult` | `OnGuestIDLoginResult` | same |
| 2 | **— (no case)** | `OnAccountInfoResult` (ACCOUNT_INFO) | **v79 ABSENT** |
| 3 | `sub_5CE217` = OnCheckUserLimitResult | `OnCheckUserLimitResult` (SERVERSTATUS) | same op |
| 4 | `OnSetAccountResult` | `OnSetAccountResult` (GENDER_DONE) | same |
| 5 | `OnConfirmEULAResult` | `OnConfirmEULAResult` | same |
| 6 | `OnCheckPinCodeResult` | `OnCheckPinCodeResult` | same |
| 7 | `OnUpdatePinCodeResult` | `OnUpdatePinCodeResult` | same |
| 8 | `OnViewAllCharResult` | `OnViewAllCharResult` | same |
| 9 | `OnSelectCharacterByVACResult` | `OnSelectCharacterByVACResult` | same |
| 10 | `sub_5CE248` = OnWorldInformation | `OnWorldInformation` (WORLD_INFORMATION) | same op |
| 11 | `sub_5CE522` = char-list decode | `OnSelectWorldResult` (CHARLIST) | same op |
| 12 | `OnSelectCharacterResult` | `OnSelectCharacterResult` (SERVER_IP) | same |
| 13 | `OnCreateNewCharacterResult` (symbol) — body = CHAR_NAME_RESPONSE | `OnCheckDuplicatedIDResult` (CHAR_NAME_RESPONSE) | **same by body** (see CORRECTION below) |
| 14 | `OnDeleteCharacterResult` (symbol) — body = ADD_NEW_CHAR_ENTRY | `OnCreateNewCharacterResult` (ADD_NEW_CHAR_ENTRY) | **same by body** |
| 15 | `OnCheckDuplicatedIDResult` (symbol) — body = DELETE_CHAR_RESPONSE | `OnDeleteCharacterResult` (DELETE_CHAR_RESPONSE) | **same by body** |
| 22 | `OnSelectWorldResult` @0x5cf9ea (relog-to-title) | `sub_5FB83D` (RELOG_RESPONSE) | distinct |
| 23 / 26 / 27 / 28 | **— (no case)** | `OnEnableSPWResult` / `OnLatestConnectedWorld` / `OnRecommendWorldMessage` / `OnCheckSPWResult` | **v79 ABSENT** |

**Connect-critical deltas:**

1. **ACCOUNT_INFO (op 2) is absent in v79.** No `OnAccountInfoResult` case.
2. ~~**Char-management ops 13/14/15 are PERMUTED** (the load-bearing one):~~
   - ~~`CHAR_NAME_RESPONSE` (OnCheckDuplicatedIDResult): **v79 = 15**, v83 = 13~~
   - ~~`ADD_NEW_CHAR_ENTRY` (OnCreateNewCharacterResult): **v79 = 13**, v83 = 14~~
   - ~~`DELETE_CHAR_RESPONSE` (OnDeleteCharacterResult): **v79 = 14**, v83 = 15~~

   > ### ⚠️ CORRECTION (task-113 Stage E) — the "PERMUTED" finding was FALSE
   >
   > The permutation above was derived by trusting the v79 IDB **symbol names**
   > on the `CLogin::OnPacket` switch cases. Those symbol names are **rotated one
   > step off their actual handler bodies** in the v79 IDB. Decompiling the three
   > handler BODIES (the wire truth) shows v79 char-management is **IDENTICAL to
   > v83** — no permutation:
   >
   > | v79 op | case-symbol (IDB) | body read-order (decompiled) | packet = |
   > |---|---|---|---|
   > | **13** | `OnCreateNewCharacterResult` @`0x5ce875` | `DecodeStr + Decode1` | **CHAR_NAME_RESPONSE** (name-check) |
   > | **14** | `OnDeleteCharacterResult` @`0x5ceb55` | `Decode1 + GW_CharacterStat::Decode + AvatarLook::Decode` | **ADD_NEW_CHAR_ENTRY** |
   > | **15** | `OnCheckDuplicatedIDResult` @`0x5ce90a` | `Decode4 + Decode1` + slot-removal | **DELETE_CHAR_RESPONSE** |
   >
   > So by handler-body behavior: **CHAR_NAME_RESPONSE=13, ADD_NEW_CHAR_ENTRY=14,
   > DELETE_CHAR_RESPONSE=15 — the same as v83.** The registry (`gms_v79.yaml`)
   > and `template_gms_79_1.json` were corrected to these opcodes; the earlier
   > Add=13/Delete=14/Name=15 mapping would have crashed the client (ADD's
   > stat+avatar payload sent to op 13, whose handler reads a leading string).
3. **Absent vs v83:** `LOGIN_AUTH` (23), `LAST_CONNECTED_WORLD` (26),
   `RECOMMENDED_WORLD_MESSAGE` (27), `CHECK_SPW_RESULT` (28) — none dispatched.
4. CLogin's default forwards **118–120 → CStage::OnPacket** only (v83 forwards
   `0x7D–0x7F` → CStage and `0x80–0x82` → CMapLoadable). v79's
   SET_FIELD/SET_ITC/SET_CASH_SHOP live at **118/119/120** (v83 125/126/127).

**Handshake / encoding:**
- `LOGIN_PASSWORD` send `CLogin::SendCheckPasswordPacket` @ `0x5cbf50` builds
  `COutPacket(1)` then `EncodeStr(id) + EncodeStr(pw) + EncodeBuffer(machineId,16)
  + Encode4(gameRoomClient) + Encode1(b) + Encode1(0) + Encode1(0) +
  Encode4(partnerCode)`. Opcode **1 = v83 (Δ0)**; body shape matches the v83
  CheckPassword send.
- **CHARLIST body** (`sub_5CE522`, op 11): `Decode1 status`; on status∈{0,12,23}
  `Decode1 charCount`, then per slot (≤12): `GW_CharacterStat::Decode` +
  `AvatarLook::Decode` + `Decode1 rankEnabled` + (`Decode1` ? `DecodeBuffer(16)`
  : zero), then `Decode4`. Entry shape = `{ CharacterStat, AvatarLook,
  rankFlag, [16B rank] }` — **same as the v83 CHARLIST entry** (the opcode is
  also 11 in both); only the surrounding login opcodes were renumbered.
- `WORLD_INFORMATION` (`sub_5CE248`, op 10): worldId + name + channels list +
  balloon list — classic world-info body; opcode unchanged (10).

---

## (a) Opcode map — clientbound (writer) by dispatcher

Mapping = (v79 switch case) → (v83 op via the registry `gms_v83.yaml` fname).
"Δ" is `v79 − v83`. Each block cites its dispatcher fn+addr; the case list is
the switch evidence. Within a block the delta is uniform unless flagged.

### CWvsContext::OnPacket @ `0x953800` (ops 26–117)

| v79 run | fnames (first→last) | v83 run | Δ |
|---|---|---|---|
| 26–40 | OnInventoryOperation … OnAntiMacroResult | 29–43 | **−3** |
| 42–50 | OnClaimResult … OnEntrustedShopCheckResult | 45–53 | −3 |
| 48,49,50 | `sub_969022`(=OnSkillLearnItemResult), OnGatherItemResult, OnSortItemResult | 51,52,53 | −3 |
| 52,54,55 | OnSueCharacterResult, OnTradeMoneyLimit, OnSetGender | 55,57,58 | −3 |
| 56 | `CUIGuildBBS::OnGuildBBSPacket` (GUILD_BBS_PACKET) | 59 | −3 |
| 58–82 | OnCharacterInfo … OnHourChanged | 61–85 | −3 |
| 83 | `sub_95E24B` (= OnMiniMapOnOff, MINIMAP_ON_OFF) | 86 | −3 |
| 84–86 | OnPartyValue, OnFieldSetVariable, OnBonusExpRateChanged | 91,92,93 | **−7** |
| 87 | `OnPotionDiscountRateChanged` | — | **v79-EXTRA** (no v83 op) |
| 88–101 | OnFamilyChartResult … OnNotifyJobChange | 94–107 | **−6** |
| 103–114 | OnMapleTVUseRes … OnCancelNameChangebyOther | 109–120 | −6 |
| 115,116 | `sub_95EE37`(=OnSetBuyEquipExt), `sub_95F0D4`(=OnScriptProgressMessage) | 121,122 | −6 |
| 117 | OnMacroSysDataInit (MACRO_SYS_DATA_INIT) | 124 | **−7** |

**Absent in v79** (present in v83 CWvsContext): `CONSULT_AUTHKEY_UPDATE` (87),
`CLASS_COMPETITION_AUTHKEY_UPDATE` (88), `WEB_BOARD_AUTHKEY_UPDATE` (89),
`SESSION_VALUE` (90) — the −3→−7 jump after op 83; and `DATA_CRC_CHECK_FAILED`
(123) — the −6→−7 jump before op 117.
**v79-extra:** `OnPotionDiscountRateChanged` (v79 op 87) has no registry/v83 op.

### CStage::OnPacket @ `0x6f079f` (ops 118–120) — Δ **−7**
118 OnSetField (SET_FIELD→125) · 119 OnSetITC (SET_ITC→126) · 120 OnSetCashShop
(SET_CASH_SHOP→127).

### CMapLoadable::OnPacket @ `0x614406` (ops 121–122) — Δ **−7**
121 OnSetBackEffect (SET_BACK_EFFECT→128) · 122 `sub_614977`(=OnSetMapObjectVisible→129).
**`CLEAR_BACK_EFFECT` (v83 130) is ABSENT in v79** — the v79 switch has only two
cases; op 123 is already BLOCKED_MAP in CField. (Δ steps −7→−8 here.)

### CField::OnPacket @ `0x51c90f` (base ops 123–148) — Δ **−8**

123 OnTransferFieldReqIgnored (BLOCKED_MAP→131) · 124 OnTransferChannelReqIgnored
(132) · 125 OnFieldSpecificData (133) · 126 OnGroupMessage (MULTICHAT→134) ·
127 OnWhisper (WHISPER→135) · 128 OnCoupleMessage (SPOUSE_CHAT→136) ·
129 OnSummonItemInavailable (137) · **130 OnFieldEffect (FIELD_EFFECT→138)** ·
131 OnFieldObstacleOnOff (139) · 132 OnFieldObstacleOnOffStatus (140) ·
133 OnFieldObstacleAllReset (141) · 134 OnBlowWeather (142) · 135 OnPlayJukeBox
(143) · 136 OnAdminResult (144) · 137 OnQuiz (145) · 138 OnDesc (146) ·
139 OnClock-vtable (CLOCK→147) · 142 OnSetQuestClear (150) · 143 OnSetQuestTime
(151) · 144 OnWarnMessage (ARIANT_RESULT→152) · 145 OnSetObjectState (153) ·
146 OnDestroyClock (STOP_CLOCK→154) · 148 `sub_522DC3`(=OnStalkResult, IDA_0X09C→156).

**Absent in v79 base CField switch** (v83 CField subclass/special ops with no v79
base case): `CONTI_MOVE` (148), `CONTI_STATE` (149), `ARIANT_ARENA_SHOW_RESULT`
(155), `PYRAMID_GAUGE` (157), `PYRAMID_SCORE` (158) — these are
`CField_*`-subclass overrides; not verified for v79 in this stage (Stage B should
check `CField_ContiMove` / `CField_Massacre` subclass `OnPacket` overrides).

### CUserPool::OnPacket @ `0x8c8904` — enter/leave Δ **−11**
149 OnUserEnterField (SPAWN_PLAYER→160) · 150 OnUserLeaveField
(REMOVE_PLAYER_FROM_MAP→161). Routes 151–170→common, 171–190→remote,
191–212→local.

### CUserPool::OnUserCommonPacket @ `0x8c8c79` (ops 151–169) — Δ **−11**
151 OnChat (CHATTEXT→162) · 152 OnChat (CHATTEXT1→163) · 153 OnADBoard
(CHALKBOARD→164) · 154 OnMiniRoomBalloon (UPDATE_CHAR_BOX→165) ·
155 SetConsumeItemEffect (166) · 156 ShowItemUpgradeEffect (167) ·
**157–163 OnPetPacket** (SPAWN_PET 168 … PET_COMMAND 174) ·
**164–169 `sub_892500`** (summon cluster: SPAWN_SPECIAL_MAPOBJECT 175 …
SUMMON_SKILL 180).

### CUserPool::OnUserRemotePacket @ `0x8c8d4a` (ops 171–190) — Δ **−14**
171 OnMove (**MOVE_PLAYER→185**) · 172–175 OnAttack (CLOSE_RANGE/RANGED/MAGIC/
ENERGY 186–189) · 176 OnSkillPrepare (SKILL_EFFECT→190) · 177 OnSkillCancel
(191) · 178 OnHit (DAMAGE_PLAYER→192) · 179 CAvatar::SetEmotion
(FACIAL_EXPRESSION→193) · 180 SetActiveEffectItem (194) · 181
OnShowUpgradeTombEffect (195) · 182 SetActivePortableChair (SHOW_CHAIR→196) ·
183 OnAvatarModified (UPDATE_CHAR_LOOK→197) · 184 OnEffect (SHOW_FOREIGN_EFFECT→
198) · 185 OnSetTemporaryStat (GIVE_FOREIGN_BUFF→199) · 186 OnResetTemporaryStat
(200) · 187 OnReceiveHP (UPDATE_PARTYMEMBER_HP→201) · 188 OnGuildNameChanged
(202) · 189 OnGuildMarkChanged (203) · 190 OnThrowGrenade (THROW_GRENADE→204).
**Absent in v79:** `SPAWN_DRAGON` (181) / `MOVE_DRAGON` (182) / `REMOVE_DRAGON`
(183) — the remote switch is gap-free at −14 with no dragon arm (dragons are
Evan/v84+; reserved-but-listed in the v83 registry).

### CUserLocal::OnPacket @ `0x8a8d4b` (ops 191–212)
191 OnSitResult (CANCEL_CHAIR→205, Δ−14) · 192 OnEffect (SHOW_ITEM_GAIN_INCHAT→
206) · 193 OnTeleport (DOJO_WARP_UP→207) · 195 OnMesoGive_Succeeded
(LUCKSACK_PASS→208, Δ−13) · 196 OnMesoGive_Failed (209) · 197 OnQuestResult
(UPDATE_QUEST_INFO→211, Δ−14) · 198 `OnNotifyHPDecByField` (**v79-extra**, no v83
registry op) · 199 `nullsub_12` (dead) · 200 OnBalloonMsg (PLAYER_HINT→214,
Δ−14) · 201 OnPlayEventSound (215) · 202 OnPlayMinigameSound (216) ·
203 OnMakerResult (MAKER_RESULT→217) · 205 OnOpenClassCompetitionPage
(KOREAN_EVENT→219) · 206 OnSetDirectionMode (LOCK_UI→221, **Δ−15**) ·
207 OnSetStandAloneMode (DISABLE_UI→222) · 208 OnHireTutor (SPAWN_GUIDE→223) ·
209 OnTutorMsg (TALK_GUIDE→224) · 210 OnIncComboResponse (SHOW_COMBO→225) ·
211 OnRandomEmotion (RANDOM_EMOTION→226) · 212 OnSkillCooltimeSet (**COOLDOWN→
234, Δ−22**).

**Absent in v79** (verified via `func_query` — these symbols do **not exist** in
the v79 IDB): `OnOpenUI` (OPEN_UI 220), `OnResignQuestReturn` (227),
`OnPassMateName` (228), `OnRadioSchedule` (229), `OnOpenSkillGuide` (230),
`OnNoticeMsg` (NOTICE_MSG 231), `OnChatMsg` (CHAT_MSG 232), `OnBuffzoneEffect`
(BUFFZONE_EFFECT 233). These are post-v79 additions; their absence is why the
local-packet delta steps −14 → −15 (no OPEN_UI) → −22 (the seven 227–233 ops).

### CMobPool::OnPacket @ `0x646ce7` (ops 214–233) — Δ **−22**
214 OnMobEnterField (SPAWN_MONSTER→236) · 215 OnMobLeaveField (237) · 216
OnMobChangeController (238) · 217–233 OnMobPacket (MOVE_MONSTER 239 …
MOB_ATTACKED_BY_MOB 255) · 227 OnMobCrcKeyChanged (MOB_CRC_KEY_CHANGED→249,
carved out).

### CNpcPool::OnPacket @ `0x6687e5` (ops 235–241) — Δ **−22**
235 OnNpcEnterField (SPAWN_NPC→257) · 236 OnNpcLeaveField (258) · 237
OnNpcChangeController (259) · 238–240 OnNpcPacket (NPC_ACTION 260 …
NPC_SPECIAL_ACTION 262) · 241 `sub_668A2D`(=OnSetNpcScript, SET_NPC_SCRIPTABLE→
263). (Cases 78/79 are dead duplicates of CWvsContext OnImitatedNPCData /
OnLimitedNPCDisableInfo, mirroring the v83 dead-case quirk — not real dispatch.)

### Map pools (range-routed from CField) — all Δ **−22**
Derived by range arithmetic against the registry (dispatcher ranges read from
CField::OnPacket; per-pool leaf order matches v83):
- CEmployeePool 243–245 → SPAWN/DESTROY/UPDATE_HIRED_MERCHANT 265–267
- CDropPool 246–247 → DROP_ITEM_FROM_MAPOBJECT / REMOVE_ITEM_FROM_MAP 268–269
- CMessageBoxPool 248–250 → CANNOT_SPAWN_KITE / SPAWN_KITE / REMOVE_KITE 270–272
- CAffectedAreaPool 251–252 → SPAWN_MIST / REMOVE_MIST 273–274
- CTownPortalPool 253–254 → SPAWN_DOOR / REMOVE_DOOR 275–276
- CReactorPool 255–258 → REACTOR_HIT / MOVE / SPAWN / DESTROY 277–280

### CCashShop::OnPacket @ `0x471da6` (ops 301–309)
301 OnChargeParamResult · 302 OnQueryCashResult · 303 OnCashItemResult ·
304 OnPurchaseExpChanged · 305 OnCheckDuplicatedIDResult · 306
OnCheckNameChangePossibleResult · 308 OnCheckTransferWorldPossibleResult ·
309 OnCashShopGachaponStampResult. Cash-shop is a separate stage outside the
core login→map flow; the v83 CCashShop opcodes were **not** cross-decompiled in
Stage A — Stage B must map 301–309 against v83 `CCashShop::OnPacket` (checked,
not assumed).

---

## (a) Opcode map — serverbound (handler) — derived from v79 send-sites

No CSV v79 serverbound column exists; opcodes are read from the `COutPacket(N)`
constructor at each client send-site. The serverbound space is its own
enumeration with its own non-uniform shift (login region Δ0, mid region Δ−2,
social region Δ−3). Core-flow anchors swept:

| op (serverbound) | v79 send-site (addr) | v79 opcode | v83 op | Δ |
|---|---|---|---|---|
| LOGIN_PASSWORD | `CLogin::SendCheckPasswordPacket` `0x5cbf50` | `COutPacket(1)` | 1 | **0** |
| CHANGE_MAP | `CField::SendTransferFieldRequest` `0x51b950` | `COutPacket(36)` | 38 | **−2** |
| CHANGE_CHANNEL | `CField::SendTransferChannelRequest` `0x51baa2` | `COutPacket(37)` | 39 | **−2** |
| MOVE_PLAYER | `sub_91B6E6` (CVecCtrlUser flush, after `CVecCtrlUser::OnSit` `0x91b6bf`; via `xrefs_to(CMovePath::Flush 0x657924)`) `0x91b883` | `COutPacket(39)` | 41 | **−2** |
| NPC_TALK | `CUserLocal::TalkToNpc` `0x8b7e10` | `COutPacket(56)` | 58 | **−2** |
| WHISPER | `CField::SendChatMsgWhisper` `0x51a7bc` | `COutPacket(117)` | 120 | **−3** |
| PARTY_OPERATION | `CField::SendCreateNewPartyMsg` `0x51b318` | `COutPacket(121)` | 124 | **−3** |

Body shapes verified at the send-sites: CHANGE_MAP =
`Enc1(portalByte)+Enc4(targetMap)+EncStr(portal)+Enc2(x)+Enc2(y)+Enc1(0)+
Enc1(wheel)+Enc1(premium?[+Enc4+Enc4])`; CHANGE_CHANNEL = `Enc1(channel)+
Enc4(t)`; NPC_TALK = `Enc4(npcOid)+Enc2(x)+Enc2(y)`; WHISPER = `Enc1(mode)+
EncStr(target)+EncStr(text)` (mode `(found+1)|4` for whisper, `0x86` for
find-buddy locate). These match the v83 serverbound bodies for the same ops.

> **MOVE_PLAYER serverbound = opcode 39 (verified in Stage A review).** The v79
> movement send flush is the unnamed `sub_91B6E6`, reachable in one `xrefs_to`
> hop from the named `CMovePath::Flush` @ `0x657924` (it sits directly after
> `CVecCtrlUser::OnSit` @ `0x91b6bf`). At `0x91b883` it builds `COutPacket(39)`
> then `Encode1(field+276)+Encode4(field+483)` and calls `CMovePath::Flush`.
> v83 MOVE_PLAYER sb = 41 → **Δ−2**, consistent with the mid-region shift. This
> value is verified (not a prediction); Stage B may seed it directly.

The full serverbound opcode table is a **Stage B deliverable** (derive each op
from its v79 send-site, anchored on the v83 serverbound FName). Stage A
establishes the method and the core-flow anchors.

---

## (b) Operation / mode (sub-op) tables — extracted from v79 dispatcher switches

These are the version-dependent mode-byte tables (`bug_operations_mode_tables_*`,
`bug_v83_status_message_operations_off_by_one`). Read **from the v79 switch**;
do not inherit v83's values.

### Status-message — `CWvsContext::OnMessage` @ `0x96ade7` (v79 op 36 / SHOW_STATUS_INFO)
Leading `Decode1` mode; switch arms **0,1,2,3,4,5,6,7,8,9,10,11,12,13**
(contiguous, 14 arms, default = drop). The **count and contiguity match the v83
client `OnMessage` switch (0–13)**. Stage C consequence: the v79 `operations`
table for SHOW_STATUS_INFO must mirror v83's IDA 0–13 with **no spurious
`INCREASE_SKILL_POINT`** (the same off-by-one trap that produced the v83 spawn
crash applies to v79).

### World/broadcast message — `CWvsContext::OnBroadcastMsg` @ `0x96c94f` (v79 op 65 / SERVERMESSAGE)
Leading `Decode1` type; arms **0–13** (Notice 0; speaker variants; type 4 =
management speaker, reads an extra `Decode1` whisper-flag; type 8 / 11 decode a
trailing `GW_ItemSlotBase` item-speaker payload; type 6 = `Decode4`; type 7 =
util-dlg; types 12/13 = pink/notice). 14-arm table — same shape family as v83
worldmessage. Stage E should byte-confirm arms 4/8/11 (item-speaker).

### Party — `CWvsContext::OnPartyResult` @ `0x987583` (v79 op 59 / PARTY_OPERATION)
Leading `Decode1` mode. v79 arms present (hex): **4, 7, 8, 9, 0xA, 0xC, 0xD,
0xF, 0x10, 0x11, 0x12, 0x13, 0x15, 0x16, 0x17, 0x19, 0x1B, 0x1C, 0x1D, 0x1E,
0x20, 0x21, 0x22, 0x23, 0x24, 0x25** (+ default). Notable bodies:
- `4` = invite received → `Decode4(partyId)+DecodeStr(inviter)+Decode1(level)`;
  builds the YESNO + the serverbound replies `COutPacket(121)` (accept,
  `Enc1(3)+Enc4+Enc1`) and `COutPacket(122)` (block/report).
- `7`/`0x22` = load/refresh → `Decode4(partyId)+PARTYDATA::Decode`.
- `8` = join → `Decode4 + Decode4 + Decode4 + Decode2 + Decode2 + ...`.
- `0xC` = leave/expel → `Decode4 + Decode4 + Decode1 + …`.
- `0xF` = join member → `Decode4 + DecodeStr + PARTYDATA::Decode`.
- `0x25` = HP/coord update slot → `Decode1(slot<6)+Decode4+Decode4+Decode2+
  Decode2` (throws `CDisconnectException` if slot≥6).

These mode values are the **v79** party table; Stage E must diff each against the
v83 `OnPartyResult` switch and the `template_gms_83_1.json` party `operations`
map (not assumed equal — party modes shift between versions).

### Other dispatcher families with their own mode tables (located, not yet byte-extracted)
`CWvsContext::OnGuildResult` @ `0x98099f` (op 62) · `CWvsContext::OnFriendResult`
@ `0x98854f` (op 60, buddy) · `CWvsContext::OnAllianceResult` @ `0x98345d`
(op 63) · `CWvsContext::OnInventoryOperation` @ `0x96953e` (op 26) ·
`CUIGuildBBS::OnGuildBBSPacket` @ `0x95dba2` (op 56) · `CField::OnFieldEffect`
@ `0x51e577` (op 130) · messenger `CUIMessenger::OnPacket` @ `0x7bc0a5` (routed
from CField op 291). Stage E owns per-mode extraction for these — see
dispatcher-family list below.

---

## (c) Structure / encoding deltas vs v83 — login → channel → map → movement/chat + tier-1

Swept (not sampled) for the connect-critical flow. Findings:

| flow stage | v79 packet | structure vs v83 | evidence |
|---|---|---|---|
| login | LOGIN_PASSWORD (sb) | **same body** | `SendCheckPasswordPacket` 0x5cbf50 |
| login | CHARLIST (cb, op 11) | **same entry shape** (CharacterStat+AvatarLook+rankFlag+[16B]); opcode 11 both | `sub_5CE522` 0x5ce522 |
| login | WORLD_INFORMATION (cb, op 10) | same body; opcode 10 both | `sub_5CE248` 0x5ce248 |
| channel | CHANGE_CHANNEL (sb, op 37) | same body `Enc1(ch)+Enc4` | `0x51baa2` |
| map | CHANGE_MAP / TransferField (sb, op 36) | same body | `0x51b950` |
| map | SET_FIELD (cb, op 118) | dispatcher confirmed (`CStage::OnSetField`); body not byte-diffed in Stage A | `0x6f079f` |
| movement | MOVE_PLAYER (cb, op 171→v83 185) | path decode via `CUserRemote::OnMove`; `CMovePath::Decode` @ 0x6573df | `0x8c8d4a` |
| chat | general/whisper (cb op 126/127; sb whisper op 117) | OnGroupMessage/OnWhisper present; whisper sb body same | `0x51c90f`, `0x51a7bc` |
| chat | CHATTEXT (cb op 151/152) | `CUser::OnChat(pkt, 0/1)` two-arm same as v83 | `0x8c8c79` |

**No wire-structure delta was found in the swept core flow** beyond the opcode
renumbering documented in (a)/(f) — bodies match v83 at every send/decode site
read. The risk surface for v79 is **opcode numbering and absent ops**, not body
encoding, for the login→map→movement→chat path.

**tier-1 note (`docs/packets/evidence/tiers.yaml`):** `summon/clientbound/
SummonSpawn` — the version-gated trailing avatar-look byte is `GMS≥95` only
(tiers.yaml: "absent on GMS v83/v84/v87"). v79 < v83, so SummonSpawn has **no
trailing avatar byte** in v79 — **same as the v83 anchor**. The summon cluster
sits at v79 ops 164–169 (`sub_892500`). Other tier-1 prefix families
(party/guild/buddy/messenger/inventory/storage/cash/monster/pet/character/field)
are opcode-mapped above; their opaque bodies (CharacterStat, Asset,
MonsterTemporaryStat, Movement, GuildMember, Avatar) are decoded by the same
named `*::Decode` helpers as v83 — Stage E byte-fixtures confirm per packet.

## (d) "Same as anchor" — entries with explicit switch/read-order evidence

These are **verified equal**, not defaulted:
- **Login ops 0,1,3,4,5,6,7,8,9,10,11,12** — v79 `CLogin::OnPacket` cases match
  v83 case-for-case to the same handler (see (f) table).
- **CHARLIST entry encoding** — `sub_5CE522` read order = v83 CHARLIST entry.
- **LOGIN_PASSWORD send body** — `SendCheckPasswordPacket` = v83 body; opcode 1.
- **CHATTEXT two-arm** (`CUser::OnChat(.,0)` / `(.,1)`) — same as v83.
- **Status-message mode count 0–13** — `OnMessage` switch identical arm count.
- **SummonSpawn trailing-avatar absence** — v79 inherits v83's "no avatar byte."
- **Mob/Npc/employee/drop/kite/mist/door/reactor leaf order** — uniform −22 with
  no gaps ⇒ same per-pool sequence as v83, only base-offset shifted.

---

## (OQ-7) Dispatcher-family list for Stage E (sizes the per-version mode campaign)

| family | v79 dispatcher (addr) | v79 cb op | mode table vs v83 |
|---|---|---|---|
| status-message | `CWvsContext::OnMessage` 0x96ade7 | 36 | **arm count 0–13 = v83** (extract values; no INCREASE_SKILL_POINT) |
| worldmessage/broadcast | `CWvsContext::OnBroadcastMsg` 0x96c94f | 65 | arms 0–13; shape family = v83; byte-confirm item-speaker arms |
| party | `CWvsContext::OnPartyResult` 0x987583 | 59 | v79 modes listed in (b); **diff vs v83 switch — may shift** |
| guild | `CWvsContext::OnGuildResult` 0x98099f | 62 | not extracted — Stage E |
| alliance | `CWvsContext::OnAllianceResult` 0x98345d | 63 | not extracted — Stage E |
| buddy/friend | `CWvsContext::OnFriendResult` 0x98854f | 60 | not extracted — Stage E |
| guild-BBS | `CUIGuildBBS::OnGuildBBSPacket` 0x95dba2 | 56 | not extracted — Stage E |
| inventory-op | `CWvsContext::OnInventoryOperation` 0x96953e | 26 | not extracted — Stage E |
| messenger | `CUIMessenger::OnPacket` 0x7bc0a5 | (CField 291) | body-mode demuxer; Stage E |
| field-effect | `CField::OnFieldEffect` 0x51e577 | 130 | not extracted — Stage E |
| cashshop | `CCashShop::OnPacket` 0x471da6 | 301–309 | map opcodes vs v83 CCashShop — Stage B |

**Campaign size:** ~11 dispatcher families need per-version operation-table
extraction in Stage E; party + status-message + broadcast are partially done
here (values captured), the remaining 8 are located but not byte-extracted.

---

## Escalations / open items handed to Stage B/E (none blocking Stage A)

1. **CSV v79 column is placeholder-only** — seed v79 from this doc/IDB, not the
   CSV. (Documented above; not a blocker.)
2. **MOVE_PLAYER serverbound opcode = 39 (RESOLVED in Stage A review)** —
   `sub_91B6E6` flush @ `0x91b883` builds `COutPacket(39)`; Δ−2 vs v83=41.
   Stage B seeds this verified value directly (see serverbound table above).
3. **v79-extra ops** with no v83 registry equivalent: `OnPotionDiscountRateChanged`
   (cb 87), `CUserLocal::OnNotifyHPDecByField` (cb 198) — Stage B must decide
   whether to register them (v79-only) or treat as no-ops.
4. **CField subclass overrides** (ContiMove/Massacre/AriantArena) not walked for
   v79 — confirm CONTI_MOVE/PYRAMID presence in Stage B via the subclass
   `OnPacket` overrides.
5. **CCashShop 301–309** opcodes not cross-mapped to v83 — Stage B.

No unresolved fname required fabrication; every unnamed sub above was identified
from its decompiled read-order/send-opcode (`sub_5CE217/5CE248/5CE522`,
`sub_969022/95E24B/95EE37/95F0D4`, `sub_522DC3`, `sub_668A2D`, `sub_892500`,
`sub_614977`).
