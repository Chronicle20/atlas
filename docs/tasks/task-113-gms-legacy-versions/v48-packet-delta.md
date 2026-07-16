# GMS v48 — packet delta vs v61 (Stage A)

> **Source-of-truth delta doc for the v48 pass.** Stage B (registry), Stage C
> (template), and Stage E (verification campaign) consume this. **Anchor =
> `gms_v61`** (the completed prior pass — its delta doc `v61-packet-delta.md`,
> registry `gms_v61.yaml` [220 cb + 106 sb, serverbound corrected after a Stage-B
> scramble → now reliable], template `template_gms_61_1.json`, and export
> `gms_v61.json`). Every opcode / mode / structure claim below cites the v48 IDB
> (`GMS_v48_1_DEVM.exe`, port 13337) by function BODY (switch-case, decompiled
> read-order, or `COutPacket(N)` send-site) or explicit v61-anchor evidence with
> body confirmation. **v48 is the OLDEST version in scope — below every gate.**

## IDA pre-flight (re-confirmed by binary name via `list_instances`)

| Role | Port | Binary | idb |
|---|---|---|---|
| **v48 target** | **13337** | `GMS_v48_1_DEVM.exe` | `E:\...\GMS\v48\GMS_v48_1_DEVM.exe.i64` |
| v61 anchor | 13338 | `GMS_v61.1_U_DEVM.exe` | `E:\...\GMS\v61\GMS_v61.1_U_DEVM.exe.i64` |
| v72 | 13339 | `GMS_v72.1_U_DEVM.exe` | — |
| v79 | 13340 | `GMS_v79_1_DEVM.exe` | — |
| v95 tie-breaker | 13341 | `GMS_v95.0_U_DEVM.exe` | — |
| v83 | 13342 | `MapleStory_dump.exe` | — |

All reachable. `select_instance(13337)` for every v48 read below. **The v48 IDB
carries RAW mangled MSVC symbols (NOT demangled)** — e.g.
`?OnPacket@CWvsContext@@QAEXJAAVCInPacket@@@Z`. Many core-flow handlers
(`CLogin::OnPacket`, `CStage`/set-field, char-mgmt, pools) are **unnamed
`sub_XXXX`** (no symbol at all), so there is no symbol to be rotated — but every
opcode below is still matched by handler BODY, never by any label. The
`GMS v48` CSV column is the literal placeholder `0x000`; **every v48 opcode here
is derived from the v48 IDB.**

---

## Top-level routing — the shim + CWvsContext window

`CClientSocket::ProcessPacket` @ **`0x464fb6`** (`CInPacket::Decode2` scrutinee):

```
case 0x10: OnMigrateCommand    0x11: OnAliveReq   0x12: OnAuthenCodeChanged
0x13: OnAuthenMessage          0x14: sub_733B6B   0x15: sub_465378
default: if (op in [0x18,0x47]) -> CWvsContext::OnPacket(op)  [off_80C8A0]
         else                   -> current-stage vtable+8 (CLogin / CField)::OnPacket(op)
```

The CWvsContext window is **`[0x18,0x47]` (24–71)** in v48 vs **`[0x1A,0x5B]`
(26–91)** in v61. **Both bounds are LOWER and the window is NARROWER:** lower
bound **−2** (0x18 vs 0x1A), upper bound **−20** (0x47 vs 0x5B → 20 fewer
CWvsContext ops than v61). **Non-uniform shift confirmed: there is no single
global offset** — login ops 3–22 sit at their v61 numbers, CWvsContext starts
Δ−1 and deepens, the CField map region is ≈Δ−20, and the serverbound space is
its own enumeration (below).

**Top-level cases 0x14 / 0x15 differ from v61.** v61 had `0x14 →
OnPacket_CSecurityClient` and a v61-extra `0x15 → sub_4747E2`. v48 has
`0x14 → sub_733B6B` and `0x15 → sub_465378` (both present; handshake-family, not
connect-critical). Stage B maps their FNames if a migrate/authen fixture is built.

---

## (e) usesPin (OQ-2) — **false** for v48

- Anchor: `template_gms_61_1.json` / `template_gms_72_1.json` → `"usesPin": false`.
- v48 evidence: the login dispatch `sub_5007C4` (@ `0x5007c4`, CLogin::OnPacket —
  see §f) keeps the PIN slots at **case 6 (`sub_503956`)** and **case 7
  (`sub_503C92`)** in the same op positions as v61 (op 6 = OnCheckPinCodeResult,
  op 7 = OnUpdatePinCodeResult), and **lacks any second-password (SPW) case**: no
  case `0x17` (OnEnableSPWResult) and no case `0x1C` (OnCheckSPWResult) — the
  highest explicit login case is **22**. v48 is no more PIN/SPW-dependent than
  v61. **`usesPin` carries as `false`.**

---

## (f) Login-flow divergence (OQ-3) — biggest connect risk

