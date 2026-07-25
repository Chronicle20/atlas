# InteractionInteractionMiniGameStartOmok (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameStartOmok`)

- **IDA:** 0x6e469c
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (61 START; dispatch byte)` | ✅ |  |
| 1 | byte | byte `firstMover slot (first mover = slot != this byte; COmokDlg::OnUserStart §G1)` | ✅ |  |

