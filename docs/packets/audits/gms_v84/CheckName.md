# CheckName (← `CLogin::SendCheckDuplicateIDPacket`)

- **IDA:** 0x60cf5d
- **Atlas file:** `libs/atlas-packet/character/serverbound/check_name.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sCharName (character name to check) @0x60cfc3` | ✅ |  |

