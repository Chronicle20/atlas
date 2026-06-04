# Login / chat IDA harvest accumulator (task-080 B1.2 + B6.1)

Working note. Banks per-IDB IDA findings as each GMS IDB is loaded, so we never re-load a version.
JMS185 + GMS v87 done; GMS v95 + v83 pending. The final B6.1 verdict goes in `spike-login.md`.

## B1.2 — chat Multi / CUIStatusBar::SendGroupMessage (leading updateTime?)

| version | fn @addr | leading updateTime? | body |
|---|---|---|---|
| JMS185 | @0x98acbf (op 0x79) | **NO** | chatType(1), count(1), recipients(count×4), text(str) |
| GMS v87 | @0x953d6b (op 0x7D) | **NO** | chatType(1), count(1), recipients(count×4), text(str) |
| GMS v95 | _pending_ | _?_ | _?_ |
| GMS v83 | _pending_ (almost certainly NO) | _?_ | _?_ |

**Premise status:** plan's "GMS>83 carries leading updateTime" is FALSE for v87 (and JMS). Current Atlas `chat/serverbound/multi.go` has NO updateTime. → If v95 also lacks it, **B1.2 is a NO-OP** (existing code correct, record verdict like B1.3). If v95 HAS it, gate is `Region()=="GMS" && MajorVersion()>=95`. **RESOLVE WITH v95.**

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

### Pending for v95 / v83
- v95: SendGroupMessage (B1.2 decisive), the 5 addressed FNames, LoginAuth presence, SendCheckPasswordPacket/SendSelectCharPacket layouts, ServerStatus/PicResult clientbound.
- v83: same set + the B3.6 NOTE REFRESH=7-vs-wire-8 question (decompile the memo/OnMemoResult handler) + LoginAuth presence.
