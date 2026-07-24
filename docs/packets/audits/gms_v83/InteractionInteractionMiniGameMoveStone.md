# InteractionInteractionMiniGameMoveStone (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameMoveStone`)

- **IDA:** 0x6e3f5b
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (64 MOVE_STONE; dispatch byte)` | ✅ |  |
| 1 | int32 | int32 `x (int32; DecodeBuffer(8) first half; COmokDlg::OnPutStoneChecker §G5)` | ✅ |  |
| 2 | int32 | int32 `y (int32; DecodeBuffer(8) second half)` | ✅ |  |
| 3 | byte | byte `stoneType (placing player's color 1/2)` | ✅ |  |

