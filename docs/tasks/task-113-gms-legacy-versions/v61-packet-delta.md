# GMS v61 — packet delta vs v72 (Stage A)

> **Source-of-truth delta doc for the v61 pass.** Stage B (registry), Stage C
> (template), and Stage E (verification campaign) consume this. **Anchor =
> `gms_v72`** (the completed prior pass — its delta doc
> `v72-packet-delta.md`, registry `gms_v72.yaml`, template
> `template_gms_72_1.json`, and export `gms_v72.json` are corrected and
> verified). Every opcode/mode/structure claim below cites the v61 IDB
> (function name + address, switch-case, decompiled read-order) or explicit
> v72-anchor evidence with body confirmation.

## IDA pre-flight (re-confirmed by binary name via `list_instances`)

| Role | Port | Binary | idb |
|---|---|---|---|
| **v61 target** | **13338** | `GMS_v61.1_U_DEVM.exe` | `E:\...\GMS\v61\GMS_v61.1_U_DEVM.exe.i64` |
| v72 anchor | 13339 | `GMS_v72.1_U_DEVM.exe` | `E:\...\GMS\v72\GMS_v72.1_U_DEVM.exe.i64` |
| v79 | 13340 | `GMS_v79_1_DEVM.exe` | — |
| v95 tie-breaker | 13341 | `GMS_v95.0_U_DEVM.exe` | — |
| v83 | 13342 | `MapleStory_dump.exe` | — |

All present and reachable. `select_instance(13338)` for every v61 read below.
The v61 IDB uses **mangled MSVC symbols** (many demangled by IDA); names below
are the demangled forms. All addresses are v61 unless prefixed `v72:`. The
`GMS v61` CSV column is the literal placeholder `0x000`; **every v61 opcode here
is derived from the v61 IDB, not the CSV.**

## ⚠️ Critical lesson carried from v72/v79 — SYMBOL NAMES ARE ROTATED OFF THEIR BODIES

The v61 IDB reproduces the **same rotated-symbol trap** the v72/v79 passes caught
in `CLogin::OnPacket`. The three char-management handler symbols are rotated one
step off their actual handler bodies. **Every char-mgmt opcode below was matched
by the handler BODY read-order (the `CInPacket::Decode*` sequence), never by the
symbol label.** See §(f) for the body-verification proof.

---

## Top-level routing — the shim + CWvsContext window

`CClientSocket::ProcessPacket` @ **`0x47440a`** (`CInPacket::Decode2` scrutinee):

```
case 0x10: OnMigrateCommand   0x11: OnAliveReq   0x12: OnAuthenCodeChanged
0x13: OnAuthenMessage         0x14: OnPacket_CSecurityClient
0x15: sub_4747E2              (v61-EXTRA vs v72 — see below)
default: if (op in [0x1A,0x5B]) -> CWvsContext::OnPacket(op)  [g_pWvsContext]
         else -> current-stage vtable+8 (CLogin / CStage subclass)::OnPacket(op)
```

The CWvsContext window is **`[0x1A,0x5B]` (26–91)** in v61 vs **`[0x1A,0x71]`
(26–113)** in v72. Lower bound **identical** (0x1A); upper bound **−22** (v72 has
22 more CWvsContext ops than v61 — the whole HourChanged→NotifyWedding /
Family block; enumerated in §(a)). **Non-uniform shift: there is no single global
offset; the clientbound delta deepens as the opcode rises (CWvsContext 26–80 =
Δ0, then a ~19-op block absent, base/pool region settling at Δ−22 → −33 → −36).**

**v61-extra top-level case `0x15` → `sub_4747E2`.** The v72 `ProcessPacket`
switch (per the v72 delta doc) has cases only through `0x14`; v61 adds a `0x15`
arm before the default CWvsContext/stage forward. Not connect-critical (a
handshake-family op alongside 0x10–0x14); Stage B should map its FName if a
migrate/authen fixture is built.

---

## (e) usesPin (OQ-2) — **false** for v61

- Anchor: `template_gms_72_1.json` / `template_gms_83_1.json` → `"usesPin": false`.
- v61 evidence: `CLogin::OnPacket` @ `0x565668` keeps `OnCheckPinCodeResult`
  (case 6 → `0x5688ce`) and `OnUpdatePinCodeResult` (case 7 → `0x568c0b`)
  **identically to v72/v79/v83**, and it **lacks** the second-password (SPW)
  cases: no case `0x17` (OnEnableSPWResult) and no case `0x1C` (OnCheckSPWResult)
  exist in the switch. The highest explicit case is 22, then the default arm
  forwards `a2 ∈ [92,94]` → `CStage::OnPacket`. v61 is no more PIN/SPW-dependent
  than v72. **`usesPin` carries as `false`.**

---

## (f) Login-flow divergence (OQ-3) — biggest connect risk

`CLogin::OnPacket` v61 @ **`0x565668`** (decompiled). The login opcode layout is
**byte-for-byte identical to the v72 anchor** (same cases, same absent ops):

