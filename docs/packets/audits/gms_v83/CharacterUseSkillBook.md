# CharacterUseSkillBook (← `CWvsContext::SendSkillLearnItemUseRequest`)

- **IDA:** 0xa0a1b2
- **Atlas file:** `libs/atlas-packet/character/serverbound/use_skill_book.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime (get_update_time()) — v83 IDA-verified @0xa0a1b2, task-125 (function already named CWvsContext::SendSkillLearnItemUseRequest in the IDB; select_instance() is dead on the live MCP server ("Method 'select_instance' not found") so this entry was spliced by hand from a live decompile via the session-based idb tool, database=ce4ff298, filename MapleStory_dump.exe.i64, path ...\\GMS\\v83_Me\\...)` | ✅ |  |
| 1 | int16 | int16 `slot (a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (a3); gated a3/10000 in {228,229} (skill-book item prefix)` | ✅ |  |

