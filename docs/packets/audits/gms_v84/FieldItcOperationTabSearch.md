# FieldItcOperationTabSearch (← `CITCWnd_Tab::OnButtonClicked`)

- **IDA:** 0x5c77ca
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (6) @0x5c7940/@0x5c7877` | ✅ |  |
| 1 | int32 | int32 `category (m_nSelect+1) @0x5c7949/@0x5c7880` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5c7954/@0x5c788b` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5c795d/@0x5c7894` | ✅ |  |
| 4 | int32 | int32 `searchOption @0x5c7968/@0x5c789d` | ✅ |  |
| 5 | string | string `searchName @0x5c7981/@0x5c78b6` | ✅ |  |

