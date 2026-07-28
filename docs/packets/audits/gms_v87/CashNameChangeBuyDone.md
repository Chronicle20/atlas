# CashNameChangeBuyDone (← `CCashShop::OnCashItemResult#NAME_CHANGE_BUY_DONE`)

- **IDA:** 0x4874ac
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_transfer.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa7 NAME_CHANGE_BUY_DONE; op-byte consumed by dispatcher before CCashShop::OnCashItemNameChangeResBuyDone)` | ✅ |  |
| 1 | bytes | bytes `item: 55 bytes GW_CashItemInfo blob (CashInventoryItem.EncodeBytes), decoded into a freshly-inserted m_aCashItemInfo slot` | ✅ |  |

