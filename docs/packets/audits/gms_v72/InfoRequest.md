# InfoRequest (← `CWvsContext::SendCharacterInfoRequest`)

- **IDA:** 0x91c174
- **Atlas file:** `libs/atlas-packet/character/serverbound/info_request.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time @0x91c1e8` | ✅ |  |
| 1 | int32 | int32 `characterId @0x91c1f1` | ✅ |  |
| 2 | byte | byte `petInfo @0x91c1fc` | ✅ |  |

