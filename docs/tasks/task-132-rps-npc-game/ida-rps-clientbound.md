# IDA verification — `RPS_GAME` clientbound (`CRPSGameDlg::OnPacket`)

**Task 14.** This note is the authoritative source for the RPS_GAME clientbound
frame set, per-mode wire-read order, and per-version mode-byte table. Downstream
packet codecs and byte fixtures (Tasks 15, 20) cite THIS document. Every mode
byte and every field below is transcribed from a decompilation that was actually
run; the decompile address is cited for each claim. Nothing here is inferred from
general MapleStory knowledge.

IDA instances used (confirmed by binary NAME before reading, per
`reference_ida_instance_ports_shifted_idbs_v9`):

| version | port  | binary                    |
|---------|-------|---------------------------|
| v83     | 13342 | `MapleStory_dump.exe`     |
| v84     | 13345 | `GMS_v84.1_U_DEVM.exe`    |
| v87     | 13343 | `GMSv87_4GB.exe`          |
| v95     | 13341 | `GMS_v95.0_U_DEVM.exe`    |
| jms185  | 13344 | `MapleStory_dump_SCY.exe` |

**Status: all five versions VERIFIED.** No version blocked. The jms185 dump
(`_SCY`) decompiled cleanly with full `CRPSGameDlg` symbols — the SMC caveat in
`bug_matrix_redx_unverified_shared_codec` did not apply here.

---

## 0. Executive summary (the load-bearing facts)

1. **The wire format is byte-for-byte identical across all five versions.** Only
   the *opcode* shifts per version; the leading mode byte and every per-mode body
   read are the same. This is unusual for a dispatcher family (contrast the
   `operations`-table shift bug `bug_operations_mode_tables_missing_v87_v95_jms`):
   **RPS mode bytes do NOT shift across versions.**

2. **Only two clientbound modes read a body beyond the mode byte:**
   - **mode 8 (OPEN):** `Decode1`(mode) + `Decode4`(int = participation fee/ante).
   - **mode 11 (RESULT):** `Decode1`(mode) + `Decode1`(NPC throw) + `Decode1`(signed straight-victory count).
   - **All other modes (6, 7, 9, 10, 12, 13, 14) carry only the 1-byte mode** — no
     further packet fields. Their behavior is purely client-state (notices, timers,
     button enable/disable, dialog destroy).

3. Opcodes confirmed against the Global-Constraints table via the `CField::OnPacket`
   dispatch in every version: v83 `0x138`, v84 `0x13F`, v87 `0x149`, v95 `0x173`,
   jms185 `0x151`. ✅ All match.

4. **A tie is NOT a distinct clientbound frame** — it is derived client-side from
   the RESULT (mode 11) data. **`Update` is a client tick** whose only network
   effect is emitting serverbound mode 2 on selection-timeout. (Details in §16.)

### Canonical mode table (identical for all 5 versions)

| mode byte | name (behavior-derived) | body after mode byte | client behavior |
|-----------|-------------------------|----------------------|-----------------|
| 0–5 | *(unused)* | — | switch falls through / `return` — no handler |
| 6 | `FAIL_NOT_ENOUGH_MESO` | none | notice "not enough mesos for participation fee" + reset |
| 7 | `FAIL_NEED_FREE_SLOT` | none | notice "need ≥1 free inventory slot" + reset |
| 8 | `OPEN` | `Decode4` int (ante) | open util confirm dialog + construct `CRPSGameDlg(ante)` |
| 9 | `START_SELECT` | none | enable R/P/S buttons, arm 30 s selection timer |
| 10 | `SHOW_RESULT` | none | reset npc-select, straight-victories = −1, call `ShowResult` |
| 11 | `RESULT` | `Decode1` npcThrow + `Decode1` straightVictoryCount (signed) | store outcome data |
| 12 | `START_SELECT` (alias of 9) | none | identical handler to mode 9 |
| 13 | `CLOSE` | none | `CWnd::Destroy` the dialog window |
| 14 | `RESET` | none | reset buttons/state (game-over cleanup, no notice) |

Notes:
- **Design §6.1 OPEN / RESULT / END map to modes 8 / 11 / 13.** The server must
  also be able to send 6, 7, 9/12, 10, 14; enumerate all of them in the Task-20
  `operations` map.
