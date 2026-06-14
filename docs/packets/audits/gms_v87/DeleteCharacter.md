# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x62f3d3
- **Atlas file:** `libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (soft-keyboard password)` | ✅ |  |
| 1 | int32 | int32 `characterStat.dwCharacterID` | ✅ |  |

