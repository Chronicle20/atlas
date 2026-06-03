# FieldAffectedAreaRemoved (← `CAffectedAreaPool::OnAffectedAreaRemoved`)

- **IDA:** 0x4360a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/affected_area_removed.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwId (mist/affected-area object id)` | ✅ |  |


Ack: world-audit Phase 2c on 2026-05-28
