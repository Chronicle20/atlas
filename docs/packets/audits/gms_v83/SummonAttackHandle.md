# SummonAttackHandle (← `CSummoned::TryDoingAttackManual`)

- **IDA:** 0x7a4d42
- **Atlas file:** `libs/atlas-packet/summon/serverbound/attack.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `summonId ([obj+0xAC] = owner cid on v83) — TryDoingAttackManual send@0x7a57f1` | ✅ |  |
| 1 | byte | int32 `updateTime (get_update_time()) — @0x7a57ff` | ❌ | width mismatch |
| 2 | byte | byte `action byte (action&0x7F \| bLeft<<7) — @0x7a5814` | ✅ |  |
| 3 | int32 | byte `nMobCount — @0x7a5820` | ❌ | width mismatch |
| 4 | int32 | int16 `userX — @0x7a5839` | ❌ | width mismatch |
| 5 | int16 | int16 `userY — @0x7a5850` | ✅ |  |
| 6 | int32 | int16 `summonX — @0x7a5867` | ❌ | width mismatch |
| 7 | byte | int16 `summonY — @0x7a587e` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `mob[i].mobId — @0x7a58aa, loop nMobCount times` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 9 | byte | int32 `mob[i].templateId — @0x7a58dc` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 10 | byte | byte `mob[i].hitAction — @0x7a58ea` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 11 | byte | byte `mob[i].foreAction\|isLeft<<7 — @0x7a5905` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 12 | byte | byte `mob[i].frameIdx — @0x7a5913` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 13 | byte | byte `mob[i].calcDamageStatIdx — @0x7a5923` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 14 | byte | int16 `mob[i].curX — @0x7a5939` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 15 | byte | int16 `mob[i].curY — @0x7a5950` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 16 | byte | int16 `mob[i].hitX — @0x7a5966` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 17 | byte | int16 `mob[i].hitY — @0x7a597d` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 18 | byte | int16 `mob[i].tDelay — @0x7a598c` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 19 | byte | int32 `mob[i].damage — @0x7a5997` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 20 | byte | int32 `skillCRC — @0x7a59bd` | ❌ | atlas: short — missing trailing field |

