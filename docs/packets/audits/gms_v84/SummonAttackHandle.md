# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x7c99cf
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (v87[43] = owner cid on v84) — TryDoingAttackManual@0x7cafe8` | ✅ |  |
| 1 | int32 | int32 `~drInfo[0] (anti-hack obfuscated) — @0x7caffc` | ✅ |  |
| 2 | int32 | int32 `~drInfo[1] (anti-hack obfuscated) — @0x7cb010` | ✅ |  |
| 3 | int32 | int32 `updateTime — @0x7cb021` | ✅ |  |
| 4 | int32 | int32 `~drInfo[2] (anti-hack obfuscated) — @0x7cb035` | ✅ |  |
| 5 | int32 | int32 `~drInfo[3] (anti-hack obfuscated) — @0x7cb049` | ✅ |  |
| 6 | byte | byte `action byte (action&0x7F \| bLeft<<7) — @0x7cb069` | ✅ |  |
| 7 | int32 | int32 `dwKey (crc rand key) — @0x7cb0c5` | ✅ |  |
| 8 | int32 | int32 `crc32 — @0x7cb0ec` | ✅ |  |
| 9 | byte | byte `nMobCount — @0x7cb0fd` | ✅ |  |
| 10 | int16 | int16 `userX — @0x7cb144` | ✅ |  |
| 11 | int16 | int16 `userY — @0x7cb17a` | ✅ |  |
| 12 | int16 | int16 `summonX — @0x7cb1a5` | ✅ |  |
| 13 | int16 | int16 `summonY — @0x7cb1d1 (NO repeatSkillPoint follows on v84 — v95-only)` | ✅ |  |
| 14 | int32 | int32 `mob[i].mobId — @0x7cb256, loop nMobCount times` | ✅ |  |
| 15 | int32 | int32 `mob[i].templateId — @0x7cb2d9` | ✅ |  |
| 16 | byte | byte `mob[i].hitAction — @0x7cb2f0` | ✅ |  |
| 17 | byte | byte `mob[i].foreAction\|isLeft<<7 — @0x7cb31a` | ✅ |  |
| 18 | int16 | byte `mob[i].frameIdx — @0x7cb331` | ❌ | width mismatch |
| 19 | int16 | byte `mob[i].calcDamageStatIdx — @0x7cb356` | ❌ | width mismatch |
| 20 | int16 | int16 `mob[i].curX — @0x7cb385` | ✅ |  |
| 21 | int16 | int16 `mob[i].curY — @0x7cb3b5` | ✅ |  |
| 22 | int32 | int16 `mob[i].hitX — @0x7cb3e4` | ❌ | width mismatch |
| 23 | int32 | int16 `mob[i].hitY — @0x7cb414` | ❌ | width mismatch |
| 24 | int16 | int16 `mob[i].tDelay — @0x7cb42c` | ✅ |  |
| 25 | int32 | int32 `mob[i].damage — @0x7cb440` | ✅ |  |
| 26 | int32 | int32 `skillCRC — @0x7cb485` | ✅ |  |
| 27 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

