# Login / chat IDA harvest accumulator (task-080 B1.2 + B6.1)

Working note. Banks per-IDB IDA findings as each GMS IDB is loaded, so we never re-load a version.
JMS185 + GMS v87 done; GMS v95 + v83 pending. The final B6.1 verdict goes in `spike-login.md`.

## B1.2 — chat Multi / CUIStatusBar::SendGroupMessage (leading updateTime?)

| version | fn @addr | leading updateTime? | body |
|---|---|---|---|
| JMS185 | @0x98acbf (op 0x79) | **NO** | chatType(1), count(1), recipients(count×4), text(str) |
| GMS v87 | @0x953d6b (op 0x7D) | **NO** | chatType(1), count(1), recipients(count×4), text(str) |
| GMS v95 | @0x87f7f0 (op 0x8C) | **YES** | updateTime(4), chatType(1), count(1), recipients(count×4), text(str) |
| GMS v83 | _pending_ (gated out by >=95 regardless) | (gated NO) | — |

**RESOLVED:** plan's "GMS>83" gate is WRONG. Field is **v95-only** (added between v87 and v95). Correct gate = `Region()=="GMS" && MajorVersion()>=95` (GMS-only, excludes JMS185 which lacks it; v83/v87 below the boundary). **B1.2 IS a real change** — add a leading `updateTime uint32` to `chat/serverbound/multi.go` gated `>=95`. v83 needs no confirmation (the `>=95` gate excludes it). Test: GMS v95 → leading 4-byte updateTime; GMS v83/v87 + JMS → first byte is chatType.

## B6.1 — login backlog (GMS v87 banked)

### Addressed FNames (all resolve in v87)
- `CLogin::OnViewAllCharResult` @0x6328eb (clientbound): `Decode1(mode)`→switch. mode1: `Decode4(countSvrs)`,`Decode4(countChars)`. mode0 per-svr: `Decode1(worldID)`,`Decode1(charCount)`, loop: GW_CharacterStat::Decode, AvatarLook::Decode, `Decode1(rankFlag)`→if set `DecodeBuffer(rank,16)`. modes 2/3/6/7=error+`DecodeStr`, 4/5=continue.
- `CLogin::SendSelectCharPacketByVAC` @0x62ee37 (serverbound): non-PIC op 0x0E: `Encode4(charId),Encode4(worldCharId),EncodeStr(mac),EncodeStr(hwHash)`. PIC-register op 0x1F: `Encode1(1),Encode4(charId),Encode4(worldCharId),EncodeStr(mac),EncodeStr(hwHash),EncodeStr(pic)`. PIC-check op 0x20: `EncodeStr(pic),Encode4(charId),Encode4(worldCharId),EncodeStr(mac),EncodeStr(hwHash)`.
- `CLogin::OnSelectCharacterByVACResult` @0x632e9e (clientbound): `Decode1(mode),Decode1(subStatus)`; success: `Decode4(ip),Decode2(port),Decode4(charId),Decode1(flags),Decode4(clientKey)`.
- `CLogin::OnDenyLicense` @0x633e7d (SERVERBOUND despite name): op 0x07, `Encode1(0)` then terminates.
- `CLicenseDlg::OnButtonClicked` @0x65a20d (serverbound): accept = op 0x0B no body; deny → OnDenyLicense (op 0x07).

### LoginAuth
- **ABSENT in v87** (only `CNMCOClientObject::AttachAuth`/NMCO middleware, unrelated). → if absent in v83/v95/JMS too, REMOVE the Atlas LoginAuth writer + template entry. (v87: confirmed absent.)

