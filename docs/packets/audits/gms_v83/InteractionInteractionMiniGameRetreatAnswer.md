# InteractionInteractionMiniGameRetreatAnswer (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameRetreatAnswer`)

- **IDA:** 0x6e41f9
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (55 RETREAT_ANSWER; OnPacketBase dispatch byte)` | ✅ |  |
| 1 | byte | byte `accept (1 = accepted; COmokDlg::OnRetreatResult §G2)` | ✅ |  |
| 2 | byte | byte `N stones to pop from move-history tail (accept only)` | ✅ |  |
| 3 | byte | byte `turnSlot (slot whose turn follows; my-turn = turnSlot==mySlot)` | ✅ |  |

