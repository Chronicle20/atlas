# CashItemUseMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa54a2f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_megaphone.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the cash-slot item type. Confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int16 `slot - Encode2 @0xa54aaf; NO leading update_time in the header (opcode 0x4F straight to Encode2) - confirms updateTimeFirst=FALSE for gms_v84` | ❌ | width mismatch |
| 1 | byte | int32 `itemId - Encode4 @0xa54abb, then sub_489828(itemId) [get_consume_cash_item_type] drives the type switch` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `message - jumptable label 'cases 12,13,15' @0xa54d27 (type 12); shared tail EncodeStr @0xa55019` | ❌ | atlas: short — missing trailing field |
| 3 | byte | byte `whisper - Encode1 @0xa5502a, ONLY emitted when type==13 (SuperMegaphone); type==12 (Megaphone) skips this write` | ❌ | atlas: short — missing trailing field |
| 4 | string | int32 `updateTime(trailing) - falls to loc_A54CE8 'cases 33,71,72' -> CanSendExclRequest -> loc_A58E47: get_update_time() -> Encode4(result) @0xa58e4d-0xa58e50 -> SendPacket. TRAILING, matching gms_v83.` | ❌ | width mismatch |