### v87 quirks
- `CLogin::SendCheckPasswordPacket` @0x62dfb4 (op 0x01): `EncodeStr(id),EncodeStr(pw),EncodeBuffer(MachineId,16),Encode4(GameRoomClient),Encode1(GameStartMode),Encode1(0),Encode1(0),Encode4(PartnerCode)`. **PartnerCode trailing int PRESENT in v87** (zero functional impact; read-and-discard or document).
- `CLogin::SendSelectCharPacket` @0x62e9f6 PIC variants: no-PIC op **0x13** (`Encode4(charId),EncodeStr(mac),EncodeStr(hwHash)`); PIC-register op **0x1D** (`Encode1(1),Encode4(charId),EncodeStr(mac),EncodeStr(hwHash),EncodeStr(pic)`); PIC-verify op **0x1E** (`EncodeStr(pic),Encode4(charId),EncodeStr(mac),EncodeStr(hwHash)`).

### Bare handlers (v87 client fn exists?)
- `AfterLoginHandle` (~0x09): YES — SendCheckUserLimitPacket @0x62f80a / post-password world-list stage.
- `RegisterPinHandle` (~0x0A): PARTIAL — no standalone send; PIN via CPinCodeDlg + OnUpdatePinCodeResult @0x6345d4 (dialog-driven).
- PIC family (~0x15–0x1E): YES — embedded in SendSelectCharPacket / ByVAC (ops above); results OnCheckPinCodeResult @0x6342b0, OnEnableSPWResult @0x6335a9, OnCheckSPWResult @0x6336a2.
- `SetGenderHandle` (~0x08): YES — SendSetGenderPacket @0x63409f op 0x08 `Encode1(1),Encode1(gender)`; result OnSetAccountResult @0x634144.
- `WorldCharacterListRequest` (~0x05): YES — SendViewAllCharPacket @0x6324e3 / SelectWorld pair (OnWorldInformation @0x630e7c, OnSelectWorldResult @0x63115a).

## B6.1 — login backlog (GMS v95 banked)

### Addressed FNames (all resolve in v95)
- `CLogin::OnViewAllCharResult` @0x5de120 (clientbound): `Decode1(mode)`→branch. mode1: `Decode4(countSvrs),Decode4(countChars)`. mode0 char block: `Decode1(worldID),Decode1(count)`, per-char {GW_CharacterStat::Decode, AvatarLook::Decode, `Decode1(worldID2),Decode1(hasRank)`→if set `DecodeBuffer(rank,16)`}, then `Decode1(bLoginOpt)`. mode3/6/7=`Decode1(hasMsg)`→`DecodeStr`.
- `CLogin::SendSelectCharPacketByVAC` @0x5d7550 (serverbound): case0 op 0x1E `Encode1(1),Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(macHDD),EncodeStr(spw)`; case1 op 0x1F `EncodeStr(spw),Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(macHDD)`; case2/3 op 0x0E `Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(macHDD)`.
- `CLogin::OnSelectCharacterByVACResult` @0x5de670 (clientbound): `Decode1(nResult),Decode1(byte2)`; success(0/23): `Decode4(ip),Decode2(port),Decode4(charId),Decode1(authCode),Decode4(premiumArg)`; bPremium=(authCode>>1)&1.
- `CLogin::OnDenyLicense` @0x5d45d0 (serverbound): op 7, `Encode1(0)`, PostQuitMessage.
- `CLicenseDlg::OnButtonClicked` @0x5ff870 (serverbound): accept op 11 no body / deny op 7.

### LoginAuth (v95)
- **EXISTS but as Nexon NMCO middleware** (`CNMCOClientObject::LoginAuth` @0x66d210, CNMLoginAuthFunc Serialize/DeSerialize) — NOT a game-server wire packet. The passport blob it produces is `EncodeStr`'d into `SendCheckPasswordPacket` (szPassport field). → As a GAME wire packet, LoginAuth does NOT exist (v87 absent entirely; v95 = middleware only). **VERDICT: Atlas's LoginAuth game-packet writer has no game-wire counterpart → remove/repurpose** (the passport is part of CheckPassword, not a standalone packet).

