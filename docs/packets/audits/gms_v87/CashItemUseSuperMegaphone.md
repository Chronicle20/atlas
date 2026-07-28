# CashItemUseSuperMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa9fef9
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_super_megaphone.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the cash-slot item type. The verdict is capped to 🔍; confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 in the SHARED HEADER, written FIRST - confirms updateTimeFirst=TRUE for gms_v87 @0xa9ff73-0xa9ff7c` | ❌ | width mismatch |
| 1 | byte | string `message - jumptable label 'cases 12,13,15' @0xaa01ff (type 13 confirm dialog, StringPool 0x119); shared tail EncodeStr @0xaa04f1` | ❌ | atlas: short — missing trailing field |
| 2 | byte | byte `whisper - Encode1 @0xaa0502, gated on type==13 (cmp type,0xD @0xaa04f6)` | ❌ | atlas: short — missing trailing field |