| op | v61 handler (addr) | v72 op | Δ / note |
|---|---|---|---|
| 0 | OnCheckPasswordResult `0x5657ce` | 0 | same |
| 1 | OnGuestIDLoginResult `0x566290` | 1 | same |
| 2 | **— (no case)** | — (absent in v72 too) | **ACCOUNT_INFO absent** (same as v72) |
| 3 | `sub_56660E` = OnCheckUserLimitResult | 3 | same op (SERVERSTATUS) |
| 4 | OnSetAccountResult `0x56874d` | 4 | same |
| 5 | OnConfirmEULAResult `0x568869` | 5 | same |
| 6 | OnCheckPinCodeResult `0x5688ce` | 6 | same |
| 7 | OnUpdatePinCodeResult `0x568c0b` | 7 | same |
| 8 | OnViewAllCharResult `0x5671b1` | 8 | same |
| 9 | OnSelectCharacterByVACResult `0x56772b` (a2==9 branch) | 9 | same |
| 10 | `sub_56663F` = OnWorldInformation | 10 | same op (WORLD_INFORMATION) |
| 11 | `sub_56688D` = char-list decode | 11 | same op (CHARLIST) |
| 12 | OnSelectCharacterResult `0x5679fb` | 12 | same (SERVER_IP) |
| 13 | symbol `OnCreateNewCharacterResult` `0x566bab` — **body = CHAR_NAME_RESPONSE** | 13 | **same by body** (see rotation proof) |
| 14 | symbol `OnDeleteCharacterResult` `0x566eab` — **body = ADD_NEW_CHAR_ENTRY** | 14 | **same by body** |
| 15 | symbol `OnCheckDuplicatedIDResult` `0x566c86` — **body = DELETE_CHAR_RESPONSE** | 15 | **same by body** |
| 22 | OnSelectWorldResult `0x567ccb` (relog-to-title) | 22 | same (RELOG_RESPONSE) |
| 92/93/94 | default → CStage::OnPacket (SET_FIELD/SET_ITC/SET_CASH_SHOP) | 114/115/116 | **Δ−22** (see §(a)) |
| 23/26/27/28 | **— (no case)** | — (absent in v72 too) | LOGIN_AUTH / LAST_CONNECTED_WORLD / RECOMMENDED_WORLD / CHECK_SPW absent |

### ⚠️ Char-management BODY-verification (the rotated-symbol proof)

The v61 IDB symbol labels on `CLogin::OnPacket` cases 13/14/15 are **rotated one
step off their handler bodies** — identical to the v72/v79 trap. Decompiling the
three BODIES (wire truth) shows v61 char-management is **IDENTICAL to v72/v79/v83**:

| v61 op | case-symbol (IDB, ROTATED) | body read-order (decompiled) | packet = |
|---|---|---|---|
| **13** | `OnCreateNewCharacterResult` @`0x566bab` | `DecodeStr(name) + Decode1(result)` | **CHAR_NAME_RESPONSE** (name-check) |
| **14** | `OnDeleteCharacterResult` @`0x566eab` | `Decode1(result)`; success path `GW_CharacterStat::Decode(v9,a2,0) + AvatarLook::Decode(v9+195,a2)` into free slot | **ADD_NEW_CHAR_ENTRY** |
| **15** | `OnCheckDuplicatedIDResult` @`0x566c86` | `Decode4(cid) + Decode1(result)` + slot-removal (624-byte-entry `qmemcpy` shift + `memcpy`/`memset`) | **DELETE_CHAR_RESPONSE** |

By handler-body behaviour: **CHAR_NAME_RESPONSE=13, ADD_NEW_CHAR_ENTRY=14,
DELETE_CHAR_RESPONSE=15 — same as v72/v79/v83.** Stage B must register these by the
**body-canonical (v83) FName**, not the rotated IDB symbol (mirror the corrected
`gms_v72.yaml` entries). Trusting the symbol would send ADD's stat+avatar payload
to op 13 whose handler leads with `DecodeStr` → client crash.

**Handshake / encoding:**
- `LOGIN_PASSWORD` send `CLogin::SendCheckPasswordPacket` @ `0x564418` builds
  `COutPacket(1)` then `EncodeStr(id) + EncodeStr(pw) + EncodeBuffer(machineId,16)
  + Encode4(gameRoomClient) + Encode1(b) + Encode1(0) + Encode1(0) +
  Encode4(partnerCode)`. Opcode **1 = v72 (Δ0)**; body shape identical.
- **CHARLIST** (`sub_56688D`, op 11): `Decode1 status`; on status∈{0,12,23}
  `Decode1 charCount`, then per slot: `GW_CharacterStat::Decode + AvatarLook::Decode
  + Decode1 rankEnabled` (`Decode1`? `DecodeBuffer(16)` : `memset 16`), then
  `Decode4`. **Same entry shape and opcode (11) as v72.**
- `WORLD_INFORMATION` (`sub_56663F`, op 10): opcode unchanged (10).

**Connect-critical conclusion:** v61's login flow is structurally the **same as
the v72 anchor** — same opcode layout, same absent ops (2, 23/26/27/28), same
char-mgmt body mapping, same LOGIN_PASSWORD / CHARLIST bodies. The only
login-region shift is the CStage SET_FIELD/ITC/CASH_SHOP block at **92/93/94**
(v72 114/115/116, Δ−22).

