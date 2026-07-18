# CashItemUseItemMegaphone (← `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`)

- **IDA:** 0x55dc01
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int16 `slot (*(WORD*)(this+120)) @0x55dce7` | ❌ | width mismatch |
| 1 | byte | int32 `itemId (*(DWORD*)(this+124)) @0x55dcf2` | ❌ | width mismatch |
| 2 | byte | string `message (CCtrlEdit::GetText()) @0x55dd13` | ❌ | width mismatch |
| 3 | int32 | byte `whisper (*(DWORD*)(*(DWORD*)(this+1396)+72)) @0x55dd24` | ❌ | width mismatch |
| 4 | int32 | byte `hasItem (*(DWORD*)(this+140)!=0) @0x55dd36` | ❌ | width mismatch |
| 5 | int32 | int32 `invType (*(DWORD*)(this+128)) @0x55dd4c` | ✅ |  |
| 6 | byte | int32 `slot (*(DWORD*)(this+132)) @0x55dd5a` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `updateTime (SetExclRequestSent() return) @0x55dd68` | ❌ | atlas: short — missing trailing field |

