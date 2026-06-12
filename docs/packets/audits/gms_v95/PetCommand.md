# PetCommand (← `CPet::ParseCommand`)

- **IDA:** 0x6a3cc0
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/command.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes — _LARGE_INTEGER, m_liPetLockerSN)` | ✅ |  |
| 1 | byte | byte `commandWithName (bCommandWithName bool)` | ✅ |  |
| 2 | byte | byte `command` | ✅ |  |

