# FieldItcOperationChangedCategorySub (← `CITC::OnChangedCategorySub`)

- **IDA:** 0x5cf0d9
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5cf10b` | ✅ |  |
| 1 | int32 | int32 `category @0x5cf116` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5cf122` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5cf12b` | ✅ |  |
| 4 | byte | byte `sortType @0x5cf136` | ✅ |  |
| 5 | byte | byte `sortColumn @0x5cf141` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x5cf174/@0x5cf150` | ✅ |  |
| 7 | string | string `searchCondition @0x5cf190` | ✅ |  |

