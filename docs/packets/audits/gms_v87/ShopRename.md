# ShopRename (← `CWvsContext::OnEntrustedShopCheckResult#ShopRename`)

- **IDA:** 0xabf9ea
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0xE = SHOP_RENAME)` | ✅ |  |
| 1 | byte | byte `success flag (if 0 return early; if 1 show chat-log success message)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnEntrustedShopCheckResult` @ 0xabf9ea case 0xE: mode + Decode1(success flag). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
