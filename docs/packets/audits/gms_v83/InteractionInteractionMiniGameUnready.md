# InteractionInteractionMiniGameUnready (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameUnready`)

- **IDA:** 0x6e466e
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (59 UNREADY; dispatch byte; OnUserCancelReady reads no body)` | ✅ |  |

