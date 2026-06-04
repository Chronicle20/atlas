# FieldAffectedAreaRemoved (← `CAffectedAreaPool::OnAffectedAreaRemoved`)

- **IDA:** 0x43234d
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/affected_area_removed.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id, v39)` | ✅ |  |

