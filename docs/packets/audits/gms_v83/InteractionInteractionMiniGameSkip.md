# InteractionInteractionMiniGameSkip (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameSkip`)

- **IDA:** 0x6e472e
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (63 SKIP; dispatch byte)` | ✅ |  |
| 1 | byte | byte `who (next-mover slot; COmokDlg::OnTimeOver §G5)` | ✅ |  |

