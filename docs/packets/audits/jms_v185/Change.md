# Change (← `CWvsContext::SendGivePopularityRequest`)

- **IDA:** 0xb0b21e
- **Atlas file:** `libs/atlas-packet/fame/serverbound/change.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId (opcode 0x5A; taken from CUser object at offset +4992)` | ✅ |  |
| 1 | byte | byte `nType (bInc, 1=fame, 0=defame)` | ✅ |  |

