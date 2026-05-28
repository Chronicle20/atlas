# AuthLoginFailed (← `CLogin::OnCheckPasswordResult#AuthLoginFailed`)

- **IDA:** 0x5f83ee
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/auth_login_failed.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (failure code)` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch)` | ✅ |  |

