# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0xa2370b
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId` | ✅ |  |
| 1 | byte | byte `nLevel` | ✅ |  |
| 2 | int16 | int16 `nJob` | ✅ |  |
| 3 | int16 | int16 `nPOP (fame)` | ✅ |  |
| 4 | byte | byte `bMarriageRing (bool)` | ✅ |  |
| 5 | string | string `sCommunity (guild name)` | ✅ |  |
| 6 | string | string `sAlliance (alliance name)` | ✅ |  |
| 7 | byte | byte `pMedalInfo (medal slot byte)` | ✅ |  |
| 8 | byte | byte `v7 (pet count; if >0: SetMultiPetInfo reads pets in bool-terminated loop)` | ✅ |  |
| 9 | int32 | byte `taming mob active flag` | ❌ | width mismatch |
| 10 | string | byte `wish list count` | ❌ | width mismatch |
| 11 | int32 | int32 `MedalAchievementInfo: nEquipedMedalID` | ✅ |  |
| 12 | int16 | int16 `MedalAchievementInfo: ausMedalQuestID count` | ✅ |  |
| 13 | int32 | int32 `chair list count (ZArray<long>::_Alloc + DecodeBuffer with 4 * count bytes)` | ✅ |  |
| 14 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

