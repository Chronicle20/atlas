# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x751240
- **Atlas file:** `../../libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `oid (m_dwSummonedID) — TryDoingAttackManual@0x752287` | ✅ |  |
| 1 | byte | int32 `~drInfo[0] (anti-hack obfuscated) — @0x75229b` | ❌ | width mismatch |
| 2 | byte | int32 `~drInfo[1] (anti-hack obfuscated) — @0x7522af` | ❌ | width mismatch |
| 3 | int32 | int32 `updateTime — @0x7522c0` | ✅ |  |
| 4 | int16 | int32 `~drInfo[2] (anti-hack obfuscated) — @0x7522d4` | ❌ | width mismatch |
| 5 | int32 | int32 `~drInfo[3] (anti-hack obfuscated) — @0x7522e8` | ✅ |  |
| 6 | byte | byte `action byte (action&0x7F \| bLeft<<7) — @0x752302` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `dwKey (crc rand key) — @0x752325` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `crc32 — @0x75234c` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `nMobCount — @0x75235c` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int16 `userX — @0x7523a5` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int16 `userY — @0x7523dd` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int16 `summonX — @0x75240a` | ❌ | atlas: short — missing trailing field |
| 13 | byte | int16 `summonY — @0x752438` | ❌ | atlas: short — missing trailing field |
| 14 | byte | int32 `repeatSkillPoint — @0x752450` | ❌ | atlas: short — missing trailing field |
| 15 | byte | int32 `mob[i].mobId — @0x7524ac, loop nMobCount times` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 16 | byte | int32 `mob[i].templateId — @0x7524cc` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 17 | byte | byte `mob[i].hitAction — @0x7524e2` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 18 | byte | byte `mob[i].foreAction\|isLeft<<7 — @0x75250c` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 19 | byte | byte `mob[i].frameIdx — @0x752522` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 20 | byte | byte `mob[i].calcDamageStatIdx — @0x75253b` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 21 | byte | int16 `mob[i].hitX — @0x75256c` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 22 | byte | int16 `mob[i].hitY — @0x7525a0` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 23 | byte | int16 `mob[i].posX — @0x7525d3` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 24 | byte | int16 `mob[i].posY — @0x752607` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 25 | byte | int16 `mob[i].tDelay — @0x75261d` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 26 | byte | int32 `mob[i].damage — @0x752632` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 27 | byte | int32 `skillCRC — @0x75266f` | ❌ | atlas: short — missing trailing field |

