# AuthLoginFailed (← `CLogin::OnCheckPasswordResult#AuthLoginFailed`)

- **IDA:** 0x5cd38f
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_login_failed.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (failure code) — v79 OnCheckPasswordResult@0x5cd38f v105=Decode1` | ✅ |  |
| 1 | byte | byte `post-auth flag — this+416=Decode1` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch) — Decode4(v2)` | ✅ |  |

