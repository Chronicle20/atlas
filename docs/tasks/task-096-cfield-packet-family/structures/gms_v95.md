# gms_v95 CField layouts (Cluster 2)

IDA-harvested client read order (Stage 1, task-096 Cluster 2). Source IDB:
`GMS_v95.0_U_DEVM.exe`, IDA-MCP port **13340** (session 2026-06-14). Dispatcher of
record: `CField::OnPacket` @ `0x546d50` (decompiled this session). All ops below
are **clientbound** (`On*` reading from `CInPacket`) unless noted; the listed
field order is the exact client *read* order = server *write* order.

Widths: 1 = Decode1 (byte), 2 = Decode2 (int16), 4 = Decode4 (int32),
str = DecodeStr (int16 len-prefixed ASCII), buf(N) = DecodeBuffer of N bytes.

**Diff baseline:** `structures/gms_v83.md`. Where a v95 handler is byte-identical
to v83 it is marked `LAYOUT ≡ v83`. The dispatcher CASE opcode (decimal) is
ground truth; every Cluster 2 case matched the registry decimal exactly — **no
v95 registry fixes were required for Cluster 2.**

---

## BLOCKED_MAP
- fname: `CField::OnTransferFieldReqIgnored`
- address: `0x52f3b0`
- dispatcher: `CField::OnPacket` case `147` (0x93)
- registry opcode: 147 — matches dispatcher
- **LAYOUT ≡ v83.** `reason : 1 : Decode1` switch (cases 1-8). Single byte.

## BLOCKED_SERVER
- fname: `CField::OnTransferChannelReqIgnored`
- address: `0x52f5f0`
- dispatcher: `CField::OnPacket` case `148` (0x94)
- registry opcode: 148 — matches dispatcher
- **LAYOUT ≡ v83.** `reason : 1 : Decode1` switch (cases 1-5). Single byte.

## FORCED_MAP_EQUIP
- fname: `CField::OnFieldSpecificData`
- address: `0x52a7e0`
- dispatcher: `CField::OnPacket` case `149` (0x95)
- registry opcode: 149 — matches dispatcher
- **LAYOUT ≡ v83.** Body calls the virtual `CField::DecodeFieldSpecificData(this,
  CUserLocal, iPacket)` — in v95 this virtual is named (base @ 0x53cb50, with
  subclass overrides for Battlefield/Coconut/MonsterCarnival/ShowaBath/Tutorial).
  The base-CField override does no direct reads. Opcode-only at the base tier,
  same as v83. (Subclass payload handling is out of Cluster 2 scope.)

## SUMMON_ITEM_INAVAILABLE
- fname: `CField::OnSummonItemInavailable`
- address: `0x52f7b0`
- dispatcher: `CField::OnPacket` case `153` (0x99)
- registry opcode: 153 — matches dispatcher
- **LAYOUT ≡ v83.** `flag : 1 : Decode1` bool. Single byte.

## FIELD_OBSTACLE_ONOFF
- fname: `CField::OnFieldObstacleOnOff`
- address: `0x535a80`
- dispatcher: `CField::OnPacket` case `155` (0x9B)
- registry opcode: 155 — matches dispatcher
- **LAYOUT ≡ v83.** `name : str : DecodeStr`, `state : 4 : Decode4`.

## FIELD_OBSTACLE_ONOFF_LIST
- fname: `CField::OnFieldObstacleOnOffStatus`
- address: `0x535b00`
- dispatcher: `CField::OnPacket` case `156` (0x9C)
- registry opcode: 156 — matches dispatcher
- **LAYOUT ≡ v83.** `count : 4 : Decode4` (loop if > 0), then loop `count`×:
  `name : str : DecodeStr`, `state : 4 : Decode4`.

## FIELD_OBSTACLE_ALL_RESET
- fname: `CField::OnFieldObstacleAllReset`
- address: `0x52c830`
- dispatcher: `CField::OnPacket` case `157` (0x9D)
- registry opcode: 157 — matches dispatcher
- **LAYOUT ≡ v83.** EMPTY BODY — walks `m_lpObstacle` ZList and calls
  SetObjectState(name, 0) per entry. Opcode-only on the wire.

## SET_OBJECT_STATE
- fname: `CField::OnSetObjectState`
- address: `0x539890`
- dispatcher: `CField::OnPacket` case `169` (0xA9)
- registry opcode: 169 — matches dispatcher
- **LAYOUT ≡ v83.** `name : str : DecodeStr`, `state : 4 : Decode4`. Byte-identical
  to FIELD_OBSTACLE_ONOFF.

## SET_QUEST_CLEAR
- fname: `CField::OnSetQuestClear`
- address: `0x52c870`
- dispatcher: `CField::OnPacket` case `166` (0xA6)
- registry opcode: 166 — matches dispatcher
- **LAYOUT ≡ v83.** EMPTY BODY — `ZArray<MODQUESTTIME>::RemoveAll(CQuestMan->
  m_aModifiedQuestTime)`. Opcode-only on the wire. (Same effect as v83's buffer
  free; v95 names the member.)

