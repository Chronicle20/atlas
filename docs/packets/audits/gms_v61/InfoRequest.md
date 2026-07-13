# InfoRequest (← `CWvsContext::SendCharacterInfoRequest`)

- **IDA:** 0x845b68
- **Atlas file:** `libs/atlas-packet/character/serverbound/info_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time @0x845bdc` | ✅ |  |
| 1 | int32 | int32 `characterId @0x845be5` | ✅ |  |
| 2 | byte | byte `petInfo @0x845bf0` | ✅ |  |

