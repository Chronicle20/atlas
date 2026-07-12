# FieldItcOperationChangedCategorySub (← `CITC::OnChangedCategorySub`)

- **IDA:** 0x59f376
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x59f3a8` | ✅ |  |
| 1 | int32 | int32 `category @0x59f3b3` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x59f3bf` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x59f3c8` | ✅ |  |
| 4 | byte | byte `sortType @0x59f3d3` | ✅ |  |
| 5 | byte | byte `sortColumn @0x59f3de` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x59f3ed/@0x59f411` | ✅ |  |
| 7 | string | string `searchCondition @0x59f42d` | ✅ |  |

