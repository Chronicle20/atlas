# FieldItcOperationTabSearch (← `CITCWnd_Tab::OnButtonClicked`)

- **IDA:** 0x5b7106
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6) @0x5b71b3` | ✅ |  |
| 1 | int32 | int32 `category (m_nSelect+1) @0x5b71bc` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5b71c7` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5b71d0` | ✅ |  |
| 4 | int32 | int32 `searchOption @0x5b71d9` | ✅ |  |
| 5 | string | string `searchName @0x5b71f2` | ✅ |  |

