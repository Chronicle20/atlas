# InteractionInviteResult (← `CMiniRoomBaseDlg::OnPacketBase#InviteResult`)

- **IDA:** 0x637d70
- **Atlas file:** `../../libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (3; dispatch byte)` | ✅ |  |
| 1 | byte | byte `result (v1)` | ✅ |  |
| 2 | string | string `message (sTargetName; ONLY read for result 2/3/4, NOT result 1)` | ✅ |  |


> note: client reads the trailing message string only for result codes 2/3/4
> (OnInviteResultStatic@0x637d70). See `docs/packets/ida-exports/_pending.md` →
> "InteractionInviteResult — conditional trailing string".
