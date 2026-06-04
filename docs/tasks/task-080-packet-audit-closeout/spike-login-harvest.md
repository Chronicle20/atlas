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

### Still pending for v83
- v83: the 5 addressed FNames + LoginAuth presence (expect absent/middleware) + SendCheckPasswordPacket/SendSelectCharPacket v83 layouts (PartnerCode present? PIC ops?) + the B3.6 NOTE REFRESH=7-vs-wire-8 question (decompile the memo/OnMemoResult handler).
