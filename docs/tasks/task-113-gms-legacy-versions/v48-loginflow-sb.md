# GMS v48 — LOGIN-FLOW serverbound parity (Task 4.B2f)

> Closes the v48 login/selection-flow client→server gap the Stage B gameplay
> harvest (58 ops) skipped and Stage C surfaced ("no login-side handlers except
> LoginHandle/SetGenderHandle"). Anchor = `gms_v61`. Every v48 opcode was
> BODY-VERIFIED against its v61 twin send-site (never blind-shifted). v79 closed
> the identical gap; the v48 handler set now matches the v61/v79 anchor exactly.
> IDB: `GMS_v48_1_DEVM.exe` (port 13337). COutPacket ctor @0x57b77e.

## Method
Harvested every COutPacket(N) send-site in the CLogin region (0x4fe000–0x504000)
and CClientSocket (OnConnect/OnAliveReq) via xrefs to the COutPacket ctor, then
matched each body to the corresponding v61 sender (decompiled in the v61 IDB,
port 13338) to assign the Atlas handler. v48 login serverbound is its OWN
enumeration (non-uniform shift): the world/select region is Δ-1 vs v61,
CHAR_SELECT/PLAYER_LOGGEDIN/CHECK/CREATE are Δ-4, PONG/START_ERROR Δ-1.

## Registered (17 new serverbound ops → 17 new handlers)

| Atlas handler | v48 op (hex) | v48 sender (addr) | v61 twin (op) | Δ | body (verified) |
|---|---|---|---|---|---|
| ServerListRequestHandle (SERVERLIST_REREQUEST) | 3 / 0x03 | sub_4FE15D @0x4fe7ff | sub_56261B (4) | −1 | bare COutPacket(3) on world-select-screen init (twin of v61 bare op4) |
| CharacterListWorldHandle (CHARLIST_REQUEST) | 4 / 0x04 | sub_4FFFC4 @0x5000e6 | SendLoginPacket sub_564DC9 (5) | −1 | Enc1(world)+Enc1(channel) |
| ServerStatusHandle (SERVERSTATUS_REQUEST) | 5 / 0x05 | sub_50073B @0x50076e | SendCheckUserLimitPacket sub_5655DF (6) | −1 | Enc2(worldId) |
| AcceptTosHandle (ACCEPT_TOS) | 6 / 0x06 | sub_5034B2 @0x5034dd | sub_56842A (7) | −1 | Enc1(1) accept / (decline arm sub_503517 Enc1(0)) |
| AfterLoginHandle (AFTER_LOGIN) | 8 / 0x08 | sub_503956 @0x503bb8 | OnCheckPinCodeResult sub_5688CE case2 (9) | −1 | Enc1(pin)+Enc1(0)+Enc4(accountId)+EncStr |
| RegisterPinHandle (REGISTER_PIN) | 9 / 0x09 | sub_503956 @0x5039f3 | sub_5688CE case1 (10) | −1 | Enc1(1)+EncStr / Enc1(0) |
| ServerListRequestHandle (SERVERLIST_REQUEST) | 10 / 0x0A | sub_503956 @0x503956 | sub_5688CE case0 (11) | −1 | bare COutPacket(10) |
| CharacterViewAllHandle (VIEW_ALL_CHAR) | 12 / 0x0C | sub_502293 @0x5022e1 | sub_567117 (13) | −1 | bare op12 (preceded by bare op11 = unregistered v61 op12) |
| CharacterViewAllSelectedHandle (PICK_ALL_CHAR) | 13 / 0x0D | sub_500254 @0x50037f | SendSelectCharPacketByVAC sub_5650B6 (14) | −1 | Enc4(charId)+Enc4(worldCharId)+EncStr(mac) |
| CharacterViewAllPongHandle (VAC) | 14 / 0x0E | sub_50273F @0x502773 | MakeVACDlg sub_5675C4 (15) | −1 | Enc1(1) / (ResetVAC Enc1(0)) |
| CharacterSelectedHandle (CHAR_SELECT) | 15 / 0x0F | sub_500174 @0x5001c5 | SendSelectCharPacket sub_564F79 (19) | −4 | Enc4(charId)+EncStr(mac) |
| CharacterLoggedInHandle (PLAYER_LOGGEDIN) | 16 / 0x10 | CClientSocket::OnConnect @0x463d9f | OnConnect else-branch (20) | −4 | Enc4(charId)+Enc1(0) |
| CharacterCheckNameHandle (CHECK_CHAR_NAME) | 17 / 0x11 | sub_500693 @0x5006e7 | SendCheckDuplicateIDPacket (21) | −4 | EncStr(name) |
| CreateCharacterHandle (CREATE_CHAR) | 21 / 0x15 | sub_500545 @0x50058b | SendNewCharPacket (22) | −1 | EncStr(name)+8×Enc4(appearance)+Enc1(gender)+mac |
| DeleteCharacterHandle (DELETE_CHAR) | 22 / 0x16 | sub_50043F @0x5004bb | SendDeleteCharPacket (23) | −1 | Enc4(DOB)+Enc4(charId) |
| PongHandle (PONG) | 23 / 0x17 | CClientSocket::OnAliveReq @0x46509a | OnAliveReq (24) | −1 | bare COutPacket(23) |
| StartErrorHandle (CLIENT_START_ERROR) | 24 / 0x18 | CClientSocket::OnConnect @0x463d42 | OnConnect exception (25) | −1 | Enc2(len)+EncBuffer(crashlog) — this IS the ExceptionLog send |

Already present (not duplicated): **LOGIN_PASSWORD** (op1, LoginHandle),
**SET_GENDER** (op7, SetGenderHandle). v48 SET_GENDER=op7 is a mode-prefixed
op (Enc1(mode)+..., mode1=set/gender, mode0=decline via sub_503787).

## Validators (match v61/v79 anchor)
- **NoOpValidator** (pre-LoggedIn): CharacterLoggedInHandle (0x10), PongHandle
  (0x17), StartErrorHandle (0x18). (Plus the pre-existing LoginHandle 0x01.)
- **LoggedInValidator**: all 14 others.
- 0 handlers missing a validator (would be silently dropped by BuildHandlerMap).

## PIN/PIC verification (usesPin=false)
- **PIC (character 2nd-password / SPW)**: no v48 send-site — the CLogin dispatcher
  (sub_5007C4) has NO SPW cases and the login clientbound registry carries
  usesPin=false (Stage A/C). Genuinely absent → NOT registered. Correct.
- **PIN (RegisterPin op9 / AfterLogin op8)**: the send-sites ARE body-present in
  the binary (sub_503956 arms). usesPin=false gates the client PIN *UI* at
  runtime, not the handler code. Registered for full anchor parity (v61 AND v79
  templates both include AfterLoginHandle+RegisterPinHandle) so a runnable tenant
  never silently drops the packet if the client emits it.

## v48-absent / not-a-handler
- **ExceptionLog**: not a separate op — it IS CLIENT_START_ERROR (op24), the
  crash-log the client sends on reconnect. Registered as StartErrorHandle.
- **MigrateToChannel**: already covered by the gameplay ChannelChangeHandle
  (0x1F / CHANGE_CHANNEL op31), present since Stage C. No login-region variant.
- **v48 op11** (bare, sub_502293 leading packet) and **op27** (bare, sub_4FE15D
  second packet) are twins of v61 op12 / op28 — neither is a registered Atlas
  handler in any version. Left unregistered (documented, not fabricated).

## Flagged / blocked
- None. Every target login op resolved to a body-verified send-site + v61 twin.

## Counts & validation
- Registry `gms_v48.yaml`: serverbound 58 → **75** (+17); clientbound 93
  (unchanged). 0 duplicate opcodes per direction; all 168 entries carry
  fname+provenance(ida-discovered)+ida.address.
- Template `template_gms_48_1.json`: handlers 57 → **74** (+17). Valid JSON
  (`json.tool`). 0 duplicate opCodes. 0 handlers missing a validator.
- `go build ./...` clean in atlas-configurations; `go test ./templates/...` ok.

## Summary
17 login-flow serverbound ops registered / 0 absent (PIC genuinely absent, not a
gap) / 0 flagged. New handler total: 74.
