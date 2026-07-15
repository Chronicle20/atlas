# OwlAction (← `CUIShopScanner::OnCreate`)

- **IDA:** 0x848b90
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/owl_action.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=5; request hot list on scanner open). COutPacket(0x48)` | ✅ |  |

