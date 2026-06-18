# FieldMtsResultSuccessBidInfo (← `CITC::OnNormalItemResult#SuccessBidInfo`)

- **IDA:** 0x5a52de
- **Atlas file:** `libs/atlas-packet/field/clientbound/mts_operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `MTS result mode byte (0x3E SuccessBidInfoResult)` | ✅ |  |
| 1 | byte | byte `sold(1)/bought flag -> StringPool 0x12AA/0x12AB` | ✅ |  |
| 2 | int32 | int32 `ITC item id; <=0 ends the body` | ✅ |  |
| 3 | int32 | int32 `meso price (itemId>0 branch only)` | ✅ |  |
| 4 | bytes | bytes `8-byte FILETIME contract date (itemId>0 branch only)` | ✅ |  |

