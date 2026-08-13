# FieldSueCharacter (← `CField::SendChatMsgSlash#SueCharacter`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/field/serverbound/sue_character.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

