# IDA verification notes — task-133 miniroom minigames (gates G1–G5)

Byte-layout source of truth for the Omok / Match Cards packet work (tasks 2–8), the
gameplay engine (task 15) and the seed templates (task 20). Every claim below cites the
decompiled client function it was derived from (function name + address + snippet). Two
IDBs were used:

- **v83** — `MapleStory_dump.exe` (IDA port 13342, imagebase 0x400000). Primary reference.
- **v95** — `GMS_v95.0_U_DEVM.exe` (IDA port 13341). Cross-check for G3 + G5.

Both instances were matched by binary name via `list_instances` (the port set rotates).

Shared facts used throughout:

- The client uses **one mode enum for both directions** (serverbound + clientbound).
  A dialog receives via `CMiniRoomBaseDlg::OnPacketBase` (v83 `0x65df4c`) whose `default`
  arm calls the dialog vtable `OnPacket(nType, pkt)`; the serverbound sends reuse the same
  numbers. The seeded serverbound table `template_gms_83_1.json:571-584` is therefore
  authoritative for the clientbound modes too on v83/v84.
- **Player slot** (`m_nMyPosition`, v83 field `this[50]`): a byte the SERVER assigns to each
  client in the enter-result. `CMiniRoomBaseDlg::OnEnterResultBase` (v83 `0x65ec3d`) reads
  `*(this+51)=Decode1` (capacity) then `*(this+50)=Decode1` (that client's own slot). Slot
  `0` = owner/creator, `1` = visitor. Every "is this me?" test in the handlers compares a
  decoded slot byte against `this[50]` / `m_nMyPosition`.
- **Stone/piece color** is a **1-based** value (1 or 2), distinct from the 0-based slot.
  `COmokDlg::OnUserStart` sets `m_nPlayerColor = 2 - (startByte != mySlot)`.

---

## G1 start-byte

**Resolved rule.** The `START` (mode 61) body's first byte is a **player slot index**
(0 = owner, 1 = visitor). The client grants the first move to the player whose slot is
**NOT equal** to the START byte; the player whose slot **equals** the byte becomes the
second mover (and is assigned `m_nPlayerColor = 2`). Match Cards `START` prepends the same
first-mover slot byte, then a card count, then `count` little-endian int32 card ids.

Server implication: send `START` byte = the slot of the **second mover** (per Cosmic = the
previous winner's slot; initial value `1`), and initialise server `currentTurn` (first
mover) to the **other** slot. With Cosmic's values this yields: first game → owner moves
first (byte 1 ⇒ first mover = slot 0); after an owner win → visitor moves first (byte 0);
after a visitor win → owner moves first (byte 1). I.e. the **loser of the previous game
moves first**, which matches Cosmic's extracted `0`-after-owner-win / `1`-after-visitor-win /
initial-`1` (design §13/G1). The client is authoritative for the grant direction; Cosmic is
authoritative for the byte's raw value.

**Evidence — Omok START handler (v83 `COmokDlg::OnUserStart` @ 0x6e469c):**
```
v3 = CInPacket::Decode1(a2) != this[50];   // v3 = (startByte != mySlot)
this[694] = v3;                            // turn flag
this[692] = !v3 + 1;                       // my color: 2 if startByte==mySlot else 1
```
v95 confirms with typed names (`COmokDlg::OnUserStart` @ 0x684a00):
```
v3 = CInPacket::Decode1(iPacket) != this->m_nMyPosition;
this->m_bCurTurn = v3;
this->m_nPlayerColor = 2 - v3;
```

**Turn-flag polarity is nailed by the move-send gate.** The board-click handler only
sends a stone when the turn flag == 1 (v83 `sub_6E4D1F` @ 0x6e4d1f, the caller of the
move-send `COmokDlg::PutStoneChecker` @ 0x6e8a19):
```
if ( this[693] )            // game in progress
  if ( this[694] == 1 )     // == my turn
     ... COmokDlg::PutStoneChecker(v6, x, y);   // send MOVE_STONE
```
Corroboration from `COmokDlg::OnPutStoneChecker` (v83 @ 0x6e3f5b): receiving a stone of
**my** color sets `this[694]=0` (I just moved → not my turn); an opponent stone sets
`this[694]=1` (my turn). Therefore `this[694]==1 ⇔ my turn`, and since START sets
`this[694] = (startByte != mySlot)`, **first mover = slot ≠ startByte**.

**Match Cards START body (v83 `CMemoryGameDlg::OnUserStart` @ 0x64e632):**
```
v3 = CInPacket::Decode1(a2);                 // byte: first-mover slot
v4 = CInPacket::Decode1(a2);                 // byte: card count (12/20/30)
this[480] = v4;
ZArray<long>::_Alloc(this+132, v4, ...);
CInPacket::DecodeBuffer(a2, this[132], 4*this[480]);  // count × int32 card id
this[463] = v3 != this[50];                  // same "!= mySlot" first-mover test
```
So Match Cards `START` = `byte firstMoverSlot, byte count, count × int32 cardId`.

---

## G2 retreat

No Cosmic reference exists (design D10) — the v83/v95 client is the sole authority. Retreat
uses mode **54 = ASK_RETREAT** and mode **55 = RETREAT_ANSWER** (both directions, single
enum). Verified on **v83 and v95** (brief required only v83).

**Serverbound ASK_RETREAT (mode 54) is bodyless.** `COmokDlg::SendRetreatRequest`
(v83 @ 0x6e8bc2), gated on having ≥1 stone placed and no pending request:
```
COutPacket::COutPacket(v9, 123);   // opcode 0x7B
COutPacket::Encode1(v9, 54u);      // mode 54, no body
```

**Clientbound ASK_RETREAT (mode 54) is bodyless** — the receiver just shows a Yes/No prompt
and replies serverbound. `COmokDlg::OnRetreatRequest` (v83 @ 0x6e416b):
```
COutPacket::COutPacket(v10, 123);
COutPacket::Encode1(v10, 0x37u);                       // reply mode 0x37 = 55 = RETREAT_ANSWER
LOBYTE(v9) = CUtilDlg::YesNo(...) == 6;                // accept bool
COutPacket::Encode1(v10, v9);                          // serverbound body: byte accept
```
This matches the existing serverbound decoder
`libs/atlas-packet/interaction/serverbound/operation_memory_game_retreat_answer.go`
(reads one `bool`).

**Clientbound RETREAT_ANSWER (mode 55) — the load-bearing layout.**
`COmokDlg::OnRetreatResult` (v83 @ 0x6e41f9):
```
if ( CInPacket::Decode1(a2) ) {        // byte0: accept (1 = accepted)
   v4  = CInPacket::Decode1(a2);       // byte1: N = number of stones to pop
   v19 = CInPacket::Decode1(a2);       // byte2: slot whose turn follows
   // loop N times: pop the tail stone from the board; decrement my stone count
   //               when the popped stone's color == my color
   if ( this[50] == v19 ) this[694] = 1;   // turnSlot==mySlot -> my turn
   else                   this[694] = 0;
} else {
   // decline: show SP_464 "your opponent denied your request"
}
```
v95 confirms (`COmokDlg::OnRetreatResult` @ 0x684620):
```
if ( CInPacket::Decode1(iPacket) ) {
   v5         = CInPacket::Decode1(v3);   // N stones to pop
   nCurTurnIdx = CInPacket::Decode1(v3);  // turn slot
   // loop: ZList<STONELAYER>::RemoveAt(tail); if tail color==m_nPlayerColor --m_nMyStoneNo
   if ( this->m_nMyPosition == nCurTurnIdx ) this->m_bCurTurn = 1;
   else                                      this->m_bCurTurn = 0;
} else { AddChatText(SP "opponent denied", red) }
```

**RETREAT_ANSWER wire layout (clientbound, mode 55):**
```
byte accept
if accept == 1:
    byte N         # number of stones the client pops from the tail of the move history
    byte turnSlot  # slot whose turn it is after the pop; client sets my-turn = (turnSlot == mySlot)
```
On decline the body is just `byte accept(0)`. **The client pops exactly N stones from the
tail and honours `turnSlot` verbatim** — N and turnSlot are chosen by the server; the server
board must mirror the same N-stone pop. (Task 15 decides N and turnSlot; the wire supports
any values.)

---

## G3 balloon

**Resolved layout — identical on v83 and v95**, and it exactly matches the existing
`MiniRoomBase.Spawn` writer (`libs/atlas-packet/interaction/mini_room.go:69-85`). The
handler `CUser::OnMiniRoomBalloon` reads the trailing fields; the leading `int32 characterId`
is consumed by the packet router to locate the `CUser` (the writer emits it first). There is
**no per-roomType (4/5 shop) branch** — the read order is uniform for all room types, so
mapping the `MiniRoom` writer to this packet cannot crash merchant/personal-shop balloons.

**v95 (typed field names, `CUser::OnMiniRoomBalloon` @ 0x8e8d30):**
```
v4 = CInPacket::Decode1(iPacket); this->m_nMiniRoomType = v4;   // byte type; 0 => destroy balloon
if ( v4 ) {
   this->m_dwMiniRoomSN  = CInPacket::Decode4(v3);              // int32 roomId / serialNumber
   ... DecodeStr -> this->m_sMiniRoomTitle;                     // string title
   this->m_bPrivate  = CInPacket::Decode1(v3);                  // byte private
   this->m_nGameKind = CInPacket::Decode1(v3);                  // byte gameKind / pieceType
   this->m_nCurUsers = CInPacket::Decode1(v3);                  // byte occupancy (current users)
   this->m_nMaxUsers = CInPacket::Decode1(v3);                  // byte capacity (max users)
   this->m_bGameOn   = CInPacket::Decode1(v3);                  // byte inProgress
} else {
   CChatBalloon::DestroyMiniRoomBalloon(...);                   // type 0 removes the balloon
}
```

**v83 (`CUser::OnMiniRoomBalloon` @ 0x938ba5)** reads the same order (fields land at
out-of-numeric-order offsets but the read sequence is identical):
```
v4 = Decode1; this[1958]=v4;             // type
if (v4) {
   this[1959] = Decode4;                 // int32 roomId
   DecodeStr -> this[1960];              // title
   this[1961] = Decode1;                 // private
   this[1962] = Decode1;                 // gameKind
   this[1964] = Decode1;                 // occupancy   (read 3rd of the trailing bytes)
   this[1963] = Decode1;                 // capacity    (read 4th)
   this[1965] = Decode1;                 // inProgress
} else DestroyMiniRoomBalloon;
```
(v83 offset cross-check: `this[1959]`=roomId and `this[1961]`=private are exactly the fields
the double-click enter path reads back — see G4 — confirming the mapping.)

**Balloon wire layout (clientbound UPDATE_CHAR_BOX):**
```
int32  characterId      # emitted by writer; consumed by router (not by OnMiniRoomBalloon)
byte   roomType         # 0 = remove balloon
if roomType != 0:
    int32  roomId       # == owner character id (design D2)
    string title
    byte   private
    byte   pieceType    # gameKind
    byte   occupancy    # current users
    byte   capacity     # == 2 for games
    byte   inProgress
```
Task 20 note: opcodes already present-but-unverified in the registries — 0x0A5 (v83) /
0x0B8 (v95). This layout is verified for both.

---

## G4 visit

**Resolved rule.** The serverbound game-room join is **mode 4** (ENTER/VISIT), NOT the
trade-shaped body that `libs/atlas-packet/interaction/serverbound/operation_visit.go` decodes
today. The client sends `int32 serialNumber` (the room id from the balloon) + a
`byte hasPassword` + an optional password string + a trailing constant `byte 0`. Verified on
v83 (the send path). This confirms Cosmic's "serialNumber + password" shape.

**Evidence — v83 `CUserLocal::HandleLButtonDblClk` @ 0x94fbbf** (double-click on a user's
miniroom balloon; `CUserPool::FindBalloon` returns the target remote user `v9`):
```
v10 = *(Balloon + 7844);                 // byte at +7844 = target's m_bPrivate  (== this[1961], G3)
v11 = 0;
if ( v10 ) {                             // private room -> prompt for password
   StringPool::GetString(..., SP_470_PLEASE_ENTER_THE_PASSWORD);
   ... CUtilDlgEx::GetInputStr_Result -> v45;   // typed password
   v11 = 1;
}
COutPacket::COutPacket(&v38, 123);       // opcode 0x7B
COutPacket::Encode1(&v38, 4u);           // mode 4 = ENTER / VISIT
COutPacket::Encode4(&v38, *(v9 + 7836)); // int32 serialNumber (target's m_dwMiniRoomSN == this[1959], G3)
COutPacket::Encode1(&v38, v11);          // byte hasPassword (1 if private)
if ( v11 )
   COutPacket::EncodeStr(&v38, ...v45);  // string password
COutPacket::Encode1(&v38, 0);            // byte 0 (constant trailing)
CClientSocket::SendPacket(...);
```
(The employee/merchant-store branch of the same function sends the same mode 4 with
`Encode4(serial), Encode1(0), Encode1(0)` — no password — so the game and shop visit share
the ENTER opcode/mode.)

**Serverbound VISIT wire layout (mode 4):**
```
byte   mode = 4
int32  serialNumber        # room id (owner character id)
byte   hasPassword         # 0 / 1
if hasPassword: string password
byte   0                   # constant trailing byte
```
Task 6 note: `operation_visit.go` (serialNumber, errorCode, errorMessage, something bool,
unk1, cashSerial) is the trade-shaped **clientbound EnterResult** decoder, not this send —
a game-room VISIT decoder is required.

For completeness, the **clientbound EnterResult** (mode 5, `getMiniRoomError`) read order is
`byte roomType (0 = error); if 0: byte errorCode` (`CMiniRoomBaseDlg::OnEnterResultStatic`
@ 0x65dff3). The error-code strings confirm design §3.4: 1 = room already closed,
2 = full capacity, 4 = you're dead, 6 = character unable, 11 = can't start game here,
13 = can't establish miniroom here, 22 = wrong password.

---

## G5 modes+layouts

**Mode values — verified byte-for-byte identical on v83 and v95** (enum is stable across
versions), and they agree with the seeded serverbound table `template_gms_83_1.json:571-584`.

| Meaning (clientbound handler) | Mode | v83 dispatch | v95 dispatch |
|---|---|---|---|
| ASK_TIE (`OnTieRequest`) | **50** | COmokDlg::OnPacket 0x6e37eb | COmokDlg::OnPacket 0x688b70 |
| TIE_ANSWER / result (`OnTieResult`) | **51** | " | " |
| GIVE_UP / FORFEIT (serverbound only) | **52** | seed :573 | seed :573 |
| ASK_RETREAT (`OnRetreatRequest`) | **54** | " | " |
| RETREAT_ANSWER (`OnRetreatResult`) | **55** | " | " |
| EXIT_AFTER_GAME (sb) | **56** | seed :576 | seed :576 |
| CANCEL_EXIT_AFTER_GAME (sb) | **57** | seed :577 | seed :577 |
| READY (`OnUserReady`) | **58** | " | " |
| UNREADY (`OnUserCancelReady`) | **59** | " | " |
| EXPEL (sb) | **60** | seed :580 | seed :580 |
| START (`OnUserStart`) | **61** | " | " |
| RESULT / GET_RESULT (`OnGameResult`, clientbound-only) | **62** | " | " |
| SKIP / time-over (`OnTimeOver`) | **63** | " | " |
| MOVE_STONE (`OnPutStoneChecker`) | **64** | " (Omok only) | " |
| MOVE_STONE error (`OnPutStoneCheckerErr`) | **65** | " (Omok only) | " |
| FLIP_CARD (`OnTurnUpCard`) | **68** | CMemoryGameDlg::OnPacket 0x64db30 (MatchCards only) | 0x634020 |

v83 Omok dispatch (`COmokDlg::OnPacket` @ 0x6e37eb) cases: 50,51,54,55,58,59,61,62,63,64,65.
v83 MatchCards dispatch (`CMemoryGameDlg::OnPacket` @ 0x64db30) cases: 50,51,58,59,61,62,63,68.
v95 identical (`COmokDlg::OnPacket` @ 0x688b70, `CMemoryGameDlg::OnPacket` @ 0x634020).

### START (Omok / Match Cards)
See §G1: Omok = `byte firstMoverSlot`; Match Cards = `byte firstMoverSlot, byte count,
count × int32 cardId`.

### MOVE_STONE (mode 64)
`COmokDlg::OnPutStoneChecker` (v83 @ 0x6e3f5b):
```
CInPacket::DecodeBuffer(arg0, this+707, 8);   // 8-byte buffer = int32 x, int32 y
v4 = CInPacket::Decode1(arg0);                // byte stoneType (== placing player's color 1/2)
```
v95 (`@ 0x6866a0`) reads `DecodeBuffer(&m_pt, 8)` (tagPOINT x,y) then `Decode1` stoneType —
identical. **Wire: `int32 x, int32 y, byte stoneType`.**

### SELECT_CARD / FLIP_CARD (mode 68)
`CMemoryGameDlg::OnTurnUpCard` (v83 @ 0x64e1c1, v95 @ 0x62f060):
```
v3 = Decode1;                 // byte turn: 1 = first flip, 0 = second flip
v4 = Decode1;                 // byte slot (index of the card being turned up)
if ( !v3 ) v13 = Decode1;     // byte firstSlot  (second flip only)
...
if ( !v3 ) {
   v5 = Decode1;              // byte type  (second flip only)
   if ( v5 < 2 )  ... mismatch   // v5 < m_nMaxUsers(=2)
   else           ... match      // ++score[v5 - 2]  => owner(0)/visitor(1) pair
}
```
`m_nMaxUsers == 2` (v95 typed) is the mismatch/match threshold. **Wire:**
```
byte turn                    # 1 = first flip, 0 = second flip
byte slot
if turn == 0:
    byte firstSlot
    byte type                # 0 owner-mismatch, 1 visitor-mismatch, 2 owner-match, 3 visitor-match
```
(First flip is forwarded to the opponent only; second flip to both — design §3.2.)

### RESULT / GET_RESULT (mode 62) — three shapes
`COmokDlg::OnGameResult` (v83 @ 0x6e4463) and `CMemoryGameDlg::OnGameResult`
(v83 @ 0x64e423) are byte-identical in shape:
```
v3 = Decode1;                 // byte resultType
if ( v3 == 1 ) {              // TIE  -> no winner byte
   ...
} else {                     // WIN (0) or FORFEIT-win (2)
   v9 = Decode1;              // byte winnerSlot; "you win" if winnerSlot == mySlot
}
sub_4E42FC(pkt);              // record blob A  (20 bytes)
sub_4E42FC(pkt);              // record blob B  (20 bytes)
```
`sub_4E42FC` (v83 @ 0x4e42fc) = `DecodeBuffer(ptr, 0x14)` — exactly **20 bytes = 5 × int32**,
matching the existing `MiniGameRecord{Unknown, Wins, Ties, Losses, Points}`
(`mini_room.go:348-366`). **Wire:**
```
byte resultType              # 1 = tie; else (win/forfeit) a winnerSlot byte follows
if resultType != 1: byte winnerSlot
<20-byte record>             # player A (owner)
<20-byte record>             # player B (visitor)
```
The three "shapes" are: tie (`01 + 2 records`), win (`00 + winnerSlot + 2 records`),
forfeit (`02 + winnerSlot + 2 records`). resultType is stored raw by the client; only
`==1` (tie) suppresses the winnerSlot byte.

### SKIP (mode 63) — the byte after the mode
`COmokDlg::OnTimeOver` (v83 @ 0x6e472e):
```
v3 = this[50];                              // my slot
this[694] = (v3 == CInPacket::Decode1(a2)); // byte = slot whose turn it now becomes
```
The single byte is the **slot whose turn it now is** (i.e. the non-skipper). The client sets
my-turn = `(byte == mySlot)`. This reconciles with Cosmic's "owner 0x01 / visitor 0x00"
(design §6.1): when the **owner (slot 0) skips**, the turn passes to the visitor, so the byte
= `1`; when the **visitor (slot 1) skips**, the byte = `0`. **Wire: `byte turnSlot`** (the
slot to move next). No contradiction with Cosmic once read as "next mover", not "who skipped".

### READY / UNREADY (modes 58 / 59)
Bodyless (mode only). `COmokDlg::OnUserReady` (v83 @ 0x6e4608) / `OnUserCancelReady`
(@ 0x6e466e) read no packet fields — they just toggle the ready button UI.

### Room-enter blob (`getMiniGame`) — IMPORTANT: differs from the current model
The full room snapshot sent to a joiner (clientbound EnterResult success path, mode 5) is
assembled by `CMiniRoomBaseDlg::OnEnterResultStatic` (0x65dff3) →
`CMiniRoomBaseDlg::OnEnterResultBase` (0x65ec3d) → the dialog's virtual `OnEnterResult`
(`COmokDlg::OnEnterResult` 0x6e388e). Read order:
```
byte roomType                              # OnEnterResultStatic: nonzero => success; creates the dialog
byte capacity                              # OnEnterResultBase: *(this+51)
byte yourSlot                              # OnEnterResultBase: *(this+50) = m_nMyPosition   <-- server tells you your slot
# --- avatar list (OnEnterResultBase while-loop, 0xFF-terminated) ---
repeat:
    byte slot                              # < 0 (0xFF) terminates
    <avatar blob>                          # CMiniRoomBaseDlg::DecodeAvatar
    string name
# --- record list (dialog OnEnterResult while-loop, 0xFF-terminated) ---
repeat:
    byte slot                              # 0xFF terminates
    <20-byte record>                       # sub_4E42FC
string title
byte  gameKind                             # 0 for Omok, 2 for Match Cards (MiniRoomType byte-2 note, design §6.1)
byte  tournament                           # bool
if tournament: byte round
```
**Discrepancies vs the current `GameMiniRoom.Enter` model (`mini_room.go:121-140`) that the
packet task (2–8) MUST reconcile with a byte fixture:**
1. There is a **`yourSlot` byte** after `capacity` that the current model does not write.
2. Avatars+names and the 20-byte records are in **two separate 0xFF-terminated lists**, not
   interleaved per visitor as the current model encodes them.
3. For game rooms the owner entry in the avatar loop has a special int32 branch
   (`OnEnterResultBase` `Decode4` when `!(vtable+92) && slot==0`) that needs a byte fixture to
   pin down; flagged for tasks 2–8 rather than guessed here.
This is the "fresh fixture against getMiniGame/getMatchCard" the design (§6.1) called for.

---

## Coverage summary

| Gate | v83 | v95 | Status |
|---|---|---|---|
| G1 start-byte | ✔ (OnUserStart + move-send gate) | ✔ (structural) | RESOLVED |
| G2 retreat | ✔ | ✔ (bonus) | RESOLVED |
| G3 balloon | ✔ | ✔ | RESOLVED (uniform, no shop branch) |
| G4 visit | ✔ (HandleLButtonDblClk) | n/a (brief: v83) | RESOLVED |
| G5 modes+layouts | ✔ | ✔ (modes identical) | RESOLVED (+ room-enter model discrepancy flagged) |

No unresolved fnames. No decompile contradicted both Cosmic and the seed. The one nuance to
carry forward (client grants first move to slot ≠ START byte; and the room-enter blob differs
from the current interleaved model) is documented above with citations, not left as a guess.