- Modes **9 and 12 invoke the same handler body** (in the decompile, `case 9:`
  `goto LABEL_13;` which is `case 12:`). They are wire-distinct bytes but
  client-behaviorally identical (enable selection + arm timer).
- The names in the table are *behavior-derived descriptors*, grounded in the
  decompiled handler bodies and (for v95) the symbolized field names. atlas-rps /
  the writer may assign its own symbolic constants; the **byte values** are the
  contract.

---

## 1. v83 — `MapleStory_dump.exe` (port 13342)

### Location
- `CRPSGameDlg::OnPacket` = **`0x73fff1`** (`?OnPacket@CRPSGameDlg@@SAXAAVCInPacket@@@Z`),
  found via `func_query name_regex="RPS"`.
- Per-mode dispatcher (modes 6,7,9–12,14) = `sub_74024B` (`0x74024b`), a
  `CRPSGameDlg` method called from `OnPacket` at `0x740238`.

### Opcode confirmation → `0x138` ✅
`OnPacket` is xref'd from `CField::OnPacket` (`0x531325`) at `0x5317b0`. The upper
dispatch cascade at `loc_5314E8`: `sub edx,0x12F` (`0x5314ea`) → `sub edx,6`
(`0x5314f6`) → `sub edx,3; jz loc_5317AD` (`0x5314ff`/`0x531502`). Accumulated
compare = `0x12F + 6 + 3 = 0x138`; `loc_5317AD` contains the call to
`CRPSGameDlg::OnPacket` (xref at `0x5317b0`). **nType `0x138` → RPS.**

### Mode switch — `OnPacket` (`0x73fff1`)
- Leading mode byte: `v1 = CInPacket::Decode1(a1)` at **`0x740003`**.
- `if (v1 < 6u) return;` (`0x74000e`) — modes 0–5 ignored.
- `if (v1 <= 7u) goto LABEL_23;` (`0x74001c`) — modes 6,7 → dispatcher `sub_74024B`.
- `case 8` (OPEN):
  - `v3 = CInPacket::Decode4(a1)` at **`0x7400ec`** — the ante (int).
  - `StringPool::GetString(...SP_3675_..PARTICIPATION_FEE..)` (`0x7400fe`) — notice
    text is a **static** StringPool string, NOT a packet field (the `a1` slot is
    reused to hold the ZXString after `Decode4`).
  - `sub_73D05B(v10, v3)` (`0x7401b7`) constructs the game dialog with `v3`.
  - **Body read: `Decode4` only.**
- modes 9,10,11,12 (`v1 <= 0xC`) → `LABEL_23` → `sub_74024B` (`0x740238`).
- `case 13` (CLOSE): `CWnd::Destroy(...)` at `0x74009e` — no read.
- `case 14`: → `LABEL_23` → `sub_74024B`.

### Per-mode reads — dispatcher `sub_74024B` (`0x74024b`)
- `case 10`: `this[39]=0; this[36]=-1; this[41]=-1; sub_73EBA9(this)` (`0x7402c8`–`0x7402df`). No read.
- `case 11` (RESULT):
  - `this[36] = CInPacket::Decode1(a3)` at **`0x740298`** — byte1 (NPC throw).
  - `v4  = CInPacket::Decode1(a3)` at **`0x7402a3`** — byte2 (straight-victory count, signed).
  - `if (v4 < 0) v5 = (this[41]==0);  this[42]=v5;  this[41]=v4;` (`0x7402aa`–`0x7402bd`).
  - **Body read: two `Decode1` bytes.**
- `case 9` / `case 12` (`LABEL_13`): `this[36]=-1; this[39]=120;` + `get_update_time()`
  timers + re-enable 3 buttons (`0x7402e9`–`0x740349`). No read.
- `case 6` / `case 7` / `case 14` (fall-through block `0x74034f`+): reset fields,
  then case 7 → notice `SP_3683_..AT_LEAST_ONE_FREE_SLOT..` (`0x740399`), case 6 →
  notice `SP_3684_..NOT_ENOUGH_MESOS..1000_MESOS` (`0x7403af`), case 14 → reset
  only. No read.

