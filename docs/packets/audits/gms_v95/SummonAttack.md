# SummonAttack (← `CSummonedPool::OnAttack`)

- **IDA:** 0x759860
- **Atlas file:** `../../libs/atlas-packet/summon/clientbound/attack.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `charId — read by CSummonedPool::OnPacket@0x75ac70 before dispatch` | ✅ |  |
| 1 | int32 | int32 `oid (dwSummonedID) — CUser::OnSummonedAttack@0x8e3922` | ✅ |  |
| 2 | byte | byte `charLevel (m_nCharLevel) — CSummoned::OnAttack@0x753413; atlas writes fixed 0` | ✅ |  |
| 3 | byte | byte `action byte (low7=action, bit7=bLeft direction) — CSummoned::OnAttack@0x75341e; atlas 'direction'` | ✅ |  |
| 4 | byte | byte `count (mob count) — CSummoned::OnAttack@0x75347c` | ✅ |  |
| 5 | int32 | int32 `target[i].monsterOid — CSummoned::OnAttack@0x7534b2, loop count times` | ✅ |  |
| 6 | byte | byte `target[i].byte (only when monsterOid!=0) — CSummoned::OnAttack@0x7534ca; atlas writes fixed 6` | ✅ |  |
| 7 | int32 | int32 `target[i].damage (only when monsterOid!=0) — CSummoned::OnAttack@0x7534d2` | ✅ |  |
| 8 | byte | byte `trailing byte — CSummoned::OnAttack@0x7534e1 (consumed after the target loop)` | ✅ |  |

