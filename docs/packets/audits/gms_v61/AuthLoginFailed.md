# AuthLoginFailed (← `CLogin::OnCheckPasswordResult#AuthLoginFailed`)

- **IDA:** 0x5657ce
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_login_failed.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `status/result @0x5657f9 (v4)` | ✅ |  |
| 1 | byte | byte `@0x5657ff (stored this+384)` | ✅ |  |
| 2 | int32 | int32 `@0x56580d (GMS int, discarded)` | ✅ |  |

