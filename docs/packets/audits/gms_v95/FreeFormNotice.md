# FreeFormNotice (← `CWvsContext::OnEntrustedShopCheckResult#FreeFormNotice`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x12 = FREE_FORM_NOTICE)` | ✅ |  |
| 1 | byte | byte `flag (if 0 return immediately; atlas always encodes 1)` | ✅ |  |
| 2 | string | string `sMsg — message string (only read when flag != 0)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 0x12 (18 = FREE_FORM_NOTICE).

**Per-mode wire layout (case 0x12):**

```
Decode1   → mode byte (0x12)
Decode1   → flag  (if flag == 0 → return; otherwise continue)
DecodeStr → sMsg  (ZXString<char> — 2-byte LE length prefix + bytes)
// client shows a CUtilDlg::Notice with sMsg
```

The atlas encoder hardcodes `WriteBool(true)` (flag = 1) and always includes the message.
The `FreeFormNotice.Decode` reads both fields (discards the bool, stores the message),
which correctly handles any flag value.

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |
| flag | Decode1 (byte) | WriteBool(true) = 0x01 | ✅ |
| message | DecodeStr (2+len bytes) | WriteAsciiString (2+len bytes) | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

The `FreeFormNotice` struct is data-dependent only in that a 0-flag packet (no message)
can be decoded but not generated. Since atlas always sends flag=1 with a message, the
static wire is fully modeled.

Wire shape verified by `TestFreeFormNoticeWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 7 bytes for message `"Hi!"`:
1 (mode) + 1 (flag=1) + 2 (len prefix=3) + 3 (message bytes).

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
