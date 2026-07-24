# CashItemUseSuperMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xaef2f5
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_super_megaphone.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 in the SHARED HEADER, written FIRST - confirms updateTimeFirst=TRUE for jms185 @0xaef36c-0xaef375` | ❌ | width mismatch |
| 1 | byte | string `message - jumptable comment 'cases 12,13,15,47,48' @0xaef5b9 (type 47/Heart and type 48/Skull); shared tail EncodeStr @0xaef98a` | ❌ | atlas: short — missing trailing field |
| 2 | byte | byte `whisper - Encode1 @0xaef9a7, gated on type==13(0x0D)\|\|type==47(0x2F)\|\|type==48(0x30) @0xaef98f-0xaef9a7 (type 47/48, TAKEN)` | ❌ | atlas: short — missing trailing field |

