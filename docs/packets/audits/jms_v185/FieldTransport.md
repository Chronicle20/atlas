# FieldTransport (← `CField_ContiMove::OnContiState`)

- **IDA:** 0x58e280
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/transport.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nState (transport/ship state 0-6)` | ✅ |  |
| 1 | byte | byte `overrideAppear flag (v4; checked v4==1 in AppearShip case)` | ✅ |  |

