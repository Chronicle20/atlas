# CashItemUseTripleMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa54a2f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_triple_megaphone.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator (the cash-item sub-type driving `get_consume_cash_item_type`) a flat positional diff cannot model. The verdict is capped to 🔍; the byte-fixture test(s) linked via the `packet-audit:verify` marker are the actual per-branch verification (task-123 megaphone/MapleTV gap-fill pass).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | (see byte-fixture) | `case 60 (@0xa54ae4): count(byte)+count x line(str)+whisper(bool), TRAILING update_time in the shared send tail (updateTimeFirst=FALSE) — task-123 megaphone gap-fill pass.` | 🔍 | flat-diff-invalid: data-dependent branch — see byte-fixture test for the verified per-branch wire shape |
