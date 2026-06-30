# SummonAttack (← `CSummonedPool::OnAttack`)

- **IDA:** 0x71cfe9
- **Atlas file:** `libs/atlas-packet/summon/clientbound/attack.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket@0x8c8c84 (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid — summon cluster dispatcher sub_892500@0x89253f (read before leaf dispatch)` | ✅ |  |
| 2 | byte | byte `action byte: bLeft\|direction (sub_71CFE9@0x71d06f); no leading charLevel on v79` | ✅ |  |
| 3 | byte | byte `count (@0x71d08b)` | ✅ |  |
| 4 | byte | int32 `monsterOid (@0x71d0bf)` | ❌ | width mismatch |
| 5 | int32 | byte `byte 6 (@0x71d0cd)` | ❌ | width mismatch |
| 6 | byte | int32 `damage (@0x71d0e0)` | ❌ | width mismatch |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

