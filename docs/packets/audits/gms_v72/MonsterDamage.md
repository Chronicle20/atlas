# MonsterDamage (← `CMob::OnDamaged`)

- **IDA:** 0x61b933
- **Atlas file:** `libs/atlas-packet/monster/clientbound/damage.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | int32 `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

