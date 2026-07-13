# CheckName (← `CLogin::SendCheckDuplicateIDPacket`)

- **IDA:** 0x46bb81
- **Atlas file:** `libs/atlas-packet/character/serverbound/check_name.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name @0x46bc04` | ✅ |  |

