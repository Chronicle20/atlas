# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x62fb84
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `post-auth flag (m_nRegStatID stored)` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch)` | ✅ |  |
| 3 | int32 | int32 `accountId` | ✅ |  |
| 4 | byte | byte `gender` | ✅ |  |
| 5 | byte | byte `GM/admin flag` | ✅ |  |
| 6 | byte | byte `admin byte (v87 still 2×Decode1; v95 widened to Decode2)` | ✅ |  |
| 7 | byte | byte `countryCode` | ✅ |  |
| 8 | string | string `nexonClubID` | ✅ |  |
| 9 | byte | byte `purchaseExp/quiet-ban reason` | ✅ |  |
| 10 | byte | byte `quiet-ban code` | ✅ |  |
| 11 | int64 | int64 `chatUnblockDate (DecodeBuffer 8 bytes)` | ✅ |  |
| 12 | int64 | int64 `registerDate (DecodeBuffer 8 bytes)` | ✅ |  |
| 13 | int32 | int32 `numOfCharacter` | ✅ |  |
| 14 | byte | byte `pinFlag` | ✅ |  |
| 15 | byte | byte `picFlag (m_bLoginOpt)` | ✅ |  |
| 16 | int64 | int64 `clientKey (DecodeBuffer 8 bytes — identical to v83/v95)` | ✅ |  |

