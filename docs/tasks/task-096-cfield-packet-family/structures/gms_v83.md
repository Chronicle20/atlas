# gms_v83 CField layouts

IDA-harvested client read order (Stage 1). Source IDB: `MapleStory_dump.exe`
(`v83_Me`), IDA-MCP port 13342 (session 2026-06-14). Dispatcher of record:
`CField::OnPacket` @ `0x531325` (the registry note already cites this). All ops
below are **clientbound** (`On*` reading from `CInPacket`); the listed field
order is the exact client *read* order, which is also the server *write* order.

Widths: 1 = Decode1 (byte), 2 = Decode2 (int16), 4 = Decode4 (int32),
str = DecodeStr (int16 len-prefixed ASCII), buf(N) = DecodeBuffer of N bytes.

Opcode reconciliation: every dispatcher case below matches the registry decimal
opcode exactly (case `0x83`=131=BLOCKED_MAP … `0x9A`=154=STOP_CLOCK). The
clientbound `On*` handlers do **not** build a COutPacket (they consume one), so
there is no COutPacket-opcode to cross-check; the dispatcher case IS the
ground-truth opcode. No registry fixes were required for Cluster 2.

---

## BLOCKED_MAP
- fname: `CField::OnTransferFieldReqIgnored` (`?OnTransferFieldReqIgnored@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x53185c`
- dispatcher: `CField::OnPacket` case `0x83`
- registry opcode: 131 (0x83) — matches dispatcher
- fields (client read order):
  - `reason : 1 : Decode1 — switch over reason code (1=portal closed, 2=cannot go there, 3/4=force-of-ground, 5=party-only, 6=cash-shop-unavailable). Single byte, no further reads.`

## BLOCKED_SERVER
- fname: `CField::OnTransferChannelReqIgnored` (`?OnTransferChannelReqIgnored@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x531a08`
- dispatcher: `CField::OnPacket` case `0x84`
- registry opcode: 132 (0x84) — matches dispatcher
- fields (client read order):
  - `reason : 1 : Decode1 — switch over reason code (1=cannot move channel, 2=cannot enter cash shop, 3=trade shop unavailable, 4=trade shop full, 5=level requirement). Single byte, no further reads.`

## FIELD_OBSTACLE_ALL_RESET
- fname: `CField::OnFieldObstacleAllReset` (`?OnFieldObstacleAllReset@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5330b6`
- dispatcher: `CField::OnPacket` case `0x8D`
- registry opcode: 141 (0x8D) — matches dispatcher; registry note already
  documents the CSV name truncation fix (manual provenance).
- fields (client read order):
  - **EMPTY BODY — no CInPacket reads.** Iterates the field's internal obstacle
    list (`this->m_lpObstacle[3]`) and calls `CMapLoadable::SetObjectState(name, 0)`
    for each. Wire payload is opcode-only.

## FIELD_OBSTACLE_ONOFF
- fname: `CField::OnFieldObstacleOnOff` (`?OnFieldObstacleOnOff@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x53300b`
- dispatcher: `CField::OnPacket` case `0x8B`
- registry opcode: 139 (0x8B) — matches dispatcher
- fields (client read order):
  - `name  : str : DecodeStr — obstacle object name`
  - `state : 4   : Decode4 — object state; passed to CMapLoadable::SetObjectState(name, state)`

## FIELD_OBSTACLE_ONOFF_LIST
- fname: `CField::OnFieldObstacleOnOffStatus` (`?OnFieldObstacleOnOffStatus@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x533057`
- dispatcher: `CField::OnPacket` case `0x8C`
- registry opcode: 140 (0x8C) — matches dispatcher
- fields (client read order):
  - `count : 4 : Decode4 — element count (loop bound; loop runs only if count > 0)`
  - loop `count` times:
    - `name  : str : DecodeStr — obstacle object name`
    - `state : 4   : Decode4 — object state`
  - (each iteration calls CMapLoadable::SetObjectState(name, state))

## SET_OBJECT_STATE
- fname: `CField::OnSetObjectState` (`?OnSetObjectState@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x537a1e`
- dispatcher: `CField::OnPacket` case `0x99`
- registry opcode: 153 (0x99) — matches dispatcher
- fields (client read order):
  - `name  : str : DecodeStr — object name`
  - `state : 4   : Decode4 — object state; CMapLoadable::SetObjectState(name, state)`
  - NOTE: byte-identical layout to FIELD_OBSTACLE_ONOFF (str + int4).

## SET_QUEST_CLEAR
- fname: `CField::OnSetQuestClear` (`?OnSetQuestClear@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5378ba`
- dispatcher: `CField::OnPacket` case `0x96`
- registry opcode: 150 (0x96) — matches dispatcher
- fields (client read order):
  - **EMPTY BODY — no CInPacket reads.** Body is `sub_539A13(dword_BED614 + 71)`
    which frees an internal quest buffer (`ZAllocEx::Free`). Wire payload is
    opcode-only.

## SET_QUEST_TIME
- fname: `CField::OnSetQuestTime` (`?OnSetQuestTime@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5378cd`
- dispatcher: `CField::OnPacket` case `0x97`
- registry opcode: 151 (0x97) — matches dispatcher
- fields (client read order):
  - `count : 1 : Decode1 — entry count (loop bound; loop runs only if count > 0)`
  - loop `count` times:
    - `questId   : 4      : Decode4`
    - `startTime : buf(8) : DecodeBuffer(8) — FILETIME (two dwords: lo, hi)`
    - `endTime   : buf(8) : DecodeBuffer(8) — FILETIME (two dwords: lo, hi)`
  - (each iteration calls CQuestMan::SetQuestTime(questId, start.lo, start.hi, end.lo, end.hi))

## STOP_CLOCK
- fname: `CField::OnDestroyClock` (`?OnDestroyClock@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x53184a`
- dispatcher: `CField::OnPacket` case `0x9A`
- registry opcode: 154 (0x9A) — matches dispatcher
- fields (client read order):
  - **EMPTY BODY — no CInPacket reads.** Destroys the clock window
    (`this->m_nMobIconHeight` -> CWnd::Destroy). Wire payload is opcode-only.

## FORCED_MAP_EQUIP
- fname: `CField::OnFieldSpecificData` (`?OnFieldSpecificData@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x531b7b`
- dispatcher: `CField::OnPacket` case `0x85`
- registry opcode: 133 (0x85) — matches dispatcher
- fields (client read order):
  - **NO DIRECT READS in OnFieldSpecificData.** Body forwards the CInPacket to a
    `CField` virtual method: `(*(this->__vftable + 0x14))(this, dword_BEBF98, a2)`
    — vtable slot 5. Resolved the CField vtable (base `0xaf3f40`, OnPacket at
    slot 15/`0xaf3f7c` confirms the base): **slot 5 = `0x528040`**, which is a
    no-op stub (`int sub_528040(){ return 0; }`). So for a plain `CField` in v83,
    FORCED_MAP_EQUIP consumes nothing at the CField level (any payload handling
    would be a subclass override; base CField is a stub). Treat as opcode-only /
    no-op at the CField tier in v83.

## SUMMON_ITEM_INAVAILABLE
- fname: `CField::OnSummonItemInavailable` (`?OnSummonItemInavailable@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x532fcf`
- dispatcher: `CField::OnPacket` case `0x89`
- registry opcode: 137 (0x89) — matches dispatcher
- fields (client read order):
  - `flag : 1 : Decode1 — bool. If 0 (false), pops the "you can't use it here in this map" notice. Single byte, no further reads.`

## GMEVENT_INSTRUCTIONS
- fname: `CField::OnDesc` (`?OnDesc@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5360c0`
- dispatcher: `CField::OnPacket` case `0x92`
- registry opcode: 146 (0x92) — matches dispatcher
- fields (client read order):
  - `index : 1 : Decode1 — index into the pre-loaded desc-image array (m_apLayerTownPortal[1]); bounds-checked against array size. Single byte, no further reads.`

## OX_QUIZ
- fname: `CField::OnQuiz` (`?OnQuiz@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x535a57`
- dispatcher: `CField::OnPacket` case `0x91`
- registry opcode: 145 (0x91) — matches dispatcher
- fields (client read order):
  - `show     : 1 : Decode1 — flag (v78); nonzero => show quiz image, zero => show answer text path`
  - `category : 1 : Decode1 — quiz category (v2; Int2StrW key into the OXQuizImg WZ node)`
  - `number   : 2 : Decode2 — question number (v3; if 0, early-out / clears the status-bar problem)`
  - NOTE: remaining work is WZ-image lookup (Etc/OXQuizImg) only; exactly three wire fields: 1 + 1 + 2.

## PLAY_JUKEBOX
- fname: `CField::OnPlayJukeBox` (`?OnPlayJukeBox@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x535224`
- dispatcher: `CField::OnPacket` case `0x8F`
- registry opcode: 143 (0x8F) — matches dispatcher
- fields (client read order):
  - `jukeboxItemId : 4   : Decode4 — stored as this->m_nJukeBoxItemID`
  - `playerName    : str : DecodeStr — read ONLY when jukeboxItemId >= 0 (item present). When jukeboxItemId < 0 (jukebox stopped) the string is absent. Guard: get_consume_cash_item_type(itemId)==20 wraps the whole body; the >= 0 sub-branch is where DecodeStr runs.`
  - NOTE: conditional trailing string — int4 always, then str only if itemId >= 0.

## FOOTHOLD_INFO — VERSION-ABSENT (v83)
- registry row: `FOOTHOLD_INFO` (serverbound, opcode 226 / 0xE2, fname `CField::OnRequestFootHoldInfo`)
- **VERSION-ABSENT in v83.** Evidence:
  - `lookup_funcs` / `func_query name_regex=(?i)foothold` find NO
    `CField::OnRequestFootHoldInfo`, `OnFootHoldInfo`, or any `CField`-class
    foothold packet handler. The only `foothold` matches are physics/geometry
    helpers (`CWvsPhysicalSpace2D::GetFoothold*`, `CStaticFoothold`,
    `CAnimationDisplayer::MakeLayer_FootHold`, etc.) — none read/write a packet.
  - The clientbound dispatcher `CField::OnPacket` (0x531325) has no case routing
    to any foothold handler (confirmed by full decompile of all cases).
  - Matches task-0.3's finding that `OnRequestFootHoldInfo`/`OnFootHoldInfo` do
    not exist as functions in the v83 IDB.
- Expected to appear in v87/v95 (handle there; do not invent a v83 layout).
