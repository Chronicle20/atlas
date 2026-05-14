# EffectSimple (← `CUser::OnEffect`)

- **IDA:** 0x9b1ef0
- **Atlas file:** `libs/atlas-packet/character/clientbound/effect.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `characterId — read by CUserPool::OnUserRemotePacket before dispatch (foreign path)` | ❌ | width mismatch |
| 1 | byte | byte `nMode — sub-op byte (16+ effect branches); sub-op enum not modeled by pipeline` | ❌ | atlas: short — missing trailing field |

