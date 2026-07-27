# CashItemUseMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa0a63f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_megaphone.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `to: AP stat flag (5050000) / SP skill id (5050001-4). Encode4 @case23/24 in CWvsContext::SendConsumeCashItemUseRequest 0xa0a63f` | ❌ | width mismatch |
| 1 | int32 | int32 `from: AP stat flag / SP skill id. second Encode4` | ✅ |  |
| 2 | byte | int32 `trailing update_time = get_update_time(); Encode4 in common send tail (LABEL_41)` | ❌ | atlas: short — missing trailing field |

