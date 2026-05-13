# Pending IDA function exports

This list tracks IDA functions referenced by the login-domain audit matrix
(task-027) but NOT yet in `gms_v95.json`. Each row needs a future maintainer
run of `packet-audit export ...` (live IDA-MCP) or hand-derivation from a
focused spike doc to add the function's wire-layout.

## Resolved (now in gms_v95.json)

- `CLogin::OnSetAccountResult` → SetAccountResult ✅
- `CLogin::OnCheckPinCodeResult` → PinOperation, PinUpdate ✅
- `CLogin::SendCheckUserLimitPacket` → ServerStatusRequest ✅ (fix shipped)
- `CLogin::OnRecommendWorldMessage` → ServerListRecommendations 🔍 (sub-struct loop)
- `CLogin::OnLatestConnectedWorld` → SelectWorld ✅ (re-mapped; LoginAuth doesn't match this FName)

## Still pending

| FName | Atlas writer/handler | Direction | Notes |
|---|---|---|---|
| `CLogin::OnCheckPasswordResult` (failure branch) | AuthLoginFailed, AuthTemporaryBan, AuthPermanentBan | clientbound | shares opcode 0x00 with success branch; failure path has shorter wire. Needs synthetic FName mechanism in pipeline to distinguish branches. |
| `CLogin::SendCheckPinCodePacket` | RegisterPinHandle | serverbound | FName not found by that exact name in IDA — needs reverse search via opcode |
| `CLogin::SendSelectCharPacketByPIC` | CharacterSelectedPicHandle, RegisterPicHandle | serverbound | FName not found by that exact name. PIC verify/register branches of opcode 0x13 family. |
| `CLogin::SendAcceptLicensePacket` / `CLicenseDlg::OnButtonClicked` | AcceptTosHandle | serverbound | FName not found by either exact name. |
| `CLogin::SendViewAllCharPacket` | AllCharacterListRequest | serverbound | Found at 0x5dfb40 (size 0x4dd); decompile not yet processed |
| `CLogin::OnViewAllCharResult` | AllCharacterListPong | clientbound | Found at 0x5de120 (size 0x521); decompile not yet processed |
| `CWvsApp::SendBackupPacket` | NoOpHandler (opcode 0x24) | serverbound | atlas-side handler is a no-op; nothing to audit. Found at 0x9c7a80; can stub the export entry for future use |
| `CLogin::OnSelectCharByVAC` / `MakeVACDlg` | (VAC family) | clientbound | Function family not searched yet |
| `LoginAuth` (atlas writer) | — | clientbound | orphan: atlas writes `WriteAsciiString(screen)`. No IDA function matches. May be legacy v83 packet. |

## Workflow notes

The current `candidatesFromFName` map in `tools/packet-audit/cmd/run.go` is
the source of truth for which FNames the audit pipeline visits. Adding a
new IDA export entry requires updating both the JSON and that map.

Refreshing via MCP: `mcp__ida-pro__get_function_by_name` (resolve address)
followed by `mcp__ida-pro__decompile_function` (extract C source). Parse
the `CInPacket::DecodeN` / `COutPacket::EncodeN` call sequence in lexical
order. Complex multi-branch functions (e.g., `OnCheckPasswordResult`,
`OnCheckPinCodeResult`) need manual filtering for the success path.
