# ItemCancel (← `CWvsContext::SendStatChangeItemCancelRequest`)

- **IDA:** 0x904088
- **Atlas file:** `libs/atlas-packet/character/serverbound/item_cancel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `sourceId Encode4 @0x904126` | ✅ |  |

