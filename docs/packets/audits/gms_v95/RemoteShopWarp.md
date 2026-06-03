# RemoteShopWarp (← `CWvsContext::OnEntrustedShopCheckResult#RemoteShopWarp`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x10 = REMOTE_SHOP_WARP)` | ✅ |  |
| 1 | int32 | int32 `shopId (v24)` | ✅ |  |
| 2 | byte | byte `channelId (v14 — 0xFE/0xFD/0xFF = error; otherwise shows YesNo warp dialog)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 0x10 (16 = REMOTE_SHOP_WARP).

**Per-mode wire layout (case 0x10):**

```
Decode1 → mode byte (0x10)
Decode4 → v24  (shopId)
Decode1 → v14  (channelId)
  if (v14 == 0xFE || v14 == 0xFD || v14 == 0xFF) {
    // shows error notice (string-pool 3499) — not a warp
  } else {
    // calls GetChannelName(v14), prompts YesNo warp dialog
    // if user confirms: CField::SendTransferChannelRequest(v14)
  }
```

The error path (`channelId == 0xFF`) is used by `HiredMerchantOperationRemoteShopWarpErrorBody()` which calls `NewRemoteShopWarp(mode, 0, 255)`.

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |
| shopId | Decode4 (uint32) | WriteInt (uint32) | ✅ |
| channelId | Decode1 (byte) | WriteByte | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

Wire shape verified by `TestRemoteShopWarpWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 6 bytes (1 mode + 4 shopId LE + 1 channelId).

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
