# CashLimitGoodsStockChanged (← `CCashShop::OnCashItemResult#LIMIT_GOODS_STOCK_CHANGED`)

- **IDA:** 0x48d4d4
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x60=96 LIMIT_GOODS_STOCK_CHANGED; op-byte consumed by dispatcher before CCashShop__OnCashItemResLimitGoodsStockChanged)` | ✅ |  |
| 1 | byte | byte `result byte (v4; NoticeFailReason code; 205/206 gate the itemId read; 177/178/179 trigger SendTransferFieldPacket)` | ✅ |  |
| 2 | int32 | int32 `CONDITIONAL if result==205 or result==206: itemId (uint32, passed to UpdateStock/ChangeLimitGoodsState)` | ✅ |  |

