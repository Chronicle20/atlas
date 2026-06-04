# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x9055f0
- **Atlas file:** `libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (window mode byte; dispatched to CWvsContext::UI_Open)` | ✅ |  |

## Manual analysis

**IDA function:** `CUserLocal::OnOpenUI` @ 0x9055f0

```
Decode1  → nUIID / window-mode byte (passed to CWvsContext::UI_Open)
```

Total: 1 byte.

**Atlas encoder (`ui/clientbound/open.go`):**
```
WriteByte(windowMode)    → 1 byte
```

**Wire comparison:**

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| windowMode / nUIID | 1 byte (Decode1) | 1 byte (WriteByte) | ✅ |

**SUMMARY row collision check:** Atlas file path resolves to `libs/atlas-packet/ui/clientbound/open.go` — correctly points at `ui/`. No name collision with other domains.

### No bug — already correct

`Open.Encode` matches v95 exactly. No fix needed. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestUiOpenWireShape` in `libs/atlas-packet/ui/clientbound/open_test.go`: all four variants produce exactly 1 byte equal to the windowMode value.

Ack: misc-audit Phase 2d on 2026-06-03

