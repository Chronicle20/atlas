# CashItemUseMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa9fef9
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_megaphone.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (cash-slot item type). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 in the SHARED HEADER, written FIRST (before slot/itemId) - confirms updateTimeFirst=TRUE for gms_v87 @0xa9ff73-0xa9ff7c` | ❌ | width mismatch |
| 1 | int32 | int16 `slot - Encode2 @0xa9ff84-0xa9ff87` | ❌ | width mismatch |
| 2 | byte | int32 `itemId - Encode4 @0xa9ff90-0xa9ff93, then get_consume_cash_item_type(itemId) drives the type switch` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `message - jumptable label 'cases 12,13,15' @0xaa01ff (type 12); shared tail EncodeStr @0xaa04f1` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1 @0xaa0502, ONLY emitted when type==13 (SuperMegaphone); type==12 (Megaphone) skips this write` | ❌ | atlas: short — missing trailing field |
