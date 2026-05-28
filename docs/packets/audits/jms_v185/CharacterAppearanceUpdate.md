# CharacterAppearanceUpdate (← `CUserRemote::OnAvatarModified`)

- **IDA:** 0xa57221
- **Atlas file:** `libs/atlas-packet/character/clientbound/appearance_update.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `v3 flags byte: bit0=avatarLook, bit1=speed, bit2=carryItem` | ❌ | width mismatch |
| 1 | byte | bytes `AvatarLook::Decode (if bit0)` | ❌ | width mismatch |
| 2 | byte | byte `nSpeed (if bit1)` | ✅ |  |
| 3 | byte | byte `nCarryItemEffect (if bit2)` | ✅ |  |
| 4 | int32 | byte `bCouple count / flag` | ❌ | width mismatch |
| 5 | byte | int32 `couple count (if bCouple > 0)` | ❌ | width mismatch |
| 6 | int32 | bytes `couple item SN (16 bytes per entry)` | ❌ | width mismatch |
| 7 | byte | int32 `pair characterId (per entry)` | ❌ | width mismatch |
| 8 | int32 | byte `bFriendship count / flag` | ❌ | width mismatch |
| 9 | byte | int32 `friendship count (if > 0)` | ❌ | width mismatch |
| 10 | byte | bytes `friendship item SN (16 bytes per entry)` | ❌ | width mismatch |
| 11 | int32 | int32 `friendship pair characterId (per entry)` | ✅ |  |
| 12 | byte | byte `bMarriage flag` | ✅ |  |
| 13 | int32 | int32 `dwMarriageCharacterID (if bMarriage)` | ✅ |  |
| 14 | int32 | int32 `dwMarriagePairCharacterID (if bMarriage)` | ✅ |  |
| 15 | int32 | int32 `nWeddingRingID (if bMarriage)` | ✅ |  |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