## SET_QUEST_TIME
- fname: `CField::OnSetQuestTime`
- address: `0x52b790`
- dispatcher: `CField::OnPacket` case `167` (0xA7)
- registry opcode: 167 — matches dispatcher
- **LAYOUT ≡ v83.** `count : 1 : Decode1` (loop if > 0), then loop `count`×:
  `questId : 4 : Decode4`, `startTime : buf(8)`, `endTime : buf(8)` → SetQuestTime.

## STOP_CLOCK
- fname: `CField::OnDestroyClock`
- address: `0x52a7c0`
- dispatcher: `CField::OnPacket` case `170` (0xAA)
- registry opcode: 170 — matches dispatcher
- **LAYOUT ≡ v83.** EMPTY BODY — `m_pClock -> CWnd::Destroy`. Opcode-only.

## GMEVENT_INSTRUCTIONS
- fname: `CField::OnDesc`
- address: `0x5313d0`
- dispatcher: `CField::OnPacket` case `162` (0xA2)
- registry opcode: 162 — matches dispatcher
- **LAYOUT ≡ v83.** `index : 1 : Decode1` — bounds-checked index into
  `m_asHelpMsg`. Single byte.

## OX_QUIZ
- fname: `CField::OnQuiz`
- address: `0x537a90`
- dispatcher: `CField::OnPacket` case `161` (0xA1)
- registry opcode: 161 — matches dispatcher
- **LAYOUT ≡ v83.** `show : 1 : Decode1`, `category : 1 : Decode1`,
  `number : 2 : Decode2`. Three wire fields (1+1+2).

## PLAY_JUKEBOX
- fname: `CField::OnPlayJukeBox`
- address: `0x537940`
- dispatcher: `CField::OnPacket` case `159` (0x9F)
- registry opcode: 159 — matches dispatcher
- **LAYOUT ≡ v83.** `jukeboxItemId : 4 : Decode4`, then `playerName : str :
  DecodeStr` read ONLY when jukeboxItemId >= 0 (get_consume_cash_item_type==20
  guard). Conditional trailing string. (v95 routes the chat add through
  CUIStatusBar::ChatLogAdd; wire read order unchanged.)

## FOOTHOLD_INFO  (clientbound — PRESENT in v95, DELTA vs v83 AND vs v87)
- fname: `CField::OnFootHoldInfo` (`?OnFootHoldInfo@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x53a810`
- dispatcher: `CField::OnPacket` case `176` (0xB0)
- registry opcode: 176 (0xB0) — matches dispatcher (registry row `IDA_0X0B0`,
  fname `CField::OnFootHoldInfo`)
- **DELTA vs v83 (absent) AND vs v87 (single-entry).** v95 wraps the body in an
  OUTER COUNT loop and adds an inner foothold-id sub-list per entry. Full client
  read order:
  - `count : 4 : Decode4` — number of foothold-info entries (loop if > 0)
  - loop `count`×:
    - `name     : str : DecodeStr` — dynamic-obj name (ZMap key)
    - `mode     : 4   : Decode4` — state/action code
    - `idCount  : 4   : Decode4` — number of foothold ids in this entry's list
    - loop `idCount`×: `footholdId : 4 : Decode4` (appended to a ZList<long>)
    - **if `mode == 2`** (move case) — fixed trailing block (same 10 fields as
      v87): `f0 4` `f1 4` `f2 4` `f3 4` `f4 4` `x 4` `y 4` `f7 4` `b0 1` `b1 1`
      = **8× Decode4 + 2× Decode1**.
  - Summary of the v87→v95 delta: v95 prepends an outer `count` (4) and inserts a
    per-entry `idCount`(4) + id-list before the conditional mode==2 block. v87 is
    a single entry with no outer count and no id sub-list.

## REQUEST_FOOTHOLD_INFO trigger / FOOTHOLD_INFO (serverbound send) — PRESENT in v95
- fname: `CField::OnRequestFootHoldInfo` (`?OnRequestFootHoldInfo@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x52ddd0`
- dispatcher: `CField::OnPacket` case `177` (0xB1) — registry row `IDA_0X0B1`,
  fname `CField::OnRequestFootHoldInfo` (matches dispatcher)
- **Direction: clientbound trigger that BUILDS A SERVERBOUND SEND.** The handler
  reads NOTHING from the inbound CInPacket. It is invoked by the server as a
  clientbound poke (case 177, empty inbound body) and RESPONDS by constructing
  `COutPacket(270)` and `CClientSocket::SendPacket` — i.e. it emits the
  serverbound `FOOTHOLD_INFO` (registry sb opcode 270).
- Serverbound FOOTHOLD_INFO (op 270) write order, per dynamic-obj entry (iterates
  `m_lDynamicObjs`; no leading count — the server reads to EOF / list end):
  - `nCurState : 4 : Encode4`
  - if the obj has a MOVING_OBJ_INFO:
    - `nCurX : 4 : Encode4`
    - `nCurY : 4 : Encode4`
    - `bReverseVertical   : 1 : Encode1`
    - `bReverseHorizontal : 1 : Encode1`
  - else (no moving info): `0(4)` `0(4)` `0(1)` `0(1)` (same field widths, zeroed)
- Inbound (clientbound) wire payload for the case-177 trigger is opcode-only
  (handler ignores the reader). NB: registry currently lists the serverbound
  `FOOTHOLD_INFO` row at op **270** with fname `OnRequestFootHoldInfo` — that
  matches the COutPacket(270) ctor here; consistent, no fix needed.
