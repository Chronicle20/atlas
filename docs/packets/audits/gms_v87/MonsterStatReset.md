# MonsterStatReset (← `CMob::OnStatReset`)

- **IDA:** 0x6a72ef
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/stat.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | bytes | bytes `uFlagReset: 16-byte UINT128 stat mask` | ✅ |  |
| 2 | int16 | bytes `per-stat reset body via CMob::ProcessStatReset` | ✅ |  |
| 3 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 4 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

