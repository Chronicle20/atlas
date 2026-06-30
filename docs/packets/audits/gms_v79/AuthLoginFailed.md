# AuthLoginFailed (← `CLogin::OnCheckPasswordResult#AuthLoginFailed`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_login_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

