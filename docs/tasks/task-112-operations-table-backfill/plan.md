# task-112 — Dispatcher-family operations-table backfill (v87/v95/jms)

## Problem
The `operations` mode tables for several clientbound dispatcher-family writers exist only
in the gms_83/gms_84 seed templates; the v87/v95/jms templates (created during later
version bring-ups) dropped them. Result: `atlas_packet.ResolveCode("operations", KEY)`
finds no table → returns the sentinel **99** → wrong sub-op mode byte → client crash when
that packet is sent. (Same bug family as the task-109 character-effect fix.)
Reference: project memory `bug_operations_mode_tables_missing_v87_v95_jms`.

Enumeration vs gms_83 (the canonical 20 operations-bearing writers): **messenger / cashshop
/ storage / npcshop / interaction / party / buddy / guild-result etc. already carry their
tables** in v87/v95/jms (ported correctly). The genuine gaps are 8 families, in two classes.

## Scope (user-approved: comprehensive A+B, skip MTS)

### Class A — writer IS wired, operations table MISSING (low-risk: add table only)
Opcode already correct in the template; only add `options.operations` with version-correct
mode bytes read from the dispatcher switch.

| family | writer name | gms_83 op | dispatcher fname (clientbound) | keys |
|---|---|---|---|---|
| FameResponse | `FameResponse` | 0x26 | `CWvsContext::OnGivePopularityResult` | 7 |
| HiredMerchantOperation | `HiredMerchantOperation` | 0x32 | hired-merchant result dispatcher (CHiredMerchant*/CMiniRoom*) | 11 |
| UiOpen | `UiOpen` | 0xDC | the UI-open dispatcher (`CWvsContext::OnUIOpen*` / `OnQuestResult`-adjacent) | 21 |

Present in v87 (0x26/0x32/0xE9), v95 (0x25/0x31/0xFB), jms (0x24/0x2F/0xE5) — opcodes already
in each template; **do NOT change the opcode**, only add the table.

### Class B — writer ENTIRELY ABSENT, must wire writer + add table (higher-risk: needs opcode)
The Go writer + an atlas-channel producer exist (`chat/clientbound/world_message.go`,
`pet/clientbound/activated.go`, guild-bbs, note), so these packets ARE emitted but silently
fail / misfire on these versions. Wire a new `socket.writers` entry `{opCode, writer,
options.operations}` with the **version-correct clientbound opcode**.

| family | writer name | gms_83 op | versions to wire | opcode source |
|---|---|---|---|---|
| WorldMessage | `WorldMessage` | 0x44 | v87, v95, jms | version clientbound opcode (registry / IDA send-site `CField::OnBroadcastMsg`-family) |
| PetActivated | `PetActivated` | 0xA8 | v87, v95, jms | version clientbound opcode (`CUserPool::OnUserRemotePacket` pet-activated sub-op) |
| GuildBBS | `GuildBBS` | 0x3B | jms only | `CUIGuildBBS::OnGuildBBSPacket` registered op (GUILD_BBS_PACKET) |
| NoteOperation | `NoteOperation` | 0x29 | jms only | `CWvsContext::OnMemoResult` registered op (MEMO_RESULT) |

These are clientbound **writers** (not inbound handlers) → **no validator field needed**.

### Explicitly SKIPPED
- `MtsOperation` jms (gms_83 0x15C, 35 keys) — present+table in v87/v95; absent in jms. The
  MTS feature is blocked/planned (memory `project_mts_feature_planned`), so the jms writer is
  correctly unwired. Document the skip; do not wire.

