# CashTransferWorldNoticeReason (← `CCashShop::OnCashItemResult#TRANSFER_WORLD_NOTICE_REASON`)

- **IDA:** 0x48e6f7
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_jms.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x93=147 TRANSFER_WORLD_NOTICE_REASON; op-byte consumed by dispatcher before CCashShop__OnCashItemResTransferWorldNoticeReason)` | ✅ |  |
| 1 | byte | byte `reason byte (v3; NoticeFailReason code; 177/178 trigger SendTransferFieldPacket)` | ✅ |  |

