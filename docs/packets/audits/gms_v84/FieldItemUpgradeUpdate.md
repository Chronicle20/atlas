# FieldItemUpgradeUpdate (← `CUIItemUpgrade::Update`)

- **IDA:** 0x8562d1
- **Atlas file:** `libs/atlas-packet/field/serverbound/item_upgrade_update.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nReturnResult (a1[34]; open-arm mode byte, set by CUIItemUpgrade::OnItemUpgradeResult's Decode1, widened to int32)` | ✅ |  |
| 1 | int32 | int32 `m_nResult (a1[35]; server-chosen round-trip token, set by the decoder's open-arm Decode4)` | ✅ |  |

