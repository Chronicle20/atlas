# FieldTransport (← `CField_ContiMove::OnContiState`)

- **IDA:** 0x54d5a0
- **Atlas file:** `libs/atlas-packet/field/clientbound/transport.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nState (transport/ship state 0-6)` | ✅ |  |
| 1 | byte | byte `overrideAppear flag (v4; checked v4==1 in AppearShip case)` | ✅ |  |


## Audit notes

Full audit (no CharacterData, no version guards). `CField_ContiMove::OnContiState` @0x54d5a0 (CONTI_STATE; GMS v95 opcode 0xA5/165, GMS v83 0x95/149) reads exactly two bytes: a state byte (switched — cases 0/1/6 → EnterShipMove, 2/5 → LeaveShipMove, 3/4 → AppearShip) and a second flag byte `v4` (checked `v4 == 1` in the AppearShip case). Atlas `Transport.Encode` writes `WriteByte(state)` + `WriteBool(overrideAppear)` — exact match. The atlas TransportState enum (0-6: Enter1/Enter2/Move1/Appear1/Appear2/Move2/Enter3) covers the full switch range. ✅

Ack: world-audit Phase 2c on 2026-05-28
