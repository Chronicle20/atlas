# FieldAffectedAreaRemoved (← `CAffectedAreaPool::OnAffectedAreaRemoved`)

- **IDA:** 0x436eda
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_removed.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (affected-area object id @line54)` | ✅ |  |

