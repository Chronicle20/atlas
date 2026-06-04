# ConfirmManage (← `CWvsContext::OnEntrustedShopCheckResult#ConfirmManage`)

- **IDA:** 0xabf9ea
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x11 = CONFIRM_MANAGE)` | ✅ |  |
| 1 | int32 | int32 `shopId / dwCharacterID (v23)` | ✅ |  |
| 2 | int16 | int16 `position / slot index (v24)` | ✅ |  |
| 3 | int64 | int64 `liCashItemSN — 8-byte serial number (DecodeBuffer 8, stored as _LARGE_INTEGER)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnEntrustedShopCheckResult` @ 0xabf9ea case 0x11: mode + Decode4(shopId) + Decode2(slot) + DecodeBuffer(8 bytes liCashItemSN). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
