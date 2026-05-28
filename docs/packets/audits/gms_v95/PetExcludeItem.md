# PetExcludeItem (← `CPet::SendUpdateExceptionListRequest`)

- **IDA:** 0x6a0dd0
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/exclude_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes — _LARGE_INTEGER)` | ❌ | width mismatch |
| 1 | byte | byte `nCount (exception list size)` | ✅ |  |
| 2 | int32 | int32 `itemId per entry — loop nCount times` | ✅ |  |

