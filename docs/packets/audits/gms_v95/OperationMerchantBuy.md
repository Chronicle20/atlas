# OperationMerchantBuy (← `CPersonalShopDlg::BuyItem#Merchant`)

- **IDA:** 0x69a7f0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_merchant_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `index (nIdx; op 0x22 entrusted)` | ✅ |  |
| 1 | int16 | int16 `quantity` | ✅ |  |
| 2 | byte | int32 `itemCRC (trailing field; not in atlas)` | ❌ | atlas: short — missing trailing field |


> defer: REAL ❌ — atlas missing trailing uint32 itemCRC (op 34 entrusted).
> Version-sensitive; no cross-version IDA. See `docs/packets/ida-exports/_pending.md` →
> "OperationPersonalStoreBuy / OperationMerchantBuy".
