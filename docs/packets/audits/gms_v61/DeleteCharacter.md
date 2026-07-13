# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x5652e3
- **Atlas file:** `libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v61
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dob @0x56536d` | ✅ |  |
| 1 | int32 | int32 `characterId @0x56538a` | ✅ |  |

