# PetActivated (← `CUserRemote::OnPetActivated`)

- **IDA:** 0x9547d0
- **Atlas file:** `libs/atlas-packet/pet/clientbound/activated.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `slot (v3, 0..2)` | ✅ |  |
| 2 | byte | byte `active flag` | ✅ |  |
| 3 | byte | byte `show (active path only) — gated active != 0` | ✅ |  |
| 4 | int32 | bytes `CPet::Init body (active path only): templateId + name + petLockerSN + x + y + stance + foothold + nameTag + chatBalloon` | ❌ | width mismatch |
| 5 | string | byte `despawnMode (inactive path only)` | ❌ | width mismatch |
| 6 | int64 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

