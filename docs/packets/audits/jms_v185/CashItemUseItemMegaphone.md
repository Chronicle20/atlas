# CashItemUseItemMegaphone (← `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`)

- **IDA:** 0x660672
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on the `hasItem` field. Confirmed per-branch via byte-level tests (task-123 phase 3).

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4(v22, update_time) written FIRST in this function's own header (before slot/itemId) - confirms updateTimeFirst=TRUE for jms_v185 @0x66075b-0x660764` | ❌ | width mismatch |
| 1 | int32 | int16 `slot - Encode2(v22, *(this+72)) @0x660776` | ❌ | width mismatch |
| 2 | byte | int32 `itemId - Encode4(v22, *(this+37)) @0x660784` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `message - EncodeStr(v22, CCtrlEdit::GetText(...)) @0x6607a5` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `whisper - Encode1(v22, *(*(this+617)+72)) @0x6607b6` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `hasItem - Encode1(v22, *(this+41)!=0) @0x6607c8` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `invType - Encode4(v22, *(this+38)) @0x6607de, iff hasItem` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `slot(2) - Encode4(v22, *(this+39)) @0x6607ec, iff hasItem` | ❌ | atlas: short — missing trailing field |
