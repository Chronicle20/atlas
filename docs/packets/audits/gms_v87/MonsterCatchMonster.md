# MonsterCatchMonster (← `CMob::OnCatchEffect`)

- **IDA:** 0x6a8585
- **Atlas file:** `libs/atlas-packet/monster/clientbound/catch_monster.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

