# ShopRename (← `CWvsContext::OnEntrustedShopCheckResult#ShopRename`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xE = SHOP_RENAME)` | ✅ |  |
| 1 | byte | byte `success flag (if 0 return early; if 1 show chat-log success message)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 0xE (14 = SHOP_RENAME).

**Per-mode wire layout (case 0xE):**

```
Decode1 → mode byte (0xE)
Decode1 → success flag
  if (success == 0) return;           // silent early-out
  // else: load string-pool 0xAE or 0xAF and add to chat log
```

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |
| success | Decode1 (bool) | WriteBool | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

Wire shape verified by `TestShopRenameWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 2 bytes (1 mode + 1 success), for both true and false.

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
