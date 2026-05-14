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