*(v83 struct layout uses field indices `this[36]`/`this[41]`/`this[42]`; wire reads
are what matter and are as above.)*

---

## 2. v84 — `GMS_v84.1_U_DEVM.exe` (port 13345)

### Location
- `CRPSGameDlg::OnPacket` = **`0x761d15`** (pre-annotated
  `CRPSGameDlg__OnPacket_recv_0x13F`), via `func_query name_regex="RPS"`.
- Dispatcher = `sub_761F6F` (`0x761f6f`), called from `OnPacket` at `0x761f5c`.

### Opcode confirmation → `0x13F` ✅
`CField::OnPacket` (`0x53d5a7`) upper switch: `case 0x13F: CRPSGameDlg__OnPacket_recv_0x13F(iPacket);`
(decompiled, xref at `0x53da36`). **Confirmed by dispatch, not just the annotated name.**

### Mode switch — `OnPacket` (`0x761d15`)
Structurally identical to v83:
- Mode byte: `Decode1` at **`0x761d27`**.
- `case 8` (OPEN): `v3 = CInPacket::Decode4(a1)` at **`0x761e10`**; `StringPool::GetString(..,3678)`
  (static); `sub_75ED7F(v3)` builds dialog. **Body read: `Decode4` only.**
- modes 6,7,9–12,14 → `sub_761F6F`. `case 13` → destroy (`sub_A28335`, `0x761dc2`), no read.

### Per-mode reads — dispatcher `sub_761F6F` (`0x761f6f`)
- `case 11` (RESULT):
  - `a1[36] = (unsigned __int8)CInPacket::Decode1(a4)` at **`0x761fbc`** — byte1.
  - `v5     = CInPacket::Decode1(a4)` at **`0x761fc7`** — byte2 (signed).
  - `if (v5<0) v6=(a1[41]==0); a1[42]=v6; a1[41]=v5;` (`0x761fce`–`0x761fe1`).
  - **Body read: two `Decode1` bytes.**
- `case 10`: reset + `sub_7608CD` (`0x761fec`+). No read.
- `case 9`/`12`: `a1[39]=120` + timers + enable buttons (`0x76200d`+). No read.
- `case 6`/`7`/`14`: reset; case 7 → notice str `3686` (`0x7620bd`), case 6 → notice
  str `3687` (`0x7620d3`), case 14 → reset only. No read.

Byte-for-byte the same wire format as v83.

---

## 3. v87 — `GMSv87_4GB.exe` (port 13343)

Full `CRPSGameDlg` symbols present (`OnBtStart/Continue/Retry/Exit`,
`SendSelection`, `Update`, `ShowResult`) — used to resolve §16.

### Location
- `CRPSGameDlg::OnPacket` = **`0x78aba7`**.
- Dispatcher (modes 6,7,9–12,14) = `sub_78AE23` (`0x78ae23`), called at `0x78ae10`.

### Opcode confirmation → `0x149` ✅
`CField::OnPacket` (`0x558b48`) decompiled: `case 0x149: CRPSGameDlg::OnPacket(iPacket);`
(xref at `0x559029`). **nType `0x149` → RPS.**

### Mode switch — `OnPacket` (`0x78aba7`)
- Mode byte: `Decode1` at **`0x78abb9`**.
- `case 8` (OPEN): `v3 = CInPacket::Decode4(iPacket)` at **`0x78acb0`**;
  `StringPool::GetString(..,3684)` (static); `sub_787C11(v3)`. **Body read: `Decode4` only.**
- modes 6,7,9–12,14 → `sub_78AE23`. `case 13` → `CWnd::Destroy` (`0x78ac5a`), no read.

### Per-mode reads — dispatcher `sub_78AE23` (`0x78ae23`)
- `case 11` (RESULT):
  - `a1[40] = CInPacket::Decode1(a4)` at **`0x78ae70`** — byte1 (NPC throw).
  - `v5     = CInPacket::Decode1(a4)` at **`0x78ae7b`** — byte2 (signed count).
  - `if (v5<0) v6=(a1[45]==0); a1[46]=v6; a1[45]=v5;` (`0x78ae82`–`0x78ae95`).
  - **Body read: two `Decode1` bytes.**
