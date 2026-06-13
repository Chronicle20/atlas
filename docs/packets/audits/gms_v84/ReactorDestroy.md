# ReactorDestroy (← `CReactorPool::OnReactorLeaveField`)

- **IDA:** 0x752b14
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/destroy.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |

