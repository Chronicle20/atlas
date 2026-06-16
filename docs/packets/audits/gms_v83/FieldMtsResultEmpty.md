# FieldMtsResultEmpty (← `CITC::OnNormalItemResult#Empty`)

- **IDA:** 0x5a4674
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation_body.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (switch discriminator; e.g. 0x1D RegisterSaleEntryDone)` | ✅ |  |

