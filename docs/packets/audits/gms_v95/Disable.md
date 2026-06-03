# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0x905550
- **Atlas file:** `libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (enable flag; stored to CWvsContext+0xE15)` | ✅ |  |

## Manual analysis

**IDA function:** `CUserLocal::OnSetStandAloneMode` @ 0x905550

```
Decode1  → bStandAlone (enable/standalone mode flag; compared and stored at CWvsContext+0xE15)
```

Total: 1 byte.

**Atlas encoder (`ui/clientbound/disable.go`):**
```
WriteBool(enable)    → 1 byte (0x00 or 0x01)
```

**Wire comparison:**

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| bStandAlone / enable | 1 byte (Decode1) | 1 byte (WriteBool) | ✅ |

`WriteBool` emits a single byte (0 or 1); `Decode1` reads a single byte — wire-equivalent.

**SUMMARY row collision check:** Atlas file path resolves to `libs/atlas-packet/ui/clientbound/disable.go` — correctly points at `ui/`. No name collision with other domains.

### No bug — already correct

`Disable.Encode` matches v95 exactly. No fix needed. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestUiDisableWireShape` in `libs/atlas-packet/ui/clientbound/disable_test.go`: all four variants produce exactly 1 byte (0x01 for enable=true, 0x00 for enable=false).

Ack: misc-audit Phase 2d on 2026-06-03

