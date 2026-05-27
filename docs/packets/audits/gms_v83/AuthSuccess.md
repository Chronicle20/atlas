# AuthSuccess (← `CLogin::OnCheckPasswordResult`)

- **IDA:** 0x5f83ee
- **Atlas file:** `libs/atlas-packet/login/clientbound/auth_success.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `post-auth flag` | ✅ |  |
| 2 | int32 | int32 `reserved (always decoded before branch)` | ✅ |  |
| 3 | int32 | int32 `accountId` | ✅ |  |
| 4 | byte | byte `gender` | ✅ |  |
| 5 | byte | byte `GM/admin flag` | ✅ |  |
| 6 | byte | byte `admin byte (v83 byte, v95 widened to int16)` | ✅ |  |
| 7 | byte | byte `countryCode` | ✅ |  |
| 8 | string | string `nexonClubID (atlas writes character name into this slot)` | ✅ |  |
| 9 | byte | byte `purchaseExp/quiet-ban reason` | ✅ |  |
| 10 | byte | byte `quiet-ban code` | ✅ |  |
| 11 | int64 | int64 `chatUnblockDate FILETIME (8-byte buffer)` | ✅ |  |
| 12 | int64 | int64 `registerDate FILETIME (8-byte buffer)` | ✅ |  |
| 13 | int32 | int32 `numOfCharacter` | ✅ |  |
| 14 | byte | byte `pinFlag` | ✅ |  |
| 15 | byte | byte `picFlag` | ✅ |  |

