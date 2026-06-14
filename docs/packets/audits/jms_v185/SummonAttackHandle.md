# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x824a81
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId (owner cid on jms185) — emit virtualized; mirrors v95 TryDoingAttackManual@0x752287` | ✅ |  |
| 1 | int32 | int32 `~drInfo[0] (anti-hack; from DR_check@0x826202) — emit virtualized; mirrors v95@0x75229b` | ✅ |  |
| 2 | int32 | int32 `~drInfo[1] (anti-hack; from DR_check@0x826202) — emit virtualized; mirrors v95@0x7522af` | ✅ |  |
| 3 | int32 | int32 `updateTime — emit virtualized; mirrors v95@0x7522c0` | ✅ |  |
| 4 | int32 | int32 `~drInfo[2] (anti-hack; from DR_check@0x826202) — emit virtualized; mirrors v95@0x7522d4` | ✅ |  |
| 5 | int32 | int32 `~drInfo[3] (anti-hack; from DR_check@0x826202) — emit virtualized; mirrors v95@0x7522e8` | ✅ |  |
| 6 | byte | byte `action byte (action&0x7F \| bLeft<<7) — emit virtualized; mirrors v95@0x752302` | ✅ |  |
| 7 | int32 | int32 `dwKey (crc rand key) — emit virtualized; mirrors v95@0x752325` | ✅ |  |
| 8 | int32 | int32 `crc32 — emit virtualized; mirrors v95@0x75234c` | ✅ |  |
| 9 | byte | byte `nMobCount — emit virtualized; mirrors v95@0x75235c` | ✅ |  |
| 10 | int16 | int16 `userX — emit virtualized; mirrors v95@0x7523a5` | ✅ |  |
| 11 | int16 | int16 `userY — emit virtualized; mirrors v95@0x7523dd` | ✅ |  |
| 12 | int16 | int16 `summonX — emit virtualized; mirrors v95@0x75240a` | ✅ |  |
| 13 | int16 | int16 `summonY — emit virtualized; mirrors v95@0x752438` | ✅ |  |
| 14 | int32 | int32 `repeatSkillPoint (post-v95 envelope tail; lineage-inferred for jms185) — emit virtualized; mirrors v95@0x752450` | ✅ |  |
| 15 | int32 | int32 `mob[i].mobId — emit virtualized; mirrors v95@0x7524ac, loop nMobCount times` | ✅ |  |
| 16 | int32 | int32 `mob[i].templateId — mirrors v95@0x7524cc` | ✅ |  |
| 17 | int16 | byte `mob[i].hitAction — mirrors v95@0x7524e2` | ❌ | width mismatch |
| 18 | int16 | byte `mob[i].foreAction\|isLeft<<7 — mirrors v95@0x75250c` | ❌ | width mismatch |
| 19 | int16 | byte `mob[i].frameIdx — mirrors v95@0x752522` | ❌ | width mismatch |
| 20 | int32 | byte `mob[i].calcDamageStatIdx — mirrors v95@0x75253b` | ❌ | width mismatch |
| 21 | int32 | int16 `mob[i].curX — mirrors v95@0x75256c` | ❌ | width mismatch |
| 22 | int16 | int16 `mob[i].curY — mirrors v95@0x7525a0` | ✅ |  |
| 23 | int16 | int16 `mob[i].hitX — mirrors v95@0x7525d3` | ✅ |  |
| 24 | int16 | int16 `mob[i].hitY — mirrors v95@0x752607` | ✅ |  |
| 25 | int16 | int16 `mob[i].tDelay — mirrors v95@0x75261d` | ✅ |  |
| 26 | int32 | int32 `mob[i].damage — mirrors v95@0x752632` | ✅ |  |
| 27 | int16 | int32 `skillCRC — emit virtualized; mirrors v95@0x75266f` | ❌ | width mismatch |
| 28 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

