# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0x9e312a
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock mode flag) — v87 reads ONLY 1 byte; tAfterLeaveDirectionMode int32 absent (gate >=90)` | ✅ |  |

