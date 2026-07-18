# IDA verification — `RPS_ACTION` serverbound (`CRPSGameDlg` senders)

**Task 16.** This note is the authoritative source for the RPS_ACTION serverbound
sub-op mode table, each arm's body layout, and the per-version opcode. Downstream
serverbound codec + byte fixtures (Task 17) and the seed `operations` tables
(Task 20) cite THIS document. Every sub-op byte and every field below is
transcribed from a decompilation that was actually run this session; the decompile
address is cited for each claim. Nothing here is inferred from general MapleStory
knowledge. Companion clientbound note: `ida-rps-clientbound.md` (Task 14).

IDA instances used (confirmed by binary NAME before reading, per
`reference_ida_instance_ports_shifted_idbs_v9`):

| version | port  | binary                    |
|---------|-------|---------------------------|
| v83     | 13342 | `MapleStory_dump.exe`     |
| v84     | 13345 | `GMS_v84.1_U_DEVM.exe`    |
| v87     | 13343 | `GMSv87_4GB.exe`          |
| v95     | 13341 | `GMS_v95.0_U_DEVM.exe`    |
| jms185  | 13344 | `MapleStory_dump_SCY.exe` |

**Status: all five versions VERIFIED.** No version blocked. Every sender was
decompiled and its `COutPacket(opcode)` / `Encode1(mode)` / `Encode1(throw)` /
`SendPacket` sequence read directly.

---

## 0. Executive summary (the load-bearing facts)

1. **The serverbound wire format is byte-for-byte identical across all five
   versions.** Only the *opcode* shifts per version; the leading sub-op byte each
   sender writes, and its trailing body, are the same everywhere. **RPS_ACTION
   sub-op mode bytes do NOT shift across versions** (mirrors the clientbound side).

2. **There are six senders and six sub-op bytes:**

   | sender | sub-op byte | body after sub-op byte |
   |--------|-------------|------------------------|
   | `OnBtStart`      | **0** | none (bodyless) |
   | `SendSelection`  | **1** | `Encode1` throw (1 byte, raw `0`/`1`/`2`) |
   | `Update` (selection-timeout) | **2** | none (bodyless) |
   | `OnBtContinue`   | **3** | none (bodyless) |
   | `OnBtExit`       | **4** | none (bodyless) |
   | `OnBtRetry`      | **5** | none (bodyless) |

   **Only `SendSelection` (mode 1) carries a body** — a single `Encode1` throw
   byte. All other arms are the sub-op byte alone.

3. **The throw byte is a raw `0`/`1`/`2`.** In every version `OnButtonClicked`
   maps R/P/S button ids `0x7D0`/`0x7D1`/`0x7D2` (2000/2001/2002) to
   `SendSelection(this, nId - 2000)`, and `SendSelection` does `Encode1(1)` then
   `Encode1(throw)` with that `0`/`1`/`2` value unmodified. (v95 stores it as
   `this->m_nUserSelect = nRPS`.) The semantic mapping of `0`/`1`/`2` to
   rock/paper/scissors is a UI-button-order convention; the *byte* contract is
   raw `0`/`1`/`2`.

4. **`Update` mode 2 is a one-shot selection-timeout notification, not a poll.**
   It is emitted from `CRPSGameDlg::Update` only in the branch where the selection
   limit timer has elapsed (current time > `m_tLimit`), then the timer is cleared
   and the R/P/S buttons are disabled. No periodic keepalive is sent otherwise.

### Canonical serverbound sub-op table (identical for all 5 versions)

| sub-op byte | sender | body | meaning |
|-------------|--------|------|---------|
| 0 | `OnBtStart`     | — | begin a game / ante-in |
| 1 | `SendSelection` | `Encode1` throw (`0`/`1`/`2`) | submit this round's R/P/S choice |
| 2 | `Update`        | — | client-side selection countdown expired (auto-timeout) |
| 3 | `OnBtContinue`  | — | post-**win** "continue the winning-streak challenge" |
| 4 | `OnBtExit`      | — | quit / leave the game (forfeit) |
| 5 | `OnBtRetry`     | — | post-**loss** "restart / try again" (costs the participation fee) |

Because the sub-op bytes are uniform, the Task-20 `operations` table for
RPS_ACTION serverbound is the same for gms_83 / gms_84 / gms_87 / gms_95 /
jms_185 — only the opcode row differs.

---

## 1. v83 — `MapleStory_dump.exe` (port 13342)  → opcode `0x088` ✅

