# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x5cb860
- **Atlas file:** `libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v92
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |

