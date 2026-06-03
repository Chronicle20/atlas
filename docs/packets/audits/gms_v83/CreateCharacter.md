# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x5f7e7a
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name (checked character name)` | ✅ |  |
| 1 | int32 | int32 `m_nCurSelectedRace (job/race index) — NOTE: v83 has NO Encode2(subJob) before the AL items` | ✅ |  |
| 2 | int32 | int32 `GetSelectedAL(0) face` | ✅ |  |
| 3 | int32 | int32 `GetSelectedAL(1) hair` | ✅ |  |
| 4 | int32 | int32 `GetSelectedAL(2) hairColor` | ✅ |  |
| 5 | int32 | int32 `GetSelectedAL(3) skinColor` | ✅ |  |
| 6 | int32 | int32 `GetSelectedAL(4) top` | ✅ |  |
| 7 | int32 | int32 `GetSelectedAL(5) bottom` | ✅ |  |
| 8 | int32 | int32 `GetSelectedAL(6) shoes` | ✅ |  |
| 9 | int32 | int32 `GetSelectedAL(7) weapon` | ✅ |  |
| 10 | byte | byte `m_nGender` | ✅ |  |

