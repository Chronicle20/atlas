# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x5f7c4a
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (secondary password / deletion password from dialog)` | ✅ |  |
| 1 | int32 | int32 `characterStat.dwCharacterID` | ✅ |  |

