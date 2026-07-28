# FieldItcOperationMoveLtoS (← `CITC::OnMoveITCPurchaseItemLtoS`)

- **IDA:** 0x605175
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (8) @0x6051a1` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x6051b2` | ✅ |  |

