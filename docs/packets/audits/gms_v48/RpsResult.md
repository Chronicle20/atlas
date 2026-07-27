# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x5ade39
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher sub_5ADB94 delegates modes 6/7/9-12/14 to sub_5ADDEC @0x5addec; case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x5ade39)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x5ade44; client treats as signed __int8, branches on <0)` | ✅ |  |

