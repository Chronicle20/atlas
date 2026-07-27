# FieldItcOperationTabSearch (← `CITCWnd_Tab::OnButtonClicked`)

- **IDA:** 0x61e3cf
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6) @0x61e47c` | ✅ |  |
| 1 | int32 | int32 `category (m_nSelect+1) @0x61e485` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x61e490` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x61e499` | ✅ |  |
| 4 | int32 | int32 `searchOption @0x61e4a2` | ✅ |  |
| 5 | string | string `searchName @0x61e4bb` | ✅ |  |

