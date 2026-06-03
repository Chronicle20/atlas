# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x62f603
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name` | ✅ |  |
| 1 | int32 | int32 `m_nCurSelectedRace (job/race index)` | ✅ |  |
| 2 | int16 | int16 `m_nCurSelectedSubJob (sub-job; literal 0 in v87 — present but zero-forced)` | ✅ |  |
| 3 | int32 | int32 `GetSelectedAL(0) face` | ✅ |  |
| 4 | int32 | int32 `GetSelectedAL(1) hair` | ✅ |  |
| 5 | int32 | int32 `GetSelectedAL(2) hairColor` | ✅ |  |
| 6 | int32 | int32 `GetSelectedAL(3) skinColor` | ✅ |  |
| 7 | int32 | int32 `GetSelectedAL(4) top` | ✅ |  |
| 8 | int32 | int32 `GetSelectedAL(5) bottom` | ✅ |  |
| 9 | int32 | int32 `GetSelectedAL(6) shoes` | ✅ |  |
| 10 | int32 | int32 `GetSelectedAL(7) weapon` | ✅ |  |
| 11 | byte | byte `m_nGender` | ✅ |  |

