# PetCommandResponse (← `CPet::OnActionCommand`)

- **IDA:** 0x6a3930
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/command.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | byte | byte `mode (v5 — 0/1 picks reaction table; 2+ falls through)` | ✅ |  |
| 3 | byte | byte `reaction index — gated on mode == 0 (interaction) or mode == 1 (food)` | ✅ |  |
| 4 | byte | byte `success flag — gated same as reaction index` | ✅ |  |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

