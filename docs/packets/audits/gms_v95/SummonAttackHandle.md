# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x751240
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid (m_dwSummonedID) — TryDoingAttackManual@0x752287` | ✅ |  |
| 1 | int32 | int32 `~drInfo[0] (anti-hack obfuscated) — @0x75229b` | ✅ |  |
| 2 | int32 | int32 `~drInfo[1] (anti-hack obfuscated) — @0x7522af` | ✅ |  |
| 3 | int32 | int32 `updateTime — @0x7522c0` | ✅ |  |
| 4 | int32 | int32 `~drInfo[2] (anti-hack obfuscated) — @0x7522d4` | ✅ |  |
| 5 | int32 | int32 `~drInfo[3] (anti-hack obfuscated) — @0x7522e8` | ✅ |  |
| 6 | byte | byte `action byte (action&0x7F \| bLeft<<7) — @0x752302` | ✅ |  |
| 7 | int32 | int32 `dwKey (crc rand key) — @0x752325` | ✅ |  |
| 8 | int32 | int32 `crc32 — @0x75234c` | ✅ |  |
| 9 | byte | byte `nMobCount — @0x75235c` | ✅ |  |
| 10 | int16 | int16 `userX — @0x7523a5` | ✅ |  |
| 11 | int16 | int16 `userY — @0x7523dd` | ✅ |  |
| 12 | int16 | int16 `summonX — @0x75240a` | ✅ |  |
| 13 | int16 | int16 `summonY — @0x752438` | ✅ |  |
| 14 | int32 | int32 `repeatSkillPoint — @0x752450` | ✅ |  |
| 15 | int32 | int32 `mob[i].mobId — @0x7524ac, loop nMobCount times` | ✅ |  |
| 16 | int32 | int32 `mob[i].templateId — @0x7524cc` | ✅ |  |
| 17 | int16 | byte `mob[i].hitAction — @0x7524e2` | ❌ | width mismatch |
| 18 | int16 | byte `mob[i].foreAction\|isLeft<<7 — @0x75250c` | ❌ | width mismatch |
| 19 | int16 | byte `mob[i].frameIdx — @0x752522` | ❌ | width mismatch |
| 20 | int32 | byte `mob[i].calcDamageStatIdx — @0x75253b` | ❌ | width mismatch |
| 21 | int32 | int16 `mob[i].hitX — @0x75256c` | ❌ | width mismatch |
| 22 | int16 | int16 `mob[i].hitY — @0x7525a0` | ✅ |  |
| 23 | int16 | int16 `mob[i].posX — @0x7525d3` | ✅ |  |
| 24 | int16 | int16 `mob[i].posY — @0x752607` | ✅ |  |
| 25 | int16 | int16 `mob[i].tDelay — @0x75261d` | ✅ |  |
| 26 | int32 | int32 `mob[i].damage — @0x752632` | ✅ |  |
| 27 | int16 | int32 `skillCRC — @0x75266f` | ❌ | width mismatch |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

