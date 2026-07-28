# FieldItcOperationChangedCategory (← `CITC::OnChangedCategory`)

- **IDA:** 0x5744a0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5 GetItcList browse) @0x57452d` | ✅ |  |
| 1 | int32 | int32 `category @0x57453b` | ✅ |  |
| 2 | int32 | int32 `categorySub (const 0) @0x574546` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x574551` | ✅ |  |
| 4 | byte | byte `sortType (const 1) @0x57455c` | ✅ |  |
| 5 | byte | byte `sortColumn (const 1) @0x574567` | ✅ |  |
| 6 | int32 | int32 `searchOption (const 1) @0x574572` | ✅ |  |
| 7 | string | string `searchCondition (const empty) @0x5745ac` | ✅ |  |

