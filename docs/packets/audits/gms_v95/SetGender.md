# SetGender (← `CLogin::SendSetGenderPacket`)

- **IDA:** 0x5d4650
- **Atlas file:** `libs/atlas-packet/account/serverbound/set_gender.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `set flag (literal 1u when setting gender)` | ✅ |  |
| 1 | byte | byte `nGender byte (only when set=1)` | ✅ |  |

## Manual analysis

**IDA functions:**
- `CLogin::SendSetGenderPacket` @ 0x5d4650 — sends gender selection
- `CLogin::SendCancelGenderPacket` @ 0x5d46e0 — sends gender cancel

`SendSetGenderPacket` takes `nGender (unsigned char)` and constructs opcode 8:

```
COutPacket::COutPacket(&oPacket, 8)   // opcode 8 → SET_GENDER
Encode1(1u)                            // set flag = true (literal 1)
Encode1(nGender)                       // gender byte (0=male, 1=female)
```

`SendCancelGenderPacket` constructs the same opcode 8 but cancel path:

```
COutPacket::COutPacket(&oPacket, 8)   // opcode 8 → SET_GENDER
Encode1(0)                             // set flag = false (literal 0)
// (no gender byte follows)
```

Total wire for "set" path: 2 bytes.
Total wire for "cancel" path: 1 byte.

### Dispatcher-offset finding

Same pattern as the other account-bucket packets: `COutPacket` is constructed with the opcode directly; no accountId/sessionId prefix is prepended to the payload. The atlas decoder reads from byte 0 of the payload. **No offset discrepancy.** Consistent with AcceptTos and RegisterPin.

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| set flag | 1 byte (Encode1, literal 1u or 0) | 1 byte (WriteBool) | ✅ |
| nGender | 1 byte (Encode1) | 1 byte (WriteByte) | ✅ |

### Atlas decoder (`account/serverbound/set_gender.go`)

```go
m.set = r.ReadBool()    // 1 byte
if m.set {
    m.gender = r.ReadByte() // 1 byte
}
```

Matches both IDA send paths exactly for opcode 8.

### No bug — already correct

`SetGender.Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/account/serverbound/set_gender.go` — correct account/ path. AcceptTos row unchanged ✅.

Ack: misc-audit Phase 2h on 2026-06-03
