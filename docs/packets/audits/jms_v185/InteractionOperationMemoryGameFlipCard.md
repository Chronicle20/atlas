# InteractionOperationMemoryGameFlipCard (← `CMemoryGameDlg::SendTurnUpCard`)

- **IDA:** 0x6c8b94
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_flip_card.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: short — missing trailing field |

