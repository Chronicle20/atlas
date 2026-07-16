# SummonDamageHandle (← `CSummoned::SetDamaged`)

- **IDA:** 0x71c7a7
- **Atlas file:** `libs/atlas-packet/summon/serverbound/damage.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId = *(this+42) (sub_71C7A7 COutPacket(173)@0x71c9e9; Encode4@0x71c9fe)` | ✅ |  |
| 1 | byte | byte `attackIdx (@0x71ca25)` | ✅ |  |
| 2 | int32 | int32 `damage (@0x71ca2e)` | ✅ |  |
| 3 | byte | int32 `monsterTemplateId (@0x71ca4b)` | ❌ | width mismatch |
| 4 | int32 | byte `dir<0 flag (@0x71ca5b)` | ❌ | width mismatch |
| 5 | int32 | byte `0xFE sentinel (@0x71ca0f)` | ❌ | width mismatch |
| 6 | byte | int32 `damage (@0x71ca18)` | ❌ | width mismatch |

