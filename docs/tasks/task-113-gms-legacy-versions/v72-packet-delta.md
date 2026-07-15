# GMS v72 — packet delta vs v79 (Stage A)

> **Source-of-truth delta doc for the v72 pass.** Stage B (registry), Stage C
> (template), and Stage E (verification campaign) consume this. **Anchor =
> `gms_v79`** (the completed second pass — its delta doc, registry
> `gms_v79.yaml`, template `template_gms_79_1.json`, and export are corrected
> and verified). Every opcode/mode/structure claim below cites the v72 IDB
> (function name + address, switch-case, decompiled read-order) or explicit
> v79-anchor evidence with body confirmation.

## IDA pre-flight (re-confirmed by binary name via `list_instances`)

| Role | Port | Binary | idb |
|---|---|---|---|
| **v72 target** | **13339** | `GMS_v72.1_U_DEVM.exe` | `E:\...\GMS\v72\GMS_v72.1_U_DEVM.exe.i64` |
| v79 anchor | 13340 | `GMS_v79_1_DEVM.exe` | `E:\...\GMS\v79\GMS_v79_1_DEVM.exe.i64` |
| v95 tie-breaker | 13341 | `GMS_v95.0_U_DEVM.exe` | — |
| v83 | 13342 | `MapleStory_dump.exe` | — |

All present and reachable. `select_instance(13339)` for every v72 read below.
The v72 IDB uses **mangled MSVC symbols**; names below are the demangled forms.
All addresses are v72 unless prefixed `v79:`. The `GMS v72` CSV column is the
literal placeholder `0x000` (like all four legacy versions — see preflight);
**every v72 opcode here is derived from the v72 IDB, not the CSV.**

## ⚠️ Critical lesson carried from v79 — SYMBOL NAMES ARE ROTATED OFF THEIR BODIES

The v72 IDB reproduces the **same rotated-symbol trap** the v79 pass caught in
`CLogin::OnPacket`. The three char-management handler symbols are rotated one
step off their actual handler bodies. **Every opcode below was matched by the
handler BODY read-order (the `CInPacket::Decode*` sequence), never by the symbol
label.** See §(f) for the body-verification proof.

---

## Top-level routing — the shim + CWvsContext window

`CClientSocket::ProcessPacket` @ **`0x486922`** (`CInPacket::Decode2` scrutinee):

```
case 0x10: OnMigrateCommand   0x11: OnAliveReq   0x12: OnAuthenCodeChanged
0x13: OnAuthenMessage         0x14: CSecurityClient::OnPacket
default: if (op in [0x1A,0x71]) -> CWvsContext::OnPacket(op)  [g_pWvsContext]
         else -> current-stage vtable+8 (CLogin / CStage subclass)::OnPacket(op)
```

The CWvsContext window is **`[0x1A,0x71]` (26–113)** in v72 vs **`[0x1A,0x75]`
(26–117)** in v79. Lower bound **identical** (0x1A); upper bound **−4** (v79 has
4 CWvsContext ops v72 lacks — enumerated in §(a)). **Non-uniform shift: there is
no single global offset; the clientbound delta grows as the opcode rises
(Δ0 → −1 → −2 → −4 → −6 → −8).** Each dispatcher is mapped fname-by-fname / by
body below.

---

## (e) usesPin (OQ-2) — **false** for v72

- Anchor: `template_gms_79_1.json` / `template_gms_83_1.json` → `"usesPin": false`.
- v72 evidence: `CLogin::OnPacket` @ `0x5b2411` keeps `OnCheckPinCodeResult`
  (case 6 → `0x5b56b9`) and `OnUpdatePinCodeResult` (case 7 → `0x5b59f8`)
  **identically to v79/v83**, and it **lacks** the second-password (SPW) cases:
  no case `0x17` (OnEnableSPWResult / LOGIN_AUTH) and no case `0x1C`
  (OnCheckSPWResult / CHECK_SPW_RESULT) exist in the switch (highest case is 22,
  then default forwards 114–116 → CStage). v72 is no more PIN/SPW-dependent than
  v79. **`usesPin` carries as `false`.**

---

## (f) Login-flow divergence (OQ-3) — biggest connect risk

`CLogin::OnPacket` v72 @ **`0x5b2411`** (decompiled). The login opcode layout is
**byte-for-byte identical to the v79 anchor** (same cases, same absent ops):

