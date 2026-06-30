# FieldCoupleMessage (← `CUIStatusBar::SendCoupleMessage`)

- **IDA:** 0x83cd67
- **Atlas file:** `libs/atlas-packet/field/serverbound/spouse_chat.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `spouseName @0x83ce13` | ✅ |  |
| 1 | string | string `message @0x83ce2b` | ✅ |  |

