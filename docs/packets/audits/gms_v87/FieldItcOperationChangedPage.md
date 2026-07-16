# FieldItcOperationChangedPage (← `CITC::OnChangedPage`)

- **IDA:** 0x5cf1c8
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5cf1f8` | ✅ |  |
| 1 | int32 | int32 `category @0x5cf203` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5cf20e` | ✅ |  |
| 3 | int32 | int32 `page @0x5cf219` | ✅ |  |
| 4 | byte | byte `sortType @0x5cf227` | ✅ |  |
| 5 | byte | byte `sortColumn @0x5cf235` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x5cf243` | ✅ |  |
| 7 | string | string `searchCondition @0x5cf260` | ✅ |  |

