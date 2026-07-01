# InventoryScrollUse (← `CWvsContext::SendUpgradeItemUseRequest`)

- **IDA:** 0x954f9b
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scroll_use.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime — sub_954F9B COutPacket(84) Encode4 get_update_time @0x954fd6` | ✅ |  |
| 1 | int16 | int16 `scrollSlot (nUOL/nUpgradeItemPos, a2) — Encode2 @0x954fe1` | ✅ |  |
| 2 | int16 | int16 `equipSlot (nEquipPos, a3) — Encode2 @0x954fec` | ✅ |  |
| 3 | int16 | int16 `bWhiteScroll (a4) — Encode2 @0x954ff7` | ✅ |  |
| 4 | byte | byte `legendarySpirit (bLegendarySpirit, a5) — Encode1 @0x955002` | ✅ |  |

