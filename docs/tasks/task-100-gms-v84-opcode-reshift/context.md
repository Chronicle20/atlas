# task-100 — complete the gms_v84 opcode-table reshift

## Problem
`docs/packets/registry/gms_v84.yaml` has **46 duplicate `(opcode, direction)` pairs**. The CSVs have no v84 column, so v84 was seeded by copying v83 opcodes; only ops that a prior task touched got re-derived. In **every** duplicate pair: one row is `manual`/`ida-discovered` (the REAL, v84-IDB-verified op at its correct opcode) and the other is `csv-import` (a DIFFERENT op still carrying its **stale v83 opcode**, coincidentally colliding). v84's opcode table is shifted vs v83 (≈ +3 low range, +6/+7 high range, +4 door/wedding/family) — the exact shift varies by neighborhood, so it must be **read from the IDB, not derived**.

## Goal
Every v84 registry opcode matches the v84 IDB dispatch; **zero duplicate `(opcode,direction)` pairs**; provenance upgraded `csv-import` → `ida-discovered` (with `ida.address`). `tools/packet-audit matrix --check` exit 0, no regression. No task-096 CField op is in any duplicate (already clean) — this is the NON-CField remainder.

## Method (per stale `csv-import` op)
1. The `manual`/`ida-discovered` row in each pair is CORRECT — leave it.
2. For the `csv-import` row: its `fname` names the op. Find that function in the **v95 IDB (PDB-named — port 13340)** to confirm identity + the v95 opcode; then find/decompile the **v84** equivalent (port 13337) and READ its real v84 opcode (clientbound: the family recv dispatcher case → handler; serverbound: the `COutPacket(N)` build at the send-site). Cross-check the v83 value (port 13342) for the shift progression.
3. **Name the v84 function** to the canonical demangled name (mangled MSVC symbol) and **`idb_save`** — so the v84 IDB becomes a properly-named reference like v95's PDB (and future exports resolve by name). This is a standing directive for this task.
4. Set the registry row's opcode to the real v84 value (`provenance: ida-discovered`, `ida.address` = the dispatcher/send-site addr, citation in `note`).

## Ports (confirm via list_instances; not stable across sessions)
v83=13342, v84=13337 (GMS_v84.1_U_DEVM — the IDB to NAME+SAVE), v87=13341, v95=13340 (PDB reference), jms=13339.

## Work-list — the 46 stale `csv-import` ops (grouped by family/dispatcher)
Each entry = the op whose v84 opcode is stale and must be re-read. (The colliding correct op is in parens.)

