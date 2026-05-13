# Pending IDA function exports

This list tracks IDA functions referenced by the login-domain audit matrix
(task-027) but NOT yet in `gms_v95.json`. Each row needs a future maintainer
run of `packet-audit export ...` (live IDA-MCP) or hand-derivation from a
focused spike doc to add the function's wire-layout.

| FName | Atlas writer/handler | Direction | Notes |
|---|---|---|---|
| `CLogin::OnCheckPasswordResult` (failure branch) | AuthLoginFailed, AuthTemporaryBan, AuthPermanentBan | clientbound | shares opcode 0x00 with success branch; failure path has shorter wire (byte + reason + reserved int32) |
| `CLogin::OnSetAccountResult` | SetAccountResult | clientbound | |
| `CLogin::OnCheckPinCodeResult` | PinOperation, PinUpdate | clientbound | |
| `CLogin::SendCheckPinCodePacket` | RegisterPinHandle, etc. | serverbound | |
| `CLogin::SendSelectCharPacketByPIC` | CharacterSelectedPicHandle, RegisterPicHandle | serverbound | PIC verify/register branches of opcode 0x13 family |
| `CLogin::SendCheckUserLimitPacket` | ServerStatusRequest | serverbound | |
| `CLogin::SendAcceptLicensePacket` / `CLicenseDlg::OnButtonClicked` | AcceptTosHandle | serverbound | |
| `CLogin::SendViewAllCharPacket` | AllCharacterListRequest | serverbound | |
| `CLogin::OnAllCharlistResult` / `OnViewAllCharResult` | AllCharacterListPong | clientbound | |
| `CWvsApp::SendBackupPacket` / `SendClearStackLog` | AfterLoginHandle | serverbound | |
| `CLogin::OnRecommendWorldMessage` | ServerListRecommendations | clientbound | |
| `CLogin::OnLatestConnectedWorld` | LoginAuth | clientbound | |
| `CLogin::OnSelectCharByVAC` / `MakeVACDlg` | (VAC family) | clientbound | |

When refreshing: run `packet-audit export --ida-source mcp ...` on a maintainer
workstation, or hand-derive via a focused spike doc.
