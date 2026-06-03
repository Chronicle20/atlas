# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x5345c0
- **Atlas file:** `libs/atlas-packet/field/serverbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (v9 = get_field()->m_bFieldKey)` | ✅ |  |
| 1 | int32 | int32 `targetId (dwTargetField)` | ✅ |  |
| 2 | string | string `portalName (sPortal; empty ZXString when sPortal==NULL)` | ✅ |  |
| 3 | int16 | int16 `x (target position X)` | ✅ |  |
| 4 | int16 | int16 `y (target position Y)` | ✅ |  |
| 5 | byte | byte `unused (constant 0)` | ✅ |  |
| 6 | byte | byte `premium (bPremium)` | ✅ |  |
| 7 | byte | byte `chase (global s_bChase)` | ✅ |  |
| 8 | int32 | int32 `targetX (nTargetPosition_X)` | ✅ |  |
| 9 | int32 | int32 `targetY (nTargetPosition_Y)` | ✅ |  |


Ack: world-audit Phase 2b on 2026-05-28
