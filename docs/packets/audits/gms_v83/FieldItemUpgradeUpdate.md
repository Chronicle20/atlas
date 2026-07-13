# FieldItemUpgradeUpdate (← `CUIItemUpgrade::Update`)

- **IDA:** 0x82ae28
- **Atlas file:** `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nReturnResult (v3=this[34]; open-arm mode byte, set by sub_82B2C3's Decode1, widened to int32)` | ✅ |  |
| 1 | int32 | int32 `m_nResult (this[35]; server-chosen round-trip token, set by sub_82B2C3's open-arm Decode4)` | ✅ |  |

