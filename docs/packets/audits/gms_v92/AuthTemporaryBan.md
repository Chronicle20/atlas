# AuthTemporaryBan (← `CLogin::OnCheckPasswordResult#AuthTemporaryBan`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_temporary_ban.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

