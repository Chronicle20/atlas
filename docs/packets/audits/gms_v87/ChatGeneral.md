# ChatGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x552b67
- **Atlas file:** `../../libs/atlas-packet/chat/serverbound/general.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (remote tick)` | ✅ |  |
| 1 | string | string `message text` | ✅ |  |
| 2 | byte | byte `nEmotion` | ✅ |  |

