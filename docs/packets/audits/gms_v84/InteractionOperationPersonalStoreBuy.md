# InteractionOperationPersonalStoreBuy (← `CPersonalShopDlg::BuyItem`)

- **IDA:** 0x71951e
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_buy.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int16 | byte `` | ❌ | width mismatch |
| 2 | int32 | int16 `` | ❌ | width mismatch |
| 3 | byte | int32 `` | ❌ | atlas: short — missing trailing field |

