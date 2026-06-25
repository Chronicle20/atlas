# HealOverTime (← `CWvsContext::SendStatChangeRequest`)

- **IDA:** 0xb054d6
- **Atlas file:** `libs/atlas-packet/character/serverbound/heal_over_time.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time())` | ✅ |  |
| 1 | int32 | int32 `val (constant 0x1400 = 5120 in the sitdown-heal path)` | ✅ |  |
| 2 | int16 | int16 `nHP (uint16 HP recovery amount)` | ✅ |  |
| 3 | int16 | int16 `nMP (uint16 MP recovery amount)` | ✅ |  |
| 4 | byte | byte `nOption (0=normal, 2=sitting)` | ✅ |  |
| 5 | int32 | int32 `jms-only trailing validation dword (dword_CDA4F8)` | ✅ |  |

