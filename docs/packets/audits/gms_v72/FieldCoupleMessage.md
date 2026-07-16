# FieldCoupleMessage (← `CUIStatusBar::SendCoupleMessage`)

- **IDA:** 0x7f4651
- **Atlas file:** `libs/atlas-packet/field/serverbound/spouse_chat.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `spouseName @0x7f46fd` | ✅ |  |
| 1 | string | string `message @0x7f4715` | ✅ |  |

