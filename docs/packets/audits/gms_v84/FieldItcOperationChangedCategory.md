# FieldItcOperationChangedCategory (← `CITC::OnChangedCategory`)

- **IDA:** 0x5af5fa
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5af64f` | ✅ |  |
| 1 | int32 | int32 `category @0x5af65a` | ✅ |  |
| 2 | int32 | int32 `categorySub (const 0) @0x5af663` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x5af66c` | ✅ |  |
| 4 | byte | byte `sortType (const 1) @0x5af678` | ✅ |  |
| 5 | byte | byte `sortColumn (const 1) @0x5af681` | ✅ |  |
| 6 | int32 | int32 `searchOption (const 1) @0x5af68a` | ✅ |  |
| 7 | string | string `searchCondition (const empty) @0x5af6a2` | ✅ |  |

