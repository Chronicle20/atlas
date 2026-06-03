# CashQueryResult (← `CCashShop::OnQueryCashResult`)

- **IDA:** 0x496400
- **Atlas file:** `libs/atlas-packet/cash/clientbound/query_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nNexonCash (credit)` | ✅ |  |
| 1 | int32 | int32 `nMaplePoint (points)` | ✅ |  |
| 2 | int32 | int32 `nPrepaidNXCash (prepaid; v95 reads unconditionally)` | ✅ |  |

