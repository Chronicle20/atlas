# AllCharacterListSelectWithPicRegister (← `CLogin::SendSelectCharPacketByVAC#AllCharacterListSelectWithPicRegister`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_select_with_pic_register.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | string | byte `` | ❌ | atlas: extra — client never reads this field |

