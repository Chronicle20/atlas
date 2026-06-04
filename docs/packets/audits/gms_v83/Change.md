# Change (← `CWvsContext::SendGivePopularityRequest`)

- **IDA:** 0xa23eb5
- **Atlas file:** `../../libs/atlas-packet/fame/serverbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId (target character ID as uint32 @0xa23f7e)` | ✅ |  |
| 1 | byte | byte `bInc (1=fame, 0=defame @0xa23f89)` | ✅ |  |