| op | v72 handler (addr) | v79 op | Δ / note |
|---|---|---|---|
| 0 | OnCheckPasswordResult `0x5b2577` | 0 | same |
| 1 | OnGuestIDLoginResult `0x5b3040` | 1 | same |
| 2 | **— (no case)** | — (absent in v79 too) | **ACCOUNT_INFO absent** (same as v79) |
| 3 | `sub_5B33C7` = OnCheckUserLimitResult | 3 | same op (SERVERSTATUS) |
| 4 | OnSetAccountResult `0x5b553a` | 4 | same |
| 5 | OnConfirmEULAResult `0x5b5654` | 5 | same |
| 6 | OnCheckPinCodeResult `0x5b56b9` | 6 | same |
| 7 | OnUpdatePinCodeResult `0x5b59f8` | 7 | same |
| 8 | OnViewAllCharResult `0x5b3f7d` | 8 | same |
| 9 | OnSelectCharacterByVACResult `0x5b44f8` (else-if a2==9 branch) | 9 | same |
| 10 | `sub_5B33F8` = OnWorldInformation | 10 | same op (WORLD_INFORMATION) |
| 11 | `sub_5B3646` = char-list decode | 11 | same op (CHARLIST) |
| 12 | OnSelectCharacterResult `0x5b47da` | 12 | same (SERVER_IP) |
| 13 | symbol `OnCreateNewCharacterResult` `0x5b3983` — **body = CHAR_NAME_RESPONSE** | 13 | **same by body** (see rotation proof) |
| 14 | symbol `OnDeleteCharacterResult` `0x5b3c65` — **body = ADD_NEW_CHAR_ENTRY** | 14 | **same by body** |
| 15 | symbol `OnCheckDuplicatedIDResult` `0x5b3a18` — **body = DELETE_CHAR_RESPONSE** | 15 | **same by body** |
| 22 | OnSelectWorldResult `0x5b4abc` (relog-to-title) | 22 | same (RELOG_RESPONSE) |
| 114/115/116 | default → CStage::OnPacket (SET_FIELD/SET_ITC/SET_CASH_SHOP) | 118/119/120 | **Δ−4** (see §(a)) |
| 23/26/27/28 | **— (no case)** | — (absent in v79 too) | LOGIN_AUTH / LAST_CONNECTED_WORLD / RECOMMENDED_WORLD / CHECK_SPW absent |

### ⚠️ Char-management BODY-verification (the rotated-symbol proof)

The v72 IDB symbol labels on `CLogin::OnPacket` cases 13/14/15 are **rotated one
step off their handler bodies** — identical to the v79 trap. Decompiling the
three BODIES (wire truth) shows v72 char-management is **IDENTICAL to v79/v83**:

| v72 op | case-symbol (IDB, ROTATED) | body read-order (decompiled) | packet = |
|---|---|---|---|
| **13** | `OnCreateNewCharacterResult` @`0x5b3983` | `DecodeStr + Decode1` | **CHAR_NAME_RESPONSE** (name-check) |
| **14** | `OnDeleteCharacterResult` @`0x5b3c65` | `Decode1 + GW_CharacterStat::Decode + AvatarLook::Decode` (success path) | **ADD_NEW_CHAR_ENTRY** |
| **15** | `OnCheckDuplicatedIDResult` @`0x5b3a18` | `Decode4 + Decode1` + slot-removal (memcpy shift) | **DELETE_CHAR_RESPONSE** |

By handler-body behaviour: **CHAR_NAME_RESPONSE=13, ADD_NEW_CHAR_ENTRY=14,
DELETE_CHAR_RESPONSE=15 — same as v79/v83.** Stage B must register these by the
**body-canonical (v83) FName**, not the rotated IDB symbol (mirror the corrected
`gms_v79.yaml` entries). Trusting the symbol would send ADD's stat+avatar payload
to op 13 whose handler leads with `DecodeStr` → client crash.

**Handshake / encoding:**
- `LOGIN_PASSWORD` send `CLogin::SendCheckPasswordPacket` @ `0x5b1170` builds
  `COutPacket(1)` then `EncodeStr(id) + EncodeStr(pw) + EncodeBuffer(machineId,16)
  + Encode4(gameRoomClient) + Encode1(b) + Encode1(0) + Encode1(0) +
  Encode4(partnerCode)`. Opcode **1 = v79 (Δ0)**; body shape identical.
- **CHARLIST** (`sub_5B3646`, op 11): `Decode1 status`; on status∈{0,12,23}
  `Decode1 charCount`, then per slot: `GW_CharacterStat::Decode + AvatarLook::Decode
  + Decode1 rankEnabled` (`Decode1`? `DecodeBuffer(16)` : `memset 16`), then
  `Decode4`. **Same entry shape and opcode (11) as v79.**
- `WORLD_INFORMATION` (`sub_5B33F8`, op 10): opcode unchanged (10).

**Connect-critical conclusion:** v72's login flow is structurally the **same as
the v79 anchor** — same opcode layout, same absent ops (2, 23/26/27/28), same
char-mgmt body mapping. The only login-region shift is the CStage SET_FIELD/ITC/
CASH_SHOP block at **114/115/116** (v79 118/119/120, Δ−4).

---

## (a) Opcode map — clientbound (writer) by dispatcher

Mapping = (v72 switch case, by body) → (v79 op via `gms_v79.yaml`). "Δ" is
`v72 − v79`. Each block cites its dispatcher fn+addr.

### CWvsContext::OnPacket @ `0x9025c8` (ops 26–113)

Full switch enumerated. Delta accumulates as ops v79 has are dropped:

