# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x5f7e7a
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name (checked character name)` | ✅ |  |
| 1 | int32 | int32 `m_nCurSelectedRace (job/race index) — NOTE: v83 has NO Encode2(subJob) before the AL items` | ✅ |  |
| 2 | int16 | int32 `GetSelectedAL(0) face` | ❌ | width mismatch |
| 3 | int32 | int32 `GetSelectedAL(1) hair` | ✅ |  |
| 4 | int32 | int32 `GetSelectedAL(2) hairColor` | ✅ |  |
| 5 | int32 | int32 `GetSelectedAL(3) skinColor` | ✅ |  |
| 6 | int32 | int32 `GetSelectedAL(4) top` | ✅ |  |
| 7 | int32 | int32 `GetSelectedAL(5) bottom` | ✅ |  |
| 8 | int32 | int32 `GetSelectedAL(6) shoes` | ✅ |  |
| 9 | int32 | int32 `GetSelectedAL(7) weapon` | ✅ |  |
| 10 | int32 | byte `m_nGender` | ❌ | width mismatch |
| 11 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

