# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x7ae683
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher sub_7AE636 (0x7ae636) case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x7ae683)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x7ae68e; client treats as signed __int8, branches on <0)` | ✅ |  |

