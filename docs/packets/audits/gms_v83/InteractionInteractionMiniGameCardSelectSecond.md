# InteractionInteractionMiniGameCardSelectSecond (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameCardSelectSecond`)

- **IDA:** 0x64e1c1
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (68 FLIP_CARD; dispatch byte)` | ✅ |  |
| 1 | byte | byte `turn (0 = second flip; CMemoryGameDlg::OnTurnUpCard §G5)` | ✅ |  |
| 2 | byte | byte `slot (index of the card turned up)` | ✅ |  |
| 3 | byte | byte `firstSlot (first flip's slot; second flip only)` | ✅ |  |
| 4 | byte | byte `type (0 owner-mismatch,1 visitor-mismatch,2 owner-match,3 visitor-match)` | ✅ |  |

