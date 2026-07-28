# RpsOpen (← `CRPSGameDlg::OnPacket#OPEN`)

- **IDA:** 0x7ae4d7
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (OnPacket entry Decode1 @0x7ae3ee; case 8 = OPEN)` | ✅ |  |
| 1 | int32 | int32 `ante — participation fee (CInPacket::Decode4 @0x7ae4d7; the StringPool notice string that follows is a static resource, not a packet field)` | ✅ |  |