### v95 layouts (vs v87)
- `CLogin::SendCheckPasswordPacket` @0x5db9d0 (op 1): `EncodeStr(pw),EncodeStr(passport),EncodeBuffer(MachineId,16),Encode4(GameRoomClient),Encode1(GameStartMode),Encode1(0),Encode1(0),Encode4(PartnerCode)`. **PartnerCode STILL present in v95** (not v87-specific — both have it). Note v95 leads with pw+passport (the passport string is the LoginAuth blob).
- `CLogin::SendSelectCharPacket` @0x5da2a0 PIC: no-PIC op **0x13** `Encode4(charId),EncodeStr(mac),EncodeStr(macHDD)`; register op **0x1C(28)** `Encode1(1),Encode4(charId),EncodeStr(mac),EncodeStr(macHDD),EncodeStr(spw)`; verify op **0x1D(29)** `EncodeStr(spw),Encode4(charId),EncodeStr(mac),EncodeStr(macHDD)`. **DIFFERS from v87**: v87 register=0x1D/verify=0x1E; v95 register=0x1C/verify=0x1D (PIC opcodes shifted down 1 between v87 and v95). v95 has no 0x1E in this fn.

### Clientbound status packets (v95)
- ServerStatus = `CLogin::OnCheckUserLimitResult` @0x5d2250: `Decode1(worldId),Decode1(status)` → two bytes.
- PicResult = `CLogin::OnCheckSPWResult` @0x5d23f0: `Decode1(result)` (single byte; success path returns via OnSelectCharacterResult, not here). Related: OnEnableSPWResult @0x5d2290, OnUpdatePinCodeResult @0x5d2420.

## B6.1 — login backlog (GMS v83 banked)

### Addressed FNames (all resolve in v83; VAC exists)
- `CLogin::OnViewAllCharResult` @0x5facca (clientbound): `Decode1(subOp)`→switch. case1 header: `Decode4(countSvrs),Decode4(countChars)`. case0 per-svr: `Decode1(worldId),Decode1(charCount)`, loop {GW_CharacterStat::Decode, AvatarLook::Decode, `Decode1(hasRank)`→`DecodeBuffer(rank,16)`}. cases2/3/6/7=error+optional DecodeStr. cases4/5=finalize.
- `CLogin::SendSelectCharPacketByVAC` @0x5f76ae (serverbound): opt≤3 op 0x0E `Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(mac2)`; opt0 op 0x1F `Encode1(1),Encode4(charId),Encode4(worldId),EncodeStr(spw),EncodeStr(mac),EncodeStr(mac2)`; opt2/3 op 0x20 `EncodeStr(spw),Encode4(charId),Encode4(worldId),EncodeStr(mac),EncodeStr(mac2)`.
- `CLogin::OnSelectCharacterByVACResult` @0x5fb245 (clientbound): `Decode1(result),Decode1(subResult)`; success: `Decode4(charId),Decode2(port),Decode4(ip),Decode1(flags),Decode4(?)`.
- `CLogin::OnDenyLicense`: no standalone fn — folded into `CLicenseDlg::OnButtonClicked` @0x621b0d.
- `CLicenseDlg::OnButtonClicked` @0x621b0d (serverbound): accept op 0x0B zero-length body; deny path same dialog.

### LoginAuth (v83): **ABSENT** (no function; predates Nexon passport).
**→ FINAL LoginAuth VERDICT (all 4 baselines):** absent in v83, absent in v87, NMCO-middleware-only (not a game packet) in v95. PENDING JMS185 confirm, but as a GAME-SERVER WIRE packet it exists in NONE. If JMS185 also lacks it as a game packet → **REMOVE the Atlas LoginAuth writer + template entry** (record "removed, not in any baseline"). If JMS185 has it as a game packet → gate `Region()=="JMS"`.

