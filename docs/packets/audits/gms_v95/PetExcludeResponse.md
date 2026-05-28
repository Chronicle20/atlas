# PetExcludeResponse (← `CPet::OnLoadExceptionList`)

- **IDA:** 0x6a1510
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/exclude.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | int64 | bytes `petLockerSN (8 bytes — _LARGE_INTEGER)` | ❌ | width mismatch |
| 3 | byte | byte `nCount (exception list size)` | ✅ |  |
| 4 | int32 | int32 `excluded item id — per entry, loop nCount times` | ✅ |  |

