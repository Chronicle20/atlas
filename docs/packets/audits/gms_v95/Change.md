# Change (← `CWvsContext::SendGivePopularityRequest`)

- **IDA:** 0x9f67e0
- **Atlas file:** `../../libs/atlas-packet/fame/serverbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId (target character ID as uint32)` | ✅ |  |
| 1 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |

