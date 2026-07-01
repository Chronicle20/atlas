# InfoRequest (← `CWvsContext::SendCharacterInfoRequest`)

- **IDA:** 0x96e184
- **Atlas file:** `libs/atlas-packet/character/serverbound/info_request.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time()) @0x96e1f8` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID (target character ID) @0x96e201` | ✅ |  |
| 2 | byte | byte `bPetInfo (1 = include pet info) @0x96e20c` | ✅ |  |

