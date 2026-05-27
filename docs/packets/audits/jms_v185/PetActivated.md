# PetActivated (← `CUserRemote::OnPetActivated`)

- **IDA:** 0xa576d3
- **Atlas file:** `libs/atlas-packet/pet/clientbound/activated.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot` | ✅ |  |
| 2 | byte | byte `active flag` | ✅ |  |
| 3 | byte | byte `show — gated active != 0` | ✅ |  |
| 4 | int32 | bytes `CPet::Init body — gated active != 0` | ❌ | width mismatch |
| 5 | string | byte `despawnMode — gated active == 0` | ❌ | width mismatch |
| 6 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

