# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x6d7372
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (CRPSGameDlg::ProcessPacket (0x6d72d0) case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x6d7372)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x6d737d; client treats as signed __int8, branches on <0)` | ✅ |  |

