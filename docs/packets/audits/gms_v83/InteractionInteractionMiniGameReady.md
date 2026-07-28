# InteractionInteractionMiniGameReady (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameReady`)

- **IDA:** 0x6e4608
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (58 READY; dispatch byte; OnUserReady reads no body)` | ✅ |  |

