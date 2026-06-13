# InteractionOperationMemoryGameMoveStone (← `COmokDlg::PutStoneChecker`)

- **IDA:** 0x6e8a19
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_move_stone.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | width mismatch |
| 1 | byte | bytes `` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: short — missing trailing field |