`CLogin::OnPacket` v48 = **`sub_5007C4` @ `0x5007c4`** (resolved via the CLogin
vtable: ctor `0x4fdc72` sets the OnPacket sub-object vtable ptr to the value at
`off_79EA08` = `0x5007c4`; the dispatch site is `ProcessPacket`'s `(**(v6+8))`).
Decompiled switch scrutinee = raw opcode:

```
switch(op) {
  case 1: sub_500931   case 2: nullsub_11 (DEAD)   case 3: sub_5011D6
  case 4: sub_5037D5   case 5: sub_5038F1          case 6: sub_503956
  case 7: sub_503C92   case 8: sub_50232D          case 9: sub_5028A6
  case 10: sub_50120A  case 11: sub_5013ED         case 12: sub_502B70
  case 13: sub_5016DB  case 14: sub_501973         case 15: sub_5017B6
  case 22: sub_502E3A  default: if (op in [72,75]) sub_5C45ED  // SET_FIELD region
}
```

### ⚠️ Login opcode divergence vs v61 — the login RESULT is at op **1**, not 0

**There is NO case 0 in the v48 login switch**, and **case 1 (`sub_500931`) is
the authentication/login-result handler** (body: `Decode1(result) +
Decode1(regStatus→this+344) + Decode4`, a large result-code → error-dialog
cascade, then on `result ∈ {0,12,23}` it decodes account/character data,
`CWvsContext::SetCharacterData`, and emits the game-server migrate
`COutPacket(8) Enc1(1)+Enc1(1)+Enc4(id)+EncStr`). This is **OnCheckPasswordResult**
by body. So **v48 LOGIN_STATUS (auth result) = op 1** (v61 = op 0); **op 0 is
unused** and **case 2 is a dead `nullsub`** (v61 op 1 = OnGuestIDLoginResult is
absent). **Every other login op 3–22 is IDENTICAL to v61** (anchored below):

| op | v48 handler | body / identity | v61 op | Δ / note |
|---|---|---|---|---|
| 0 | **— (no case)** | — | 0 (CheckPassword) | **login result MOVED to op 1** |
| 1 | `sub_500931` | Decode1 result + regStatus + Decode4, err-cascade, SetCharacterData + `COutPacket(8)` migrate | (0) | **= OnCheckPasswordResult by body** |
| 2 | `nullsub_11` | dead | 1 (GuestID) | **GuestIDLogin absent** |
| 3 | `sub_5011D6` | (SERVERSTATUS slot) | 3 | same op |
| 4 | `sub_5037D5` | (GENDER_DONE slot) | 4 | same op |
| 5 | `sub_5038F1` | (CONFIRM_EULA slot) | 5 | same op |
| 6 | `sub_503956` | (CHECK_PINCODE slot) | 6 | same op |
| 7 | `sub_503C92` | (UPDATE_PINCODE slot) | 7 | same op |
| 8 | `sub_50232D` | **OnViewAllCharResult** — Decode1 sub-mode 0–7, per-char AvatarLook::Decode + world-count accum | 8 | **same by body** (VIEW_ALL_CHAR) |
| 9 | `sub_5028A6` | (SELECT_BY_VAC slot) | 9 | same op |
| 10 | `sub_50120A` | (WORLD_INFORMATION slot) | 10 | same op |
| 11 | `sub_5013ED` | **CHARLIST** — Decode1 status; on status∈{0,12,23} Decode1 count, per slot `GW_CharacterStat::Decode`(`sub_49B627`)+`AvatarLook::Decode`+Decode1?DecodeBuffer(16):memset | 11 | **same by body** (see structure delta) |
| 12 | `sub_502B70` | (SERVER_IP / OnSelectCharacterResult slot) | 12 | same op |
| 13 | `sub_5016DB` | **CHAR_NAME_RESPONSE** — `DecodeStr(name)+Decode1(result)` | 13 | **same by body** |
| 14 | `sub_501973` | **ADD_NEW_CHAR_ENTRY** — `Decode1(result)`; success: `GW_CharacterStat::Decode`(`sub_49B627`)+`AvatarLook::Decode` into free slot | 14 | **same by body** |
| 15 | `sub_5017B6` | **DELETE_CHAR_RESPONSE** — `Decode4(cid)+Decode1(result)`+slot-removal (436-byte-entry `qmemcpy` shift + `memmove`/`memset`) | 15 | **same by body** |
| 22 | `sub_502E3A` | (RELOG_RESPONSE slot) | 22 | same op |

### ⚠️ Char-management BODY-verification (cases 13/14/15)

Unlike v61 (where the IDB SYMBOL labels on cases 13/14/15 were rotated one step
off their bodies), the v48 handlers are **unnamed `sub_XXXX`** so there is no
misleading label. Decompiling the three bodies confirms the **same body→op
mapping as v61/v83**: **CHAR_NAME_RESPONSE = 13, ADD_NEW_CHAR_ENTRY = 14,
DELETE_CHAR_RESPONSE = 15.** Stage B registers these three by the body-canonical
(v83) FName (mirror `gms_v61.yaml`), keyed to the v48 addresses above.

### Handshake / encoding bodies

- **LOGIN_PASSWORD** send `CLogin::SendCheckPasswordPacket` @ `0x4ffeb2` builds
  `COutPacket(1)` then `EncodeStr(id)+EncodeStr(pw)+EncodeBuffer(machineId,16)+
  Encode4(gameRoomClient)+Encode1(b)+Encode1(0)`. Opcode **1 = v61 (Δ0)**, BUT the
  body is **SHORTER than v61**: v48 stops after `Enc1(b)+Enc1(0)` and sends — it
  **omits v61's trailing `Encode1(0)+Encode4(partnerCode)`**. ⚠️ structure delta.
- **CHARLIST** (`sub_5013ED`, op 11): per-slot `GW_CharacterStat::Decode +
  AvatarLook::Decode + Decode1 rankFlag?DecodeBuffer(16):memset`, then the loop
  **ENDS — v48 does NOT read the trailing `Decode4`** that v61's CHARLIST had
  after the slot loop. ⚠️ structure delta (one fewer trailing field).
- **VIEW_ALL_CHAR** (`sub_50232D`, op 8): sub-mode 0 = per-char
  `GW_CharacterStat::Decode + AvatarLook::Decode + Decode1?DecodeBuffer(16):memset`;
  sub-mode 1 = `Decode4 count + Decode4`; same shape family as v61.

**Connect-critical conclusion:** v48's login flow is **structurally close to v61
but with three real divergences**: (1) the auth/login-result opcode is **1** (v61
0), op 0 unused, GuestID (v61 op 1) absent; (2) **LOGIN_PASSWORD body is 2 fields
shorter** (no trailing `Enc1(0)+Enc4(partnerCode)`); (3) **CHARLIST has no
trailing `Decode4`**. Char-mgmt (13/14/15) and login ops 3–22 are identical to
v61. Stage B/C MUST NOT blind-copy v61's LOGIN_PASSWORD/CHARLIST body length.

---

## (a) Opcode map — clientbound (writer)

Mapping = (v48 switch case, by body) → (v61 op via `gms_v61.yaml`). Each block
cites its dispatcher fn+addr. **Non-uniform** — no global offset.

### CWvsContext::OnPacket @ `0x70d215` (window 24–71; switch cases 25–70)

Full switch enumerated. **v48 is NOT a uniform shift of v61** — it starts Δ−1
(InventoryOperation 25 vs v61 26) and deepens because v48 **drops ops** and
**reorders** the mid-block. Mapped by fname:

| v48 op | fname / sub (v48) | v61 op (same fname) | note |
|---|---|---|---|
| 25 | OnInventoryOperation | 26 | Δ−1 |
| 26 | OnInventoryGrow | 27 | Δ−1 |
| 27 | OnStatChanged | 28 | Δ−1 |
| 28 | OnTemporaryStatSet (GIVE_BUFF) | 29 | Δ−1 |
| 29 | OnTemporaryStatReset (CANCEL_BUFF) | 30 | Δ−1 |
| — | *(v61 31 OnForcedStatSet, 32 OnForcedStatReset — **ABSENT in v48**)* | 31,32 | **2 ops dropped** |
| 30 | OnChangeSkillRecordResult (UPDATE_SKILLS) | 33 | Δ−3 |
| 31 | OnSkillUseResult | 34 | Δ−3 |
| 32 | OnGivePopularityResult (FAME_RESPONSE) | 35 | Δ−3 |
| 33 | **OnMessage (SHOW_STATUS_INFO)** | 36 | Δ−3 |
| — | *(v61 37 OnOpenFullClientDownloadLink — **ABSENT in v48**)* | 37 | **1 op dropped** |
| 34 | OnMemoResult | 38 | Δ−4 |
| 35 | OnMapTransferResult | 39 | Δ−4 |
| 36 | OnAntiMacroResult (WEDDING_PHOTO) | 40 | Δ−4 |
| 37 | OnClaimResult | 42 | Δ−5 |
| 38 | `sub_71F525` (CLAIM_AVAILABLE_TIME slot) | 43 | Δ−5 |
| 39 | `sub_71F54E` (CLAIM_STATUS_CHANGED slot) | 44 | Δ−5 |
| 40 | `sub_72032B` (SET_TAMING_MOB_INFO slot) | 45 | Δ−5 |
| 41 | OnQuestClear | 46 | Δ−5 |
| 42 | OnIncubatorResult | 66 | reordered (v48 groups incubator early) |
| 43 | `sub_71A135` (SKILL_LEARN_ITEM / entrusted-shop family) | 47/48 | Stage B body-map |
| 44 | OnSueCharacterResult | 52 | reordered |
| 46 | `sub_71CE62` | (Stage B) | body-map |
| 47 | `sub_71CE49` | (Stage B) | body-map |
| 49 | **OnCharacterInfo (CHAR_INFO)** | 58 | reordered |
| 50 | **OnPartyResult (PARTY_OPERATION)** | 59 | reordered; mode table §b |
| 51 | **OnAllianceResult (ALLIANCE_OPERATION)** | 63 | reordered |
| 53 | **OnGuildResult (GUILD_OPERATION)** | 62 | reordered |
| 54 | OnTownPortal (SPAWN_PORTAL) | 64 | reordered |
| 55 | **OnBroadcastMsg (SERVERMESSAGE)** | 65 | mode table §b |
| 56 | OnShopScannerResult | 67 | |
| 57 | `sub_71FF8E` | (Stage B) | body-map |
| 58 | `sub_72025D` | (Stage B) | body-map |
| 59 | `sub_720928` | (Stage B) | body-map |
| 60 | `sub_720A89` | (Stage B) | body-map |
| 61 | `sub_721129` | (Stage B) | body-map |
| 62 | `sub_713202` | (Stage B) | body-map |
| 63 | `sub_720293` | (Stage B) | body-map |
| 64 | OnMapleTVUseRes | (Stage B) | |
| 65 | OnAvatarMegaphoneRes | 83 | |
| 66 | OnSetAvatarMegaphone | 84 | |
| 67 | `sub_721465` | (Stage B) | body-map |
| 68 | `sub_721481` | (Stage B) | body-map |
| 69 | OnDestroyShopResult | 88 | |
| 70 | `sub_7215EA` | (Stage B) | body-map |

**Absent in v48 vs v61 (CWvsContext):** OnForcedStatSet (v61 31), OnForcedStatReset
(v61 32), OnOpenFullClientDownloadLink (v61 37), plus v61's whole upper tail
82–91 (MapleTV/AvatarMegaphone/CancelNameChange/DestroyShop/FakeGMNotice/
SuccessInUsegachaponBox/MacroSysDataInit) is **compressed into 64–70** in v48
(fewer ops overall). **Stage B maps the unnamed subs 43/46/47/57–63/67/68/70 by
body against `gms_v61.yaml`** (checked, not a uniform shift). The `Delta`
deepens 0→−1→−3→−4→−5 across the block, then reorders.

### CField::OnPacket @ `0x4c66f2` — base char-case region 'M'(77)–'b'(98), Δ **≈−20**
Base switch (op = ASCII), verified from the decompile:

| v48 op | v48 handler | packet | v61 op | Δ |
|---|---|---|---|---|
| `'M'`(77) | OnTransferFieldReqIgnored | BLOCKED_MAP | 97 | −20 |
| `'N'`(78) | OnTransferChannelReqIgnored | BLOCKED_SERVER | 98 | −20 |
| `'O'`(79) | OnFieldSpecificData | FORCED_MAP_EQUIP | 99 | −20 |
| `'P'`(80) | OnGroupMessage | MULTICHAT | 100 | −20 |
| `'Q'`(81) | OnWhisper | WHISPER | 101 | −20 |
| `'R'`(82) | OnCoupleMessage | SPOUSE_CHAT | 102 | −20 |
| `'S'`(83) | OnSummonItemInavailable | SUMMON_ITEM_INAVAILABLE | 103 | −20 |
| `'T'`(84) | `sub_4C7B59` (FIELD_EFFECT family) | FIELD_EFFECT | 104 | −20 |
| `'U'`(85) | `sub_4C930A` | (FIELD_OBSTACLE) | 105 | −20 |
| `'V'`(86) | `sub_4C95F2` | (BLOW_WEATHER) | 106 | −20 |
| `'W'`(87) | OnAdminResult | ADMIN_RESULT | 107 | −20 |
| `'X'`(88) | OnQuiz | OX_QUIZ | 108 | −20 |
| `'Y'`(89) | OnDesc | GMEVENT_INSTRUCTIONS | 109 | −20 |
| `'Z'`(90) | vtable+28 (OnClock) | CLOCK | 110 | −20 |
| `']'`(93) | `sub_4CBB78` | SET_QUEST_CLEAR | 113 | −20 |
| `'^'`(94) | `sub_4CBC9A` | (SET_QUEST_TIME sibling) | 114 | −20 |
| `'_'`(95) | OnSetQuestTime | SET_QUEST_TIME | 114 | — |
| `` '`' ``(96) | OnWarnMessage | ARIANT_RESULT | 115 | −19 |
| `'a'`(97) | OnSetObjectState | SET_OBJECT_STATE | 116 | −19 |
| `'b'`(98) | `sub_4C6AEF` | STOP_CLOCK | 117 | −19 |

Gaps at `'['`(91)/`'\'`(92) (mirror v61 base-switch subclass gaps, shifted). The
named social ops sit at **Δ−20** (WHISPER 81 vs v61 101). **SET_FIELD family** is
the **72–75** range → `sub_5C45ED` (reached from BOTH the login default arm and
CField's LABEL_29): v48 SET_FIELD/SET_ITC/SET_CASH_SHOP ≈ **72–75** vs v61 92–94
(**Δ≈−20**) — Stage B decomposes the 72–75 set-field demux by body.

### CField pool-routing ranges (read verbatim from `0x4c66f2`)
CField dispatches these opcode ranges (Stage B diffs each pool's internal leaf
switch against v61; boundaries verified, per-leaf order is Stage B):

| v48 range | v48 dispatcher | likely pool (v61 counterpart) |
|---|---|---|
| 72–75 | `sub_5C45ED` | CStage set-field family (v61 92–94) |
| 99–156 | `sub_6B2710` | **CUserPool** (v61 120–174; start Δ≈−21) |
| 157–175 | `sub_559340` | user-sibling / summon region |
| 176–186 | `sub_56D413` | **CMobPool** (v61 175–193) |
| 187–191 | `sub_4B3264` | (mob-sibling) |
| 192–195 | `sub_4AACCF` | **CNpcPool** (v61 194–201) |
| 196–200 | `sub_54329B` | CEmployee/Drop family |
| 201–204 | `sub_42182F` | (pool) |
| 205–208 | `sub_5E318D` | (pool) |
| 209–214 | `sub_5A5390` | CReactor/MessageBox family |
| 225–227 | `sub_5B0ACE` | (pool) |
| 228–231 | CStoreBankDlg::OnPacket | storebank dialog |
| 232 | `sub_5EC4F3` | (single) |
| 233–236 | CRPSGameDlg::OnPacket | RPS dialog |
| 237 | `sub_5ADB94` | dialog |
| 238 | `sub_61D8B8` | dialog |
| 239 | `sub_5459C4` | dialog |
| 247 | CTrunkDlg::OnPacket | trunk dialog |
| 263–266 | `sub_4E5F06` | (dispatcher-family) |
| 267–272 | `sub_527238` | (dispatcher-family) |

The map region opens at **Δ≈−21** (CUserPool base 99 vs v61 120) and pool
boundaries are re-packed (v48 has fewer ops per pool). **Stage B walks each
`sub_XXXX` pool dispatcher's internal switch by body** — do NOT assume v61 leaf
order or v61 pool boundaries.

### CCashShop::OnPacket @ `0x4534d6`
Present (named). Cash-shop is outside the core login→map flow; Stage B maps its
switch against v61's `CCashShop` block by body.

---

## (a) Opcode map — serverbound (handler) — from v48 `COutPacket(N)` send-sites

No CSV v48 serverbound column exists; opcodes read from the `COutPacket(N)`
constructor at each send-site. **The serverbound space is its own enumeration**
(distinct from the clientbound shift). Verified:

| op (serverbound) | v48 send-site (addr) | v48 opcode | v61 op | Δ vs v61 |
|---|---|---|---|---|
| LOGIN_PASSWORD | `CLogin::SendCheckPasswordPacket` `0x4ffeb2` | `COutPacket(1)` | 1 | **0** |
| CHANGE_MAP | `CField::SendTransferFieldRequest` `0x4c5733` | `COutPacket(30)` | 35 | **−5** |
| PARTY_OPERATION | `CField::SendJoinPartyMsg` `0x4c54dd` (accept: `Enc1(4)+EncStr`) | `COutPacket(94)` | 112 | **−18** |
| PARTY block/report | `CWvsContext::OnPartyResult` case-4 reply `0x729935` | `COutPacket(95)` | 113 | **−18** |

Body shapes verified at each send-site (match v61 except LOGIN_PASSWORD length):
- **LOGIN_PASSWORD** = `EncStr(id)+EncStr(pw)+EncBuffer(16)+Enc4(gameRoom)+Enc1(b)
  +Enc1(0)` — **2 fields shorter than v61** (no `Enc1(0)+Enc4(partnerCode)` tail).
- **CHANGE_MAP** = `Enc1(portalByte)+Enc4(targetMap)+EncStr(portal)+[Enc2(x)+Enc2(y)]
  +Enc1(0)+Enc1(wheel)+Enc1(premiumFlag)[+Enc4+Enc4]` — same shape as v61, op 30.
- **PARTY_OPERATION** accept-invite = `COutPacket(94) Enc1(4)+EncStr(name)`;
  block/report reply = `COutPacket(95) Enc1(19|20)+EncStr+EncStr`. v61 = 112/113.

> **The full serverbound opcode table is a Stage B deliverable** — derive each op
> from its v48 send-site anchored on the v61 serverbound FName. Stage A fixes the
> method + the anchors above. **CHANGE_CHANNEL and MOVE_PLAYER serverbound were
> NOT located by symbol** (the v48 IDB has no named `SendTransferChannelRequest`
> or move-flush symbol; `CField` Send-methods present: `SendJoinPartyMsg` 0x4c54dd,
> `SendKickPartyMsg` 0x4c55f3, `SendTransferFieldRequest` 0x4c5733,
> `SendCreateGuildAgreeMsg` 0x4c5a18, `SendInviteGuildMsg` 0x4c5a89,
> `SendKickGuildMsg` 0x4c5e06, `SendSetGuildMarkMsg` 0x4c635c). CHANGE_CHANNEL is
> adjacent to CHANGE_MAP=30 in v61 (35/36) so ≈31, and MOVE_PLAYER followed at
> CHANGE_MAP+3 in v61 → ≈33 — **but these are NOT verified; Stage B MUST read them
> from the actual send-sites, not seed the predicted values.** WHISPER/NPC_TALK
> serverbound likewise unresolved by symbol → Stage B.

---

## (b) Operation / mode (sub-op) tables — from v48 dispatcher switches

Read **from the v48 switch**; do not inherit v61's values.

### Status-message — `CWvsContext::OnMessage` @ `0x71b1b8` (v48 op 33 / SHOW_STATUS_INFO)
Leading `Decode1` mode; switch arms **0,1,2,3,4,5,6,7,8,9** — **10 arms (0–9),
default = drop.** **SAME as v61 (0–9).** Contiguous, no spurious
`INCREASE_SKILL_POINT` (the off-by-one crash trap does not apply). Stage C:
v48 SHOW_STATUS_INFO operations table = 0–9.

### World/broadcast message — `CWvsContext::OnBroadcastMsg` @ `0x71c356` (v48 op 55 / SERVERMESSAGE)
Leading `Decode1` type; switch cases **0,1,2,3,4,5,6,7,8,9** (8 & 9 share the
notice path) — **10 arms (0–9), default = drop.** **NARROWER than v61 (0–10, 11
arms):** v48 lacks type 0xA, and **v48 has NO item-speaker/item-megaphone arm**
(no `GW_ItemSlotBase::Decode` anywhere in the switch — unlike v61 type 8). Notable
bodies: type 4 reads an extra leading `Decode1` whisper-flag before the string;
type 3 = `DecodeStr + Decode1 + Decode1`; type 7 = util-dlg (`Decode4`); types 8/9
= notice variants (share). Stage C: v48 broadcast operations table = 0–9.

### Party — `CWvsContext::OnPartyResult` @ `0x729935` (v48 op 50 / PARTY_OPERATION)
Leading `Decode1` mode. **v48 arms present (decimal):** `4, 6, 7, 8, 9, 11, 12,
14, 15, 16, 17, 19, 20, 21, 24, 25, 26, 27, 28, 29` (+ default; `26` shares the
load/refresh arm with `6`).

**⚠️ The v48 party mode table DIFFERS from v61** (unlike v61, which matched v72).
The arm VALUES are LOWER and re-packed. Body-anchored:
- `4` invite → `Decode4(partyId)+DecodeStr(inviter)`, builds YESNO + serverbound
  `COutPacket(95)` block-report (`Enc1(19|20)`).
- `6`/`26` load/refresh → `Decode4 + PARTYDATA::Decode`.
- `7` join(create) → `Decode4+Decode4+Decode4+Decode2+Decode2` + party-slot init.
- `11` leave/expel → `Decode4+Decode4+Decode1(+DecodeStr on notice)+PARTYDATA::Decode`.
- `14` join member → `Decode4+DecodeStr+PARTYDATA::Decode`.
- `27` HP/max update (FindUser) → `Decode4(cid)+Decode4+Decode4`.
- **`29` = HP/coord slot-update** → `Decode1 slot (if ≥6 → `CDisconnectException`)
  + Decode4 + Decode4 + Decode2 + Decode2` — **v61 had this at mode `0x24`(36)**.
- `8,9,12,15,16,17,24,25` = simple chatlog-notice arms; `19,20,21` = `DecodeStr +
  chatlog`; `28` = `Decode1 ? DecodeStr+chatlog : notice`.

**v61 party arms were `4,7,8,9,0xA,0xC,0xD,0xF,0x10,0x11,0x12,0x13,0x15,0x16,0x17,
0x1A,0x1B,0x1C,0x1D,0x1F,0x20,0x21,0x22,0x23,0x24`.** The v48 set is a DIFFERENT,
lower-valued, smaller enumeration. **Stage C/E MUST extract the v48 party
operations table from this switch — do NOT carry v61's party table.**

### Other dispatcher families with mode tables (located, not byte-extracted)
`CWvsContext::OnGuildResult` @ `0x725559` (op 53) · `CWvsContext::OnAllianceResult`
@ `0x72a591` (op 51) · buddy/friend routing (unnamed subs in the 57–63 block) ·
`CWvsContext::OnInventoryOperation` @ `0x71a4f6` (op 25) · guild-BBS (Stage B
locate) · shop/trunk/storebank dialog demuxers (`CStoreBankDlg::OnPacket`
`0x5b7a38`, `CTrunkDlg::OnPacket` `0x58332c`, `CRPSGameDlg::OnPacket` `0x5d5544`,
CField 237–239/263–272 sub-demuxers). Stage E owns per-mode extraction.

---

## (c) Structure / encoding deltas vs v61 — login → channel → map → movement/chat + tier-1

Swept (not sampled) for the connect-critical flow:

| flow stage | v48 packet | structure vs v61 | evidence |
|---|---|---|---|
| login | LOGIN_PASSWORD (sb 1) | **SHORTER** — no trailing `Enc1(0)+Enc4(partnerCode)` | `SendCheckPasswordPacket` 0x4ffeb2 |
| login | auth result (cb **op 1**, not 0) | **op moved 0→1**; GuestID(v61 op1) absent | `sub_5007C4`, `sub_500931` |
| login | CHARLIST (cb 11) | **no trailing `Decode4`** after the slot loop (v61 had one); slot shape same | `sub_5013ED` 0x5013ed |
| login | VIEW_ALL_CHAR (cb 8) | same sub-mode/AvatarLook shape; op 8 both | `sub_50232D` 0x50232d |
| login | char-mgmt (cb 13/14/15) | **same by body** (CHAR_NAME/ADD/DELETE); unnamed subs, no rotation | 0x5016db / 0x501973 / 0x5017b6 |
| channel | CHANGE_CHANNEL (sb) | body expected `Enc1(ch)+Enc4`; **opcode UNVERIFIED** (send-site not symbol-named) | Stage B |
| map | CHANGE_MAP (sb 30) | **same body** as v61; opcode 30 (v61 35) | `0x4c5733` |
| map | SET_FIELD (cb ~72) | 72–75 → `sub_5C45ED` demux; body not byte-diffed in Stage A | `0x4c66f2` |
| movement | MOVE_PLAYER (cb) | remote path via CUserPool range 99–156; leaf op Stage B | `sub_6B2710` |
| movement | MOVE_PLAYER (sb) | **opcode UNVERIFIED** (move-flush send-site not symbol-named) | Stage B |
| chat | MULTICHAT 80 / WHISPER 81 / SPOUSE_CHAT 82 (cb) | OnGroupMessage/OnWhisper/OnCoupleMessage at base 'P'/'Q'/'R'; Δ−20 vs v61 | `0x4c66f2` |
| chat | CHATTEXT (cb, CUserPool common region) | two-arm chat same family; leaf op Stage B | `sub_6B2710` |

**Wire-structure deltas found in the swept core flow (beyond opcode renumbering):**
(1) **LOGIN_PASSWORD is 2 fields shorter**; (2) **CHARLIST omits the trailing
`Decode4`**; (3) **auth-result opcode moved 0→1** with GuestID dropped. Otherwise
bodies match v61 at every send/decode site read. Mode-table deltas:
**status-message 0–9 (= v61), broadcast 0–9 (narrower, no item-megaphone), party
DIFFERS (lower-valued arms, slot-update at 29 not 0x24).**

**tier-1 note (`docs/packets/evidence/tiers.yaml`):** `summon/clientbound/
SummonSpawn` version-gated trailing avatar-look byte is `GMS≥95` only. v48 < v61
< v95 → SummonSpawn has **no trailing avatar byte** in v48 (same as v61). Other
tier-1 prefix families are opcode-mapped above; their opaque bodies decode via the
same `*::Decode` helpers as v61 — Stage E byte-fixtures confirm per packet.

## (d) "Same as anchor (v61)" — entries with explicit switch/body evidence

Verified equal (not defaulted):
- **Login ops 3,4,5,6,7,8,9,10,11,12,13,14,15,22** — v48 `sub_5007C4` cases match
  v61 op numbers case-for-case (op 8 = ViewAll, op 11 = CHARLIST, op 13/14/15 =
  char-mgmt, all body-confirmed).
- **Char-mgmt body mapping 13/14/15** — CHAR_NAME/ADD/DELETE identical to v61 by
  decompiled read-order.
- **CHANGE_MAP body** — same `Enc1+Enc4+EncStr+[Enc2/Enc2]+Enc1×3[+Enc4×2]`; op 30.
- **Status-message mode table (op 33)** — arms 0–9, identical to v61.
- **usesPin = false** — PIN slots at op 6/7 as v61, no SPW cases (max case 22).
- **No trailing SummonSpawn avatar byte** — same absence as v61 (< v95 gate).

## Differences from anchor (v61) — summary for Stage B/C/E (in-scope)

1. **CWvsContext window `[0x18,0x47]`** (24–71) vs v61 `[0x1A,0x5B]`; non-uniform.
2. **Login auth-result opcode = 1** (v61 0); op 0 unused; GuestID (v61 op 1) absent.
3. **LOGIN_PASSWORD body 2 fields shorter** (no `Enc1(0)+Enc4(partnerCode)`).
4. **CHARLIST has no trailing `Decode4`.**
5. **CWvsContext drops OnForcedStatSet/OnForcedStatReset (v61 31/32) and
   OnOpenFullClientDownloadLink (v61 37)**; upper tail compressed into 64–70.
6. **CField base region Δ≈−20** (WHISPER 81 vs 101; BLOCKED_MAP 77 vs 97;
   SET_FIELD family 72–75 vs 92–94). Map pools re-packed (CUserPool base 99 vs 120).
7. **Serverbound is its own enumeration:** LOGIN_PASSWORD 1 (Δ0), CHANGE_MAP 30
   (v61 35, Δ−5), PARTY 94 / block 95 (v61 112/113, Δ−18). CHANGE_CHANNEL &
   MOVE_PLAYER & WHISPER-sb & NPC_TALK-sb send-sites **not symbol-named →
   Stage B reads from the send-site (no blind seed).**
8. **Broadcast mode table SHRINKS to 0–9** (v61 0–10) and has **no item-megaphone**.
9. **Party mode table DIFFERS** — lower-valued re-packed arms
   `4,6,7,8,9,11,12,14,15,16,17,19,20,21,24,25,26,27,28,29`; slot-update at **29**
   (v61 0x24). Stage C/E must extract from the v48 switch, not carry v61's.
10. **Top-level 0x14/0x15 subs differ** (`sub_733B6B`/`sub_465378` vs v61's
    OnPacket_CSecurityClient/`sub_4747E2`) — handshake-family, Stage B FName.

## (OQ-7) Dispatcher-family list for Stage E

| family | v48 dispatcher (addr) | v48 cb op | mode table vs v61 |
|---|---|---|---|
| status-message | `CWvsContext::OnMessage` 0x71b1b8 | 33 | **SAME** — arms 0–9 |
| worldmessage/broadcast | `CWvsContext::OnBroadcastMsg` 0x71c356 | 55 | **DIFFERS** — 0–9 only, no item-megaphone |
| party | `CWvsContext::OnPartyResult` 0x729935 | 50 | **DIFFERS** — re-packed arms, slot-update 29 |
| guild | `CWvsContext::OnGuildResult` 0x725559 | 53 | not extracted — Stage E |
| alliance | `CWvsContext::OnAllianceResult` 0x72a591 | 51 | not extracted — Stage E |
| inventory-op | `CWvsContext::OnInventoryOperation` 0x71a4f6 | 25 | not extracted — Stage E |
| buddy/friend | unnamed subs in 57–63 block | 57–63 | body-map — Stage B/E |
| storebank/trunk/RPS | CStoreBankDlg 0x5b7a38 / CTrunkDlg 0x58332c / CRPSGameDlg 0x5d5544 | 228–247 | body-mode demuxers; Stage E |
| set-field family | `sub_5C45ED` (72–75) | 72–75 | Stage B (v61 92–94 counterpart) |
| field-effect family | `sub_4C7B59` (base 'T'=84) | 84 | Stage E (v61 FIELD_EFFECT) |
| cashshop | `CCashShop::OnPacket` 0x4534d6 | (Stage B) | body-map vs v61 |

**Campaign size:** ~9 dispatcher families need per-version operation-table
extraction. status-message (0–9, = v61), broadcast (0–9), party (re-packed) are
captured here; the rest are located (addresses above) but not byte-extracted.

---

## Escalations / open items handed to Stage B/E (none blocking Stage A)

1. **CSV v48 column is placeholder-only** — seed v48 from this doc/IDB, not the CSV.
2. **Login auth-result opcode = 1, op 0 unused, GuestID absent** — confirm in
   Stage B whether any pre-stage path handles op 0 (none found in the login switch).
3. **LOGIN_PASSWORD is 2 fields shorter** and **CHARLIST drops the trailing
   Decode4** — Stage B/C must not clone v61 body lengths.
4. **CHANGE_CHANNEL / MOVE_PLAYER / WHISPER-sb / NPC_TALK-sb serverbound opcodes
   NOT resolved by symbol** — Stage B reads from the actual send-sites (adjacency
   predicts CHANGE_CHANNEL≈31, MOVE≈33 but these are UNVERIFIED — do not seed blind).
5. **Party mode table DIFFERS from v61** — extract the v48 arm set from
   `CWvsContext::OnPartyResult` (slot-update at mode 29, not 0x24).
6. **Broadcast mode table 0–9, no item-megaphone** — 10-arm operations table.
7. **CWvsContext unnamed subs 43/46/47/57–63/67/68/70** — body-map against
   `gms_v61.yaml` (non-uniform; reordered mid-block).
8. **CField pool boundaries re-packed** — Stage B walks each `sub_XXXX` pool
   dispatcher's internal switch by body (CUserPool `sub_6B2710`, mob `sub_56D413`,
   npc `sub_4AACCF`, etc.); do NOT assume v61 leaf order or pool boundaries.
9. **Top-level 0x14/0x15 handshake subs** (`sub_733B6B`/`sub_465378`) — FName in
   Stage B if a handshake fixture needs them.
10. **SET_FIELD family at 72–75** (`sub_5C45ED`) — decompose the set-field demux
    (SET_FIELD/SET_ITC/SET_CASH_SHOP) by body in Stage B.

Every unnamed sub above was identified from its decompiled read-order / send-opcode
or left explicitly for Stage B with its address. No unresolved fname required
fabrication; no opcode was seeded from a prediction.
