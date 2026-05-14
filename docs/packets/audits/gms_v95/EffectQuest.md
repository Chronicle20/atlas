# EffectQuest (← `CUser::OnEffect`)

- **IDA:** 0x8f9a70
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect_quest.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path); absent on self-effect opcode` | ❌ | width mismatch |
| 1 | byte | byte `nMode — sub-op byte dispatching to 16+ effect branches (case 0..15+); sub-op enum not modeled by pipeline` | ✅ |  |
| 2 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

---

**ack: sub-op enum drift — deferred to `_pending.md § Sub-op enum drift — character domain`**

This report represents the effect_quest.go file. Same root cause as EffectSimple:
the IDA export describes the foreign path while EffectQuest has no characterId
(row 0), and the mode byte's sub-op tree cannot be modeled (rows 2+). The
"atlas: extra" rows are quest-specific fields (reward count, message, nEffect)
that appear beyond the pipeline's view of the IDA flat sequence. Deferred to
Phase 3 for per-mode case-arm verification.
