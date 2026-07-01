# InteractionOperationMemoryGameFlipCard (← `CMemoryGameDlg::SendTurnUpCard`)

- **IDA:** 0x5ff6ba
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_memory_game_flip_card.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `first` | ✅ |  |
| 1 | byte | byte `index` | ✅ |  |

