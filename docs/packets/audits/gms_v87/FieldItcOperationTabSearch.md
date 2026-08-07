# FieldItcOperationTabSearch (← `CITCWnd_Tab::OnButtonClicked`)

- **IDA:** 0x5e7a64
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6) @0x5e7bda/@0x5e7b11` | ✅ |  |
| 1 | int32 | int32 `category (m_nSelect+1) @0x5e7be3/@0x5e7b1a` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5e7bee/@0x5e7b25` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5e7bf7/@0x5e7b2e` | ✅ |  |
| 4 | int32 | int32 `searchOption @0x5e7c02/@0x5e7b37` | ✅ |  |
| 5 | string | string `searchName @0x5e7c1b/@0x5e7b50` | ✅ |  |

