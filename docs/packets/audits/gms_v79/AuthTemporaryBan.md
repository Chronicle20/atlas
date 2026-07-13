# AuthTemporaryBan (← `CLogin::OnCheckPasswordResult#AuthTemporaryBan`)

- **IDA:** 0x5cd38f
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_temporary_ban.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode (== 2)` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch)` | ✅ |  |
| 3 | byte | byte `ban reason (v6=Decode1)` | ✅ |  |
| 4 | int64 | int64 `chatUnblockDate FILETIME (DecodeBuffer 8)` | ✅ |  |

