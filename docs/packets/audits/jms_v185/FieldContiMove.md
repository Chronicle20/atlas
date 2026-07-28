# FieldContiMove (← `CField_ContiMove::OnContiMove`)

- **IDA:** 0x58e21b
- **Atlas file:** `libs/atlas-packet/field/clientbound/conti_move.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `state; (state-7) selects one of 6 arms` | ✅ |  |
| 1 | byte | byte `subState (state-gated: only arms 8/10/12 -- OnStartShipMoveField/OnMoveField/OnEndShipMoveField -- read a second byte; arms 7/9/11 are nullsubs and read nothing further)` | ✅ |  |

