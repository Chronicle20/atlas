# FieldItcOperationViewWish (← `CITC::OnViewWish`)

- **IDA:** 0x57afbe
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0xB) @0x59fa4d` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x59fa5e` | ✅ |  |

