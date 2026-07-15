# FieldAdminChat (← `CField::SendChatMsgSlash#AdminChat`)

- **IDA:** 0x5194ac
- **Atlas file:** `libs/atlas-packet/field/serverbound/admin_chat.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `chatType @0x5194bf` | ✅ |  |
| 1 | byte | byte `flag @0x5194c9` | ✅ |  |
| 2 | string | string `message` | ✅ |  |

