# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x557b5a
- **Atlas file:** `libs/atlas-packet/field/serverbound/change.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (v8 = get_field()->m_bFieldKey, @0x557bbc)` | ✅ |  |
| 1 | int32 | int32 `targetId (arg0 dwTargetField, @0x557bc7)` | ✅ |  |
| 2 | string | string `portalName (sPortal; empty ZXString when arg4==NULL, @0x557bec)` | ✅ |  |
| 3 | int16 | int16 `x (target position X, @0x557c0e)` | ✅ |  |
| 4 | int16 | int16 `y (target position Y, @0x557c2c)` | ✅ |  |
| 5 | byte | byte `unused (constant 0, @0x557c35)` | ✅ |  |
| 6 | byte | byte `premium (a2 bPremium, @0x557c40)` | ✅ |  |
| 7 | byte | byte `chase (global dword_CA02A0, @0x557c50)` | ✅ |  |
| 8 | int32 | int32 `targetX (a5, @0x557c63)` | ✅ |  |
| 9 | int32 | int32 `targetY (a6, @0x557c6e)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
