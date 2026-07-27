# RpsResult (← `CRPSGameDlg::OnPacket#RESULT`)

- **IDA:** 0x63c1b3
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (dispatcher @0x63bf0e delegates modes 6/7/9-12/14 to sub_63C166 @0x63c166; case 11 = RESULT)` | ✅ |  |
| 1 | byte | byte `npcThrow (CInPacket::Decode1 @0x63c1b3)` | ✅ |  |
| 2 | byte | byte `straightVictoryCount, SIGNED int8 (CInPacket::Decode1 @0x63c1be; client treats as signed __int8, branches on <0)` | ✅ |  |

