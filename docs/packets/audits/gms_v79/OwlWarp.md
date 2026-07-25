# OwlWarp (← `CUIShopScanResult::OnButtonClicked`)

- **IDA:** 0x80ce8a
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/owl_warp.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `ownerId (dwMiniRoomSN; ITEMDATA+16) - owl-warp button send, COutPacket owl-warp opcode` | ✅ |  |
| 1 | int32 | int32 `mapId (dwFieldID; ITEMDATA+20)` | ✅ |  |

