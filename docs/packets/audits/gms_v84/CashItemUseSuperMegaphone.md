# CashItemUseSuperMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa54a2f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_super_megaphone.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the cash-slot item type. Confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `message - jumptable label 'cases 12,13,15' @0xa54d27 (type 13 confirm dialog, StringPool 0x112); shared tail EncodeStr @0xa55019` | ❌ | atlas: short — missing trailing field |
| 1 | byte | byte `whisper - Encode1 @0xa5502a, gated on type==13 (cmp type,0xD @0xa5501e)` | ❌ | atlas: short — missing trailing field |
| 2 | string | int32 `updateTime(trailing) - falls to loc_A54CE8 'cases 33,71,72' -> CanSendExclRequest -> loc_A58E47: get_update_time() -> Encode4(result) -> SendPacket. TRAILING, matching gms_v83.` | ❌ | width mismatch |
