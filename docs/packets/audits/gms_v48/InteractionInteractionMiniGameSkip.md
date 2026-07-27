# InteractionInteractionMiniGameSkip (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameSkip`)

- **IDA:** 0x5740df
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `who (sub_5740DF @0x5740df Omok / sub_5380C8 @0x5380c8 MemoryGame): turn passes to this slot and sets the 30000ms move timer (this[678]=30000) — the SKIP/turn-advance arm, mode 56, distinct from RESULT(55) which updates the win-loss record. No result strings.` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

