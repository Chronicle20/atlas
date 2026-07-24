# CashItemUseSuperMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0x9eb3e0
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_super_megaphone.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 in the SHARED HEADER, written FIRST (before nPOS/itemId) - definitively confirms updateTimeFirst=TRUE for gms_v95 @0x9eb4b7-0x9eb4c1` | ❌ | width mismatch |
| 1 | byte | int16 `nPOS - Encode2 @0x9eb4ce-0x9eb4d2` | ❌ | width mismatch |
| 2 | int32 | int32 `itemId - Encode4 @0x9eb4df-0x9eb4e3, then get_consume_cash_item_type(itemId) drives the type switch` | ✅ |  |
| 3 | byte | string `message - shared cases 12/13/15/45 body, EncodeStr @0x9ebc59 (after CSpeakerWorldDlg/CUtilDlgEx message-input dialog returns)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1 @0x9ebc75, ONLY emitted when type==13 (SuperMegaphone) or type==45; type==12 (Megaphone) skips this write` | ❌ | atlas: short — missing trailing field |

