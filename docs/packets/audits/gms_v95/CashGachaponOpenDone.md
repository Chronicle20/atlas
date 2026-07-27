# CashGachaponOpenDone (← `CCashShop::OnCashItemResult#GACHAPON_OPEN_SUCCESS`)

- **IDA:** 0x494ac0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gachapon.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xb7 GACHAPON_OPEN_SUCCESS; op-byte consumed by dispatcher before CCashShop::OnCashItemResCashGachaponOpenDone)` | ✅ |  |
| 1 | int64 | bytes `sn: 8 bytes LARGE_INTEGER (int64), matched against existing m_aCashItemInfo[i].liSN` | ✅ |  |
| 2 | int32 | int32 `remain (int32, new quantity; 0 removes the locker entry)` | ✅ |  |
| 3 | byte | byte `isCashItem (byte flag)` | ✅ |  |
| 4 | bytes | bytes `CONDITIONAL if isCashItem != 0: newItem, 55 bytes GW_CashItemInfo blob, appended to m_aCashItemInfo` | ✅ |  |
| 5 | int32 | int32 `resultCode (int32, passed to CUICashGachapon::OnCashGachaponOpenResult)` | ✅ |  |
| 6 | byte | byte `resultParam2 (byte, second param to the same call)` | ✅ |  |

