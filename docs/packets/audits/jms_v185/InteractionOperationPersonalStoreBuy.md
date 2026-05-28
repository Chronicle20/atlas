# InteractionOperationPersonalStoreBuy (← `CPersonalShopDlg::BuyItem`)

- **IDA:** 0x762365
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nIdx (index). op-byte 0x14 personal (or 0x1F entrusted) consumed by dispatcher` | ✅ |  |
| 1 | int16 | int16 `Src (quantity)` | ✅ |  |
| 2 | int32 | int32 `ItemCRC (CItemInfo::GetItemCRC). PRESENT in JMS — atlas unconditional itemCRC fix is correct` | ✅ |  |

