# InteractionInteractionMiniGameStartMatchCards (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameStartMatchCards`)

- **IDA:** 0x64e632
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (61 START; dispatch byte)` | ✅ |  |
| 1 | byte | byte `firstMover slot (first mover = slot != this byte; CMemoryGameDlg::OnUserStart §G1)` | ✅ |  |
| 2 | byte | byte `count (card count; m_nCount)` | ✅ |  |
| 3 | int32 | bytes `deck (count x int32 cardId; DecodeBuffer(4*count) §G1)` | ✅ |  |