---

## (a) Opcode map — clientbound (writer) by dispatcher

Mapping = (v61 switch case, by body) → (v72 op via `gms_v72.yaml`). "Δ" is
`v61 − v72`. Each block cites its dispatcher fn+addr.

### CWvsContext::OnPacket @ `0x8303eb` (ops 26–91)

Full switch enumerated. **v61 26–80 is byte-for-byte identical to v72 26–80
(same fname per op, same gaps at 41/51/53/57/61) → Δ0.** Then v61 **drops the
entire v72 81–100 block** (HourChanged, MiniMapOnOff, PartyValue,
FieldSetVariable, BonusExpRateChanged, PotionDiscountRateChanged, Family ops
87–97, NotifyLevelUp, NotifyWedding) and resumes with a compressed tail.

| v61 op | fname (v61) | v72 op | Δ |
|---|---|---|---|
| 26–40 | OnInventoryOperation … OnAntiMacroResult (WEDDING_PHOTO=40) | 26–40 | **0** |
| 42–47 | OnClaimResult … OnEntrustedShopCheckResult | 42–47 | 0 |
| 48 | `sub_841E5F` (= OnSkillLearnItemResult, SKILL_LEARN_ITEM_RESULT) | 48 | 0 |
| 49,50 | OnGatherItemResult, OnSortItemResult | 49,50 | 0 |
| 52,54,55 | OnSueCharacterResult, OnTradeMoneyLimit, OnSetGender | 52,54,55 | 0 |
| 56 | `CUIGuildBBS::OnGuildBBSPacket` (GUILD_BBS_PACKET) | 56 | 0 |
| 58–74 | OnCharacterInfo, OnPartyResult, OnFriendResult, OnGuildResult, OnAllianceResult, OnTownPortal, OnBroadcastMsg, OnIncubatorResult, OnShopScannerResult, OnShopLinkResult, OnMarriageRequest, OnMarriageResult, OnWeddingGiftResult, OnNotifyMarriedPartnerMapTransfer, OnCashPetFoodResult, OnSetWeekEventMessage | 58–74 | 0 |
| 75 | `sub_8422E3` (= OnSetPotionDiscountRate, SET_POTION_DISCOUNT_RATE) | 75 | 0 |
| 76 | OnBridleMobCatchFail | 76 | 0 |
| 77 | `sub_830AFF` (= OnImitatedNPCResult) | 77 | 0 |
| 78 | `sub_830B0B` (= OnImitatedNPCData; routes CNpcPool case 78) | 78 | 0 |
| 79 | OnMonsterBookSetCard | 79 | 0 |
| 80 | OnMonsterBookSetCover | 80 | 0 |
| — | *(v72 ops **81–100 ABSENT in v61**: OnHourChanged, OnMiniMapOnOff, OnPartyValue, OnFieldSetVariable, OnBonusExpRateChanged, OnPotionDiscountRateChanged, Family 87–97, OnNotifyLevelUp, OnNotifyWedding)* | 81–100 | Δ steps 0→**−19** |
| 82 | OnMapleTVUseRes | 101 | −19 |
| 83 | OnAvatarMegaphoneRes | 102 | −19 |
| 84 | OnSetAvatarMegaphone | 103 | −19 |
| 85 | OnClearAvatarMegaphone | 104 | −19 |
| 86 | OnCancelNameChangeResult | 105 | −19 |
| 87 | OnCancelTransferWorldResult | 106 | −19 |
| 88 | OnDestroyShopResult | (107) | −19 |
| 89 | OnFakeGMNotice | (108) | −19 |
| 90 | OnSuccessInUsegachaponBox | (109) | −19 |
| 91 | OnMacroSysDataInit (MACRO_SYS_DATA_INIT) | 113 | **−22** |

**Absent in v61 vs the v72 anchor (CWvsContext):** the whole v72 81–100 block
(19 ops: HourChanged, MiniMapOnOff, PartyValue, FieldSetVariable,
BonusExpRateChanged, PotionDiscountRateChanged, Family 87–97, NotifyLevelUp,
NotifyWedding), plus v72's extra name-change/cancel ops in 107–112 that v61's
88–90 does not cover (e.g. OnCancelNameChangebyOther). **Stage B maps the v61
tail (82–91) case-by-case against `gms_v72.yaml` (checked, not a uniform shift);
88/89/90 v72-op numbers above are the Δ−19 alignment guess and must be
FName-confirmed against the v72 registry.**

### CStage::OnPacket @ `0x659f99` (ops 92–94) — Δ **−22**
`'\'`(92) OnSetField (SET_FIELD) · `']'`(93) OnSetITC (SET_ITC) · `'^'`(94)
OnSetCashShop (SET_CASH_SHOP). v72 = 114/115/116.

