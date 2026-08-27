# CashShopOperationBuyOtherPackage (← `CCashShop::OnGiftPackage`)

- **IDA:** 0x4907b0
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (second password, from ask_SPW); COutPacket::EncodeStr @0x490b93` | ✅ |  |
| 1 | int32 | int32 `nCommSN (commodity serial number); inlined 4-byte write into COutPacket::m_aSendBuff (v45 is oPacket's own m_aSendBuff field, offset 0x4 per COutPacket's type layout — Hex-Rays split it into a separate local because it lost type info at this call site) @0x490be2, not a named EncodeStr/Encode4 call` | ✅ |  |
| 2 | string | string `sGiveTo (recipient character name, from CUISendGift::GetResult); COutPacket::EncodeStr @0x490c01` | ✅ |  |
| 3 | string | string `sText (gift message, from CUISendGift::GetResult); COutPacket::EncodeStr @0x490c1d` | ✅ |  |