- `case 10`: reset + `CRPSGameDlg::ShowResult(a1,a2)` (`0x78aeb7`). No read.
- `case 9`/`12`: `a1[43]=120` + `get_update_time()` timers + enable 3 buttons (`0x78aec1`+). No read.
- `case 6`/`7`/`14`: reset; case 7 → notice str `3692`, case 6 → notice str `3693`,
  case 14 → reset only. No read.

*(v87 struct layout uses `a1[40]`/`a1[45]`/`a1[46]` — different indices than v83/v84,
same wire reads.)*

---

## 4. v95 — `GMS_v95.0_U_DEVM.exe` (port 13341)

Fully symbolized **including named struct fields** — this version confirms the
semantic meaning of the RESULT bytes.

### Location
- `CRPSGameDlg::OnPacket` = **`0x6d9e00`**.
- Per-mode helper = `CRPSGameDlg::ProcessPacket` (`0x6d72d0`), a real named method,
  called from `OnPacket` at `0x6da092`.

### Opcode confirmation → `0x173` ✅
`CField::OnPacket` (`0x546d50`) decompiled: `case 371: CRPSGameDlg::OnPacket(iPacket);`
— 371 = **`0x173`** (xref at `0x546f3d`). **nType `0x173` → RPS.**

### Mode switch — `OnPacket` (`0x6d9e00`)
Real `switch(v1)` (`v1 = CInPacket::Decode1(iPacket)` at **`0x6d9e30`**):
- `case 6,7,9,10,11,12,14:` → `CRPSGameDlg::ProcessPacket(this, v1, iPacket)` (`0x6da092`).
- `case 8` (OPEN): `v2 = CInPacket::Decode4(iPacket)` at **`0x6d9e82`** (the ante);
  `StringPool::GetString(..,0xE83)` (static); `CRPSGameDlg::CRPSGameDlg(v8, v2)`
  (`0x6d9f51`). **Body read: `Decode4` only.**
- `case 13` (CLOSE): `CWnd::Destroy(...)` (`0x6d9ff0`). No read.
- `default: return;`

### Per-mode reads — `ProcessPacket` (`0x6d72d0`)
- `case 11` (RESULT):
  - `this->m_nNpcSelect = CInPacket::Decode1(iPacket)` at **`0x6d7372`** — byte1 = **NPC's throw**.
  - `v8 = CInPacket::Decode1(iPacket)` at **`0x6d737d`** — byte2 (`signed __int8`).
  - `if (v8<0) v9 = (this->m_nCntStraightVictories==0);`
    `this->m_bReceiveCompensation = v9;`
    `this->m_nCntStraightVictories = v8;` (`0x6d7384`–`0x6d7399`).
  - **Body read: two `Decode1` bytes.** byte2 = **`m_nCntStraightVictories`** (the
    consecutive-win count / ladder rung); a **negative** value signals game-over/final,
    and if the prior count was 0 it sets **`m_bReceiveCompensation`** (prize flag).
- `case 9,12` (START_SELECT): `m_nNpcSelect=-1; m_tSwitchingTerm=120;
  m_tLimit = get_update_time()+30000;` + enable `m_pBtRPS[0..2]` (`0x6d72ec`–`0x6d735a`). No read.
- `case 10` (SHOW_RESULT): `m_tSwitchingTerm=0; m_nNpcSelect=-1;
  m_nCntStraightVictories=-1; CRPSGameDlg::ShowResult(this)` (`0x6d73a8`–`0x6d73be`). No read.
- `case 6,7,14`: `SetMainButton` + reset; case 7 → notice str `3723`, case 6 →
  notice str `3724`, case 14 → reset only (`0x6d73cb`+). No read.

The v95 field names (`m_nNpcSelect`, `m_nCntStraightVictories`, `m_bReceiveCompensation`,
`m_nUserSelect`, `m_tLimit`, `m_tSwitchingTerm`) retroactively confirm the identical
2-byte RESULT read in v83/v84/v87/jms.

---

## 5. jms185 — `MapleStory_dump_SCY.exe` (port 13344)

Decompiled cleanly with `CRPSGameDlg` symbols (no SMC obstruction for this class).

