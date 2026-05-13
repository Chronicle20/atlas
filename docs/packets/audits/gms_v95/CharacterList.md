# CharacterList (← `CLogin::OnSelectWorldResult`)

- **IDA:** 0x5dda00
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `nCount (character entries)` | ✅ |  |
| 2 | int32 | string `GW_CharacterStat::Decode start: characterName (loop body entry 0)` | ❌ | width mismatch |
| 3 | byte | int32 `characterId` | ❌ | width mismatch |
| 4 | byte | int32 `level` | ❌ | width mismatch |
| 5 | int32 | int32 `job` | ✅ |  |
| 6 | int32 | byte `subJob (?)` | ❌ | width mismatch |
| 7 | int64 | int32 `str` | ❌ | width mismatch |
| 8 | byte | int32 `dex` | ❌ | width mismatch |
| 9 | int16 | int32 `int` | ❌ | width mismatch |
| 10 | int16 | int32 `luk` | ❌ | width mismatch |
| 11 | int16 | int32 `hp` | ❌ | width mismatch |
| 12 | int16 | int32 `maxHp` | ❌ | width mismatch |
| 13 | int16 | int32 `mp` | ❌ | width mismatch |
| 14 | int16 | int32 `maxMp` | ❌ | width mismatch |
| 15 | int16 | int32 `ap` | ❌ | width mismatch |
| 16 | int16 | int32 `sp` | ❌ | width mismatch |
| 17 | int16 | int32 `exp` | ❌ | width mismatch |
| 18 | int16 | int32 `fame` | ❌ | width mismatch |
| 19 | int16 | int32 `gachaExp (?)` | ❌ | width mismatch |
| 20 | int32 | int32 `mapId` | ✅ |  |
| 21 | int16 | byte `spawnPoint` | ❌ | width mismatch |
| 22 | int32 | int32 `subJob2 (?)` | ✅ |  |
| 23 | int32 | byte `gender` | ❌ | width mismatch |
| 24 | byte | byte `skin` | ✅ |  |
| 25 | int32 | int32 `face` | ✅ |  |
| 26 | int16 | byte `megaphoneFlag (AvatarLook)` | ❌ | width mismatch |
| 27 | byte | int32 `hair` | ❌ | width mismatch |
| 28 | byte | int32 `equip slot 0 itemId (AvatarLook equipment loop body)` | ❌ | width mismatch |
| 29 | int32 | int32 `equip slot 0 itemId masked (AvatarLook masked-equip loop body)` | ✅ |  |
| 30 | byte | int32 `pet 0 itemId (AvatarLook pet loop body)` | ❌ | width mismatch |
| 31 | int32 | byte `onFamily` | ❌ | width mismatch |
| 32 | byte | byte `hasRank` | ✅ |  |
| 33 | byte | int32 `worldRank` | ❌ | width mismatch |
| 34 | int32 | int32 `worldRankMove` | ✅ |  |
| 35 | int32 | int32 `jobRank` | ✅ |  |
| 36 | int32 | int32 `jobRankMove` | ✅ |  |
| 37 | int32 | byte `m_bLoginOpt (hasPic)` | ❌ | width mismatch |
| 38 | int64 | int32 `m_nSlotCount` | ❌ | width mismatch |
| 39 | int64 | int32 `m_nBuyCharCount` | ❌ | width mismatch |
| 40 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 41 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 42 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

