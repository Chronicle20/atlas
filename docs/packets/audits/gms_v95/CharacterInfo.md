# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0xa05750
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

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
| 11 | byte | int32 `MedalAchievementInfo: nEquipedMedalID` | ❌ | width mismatch |
| 12 | int16 | int16 `MedalAchievementInfo: ausMedalQuestID count` | ✅ |  |
| 13 | byte | int32 `chair list count (ZArray<long>::_Alloc + DecodeBuffer with 4 * count bytes)` | ❌ | width mismatch |
| 14 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

ack: tool-limitation false positive — multiple overlapping causes: (1) loop linearization: the
bool-terminated pet list (SetMultiPetInfo do-while) is modelled as a flat sequence of individual
field writes rather than a loop body, causing the analyzer to misalign all subsequent fields;
(2) conditional sub-struct expansion: the optional taming mob block (if-guarded) and the wishList
count+loop are flattened independently, shifting alignment further; (3) version guard interaction:
the GMS-v87 monster book block (absent in v95) is correctly suppressed by the guard, but the
analyzer's flattened view emits extra fields from both guard branches; (4) method-boundary: the
MedalAchievementInfo::Decode sub-struct is modelled as two raw Decode calls (Decode4 medalId +
Decode2 questCount) which the flattener treats as inline with the parent sequence.

Manual cross-check against IDA CWvsContext::OnCharacterInfo (0xa05750) confirms the encoding is
correct for GMS v95: monster book block absent, pet list writes terminate correctly with bool=false,
MedalAchievementInfo writes int32(medalId)+short(0) matching Decode4+Decode2, chair list writes
int32(0) count + no buffer matching Decode4(0). No wire bug present. See _pending.md
"Known false positives — character misc-state bucket (Task 10)".
