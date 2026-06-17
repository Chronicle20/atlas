# gms_v84 CField opcodes (Cluster 2)

Stage 1, task-096 Cluster 2. Source IDB: `GMS_v84.1_U_DEVM.exe`, IDA-MCP port
**13337** (session 2026-06-14). Dispatcher of record: `CField::OnPacket` @
`0x53d5a7` (decompiled this session for opcode verification).

**Layout rule (task context §4 / bug_v84_opcode_table_shifted_vs_v83):** v84 is
byte-identical to v83 in handler *structure*; only the dispatcher *opcode table*
shifts. So every Cluster 2 op below is `LAYOUT ≡ v83` — fields are not re-listed
(see `structures/gms_v83.md`). What v84 needs is the correct per-version opcode.

**⚠ REGISTRY WAS STALE.** task-085 flagged that v84 clientbound opcodes ≥ ~0x3F
were left at the inherited v83 values pending re-derivation. The CField low-range
(BLOCKED_MAP onward) is shifted **+3** vs v83 (verified against the live v84
dispatcher this session: case bodies decompiled and matched to v83 layouts). The
`gms_v84.yaml` Cluster 2 rows have been corrected (provenance: manual, IDA cite).

Dispatcher low-switch evidence (case hex → handler @ addr → identified op by body):

| dispatcher case | handler addr (sub) | identified op | body match |
|---|---|---|---|
| 0x86 | sub_53DAE2 | BLOCKED_MAP | Decode1 reason switch (StringPool) ≡ v83 OnTransferFieldReqIgnored |
| 0x87 | sub_53DC8E | BLOCKED_SERVER | Decode1 reason switch ≡ v83 OnTransferChannelReqIgnored |
| 0x88 | sub_53DE01 | FORCED_MAP_EQUIP | vtable slot 5 (offset 20) + CUserLocal ≡ v83 OnFieldSpecificData |
| 0x8C | sub_53F255 | SUMMON_ITEM_INAVAILABLE | Decode1 flag ≡ v83 OnSummonItemInavailable |
| 0x8E | sub_53F291 | FIELD_OBSTACLE_ONOFF | str + Decode4 ≡ v83 OnFieldObstacleOnOff |
| 0x8F | sub_53F2DD | FIELD_OBSTACLE_ONOFF_LIST | Decode4 + loop(str+Decode4) ≡ v83 |
| 0x90 | sub_53F33C | FIELD_OBSTACLE_ALL_RESET | empty/iterate obstacle list ≡ v83 |
| 0x92 | sub_5414AA | PLAY_JUKEBOX | Decode4 + conditional DecodeStr ≡ v83 OnPlayJukeBox |
| 0x94 | sub_541D5B | OX_QUIZ | Decode1+Decode1+Decode2 ≡ v83 OnQuiz |
| 0x95 | sub_5423C4 | GMEVENT_INSTRUCTIONS | Decode1 index ≡ v83 OnDesc |
| 0x99 | sub_543BB8 | SET_QUEST_CLEAR | empty body (CQuestMan buffer free) ≡ v83 |
| 0x9A | sub_543BCB | SET_QUEST_TIME | Decode1 + loop(Decode4+buf8+buf8) ≡ v83 |
| 0x9C | sub_543D1C | SET_OBJECT_STATE | str + Decode4 ≡ v83 OnSetObjectState |
| 0x9D | sub_53DAD0 | STOP_CLOCK | empty body (clock destroy) ≡ v83 OnDestroyClock |

(Adjacent named handlers anchor the +3 shift: 0x89=CField__OnGroupMessage,
0x8A=CField__OnWhisper, 0x91=CField::OnBlowWeather, 0x96=CLOCK vtable call —
already correct in the registry. Case 0xA2=sub_748BFF is a CMob move handler,
NOT a foothold op — there is no clientbound foothold case in the v84 low-switch.)

---

## Corrected v84 opcodes (LAYOUT ≡ v83 for all)

- **BLOCKED_MAP**: opcode 0x86 (134) — LAYOUT ≡ v83. (was 131; +3)
- **BLOCKED_SERVER**: opcode 0x87 (135) — LAYOUT ≡ v83. (was 132; +3)
- **FORCED_MAP_EQUIP**: opcode 0x88 (136) — LAYOUT ≡ v83. (was 133; +3)
- **SUMMON_ITEM_INAVAILABLE**: opcode 0x8C (140) — LAYOUT ≡ v83. (was 137; +3)
- **FIELD_OBSTACLE_ONOFF**: opcode 0x8E (142) — LAYOUT ≡ v83. (was 139; +3)
- **FIELD_OBSTACLE_ONOFF_LIST**: opcode 0x8F (143) — LAYOUT ≡ v83. (was 140; +3)
- **FIELD_OBSTACLE_ALL_RESET**: opcode 0x90 (144) — LAYOUT ≡ v83. (was 141; +3)
- **PLAY_JUKEBOX**: opcode 0x92 (146) — LAYOUT ≡ v83. (was 143; +3)
- **OX_QUIZ**: opcode 0x94 (148) — LAYOUT ≡ v83. (was 145; +3)
- **GMEVENT_INSTRUCTIONS**: opcode 0x95 (149) — LAYOUT ≡ v83. (was 146; +3)
- **SET_QUEST_CLEAR**: opcode 0x99 (153) — LAYOUT ≡ v83. (was 150; +3)
- **SET_QUEST_TIME**: opcode 0x9A (154) — LAYOUT ≡ v83. (was 151; +3)
- **SET_OBJECT_STATE**: opcode 0x9C (156) — LAYOUT ≡ v83. (was 153; +3)
- **STOP_CLOCK**: opcode 0x9D (157) — LAYOUT ≡ v83. (was 154; +3)

## FOOTHOLD_INFO — VERSION-ABSENT (clientbound) in v84
- No `CField::OnFootHoldInfo` / `OnRequestFootHoldInfo` function exists in the v84
  IDB (`func_query name_regex=FootHold` → 0 results) and the v84 `CField::OnPacket`
  low-switch has no clientbound foothold case (it ends at 0xA2 = a CMob move
  handler, sub_748BFF). Matches v83 (also version-absent). The clientbound
  FOOTHOLD_INFO op is a v87+ addition.
- The registry serverbound `FOOTHOLD_INFO` row (op 226, fname
  `CField::OnRequestFootHoldInfo`) is left as-is: the serverbound send-builder is
  not a named function in v84 and is out of Cluster 2 clientbound scope; recorded
  as version-absent at the named-handler tier, consistent with v83.
