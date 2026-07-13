# FieldAdminChat (← `CField::SendChatMsgSlash#AdminChat`)

- **IDA:** 0x50f8b8
- **Atlas file:** `libs/atlas-packet/field/serverbound/admin_chat.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `chatType @0x50f8d1` | ✅ |  |
| 1 | byte | byte `flag @0x50f8de` | ✅ |  |
| 2 | string | string `message @0x50f8f8` | ✅ |  |

