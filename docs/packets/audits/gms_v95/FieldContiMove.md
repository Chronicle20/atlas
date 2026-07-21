# FieldContiMove (← `CField_ContiMove::OnContiMove`)

- **IDA:** 0x54d680
- **Atlas file:** `libs/atlas-packet/field/clientbound/conti_move.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `state; (state-7) selects one of 6 arms (v95 uses a switch: case 8/0xA/0xC)` | ✅ |  |
| 1 | byte | byte `subState (state-gated: only cases 8/10/12 -- OnStartShipMoveField/OnMoveField/OnEndShipMoveField -- read a second byte; other cases hit default and read nothing further)` | ✅ |  |

