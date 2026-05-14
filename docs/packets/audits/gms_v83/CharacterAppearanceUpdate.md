# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0x98367e
- **Atlas file:** `libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch` | ✅ |  |
| 1 | byte | byte `v4 flags byte: bit0=avatarLook, bit1=speed, bit2=carryItem` | ✅ |  |
| 2 | byte | bytes `AvatarLook::Decode — full avatar look data (guard: v4 & 1)` | ❌ | width mismatch |
| 3 | byte | byte `nSpeed (guard: v4 & 2)` | ✅ |  |
| 4 | int32 | byte `nCarryItemEffect (guard: v4 & 4)` | ❌ | width mismatch |
| 5 | byte | byte `bCouple flag` | ✅ |  |
| 6 | int32 | bytes `liCoupleItemSN (8 bytes) + liPairItemSN (8 bytes) + dwPairCharacterId (4 bytes)` | ❌ | width mismatch |
| 7 | byte | byte `bFriendship flag` | ✅ |  |
| 8 | int32 | bytes `liFriendshipItemSN (8 bytes) + liFriendshipPairItemSN (8 bytes) + dwFriendCharacterId (4 bytes)` | ❌ | width mismatch |
| 9 | byte | byte `bMarriage flag` | ✅ |  |
| 10 | byte | int32 `dwMarriageCharacterID (guard: bMarriage)` | ❌ | width mismatch |
| 11 | int32 | int32 `dwMarriagePairCharacterID (guard: bMarriage)` | ✅ |  |
| 12 | byte | int32 `nWeddingRingID (guard: bMarriage)` | ❌ | width mismatch |
| 13 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

