# FieldItcOperationChangedPage (← `CITC::OnChangedPage`)

- **IDA:** 0x573af0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5 GetItcList browse) @0x573b3c` | ✅ |  |
| 1 | int32 | int32 `category (m_nCurCategory) @0x573b49` | ✅ |  |
| 2 | int32 | int32 `categorySub (m_nCurCategorySub) @0x573b56` | ✅ |  |
| 3 | int32 | int32 `page (nPage) @0x573b64` | ✅ |  |
| 4 | byte | byte `sortType (m_nSortType) @0x573b72` | ✅ |  |
| 5 | byte | byte `sortColumn (m_nSortColumn) @0x573b80` | ✅ |  |
| 6 | int32 | int32 `searchOption (m_nSearchOption) @0x573b90` | ✅ |  |
| 7 | string | string `searchCondition (m_sSearchCondition) @0x573bb2` | ✅ |  |

