# CashItemUseItemMegaphone (← `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`)

- **IDA:** 0x623728
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the `hasItem` field. The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 written FIRST in this function's own header (before slot/itemId) - confirms updateTimeFirst=TRUE for gms_v87 @0x623811-0x62381a` | ❌ | width mismatch |
| 1 | int32 | int16 `slot - Encode2(*(this+144)) @0x623812` | ❌ | width mismatch |
| 2 | byte | int32 `itemId - Encode4(*(this+148)) @0x62381a` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `message - EncodeStr(CCtrlEdit::GetText) @0x62385b` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1(*(*(this+1504)+72)) @0x62386c` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `hasItem - Encode1(*(this+164)!=0) @0x62387e` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `invType - Encode4(*(this+152)) @0x623894, iff hasItem` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `slot(2) - Encode4(*(this+156)) @0x6238a2, iff hasItem` | ❌ | atlas: short — missing trailing field |
