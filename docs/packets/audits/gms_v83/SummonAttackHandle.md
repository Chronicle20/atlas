# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x7a4d42
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId ([obj+0xAC] = owner cid on v83) — TryDoingAttackManual send@0x7a57f1` | ✅ |  |
| 1 | int32 | int32 `updateTime (get_update_time()) — @0x7a57ff` | ✅ |  |
| 2 | int32 | byte `action byte (action&0x7F \| bLeft<<7) — @0x7a5814` | ❌ | width mismatch |
| 3 | int32 | byte `nMobCount — @0x7a5820` | ❌ | width mismatch |
| 4 | int32 | int16 `userX — @0x7a5839` | ❌ | width mismatch |
| 5 | int32 | int16 `userY — @0x7a5850` | ❌ | width mismatch |
| 6 | byte | int16 `summonX — @0x7a5867` | ❌ | width mismatch |
| 7 | int32 | int16 `summonY — @0x7a587e` | ❌ | width mismatch |
| 8 | int32 | int32 `mob[i].mobId — @0x7a58aa, loop nMobCount times` | ✅ |  |
| 9 | byte | int32 `mob[i].templateId — @0x7a58dc` | ❌ | width mismatch |
| 10 | int16 | byte `mob[i].hitAction — @0x7a58ea` | ❌ | width mismatch |
| 11 | int16 | byte `mob[i].foreAction\|isLeft<<7 — @0x7a5905` | ❌ | width mismatch |
| 12 | int16 | byte `mob[i].frameIdx — @0x7a5913` | ❌ | width mismatch |
| 13 | int16 | byte `mob[i].calcDamageStatIdx — @0x7a5923` | ❌ | width mismatch |
| 14 | int32 | int16 `mob[i].curX — @0x7a5939` | ❌ | width mismatch |
| 15 | int32 | int16 `mob[i].curY — @0x7a5950` | ❌ | width mismatch |
| 16 | int16 | int16 `mob[i].hitX — @0x7a5966` | ✅ |  |
| 17 | int16 | int16 `mob[i].hitY — @0x7a597d` | ✅ |  |
| 18 | int16 | int16 `mob[i].tDelay — @0x7a598c` | ✅ |  |
| 19 | int32 | int32 `mob[i].damage — @0x7a5997` | ✅ |  |
| 20 | int32 | int32 `skillCRC — @0x7a59bd` | ✅ |  |
| 21 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 31 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 32 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

