# InteractionOperationMemoryGameTieAnswer (← `CMemoryGameDlg::OnTieRequest`)

- **IDA:** 0x61bed0
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_tie_answer.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: short — missing trailing field |