| v72 run | fnames (first→last) | v79 run | Δ |
|---|---|---|---|
| 26–40 | OnInventoryOperation … OnAntiMacroResult (WEDDING_PHOTO=40) | 26–40 | **0** |
| 42–47 | OnClaimResult … OnEntrustedShopCheckResult | 42–47 | 0 |
| 48 | `sub_9175E6` (= OnSkillLearnItemResult, SKILL_LEARN_ITEM_RESULT) | 48 | 0 |
| 49,50 | OnGatherItemResult, OnSortItemResult | 49,50 | 0 |
| 52,54,55 | OnSueCharacterResult, OnTradeMoneyLimit, OnSetGender | 52,54,55 | 0 |
| 56 | `CUIGuildBBS::OnGuildBBSPacket` (GUILD_BBS_PACKET) | 56 | 0 |
| 58–74 | OnCharacterInfo … OnSetWeekEventMessage | 58–74 | 0 |
| 75 | `sub_917AB7` (= OnSetPotionDiscountRate, SET_POTION_DISCOUNT_RATE) | 75 | 0 |
| 76 | OnBridleMobCatchFail | 76 | 0 |
| 77 | `sub_902E77` (= OnImitatedNPCResult; body `Decode1`) | 77 | 0 |
| 78 | `sub_902E83` (= OnImitatedNPCData; body forwards `CNpcPool::OnPacket(78)`) | 78 | 0 |
| — | *(v79 op 79 `LIMITED_NPC_DISABLE_INFO` is **ABSENT** in v72)* | 79 | Δ steps 0→**−1** |
| 79,80 | OnMonsterBookSetCard, OnMonsterBookSetCover | 80,81 | −1 |
| 81 | OnHourChanged | 82 | −1 |
| 82 | `sub_90CE83` (= OnMiniMapOnOff, MINIMAP_ON_OFF; body `Decode1` sets flag) | 83 | −1 |
| 83–86 | OnPartyValue, OnFieldSetVariable, OnBonusExpRateChanged, OnPotionDiscountRateChanged | 84–87 | −1 |
| 87–97 | OnFamilyChartResult … OnFamilySummonRequest | 88–98 | −1 |
| 98 | `sub_90D651` (= OnNotifyLevelUp, NOTIFY_LEVELUP; body `UI_Open(29,-1)`) | 99 | −1 |
| 99 | `sub_90D65D` (= OnNotifyWedding, NOTIFY_MARRIAGE; body `Decode1+Decode4+DecodeStr`, marriage chatlog) | 100 | −1 |
| — | *(v79 op 101 `NOTIFY_JOB_CHANGE` is **ABSENT** in v72)* | 101 | Δ steps −1→**−2** |
| 101–104 | OnMapleTVUseRes, OnAvatarMegaphoneRes, OnSetAvatarMegaphone, OnClearAvatarMegaphone | 103–106 | −2 |
| 105–112 | OnCancelNameChangeResult … OnCancelNameChangebyOther | 107–114 | −2 |
| — | *(v79 ops 115 `SET_EXTRA_PENDANT_SLOT` + 116 `SCRIPT_PROGRESS_MESSAGE` are **ABSENT** in v72)* | 115,116 | Δ steps −2→**−4** |
| 113 | OnMacroSysDataInit (MACRO_SYS_DATA_INIT) | 117 | **−4** |

**Absent in v72 vs the v79 anchor (CWvsContext):** `LIMITED_NPC_DISABLE_INFO`
(v79 79), `NOTIFY_JOB_CHANGE` (v79 101), `SET_EXTRA_PENDANT_SLOT` (v79 115),
`SCRIPT_PROGRESS_MESSAGE` (v79 116). No v72-extra CWvsContext op (v79's
`OnPotionDiscountRateChanged` extra is retained at v72 op 86).

### CStage::OnPacket @ `0x6c0c61` (ops 114–116) — Δ **−4**
`'r'`(114) OnSetField (SET_FIELD) · `'s'`(115) OnSetITC (SET_ITC) · `'t'`(116)
OnSetCashShop (SET_CASH_SHOP). v79 = 118/119/120.

### CMapLoadable::OnPacket @ `0x5f59e3` (ops 117–118) — Δ **−4**
117 OnSetBackEffect (SET_BACK_EFFECT) · 118 `sub_5F5F54` (= OnSetMapObjectVisible,
SET_MAP_OBJECT_VISIBLE). v79 = 121/122. **`CLEAR_BACK_EFFECT` ABSENT** (only two
cases), same as v79.

### CField::OnPacket @ `0x515879` (base ops 119–144) — Δ **−4**
119 OnTransferFieldReqIgnored (BLOCKED_MAP) · 120 OnTransferChannelReqIgnored ·
121 OnFieldSpecificData (FORCED_MAP_EQUIP) · 122 OnGroupMessage (MULTICHAT) ·
123 OnWhisper (WHISPER) · 124 OnCoupleMessage (SPOUSE_CHAT) · 125
OnSummonItemInavailable · **126 OnFieldEffect (FIELD_EFFECT)** · 127
OnFieldObstacleOnOff · 128 OnFieldObstacleOnOffStatus · 129 OnFieldObstacleAllReset
· 130 OnBlowWeather · 131 OnPlayJukeBox · 132 OnAdminResult · 133 OnQuiz · 134
OnDesc · 135 vtable+0x20 (CLOCK, OnClock) · 138 OnSetQuestClear · 139
OnSetQuestTime · 140 OnWarnMessage (ARIANT_RESULT) · 141 OnSetObjectState · 142
OnDestroyClock (STOP_CLOCK) · 144 `sub_51BCE9` (= OnStalkResult, IDA_0X09C).
All Δ−4 vs v79 (123…148). Gaps at 136/137 and 143 mirror the v79 base-switch gaps
(CONTI_MOVE/CONTI_STATE/ARIANT_ARENA subclass ops), shifted −4.

