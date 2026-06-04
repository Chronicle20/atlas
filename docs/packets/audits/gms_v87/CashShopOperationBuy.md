# CashShopOperationBuy (← `CCashShop::OnBuy`)

- **IDA:** 0x477bd9
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (v48==2)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `dwOption (v48)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `nCommSN (a2 serialNumber)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `m_bRequestBuyOneADay byte (*(this+9928)). PRESENT at v87 (line 443) — NOT v95-only. Gate tightened to GMS>=87.` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `nEventSN int (v44). PRESENT at v87 (line 444) — NOT v95-only. Gate tightened to GMS>=87.` | ❌ | atlas: short — missing trailing field |

