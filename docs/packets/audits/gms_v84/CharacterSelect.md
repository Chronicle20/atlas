# CharacterSelect (← `CLogin::SendSelectCharPacket`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/login/serverbound/character_select.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |

