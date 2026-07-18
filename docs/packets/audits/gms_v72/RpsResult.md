# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x69c7f2
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher sub_69C7A5 (0x69c7a5) case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x69c7f2)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x69c7fd; client treats as signed __int8, branches on <0)` | ✅ |  |

