# FieldMtsResultMoveItcPurchaseItemLtoSDone (← `CITC::OnNormalItemResult#MoveItcPurchaseItemLtoSDone`)

- **IDA:** 0x5a4d68
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x27 MoveItcPurchaseItemLtoSDone)` | ✅ |  |
| 1 | int32 | int32 `Decode4 tab (-> CCtrlTab::SetTab(tab-1))` | ✅ |  |
| 2 | int32 | int32 `Decode4 selectedNo` | ✅ |  |