Only `CRPSGameDlg::OnPacket` is symbolized; the senders were located by their
signature. The four bodyless senders are four consecutive `0x74`-byte functions
directly after the per-mode dispatcher, followed by the `0x8b`-byte
`SendSelection` — the exact size/layout signature confirmed in the symbolized
v87/jms185 IDBs. `OnButtonClicked` was reached via the sole xref to
`SendSelection`.

### Senders
| sender | fn addr | `COutPacket(opcode)` | `Encode1(mode)` | body |
|--------|---------|----------------------|-----------------|------|
| `OnBtStart`     | `sub_7403D0` | `0x88` @ `0x7403e8` | `0` @ `0x7403f6` | none |
| `OnBtContinue`  | `sub_740444` | `0x88` (136) @ `0x74045c` | `3` @ `0x74046a` | none |
| `OnBtRetry`     | `sub_7404B8` | `0x88` @ `0x7404d0` | `5` @ `0x7404de` | none |
| `OnBtExit`      | `sub_74052C` | `0x88` @ `0x740544` | `4` @ `0x740552` | none |
| `SendSelection` | `sub_7405A0` | `0x88` @ `0x7405b9` | `1` @ `0x7405c7`; **throw** `Encode1(a2)` @ `0x7405d3` | 1-byte throw |

- Throw source — `OnButtonClicked` (`sub_73E41F`): `sub_7405A0(this, a2 - 2000)`
  @ `0x73e48b`, guarded by `a2 >= 0x7D0 && a2 <= 0x7D2` (`0x73e436`/`0x73e43e`).
  Button `0xBB8`→OnBtStart, `0xBB9`→OnBtContinue, `0xBBA`→OnBtRetry,
  `0xBBB`→OnBtExit (`0x73e446`).
- Mode 2 (timeout) — `CRPSGameDlg::Update` (`sub_73F140`): in the else-branch of
  `if (curTime - m_tLimit <= 0)` (`0x73f246`), i.e. **timer elapsed**:
  `COutPacket(0x88)` @ `0x73f250`, `Encode1(2)` @ `0x73f261`, `SendPacket`
  @ `0x73f270`, then `m_tLimit = 0` @ `0x73f275` + disable 3 R/P/S buttons.
- Opcode confirmation: every sender constructs `COutPacket` with `0x88` (see
  addresses above); `0x88` = the v83 RPS_ACTION serverbound opcode. ✅

### Semantics corroboration (v83 StringPool constants read in `Update`)
The tip strings resolved by symbol confirm the button meanings:
`SP_3676_PRESS_START_TO_START_THE_GAME` (Start),
`SP_3678_..WINNING_STREAK_CHALLENGE_PRESS_CONTINUE..` (Continue = advance streak),
`SP_3681_HERES_500_MESOS_AS_A_CONSOLATION_PRIZE_PRESS_RESTART_TO_TRY_AGAIN..` and
`SP_3682_PRESS_RESTART_TO_TRY_AGAIN_WHICH_WILL_COST_YOU_THE_PARTICIPATION_FEE_OF_1000_MES`
(Restart/Retry = post-loss, costs participation fee).

---

## 2. v84 — `GMS_v84.1_U_DEVM.exe` (port 13345)  → opcode `0x08C` ✅

`Update` is pre-annotated `CRPSGameDlg__Update_send_0x8C`; `OnPacket` is
`CRPSGameDlg__OnPacket_recv_0x13F`. Senders located by the same size/layout
signature as v83 (four `0x74` + one `0x8b`), directly after dispatcher
`sub_761F6F`.

### Senders
| sender | fn addr | `COutPacket(opcode)` | `Encode1(mode)` | body |
|--------|---------|----------------------|-----------------|------|
| `OnBtStart`     | `sub_7620F4` | `0x8C` (140) @ `0x76210c` | `0` @ `0x76211a` | none |
| `OnBtContinue`  | `sub_762168` | `0x8C` @ `0x762180` | `3` @ `0x76218e` | none |
| `OnBtRetry`     | `sub_7621DC` | `0x8C` @ `0x7621f4` | `5` @ `0x762202` | none |
| `OnBtExit`      | `sub_762250` | `0x8C` @ `0x762268` | `4` @ `0x762276` | none |
| `SendSelection` | `sub_7622C4` | `0x8C` @ `0x7622dd` | `1` @ `0x7622eb`; **throw** `Encode1(a2)` @ `0x7622f7` | 1-byte throw |

