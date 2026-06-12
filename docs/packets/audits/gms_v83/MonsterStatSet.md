# MonsterStatSet (← `CMob::OnStatSet`)

- **IDA:** 0x66c301
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/stat.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | bytes | bytes `uFlagSet: 16-byte UINT128 stat mask` | ✅ |  |
| 2 | int16 | bytes `per-stat body via CMob::ProcessStatSet` | ✅ |  |
| 3 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

