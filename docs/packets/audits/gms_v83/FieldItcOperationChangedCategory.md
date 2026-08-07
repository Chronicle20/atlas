# FieldItcOperationChangedCategory (← `CITC::OnChangedCategory`)

- **IDA:** 0x59f297
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x59f2ec` | ✅ |  |
| 1 | int32 | int32 `category @0x59f2f7` | ✅ |  |
| 2 | int32 | int32 `categorySub (const 0) @0x59f300` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x59f309` | ✅ |  |
| 4 | byte | byte `sortType (const 1) @0x59f315` | ✅ |  |
| 5 | byte | byte `sortColumn (const 1) @0x59f31e` | ✅ |  |
| 6 | int32 | int32 `searchOption (const 1) @0x59f327` | ✅ |  |
| 7 | string | string `searchCondition (const empty) @0x59f33f` | ✅ |  |

