# ChatGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x534000
- **Atlas file:** `libs/atlas-packet/chat/serverbound/general.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time(); gated GMS>83 in atlas — v95 always present)` | ✅ |  |
| 1 | string | string `chat message text (sText)` | ✅ |  |
| 2 | byte | byte `bOnlyBalloon flag` | ✅ |  |

