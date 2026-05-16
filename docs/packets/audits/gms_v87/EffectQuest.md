# EffectQuest (← `CUser::OnEffect`)

- **IDA:** 0x9b1ef0
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_quest.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path)` | ❌ | width mismatch |
| 1 | byte | byte `nMode — sub-op byte (16+ effect branches); sub-op enum not modeled by pipeline` | ✅ |  |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

