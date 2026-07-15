# FieldItcOperationChangedCategory (← `CITC::OnChangedCategory`)

- **IDA:** 0x6047bf
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x604816` | ✅ |  |
| 1 | int32 | int32 `category @0x604821` | ✅ |  |
| 2 | int32 | int32 `categorySub (const 0) @0x60482a` | ✅ |  |
| 3 | int32 | int32 `page (const 0) @0x604833` | ✅ |  |
| 4 | byte | byte `sortType (const 1) @0x60483c` | ✅ |  |
| 5 | byte | byte `sortColumn (const 1) @0x604845` | ✅ |  |
| 6 | int32 | int32 `searchOption (const 1) @0x60484e` | ✅ |  |
| 7 | string | string `searchCondition (const empty) @0x604866` | ✅ |  |