### Location
- `CRPSGameDlg::OnPacket` = **`0x7ae3dc`**.
- Dispatcher = `sub_7AE636` (`0x7ae636`), called at `0x7ae623`.

### Opcode confirmation → `0x151` ✅
`CField::OnPacket` (`0x56e721`) decompiled: `case 0x151: CRPSGameDlg::OnPacket(iPacket);`
(xref at `0x56ebcf`). **nType `0x151` → RPS.**

### Mode switch — `OnPacket` (`0x7ae3dc`)
Identical structure to v83/v84/v87:
- Mode byte: `Decode1` at **`0x7ae3ee`**.
- `case 8` (OPEN): `v3 = CInPacket::Decode4(iPacket)` at **`0x7ae4d7`**;
  `StringPool::GetString(..,0xEFC)` (static); `CRPSGameDlg::CRPSGameDlg(v10, v3)`
  (`0x7ae5a2`). **Body read: `Decode4` only.**
- modes 6,7,9–12,14 → `sub_7AE636`. `case 13` → `CWnd::Destroy` (`0x7ae489`), no read.

### Per-mode reads — dispatcher `sub_7AE636` (`0x7ae636`)
- `case 11` (RESULT):
  - `a1[40] = CInPacket::Decode1(a4)` at **`0x7ae683`** — byte1 (NPC throw).
  - `v5     = CInPacket::Decode1(a4)` at **`0x7ae68e`** — byte2 (signed count).
  - `if (v5<0) v6=(a1[45]==0); a1[46]=v6; a1[45]=v5;` (`0x7ae695`–`0x7ae6a8`).
  - **Body read: two `Decode1` bytes.**
- `case 10`: reset + `sub_7ACF94(a1,a2)` (ShowResult) (`0x7ae6b3`+). No read.
- `case 9`/`12`: `a1[43]=120` + timers + enable buttons (`0x7ae6d4`+). No read.
- `case 6`/`7`/`14`: reset; case 7 → notice str `3844`, case 6 → notice str `3845`,
  case 14 → reset only. No read.

Byte-for-byte identical wire format to the four GMS versions.

---

## 6. Per-version opcode + mode-byte table (feeds Task-20 `operations`)

**Opcode (shifts per version):**

| version | RPS_GAME clientbound opcode | source |
|---------|-----------------------------|--------|
| v83     | `0x138` | `CField::OnPacket` cascade `0x12F+6+3` → `loc_5317AD` |
| v84     | `0x13F` | `CField::OnPacket` `case 0x13F` |
| v87     | `0x149` | `CField::OnPacket` `case 0x149` |
| v95     | `0x173` | `CField::OnPacket` `case 371` |
| jms185  | `0x151` | `CField::OnPacket` `case 0x151` |

All match the design/STATUS.md Global-Constraints table. ✅

**Mode bytes (IDENTICAL across all five versions — no per-version shift):**

| mode name | byte | body reads (after mode byte) |
|-----------|------|------------------------------|
| `FAIL_NOT_ENOUGH_MESO` | 6  | — |
| `FAIL_NEED_FREE_SLOT`  | 7  | — |
| `OPEN`                 | 8  | `Decode4` int (ante) |
| `START_SELECT`         | 9  | — |
| `SHOW_RESULT`          | 10 | — |
| `RESULT`               | 11 | `Decode1` npcThrow, `Decode1` straightVictoryCount (signed) |
| `START_SELECT` (alias) | 12 | — |
| `CLOSE`                | 13 | — |
| `RESET`                | 14 | — |

Because the mode bytes are uniform, the Task-20 `operations` table for RPS is the
same for gms_83 / gms_84 / gms_87 / gms_95 / jms_185 — only the opcode row differs.

---

## Design §16 resolutions (with decompile evidence)

### §16 Item 2 — OnBtRetry vs tie-redraw; is a tie a distinct frame or an outcome code?

**Answer: a tie is NEITHER a distinct clientbound frame NOR a distinct outcome
mode. It is derived entirely client-side from the RESULT frame (mode 11).**

Evidence:
- The `OnPacket` switch has **no tie mode** — the only clientbound modes are 6–14
  (§0 table), none tie-specific.
