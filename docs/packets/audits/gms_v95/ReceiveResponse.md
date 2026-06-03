# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 0x9fea60
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 5 = RECEIVE)` | ✅ |  |
| 1 | string | string `fromName (character who gave fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |

## Manual analysis

**IDA function:** `CWvsContext::OnGivePopularityResult` @ 0x9fea60, case 5

```
Decode1  → mode (switch; case 5 = RECEIVE)
DecodeStr → fromName (ZXString, 2-byte LE length + ShiftJIS bytes)
Decode1  → bInc (fame=1, defame=0)
```

Total after opcode: 1 (mode) + (2+len) (fromName) + 1 (bInc) bytes.

### Atlas encoder (`fame/clientbound/response.go`)

```
WriteByte(mode)            → 1 byte   (mode=5 / RECEIVE)
WriteAsciiString(fromName) → 2+len bytes
WriteInt8(fameMode)        → 1 byte   (fameMode = (amount+1)/2; 0=defame, 1=fame)
```

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| mode | 1 byte (Decode1) | 1 byte (WriteByte) | ✅ |
| fromName | 2+len bytes (DecodeStr) | 2+len bytes (WriteAsciiString) | ✅ |
| bInc | 1 byte (Decode1) | 1 byte (WriteInt8) | ✅ |

**SUMMARY row collision check:** Atlas file path resolves to
`libs/atlas-packet/fame/clientbound/response.go` — correctly points at `fame/`.

### No bug — already correct

`ReceiveResponse.Encode` matches v95 exactly. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestReceiveFameResponseWireShape` in
`libs/atlas-packet/fame/clientbound/response_test.go`:
all four variants produce exactly 6 bytes for a 2-character name `"P1"`, with
byte[0]=0x05 (mode), bytes[1-2]=LE length=2, bytes[3-4]='P','1', byte[5]=0x01 (inc).

Ack: misc-audit Phase 2e on 2026-06-03
