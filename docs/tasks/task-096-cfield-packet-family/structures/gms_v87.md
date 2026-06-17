# gms_v87 CField layouts (Cluster 2)

IDA-harvested client read order (Stage 1, task-096 Cluster 2). Source IDB:
`GMSv87_4GB.exe`, IDA-MCP port **13341** (session 2026-06-14). Dispatcher of
record: `CField::OnPacket` @ `0x558b48` (decompiled this session). All ops below
are **clientbound** (`On*` reading from `CInPacket`) unless noted; the listed
field order is the exact client *read* order = server *write* order.

Widths: 1 = Decode1 (byte), 2 = Decode2 (int16), 4 = Decode4 (int32),
str = DecodeStr (int16 len-prefixed ASCII), buf(N) = DecodeBuffer of N bytes.

**Diff baseline:** `structures/gms_v83.md`. Where a v87 handler is byte-identical
to v83 it is marked `LAYOUT ≡ v83` (fields not re-listed). The dispatcher CASE
opcode (hex) is ground truth; every Cluster 2 case below matched the registry
decimal opcode exactly — **no v87 registry fixes were required for Cluster 2.**

Opcode note: v87 clientbound table sits +8 above v83 in this region (v83
BLOCKED_MAP 0x83 → v87 0x8B), reflecting the cumulative insertions documented in
gms_v87.yaml. The dispatcher case hex below is the live IDB value.

---

