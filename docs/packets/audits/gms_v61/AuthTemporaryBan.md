# AuthTemporaryBan (← `CLogin::OnCheckPasswordResult#AuthTemporaryBan`)

- **IDA:** 0x5657ce
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_temporary_ban.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bannedCode/resultCode @0x5657f9 (==2)` | ✅ |  |
| 1 | byte | byte `post-auth flag @0x5657ff` | ✅ |  |
| 2 | int32 | int32 `reserved GMS int @0x56580d` | ✅ |  |
| 3 | byte | byte `ban reason @0x56583f (Value)` | ✅ |  |
| 4 | int64 | int64 `chatUnblockDate FILETIME @0x565848 (DecodeBuffer 8)` | ✅ |  |

