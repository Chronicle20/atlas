# RemoteShopWarp (← `CWvsContext::OnEntrustedShopCheckResult#RemoteShopWarp`)

- **IDA:** 0xa27d75
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x10 = REMOTE_SHOP_WARP)` | ✅ |  |
| 1 | int32 | int32 `shopId (v58 / dwCharacterID)` | ✅ |  |
| 2 | byte | byte `channelId (v14 — 0xFE/0xFD/0xFF = error; otherwise shows YesNo warp dialog)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnEntrustedShopCheckResult` @ 0xa27d75, case 16 — Decode4(shopId), Decode1(channelId). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
