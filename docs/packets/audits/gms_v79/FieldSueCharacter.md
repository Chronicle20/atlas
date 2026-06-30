# FieldSueCharacter (← `CField::SendChatMsgSlash#SueCharacter`)

- **IDA:** 0x51825e
- **Atlas file:** `libs/atlas-packet/field/serverbound/sue_character.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId @0x518275` | ✅ |  |
| 1 | byte | byte `flag @0x51827e` | ✅ |  |
| 2 | string | string `reason` | ✅ |  |

