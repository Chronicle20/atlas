# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x53035d
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()+308)` | ✅ |  |
| 1 | int32 | int32 `targetId (a1)` | ✅ |  |
| 2 | string | string `portalName (arg4; empty when NULL)` | ✅ |  |
| 3 | int16 | int16 `x (target position X; only when portal name present)` | ✅ |  |
| 4 | int16 | int16 `y (target position Y; only when portal name present)` | ✅ |  |
| 5 | byte | byte `unused (constant 0)` | ✅ |  |
| 6 | byte | byte `premium (a2)` | ✅ |  |
| 7 | byte | byte `chase (dword_BED760)` | ✅ |  |
| 8 | int32 | int32 `targetX (a4; only when chase!=0)` | ✅ |  |
| 9 | int32 | int32 `targetY (a5; only when chase!=0)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