## BLOCKED_MAP
- fname: `CField::OnTransferFieldReqIgnored` (`?OnTransferFieldReqIgnored@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5590e1`
- dispatcher: `CField::OnPacket` case `0x8B`
- registry opcode: 139 (0x8B) — matches dispatcher
- **LAYOUT ≡ v83.** `reason : 1 : Decode1` — switch over reason code (cases 1-8;
  v87 adds cases 7/8 vs v83's 1-6, but still a single leading byte, no further
  CInPacket reads). Wire payload = 1 byte.

## BLOCKED_SERVER
- fname: `CField::OnTransferChannelReqIgnored` (`?OnTransferChannelReqIgnored@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5592bd`
- dispatcher: `CField::OnPacket` case `0x8C`
- registry opcode: 140 (0x8C) — matches dispatcher
- **LAYOUT ≡ v83.** `reason : 1 : Decode1` — switch over reason code, single byte.

## FORCED_MAP_EQUIP
- fname: `CField::OnFieldSpecificData` (`?OnFieldSpecificData@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55941a`
- dispatcher: `CField::OnPacket` case `0x8D`
- registry opcode: 141 (0x8D) — matches dispatcher
- **LAYOUT ≡ v83.** Body forwards CInPacket to a CField virtual method
  (`(*(*this + 20))(this, CUserLocal, a2)` = vtable slot 5) passing the
  CUserLocal singleton. No direct reads at the CField tier (base = stub /
  subclass-override). Opcode-only / no-op at the base CField tier, same as v83.

## SUMMON_ITEM_INAVAILABLE
- fname: `CField::OnSummonItemInavailable` (`?OnSummonItemInavailable@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55a7e8`
- dispatcher: `CField::OnPacket` case `0x91`
- registry opcode: 145 (0x91) — matches dispatcher
- **LAYOUT ≡ v83.** `flag : 1 : Decode1` — bool; if 0, pops the "can't use here"
  notice. Single byte.

## FIELD_OBSTACLE_ONOFF
- fname: `CField::OnFieldObstacleOnOff` (`?OnFieldObstacleOnOff@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55a824`
- dispatcher: `CField::OnPacket` case `0x93`
- registry opcode: 147 (0x93) — matches dispatcher
- **LAYOUT ≡ v83.** `name : str : DecodeStr`, `state : 4 : Decode4`
  (→ SetObjectState).

## FIELD_OBSTACLE_ONOFF_LIST
- fname: `CField::OnFieldObstacleOnOffStatus` (`?OnFieldObstacleOnOffStatus@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55a870`
- dispatcher: `CField::OnPacket` case `0x94`
- registry opcode: 148 (0x94) — matches dispatcher
- **LAYOUT ≡ v83.** `count : 4 : Decode4` (loop runs only if > 0), then loop
  `count`×: `name : str : DecodeStr`, `state : 4 : Decode4`.

## FIELD_OBSTACLE_ALL_RESET
- fname: `CField::OnFieldObstacleAllReset` (`?OnFieldObstacleAllReset@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55a8cf`
- dispatcher: `CField::OnPacket` case `0x95`
- registry opcode: 149 (0x95) — matches dispatcher (registry already carries the
  CSV truncation fix, manual provenance)
- **LAYOUT ≡ v83.** EMPTY BODY — no CInPacket reads. Iterates the field's
  internal obstacle list and calls SetObjectState(name, 0) per entry. Wire
  payload is opcode-only.

## SET_OBJECT_STATE
- fname: `CField::OnSetObjectState` (`?OnSetObjectState@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55f399`
- dispatcher: `CField::OnPacket` case `0xA1`
- registry opcode: 161 (0xA1) — matches dispatcher
- **LAYOUT ≡ v83.** `name : str : DecodeStr`, `state : 4 : Decode4`. Byte-identical
  to FIELD_OBSTACLE_ONOFF.

## SET_QUEST_CLEAR
- fname: `CField::OnSetQuestClear` (`?OnSetQuestClear@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55f22f`
- dispatcher: `CField::OnPacket` case `0x9E`
- registry opcode: 158 (0x9E) — matches dispatcher
- **LAYOUT ≡ v83.** EMPTY BODY — no CInPacket reads. Frees the internal modified-
  quest-time buffer (`sub_562415(CQuestMan + 288)`). Wire payload is opcode-only.

## SET_QUEST_TIME
- fname: `CField::OnSetQuestTime` (`?OnSetQuestTime@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55f242`
- dispatcher: `CField::OnPacket` case `0x9F`
- registry opcode: 159 (0x9F) — matches dispatcher
- **LAYOUT ≡ v83.** `count : 1 : Decode1` (loop runs only if > 0), then loop
  `count`×: `questId : 4 : Decode4`, `startTime : buf(8)`, `endTime : buf(8)`
  (two FILETIMEs) → CQuestMan::SetQuestTime.

## STOP_CLOCK
- fname: `CField::OnDestroyClock` (`?OnDestroyClock@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x5590cf`
- dispatcher: `CField::OnPacket` case `0xA2`
- registry opcode: 162 (0xA2) — matches dispatcher
- **LAYOUT ≡ v83.** EMPTY BODY — destroys the clock window
  (`this[122] -> CWnd::Destroy`). Wire payload is opcode-only.

## GMEVENT_INSTRUCTIONS
- fname: `CField::OnDesc` (`?OnDesc@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55d962`
- dispatcher: `CField::OnPacket` case `0x9A`
- registry opcode: 154 (0x9A) — matches dispatcher
- **LAYOUT ≡ v83.** `index : 1 : Decode1` — bounds-checked index into the help-
  msg array; single byte, no further reads.

## OX_QUIZ
- fname: `CField::OnQuiz` (`?OnQuiz@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55d2da`
- dispatcher: `CField::OnPacket` case `0x99`
- registry opcode: 153 (0x99) — matches dispatcher
- **LAYOUT ≡ v83.** `show : 1 : Decode1`, `category : 1 : Decode1`,
  `number : 2 : Decode2`. Exactly three wire fields (1+1+2); remainder is WZ
  image lookup. (Early-out when number==0.)

## PLAY_JUKEBOX
- fname: `CField::OnPlayJukeBox` (`?OnPlayJukeBox@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x55c9fe`
- dispatcher: `CField::OnPacket` case `0x97`
- registry opcode: 151 (0x97) — matches dispatcher
- **LAYOUT ≡ v83.** `jukeboxItemId : 4 : Decode4`, then `playerName : str :
  DecodeStr` read ONLY when jukeboxItemId >= 0 (guarded by
  get_consume_cash_item_type()==20). Conditional trailing string.

## FOOTHOLD_INFO  (clientbound — PRESENT in v87, NEW vs v83)
- fname: `CField::OnFootHoldInfo` (`?OnFootHoldInfo@CField@@IAEXAAVCInPacket@@@Z`)
- address: `0x560fec`
- dispatcher: `CField::OnPacket` case `0xAA`
- registry opcode: 170 (0xAA) — matches dispatcher (registry row `IDA_0X0AA`,
  fname `CField::OnFootHoldInfo`)
- **DELTA vs v83: VERSION-PRESENT (v83 was version-absent).** Full client read
  order (single-entry form in v87 — no outer count):
  - `name  : str : DecodeStr` — foothold/dynamic-obj name (ZMap key)
  - `mode  : 4   : Decode4` — state/action code (drives FootHoldStateChange)
  - **if `mode == 2`** (the move case) — read a fixed trailing block:
    - `f0 : 4 : Decode4`
    - `f1 : 4 : Decode4`
    - `f2 : 4 : Decode4`
    - `f3 : 4 : Decode4`
    - `f4 : 4 : Decode4`
    - `x  : 4 : Decode4` (new X; compared against prior to trigger FootHoldMove)
    - `y  : 4 : Decode4` (new Y)
    - `f7 : 4 : Decode4`
    - `b0 : 1 : Decode1`
    - `b1 : 1 : Decode1`
  - So mode==2 appends **8× Decode4 + 2× Decode1**. When mode != 2 the body ends
    after `mode`. (No id-list sub-loop in v87 — that is a v95-only addition; see
    gms_v95.md.)

## FOOTHOLD_INFO  (serverbound / request-trigger) — VERSION-ABSENT in v87
- registry rows: `REQUEST_FOOTHOLD_INFO` (sb, op 238) + `FOOTHOLD_INFO` (sb,
  op 239, fname `CField::OnRequestFootHoldInfo`).
- **`CField::OnRequestFootHoldInfo` does NOT exist in the v87 IDB** (`func_query
  name_regex=RequestFootHold` → 0 results) and the dispatcher `CField::OnPacket`
  (0x558b48) has no case routing to it. The v87 CField low-switch ends at case
  0xAA (OnFootHoldInfo); there is no request/send-foothold handler. The
  client→server foothold-info send is therefore absent at the named-handler tier
  in v87 (the `OnRequestFootHoldInfo` send-builder is a v95-era addition; see
  gms_v95.md where it exists @ 0x52ddd0 and builds COutPacket(270)).
- Treat the serverbound FOOTHOLD_INFO row as VERSION-ABSENT for v87.
