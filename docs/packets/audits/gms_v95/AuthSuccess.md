# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x5dc600
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** GMS/v95/modified
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `reserved (unused)` | ✅ |  |
| 3 | int32 | int32 `accountId` | ✅ |  |
| 4 | byte | byte `gender` | ✅ |  |
| 5 | byte | byte `GM/admin flag` | ✅ |  |
| 6 | int16 | int16 `subGradeCode+testerAccount` | ✅ |  |
| 7 | byte | byte `countryCode` | ✅ |  |
| 8 | string | string `nexonClubID` | ✅ |  |
| 9 | byte | byte `purchaseExp/quiet-ban reason` | ✅ |  |
| 10 | byte | byte `quiet-ban code` | ✅ |  |
| 11 | int64 | int64 `chatUnblockDate (FILETIME)` | ✅ |  |
| 12 | int64 | int64 `registerDate (FILETIME)` | ✅ |  |
| 13 | int32 | int32 `numOfCharacter` | ✅ |  |
| 14 | byte | byte `pinFlag` | ✅ |  |
| 15 | byte | byte `picFlag` | ✅ |  |
| 16 | int64 | int64 `clientKey/MAC (unconditional)` | ✅ |  |

