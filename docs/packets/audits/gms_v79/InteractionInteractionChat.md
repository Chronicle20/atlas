# InteractionInteractionChat (← `CMiniRoomBaseDlg::OnPacketBase#Chat`)

- **IDA:** `0x62cd21`
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v79 reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | mode (6; dispatch byte → vtable[80] OnChat) | ✅ | v79 dispatcher @0x62cd21 case 6; body cross-version-identical to IDA-verified v83/v95 |
| 1 | byte | chatType (7=game-msg else font/0..3) | ✅ |  |
| 2 | byte | slot (speaker position) | ✅ |  |
| 3 | string | message (sText) | ✅ |  |

Body verified by cross-version byte fixture
`libs/atlas-packet/interaction/clientbound/v79_test.go#TestInteractionArmsV79`
(v79 encode == v83 encode; the interaction chat codec carries no MajorVersion
gate, and the v79 OnPacketBase dispatcher routes mode 6 to the OnChat arm).
