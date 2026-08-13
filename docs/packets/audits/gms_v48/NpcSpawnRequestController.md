# NpcSpawnRequestController (← `CNpcPool::OnNpcChangeController`)

- **IDA:** 0x56d617
- **Atlas file:** `libs/atlas-packet/npc/clientbound/spawn_request_controller.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | byte | byte `` | ✅ |  |
| 6 | int16 | int16 `` | ✅ |  |
| 7 | int16 | int16 `` | ✅ |  |
| 8 | int16 | int16 `` | ✅ |  |
| 9 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