### CField pool-routing ranges (read from `0x515879`) and per-pool deltas
CField dispatches these opcode ranges to the pools:

| pool | v72 dispatcher | v72 range | v79 range | Δ (start) |
|---|---|---|---|---|
| CUserPool | `0x87bc00` | 145–207 | 149–212 | **−4** |
| CMobPool | `0x6255ae` | 208–226 | 214–233 | **−6** |
| CNpcPool | `0x645ca5` | 227–234 | 235–241 | **−8** |
| CEmployeePool | `0x4f4962` | 235–237 | 243–245 | −8 |
| CDropPool | `0x4e9947` | 238–239 | 246–247 | −8 |
| CMessageBoxPool | `0x60b13c` | 240–242 | 248–250 | −8 |
| CAffectedAreaPool | `0x42e347` | 243–244 | 251–252 | −8 |
| CTownPortalPool | `0x6f967d` | 245–246 | 253–254 | −8 |
| CReactorPool | `0x691f4e` | 247–250 | 255–258 | −8 |

The Δ deepens **−4 → −6 → −8** across the user/mob region (below); Npc and all
lower map-pools settle at Δ−8.

### CUserPool::OnPacket @ `0x87bc00` — enter/leave Δ **−4**
145 OnUserEnterField (SPAWN_PLAYER) · 146 OnUserLeaveField
(REMOVE_PLAYER_FROM_MAP). Routes 147–166→common, 167–186→remote, 187–206→local.

### CUserPool::OnUserCommonPacket @ `0x87bf75` (ops 147–165) — Δ **−4**
Reads `Decode4 cid`. 147 OnChat(pkt,0) (CHATTEXT) · 148 OnChat(pkt,1) (CHATTEXT1,
with the user/broadcast fallback) · 149 OnADBoard (CHALKBOARD) · 150
OnMiniRoomBalloon (UPDATE_CHAR_BOX) · 151 SetConsumeItemEffect · 152
ShowItemUpgradeEffect · **153–159 OnPetPacket** (SPAWN_PET 153 … PET_COMMAND 159)
· **160–165 `sub_848023`** (summon cluster: SPAWN_SPECIAL_MAPOBJECT 160 …
SUMMON_SKILL/DAMAGE 165). Uniform Δ−4 vs v79.

### CUserPool::OnUserRemotePacket @ `0x87c046` (ops 167–186) — Δ **−4**
Reads `Decode4 cid`. 167 OnMove (**MOVE_PLAYER**) · 168–171 OnAttack
(CLOSE/RANGED/MAGIC/ENERGY) · 172 OnSkillPrepare (SKILL_EFFECT) · 173 OnSkillCancel
· 174 OnHit (DAMAGE_PLAYER) · 175 SetEmotion (FACIAL_EXPRESSION, `Decode4`) · 176
SetActiveEffectItem (SHOW_ITEM_EFFECT, `Decode4`) · 177 OnShowUpgradeTombEffect ·
178 chair-inline (`Decode4`→RemoteUser+2902, SHOW_CHAIR) · 179 OnAvatarModified
(UPDATE_CHAR_LOOK) · 180 OnEffect (SHOW_FOREIGN_EFFECT) · 181 OnSetTemporaryStat
(GIVE_FOREIGN_BUFF) · 182 OnResetTemporaryStat · 183 OnReceiveHP
(UPDATE_PARTYMEMBER_HP) · 184 OnGuildNameChanged · 185 OnGuildMarkChanged · 186
OnThrowGrenade. **Uniform Δ−4** vs v79 (171–190). **No dragon arm**
(SPAWN/MOVE/REMOVE_DRAGON absent — same as v79; dragons are Evan/v84+).

### CUserLocal::OnPacket @ `0x85dca2` (ops 187–206) — Δ **−4 → −6**
187 OnSitResult (CANCEL_CHAIR) · 188 OnEffect (SHOW_ITEM_GAIN_INCHAT) · 189
OnTeleport (DOJO_WARP_UP) · 191 OnMesoGive_Succeeded (LUCKSACK_PASS) · 192
OnMesoGive_Failed (LUCKSACK_FAIL) · 193 OnQuestResult (UPDATE_QUEST_INFO) · 194
OnNotifyHPDecByField (**v72/v79-extra**, no v83 registry op) · 195 `nullsub_12`
(dead) · 196 OnBalloonMsg (PLAYER_HINT) · 197 OnPlayEventSound · 198
OnPlayMinigameSound · 199 OnMakerResult (MAKER_RESULT) · 201
OnOpenClassCompetitionPage (KOREAN_EVENT) · 202 OnSetDirectionMode (LOCK_UI) · 203
`sub_86C50E` (= OnSetStandAloneMode, DISABLE_UI) · 204 `sub_86C65C` (= OnHireTutor,
SPAWN_GUIDE; guarded forward `sub_716AE9`) · 205 OnRandomEmotion (RANDOM_EMOTION)
· 206 OnSkillCooltimeSet (COOLDOWN).

