# FieldItcOperationTabSearch (← `CITCWnd_Tab::OnButtonClicked`)

- **IDA:** 0x584b10
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6 search-by-name) @0x584bd7/@0x584cd9` | ✅ |  |
| 1 | int32 | int32 `category (m_nSelect+1) @0x584be1/@0x584ce3` | ✅ |  |
| 2 | int32 | int32 `categorySub (m_nSelect) @0x584beb/@0x584ced` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x584bf5/@0x584cf8` | ✅ |  |
| 4 | int32 | int32 `searchOption @0x584bff/@0x584d02` | ✅ |  |
| 5 | string | string `searchCondition (edit-box text) @0x584c1b/@0x584d22` | ✅ |  |

