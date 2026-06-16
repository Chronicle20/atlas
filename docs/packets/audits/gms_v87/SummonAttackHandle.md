# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x7f6666
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (v86[43] = owner cid on v87) — TryDoingAttackManual@0x7f7c7f` | ✅ |  |
| 1 | int32 | int32 `~drInfo[0] (anti-hack obfuscated) — @0x7f7c93` | ✅ |  |
| 2 | int32 | int32 `~drInfo[1] (anti-hack obfuscated) — @0x7f7ca7` | ✅ |  |
| 3 | int32 | int32 `updateTime — @0x7f7cb8` | ✅ |  |
| 4 | int32 | int32 `~drInfo[2] (anti-hack obfuscated) — @0x7f7ccc` | ✅ |  |
| 5 | int32 | int32 `~drInfo[3] (anti-hack obfuscated) — @0x7f7ce0` | ✅ |  |
| 6 | byte | byte `action byte (action&0x7F \| bLeft<<7) — @0x7f7d00` | ✅ |  |
| 7 | int32 | int32 `dwKey (crc rand key) — @0x7f7d5c` | ✅ |  |
| 8 | int32 | int32 `crc32 — @0x7f7d83` | ✅ |  |
| 9 | byte | byte `nMobCount — @0x7f7d94` | ✅ |  |
| 10 | int16 | int16 `userX — @0x7f7ddb` | ✅ |  |
| 11 | int16 | int16 `userY — @0x7f7e11` | ✅ |  |
| 12 | int16 | int16 `summonX — @0x7f7e3c` | ✅ |  |
| 13 | int16 | int16 `summonY — @0x7f7e68 (NO repeatSkillPoint follows on v87 — v95-only)` | ✅ |  |
| 14 | int32 | int32 `mob[i].mobId — @0x7f7eed, loop nMobCount times` | ✅ |  |
| 15 | int32 | int32 `mob[i].templateId — @0x7f7f70` | ✅ |  |
| 16 | byte | byte `mob[i].hitAction — @0x7f7f87` | ✅ |  |
| 17 | byte | byte `mob[i].foreAction\|isLeft<<7 — @0x7f7fb1` | ✅ |  |
| 18 | int16 | byte `mob[i].frameIdx — @0x7f7fc8` | ❌ | width mismatch |
| 19 | int16 | byte `mob[i].calcDamageStatIdx — @0x7f7fed` | ❌ | width mismatch |
| 20 | int16 | int16 `mob[i].curX — @0x7f801c` | ✅ |  |
| 21 | int16 | int16 `mob[i].curY — @0x7f804c` | ✅ |  |
| 22 | int32 | int16 `mob[i].hitX — @0x7f807b` | ❌ | width mismatch |
| 23 | int32 | int16 `mob[i].hitY — @0x7f80ab` | ❌ | width mismatch |
| 24 | int16 | int16 `mob[i].tDelay — @0x7f80c3` | ✅ |  |
| 25 | int32 | int32 `mob[i].damage — @0x7f80d7` | ✅ |  |
| 26 | int32 | int32 `skillCRC — @0x7f811c` | ✅ |  |
| 27 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