- The RESULT frame (mode 11) delivers `[npcThrow, straightVictoryCount]` (two
  `Decode1` bytes; §4 v95 = `m_nNpcSelect`, `m_nCntStraightVictories`).
- The client detects the tie itself in `ShowResult` (v87 `0x78975f`, v95 `0x6d5350`):
  the win/tie branch is entered when the result byte is non-negative
  (`if (*(this+45) >= 0)`, v87 `0x78977c`), and **tie vs win is decided by comparing
  the player's own selection against the NPC throw** — v87 `0x7899a2`:
  `if (*(this+39) == *(this+40))` → tie branch (emotion/strings 10–14) else win
  (strings 15–19). `*(this+39)` is the local `m_nUserSelect` set by `SendSelection`
  (v95 `0x6d6b46 this->m_nUserSelect = nRPS`); `*(this+40)` is the received NPC throw.
- On a tie the client **auto-re-enables the R/P/S selection buttons locally** with no
  server round-trip — `CRPSGameDlg::Update` v87 `0x78a11f`–`0x78a178`
  (`if (*(this+156)==*(this+160)) { *(this+160)=-1; *(this+172)=120; ...enable 3 buttons... }`).
  The player simply re-selects (serverbound mode 1 again).
- **`OnBtRetry` is a separate serverbound action, not a tie mechanic.** It sends
  serverbound mode **5** (v87 `0x78b090 Encode1(5)`, v95 `0x6d69a0 Encode1(5)`,
  jms `0x7ae8a3 Encode1(5)`) — the post-**loss** "play again" button. It is distinct
  from `OnBtContinue` = serverbound mode **3** (v87 `0x78b01c Encode1(3)`), the
  post-**win** "advance the ladder" button.

**Implication for atlas-rps / the writer:** on a tie, the server re-issues the
round-start frames (mode 9/12 `START_SELECT` + the mode-11 `RESULT` carrying the
tie throws); there is no separate tie opcode or tie mode to encode. Model tie as an
outcome computed from the RESULT payload, and keep the rung unchanged / status
back to awaiting-select (which is exactly what design §5.1/§6.1 already assume).

### §16 Item 3 — the `Update` sub-action: client-only tick or server-driven?

**Answer: `CRPSGameDlg::Update` is a per-frame client UI tick (a `CWnd::Update`
override) that is server-relevant in exactly ONE path — it emits serverbound mode 2
when the selection countdown timer expires.**

Evidence (v87 `CRPSGameDlg::Update` `0x789cf6`; the largest RPS function, `0xd69`
bytes — matches the v84 pre-annotation `CRPSGameDlg__Update_send_0x8C`):
- It drives avatar animation (`CAvatar::Update` `0x789d23`), the R/P/S button-cycle
  animation, minigame sounds, and countdown-text rendering — all client-local.
- Its **only** network I/O: when the selection-limit timer elapses
  (`*(this+176)` path, `0x789dfc` false branch), it builds a `COutPacket(0x90)` and
  `COutPacket::Encode1(&pvarg, 2u)` then `CClientSocket::SendPacket`
  (**`0x789e06`–`0x789e26`**) — **serverbound mode 2**, a client-driven
  *selection-timeout* notification. No periodic keepalive is sent otherwise.

So from the clientbound side: `Update` is a client tick, **except** it auto-emits
serverbound mode 2 on selection timeout. Whether atlas-rps must act on mode 2
(auto-pick / forfeit the round) is the **serverbound** question deferred to Task 16;
the clientbound evidence establishes that mode 2 is a one-shot timeout signal, not a
poll. (Serverbound sender map, for Task-16 cross-reference — all `Encode1(mode)` on
opcode `0x90`/v87, `0xA0`/v95, `0x8B`/jms: `0`=OnBtStart, `1`=SendSelection+choice,
`2`=Update-timeout, `3`=OnBtContinue, `5`=OnBtRetry; mode `4` not observed.)

---

## Self-check

Every mode byte in §0/§6 and every field read in §1–§5 cites a concrete decompile
address from a decompilation actually run in this session. Opcodes are confirmed by
the `CField::OnPacket` dispatch in each IDB (not by the STATUS.md table alone). No
value here is sourced from memory or general MapleStory knowledge. No version is
blocked.
