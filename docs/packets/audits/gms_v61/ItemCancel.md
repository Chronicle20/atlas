# ItemCancel (← `CWvsContext::SendStatChangeItemCancelRequest`)

- **IDA:** 0x831bb7
- **Atlas file:** `libs/atlas-packet/character/serverbound/item_cancel.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `sourceId Encode4(a1) @0x831c55 (COutPacket(68) ctor @0x831c43)` | ✅ |  |

