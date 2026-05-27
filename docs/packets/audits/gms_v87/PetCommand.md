# PetCommand (← `CPet::ParseCommand`)

- **IDA:** 0x748a35
- **Atlas file:** `libs/atlas-packet/pet/serverbound/command.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `petLockerSN (8 bytes)` | ❌ | width mismatch |
| 1 | byte | byte `command mode` | ✅ |  |
| 2 | byte | byte `reaction index` | ✅ |  |
| 3 | byte | byte `success flag` | ❌ | atlas: short — missing trailing field |

