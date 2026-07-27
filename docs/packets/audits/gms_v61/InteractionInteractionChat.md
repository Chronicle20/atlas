# InteractionInteractionChat (← `CMiniRoomBaseDlg::OnPacketBase#Chat`)

- **IDA:** `0x5bec69`
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v61 reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (6; dispatch byte routed to vtable[80] OnChat)` | ✅ | v61 CMiniRoomBaseDlg::OnPacketBase = sub_5BEC69 @0x5bec69 case 6; body verified by cross-version fixture (== IDA-verified v83 read order, version-stable — no MajorVersion gate) |
| 1 | byte | byte `chatType (7=game-msg else font/0..3)` | ✅ | |
| 2 | byte | byte `slot (speaker position)` | ✅ | |
| 3 | string | string `message (sText)` | ✅ | |
