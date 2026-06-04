# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0x9e312a
- **Atlas file:** `libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock mode flag) — v87 reads ONLY 1 byte; tAfterLeaveDirectionMode int32 absent (gate >=90)` | ✅ |  |


## Manual analysis

**v87 IDA:** `CUserLocal::SetDirectionMode` @ 0x9e312a — reads **ONLY ONE** byte (`Decode1(a1)`). The `tAfterLeaveDirectionMode` int32 is **absent in v87**. (Named `SetDirectionMode` not `OnSetDirectionMode` in v87, same as v83; it IS the packet handler per CSV opcode mapping.)

**v87 vs v95 gate:** v95 @ 0x9054f0 reads both `Decode1(bSet)` and `Decode4(tAfterLeaveDirectionMode)`. The atlas gate `GMS && MajorVersion >= 90` is **CONFIRMED CORRECT** — v87 (major version < 90) receives only 1 byte ✅. v87 mirrors v83 exactly.

Ack: misc-audit Phase 3 v87 on 2026-06-03