Ops 187–204 are **Δ−4** vs v79 (191–208). Then **v79 ops 209 `TALK_GUIDE` and 210
`SHOW_COMBO` are ABSENT in v72** — v72 jumps directly SPAWN_GUIDE(204) →
RANDOM_EMOTION(205), so **RANDOM_EMOTION=205 / COOLDOWN=206 are Δ−6** vs v79
(211/212). This local-region shrinkage (2 dropped ops) is what deepens the map
delta from −4 to −6. *(Stage B: `sub_86C65C`→`sub_716AE9` labelled SPAWN_GUIDE by
Δ−4 alignment + named neighbours; confirm the exact retained op if a guide/combo
fixture is built. The −6 conclusion is independent — RANDOM_EMOTION/COOLDOWN are
named handlers.)*

### CMobPool::OnPacket @ `0x6255ae` — enter Δ **−6**, deepens to −8
208 OnMobEnterField (SPAWN_MONSTER) · 209 OnMobLeaveField · 210
OnMobChangeController · 211–225 OnMobPacket (MOVE_MONSTER 211 … ) · 221
OnMobCrcKeyChanged (MOB_CRC_KEY_CHANGED, carved out). v72 OnMobPacket range =
**211–225 (14 leaves + crckey)** vs v79 217–233 (16 leaves + crckey): **2 mob
sub-ops fewer in v72** → the delta steps **−6 → −8** across the mob range. Stage B
must diff the `CMobPool::OnMobPacket` internal leaf switch (`0x62560d`) to
identify the 2 absent mob leaves.

### CNpcPool::OnPacket @ `0x645ca5` (ops 227–233) — Δ **−8**
227 OnNpcEnterField (SPAWN_NPC) · 228 OnNpcLeaveField · 229 OnNpcChangeController
· 230–232 OnNpcPacket (NPC_ACTION …) · 233 `sub_645E9C` (= OnSetNpcScript,
SET_NPC_SCRIPTABLE). (Case 78 = `OnNpcImitateData`, the routed IMITATED_NPC_DATA
target from CWvsContext op 78 — not a real 78-opcode dispatch, same dead-case
family as v79.) Uniform Δ−8, same 7-op leaf order as v79 (235–241).

### Map pools (range-routed from CField) — all Δ **−8**
Per-pool leaf order matches v79; only the base offset shifts −8:
- CEmployeePool 235–237 → SPAWN/REMOVE/UPDATE_HIRED_MERCHANT
- CDropPool 238–239 → DROP_ITEM_FROM_MAPOBJECT / REMOVE_ITEM_FROM_MAP
- CMessageBoxPool 240–242 → CANNOT_SPAWN_KITE / SPAWN_KITE / REMOVE_KITE
- CAffectedAreaPool 243–244 → SPAWN_MIST / REMOVE_MIST
- CTownPortalPool 245–246 → SPAWN_DOOR / REMOVE_DOOR
- CReactorPool 247–250 → REACTOR_HIT / MOVE / SPAWN / DESTROY

### CField high-region routes (dialog / script / dispatcher-families) @ `0x515879`
268 OnHontailTimer · 269 OnZakumTimer · 270 CScriptMan::OnPacket · 271–272
CShopDlg · 273–274 CAdminShopDlg · 275 CTrunkDlg · 276–277 CStoreBankDlg · 278
CRPSGameDlg · **279 CUIMessenger** · 280 CMiniRoomBaseDlg · 288 CParcelDlg ·
298–300 CFuncKeyMappedMan · 303–308 CMapleTVMan · 312–315 `sub_51C05F`. (These
demuxers use a secondary MODE byte, not the top-level opcode — Stage E family
work; see §(OQ-7).)

### CCashShop::OnPacket @ `0x470b2d` (ops 289–296) — **structurally divergent**
289 OnNoticeFreeCashItem · 290 OnOneADay · 291 OnCashItemResult · 292
OnCashItemGachaponResult · 293 `sub_473519` · 294 OnCashShopGachaponStampResult ·
296 OnCheckTransferWorldPossibleResult. **The v72 cash-shop opcode block
(289–296) differs in both range and membership from the v79 anchor (301–309)** —
no uniform delta; the op sets are not a shifted copy (v72 adds OnNoticeFreeCashItem
/ OnOneADay / OnCashItemGachaponResult; v79 has OnChargeParamResult /
OnQueryCashResult / OnPurchaseExpChanged not seen here). Cash-shop is a separate
stage outside the core login→map flow — **Stage B must map 289–296 against the v79
`CCashShop::OnPacket` case-by-case (checked, not shifted).**

---

## (a) Opcode map — serverbound (handler) — from v72 `COutPacket(N)` send-sites

No CSV v72 serverbound column exists; opcodes are read from the `COutPacket(N)`
constructor at each send-site. **The serverbound space is its own enumeration and
is NOT simply "lower than v79" — in the mid/social region v72 is Δ+1 HIGHER than
v79** (opposite sign to the clientbound shift). Core-flow anchors, all verified:

