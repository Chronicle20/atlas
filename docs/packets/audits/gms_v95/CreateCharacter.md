# CreateCharacter (← `CLogin::SendNewCharPacket`)

- **IDA:** 0x5d7bd0
- **Atlas file:** `libs/atlas-packet/character/serverbound/create.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ✅ (opcode 0x16 path only; opcode 0x17 / bCharSale path is unmodeled — see Known gaps below)

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name (checked character name)` | ✅ |  |
| 1 | int32 | int32 `m_nCurSelectedRace (job/race index)` | ✅ |  |
| 2 | int16 | int16 `m_nCurSelectedSubJob (sub-job index)` | ✅ |  |
| 3 | int32 | int32 `GetSelectedAL(0) face` | ✅ |  |
| 4 | int32 | int32 `GetSelectedAL(1) hair` | ✅ |  |
| 5 | int32 | int32 `GetSelectedAL(2) hairColor` | ✅ |  |
| 6 | int32 | int32 `GetSelectedAL(3) skinColor` | ✅ |  |
| 7 | int32 | int32 `GetSelectedAL(4) top` | ✅ |  |
| 8 | int32 | int32 `GetSelectedAL(5) bottom` | ✅ |  |
| 9 | int32 | int32 `GetSelectedAL(6) shoes` | ✅ |  |
| 10 | int32 | int32 `GetSelectedAL(7) weapon` | ✅ |  |
| 11 | byte | byte `m_nGender` | ✅ |  |

## Known gaps

The atlas `CreateCharacter` decoder models only the standard character-creation path (opcode `0x16` / 22). IDA `CLogin::SendNewCharPacket@0x5d7bd0` has a second branch gated on `this->m_bCharSale == true` that emits a different shape:

- **Opcode** `0x17` (23) instead of 0x16
- **Wire** `EncodeStr(name) + Encode4(race) + Encode4(charSaleJob - 1) + 9 × Encode4(AL items)` — nine AL items (no SubJob, no gender)

Atlas does not currently decode opcode `0x17`. The CharSale flow (Cash Shop character creation promotion) is therefore not wired through atlas-login → atlas-character. Verdict ✅ is scoped to the opcode-22 path only; the opcode-23 path is an unmodeled gap deferred to a follow-up task. See `_pending.md` § "Still pending — character domain".