### CWvsContext clientbound
- GUILD_OPERATION (vs BUDDYLIST@65) · SET_POTION_DISCOUNT_RATE (vs CASH_PET_FOOD_RESULT@78) · HOUR_CHANGED (vs MONSTER_BOOK_SET_CARD@85) · MINIMAP_ON_OFF (vs MONSTER_BOOK_SET_COVER@86)
### CWvsContext serverbound (family ops)
- ADD_FAMILY (vs ALLIANCE_OPERATION@147) · SEPARATE_FAMILY_BY_SENIOR (vs DENY_ALLIANCE_REQUEST@148)
### CField clientbound/serverbound (stale leftovers — dedupe vs task-096's correct rows)
- WHISPER serverbound SendLocationWhisper (vs ADMIN_CHAT@120) · RING_ACTION CEngageDlg::SetRet (vs USE_DOOR@137) · FIELD_EFFECT (vs WHISPER@138) · SPOUSE_CHAT csv row (vs FORCED_MAP_EQUIP@136) · SET_FIELD CStage::OnSetField (vs SCRIPT_PROGRESS_MESSAGE@125)
### CUser / CUserRemote clientbound (attack/skill/chat block)
- CHATTEXT1 (vs SPAWN_PLAYER@163) · CHALKBOARD (vs REMOVE_PLAYER_FROM_MAP@164) · ENERGY_ATTACK (vs MOVE_PLAYER@189) · SKILL_EFFECT (vs CLOSE_RANGE_ATTACK@190) · CANCEL_SKILL_EFFECT (vs RANGED_ATTACK@191) · DAMAGE_PLAYER (vs MAGIC_ATTACK@192) · GUILD_NAME_CHANGED (vs SHOW_FOREIGN_EFFECT@202) · GUILD_MARK_CHANGED (vs GIVE_FOREIGN_BUFF@203) · RANDOM_EMOTION (vs LOCK_UI@226) · RESIGN_QUEST_RETURN (vs DISABLE_UI@227) · RADIO_SCHEDULE (vs TALK_GUIDE@229)
### CPet / CSummoned serverbound
- MOVE_SUMMON (vs PET_LOOT@175) · SUMMON_ATTACK (vs PET_AUTO_POT@176) · DAMAGE_SUMMON (vs PET_EXCLUDE_ITEMS@177)
### CMob / CNpc clientbound + serverbound
- NPC_SPECIAL_ACTION sb (vs MONSTER_BOMB@198) · DAMAGE_MONSTER (vs MOVE_MONSTER_RESPONSE@246) · REMOVE_NPC (vs CATCH_MONSTER_WITH_ITEM@258) · UPDATE_LIMITED_INFO (vs MOB_SKILL_DELAY@261) · NPC_SPECIAL_ACTION cl (vs MOB_ATTACKED_BY_MOB@262)
### CEmployeePool / CTownPortalPool / CNpcPool clientbound
- DESTROY_HIRED_MERCHANT (vs SPAWN_NPC_REQUEST_CONTROLLER@266) · UPDATE_HIRED_MERCHANT (vs NPC_ACTION@267) · SPAWN_DOOR (vs DROP_ITEM_FROM_MAPOBJECT@275) · REMOVE_DOOR (vs REMOVE_ITEM_FROM_MAP@276) · IMITATED_NPC_DATA (vs BRIDLE_MOB_CATCH_FAIL@81)
### Dialog/shop/cashshop/script clientbound
- NPC_TALK CScriptMan (vs ARIANT_ARENA_USER_SCORE@304) · ADMIN_SHOP_MESSAGE (vs SHEEP_RANCH_CLOTHES@307) · ADMIN_SHOP (vs WITCH_TOWER_SCORE_UPDATE@308) · FREDRICK_MESSAGE CStoreBankDlg (vs ZAKUM_SHRINE@310) · RPS_GAME (vs OPEN_NPC_SHOP@312) · MESSENGER CUIMessenger (vs CONFIRM_SHOP_TRANSACTION@313) · PARCEL CParcelDlg (vs TOURNAMENT@322) · CHARGE_PARAM_RESULT (vs TOURNAMENT_MATCH_TABLE@323) · CASHSHOP_PURCHASE_EXP_CHANGED (vs TOURNAMENT_CHARACTERS@326) · CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT (vs QUERY_CASH_RESULT@331) · CASHSHOP_GACHAPON_STAMP_RESULT (vs CASHSHOP_OPERATION@332)

> NOTE: several "stale csv-import" rows whose fname matches a task-096 CField op (SPOUSE_CHAT, WHISPER, FIELD_EFFECT) may be LEFTOVER DUPLICATE rows of an already-correct op — verify: if the registry already has the correct row for that op at the right opcode, the csv-import duplicate is spurious → delete it (don't re-point).

## Verification
- `python3 -c "import yaml,collections; d=yaml.safe_load(open('docs/packets/registry/gms_v84.yaml')); c=collections.Counter((r['opcode'],r['direction']) for r in d); print([k for k,n in c.items() if n>1])"` → `[]`
- `go run ./tools/packet-audit matrix --check` exit 0, 0 conflicts, no regression to any verified cell.
- All touched rows `ida-discovered` with `ida.address`; v84 IDB functions named + `idb_save`'d.

## Stacking
Branched off `task-096-cfield-packet-family` (PR #794) — it has the v84 CField corrections. Rebase onto main once #794 merges (the shared v84 changes will already be there).
