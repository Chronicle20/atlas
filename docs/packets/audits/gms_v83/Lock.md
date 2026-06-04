# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0x95ff5a
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock mode flag) — v83 reads ONLY 1 byte; tAfterLeaveDirectionMode int32 absent (confirmed: v83 SetDirectionMode @0x95ff5a)` | ✅ |  |