| op (serverbound) | v72 send-site (addr) | v72 opcode | v79 op | Δ vs v79 |
|---|---|---|---|---|
| LOGIN_PASSWORD | `CLogin::SendCheckPasswordPacket` `0x5b1170` | `COutPacket(1)` | 1 | **0** |
| CHANGE_MAP | `CField::SendTransferFieldRequest` `0x5148b1` | `COutPacket(37)` | 36 | **+1** |
| CHANGE_CHANNEL | `CField::SendTransferChannelRequest` `0x514a03` | `COutPacket(38)` | 37 | **+1** |
| MOVE_PLAYER | move flush `sub_8CB63E` `0x8cb63e` (after `CVecCtrlUser::OnSit` `0x8cb617`; via `xrefs_to(CMovePath::Flush 0x6350f5)`) | `COutPacket(40)` | 39 | **+1** |
| WHISPER | `CField::SendChatMsgWhisper` `0x513743` | `COutPacket(118)` | 117 | **+1** |
| PARTY_OPERATION | `CField::SendCreateNewPartyMsg` `0x5142b0` | `COutPacket(122)` | 121 | **+1** |

Body shapes verified at each send-site (all match v79/v83):
- **CHANGE_MAP** = `Enc1(portalByte)+Enc4(targetMap)+EncStr(portal)+[Enc2(x)+Enc2(y)]
  +Enc1(0)+Enc1(wheel)+Enc1(premiumFlag)[+Enc4+Enc4]`.
- **CHANGE_CHANNEL** = `Enc1(channel)+Enc4(t)`.
- **MOVE_PLAYER** = `Enc1(field+276)+Enc4(fieldKey)` then `CMovePath::Flush`.
- **WHISPER** = `Enc1(mode)+EncStr(target)+EncStr(text)`; mode `(found+1)|4` for
  whisper, `0x86` for find-buddy locate.
- **PARTY_OPERATION** create = `Enc1(1)`; the accept/decline replies emitted from
  `OnPartyResult` use `COutPacket(122)` (accept: `Enc1(3)+Enc4+Enc1`) and
  `COutPacket(123)` (block/report: `Enc1(21|22)+EncStr+EncStr`). v79 = 121/122.

> **The full serverbound opcode table is a Stage B deliverable** — derive each op
> from its v72 send-site anchored on the v79 serverbound FName. Stage A fixes the
> method + the six core-flow anchors above. **NPC_TALK** (v79 = 56, in the
> mid-region) was not locatable by symbol in v72 (`CUserLocal::TalkToNpc` is not a
> named symbol here); the Δ+1 mid-region pattern predicts ~57 but this is
> **UNVERIFIED** — Stage B must read it from the actual npc-select send-site
> (do not seed the predicted value blind).

---

## (b) Operation / mode (sub-op) tables — from v72 dispatcher switches

Read **from the v72 switch**; do not inherit v79's values (they shift).

### Status-message — `CWvsContext::OnMessage` @ `0x9191ee` (op 36 / SHOW_STATUS_INFO)
Leading `Decode1` mode; switch arms **0,1,2,3,4,5,6,7,8,9,10,11** — **12 arms
(0–11), default = drop.** **This is NARROWER than v79/v83 (which have 0–13, 14
arms).** v72 lacks modes 12 and 13. **Stage C consequence: the v72 `operations`
table for SHOW_STATUS_INFO must be 0–11, and must still have NO spurious
`INCREASE_SKILL_POINT`** (the off-by-one trap that crashed v83 on spawn applies
here too — keep the table contiguous 0–11 matching the IDA switch).

### World/broadcast message — `CWvsContext::OnBroadcastMsg` @ `0x91aaac` (op 65 / SERVERMESSAGE)
Leading `Decode1` type; arms **0–13 (14 arms)** — **same shape as v79.** Notable
bodies: type 4 = management speaker (reads extra `Decode1` whisper-flag); type 8 =
`Decode1+Decode1+Decode1? GW_ItemSlotBase::Decode` (item-speaker); type 11 =
`Decode4+DecodeStr+GW_ItemSlotBase::Decode` (item megaphone); type 6 = `Decode4`;
type 7 = util-dlg (`Decode4`); types 12/13 = pink/notice. **SERVERMESSAGE op = 65
(Δ0 vs v79).** Stage E should byte-confirm arms 4/8/11 (item-speaker).

### Party — `CWvsContext::OnPartyResult` @ `0x934f3c` (op 59 / PARTY_OPERATION)
Leading `Decode1` mode. **v72 arms present (hex):** `4, 7, 8, 9, 0xA, 0xC, 0xD,
0xF, 0x10, 0x11, 0x12, 0x13, 0x15, 0x16, 0x17, 0x1A, 0x1B, 0x1C, 0x1D, 0x1F, 0x20,
0x21, 0x22, 0x23, 0x24` (+ default; `0x21` shares the load/refresh arm with `7`).

