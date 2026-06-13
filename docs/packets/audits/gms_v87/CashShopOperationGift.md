# CashShopOperationGift (← `CCashShop::SendGiftsPacket`)

- **IDA:** 0x47a168
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_gift.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | int32 `leading int (*(this+1308)). v87 is a 4-byte INT (line 147), NOT EncodeStr sSPW. The leading SPW string is v95-only; SPW gate >=95 CORRECT.` | ✅ |  |
| 2 | byte | int32 `serialNumber (*(this+1312))` | ❌ | width mismatch |
| 3 | string | byte `m_bRequestBuyOneADay byte (*(this+9928)). PRESENT at v87 (line 149) before name — NOT v95-only. oneADay gate tightened to GMS>=87 (split from the v95-only SPW gate).` | ❌ | width mismatch |
| 4 | string | string `recipient name (v33)` | ✅ |  |
| 5 | byte | string `message (*(this+1316))` | ❌ | atlas: short — missing trailing field |

