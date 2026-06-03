# ErrorSimple (← `CWvsContext::OnEntrustedShopCheckResult#ErrorSimple`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (cases 9/10/15 — client shows fixed string-pool notice, no further reads)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, cases 9 / 0xA / 0xF.

**This struct covers three modes (ErrorRetrieveFromFredrick=9, ErrorAnotherCharacter=10, ErrorRetrieveFromFredrick2=15)** — all share the same wire shape: mode byte only.

**Per-mode wire layout:**

```
case 9  (ERROR_RETRIEVE_FROM_FREDRICK):    Decode1(mode) → string-pool 3510 notice
case 0xA (ERROR_ANOTHER_CHARACTER):        Decode1(mode) → string-pool 3507 notice
case 0xF (ERROR_RETRIEVE_FROM_FREDRICK_2): Decode1(mode) → string-pool 3531 notice
```

**Atlas vs IDA comparison (all three modes):**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |

**Verdict: already correct.** Mode byte only; no further payload. The auto-tool ✅ is the real verdict.

**Mode 1 (ERROR_UNABLE_TO_OPEN_THE_STORE)** is defined as a constant in atlas but is **absent from the IDA switch** for `OnEntrustedShopCheckResult`. It is either a hire-merchant mode (handled elsewhere) or a legacy/KMS-only mode. A `_pending.md` row is added for it.

**Mode 8 (ERROR_UNKNOWN)** is in the IDA switch and decodes `Decode4(shopId) + Decode1(channelId)` then shows a channel-name notice. However there is **no atlas body function** that emits mode 8. This is a missing implementation, not a struct wire bug; also recorded in `_pending.md`.

Wire shape verified by `TestMerchantErrorSimpleWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 1 byte for modes 9, 10, and 15.

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
