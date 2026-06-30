# SummonSpawn (← `CSummonedPool::OnCreated`)

- **IDA:** 0x89268a
- **Atlas file:** `libs/atlas-packet/summon/clientbound/spawn.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `cid/ownerId — CUserPool::OnUserCommonPacket@0x8c8c84 (consumed upstream before dispatch)` | ✅ |  |
| 1 | int32 | int32 `oid (sub_89268A@0x8926a5)` | ✅ |  |
| 2 | int32 | int32 `skillId (@0x8926af)` | ✅ |  |
| 3 | byte | byte `charLevel (@0x8926b9); no SLV byte on v79` | ✅ |  |
| 4 | byte | int16 `x (blob sub_719F7B@0x719f92)` | ❌ | width mismatch |
| 5 | int16 | int16 `y (@0x719f9f)` | ✅ |  |
| 6 | int16 | byte `stance (@0x719fac)` | ❌ | width mismatch |
| 7 | byte | int16 `foothold (@0x719faf)` | ❌ | width mismatch |
| 8 | int16 | byte `movementType (@0x719fc3)` | ❌ | width mismatch |
| 9 | byte | byte `!puppet (@0x719fda)` | ✅ |  |
| 10 | byte | byte `!animated (@0x71a001)` | ✅ |  |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

