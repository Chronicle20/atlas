# ConfirmManage (← `CWvsContext::OnEntrustedShopCheckResult#ConfirmManage`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x11 = CONFIRM_MANAGE)` | ✅ |  |
| 1 | int32 | int32 `shopId / dwCharacterID (v24)` | ✅ |  |
| 2 | int16 | int16 `position / slot index (v25)` | ✅ |  |
| 3 | int64 | int64 `liCashItemSN — 8-byte serial number (DecodeBuffer 8, stored as _LARGE_INTEGER)` | ✅ |  |

## Manual analysis

**Scope note:** employee-shop packet only. Hire-merchant modes are handled by task-067 (commerce/interaction).

**IDA function:** `CWvsContext::OnEntrustedShopCheckResult` @ 0x9ffcb0, case 0x11 (17 = CONFIRM_MANAGE).

**Per-mode wire layout (case 0x11):**

```
Decode1          → mode byte (0x11)
Decode4          → v24  (shopId / dwCharacterID)
Decode2          → v25  (position / slot index)
DecodeBuffer(8)  → liCashItemSN  (_LARGE_INTEGER, 8 bytes)

// Client then checks if player has a birthday (m_wstr <= 0 = no birthday)
// If no birthday: shows error notice
// If birthday: prompts YesNo dialog, then builds PLAYER_INTERACTION outbound packet
//   containing: Encode1(0xE), Encode1(4), Encode1(5),
//               EncodeStr(birthday_pin), Encode4(v24), Encode1(1), Encode2(v25),
//               EncodeBuffer(&liCashItemSN, 8)
```

**Atlas vs IDA comparison:**

| Field | IDA | Atlas | Match? |
|---|---|---|---|
| mode | Decode1 (byte) | WriteByte | ✅ |
| shopId | Decode4 (uint32) | WriteInt (uint32) | ✅ |
| position | Decode2 (uint16) | WriteShort (uint16) | ✅ |
| serialNumber | DecodeBuffer(8) = uint64 | WriteLong (uint64) | ✅ |

**Verdict: already correct.** The auto-tool ✅ is the real verdict.

Note: the data-dependent birthday dialog and outbound packet construction are entirely
client-side; they do not change the inbound wire fields. The ConfirmManage struct
models the fixed server-to-client wire layout correctly.

Wire shape verified by `TestConfirmManageWireShape` in
`libs/atlas-packet/merchant/clientbound/operation_test.go`:
all four variants produce exactly 15 bytes (1 mode + 4 shopId + 2 position + 8 serialNumber LE).

No version gate needed — layout is identical across all versions.

Ack: misc-audit Phase 2f on 2026-06-03
