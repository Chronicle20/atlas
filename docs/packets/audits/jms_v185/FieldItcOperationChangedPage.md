# FieldItcOperationChangedPage (← `CITC::OnChangedPage`)

- **IDA:** 0x60498c
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (5) @0x6049bc` | ✅ |  |
| 1 | int32 | int32 `category (this+26) @0x6049c7` | ✅ |  |
| 2 | int32 | int32 `categorySub (this+27) @0x6049d2` | ✅ |  |
| 3 | int32 | int32 `page (nPage) @0x6049dd` | ✅ |  |
| 4 | byte | byte `sortType (this+96) @0x6049eb` | ✅ |  |
| 5 | byte | byte `sortColumn (this+100) @0x6049f9` | ✅ |  |
| 6 | int32 | int32 `searchOption (this+2920) @0x604a07` | ✅ |  |
| 7 | string | string `searchCondition (this+2921) @0x604a24` | ✅ |  |

