# CharacterSelectRegisterPic (← `CLogin::SendSelectCharPacket#CharacterSelectRegisterPic`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select_register_pic.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |

