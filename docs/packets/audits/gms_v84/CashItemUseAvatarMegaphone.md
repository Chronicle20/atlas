# CashItemUseAvatarMegaphone (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xa54a2f
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_avatar_megaphone.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the cash-slot item type (case 42). Confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `line[0] - jumptable case 42 @0xa550cc (same case-number shift vs gms_v95's 43 as gms_v83/gms_v87); dialog GetResult (sub_81780F@0xa551b8) then EncodeStr(line1) @0xa5548e` | ❌ | atlas: short — missing trailing field |
| 1 | byte | string `line[1] - EncodeStr(line2) @0xa554a7` | ❌ | atlas: short — missing trailing field |
| 2 | byte | string `line[2] - EncodeStr(line3) @0xa554c0` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `line[3] - EncodeStr(line4) @0xa554d9` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1(sub_8179E2 checkbox getter) @0xa554ea` | ❌ | atlas: short — missing trailing field |
| 5 | string | int32 `updateTime(trailing) - falls to loc_A5504B -> loc_A54CE8 'cases 33,71,72' -> CanSendExclRequest -> loc_A58E47: get_update_time() -> Encode4(result) -> SendPacket. Same shared tail as Megaphone/SuperMegaphone v84.` | ❌ | width mismatch |
