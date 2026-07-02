# CheckName (← `CLogin::SendCheckDuplicateIDPacket`)

- **IDA:** 0x565537
- **Atlas file:** `libs/atlas-packet/character/serverbound/check_name.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name @0x5655a8` | ✅ |  |

