# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x0
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `leading ask_SPW int — INFERRED from gift-family parity (OnBuyCouple/OnBuyFriendship/SendGiftsPacket all use a leading Encode4 int at v87, NOT EncodeStr). No discrete v87 rebate sender symbol recovered (OnRebateLockerItem not present in v87 export; built inline in a locker UI path). LOW CONFIDENCE: address unknown. SPW gate >=95 retained (consistent with the 3 confirmed family members).` | ✅ |  |
| 1 | int64 | bytes `8-byte locker serial (inferred identical to v83/v95)` | ❌ | width mismatch |

