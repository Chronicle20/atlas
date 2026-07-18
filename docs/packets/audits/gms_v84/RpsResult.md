# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x761fbc
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher sub_761F6F (0x761f6f) case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x761fbc)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x761fc7; client treats as signed __int8, branches on <0)` | ✅ |  |

