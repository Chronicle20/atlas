# EffectSimple (← `CUser::OnEffect`)

- **IDA:** 0x8f9a70
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path); absent on self-effect opcode` | ❌ | width mismatch |
| 1 | byte | byte `nMode — sub-op byte dispatching to 16+ effect branches (case 0..15+); sub-op enum not modeled by pipeline` | ❌ | atlas: short — missing trailing field |

---

**ack: sub-op enum drift — deferred to `_pending.md § Sub-op enum drift — character domain`**

This report represents the effect.go file (`CUser::OnEffect` foreign/self split).
Row 0 mismatch: IDA export describes the foreign path (characterId prefix),
but `EffectSimple` is a self-effect struct (no characterId). Row 1: the mode
byte's sub-op tree (16+ cases) cannot be modeled by the pipeline. All effect
structs (effect.go, effect_quest.go, effect_skill_use.go) are deferred to
Phase 3 for per-mode case-arm verification.
