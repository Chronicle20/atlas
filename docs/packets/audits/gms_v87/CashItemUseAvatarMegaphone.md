# CashItemUseAvatarMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa9fef9
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_avatar_megaphone.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the cash-slot item type (case 42). Confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `line[0] - jumptable case 42 (@0xaa05a4, case-number shifted vs gms_v95's 43, same shift as gms_v83); dialog GetResult (sub_848382@0xaa0736) then EncodeStr(line1) @0xaa0a0c` | ❌ | atlas: short — missing trailing field |
| 1 | byte | string `line[1] - EncodeStr(line2) @0xaa0a25` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `line[2] - EncodeStr(line3) @0xaa0a3e` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `line[3] - EncodeStr(line4) @0xaa0a57` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1(sub_848555 checkbox getter) @0xaa0a68` | ❌ | atlas: short — missing trailing field |
