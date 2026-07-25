# CashItemUsePointReset (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa0a63f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_point_reset.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `to: AP stat flag (5050000) / SP skill id (5050001-4). Encode4 @case23/24 in CWvsContext::SendConsumeCashItemUseRequest 0xa0a63f` | ✅ |  |
| 1 | int32 | int32 `from: AP stat flag / SP skill id. second Encode4` | ✅ |  |
| 2 | int32 | int32 `trailing update_time = get_update_time(); Encode4 in common send tail (LABEL_41)` | ✅ |  |

