# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x66e79f
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `accountId (success path, resultCode==0 or 12)` | ✅ |  |
| 3 | byte | byte `nGender` | ✅ |  |
| 4 | byte | byte `nGradeCode (packed byte, differs from GMS v95 Decode2)` | ✅ |  |
| 5 | byte | byte `combined byte (bits for tester/admin flags)` | ✅ |  |
| 6 | string | string `nexon club ID 1 (JMS-specific: two strings instead of one)` | ✅ |  |
| 7 | string | string `nexon club ID 2 (JMS-specific extra string)` | ✅ |  |
| 8 | byte | byte `purchaseExp/chatBlockReason` | ✅ |  |
| 9 | byte | byte `nChatBlockReason` | ✅ |  |
| 10 | byte | byte `flag byte` | ✅ |  |
| 11 | byte | byte `flag byte 2` | ✅ |  |
| 12 | byte | byte `flag byte 3` | ✅ |  |
| 13 | byte | byte `lastByte` | ✅ |  |
| 14 | int64 | bytes `chatUnblockDate FILETIME (8 bytes)` | ✅ |  |
| 15 | string | byte `` | ✅ | absorbed by trailing opaque buffer |

