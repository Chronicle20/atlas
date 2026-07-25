# CashItemUseItemMegaphone (← `CItemSpeakerDlg::_SendConsumeCashItemUseRequest`)

- **IDA:** 0x5a7c42
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int16 `slot (*(WORD*)(this+120)) @0x5a7d35` | ❌ | width mismatch |
| 1 | byte | int32 `itemId (*(DWORD*)(this+124)) @0x5a7d40` | ❌ | width mismatch |
| 2 | byte | string `message (CCtrlEdit::GetText()) @0x5a7d61` | ❌ | width mismatch |
| 3 | int32 | byte `whisper (*(DWORD*)(*(DWORD*)(this+1444)+72)) @0x5a7d72` | ❌ | width mismatch |
| 4 | int32 | byte `hasItem (*(DWORD*)(this+140)!=0) @0x5a7d84` | ❌ | width mismatch |
| 5 | int32 | int32 `invType (*(DWORD*)(this+128)) @0x5a7d9a` | ✅ |  |
| 6 | byte | int32 `slot (*(DWORD*)(this+132)) @0x5a7da8` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `updateTime (SetExclRequestSent() return, GetTickCount-style read of g_CWvsApp+0x18) @0x5a7db6` | ❌ | atlas: short — missing trailing field |

