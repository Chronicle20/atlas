# EffectSkillUse (← `CUser::OnEffect`)

- **IDA:** 0x9377d9
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_skill_use.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path); absent on self-effect opcode` | ❌ | width mismatch |
| 1 | int32 | byte `nMode — sub-op byte dispatching to 27+ effect branches (case 0..26); sub-op enum not modeled by pipeline` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

