# CashQueryResult (← `CCashShop::OnQueryCashResult`)

- **IDA:** 0x48b3e8
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/query_result.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nCash (credit)` | ✅ |  |
| 1 | int32 | int32 `nMaplePoint (points). JMS reads only 2 ints — atlas else-branch (no prepaid) matches` | ✅ |  |

