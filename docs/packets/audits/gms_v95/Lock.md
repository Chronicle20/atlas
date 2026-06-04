# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0x9054f0
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock mode flag)` | ✅ |  |
| 1 | int32 | int32 `tAfterLeaveDirectionMode (timer value; used when bSet==0 and value>0)` | ✅ |  |