### CField::OnPacket @ `0x4e9ea3` (base ops 97–119) — Δ **−24**
Base char-case switch (op = ASCII):
`'a'`(97) OnTransferFieldReqIgnored (BLOCKED_MAP) · `'b'`(98)
OnTransferChannelReqIgnored · `'c'`(99) OnFieldSpecificData (FORCED_MAP_EQUIP) ·
`'d'`(100) OnGroupMessage (MULTICHAT) · `'e'`(101) OnWhisper (WHISPER) · `'f'`(102)
OnCoupleMessage (SPOUSE_CHAT) · `'g'`(103) OnSummonItemInavailable · `'h'`(104)
`sub_4EB523` · `'i'`(105) OnFieldObstacleOnOffStatus · `'j'`(106) `sub_4ED39C` ·
`'k'`(107) OnAdminResult · `'l'`(108) OnQuiz · `'m'`(109) OnDesc · `'n'`(110)
vtable+0x20 (CLOCK, OnClock) · `'q'`(113) OnSetQuestClear · `'r'`(114)
OnSetQuestTime · `'s'`(115) OnWarnMessage (ARIANT_RESULT) · `'t'`(116)
OnSetObjectState · `'u'`(117) OnDestroyClock (STOP_CLOCK) · `'w'`(119) `sub_4EFABF`.
Whisper at `'e'`=101 vs v72 123 → Δ−22 for the named social ops; the base region
sits around Δ−22..−24. Gaps at `'o'`(111)/`'p'`(112) and `'v'`(118) mirror the
v72 base-switch subclass gaps (CONTI_MOVE/CONTI_STATE/ARIANT_ARENA family),
shifted. `235`(OnHontailTimer) sits above the base range as its own case.

### CField pool-routing ranges (read from `0x4e9ea3`) and per-pool deltas
CField dispatches these opcode ranges to the pools:

| pool | v61 dispatcher | v61 range | v72 range | Δ (start) |
|---|---|---|---|---|
| CUserPool | `0x7bd7f3` | 120–174 | 145–207 | **−25** |
| CMobPool | `0x5d4894` | 175–193 | 208–226 | **−33** |
| CNpcPool | `0x5efbad` | 194–201 | 227–234 | −33 |
| CEmployeePool | `0x4d3450` | 202–204 | 235–237 | −33 |
| CDropPool | `0x4c9163` | 205–206 | 238–239 | −33 |
| CMessageBoxPool | `0x5bc188` | 207–209 | 240–242 | −33 |
| CAffectedAreaPool | `0x423eb7` | 210–211 | 243–244 | −33 |
| CTownPortalPool | `0x68745a` | 212–213 | 245–246 | −33 |
| CReactorPool | `0x633133` | 214–217 | 247–250 | −33 |

The Δ deepens **−25 → −33** across the user region (below); Mob and all lower
map-pools settle at Δ−33.

### CUserPool::OnPacket @ `0x7bd7f3` — enter/leave Δ **−25**
120 OnUserEnterField (SPAWN_PLAYER) · 121 OnUserLeaveField
(REMOVE_PLAYER_FROM_MAP). Routes 122–140→common, 141–159→remote, 160–173→local.
(v72 145/146; common 147–165, remote 167–186, local 187–206.)

### CUserPool::OnUserLocalPacket → CUserLocal::OnPacket @ `0x7a451a` (ops 160–173) — Δ **−27 → −33**
160 OnSitResult (CANCEL_CHAIR) · 161 OnEffect (SHOW_ITEM_GAIN_INCHAT) · 162
OnTeleport (DOJO_WARP_UP) · 164 OnMesoGive_Succeeded (LUCKSACK_PASS) · 165
OnMesoGive_Failed (LUCKSACK_FAIL) · 166 OnQuestResult (UPDATE_QUEST_INFO) · 167
OnNotifyHPDecByField (**v61/v72-extra**, no v83 registry op) · 168 `nullsub_13`
(dead) · 169 OnBalloonMsg (PLAYER_HINT) · 170 OnPlayEventSound · 171
OnPlayMinigameSound · 172 OnRandomEmotion (RANDOM_EMOTION) · 173 OnSkillCooltimeSet
(COOLDOWN). Gap at 163 mirrors the v72 local gap at 190.

**v61 local is SHORTER than v72:** v61 lacks v72's `OnMakerResult` (199),
`OnOpenClassCompetitionPage`/KOREAN_EVENT (201), `OnSetDirectionMode`/LOCK_UI
(202), `OnSetStandAloneMode`/DISABLE_UI (203), and `OnHireTutor`/SPAWN_GUIDE
(204). After OnPlayMinigameSound(171) v61 jumps straight to RANDOM_EMOTION(172) /
COOLDOWN(173). This 5-op local shrinkage is what deepens the map Δ from −25 (user
enter) to −33 (RANDOM_EMOTION 172 vs v72 205).

### CUserPool common/remote (ops 122–140 / 141–159) — Δ **−25 → −27**
`OnUserCommonPacket` @ `0x7bdb3e` (122–140) and `OnUserRemotePacket` @ `0x7bdbda`
(141–159). Per-leaf order matches v72 (CHATTEXT/CHATTEXT1, chalkboard, pet
cluster, summon cluster in common; MOVE_PLAYER, attack family, skill, hit,
emotion, buffs in remote). v61 has no gap between common(140) and remote(141)
where v72 had a gap at 166 → Δ deepens −25→−26 at the remote boundary; remote is
1 op narrower than v72 → −27 by the local boundary. **Stage B diffs the common
and remote internal leaf switches against v72 to place each op precisely.**

