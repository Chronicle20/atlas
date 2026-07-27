# FieldItcOperationChangedPage (← `CITC::OnChangedPage`)

- **IDA:** 0x5af7c8
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x5af7f8` | ✅ |  |
| 1 | int32 | int32 `category @0x5af803` | ✅ |  |
| 2 | int32 | int32 `categorySub @0x5af80e` | ✅ |  |
| 3 | int32 | int32 `page @0x5af819` | ✅ |  |
| 4 | byte | byte `sortType @0x5af827` | ✅ |  |
| 5 | byte | byte `sortColumn @0x5af835` | ✅ |  |
| 6 | int32 | int32 `searchOption @0x5af843` | ✅ |  |
| 7 | string | string `searchCondition @0x5af860` | ✅ |  |

