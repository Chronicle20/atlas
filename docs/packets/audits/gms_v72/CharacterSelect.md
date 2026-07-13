# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 0x5b1d03
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | string | string `` | ✅ |  |