### CMobPool::OnPacket @ `0x5d4894` — Δ **−33**
175 OnMobEnterField (SPAWN_MONSTER) · 176 OnMobLeaveField · 177
OnMobChangeController · 188 OnMobCrcKeyChanged (MOB_CRC_KEY_CHANGED, carved out) ·
178–192 OnMobPacket (MOVE_MONSTER …). Enter/leave/controller/crckey uniform Δ−33
(v72 208/209/210, crckey 221). Mob-packet leaf range 178–192 (15 leaves). Stage B
diffs `CMobPool::OnMobPacket` @ `0x5d48f3` internal leaf switch against v72
(`0x62560d`) for the per-leaf order.

### CNpcPool::OnPacket @ `0x5efbad` (ops 194–201) — Δ **−33**
194 OnNpcEnterField (SPAWN_NPC) · 195 OnNpcLeaveField · 196 OnNpcChangeController
· 197–199 OnNpcPacket (NPC_ACTION …) · 200 `sub_5EFDA2` (= OnSetNpcScript,
SET_NPC_SCRIPTABLE). Case 78 = `OnNpcImitateData` (the routed IMITATED_NPC_DATA
target from CWvsContext op 78 — a dead-case, not a 78-opcode dispatch; same
family as v72). Uniform Δ−33, same leaf order as v72 (227–234).

### Map pools (range-routed from CField) — all Δ **−33**
Per-pool leaf order matches v72; only the base offset shifts −33:
- CEmployeePool 202–204 → SPAWN/REMOVE/UPDATE_HIRED_MERCHANT
- CDropPool 205–206 → DROP_ITEM_FROM_MAPOBJECT / REMOVE_ITEM_FROM_MAP
- CMessageBoxPool 207–209 → CANNOT_SPAWN_KITE / SPAWN_KITE / REMOVE_KITE
- CAffectedAreaPool 210–211 → SPAWN_MIST / REMOVE_MIST
- CTownPortalPool 212–213 → SPAWN_DOOR / REMOVE_DOOR
- CReactorPool 214–217 → REACTOR_HIT / MOVE / SPAWN / DESTROY

### CField high-region routes (dialog / script / dispatcher-families) @ `0x4e9ea3`
235 OnHontailTimer · 236 CScriptMan::OnPacket · 237–238 CShopDlg · 239
`sub_6910E4` · 240–241 CStoreBankDlg · 242 CTrunkDlg · 243 `sub_6D34F1` · 244
`sub_5BEC69` · 252 CRPSGameDlg · 262–264 CFuncKeyMappedMan · 267–272 `sub_59BD51`
· 95–96 `sub_5A81B9`. (These demuxers use a secondary MODE byte, not the top-level
opcode — Stage E family work; see §(OQ-7).)

