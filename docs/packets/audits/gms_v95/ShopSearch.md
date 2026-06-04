# ShopSearch (← `CWvsContext::OnEntrustedShopCheckResult#ShopSearch`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xD = SHOP_SEARCH)` | ✅ |  |
| 1 | int32 | int32 `dwSearchedShop (stored into CUIMiniMap::m_dwSearchedShop)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 0xD (13 = SHOP_SEARCH).

**Per-mode wire layout (case 0xD):**

```
Decode1 → mode byte (0xD)
Decode4 → v31->m_dwSearchedShop   // stored into CUIMiniMap singleton
// function returns
```

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |
| shopId | Decode4 (uint32) | WriteInt (uint32) | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

Wire shape verified by `TestShopSearchWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 5 bytes (1 mode + 4 shopId LE).

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
