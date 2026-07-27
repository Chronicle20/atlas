# OwlWarp (← `CUIShopScanResult::OnButtonClicked`)

- **IDA:** 0x848e80
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/owl_warp.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `ownerId (dwMiniRoomSN; ITEMDATA+16) when row's *(+44)==0. COutPacket(0x49); buttons 3000-3007 = result rows` | ✅ |  |
| 1 | int32 | int32 `mapId (dwFieldID; ITEMDATA+20)` | ✅ |  |

