# OpenShop (← `CWvsContext::OnEntrustedShopCheckResult#OpenShop`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 7 = OPEN_SHOP; client calls SendOpenShopRequest — no further reads)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 7 (0x07).

**Per-mode wire layout (case 7):**

```
Decode1 → mode byte (7)
// client immediately calls SendOpenShopRequest(this->m_nEmployeeItemPos, this->m_nEmployeeItemID, 1)
// no further Decode calls in this case
```

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

Wire shape verified by `TestOpenShopWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 1 byte (0x07).

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
