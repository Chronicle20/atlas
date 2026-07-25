# CashItemUseItemMegaphone (← `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`)

- **IDA:** 0x5c9e70
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int32 `update_time = get_update_time(); Encode4 in this function's OWN header, written FIRST (before nPOS/itemId) @0x5c9fb3-0x5c9fbd - confirms updateTimeFirst=TRUE` | ❌ | width mismatch |
| 1 | int16 | int16 `nPOS (this->_nPOS) - Encode2 @0x5c9fce` | ✅ |  |
| 2 | int32 | int32 `itemId (this->_nItemID) - Encode4 @0x5c9fde` | ✅ |  |
| 3 | int32 | string `message = CCtrlEdit::GetText(_pEditInput) - EncodeStr @0x5c9ffa` | ❌ | width mismatch |
| 4 | int32 | byte `whisper = _pCheckBoxWhisper.p->m_bChecked - Encode1 @0x5ca00d` | ❌ | width mismatch |
| 5 | byte | byte `hasItem = (_pItem.p != 0) - Encode1 @0x5ca024` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `invType (this->_nTargetTI) - Encode4 @0x5ca03d` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `slot (this->_nTargetPOS) - Encode4 @0x5ca04d` | ❌ | atlas: short — missing trailing field |

