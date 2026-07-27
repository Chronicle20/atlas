# InteractionInteractionMiniGameEnter (← `CMiniRoomBaseDlg::OnPacketBase#EnterMiniGame`)

- **IDA:** 0x65ed1c
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction_minigame_enter.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (4; OnPacketBase dispatch byte)` | ✅ |  |
| 1 | byte | byte `slot (m_apAvatar index; OnEnterBase @0x65ed1c)` | ✅ |  |
| 2 | bytes | bytes `avatar look (DecodeAvatar AvatarLook blob)` | ✅ |  |
| 3 | string | string `name (m_asUserID)` | ✅ |  |
| 4 | int16 | int32 `record: Unknown (COmokDlg::OnEnter sub_6E3BCC @0x6e3bcc -> sub_4E42FC 20-byte = 5 x int32)` | ❌ | width mismatch |
| 5 | int32 | int32 `record: Wins` | ✅ |  |
| 6 | int32 | int32 `record: Ties` | ✅ |  |
| 7 | int32 | int32 `record: Losses` | ✅ |  |
| 8 | int32 | int32 `record: Points` | ✅ |  |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

