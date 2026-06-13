# PetCommandResponse (← `CPet::OnActionCommand`)

- **IDA:** 0x7048ab
- **Atlas file:** `libs/atlas-packet/pet/clientbound/command.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot — read by CUser::OnPetPacket before dispatch` | ✅ |  |
| 2 | byte | byte `mode` | ✅ |  |
| 3 | byte | byte `reaction index — gated mode <= 1` | ✅ |  |
| 4 | byte | byte `success flag — gated mode <= 1` | ✅ |  |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

