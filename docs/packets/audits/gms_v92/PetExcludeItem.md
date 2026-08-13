# PetExcludeItem (← `CPet::SendUpdateExceptionListRequest`)

- **IDA:** 0x6963a0
- **Atlas file:** `libs/atlas-packet/pet/serverbound/exclude_item.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