**⚠️ The party mode table SHIFTED vs v79.** v79 arms were `… 0x19, 0x1B, 0x1C,
0x1D, 0x1E, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25`. Key v72↔v79 differences:
- v72 has **`0x1A`** (invite/FindUser + join notice) where v79 had `0x19`.
- v72 has **`0x1F`** where v79 had `0x1E`.
- v72's HP/coord slot-update is at **`0x24`** (`Decode1 slot<6 + Decode4 + Decode4
  + Decode2 + Decode2`; throws `CDisconnectException` if slot≥6); v79 had it at
  `0x25`. v72 has **no `0x25`**.
- v72 `0x22` = HP/max update (`Decode4 cid + Decode4 + Decode4`, FindUser).

Notable v72 bodies: `4` invite → `Decode4(partyId)+DecodeStr(inviter)+Decode1(level)`,
builds YESNO + serverbound `COutPacket(122)` (accept) / `COutPacket(123)`
(block/report); `7`/`0x21` load → `Decode4+PARTYDATA::Decode`; `8` join →
`Decode4+Decode4+Decode4+Decode2+Decode2`; `0xC` leave/expel → `Decode4+Decode4+
Decode1`; `0xF` join member → `Decode4+DecodeStr+PARTYDATA::Decode`. **Stage E
must build the v72 party operations table from THESE values, not v79's.**

### Other dispatcher families with mode tables (located, not byte-extracted)
`CWvsContext::OnGuildResult` @ `0x92e31f` (op 62) · `CWvsContext::OnFriendResult`
@ `0x935ecf` (op 60, buddy) · `CWvsContext::OnAllianceResult` @ `0x930e16`
(op 63) · `CWvsContext::OnInventoryOperation` @ `0x917ad0` (op 26) ·
`CUIGuildBBS::OnGuildBBSPacket` @ `0x90c7da` (op 56) · `CField::OnFieldEffect`
@ `0x5174bb` (op 126) · messenger `CUIMessenger::OnPacket` @ `0x777b25` (routed
from CField op 279). Stage E owns per-mode extraction — see §(OQ-7).

---

## (c) Structure / encoding deltas vs v79 — login → channel → map → movement/chat + tier-1

Swept (not sampled) for the connect-critical flow:

| flow stage | v72 packet | structure vs v79 | evidence |
|---|---|---|---|
| login | LOGIN_PASSWORD (sb 1) | **same body** | `SendCheckPasswordPacket` 0x5b1170 |
| login | CHARLIST (cb 11) | **same entry shape** (CharacterStat+AvatarLook+rankFlag+[16B]); op 11 both | `sub_5B3646` 0x5b3646 |
| login | WORLD_INFORMATION (cb 10) | same; op 10 both | `sub_5B33F8` |
| login | char-mgmt (cb 13/14/15) | **same by body** (rotated symbols); §(f) | 0x5b3983 / 0x5b3c65 / 0x5b3a18 |
| channel | CHANGE_CHANNEL (sb 38) | same body `Enc1(ch)+Enc4` | `0x514a03` |
| map | CHANGE_MAP (sb 37) | same body | `0x5148b1` |
| map | SET_FIELD (cb 114) | dispatcher confirmed (`CStage::OnSetField`); body not byte-diffed in Stage A | `0x6c0c61` |
| movement | MOVE_PLAYER (cb 167) | remote path decode `CUserRemote::OnMove` `0x87c1f8` | `0x87c046` |
| movement | MOVE_PLAYER (sb 40) | same body (`Enc1+Enc4`+CMovePath::Flush) | `sub_8CB63E` 0x8cb63e |
| chat | general/whisper (cb 122/123; sb whisper 118) | OnGroupMessage/OnWhisper present; whisper sb body same | `0x515879`, `0x513743` |
| chat | CHATTEXT (cb 147/148) | `CUser::OnChat(pkt,0/1)` two-arm same as v79 | `0x87bf75` |

**No wire-structure delta was found in the swept core flow** beyond opcode
renumbering (§a/f) — bodies match v79 at every send/decode site read. The v72
risk surface is **opcode numbering + absent ops + the two shifted mode tables
(status-message 0–11, party)**, not body encoding, for the login→map→movement→chat
path.

**tier-1 note (`docs/packets/evidence/tiers.yaml`):** `summon/clientbound/
SummonSpawn` version-gated trailing avatar-look byte is `GMS≥95` only. v72 < v79
< v95, so SummonSpawn has **no trailing avatar byte** in v72 — same as the v79
anchor. The v72 summon cluster sits at ops **160–165** (`sub_848023`). Other
tier-1 prefix families are opcode-mapped above; their opaque bodies decode via the
same named `*::Decode` helpers as v79 — Stage E byte-fixtures confirm per packet.

## (d) "Same as anchor (v79)" — entries with explicit switch/body evidence

Verified equal (not defaulted):
- **Login ops 0,1,3,4,5,6,7,8,9,10,11,12,22** — v72 `CLogin::OnPacket` cases match
  v79 case-for-case (same absent ops 2/23/26/27/28).
- **Char-mgmt body mapping 13/14/15** — CHAR_NAME/ADD/DELETE identical to v79 by
  decompiled read-order (rotated symbols on both).
- **CHARLIST + WORLD_INFO + LOGIN_PASSWORD bodies** — same read/send order, same
  opcodes (11/10/1).
- **CWvsContext ops 26–78** — identical fname sequence, Δ0.
- **CUserPool common/remote leaf order** — uniform Δ−4, same per-leaf sequence.
- **Mob/Npc/employee/drop/kite/mist/door/reactor leaf order** — uniform Δ−8 (Npc
  and pools) with no reordering.
- **Broadcast mode arms 0–13** — same 14-arm shape as v79 (op 65, Δ0).
- **No dragon remote arm; no CLEAR_BACK_EFFECT** — same absences as v79.

---

## (OQ-7) Dispatcher-family list for Stage E (sizes the per-version mode campaign)

| family | v72 dispatcher (addr) | v72 cb op | mode table vs v79 |
|---|---|---|---|
| status-message | `CWvsContext::OnMessage` 0x9191ee | 36 | **DIFFERS — arms 0–11 only (v79 0–13)**; no INCREASE_SKILL_POINT |
| worldmessage/broadcast | `CWvsContext::OnBroadcastMsg` 0x91aaac | 65 | arms 0–13 = v79 shape; byte-confirm item-speaker arms 4/8/11 |
| party | `CWvsContext::OnPartyResult` 0x934f3c | 59 | **DIFFERS — modes shifted (0x1A/0x1F, slot-update at 0x24, no 0x25)**; values captured in §(b) |
| guild | `CWvsContext::OnGuildResult` 0x92e31f | 62 | not extracted — Stage E |
| alliance | `CWvsContext::OnAllianceResult` 0x930e16 | 63 | not extracted — Stage E |
| buddy/friend | `CWvsContext::OnFriendResult` 0x935ecf | 60 | not extracted — Stage E |
| guild-BBS | `CUIGuildBBS::OnGuildBBSPacket` 0x90c7da | 56 | not extracted — Stage E |
| inventory-op | `CWvsContext::OnInventoryOperation` 0x917ad0 | 26 | not extracted — Stage E |
| messenger | `CUIMessenger::OnPacket` 0x777b25 | (CField 279) | body-mode demuxer; Stage E |
| field-effect | `CField::OnFieldEffect` 0x5174bb | 126 | not extracted — Stage E |
| cashshop | `CCashShop::OnPacket` 0x470b2d | 289–296 | **map opcodes vs v79 CCashShop — structurally divergent — Stage B** |

**Campaign size:** ~11 dispatcher families need per-version operation-table
extraction. status-message, broadcast, party are captured here (party + status
values recorded; status-message table SHRINKS to 0–11); the remaining 8 are
located (addresses above) but not byte-extracted.

---

## Escalations / open items handed to Stage B/E (none blocking Stage A)

1. **CSV v72 column is placeholder-only** — seed v72 from this doc/IDB, not the CSV.
2. **Char-mgmt symbols are rotated** (§f) — register cb ops 13/14/15 by the
   body-canonical (v83) FName, exactly as `gms_v79.yaml` was corrected. NOT the
   IDB symbol label.
3. **Status-message mode table SHRINKS to 0–11** (v79 was 0–13) — the v72
   SHOW_STATUS_INFO `operations` table must have 12 arms; keep contiguous, no
   INCREASE_SKILL_POINT.
4. **Party mode table shifted** (§b) — Stage E builds the v72 party operations
   table from the captured v72 arms, not v79's.
5. **Serverbound is Δ+1 (HIGHER) than v79 in the mid/social region** (CHANGE_MAP
   37, CHANGE_CHANNEL 38, MOVE_PLAYER 40, WHISPER 118, PARTY 122). Login region is
   Δ0 (LOGIN_PASSWORD 1). Stage B derives the full sb table per send-site.
6. **NPC_TALK serverbound UNVERIFIED** — `CUserLocal::TalkToNpc` is unnamed in the
   v72 IDB; predicted ~57 by the +1 pattern but must be read from the send-site by
   Stage B (do not seed blind).
7. **CMobPool has 2 fewer mob-packet leaves than v79** (range 211–225 vs 217–233)
   → Stage B diffs `CMobPool::OnMobPacket` (`0x62560d`) internal switch.
8. **CUserLocal drops TALK_GUIDE + SHOW_COMBO** (v79 209/210) → local Δ deepens
   −4→−6; confirm `sub_86C65C`/`sub_716AE9` (op 204) identity if a fixture needs it.
9. **CCashShop 289–296 structurally divergent** from v79 301–309 — Stage B maps
   case-by-case (not a shifted copy).
10. **v72/v79-extra ops with no v83 registry equiv:** `OnPotionDiscountRateChanged`
    (cb 86), `CUserLocal::OnNotifyHPDecByField` (cb 194) — Stage B decides register
    vs no-op.
11. **CField subclass overrides** (ContiMove/Massacre/AriantArena, gaps at cb
    136/137/143) not walked for v72 — confirm CONTI_MOVE/PYRAMID presence in Stage B.

Every unnamed sub above was identified from its decompiled read-order / send-opcode
(`sub_5B33C7/5B33F8/5B3646`, `sub_9175E6/917AB7/902E77/902E83/90CE83/90D651/90D65D`,
`sub_51BCE9`, `sub_645E9C`, `sub_848023`, `sub_86C50E/86C65C`, `sub_8CB63E`,
`sub_5F5F54`). No unresolved fname required fabrication.
