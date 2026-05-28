# InventoryScrollUse (← `CWvsContext::SendUpgradeItemUseRequest`)

- **IDA:** 0x9d6260
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scroll_use.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (line 22)` | ✅ |  |
| 1 | int16 | int16 `nUPOS scrollSlot (line 23)` | ✅ |  |
| 2 | int16 | int16 `nEPOS equipSlot (line 24)` | ✅ |  |
| 3 | int16 | int16 `bWhiteScroll (line 25)` | ✅ |  |
| 4 | byte | byte `bEnchantSkill legendarySpirit (line 26)` | ✅ |  |

