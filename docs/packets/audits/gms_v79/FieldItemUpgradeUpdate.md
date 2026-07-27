# FieldItemUpgradeUpdate (← `CUIItemUpgrade::Update`)

- **IDA:** 0x7998da
- **Atlas file:** `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nReturnResult (v3=this[32]; open-arm mode byte, set by OnItemUpgradeResult's Decode1, widened to int32)` | ✅ |  |
| 1 | int32 | int32 `m_nResult (this[33]; server-chosen round-trip token, set by OnItemUpgradeResult's open-arm Decode4)` | ✅ |  |

