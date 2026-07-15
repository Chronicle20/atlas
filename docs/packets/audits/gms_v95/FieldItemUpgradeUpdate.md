# FieldItemUpgradeUpdate (← `CUIItemUpgrade::Update`)

- **IDA:** 0x7bef50
- **Atlas file:** `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nReturnResult (open-arm mode byte, set by OnItemUpgradeResult's Decode1, widened to int32)` | ✅ |  |
| 1 | int32 | int32 `m_nResult (server-chosen round-trip token, set by OnItemUpgradeResult's open-arm Decode4)` | ✅ |  |

