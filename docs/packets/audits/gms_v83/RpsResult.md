# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x740298
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher sub_74024B (0x74024b) case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x740298)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x7402a3; client treats as signed __int8, branches on <0)` | ✅ |  |