- Throw source — `OnButtonClicked` (`sub_760143`): `sub_7622C4(this, a2 - 2000)`
  @ `0x7601af`, guard `a2 >= 0x7D0 && a2 <= 0x7D2`; buttons `0xBB8..0xBBB` route
  Start/Continue/Retry/Exit (`0x76016a`).
- Mode 2 (timeout) — `Update` (`0x760e64`): timer-elapsed branch pushes `0x8C`
  and constructs `COutPacket` @ `0x760f74`, `Encode1(2)` @ `0x760f85`, `SendPacket`
  @ `0x760f94`, then clears `[ebx+0A0h]` (m_tLimit) + disables 3 buttons.
- Opcode confirmation: senders construct `COutPacket(0x8C=140)`. ✅

---

## 3. v87 — `GMSv87_4GB.exe` (port 13343)  → opcode `0x090` ✅

Full `CRPSGameDlg` symbols (`OnBtStart/Continue/Retry/Exit`, `SendSelection`,
`Update`, `OnButtonClicked`).

### Senders
| sender | fn addr | `COutPacket(opcode)` | `Encode1(mode)` | body |
|--------|---------|----------------------|-----------------|------|
| `OnBtStart`     | `0x78afa8` | `0x90` @ `0x78afc0` | `0` @ `0x78afce` | none |
| `OnBtContinue`  | `0x78b01c` | `0x90` (144) @ `0x78b034` | `3` @ `0x78b042` | none |
| `OnBtRetry`     | `0x78b090` | `0x90` @ `0x78b0a8` | `5` @ `0x78b0b6` | none |
| `OnBtExit`      | `0x78b104` | `0x90` @ `0x78b11c` | `4` @ `0x78b12a` | none |
| `SendSelection` | `0x78b178` | `0x90` @ `0x78b191` | `1` @ `0x78b19f`; **throw** `Encode1(a2)` @ `0x78b1ab` | 1-byte throw |

- Throw source — `OnButtonClicked` (`0x788fd5`): `SendSelection(this, a2 - 2000)`
  @ `0x789041`, guard `a2 >= 0x7D0 && a2 <= 0x7D2`; `0xBB8..0xBBB` →
  Start/Continue/Retry/Exit (`0x788ffc`).
- Mode 2 (timeout) — `Update` (`0x789cf6`): timer-elapsed else-branch of
  `if (curTime - *(this+176) <= 0)` (`0x789dfc`): `COutPacket(0x90)` @ `0x789e06`,
  `Encode1(2)` @ `0x789e17`, `SendPacket` @ `0x789e26`, `*(this+176)=0` + disable
  buttons.
- Opcode confirmation: senders construct `COutPacket(0x90)`. ✅

---

## 4. v95 — `GMS_v95.0_U_DEVM.exe` (port 13341)  → opcode `0x0A0` ✅

Fully symbolized including named struct fields (`m_nUserSelect`, `m_tLimit`,
`m_pBtRPS`).

### Senders
| sender | fn addr | `COutPacket(opcode)` | `Encode1(mode)` | body |
|--------|---------|----------------------|-----------------|------|
| `OnBtStart`     | `0x6d6860` | `0xA0` (160) @ `0x6d688f` | `0` @ `0x6d68a2` | none |
| `OnBtContinue`  | `0x6d6900` | `0xA0` @ `0x6d692f` | `3` @ `0x6d6942` | none |
| `OnBtRetry`     | `0x6d69a0` | `0xA0` @ `0x6d69cf` | `5` @ `0x6d69e2` | none |
| `OnBtExit`      | `0x6d6a40` | `0xA0` @ `0x6d6a6f` | `4` @ `0x6d6a82` | none |
| `SendSelection` | `0x6d6ae0` | `0xA0` @ `0x6d6b10` | `1` @ `0x6d6b23`; **throw** `Encode1(nRPS)` @ `0x6d6b31` | 1-byte throw |

- Throw source — `OnButtonClicked` (`0x6d6f40`): `SendSelection(this, nId - 2000)`
  @ `0x6d6f72`, guard `nId >= 0x7D0 && nId <= 0x7D2` (`0x6d6f69`); `3000`→OnBtStart,
  `0xBB9/BBA/BBB`→Continue/Retry/Exit. `SendSelection` stores
  `this->m_nUserSelect = nRPS` @ `0x6d6b46` — confirming the encoded byte is the
  raw user selection.
