# InteractionOperationMemoryGameMoveStone (← `COmokDlg::PutStoneChecker`)

- **IDA:** 0x65320c
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_move_stone.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `point(8)` | ✅ |  |
| 1 | byte | byte `color` | ✅ |  |