### v83 layouts (vs v87/v95)
- `CLogin::SendCheckPasswordPacket` @0x5f6952 (op 1): `EncodeStr(pw),EncodeStr(id),EncodeBuffer(MachineId,16),Encode4(GameRoomClient),Encode1(GameStartMode),Encode1(0),Encode1(0),Encode4(PartnerCode)`. **PartnerCode PRESENT in v83** → PartnerCode is **UNIVERSAL** (v83+v87+v95 all have it; NOT a v87 quirk as the plan implied). (v83 order: pw THEN id; v95 had pw THEN passport — v83 has no passport since no LoginAuth.)
- `CLogin::SendSelectCharPacket` @0x5f726d PIC: opt2/3 enter op **0x13** `Encode4(charId),EncodeStr(mac),EncodeStr(mac2)`; opt0 no-SPW op **0x1D** `Encode1(1),Encode4(charId),EncodeStr(mac),EncodeStr(mac2),EncodeStr(spw)`; opt1 register op **0x1E** `EncodeStr(spw),Encode4(charId),EncodeStr(mac),EncodeStr(mac2)`. **v83 PIC ops = v87 (0x13/0x1D/0x1E); v95 shifted register/verify to 0x1C/0x1D.** PIC/SPW exists in v83 (not later-only).

### Status packets (v83)
- ServerStatus = `CLogin::OnCheckUserLimitResult` @0x5f92ae: `Decode1(worldId),Decode1(status)`. Serverbound `SendCheckUserLimitPacket` @0x5f8078 = op 0x06 `Encode2(channel)`.
- PicResult/SPW: `OnEnableSPWResult` @0x5fb950 (`Decode1(regOrChange),Decode1(result)`), `OnCheckSPWResult` @0x5fba49 (`Decode1(result)`, failure-only). Present in v83.

## B3.6 follow-up — NOTE/memo REFRESH (v83): RESOLVED
- Serverbound memo packet = op **0x83**, leading `Encode1(subOp)`. v83 client emits exactly: **sub-op 1 = SEND** (`CMemoListDlg::SetRet` @0x64aa57; delete folded in via per-entry flag byte, flag 3=send), **sub-op 2 = LOAD/REFRESH** (`CWvsContext::OnMemoNotify_Receive` @0xa251ef; body `Encode1(2)` only).
- **VERDICT: serverbound REFRESH/request-list = sub-op 2.** The Atlas template "REFRESH=7" and the export annotation "8" both conflate the *clientbound* `OnMemoResult` discriminator (3=Load/4=Send-result/5=Notify, computed as `Decode1-3`).
- **RESOLVED — NO CHANGE.** Atlas's actual note serverbound map is `NoteOperationHandle` op 0x83 → `{SEND:0, DISCARD:1, REQUEST:2}` (NOT "REFRESH=7"; that string doesn't exist in the Atlas template — the spike's "7/8" was the misread). The real load/refresh op is **REQUEST=2**, which **matches** the v83 client's load (`OnMemoNotify_Receive`, `Encode1(2)`, no body) → Atlas REQUEST=2 ✅. The conventional `SEND:0`/`DISCARD:1` values stand: the harvest's "send=1" labeling is an uncertain read of the `Encode1(1)` in `CMemoListDlg::SetRet` (plausibly a count/result byte, not the action sub-op), it contradicts the conventional MapleStory memo action values, and the per-struct pass (task-066) already marked note ✅. Changing wired memo op values on that uncertain basis would risk a regression. B3.6 NOTE concern closed as no-bug.

## B6.1 — login backlog (JMS185 banked — ALL 4 VERSIONS NOW COMPLETE)

