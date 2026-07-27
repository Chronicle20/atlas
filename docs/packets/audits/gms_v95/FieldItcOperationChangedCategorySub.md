# FieldItcOperationChangedCategorySub (← `CITC::OnChangedCategorySub`)

- **IDA:** 0x5739a0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5 GetItcList browse) @0x5739ed` | ✅ |  |
| 1 | int32 | int32 `category (m_nCurCategory) @0x5739fa` | ✅ |  |
| 2 | int32 | int32 `categorySub (nCategorySub) @0x573a08` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x573a13` | ✅ |  |
| 4 | byte | byte `sortType (nSortType) @0x573a21` | ✅ |  |
| 5 | byte | byte `sortColumn (nSortColumn) @0x573a2f` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x573a3f/0x573a7e` | ✅ |  |
| 7 | string | string `searchCondition @0x573aa0` | ✅ |  |