### CCashShop::OnPacket @ `0x4610a4` (ops 253–260) — clean Δ **−36** shift of v72
253 OnNoticeFreeCashItem · 254 OnOneADay · 255 OnCashItemResult · 256
OnCashItemGachaponResult · 257 `sub_463900` · 258 OnCashShopGachaponStampResult ·
260 OnCheckTransferWorldPossibleResult (gap at 259, mirroring v72's gap at 295).
**Unlike the v72↔v79 cash-shop divergence, v61's cash-shop block is a clean
Δ−36 shift of the v72 block (289–296) with the SAME membership and same relative
gap.** Cash-shop is a separate stage outside the core login→map flow — Stage B
maps 253–260 against the v72 `CCashShop::OnPacket` case-by-case, but the shift is
uniform here.

---

## (a) Opcode map — serverbound (handler) — from v61 `COutPacket(N)` send-sites

No CSV v61 serverbound column exists; opcodes are read from the `COutPacket(N)`
constructor at each send-site. **The serverbound space is its own enumeration:
core-flow ops are Δ−2 vs v72 in the map region, deepening to Δ−10 in the party
region; LOGIN_PASSWORD is Δ0.** All verified:

| op (serverbound) | v61 send-site (addr) | v61 opcode | v72 op | Δ vs v72 |
|---|---|---|---|---|
| LOGIN_PASSWORD | `CLogin::SendCheckPasswordPacket` `0x564418` | `COutPacket(1)` | 1 | **0** |
| CHANGE_MAP | `CField::SendTransferFieldRequest` `0x4e8f58` | `COutPacket(35)` | 37 | **−2** |
| CHANGE_CHANNEL | `CField::SendTransferChannelRequest` `0x4e90ab` | `COutPacket(36)` | 38 | **−2** |
| MOVE_PLAYER | move flush `sub_801109` `0x801109` (after `CVecCtrlUser::OnSit` `0x8010e2`; via `xrefs_to(CMovePath::Flush 0x5e2ca3)`) | `COutPacket(38)` | 40 | **−2** |
| PARTY_OPERATION | `CField::SendCreateNewPartyMsg` `0x4e898b` | `COutPacket(112)` | 122 | **−10** |

Body shapes verified at each send-site (all match v72/v83):
- **CHANGE_MAP** = `Enc1(portalByte)+Enc4(targetMap)+EncStr(portal)+[Enc2(x)+Enc2(y)]
  +Enc1(0)+Enc1(wheel)+Enc1(premiumFlag)[+Enc4+Enc4]`.
- **CHANGE_CHANNEL** = `Enc1(channel)+Enc4(t)`.
- **MOVE_PLAYER** = `Enc1(fieldKeyByte)` then `CMovePath::Flush`.
- **PARTY_OPERATION** create = `COutPacket(112) Enc1(1)`. The accept/decline replies
  emitted from `OnPartyResult` (@`0x857a8c`) use `COutPacket(112)` (accept:
  `Enc1(3)+Enc4(partyId)+Enc1(level)`) and `COutPacket(113)` (block/report:
  `Enc1(21|22)+EncStr+EncStr`). v72 = 122/123.

> **The full serverbound opcode table is a Stage B deliverable** — derive each op
> from its v61 send-site anchored on the v72 serverbound FName. Stage A fixes the
> method + the five core-flow anchors above. Additional located send-sites for
> Stage B: `CField::SendJoinPartyMsg` `0x4e8b29`, `SendKickPartyMsg` `0x4e8cfb`,
> `SendCreateGuildAgreeMsg` `0x4e9260`, `SendInviteGuildMsg` `0x4e92d1`,
> `SendKickGuildMsg` `0x4e95f7`, `SendSetGuildMarkMsg` `0x4e9b0d`,
> `SendAcceptFriendMsg` `0x4e9df4`, `SendWithdrawExpeditionMsg` `0x744952`.
> **WHISPER serverbound** (v72 = 118) is **UNVERIFIED** — v61 has no named
> `SendChatMsgWhisper` symbol (only the clientbound `CField::OnWhisper` @
> `0x4eabd7`). The Δ−2 map-region pattern predicts ~116 but Stage B must read it
> from the actual whisper send-site (do not seed the predicted value blind).
> **NPC_TALK serverbound** likewise: no named `TalkToNpc` symbol in v61 — Stage B
> reads it from the npc-select send-site.

---

## (b) Operation / mode (sub-op) tables — from v61 dispatcher switches

Read **from the v61 switch**; do not inherit v72's values.

### Status-message — `CWvsContext::OnMessage` @ `0x8437ef` (op 36 / SHOW_STATUS_INFO)
Leading `Decode1` mode; switch arms **0,1,2,3,4,5,6,7,8,9** — **10 arms (0–9),
default = drop.** **This is NARROWER than v72 (0–11, 12 arms) and v83 (0–13, 14
arms).** v61 lacks modes 10 and 11 (and 12/13). **Stage C consequence: the v61
`operations` table for SHOW_STATUS_INFO must be 0–9, contiguous, and must have NO
spurious `INCREASE_SKILL_POINT`** (the off-by-one trap that crashed v83 on spawn
applies here too — keep the table contiguous 0–9 matching the IDA switch).

### World/broadcast message — `CWvsContext::OnBroadcastMsg` @ `0x844d49` (op 65 / SERVERMESSAGE)
Leading `Decode1` type; switch cases **0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xA** (types
3 and 8 share the item-speaker path; 9 and 0xA share) — **11 arms (0–10), default
= drop.** **NARROWER than v72 (0–13, 14 arms):** v61 lacks types 11 (item
megaphone), 12, 13 (pink/notice). Notable bodies: type 4 = management speaker
(reads extra leading `Decode1` whisper-flag); type 3 = `Decode1+Decode1`; type 8
= `Decode1+Decode1+Decode1? GW_ItemSlotBase::Decode` (item-speaker); type 7 =
util-dlg (`Decode4`); types 9/0xA = notice variants. **SERVERMESSAGE op = 65
(Δ0 vs v72).** Stage C: v61 broadcast operations table = 0–10.

### Party — `CWvsContext::OnPartyResult` @ `0x857a8c` (op 59 / PARTY_OPERATION)
Leading `Decode1` mode. **v61 arms present (hex):** `4, 7, 8, 9, 0xA, 0xC, 0xD,
0xF, 0x10, 0x11, 0x12, 0x13, 0x15, 0x16, 0x17, 0x1A, 0x1B, 0x1C, 0x1D, 0x1F, 0x20,
0x21, 0x22, 0x23, 0x24` (+ default; `0x21` shares the load/refresh arm with `7`).

**⚠️ The v61 party mode table is IDENTICAL to the v72 anchor** (same arm set, same
bodies): `0x1A` present (invite/FindUser join-notice), `0x1F` present, HP/coord
slot-update at **`0x24`** (`Decode1 slot<6` else `CDisconnectException` +
`Decode4 + Decode4 + Decode2 + Decode2`), **no `0x25`**, `0x22` = HP/max update
(`Decode4 cid + Decode4 + Decode4`, FindUser). Notable bodies: `4` invite →
`Decode4(partyId)+DecodeStr(inviter)+Decode1(level)`, builds YESNO + serverbound
`COutPacket(112)` accept / `COutPacket(113)` block-report; `7`/`0x21` load →
`Decode4+PARTYDATA::Decode`; `8` join → `Decode4+Decode4+Decode4+Decode2+Decode2`;
`0xC` leave/expel → `Decode4+Decode4+Decode1(+DecodeStr on notice)`; `0xF` join
member → `Decode4+DecodeStr+PARTYDATA::Decode`. **Party operations table carries
from v72 unchanged** (unlike the v72↔v79 party shift).

### Other dispatcher families with mode tables (located, not byte-extracted)
`CWvsContext::OnGuildResult` @ `0x851543` (op 62) · `CWvsContext::OnFriendResult`
@ `0x85898e` (op 60, buddy) · `CWvsContext::OnAllianceResult` @ `0x853fb7`
(op 63) · `CWvsContext::OnInventoryOperation` @ `0x8422fc` (op 26) ·
`CUIGuildBBS::OnGuildBBSPacket` @ `0x8399af` (op 56) · messenger routing (CField
high-region) · shop/trunk/storebank dialog demuxers (CField 237–242). Stage E
owns per-mode extraction — see §(OQ-7).

---

## (c) Structure / encoding deltas vs v72 — login → channel → map → movement/chat + tier-1

Swept (not sampled) for the connect-critical flow:

| flow stage | v61 packet | structure vs v72 | evidence |
|---|---|---|---|
| login | LOGIN_PASSWORD (sb 1) | **same body** | `SendCheckPasswordPacket` 0x564418 |
| login | CHARLIST (cb 11) | **same entry shape** (CharacterStat+AvatarLook+rankFlag+[16B]+Decode4); op 11 both | `sub_56688D` 0x56688d |
| login | WORLD_INFORMATION (cb 10) | same; op 10 both | `sub_56663F` |
| login | char-mgmt (cb 13/14/15) | **same by body** (rotated symbols); §(f) | 0x566bab / 0x566eab / 0x566c86 |
| channel | CHANGE_CHANNEL (sb 36) | same body `Enc1(ch)+Enc4` | `0x4e90ab` |
| map | CHANGE_MAP (sb 35) | same body | `0x4e8f58` |
| map | SET_FIELD (cb 92) | dispatcher confirmed (`CStage::OnSetField` `0x659fd3`); body not byte-diffed in Stage A | `0x659f99` |
| movement | MOVE_PLAYER (cb 160-region remote) | remote path via `CUserPool::OnUserRemotePacket` | `0x7bdbda` |
| movement | MOVE_PLAYER (sb 38) | same body (`Enc1(fieldKey)`+CMovePath::Flush) | `sub_801109` 0x801109 |
| chat | general/whisper (cb: MULTICHAT 100, WHISPER 101, SPOUSE_CHAT 102) | OnGroupMessage/OnWhisper/OnCoupleMessage present at base 'd'/'e'/'f'; bodies via same `*::Decode` | `0x4e9ea3`, OnWhisper `0x4eabd7` |
| chat | CHATTEXT (cb common 122-region) | `CUser::OnChat` two-arm same as v72 | `0x7bdb3e` |

**No wire-structure delta was found in the swept core flow** beyond opcode
renumbering (§a/f) — bodies match v72 at every send/decode site read. The v61
risk surface is **opcode numbering + absent ops + the two shrunk mode tables
(status-message 0–9, broadcast 0–10)**, not body encoding, for the
login→map→movement→chat path. (Party mode table is unchanged from v72.)

**tier-1 note (`docs/packets/evidence/tiers.yaml`):** `summon/clientbound/
SummonSpawn` version-gated trailing avatar-look byte is `GMS≥95` only. v61 < v72
< v95, so SummonSpawn has **no trailing avatar byte** in v61 — same as the v72
anchor. Other tier-1 prefix families are opcode-mapped above; their opaque bodies
decode via the same named `*::Decode` helpers as v72 — Stage E byte-fixtures
confirm per packet.

## (d) "Same as anchor (v72)" — entries with explicit switch/body evidence

Verified equal (not defaulted):
- **Login ops 0,1,3,4,5,6,7,8,9,10,11,12,22** — v61 `CLogin::OnPacket` cases match
  v72 case-for-case (same absent ops 2/23/26/27/28).
- **Char-mgmt body mapping 13/14/15** — CHAR_NAME/ADD/DELETE identical to v72 by
  decompiled read-order (rotated symbols on both).
- **CHARLIST + WORLD_INFO + LOGIN_PASSWORD bodies** — same read/send order, same
  opcodes (11/10/1).
- **CWvsContext ops 26–80** — identical fname sequence, Δ0.
- **Party mode table (op 59)** — identical arm set 4..0x24 and bodies to v72,
  including slot-update at 0x24, no 0x25, 0x1A/0x1F present.
- **CCashShop membership** — same op set as v72's 289–296 block, uniformly
  shifted Δ−36 to 253–260 (same relative gap at 259).
- **Mob/Npc/employee/drop/kite/mist/door/reactor leaf order** — uniform Δ−33 with
  no reordering.
- **No dragon remote arm; no CLEAR_BACK_EFFECT** — same absences as v72.

---

## (OQ-7) Dispatcher-family list for Stage E (sizes the per-version mode campaign)

| family | v61 dispatcher (addr) | v61 cb op | mode table vs v72 |
|---|---|---|---|
| status-message | `CWvsContext::OnMessage` 0x8437ef | 36 | **DIFFERS — arms 0–9 only (v72 0–11)**; no INCREASE_SKILL_POINT |
| worldmessage/broadcast | `CWvsContext::OnBroadcastMsg` 0x844d49 | 65 | **DIFFERS — arms 0–10 only (v72 0–13)**; no item-megaphone/pink/notice 11–13 |
| party | `CWvsContext::OnPartyResult` 0x857a8c | 59 | **SAME as v72** (arms 4..0x24, slot-update 0x24, no 0x25) |
| guild | `CWvsContext::OnGuildResult` 0x851543 | 62 | not extracted — Stage E |
| alliance | `CWvsContext::OnAllianceResult` 0x853fb7 | 63 | not extracted — Stage E |
| buddy/friend | `CWvsContext::OnFriendResult` 0x85898e | 60 | not extracted — Stage E |
| guild-BBS | `CUIGuildBBS::OnGuildBBSPacket` 0x8399af | 56 | not extracted — Stage E |
| inventory-op | `CWvsContext::OnInventoryOperation` 0x8422fc | 26 | not extracted — Stage E |
| shop/trunk/storebank | CShopDlg 0x64723c / CTrunkDlg 0x63bf0e / CStoreBankDlg 0x6755e3 | 237–242 | body-mode demuxers; Stage E |
| field-effect family | `sub_4EB523` (base 'h'=104) | 104 | Stage E (v72 FIELD_EFFECT counterpart) |
| cashshop | `CCashShop::OnPacket` 0x4610a4 | 253–260 | clean Δ−36 shift of v72 — Stage B |

**Campaign size:** ~10 dispatcher families need per-version operation-table
extraction. status-message (0–9), broadcast (0–10), party (= v72) are captured
here; the remaining families are located (addresses above) but not byte-extracted.

---

## Escalations / open items handed to Stage B/E (none blocking Stage A)

1. **CSV v61 column is placeholder-only** — seed v61 from this doc/IDB, not the CSV.
2. **Char-mgmt symbols are rotated** (§f) — register cb ops 13/14/15 by the
   body-canonical (v83) FName, exactly as `gms_v72.yaml` was corrected. NOT the
   IDB symbol label.
3. **Status-message mode table SHRINKS to 0–9** (v72 was 0–11) — the v61
   SHOW_STATUS_INFO `operations` table must have 10 arms; keep contiguous, no
   INCREASE_SKILL_POINT.
4. **Broadcast mode table SHRINKS to 0–10** (v72 was 0–13) — SERVERMESSAGE
   operations table must be 11 arms (drop item-megaphone/pink/notice 11–13).
5. **Party mode table = v72 unchanged** — carry the v72 party operations table.
6. **Serverbound is Δ−2 (LOWER) than v72 in the map region** (CHANGE_MAP 35,
   CHANGE_CHANNEL 36, MOVE_PLAYER 38) and **Δ−10 in the party region** (PARTY
   112/113). Login region is Δ0 (LOGIN_PASSWORD 1). Stage B derives the full sb
   table per send-site (non-uniform).
7. **WHISPER serverbound UNVERIFIED** — no named `SendChatMsgWhisper` in v61;
   predicted ~116 by the −2 map-region pattern but Stage B must read it from the
   whisper send-site. **NPC_TALK serverbound** likewise (no named `TalkToNpc`).
8. **CWvsContext 81–100 block ABSENT in v61** (19 ops: HourChanged, MiniMapOnOff,
   PartyValue, FieldSetVariable, BonusExpRateChanged, PotionDiscountRateChanged,
   Family 87–97, NotifyLevelUp, NotifyWedding) — the clientbound tail 82–91 maps
   to v72 101–113 case-by-case; Stage B FName-confirms 88/89/90 against v72.
9. **CUserLocal drops MakerResult / KOREAN_EVENT / LOCK_UI / DISABLE_UI /
   SPAWN_GUIDE** (v72 199/201/202/203/204) → local Δ deepens −25→−33; v61 local
   is 160–173.
10. **CMobPool leaf order** — diff `CMobPool::OnMobPacket` (`0x5d48f3`) internal
    switch against v72 (`0x62560d`) for the per-leaf MOVE_MONSTER family order.
11. **v61-extra top-level case 0x15** (`sub_4747E2`, `ProcessPacket` @ 0x47440a)
    — map its FName in Stage B if a handshake fixture needs it.
12. **v61/v72-extra op with no v83 registry equiv:** `CUserLocal::OnNotifyHPDecByField`
    (cb 167) — Stage B decides register vs no-op (same as v72's op 194).
13. **CField subclass overrides** (ContiMove/Massacre/AriantArena, gaps at base
    'o'/'p' = 111/112 and 'v' = 118) not walked for v61 — confirm CONTI_MOVE/
    PYRAMID presence in Stage B.

Every unnamed sub above was identified from its decompiled read-order / send-opcode
(`sub_56660E/56663F/56688D`, `sub_841E5F/8422E3/830AFF/830B0B`, `sub_5EFDA2`,
`sub_4EB523/4ED39C/4EFABF`, `sub_801109`, `sub_463900`, `sub_4E898B`). No
unresolved fname required fabrication.
