# PetCommandResponse (← `CPet::OnActionCommand`)

- **IDA:** 0x76a6ab
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/command.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by dispatcher` | ✅ |  |
| 1 | byte | byte `slot — read by dispatcher` | ✅ |  |
| 2 | byte | byte `mode` | ✅ |  |
| 3 | byte | byte `reaction index — gated mode <= 1` | ✅ |  |
| 4 | byte | byte `success flag — gated mode <= 1` | ✅ |  |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

