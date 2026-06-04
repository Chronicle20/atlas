# ConfirmManage (← `CWvsContext::OnEntrustedShopCheckResult#ConfirmManage`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x11 = CONFIRM_MANAGE)` | ✅ |  |
| 1 | int32 | int32 `shopId / dwCharacterID (v23)` | ✅ |  |
| 2 | int16 | int16 `position / slot index (LOWORD of Decode2 @0xa28009)` | ✅ |  |
| 3 | int64 | int64 `liCashItemSN — 8-byte serial number (DecodeBuffer 8, v57 @0xa28019)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, case 17 — Decode4(shopId), Decode2(position), DecodeBuffer(8, liCashItemSN). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
