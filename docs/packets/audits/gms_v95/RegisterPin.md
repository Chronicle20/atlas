# RegisterPin (← `CLogin::OnCheckPinCodeResult#RegisterPin`)

- **IDA:** 0x5db000
- **Atlas file:** `libs/atlas-packet/account/serverbound/register_pin.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `pinInput flag (1=pin provided, 0=cancelled)` | ✅ |  |
| 1 | string | string `pin string (only when pinInput=1)` | ✅ |  |

## Manual analysis

**IDA function:** `CLogin::OnCheckPinCodeResult` @ 0x5db000 (case 1 — create/assign PIN, opcode 10)

The `RegisterPin` packet is constructed inside the clientbound `OnCheckPinCodeResult` handler, case 1. When the server sends result code 1 (new PIN required), the client shows the `CPinCodeDlg::CreatePinCode` dialog and then builds and sends opcode 10:

```
COutPacket::COutPacket(&oPacket, 10)          // opcode 10 → REGISTER_PIN
// path: pin provided (PinCode >= 0)
Encode1(1u)                                    // pinInput flag = true
EncodeStr(formatted_pin)                       // pin string (4-digit formatted)
// path: cancelled (PinCode < 0)
Encode1(0)                                     // pinInput flag = false
// (no string follows)
```

Total wire for "set" path: 1 byte flag + 2-byte string length prefix + N string bytes.
Total wire for "cancel" path: 1 byte flag only.

### Dispatcher-offset finding

The CLogin dispatcher routes inbound (server→client) packets to handlers by opcode. Client-side outgoing packets are constructed directly with `COutPacket::COutPacket(buf, opcode)` — no accountId/sessionId prefix is prepended before the payload fields. The server-side atlas decoder sees exactly the bytes written after the opcode. **No offset discrepancy.** This is consistent across all three account-bucket packets (AcceptTos, RegisterPin, SetGender).

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| pinInput | 1 byte (Encode1) | 1 byte (WriteBool) | ✅ |
| pin | len-prefixed string (EncodeStr) | len-prefixed string (WriteAsciiString) | ✅ |

### Atlas decoder (`account/serverbound/register_pin.go`)

```go
m.pinInput = r.ReadBool()      // 1 byte
if m.pinInput {
    m.pin = r.ReadAsciiString() // len-prefixed string
}
```

Matches the IDA layout exactly for opcode 10 (create-pin flow).

### No bug — already correct

`RegisterPin.Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.

### SUMMARY path verification

`locateAtlasFile` resolves to `libs/atlas-packet/account/serverbound/register_pin.go` — correct account/ path. AcceptTos row unchanged ✅.

Ack: misc-audit Phase 2h on 2026-06-03
