# CashGachaponCopyDone (← `CCashShop::OnCashItemResult#GACHAPON_COPY_SUCCESS`)

- **IDA:** 0x47fae2
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gachapon.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xa7 GACHAPON_COPY_SUCCESS; op-byte consumed by dispatcher before CCashShop::OnCashItemResCashGachaponCopyDone)` | ✅ |  |
| 1 | byte | byte `flag1 (byte)` | ✅ |  |
| 2 | byte | byte `flag2 (byte)` | ✅ |  |
| 3 | int32 | int32 `unused1 (int32, CInPacket::Decode4 result discarded, no assignment)` | ✅ |  |
| 4 | int32 | int32 `unused2 (int32, same, discarded)` | ✅ |  |
| 5 | int32 | int32 `lostItemId (int32, nRandomItemLostItemID)` | ✅ |  |
| 6 | int32 | int32 `lostNumber (int32, nRandomItemLostNumber)` | ✅ |  |
| 7 | bytes | bytes `CONDITIONAL if flag1 != 0 AND flag2 != 0: item, 55 bytes GW_CashItemInfo blob, appended to m_aCashItemInfo` | ✅ |  |

