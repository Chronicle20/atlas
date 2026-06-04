# PetExcludeResponse (← `CPet::OnLoadExceptionList`)

- **IDA:** 0x7061a5
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/exclude.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | int64 | bytes `petLockerSN (8 bytes)` | ✅ |  |
| 3 | byte | byte `nCount` | ✅ |  |
| 4 | int32 | int32 `excluded item id per entry` | ✅ |  |

