# NoteOperationDiscard (← `CMemoListDlg::SetRet`)

- **IDA:** 0x624280
- **Atlas file:** `../../libs/atlas-packet/note/serverbound/operation_discard.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count — number of checked (nFlag==3) memos to discard` | ✅ |  |
| 1 | byte | byte `emptySlotCount — free inventory slots (tab 4, from CharacterData::GetEmptySlotCount)` | ✅ |  |
| 2 | int32 | int32 `dwSN (memo serial number) — per-entry loop body (count iterations; analyzer flattens)` | ✅ |  |
| 3 | byte | byte `nFlag (memo type flag)` | ✅ |  |

