# CharacterList (← `CLogin::OnSelectWorldResult`)

- **IDA:** 0x5dda00
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `nCount (character entries)` | ✅ |  |
| 2 | byte | string `GW_CharacterStat::Decode start: characterName (loop body entry 0)` | 🔍 | sub-struct: c — see _substruct/ |
| 3 | byte | int32 `characterId` | ❌ | width mismatch |
| 4 | int32 | int32 `level` | ✅ |  |
| 5 | int32 | int32 `job` | ✅ |  |
| 6 | byte | byte `subJob (?)` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `str` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `dex` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `int` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `luk` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `hp` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `maxHp` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int32 `mp` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `maxMp` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int32 `ap` | ❌ | atlas: short — missing trailing field |
| 16 | byte | int32 `sp` | ❌ | atlas: short — missing trailing field |
| 17 | byte | int32 `exp` | ❌ | atlas: short — missing trailing field |
| 18 | byte | int32 `fame` | ❌ | atlas: short — missing trailing field |
| 19 | byte | int32 `gachaExp (?)` | ❌ | atlas: short — missing trailing field |
| 20 | byte | int32 `mapId` | ❌ | atlas: short — missing trailing field |
| 21 | byte | byte `spawnPoint` | ❌ | atlas: short — missing trailing field |
| 22 | byte | int32 `subJob2 (?)` | ❌ | atlas: short — missing trailing field |
| 23 | byte | byte `gender` | ❌ | atlas: short — missing trailing field |
| 24 | byte | byte `skin` | ❌ | atlas: short — missing trailing field |
| 25 | byte | int32 `face` | ❌ | atlas: short — missing trailing field |
| 26 | byte | byte `megaphoneFlag (AvatarLook)` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int32 `hair` | ❌ | atlas: short — missing trailing field |
| 28 | byte | int32 `equip slot 0 itemId (AvatarLook equipment loop body)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 29 | byte | int32 `equip slot 0 itemId masked (AvatarLook masked-equip loop body)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 30 | byte | int32 `pet 0 itemId (AvatarLook pet loop body)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 31 | byte | byte `onFamily` | ❌ | atlas: short — missing trailing field |
| 32 | byte | byte `hasRank` | ❌ | atlas: short — missing trailing field |
| 33 | byte | int32 `worldRank` | ❌ | atlas: short — missing trailing field |
| 34 | byte | int32 `worldRankMove` | ❌ | atlas: short — missing trailing field |
| 35 | byte | int32 `jobRank` | ❌ | atlas: short — missing trailing field |
| 36 | byte | int32 `jobRankMove` | ❌ | atlas: short — missing trailing field |
| 37 | byte | byte `m_bLoginOpt (hasPic)` | ❌ | atlas: short — missing trailing field |
| 38 | byte | int32 `m_nSlotCount` | ❌ | atlas: short — missing trailing field |
| 39 | byte | int32 `m_nBuyCharCount` | ❌ | atlas: short — missing trailing field |

