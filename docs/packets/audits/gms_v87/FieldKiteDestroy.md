# FieldKiteDestroy (← `CMessageBoxPool::OnMessageBoxLeaveField`)

- **IDA:** 0x69544f
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/kite_destroy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bAnimation (leave animation type; 0 = play despawn animation, v1 @0x69546f)` | ✅ |  |
| 1 | int32 | int32 `dwID (kite object id, v36 @0x69547b)` | ✅ |  |