- Mode 2 (timeout) — `Update` (`0x6d8e80`): else-branch of
  `if (curTime - m_tLimit <= 0)` (`0x6d8fb6`): `COutPacket(160)` @ `0x6d8fc0`,
  `Encode1(2)` @ `0x6d8fd1`, `SendPacket` @ `0x6d8fe0`, `m_tLimit = 0` @ `0x6d8fe5`
  + disable `m_pBtRPS`.
- Opcode confirmation: senders construct `COutPacket(160 = 0xA0)`. ✅

---

## 5. jms185 — `MapleStory_dump_SCY.exe` (port 13344)  → opcode `0x08B` ✅

`CRPSGameDlg` senders + `OnButtonClicked` symbolized; `Update` = unnamed
`sub_7AD52B` (same `0xd69` size as v87's Update), located by size and its unique
`COutPacket(0x8B)+Encode1(2)` timeout emit.

### Senders
| sender | fn addr | `COutPacket(opcode)` | `Encode1(mode)` | body |
|--------|---------|----------------------|-----------------|------|
| `OnBtStart`     | `0x7ae7bb` | `0x8B` @ `0x7ae7d3` | `0` @ `0x7ae7e1` | none |
| `OnBtContinue`  | `0x7ae82f` | `0x8B` (139) @ `0x7ae847` | `3` @ `0x7ae855` | none |
| `OnBtRetry`     | `0x7ae8a3` | `0x8B` @ `0x7ae8bb` | `5` @ `0x7ae8c9` | none |
| `OnBtExit`      | `0x7ae917` | `0x8B` @ `0x7ae92f` | `4` @ `0x7ae93d` | none |
| `SendSelection` | `0x7ae98b` | `0x8B` @ `0x7ae9a4` | `1` @ `0x7ae9b2`; **throw** `Encode1(nRPS)` @ `0x7ae9be` | 1-byte throw |

- Throw source — `OnButtonClicked` (`0x7ac80a`): `SendSelection(this, nId - 2000)`
  @ `0x7ac876`, guard `nId >= 0x7D0 && nId <= 0x7D2`; `0xBB8..0xBBB` →
  Start/Continue/Retry/Exit (`0x7ac831`).
- Mode 2 (timeout) — `Update` (`sub_7AD52B`): else-branch of
  `if (curTime - *(this+176) <= 0)` (`0x7ad631`): `COutPacket(0x8B)` @ `0x7ad63b`,
  `Encode1(2)` @ `0x7ad64c`, `SendPacket` @ `0x7ad65b`, `*(this+176)=0` + disable
  buttons.
- Opcode confirmation: senders construct `COutPacket(0x8B = 139)`. ✅

---

## 6. Per-version opcode table (feeds Task-20 `operations`)

**Opcode (shifts per version):**

| version | RPS_ACTION serverbound opcode | source |
|---------|-------------------------------|--------|
| v83     | `0x088` | `COutPacket(0x88)` in all 6 senders |
| v84     | `0x08C` | `COutPacket(0x8C)` in all 6 senders |
| v87     | `0x090` | `COutPacket(0x90)` in all 6 senders |
| v95     | `0x0A0` | `COutPacket(0xA0)` in all 6 senders |
| jms185  | `0x08B` | `COutPacket(0x8B)` in all 6 senders |

All match the design/STATUS.md Global-Constraints table. ✅

**Sub-op mode bytes (IDENTICAL across all five versions — no per-version shift):**

| sub-op name | byte | body (after sub-op byte) |
|-------------|------|--------------------------|
| `START`     | 0 | — |
| `SELECT`    | 1 | `Encode1` throw (`0`/`1`/`2`) |
| `TIMEOUT`   | 2 | — |
| `CONTINUE`  | 3 | — |
| `EXIT`      | 4 | — |
| `RETRY`     | 5 | — |

(Names are behavior-derived descriptors; the **byte values** are the contract.
atlas-rps / the reader may assign its own symbolic constants.)

---

## Design §16 resolutions (with decompile evidence)

### §16 Item 2 — `OnBtRetry` vs tie-redraw; is there a separate tie sub-op?

**Answer: `OnBtRetry` (sub-op 5) is the post-LOSS "restart / try again" action,
distinct from `OnBtContinue` (sub-op 3) the post-WIN "advance the streak" action.
There is NO tie sub-op — a tie is derived entirely client-side and re-uses the
existing selection round with no serverbound message.**

Evidence:
- `OnBtRetry` = `Encode1(5)`, `OnBtContinue` = `Encode1(3)` — distinct sub-op
  bytes on distinct buttons (`0xBBA` vs `0xBB9`) in all five versions (§1–§5).
- On a **tie**, `CRPSGameDlg::Update` re-enables the R/P/S buttons **locally with
  no packet**: the branch guarded by "player selection == NPC throw" resets
  `m_nNpcSelect = -1`, re-arms the selection timer, and re-enables the 3 buttons.
  Addresses: v83 `0x73f569`, v87 `0x78a11f`, v95 `0x6d9356`
  (`m_nUserSelect == m_nNpcSelect`), jms185 `0x7ad954`. The client simply
  re-selects (serverbound sub-op 1 again). No sub-op is reserved for a tie.
- v83 tip strings confirm the semantics: Restart (`OnBtRetry`) is
  `SP_3681/3682_..PRESS_RESTART_TO_TRY_AGAIN_WHICH_WILL_COST..PARTICIPATION_FEE..`
  (post-loss), Continue (`OnBtContinue`) is
  `SP_3678_..WINNING_STREAK_CHALLENGE_PRESS_CONTINUE..` (post-win).

**Implication for atlas-rps:** model a tie as an outcome computed from the RESULT
payload (clientbound mode 11) that leaves the streak rung unchanged and returns
status to awaiting-select; do NOT expect or emit a tie sub-op. Consistent with the
clientbound "tie-is-client-derived" finding.

### §16 Item 3 — `Update` server relevance: does it send anything besides sub-op 2?

**Answer: `CRPSGameDlg::Update` sends exactly ONE thing to the server — sub-op 2,
and only on selection-countdown expiry. It is otherwise a client-only UI tick
(a `CWnd::Update` override that drives avatar animation, the R/P/S cycle
animation, minigame sounds, tip/countdown rendering — all local). It is NOT a poll
or keepalive.**

Evidence: in every version the `Encode1(2)` emit sits in the single else-branch of
`if (curTime - m_tLimit <= 0)` — reached only when the limit timer has elapsed —
and immediately after the emit the code clears the limit timer (`m_tLimit = 0`) and
disables the 3 R/P/S buttons, i.e. a one-shot timeout notification. Emit addresses:
v83 `0x73f261`, v84 `0x760f85`, v87 `0x789e17`, v95 `0x6d8fd1`, jms185 `0x7ad64c`.
No other `SendPacket` exists in `Update`.

**Implication for atlas-rps:** sub-op 2 is a client-driven "player let the
selection timer run out" signal. The server should treat it as a
timeout/auto-forfeit of the current selection (no throw byte accompanies it), not
as a periodic tick.

### §16 Item 5 — `OnBtExit` vs `OnBtContinue`; does quit-before-collect forfeit?

**Answer: `OnBtExit` (sub-op 4) is the quit/leave action, on the Exit button
(`0xBBB`), distinct from `OnBtContinue` (sub-op 3, "advance the streak"). It sends
only the bare sub-op byte 4 — no payload, no "collect" flag — so it carries no
prize/collect data; quitting via sub-op 4 is a plain leave-the-game and cannot
convey a payout.**

Evidence:
- `OnBtExit` = `Encode1(4)` bodyless in all five versions (§1–§5), routed from
  button `0xBBB` in `OnButtonClicked` (v83 `0x73e446`→`sub_74052C`,
  v87 `0x788ffc`→`OnBtExit`, v95/jms analogous).
- `OnBtContinue` = `Encode1(3)` bodyless (button `0xBB9`) — the post-win
  streak-advance; a separate action/byte.
- Because sub-op 4 has no body, the client conveys nothing beyond "I am leaving."
  Any payout/consolation logic lives server-side. The clientbound prize signal is
  the RESULT frame's `m_bReceiveCompensation` flag (set from mode 11 data, see
  clientbound §4); the serverbound Exit does not acknowledge or collect it.

**Implication for atlas-rps:** on sub-op 4, end the session; if the player quits
before the server has awarded/settled a pending win, treat it as a forfeit — the
client sends no collect request, so there is nothing to pay out on Exit. (The
server owns the payout decision from the RESULT it previously computed.)

---

## Self-check

Every sub-op byte and every field in §0–§6 cites a concrete decompile address
from a decompilation actually run this session, in the correct IDB (binary NAME
confirmed via `list_instances`). Opcodes are confirmed by the `COutPacket(opcode)`
constructor argument inside each sender (not by the STATUS.md table alone). The
throw byte's raw `0`/`1`/`2` mapping is grounded in `OnButtonClicked`
(`nId - 2000`) plus `SendSelection`'s `Encode1(throw)` in every version. No value
here is sourced from memory or general MapleStory knowledge. No version is blocked.
