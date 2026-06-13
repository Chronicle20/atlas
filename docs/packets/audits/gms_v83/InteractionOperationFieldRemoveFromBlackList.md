# InteractionOperationFieldRemoveFromBlackList (← `CField::DeleteBlackList`)

- **IDA:** 0x5379a6
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_field_remove_from_black_list.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `` | ❌ | width mismatch |
| 1 | byte | string `` | ❌ | atlas: short — missing trailing field |

