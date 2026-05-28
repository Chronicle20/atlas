# FieldKiteDestroy (← `CMessageBoxPool::OnMessageBoxLeaveField`)

- **IDA:** 0x65b2ca
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/kite_destroy.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bAnimation (leave animation type; 0=play despawn animation, v3)` | ✅ |  |
| 1 | int32 | int32 `dwID (kite object id, v38)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28
