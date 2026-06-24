# CharacterInfo (← `CWvsContext::OnCharacterInfo`)

- **IDA:** 0xb0aa6e
- **Atlas file:** `libs/atlas-packet/character/clientbound/info.go`
- **Variant:** JMS/v185
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
| 4 | byte | byte `bIsMarried (bool)` | ✅ |  |
| 5 | string | string `sCommunity (guild name)` | ✅ |  |
| 6 | string | string `sAlliance (alliance name)` | ✅ |  |
| 7 | byte | int32 `v32 (medal-related int32 — JMS reads Decode4 not Decode1)` | ❌ | width mismatch |
| 8 | byte | int32 `p (second medal int32)` | ❌ | width mismatch |
| 9 | int32 | byte `v26 (taming mob active flag)` | ❌ | width mismatch |
| 10 | string | byte `v7 (pet count for SetMultiPetInfo loop)` | ❌ | width mismatch |
| 11 | int32 | int32 `pet SN/id (SetMultiPetInfo per-pet)` | ✅ |  |
| 12 | int16 | string `pet name (SetMultiPetInfo per-pet)` | ❌ | width mismatch |
| 13 | int32 | byte `pet nLevel` | ❌ | width mismatch |
| 14 | int16 | int16 `pet nTameness` | ✅ |  |
| 15 | int32 | byte `pet nRepleteness` | ❌ | width mismatch |
| 16 | int32 | int16 `pet nPetSkill` | ❌ | width mismatch |
| 17 | int32 | int32 `pet item/expire int32` | ✅ |  |
| 18 | byte | byte `pet-loop terminator (next-pet bool)` | ✅ |  |
| 19 | byte | byte `taming mob active bool` | ✅ |  |
| 20 | int32 | int32 `taming mob field 1 (if active)` | ✅ |  |
| 21 | int32 | int32 `taming mob field 2 (if active)` | ✅ |  |
| 22 | int32 | int32 `taming mob field 3 (if active)` | ✅ |  |
| 23 | int32 | byte `wish list count (v12)` | ❌ | width mismatch |
| 24 | int32 | bytes `wish items (4 * count bytes)` | ✅ |  |
| 25 | int32 | int32 `monster book field 1 (SomethingMonsterBook)` | ✅ |  |
| 26 | int32 | int32 `monster book field 2` | ✅ |  |
| 27 | int16 | int32 `monster book field 3` | ❌ | width mismatch |
| 28 | int32 | int32 `monster book field 4` | ✅ |  |
| 29 | byte | int32 `monster book cover (mob id)` | ❌ | atlas: short — missing trailing field |
| 30 | byte | int32 `current medal id (MedalAchievementInfo::Decode)` | ❌ | atlas: short — missing trailing field |
| 31 | byte | int16 `medal/quest count` | ❌ | atlas: short — missing trailing field |
| 32 | byte | int16 `medal/quest id (per entry)` | ❌ | atlas: short — missing trailing field |
| 33 | byte | int32 `chair list count` | ❌ | atlas: short — missing trailing field |
| 34 | byte | bytes `chair list data (4 * count bytes)` | ❌ | atlas: short — missing trailing field |

