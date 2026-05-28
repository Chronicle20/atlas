# ChatGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x552b67
- **Atlas file:** `../../libs/atlas-packet/chat/serverbound/general.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `message text` | ❌ | width mismatch |
| 1 | string | int32 `nEmotion (emote ID, packed with extra data in v95)` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

