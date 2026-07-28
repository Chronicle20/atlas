# FieldItcOperationChangedPage (← `CITC::OnChangedPage`)

- **IDA:** 0x57aa02
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x59f495` | ✅ |  |
| 1 | int32 | int32 `category @0x59f4a0` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x59f4ab` | ✅ |  |
| 3 | int32 | int32 `page @0x59f4b6` | ✅ |  |
| 4 | byte | byte `sortType @0x59f4c4` | ✅ |  |
| 5 | byte | byte `sortColumn @0x59f4d2` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x59f4e0` | ✅ |  |
| 7 | string | string `searchCondition @0x59f4fd` | ✅ |  |

