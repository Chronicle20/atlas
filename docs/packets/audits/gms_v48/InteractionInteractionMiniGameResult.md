# InteractionInteractionMiniGameResult (← `CMiniRoomBaseDlg::OnPacketBase#MemoryGameResult`)

- **IDA:** 0x573e1d
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultType/outcome (sub_573E1D @0x573e1d Omok / sub_537DD9 @0x537dd9 MemoryGame; ==1 draw path shows tie strings 438/1441, else win 437/1442 or lose 439/1443 AND updates the win-loss record +669 — this is the RESULT arm, mode 55)` | ✅ |  |
| 1 | byte | byte `winnerSlot (who won; read only when resultType!=1). v48 stops here — the modern v83+ struct then writes 2x GW_MiniGameRecord (40 bytes) that the v48 client does not read (harmless trailing over-write).` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

