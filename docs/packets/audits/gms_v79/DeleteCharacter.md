# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x5cce4b
- **Atlas file:** `libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v79
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dob @0x5ccf28` | ✅ |  |
| 1 | int32 | int32 `characterId @0x5ccf45` | ✅ |  |

