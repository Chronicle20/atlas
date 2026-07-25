# FieldItcOperationMoveLtoS (← `CITC::OnMoveITCPurchaseItemLtoS`)

- **IDA:** 0x57b1c0
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (8) @0x59fc4f` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59fc60` | ✅ |  |

