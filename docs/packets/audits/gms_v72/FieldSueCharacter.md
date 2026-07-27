# FieldSueCharacter (← `CField::SendChatMsgSlash#SueCharacter`)

- **IDA:** 0x50c2c3
- **Atlas file:** `libs/atlas-packet/field/serverbound/sue_character.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId @0x50c2e3` | ✅ |  |
| 1 | byte | byte `flag @0x50c2f1` | ✅ |  |
| 2 | string | string `reason @0x50c30e` | ✅ |  |

