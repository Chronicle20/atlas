# InteractionOperationMemoryGameMoveStone (← `COmokDlg::PutStoneChecker`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_move_stone.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

