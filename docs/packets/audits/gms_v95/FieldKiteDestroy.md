# FieldKiteDestroy (← `CMessageBoxPool::OnMessageBoxLeaveField`)

- **IDA:** 0x635d60
- **Atlas file:** `libs/atlas-packet/field/clientbound/kite_destroy.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bAnimation (leave animation type; 0 = play despawn animation)` | ✅ |  |
| 1 | int32 | int32 `dwID (kite object id)` | ✅ |  |


Ack: world-audit Phase 2c on 2026-05-28
