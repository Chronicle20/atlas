# FieldTransport (← `CField_ContiMove::OnContiState`)

- **IDA:** 0x577c21
- **Atlas file:** `libs/atlas-packet/field/clientbound/transport.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nState (transport/ship state 0-6, v3 @0x577c32)` | ✅ |  |
| 1 | byte | byte `overrideAppear flag (v4; checked v4==1 in AppearShip case, @0x577c35)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
