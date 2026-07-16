# OwlAction (← `CUIShopScanner::OnCreate`)

- **IDA:** 0x8a0e9a
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/owl_action.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=5; request hot list on scanner open). COutPacket(0x42)` | ✅ |  |

