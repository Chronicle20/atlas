# CashItemUseMapleTV (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa9fef9
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_maple_tv.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator (the cash-item sub-type driving `get_consume_cash_item_type`) a flat positional diff cannot model. The verdict is capped to 🔍; the byte-fixture test(s) linked via the `packet-audit:verify` marker are the actual per-branch verification (task-123 megaphone/MapleTV gap-fill pass).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | (see byte-fixture) | `cases 46-51 (@0xaa2985-0xaa3ea4), one per tvType arm: tvType-conditional pad/ear/receiver prefix + 5xline(str), NO trailing update_time (updateTimeFirst=TRUE) — task-123 megaphone gap-fill pass.` | 🔍 | flat-diff-invalid: data-dependent branch — see byte-fixture test for the verified per-branch wire shape |
