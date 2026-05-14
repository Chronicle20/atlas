# EffectSkillUse (← `CUser::OnEffect`)

- **IDA:** 0x8f9a70
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_skill_use.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path); absent on self-effect opcode` | ❌ | width mismatch |
| 1 | int32 | byte `nMode — sub-op byte dispatching to 16+ effect branches (case 0..15+); sub-op enum not modeled by pipeline` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

---

**ack: sub-op enum drift — deferred to `_pending.md § Sub-op enum drift — character domain`**

This report represents the effect_skill_use.go file. Row 1 shows int32 vs byte
mismatch: `EffectSkillUse.Encode` writes skillId (int32) after the mode byte,
while the IDA flat sequence only has 2 fields and the second is the mode byte
itself. The pipeline cannot model the sub-op dispatch tree. Also note row 0:
IDA expects characterId (int32) but EffectSkillUse is a self-effect struct
(no characterId prefix). All mismatches are sub-op enum drift artifacts.
Deferred to Phase 3 for per-mode case-arm verification.
