# FieldItemUpgradeUpdate (← `CUIItemUpgrade::Update`)

- **IDA:** 0x88eea2
- **Atlas file:** `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nReturnResult (this[38]; open-arm mode byte, set by sub_88F348's Decode1, widened to int32)` | ✅ |  |
| 1 | int32 | int32 `m_nResult (this[39]; server-chosen round-trip token, set by sub_88F348's open-arm Decode4)` | ✅ |  |

