# ReactorSpawn (← `CReactorPool::OnReactorEnterField`)

- **IDA:** 0x69207c
- **Atlas file:** `libs/atlas-packet/reactor/clientbound/spawn.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | string | byte `` | ❌ | atlas: extra — client never reads this field |

