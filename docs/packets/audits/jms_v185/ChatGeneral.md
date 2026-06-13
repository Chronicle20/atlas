# ChatGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x564a0a
- **Atlas file:** `libs/atlas-packet/chat/serverbound/general.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time()` | ✅ |  |
| 1 | string | string `chat text` | ✅ |  |
| 2 | byte | byte `bOnlyBalloon` | ✅ |  |

