# FieldKiteDestroy (← `CMessageBoxPool::OnMessageBoxLeaveField`)

- **IDA:** 0x6d5f7f
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/kite_destroy.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `animationType (@line48)` | ✅ |  |
| 1 | int32 | int32 `id (message-box object id @line50)` | ✅ |  |

