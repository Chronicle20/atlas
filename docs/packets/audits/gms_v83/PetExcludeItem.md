# PetExcludeItem (← `CPet::SendUpdateExceptionListRequest`)

- **IDA:** 0x706393
- **Atlas file:** `libs/atlas-packet/pet/serverbound/exclude_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ❌ | width mismatch |
| 1 | byte | byte `nCount` | ✅ |  |
| 2 | int32 | int32 `itemId per entry` | ✅ |  |

