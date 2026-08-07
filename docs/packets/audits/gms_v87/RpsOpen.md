# RpsOpen (← `CRPSGameDlg::OnPacket#OPEN`)

- **IDA:** 0x78acb0
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (OnPacket entry Decode1 @0x78abb9; case 8 = OPEN)` | ✅ |  |
| 1 | int32 | int32 `ante — participation fee (CInPacket::Decode4 @0x78acb0; the StringPool notice string that follows is a static resource, not a packet field)` | ✅ |  |

