# ScriptProgress (← `CWvsContext::OnScriptProgressMessage`)

- **IDA:** 0x9e5110
- **Atlas file:** `libs/atlas-packet/quest/clientbound/script_progress.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (quest script progress string)` | ✅ |  |

## Manual analysis

**IDA function:** `CWvsContext::OnScriptProgressMessage` @ 0x9e5110

The client decoder reads a single length-prefixed ASCII string:

```
DecodeStr(iPacket, &message)   // length-prefixed ASCII string
```

Total: 2 (uint16 length) + N (message bytes).

### Atlas encoder (`quest/clientbound/script_progress.go`)

```
WriteAsciiString(message)  → uint16 length + bytes
```

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| message | DecodeStr (uint16-len + bytes) | WriteAsciiString (uint16-len + bytes) | ✅ |

### No bug — already correct

`ScriptProgress.Encode/Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestScriptProgressWireShape` in
`libs/atlas-packet/quest/clientbound/script_progress_test.go`:
all four variants produce exactly 2+N bytes (length prefix + message).

Ack: misc-audit Phase 2g on 2026-06-03