## gms_83 reference key sets (semantic order — bytes are VERSION-DEPENDENT, re-read per IDB)
- **FameResponse**: GIVE:0 INVALID_NAME:1 NOT_MINIMUM_LEVEL:2 NOT_TODAY:3 NOT_THIS_MONTH:4 RECEIVE:5 UNEXPECTED:6
- **HiredMerchantOperation**: OPEN_SHOP:7 ERROR_UNKNOWN:8 ERROR_RETRIEVE_FROM_FREDRICK:9 ERROR_ANOTHER_CHARACTER_IS_USING_THE_ITEM:10 ERROR_UNABLE_TO_OPEN_THE_STORE:11 SHOP_SEARCH:13 SHOP_RENAME:14 ERROR_RETRIEVE_FROM_FREDRICK_2:15 REMOTE_SHOP_WARP:16 CONFIRM_MANAGE:17 FREE_FORM_NOTICE:18
- **UiOpen**: ITEM:0 EQUIPMENT:1 STATISTICS:2 SKILLS:3 KEYBOARD:5 QUEST:6 MONSTER_BOOK:9 CHARACTER_INFORMATION:10 GUILD_BBS:11 MONSTER_CARNIVAL:18 ENERGY_BAR:20 PARTY_SEARCH:22 ITEM_MAKER:23 RANKING:26 FAMILY:27 FAMILY_PEDIGREE:28 OPERATOR_BOARD:29 OPERATOR_BOARD_STATE:30 MEDAL_MEDAL_QUEST:31 WEB_EVENT:32 SKILLS_EX:33
- **WorldMessage**: NOTICE:0 POP_UP:1 MEGAPHONE:2 SUPER_MEGAPHONE:3 TOP_SCROLL:4 PINK_TEXT:5 BLUE_TEXT:6 NPC:7 ITEM_MEGAPHONE:8 YELLOW_MEGAPHONE:9 MULTI_MEGAPHONE:10 WEATHER:11 GACHAPON:12 UNKNOWN_3:13 UNKNOWN_4:14 CLIPBOARD_NOTICE_1:15 CLIPBOARD_NOTICE_2:16 UNKNOWN_7:17 UNKNOWN_8:18
- **GuildBBS**: BBS_THREAD_LIST:6 BBS_THREAD:7 BBS_ENTRY_NOT_FOUND:8
- **NoteOperation**: SHOW:3 SEND_SUCCESS:4 SEND_ERROR:5 REFRESH:7
- **PetActivated**: NORMAL:0 HUNGER:1 EXPIRED:2 UNKNOWN_1:3 UNKNOWN_2:4

> The mode bytes shift per version (task-109 proved the effect table moved QUEST 3→5 in v95).
> The KEY→byte mapping MUST be re-derived from each version's dispatcher switch, never copied.

## Execution — per IDB (select_instance is global; one IDB at a time)
- **v87** (`GMSv87_4GB.exe`/13340): FameResponse, HiredMerchant, UiOpen (tables) + WorldMessage, PetActivated (wire).
- **v95** (`GMS_v95.0_U_DEVM.exe`/13339): same 5 families.
- **jms** (`MapleStory_dump_SCY.exe`/13338): FameResponse, HiredMerchant, UiOpen (tables) + WorldMessage, PetActivated, GuildBBS, NoteOperation (wire).

Per family per version:
1. Identify the dispatcher fname; decompile its switch; map every gms_83 key to that version's
   case byte by matching the case body (cite the decompile per key). Omit a key whose mode is
   genuinely absent in that version's switch (note it).
2. Class B only: derive the version clientbound **opcode** (prefer the packet-audit registry's
   registered clientbound op for that packet in that version; else IDA opcode table / send-site).
   Cite the source. Wire `{opCode, writer:"<Name>", options:{operations:{...}}}` into `socket.writers`.
3. Class A: add `options.operations` to the existing writer entry; do NOT touch its opcode.
4. JSON-validate; preserve formatting.

## Verification (no byte-fixture exists — IDA-citation + build are the gates)
- `python3 -m json.tool` each edited template (valid JSON).
- `( cd services/atlas-configurations/atlas.com/configurations && go build ./... )` green.
- `( cd services/atlas-channel/atlas.com/channel && go build ./... )` green (consumes the writers).
- Cross-check anchors where known (e.g. FameResponse GIVE should be the success mode = low byte;
  confirm against gms_83 ordering only as a sanity hint, NOT a source of truth).
- Each value cited to an IDA decompile line. Controller spot-checks the riskiest (absent-writer
  opcodes + any non-obvious reorder), mirroring the task-109 jms-effect spot-check.

## Commit discipline
- Worktree `.worktrees/task-112-operations-table-backfill`, branch `task-112-operations-table-backfill`.
- Commit per version (or per family) staging only template files. Never `git add -A`.
- One PR at the end; code-review before PR (plan-adherence + backend-guidelines).
