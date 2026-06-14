# ItemCancel (← `CWvsContext::SendStatChangeItemCancelRequest`)

- **IDA:** 0xa096af
- **Atlas file:** `libs/atlas-packet/character/serverbound/item_cancel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nItemID (stat-change item ID to cancel)` | ✅ |  |

