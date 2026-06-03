# CheckName (← `CLogin::SendCheckDuplicateIDPacket`)

- **IDA:** 0x5d5690
- **Atlas file:** `libs/atlas-packet/character/serverbound/check_name.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sCharName (character name to check)` | ✅ |  |

