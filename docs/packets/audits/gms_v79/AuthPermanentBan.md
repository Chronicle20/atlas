# AuthPermanentBan (← `CLogin::OnCheckPasswordResult#AuthPermanentBan`)

- **IDA:** 0x5cd38f
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_permanent_ban.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (== 27)` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch); v79 27-branch reads no trailing (GMS)` | ✅ |  |

