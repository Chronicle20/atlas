# ChatGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x52c315
- **Atlas file:** `../../libs/atlas-packet/chat/serverbound/general.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `text` | ✅ |  |
| 1 | byte | byte `balloon (bShowBalloon)` | ✅ |  |

