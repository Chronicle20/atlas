# CashNameChangeBuyDone (← `CCashShop::OnCashItemResult#NAME_CHANGE_BUY_DONE`)

- **IDA:** 0x474bc6
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x96 NAME_CHANGE_BUY_DONE; op-byte consumed by dispatcher before OnCashItemNameChangeResBuyDone)` | ✅ |  |
| 1 | bytes | bytes `GW_CashItemInfo blob (55B, single; decoded into a freshly-inserted m_aCashItemInfo slot)` | ✅ |  |

