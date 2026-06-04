# Lock (← `CUserLocal::OnSetDirectionMode`)

- **IDA:** 0x9054f0
- **Atlas file:** `libs/atlas-packet/ui/clientbound/lock.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bSet (direction/lock mode flag)` | ✅ |  |
| 1 | int32 | int32 `tAfterLeaveDirectionMode (timer value; used when bSet==0 and value>0)` | ✅ |  |

## Manual analysis

**IDA function:** `CUserLocal::OnSetDirectionMode` @ 0x9054f0

The CSV labels this opcode's FName as `CUserLocal::SetDirectionMode` (the internal helper @0x904240), but the actual packet handler is `OnSetDirectionMode`. The decoder is unambiguous:

```
Decode1  → bSet  (direction/lock flag; also triggers TryLeaveDirectionMode path)
Decode4  → tAfterLeaveDirectionMode  (timer; used when bSet==0 and value>0 to schedule deferred leave)
```

Total: 5 bytes. Both fields are decoded **unconditionally** — no version branch in the decompiled body.

**Atlas encoder (`ui/clientbound/lock.go`):**
```
WriteBool(enable)                                          → 1 byte
if GMS && MajorVersion >= 90 { WriteInt32(tAfterLeave) }  → 4 bytes (GMS v90+)
```

**Wire comparison under GMS v95:**

| Field | IDA width | Atlas width (v95) | Match? |
|---|---|---|---|
| bSet / enable | 1 byte (Decode1) | 1 byte (WriteBool) | ✅ |
| tAfterLeaveDirectionMode | 4 bytes (Decode4) | 4 bytes (WriteInt32, gate fires) | ✅ |

The gate `GMS && MajorVersion >= 90` fires for v95, so the int32 is written. The static-diff tool computes branch depth = 2 (two conditions in the guard), matching the `&&`-compound guard — normal.

**SUMMARY row collision check:** Atlas file path resolves to `libs/atlas-packet/ui/clientbound/lock.go` — correctly points at `ui/`. No name collision with other domains.

### No bug — already correct for v95

`Lock.Encode` matches v95 exactly. No fix needed. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestUiLockWireShape` in `libs/atlas-packet/ui/clientbound/lock_test.go`: GMS v95 produces 5 bytes (1 flag + 4 timer); non-GMS and pre-v90 produce 1 byte.

### Note on non-GMS / pre-v90 versions

The v95 IDA decoder always reads both fields. The `>= 90` gate means GMS v83 (and earlier) and non-GMS clients only receive 1 byte. If such clients also decode the int32, those paths would desync. This is pre-existing behavior assumed correct by the atlas authors; no v83 IDA evidence was available in this database to confirm or deny. Recorded as a known open item for Phase 3.

Ack: misc-audit Phase 2d on 2026-06-03

