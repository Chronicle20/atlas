# MonsterCarnivalLeave (← `CField_MonsterCarnival::OnShowMemberOutMsg`)

- **IDA:** 0x5488ef
- **Atlas file:** `libs/atlas-packet/monster/carnival/clientbound/monster_carnival_leave.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `leader (==6 => leader-quit-appointed message variant)` | ✅ |  |
| 1 | byte | byte `team (team color selector: !=0 => MAPLE_BLUE, 0 => MAPLE_RED)` | ✅ |  |
| 2 | string | string `name (quitting character name)` | ✅ |  |

