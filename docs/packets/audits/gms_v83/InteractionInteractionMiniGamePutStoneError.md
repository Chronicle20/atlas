# InteractionInteractionMiniGamePutStoneError (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGamePutStoneError`)

- **IDA:** 0x6e4065
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (dispatcher byte; PUT_STONE_ERROR)` | ✅ |  |
| 1 | byte | byte `errorCode (==double-3 code -> "you have double 3s", else "cant put it there")` | ✅ |  |

