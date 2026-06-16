# FieldMtsResultTwoInts (← `CITC::OnNormalItemResult#TwoInts`)

- **IDA:** 0x5d4e58
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation_body.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (switch discriminator; e.g. 0x27 MoveITCPurchaseItemLtoSDone)` | ✅ |  |
| 1 | int32 | int32 `first int (e.g. tab index +1 / cancel count d)` | ✅ |  |
| 2 | int32 | int32 `second int (e.g. selectedNo / cancel count x)` | ✅ |  |

