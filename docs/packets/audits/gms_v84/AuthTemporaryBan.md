# AuthTemporaryBan (← `CLogin::OnCheckPasswordResult#AuthTemporaryBan`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_temporary_ban.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |

