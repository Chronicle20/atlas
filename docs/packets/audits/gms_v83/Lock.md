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


## Manual analysis

**v83 IDA:** `CUserLocal::SetDirectionMode` @ 0x95ff5a — reads **ONLY ONE** byte (`Decode1(bSet)`). The `tAfterLeaveDirectionMode` int32 is **absent in v83**. (Note: the function is named `SetDirectionMode` not `OnSetDirectionMode` in v83; it IS the packet handler per the CSV opcode mapping.)

**v83 vs v95 gate:** v95 @ 0x9054f0 reads both `Decode1(bSet)` and `Decode4(tAfterLeaveDirectionMode)`. The atlas gate `GMS && MajorVersion >= 90` is **CONFIRMED CORRECT** — v83 (major version < 90) receives only 1 byte ✅. The ✅ auto-verdict confirms the gate fires correctly under GMS v83.


Ack: misc-audit Phase 3 v83 on 2026-06-03