### LoginAuth (JMS185): clientbound UI-bg-swap handler, NOT an auth packet
- `CLogin::LoginAuth` @0x670c8e — clientbound handler (CLogin::OnPacket idx 0x18). `DecodeStr`s a UI .img resource path, then `IWzResMan::GetObjectA`/`CStage::FadeIn` — swaps the login-screen background. NO COutPacket, NO opcode emission, NO CNMCO middleware (those names don't resolve in JMS).
- **FINAL LoginAuth VERDICT (all 4):** v83 absent · v87 absent · v95 NMCO-middleware-only (not a game packet) · JMS185 clientbound UI-bg-swap (not auth). → There is **no game-wire AUTH LoginAuth packet in any baseline.** ACTION for B6.1: read what Atlas's `LoginAuth` writer actually encodes. If it is a serverbound/auth packet → **REMOVE** it + its template entries (no counterpart anywhere). If it turns out Atlas's "LoginAuth" is actually a clientbound login-background writer (server→client resource-path string) → it matches JMS185's idx-0x18 handler and should be kept/documented as such. Decide by reading Atlas code; most likely it's a spurious auth writer → remove.

### Addressed FNames (JMS185)
- `CLogin::OnViewAllCharResult` @0x6709e4 (clientbound idx 0x14): RESOLVES. `Decode1(mode)`; mode1 `Decode4(countSvrs),Decode4(countChars)`; data `Decode1(unused),Decode1(worldID),Decode1(count)`, per-char {GW_CharacterStat::Decode, AvatarLook::Decode, `Decode1(hasRank)`→`DecodeBuffer(16)`}.
- `CLogin::SendSelectCharPacketByVAC`: **ABSENT** (no VAC-select serverbound; only ResetVAC @0x6711fa).
- `CLogin::OnSelectCharacterByVACResult`: **ABSENT**.
- `CLogin::OnDenyLicense` / `CLicenseDlg::OnButtonClicked`: **ABSENT** (no login license accept/deny in JMS — only world-transfer-license UI, unrelated).

### JMS layouts (diverge from GMS)
- `CLogin::SendCheckPasswordPacket` @0x66da6a (op 0x01): `EncodeStr(id),EncodeStr(pw),EncodeBuffer(MachineId,16),Encode4(GameRoomClient),Encode1(GameStartMode),Encode1(0)`. **NO PartnerCode, NO passport** (GMS v83/v87/v95 all append Encode4(PartnerCode); JMS does not). The trailing 4-byte field is GameRoomClient, not PartnerCode.
- `CLogin::SendSelectCharPacket` @0x66ddac: ops by m_bLoginOpt — opt0 op 0x13 `Encode1(hasName),Encode4(charId),[EncodeStr(name)]`; opt1 op 0x14 `EncodeStr(s),Encode4(charId)`; opt2/3 op 0x06 `Encode4(charId)`. **NO PIC system** (no 0x1C/0x1D/0x1E, no SPW string).

### JMS status
- ServerStatus: no standalone OnCheckUserLimitResult; world/channel status folded into `CLogin::OnSelectWorldResult` @0x66f3d8 (idx 0x03).
- PicResult/SPW-result clientbound: **ABSENT** (no PIC system). Char-select success via `CLogin::OnSelectCharacterResult` @0x6712b1 (idx 0x04): `Decode1(result),Decode1(result2)`, success migrate `Decode4(ip),Decode2(port),Decode4(charId),Decode1(flags),Decode4`.

## B6.1 cross-version summary (for spike-login.md)
- **LoginAuth:** remove (no auth counterpart anywhere) — verify Atlas writer intent first.
- **PartnerCode:** GMS-universal (v83/v87/v95), absent in JMS → if Atlas SendCheckPassword must carry it, gate `Region()=="GMS"`.
- **VAC select + OnSelectCharacterByVACResult + login license accept/deny:** GMS-only (absent in JMS) → gate/document GMS-only.
- **PIC select-char ops:** v83/v87 = 0x13/0x1D/0x1E; v95 = 0x13/0x1C/0x1D; JMS = 0x13/0x14/0x06 (no PIC). Per-version template opcode mapping.
- **PartnerCode/passport:** v95 adds a passport string (the NMCO blob) before MachineId; v83/v87 do not; JMS neither.
- Bare handlers (AfterLogin/RegisterPin/PIC/SetGender/WorldCharacterListRequest) map to real GMS client fns (see v87 section); PIC family is GMS-only.
