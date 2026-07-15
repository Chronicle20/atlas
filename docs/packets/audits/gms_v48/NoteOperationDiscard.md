# NoteOperationDiscard (← `CMemoListDlg::SetRet`)

- **IDA:** 0x534dc4
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_discard.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count — number of checked (nFlag==2) memos to discard (deleteCount @0x534e9a)` | ✅ |  |
| 1 | byte | byte `emptySlotCount — free inventory slots (tab 4, CharacterData::GetEmptySlotCount @0x534ea5)` | ✅ |  |
| 2 | int32 | int32 `dwSN (memo serial number) — per-entry loop body @0x534ed7 (count iterations; analyzer flattens)` | ✅ |  |
| 3 | byte | byte `nFlag (memo type flag) @0x534eec` | ✅ |  |

