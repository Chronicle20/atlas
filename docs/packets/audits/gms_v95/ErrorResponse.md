# ErrorResponse (← `CWvsContext::OnGivePopularityResult#ErrorResponse`)

- **IDA:** 0x9fea60
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; cases 1-4 = error codes — no additional fields)` | ✅ |  |

## Manual analysis

**IDA function:** `CWvsContext::OnGivePopularityResult` @ 0x9fea60, cases 1, 2, 3, 4, and default

Each error case reads only the mode byte from the switch dispatch; no additional
fields follow. The client looks up a localized string (e.g., "Invalid name",
"Not minimum level", "Not today", "Not this month") and displays it as a chat log
message.

```
Decode1 → mode (cases 1-4 = error; no further Decode calls in those branches)
```

Total after opcode: 1 byte (mode only).

### Atlas encoder (`fame/clientbound/response.go`)

```
WriteByte(mode)  → 1 byte
```

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| mode | 1 byte (Decode1) | 1 byte (WriteByte) | ✅ |

**SUMMARY row collision check:** Atlas file path resolves to
`libs/atlas-packet/fame/clientbound/response.go` — correctly points at `fame/`.

### No bug — already correct

`ErrorResponse.Encode` matches v95 exactly. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestFameErrorResponseWireShape` in
`libs/atlas-packet/fame/clientbound/response_test.go`:
all four variants produce exactly 1 byte.

Ack: misc-audit Phase 2e on 2026-06-03
