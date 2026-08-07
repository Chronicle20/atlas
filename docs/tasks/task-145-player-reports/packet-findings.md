# IDA Packet Findings — Sue / Claim Report System

Verified 2026-07-09 via ida-pro-mcp against:

- **v83**: `MapleStory_dump.exe` (`v83_Me` IDB, port 13342)
- **v95**: `GMS_v95.0_U_DEVM.exe` (port 13341)

Extended 2026-08-04, when gms_v48/v61/v72/v79 entered the coverage matrix, against:

- **v48**: `GMS_v48_1_DEVM.exe`
- **v61**: `GMS_v61.1_U_DEVM.exe`
- **v72**: `GMS_v72.1_U_DEVM.exe`
- **v79**: `GMS_v79_1_DEVM.exe`

(Resolve current ports via `idb_list` — they are not stable across sessions.)

These are spec-phase findings to ground the PRD. Implementation must still produce
byte-fixture tests + evidence records per the packet-verifier flow; this file is not
a substitute for that.

## 1. SUE_CHARACTER_RESULT (clientbound; v61/v72/v79 0x034, v83/v84/v87/v95 0x037, v48 0x02C unreachable)

- v83: `CWvsContext::OnSueCharacterResult` @ `0xa29739`
- v95: `CWvsContext::OnSueCharacterResult` @ `0x9fae10`

Body: **1 byte** result code. Semantics identical v83 ↔ v95:

| code | string pool | meaning |
|------|-------------|---------|
| 0 | SP_3003 | "You have successfully reported the user." |
| 1 | SP_3004 | "Unable to locate the user" |
| 2 | SP_3005 | "You may only report users 10 times a day" |
| 3 | SP_3006 | "You have been reported to the GMs by a user" (targets the **accused**) |
| other | SP_3007 | "Your request did not go through for unknown reasons. Please try again later" |

Rendered via `CHATLOG_ADD(msg, 12)` — a chat-log line, **not** a modal dialog.

## 2. CLAIM_REQUEST (serverbound; v72 0x069, v79 0x068, v83 0x06A, v84 0x06A, v87 0x06D, v95 0x076)

- v95: `CWvsContext::SendClaimRequest(ZXString asChatLog, ZXString sCharacterName)` @ `0xa05fb0`.
  Decompile shows `COutPacket::COutPacket(&oPacket, 118)` — 118 = 0x76, matching the matrix.
- v83: send-site **unnamed** in the v83_Me IDB (registry fname `CWvsContext::SendClaimRequest`
  is `csv-import` provenance). `CUIClaim::OnCreate` exists @ `0x7db17d`, and the v83
  `OnClaimResult` mode set matches v95 exactly, so the same body is expected — but naming
  the v83 send-site and byte-verifying it is required implementation work (PRD FR-6.2).
- **v72 @ `0x91f2b4` and v79 @ `0x9711ff`** — `?SendClaimRequest@CWvsContext@@QAEXAAV?$ZArray@V?$ZXString@D@@@@V?$ZXString@D@@@Z`,
  size `0x6bf` in both. Decompiled 2026-08-04: `COutPacket::COutPacket(&p, 105)` on v72
  (packet-build block @ `0x91f749`) and `COutPacket::COutPacket(&p, 104)` on v79, each
  followed by `Encode1(bChatClaim)`, `EncodeStr(targetName)`, `Encode1(reasonType)`,
  `EncodeStr(description)`, then `EncodeStr(chatLog)` **only** under `if (bChatClaim)`,
  then `CClientSocket::SendPacket`. Byte-for-byte the v95 encode order below.
  These two rows are **absent from the registry**, so the matrix renders both cells `⬜`
  (n-a) — a false negative that PRD FR-6.4 / plan Task 23b corrects. Unlike v83, no IDB
  naming work is needed; the functions are already named.

Body (v95-verified encode order):

```
byte    bChatClaim        // 1 = chat/harassment claim, 0 = regular claim
string  sTargetCharacterName
byte    nType             // reason type from CUIClaim::GetResult
string  sContext          // free-text description
if (bChatClaim == 1):
  string chatLog          // CClaimChatLog::GetChatLogOfTwoCharacters(local, target)
```

Key facts:

- The **chat log is client-supplied** (the client's own ring buffer of the two characters'
  lines). Server-side capture is corroboration, not a wire requirement.
- Client-side pre-gating before any packet is sent:
  - `m_bClaimSvrConnected` must be true (set by CLAIM_STATUS_CHANGED), else notice 0xD5A.
  - Current time must be inside `[m_nClaimSvrOpenTime, m_nClaimSvrCloseTime)`, **except
    `open == 0 && close == 0`, which is treated as always-available** (explicit
    `!open && !close` branch proceeds to the claim dialog).
  - Chat claims additionally require `CClaimChatLog::IsClaimAbleCharacter(target)` —
    i.e., the target must appear in the local chat log.

## 3. CLAIM_RESULT (clientbound; v72/v79 0x02A, v83/v84/v87 0x02D, v95 0x02C, jms 0x02A)

- v83: `CWvsContext::OnClaimResult` @ `0xa27891`
- v95: `CWvsContext::OnClaimResult` @ `0x9fa7d0`

Body: **1 mode byte**, then mode-dependent payload. Mode sets identical v83 ↔ v95:

| mode | payload | v83 string pool | meaning |
|------|---------|-----------------|---------|
| 2 | byte `hasRemaining`, int32 `remaining` | SP_3380–SP_3383 | success; "D reports left this week" / "no reports left this week" / undercover variant |
| 3 | — | SP_3384 | "You have been reported by a user" (targets the **accused**) |
| 0x41 | — | SP_3386 | "Please try again later" |
| 0x42 | — | SP_3387 | "Please re-check the character name then try again" |
| 0x43 | — | SP_3388 | "You do not have enough mesos to report" |
| 0x44 | — | SP_3389 | "Unable to connect to the server" |
| 0x45 | — | SP_3390 | "You have exceeded the number of reports available" |
| 0x47 | — | SP_3395 | "You may only report from %d to %d" (formats open/close hours) |
| 0x48 | — | SP_3397 | "Unable to report due to previously being cited for a false report" |

All modes render via `CUtilDlg::Notice` (modal), unlike the sue result.
Unknown modes are silently ignored (`default: return`).

## 4. CLAIM_AVAILABLE_TIME (clientbound; v72/v79 0x02B, v83/v84/v87 0x02E, v95 0x02D, jms 0x02B)

- v83 @ `0xa27b38`, v95 @ `0x9f1620` — identical:

```
byte openHour   -> m_nClaimSvrOpenTime
byte closeHour  -> m_nClaimSvrCloseTime
```

## 5. CLAIM_STATUS_CHANGED (clientbound; v72/v79 0x02C, v83/v84/v87 0x02F, v95 0x02E, jms 0x02C)

- v83 @ `0xa27b61`, v95 @ `0x9f1650` — identical:

```
byte connected  -> m_bClaimSvrConnected (nonzero = true)
```

Without `connected = 1` the client refuses to open the claim dialog, so Atlas must send
this (plus an availability window, or 0/0 for always-open) for the feature to exist
client-side.

## 6. Already-implemented context

- Serverbound `SUE_CHARACTER` (`CField::SendChatMsgSlash#SueCharacter`) is T1-verified for
  seven GMS versions in `libs/atlas-packet/field/serverbound/sue_character.go`:
  v61/v72/v79/v83/v84/v87 lead with accused character id (int32); v95 leads with a
  sub-command string; then `byte flag`, `string reason`. The legacy body is unchanged
  across v61–v87 (v79 send-site @ `0x51825e` re-confirmed 2026-08-04), so the existing
  codec covers the newer columns without a version branch. jms and **v48** are
  version-absent (no send-site) — see §7 for the v48 evidence.
- The atlas-channel handler for it is a decode-and-log stub
  (`services/atlas-channel/atlas.com/channel/socket/handler/sue_character.go`).

## 7. Verified absences (v48 sue; v48/v61 claim)

These are the `⬜` cells this task's scoping *depends on*, recorded with their evidence so
a future reader neither re-litigates them nor "corrects" them the way §2 corrects v72/v79.
They are **discovery-pass negatives** — a nonexistent packet cannot carry a byte-fixture.

### 7.1 `SUE_CHARACTER` is genuinely absent on v48

`CField::SendChatMsgSlash` **does exist** on v48 @ `0x4c3e96` (mangled
`?SendChatMsgSlash@CField@@QAEXABV?$ZXString@D@@@Z`) — note that
`docs/packets/ida-exports/gms_v48.json` records the `#SueCharacter` entry as
`"unresolved": true, "address": "", "function not found in IDB"`, which is a
name-resolution miss in the harvest, **not** evidence about the function.

The function was decompiled in full. It contains exactly two `COutPacket` construction
sites:

| site | opcode | body |
|---|---|---|
| `0x4c4c34` | 152 | `Encode1` (single byte) |
| `0x4c4e52` | 40 | `EncodeStr` (single string — the admin/GM chat branch) |

Neither is the sue shape (`Encode4(characterId)` + `Encode1(flag)` + `EncodeStr(reason)`).
The v48 slash dispatcher has no sue branch at all.

Corollary worth noting: `CWvsContext::OnSueCharacterResult` **does** exist on v48
(@ `0x7207db`, opcode `0x2C`). The v48 client can therefore receive a sue result it has
no way to trigger — which is why v48 gets no template entries rather than
writer-only ones.

### 7.2 `CLAIM_REQUEST` is genuinely absent on v48 and v61

Three independent checks, all negative on both versions:

1. **No claim UI class and no named send-site.** `func_query name_regex "(?i)claim"` returns
   only the clientbound trio plus `CMemoryGameDlg::SendClaimGiveUp` / `COmokDlg::SendClaimGiveUp`
   — the mini-game "give up" senders, an unrelated sense of the word. No `CUIClaim`
   (which v83 has @ `0x7db17d`), no `CUIStatusBar::SendClaim` (v83 @ `0x8d4a22`),
   no `SendClaimRequest` (v72/v79, §2).
2. **No candidate opcode in the region.** The claim-request opcode sits 8 below the sue
   opcode on both v72 (113→105) and v79 (112→104). v61's sue is 104, so a claim request
   would be ~96; `search_text "push    60h"` over `0x800000-0x8a0000` (the v61 CWvsContext
   region) returns **zero** hits.
3. **The layout slot is occupied by something else.** On v72/v79 `SendClaimRequest`
   (`0x6bf` bytes) sits immediately before `OnClaimResult`. In the same slot v48 has
   `sub_71F29B` (`0x2d` bytes, before `OnClaimResult` @ `0x71f2c8`) and v61 has
   `sub_848732` (`0x2d` bytes, before `0x84875f`) — far too small to be the send-site.

**Conclusion:** the claim mechanism enters the GMS client somewhere between v61 and v72.
v48 and v61 ship the clientbound receivers ahead of the submit path, exactly as v48 does
for sue in §7.1.

### 7.2a Task 23c reconfirmation pass — and where this evidence stands relative to §7.3/§7.4

Sessions resolved via `idb_list` matching binary NAME (task-138 discipline, never
`select_instance`/`-ida-port`): v48 = session `93cc947e` (`GMS_v48_1_DEVM.exe.i64`), v61 =
session `415bf585` (`GMS_v61.1_U_DEVM.exe.i64`).

**Re-ran the checks recorded in §7.1 and §7.2 against the current IDBs; all still hold.**

1. `decompile` on `CField::SendChatMsgSlash` @ `0x4c3e96` (v48) reproduces exactly the same
   two `COutPacket` construction sites recorded in §7.1: `0x4c4c34` (opcode 152, `Encode1`
   only) and `0x4c4e52` (opcode 40, `EncodeStr` only). Read the full pseudocode line by
   line — no third `COutPacket::COutPacket` call exists in the function, and neither site
   matches the sue shape.
2. `func_query name_regex "(?i)claim"` on v48 (`93cc947e`) and v61 (`415bf585`) returns the
   identical five hits recorded in §7.2 check 1 on both versions
   (`SendClaimGiveUp@CMemoryGameDlg`, `SendClaimGiveUp@COmokDlg`, `OnClaimResult`,
   `OnSetClaimSvrAvailableTime`, `OnClaimSvrStatusChanged`) — still no `CUIClaim`, no
   `SendClaimRequest`.
3. `search_text "push    60h"` over v61 `0x800000`–`0x8a0000` (§7.2 check 2): **0 hits**
   (`"n":0,"cursor":{"done":true}`), reproducing the recorded negative.
4. Neighbour-slot check (§7.2 check 3): `sub_71F29B` (v48, still `0x2d` bytes) ends at
   exactly `0x71f2c8`, `OnClaimResult`'s address; `sub_848732` (v61, still `0x2d` bytes)
   ends at exactly `0x84875f`, v61's `OnClaimResult` address. Both candidate slots are
   unchanged — no new function has appeared between them.

**Judgment on the evidentiary standard.** These are exactly the checks the task brief
scoped as "cheap," and they are cheap — three of the four are single-call lookups keyed on
a name or a specific instruction pattern. This is meaningfully weaker than the bar §7.3/§7.4
established for the jms_185 absences, which additionally (a) exhaustively enumerated
**every** call site to the `COutPacket` long-opcode constructor across the whole binary —
not just a named function or a guessed opcode region — and (b) individually decompiled
**every** unnamed function among the `CClientSocket::SendPacket` callers, closing the
"unnamed ≠ absent" gap directly rather than relying on `func_query` name misses.

Neither §7.1 nor most of §7.2 does that. §7.1 rests on a full decompile of the *one*
function where sue lives on every other GMS version — strong for that function, but it does
not by itself rule out a sue send-site living somewhere else in the v48 binary under a
different name. §7.2 checks 1 and 2 are name/pattern-keyed, not exhaustive over call sites;
only check 3 (neighbour-slot occupancy) is structural, and it depends on the two candidate
slots being the only place a `SendClaimRequest`-sized function could hide.

**What a full §7.3/§7.4-grade closure would require, and the scale involved.** To bound
this rather than leave it unquantified, this pass additionally pulled the
`COutPacket`-constructor and `CClientSocket::SendPacket` call-site inventories for both
versions via `xrefs_to` (limit 1000, scoped to the resolved constructor/`SendPacket`
addresses, all pages returned `"more":false`):

| version | `COutPacket::COutPacket(long)` @ | call sites | `CClientSocket::SendPacket` @ | call sites |
|---|---|---|---|---|
| v48 | `0x57b77e` | 317 | `0x464cd1` | 317 |
| v61 | `0x5ffc4f` | 391 | `0x474125` | 315 |

Every named caller in both lists is unambiguously unrelated to claim/sue (cash shop, login,
trunk, guild BBS, party/guild message senders, minigame dialogs, skill/attack requests,
etc.) — none is a `CUIClaim`, `SendClaimRequest`, or slash-command-adjacent name. The
residual risk sits entirely in the **unnamed** (`sub_XXXXXX`) callers, which were not
individually decompiled in this pass: roughly 45–50 per version by rough count of the lists
above, a scale comparable to jms_185's 35 (§7.4 Search 4) and 81 (§7.3 Search 3). Closing
that gap to §7.3/§7.4's standard would mean decompiling all of them, per version, for both
the claim and sue questions — deliberately not undertaken here, since the brief scoped this
pass as a cheap re-check of already-recorded evidence, not a new discovery campaign.

This is flagged explicitly rather than silently treated as equivalent: **the v48/v61
absences are reconfirmed at the standard already recorded in §7.1/§7.2, not upgraded to the
jms_185 standard.** If a future pass wants to close this gap, the table above gives the
exact starting point (constructor/`SendPacket` addresses and total call-site counts) so it
does not have to be re-derived.

**Conclusion: no hit.** Every check re-run in this pass came back negative, exactly
reproducing §7.1/§7.2. No claim or sue send-site was found on v48 or v61. The Task 18
template split (`91e9f4124`) stands unchallenged.

### 7.3 `CLAIM_REQUEST` is genuinely absent on jms_185 — but the clientbound trio is live-routed

Session `b6864e54` (`MapleStory_dump_SCY.exe.i64`, resolved via `idb_list`, matched by binary
NAME — the session set rotates and `-ida-port`/`select_instance` are dead, task-138). The
registry's `CLAIM_REQUEST` row (opcode 101 / `0x065`, `provenance: csv-import`) rests on the
CSV alone; every search below came back negative for a send-site matching that shape.

**The known-good shape, re-derived from GMS v92 first.** `disasm` on `CWvsContext::SendClaimRequest`
@ `0x9d9c30` (v92 session `acdfccff`) shows the exact instruction sequence:

```
push 75h                                         ; opcode 117 (0x75)
lea ecx, [esp+88h+var_2C]
call ??0COutPacket@@QAE@J@Z                      ; COutPacket::COutPacket(long)
...
call ?Encode1@COutPacket@@QAEXE@Z                ; bChatClaim
...
call ?EncodeStr@COutPacket@@QAEXV?$ZXString@D@@@Z ; sTargetCharacterName
...
call ?Encode1@COutPacket@@QAEXE@Z                ; nType
...
call ?EncodeStr@COutPacket@@QAEXV?$ZXString@D@@@Z ; sContext
...
call ?EncodeStr@COutPacket@@QAEXV?$ZXString@D@@@Z ; chatLog, guarded by bChatClaim
...
call ?SendPacket@CClientSocket@@QAEXABVCOutPacket@@@Z
```

The structural tell: `bChatClaim` is both the first `Encode1` argument and the guard on the
trailing `EncodeStr`. This — not any specific opcode value — is what the JMS searches below
were written to detect.

**Search 1 — direct name/UI search (negative, expected).** `func_query` for
`SendClaimRequest`, `CUIClaim`, and `SendClaim` on JMS all return nothing (or, for
`SendClaim`, only the unrelated minigame `SendClaimGiveUp` methods on
`CMemoryGameDlg`/`COmokDlg`). `CUIMessenger` and `CUIStatusBar` are both substantially named
in JMS (20+ methods each) but neither has a `SendClaim` member — consistent with v92, where
the send-site is also not a standalone `CUIClaim` class but a member reached from those two
UI classes plus a generic context-menu handler.

**Search 2 — opcode-immediate scan (exhaustive, negative).** Enumerated every `push 101`
(`0x65`) instruction in the entire `.text` segment via `insn_query`, paging the full address
range `0x401000`–`0xbe3034` to completion (`truncated:false`): **35 instructions total in
the whole binary.** Read 20 bytes of raw machine code at each via `get_bytes`; none is
followed by `lea ecx, ...; call ??0COutPacket@@QAE@J@Z` (JMS's long-opcode constructor,
`0x74b68d`) — all 35 are unrelated uses of the literal 101 (vtable calls, arithmetic,
`IWzCanvas::DrawTextA` pixel coordinates, loop constants).

**Search 3 — the entire JMS serverbound opcode space, not just 101 (exhaustive, negative).**
Rather than trust the CSV's opcode, enumerated **every** call site to
`??0COutPacket@@QAE@J@Z` (`0x74b68d`) — i.e. every place JMS constructs an outgoing packet
with an explicit opcode — via `insn_query`, paged to completion across the full `.text`
range: **577 call sites total**, matching `xrefs_to`'s count exactly. `CField::SendChatMsgSlash`
(`0x564ad3`, the slash-command dispatcher, ~29 KB) alone accounts for **81** of them (scoping
`insn_query` to `func: 0x564ad3` and confirming `truncated:false` is the reproducible way to
get this number — a first pass here undercounted at 79 and was corrected on review) — the most
plausible place for a hidden report/claim branch, since it is exactly the kind of function
that embeds many features behind string dispatch. Read the pushed-opcode byte for **all 81**
directly via `get_bytes` (the `lea ecx, [ebp+disp32]` — `8D 8D` + 4-byte displacement, 6
bytes — sits immediately before the `call`, so the preceding push is at a fixed, checkable
offset): the opcodes used are `0x83`, `0x78`, `0x89`, `0xdd`, `0x84`, `0xe1`, `0x29` — seven
distinct values, **none is `0x65`**, and none of the surrounding `Encode1`/`EncodeStr` calls
(read from the full decompile of `SendChatMsgSlash`) matches the claim shape — this dispatcher
reuses the same handful of opcodes for many different slash commands by varying an embedded
chat-type/target parameter, not by minting a new opcode per command.

**Search 4 — every `CClientSocket::SendPacket` call site in the binary (exhaustive, negative,
method validated against v92 first).** `SendClaimRequest` unconditionally ends by calling
`SendPacket` (confirmed above), and `func_query` confirms JMS has exactly one `SendPacket`
overload (`0x4b14f7`, non-virtual). Enumerated every `call` **and** every tail-call `jmp` to
it via `insn_query`, paged to completion: **504 calls, 0 tail-jumps.** Validated the method
first against v92 (`insn_query` scoped to `func: 0x9d9c30` correctly finds the call at
`0x9da3a4`, matching the manual disasm). Of the 504 JMS callers, 35 are unnamed (`sub_XXXXXX`)
functions; **all 35 were individually decompiled** (not filtered by size — a same-branch
review round flagged that v92's own `SendClaimRequest` is well inside a "small function" band,
so size is not a valid triage filter here). None matches the claim shape:

- `sub_7AD52B` — opcode `0x8B`, a minigame countdown-timer `Update()` routine
  (Rock-Paper-Scissors-adjacent), single `Encode1(2)` body, no strings.
- `sub_47F824`/`sub_48182A`/`sub_481F71` — cash-shop / admin-shop item-trade sends (opcodes
  `0xF7`/`0xF8`), unconditional multi-`EncodeStr` bodies, no `bChatClaim`-style gate.
- `sub_485179`, `sub_4B1BB3`, `sub_56BC92` — trivial/near-trivial sends (ping-like,
  security-challenge response, and a shared "send this already-built packet" tail helper
  called only from `SendChatMsgSlash`, already covered by Search 3).
- `sub_56E0B9` (opcode `0x81`, 2×`Encode4`, no string), `sub_575186` (opcode `0x10F`, 4×
  `Encode1` + `Encode4` + 1–2 `EncodeStr` gated on a raw parameter, not the encoded byte) —
  see note below.
- `sub_739F96`/`sub_73B409`/`sub_73C802`/`sub_740124`/`sub_74092C`/`sub_741292`/
  `sub_743515`/`sub_7435C8`/`sub_743641`/`sub_7437B9` — all opcode `0xEE`/`0xEF`, a
  mode-prefixed minigame-carnival family (`Encode1(mode)` 0–11), no `EncodeStr` at all.
- `sub_74387A` — opcode `0x29`, single unconditional `EncodeStr`, wrong field order
  (`Encode4`, `EncodeStr`, `Encode1`) and no second string.
- `sub_74763A`, `sub_7CAB93`, `sub_8B5323`, `sub_8B5F75`, `sub_8B7CF1`, `sub_8FA27F`,
  `sub_94A492`, `sub_9923A4`, `sub_9C2F45`, `sub_A3ED44`, `sub_AEDCA9`, `sub_AEDD3A`,
  `sub_AF8B98`, `sub_AF8F90`, `sub_B0BBA8` — all pure `Encode1`/`Encode2`/`Encode4` numeric
  bodies (item-upgrade confirmations, skill/combat requests, friend/wishlist actions), no
  `EncodeStr` calls, several reusing opcode bytes that are claim-adjacent in other versions
  purely by coincidence (e.g. `sub_B0BBA8` uses `0x75` — the same numeral as v92's
  `CLAIM_REQUEST` opcode — for what is structurally a periodic heartbeat/ping, 1×`Encode1(0)`,
  no strings; opcode numbering is clearly not preserved 1:1 GMS→JMS).

Note on `sub_575186` (`0x575186`, opcode `0x10F` = 271): this is the closest near-miss to the
claim shape found in this sweep — `Encode1`×4, `Encode4`(character id), `EncodeStr`(reason),
then conditionally `EncodeStr`(a second string) gated on parameter `a5`. Structurally this
looks like a **sue/report** send-site (accused id + reason string + optional detail), not
claim (which is 2×`Encode1` + 2–3×`EncodeStr`, no numeric id field, gate on the *encoded*
byte rather than a raw parameter). `xrefs_to` on `sub_575186` shows all **7** of its callers
originate inside `CField::SendChatMsgSlash` — i.e. it is reached only from the slash-command
dispatcher, which is exactly where `SUE_CHARACTER` lives on other GMS versions (§1, §6) —
materially strengthening the sue/report read beyond body shape alone. This is flagged for
whoever resolves jms `SUE_CHARACTER` (a separate op from this section) — it was not chased
further here since it is not this section's target and does not match the claim shape either
way.

**Search 5 — is the clientbound trio actually reachable, or dead code?** `xrefs_to` on all
three clientbound handlers (`OnClaimResult` `0xb0e9c3`, `OnSetClaimSvrAvailableTime`
`0xb0ec69`, `OnClaimSvrStatusChanged` `0xb0ec92`) shows exactly one caller each:
`CWvsContext::OnPacket` @ `0xaebfe7`. Decompiling that dispatcher shows a genuine compiled
`switch (nType)` (not dead/unreachable code — it is the same live dispatcher that also
routes dozens of unambiguously-used ops: inventory, stat changes, guild, family, party,
etc.) with:

```
case 0x2A: CWvsContext::OnClaimResult(this, iPacket); break;
case 0x2B: CWvsContext::OnSetClaimSvrAvailableTime(this, iPacket); break;
case 0x2C: CWvsContext::OnClaimSvrStatusChanged(this, iPacket); break;
```

`0x2A`/`0x2B`/`0x2C` = 42/43/44, exactly matching §3–§5's recorded opcodes. **This routing is
live, not dead code carried over from a shared source tree.**

**Conclusion.** JMS v185 has no `CLAIM_REQUEST` send-site anywhere in the binary — not under
any name, not under any opcode, not as a `SendPacket` caller and not as a raw
`COutPacket`-then-something-else builder. This was checked five independent ways (name/UI,
the CSV's specific opcode, the *entire* opcode space via every `COutPacket` construction site,
every `SendPacket` call site with the method validated against the known-good v92 case, and
the clientbound dispatcher's reachability) and is genuinely exhaustive **over executable code
paths that construct or send an outgoing packet** — every way the client could build and
transmit a `CLAIM_REQUEST` was enumerated and read, not merely spot-checked. It is *not*
exhaustive over every conceivable form of evidence: a JMS-localized string-table sweep for
report/claim UI text was never run (see the registry disposition below), so a reader should
not take "exhaustive" to mean "no residual doubt whatsoever," only "no gap in the code-path
coverage that would explain a missed send-site." At the same time, the three clientbound
claim-result handlers are named, correctly bodied (per §3–§5), and **live-routed** from the
real packet dispatcher. This is the odd case flagged before searching: a client that can
receive and render claim results but has no way to generate the request that would produce
one. The most coherent read is that JMS ships the shared network/receive code for a feature
whose client-side trigger (the report/claim UI) was removed or never wired up for this build
— the same pattern established for v48 sue (§7.1), just on the send side of the *other* wire
pair.

**Registry disposition — corrected by Task 24 (2026-08-05).** The paragraph originally here
asserted the row was "left unchanged" and that "the matrix's jms185 cell for `CLAIM_REQUEST`
stays `⬜`". Both halves of that sentence were wrong on inspection: the row *did* carry
`provenance: csv-import` with a live opcode (101 / `0x065`), and a registry row with an opcode
renders `❌` ("incomplete"), not `⬜` ("n-a") — `STATUS.md` line 647 confirmed the cell was
`0x065 | ❌` both before and after this section was written. The author appears to have
assumed a `csv-import` row renders as n-a; it does not. Left as written, the matrix advertised
a live opcode for a serverbound op this section had just proven, five independent ways
(name/UI search, exhaustive opcode-101-immediate scan, exhaustive whole-opcode-space
`COutPacket`-construction-site scan, exhaustive `SendPacket`-caller scan with all 35 unnamed
functions individually decompiled, and the clientbound-dispatcher reachability/sibling
cross-check in Search 5), has **no send-site anywhere in the jms binary** — a false positive
mirroring the false negative Task 23b fixed on gms v72/v79's `CLAIM_REQUEST` rows the other
direction.

Task 24 (packet verification campaign) revisited this call. The five searches above meet
`VERIFYING_A_PACKET.md`'s "Is this cell n-a?" evidentiary bar for a positive absence claim
(anchored on the opcode-construction-site invariant, not on IDB names or an address range;
the mandatory sibling cross-check was performed in Search 5 and, unusually, still came back
negative even though the clientbound trio is live-routed — the flagged "receives but can
never send" case). Task 24 therefore **removed** the `CLAIM_REQUEST` row from
`docs/packets/registry/jms_v185.yaml` so the matrix cell genuinely renders `⬜`, matching the
treatment already given to gms v48/v61 sue (§7.2a) rather than leaving a documented absence
contradicted by a live matrix cell. Task 24 attempted to promote jms `CLAIM_RESULT`,
`CLAIM_AVAILABLE_TIME`, and `CLAIM_STATUS_CHANGED` (same family) to verified on jms_v185 in
the same pass, but **that verification did not happen**: the jms IDA session (`b6864e54`) was
wedged for the entire campaign — `idb_list` reported `is_active:true` with a recent
`last_accessed`, but a direct `lookup_funcs` call against it timed out while other sessions
responded normally in the same window. The outcome is **0 of 3 promoted**, blocked on that
infrastructure outage, not a scope or design decision; the starting addresses are recorded
(handlers `0xb0e9c3`/`0xb0ec69`/`0xb0ec92`, dispatcher `0xaebfe7` cases `0x2A`/`0x2B`/`0x2C`)
for a retry pass once the instance is healthy. Because those three cells remain unverified,
this `CLAIM_REQUEST` n-a is a family-inconsistent n-a regardless of that outcome; Task 24
declared the `claim` family in `docs/packets/feature-families.yaml` and recorded the positive
absence proof (this section's five searches) in `docs/packets/feature-na-evidence.yaml` so
`matrix --check`'s n-a consistency gate has the citation it requires rather than silently
exempting the cell.

### 7.4 `SUE_CHARACTER` / `SUE_CHARACTER_RESULT` are genuinely absent on jms_185 — and `sub_575186` is not sue

Session `b6864e54` (`MapleStory_dump_SCY.exe.i64`). §7.3 flagged `sub_575186` @ `0x575186`
(opcode `0x10F` / 271, all 7 callers inside `CField::SendChatMsgSlash` @ `0x564ad3`) as a
sue-shaped near-miss — accused-id-plus-reason-plus-optional-string, reached from exactly the
right dispatcher. Two reviewers agreed on the characterization but did not chase it. This
section chases it, using a GMS sue reference shape **derived fresh from IDA** (per the task-30
brief's instruction not to trust any quoted opcode/field values) rather than assumed from the
Go codec's comments.

**Step 1 — derive the GMS reference shape from three independent versions, not one.**
`libs/atlas-packet/field/serverbound/sue_character.go` documents a version boundary between
v87 (legacy `Encode4(charId)` lead) and v95 (`EncodeStr(subCommand)` lead) but doesn't fully
pin down v92, which sits between them. All three were decompiled/disassembled directly:

| version | session | address | opcode (verified via `push` immediate) | shape |
|---|---|---|---|---|
| v87 | `d51ecbd3` | `0x553526` | `push 75h` → 117 (`0x75`) | `Encode4([edi+1444h])` → `Encode1(esi)` → `EncodeStr(reason)` |
| v92 | `acdfccff` | `0x53b7d0` | `push 7Dh` → 125 (`0x7D`) | `EncodeStr(target)` → `Encode1(edi, range 0–5)` → `EncodeStr(reason)` |
| v95 | `79906a1e` | `0x5413e5` | `push 7Eh` → 126 (`0x7E`) | `EncodeStr(sSubCmd)` → `Encode1(edi)` → `EncodeStr(reason)` |

The opcode climbs monotonically (117→125→126) and the v92 site's leading-string field sits at
the identical stack offset (`ebp+68h+0x9c`) as v95's confirmed `sSubCmd`, with the same `0x23`
(`'#'`) destructor-tracking marker immediately preceding the `ZXString` construction — strong
independent confirmation that v92 already uses the string-leading form, one version earlier
than the Go codec's comment (which only commits to "the boundary is between 87 and 95")
pins it. `libs/atlas-packet/registry/gms_v83.yaml`/`v87.yaml`/`v95.yaml`'s serverbound
`SUE_CHARACTER` rows (opcodes 114/117/126) corroborate the same monotonic climb across the
CSV-sourced versions. **The invariant across all three real, verified sue send-sites: exactly
one `Encode1` (a flag/category byte, range 0–5 in the v92/v95 form), exactly one
leading id-or-name field, exactly one trailing reason string — three fields total, always
unconditional. No version ever encodes a fourth field or a conditionally-included second
string.** v92's site was annotated `task-30: SueCharacter (v92 GMS reference)...` and the IDB
saved.

**Step 2 — compare `sub_575186` against that shape, field by field.** Full decompile of
`0x575186` (JMS):

```
sub_575186(a1: byte, n: byte, a3: byte, a4: int, a5: int, s: ZXString*, nType: const ZXString*)
  guard: skip send unless (nType non-empty) or (!a5)
  v8 = *(TSingleton<CWvsContext>::ms_pInstance + 0x2080)   /*0x5751b3*/
  COutPacket(0x10F)                                        /*0x5751c1*/
  nTypea = (a5 ? 2 : 0) | (!a4 ? 1 : 0)
  Encode1(nTypea)   Encode1(n)   Encode1(a3)   Encode1(a1)  /* four Encode1 calls */
  Encode4(v8)                                                /* SELF character id */
  EncodeStr(s)
  if (a5) EncodeStr(nType)                                   /* conditional 7th field */
  SendPacket(...)
```

Two things settle it, independent of each other:

1. **`v8` is the sender's own character id, not an accused player's.** It is read directly
   from `TSingleton<CWvsContext>::ms_pInstance` — the singleton representing the *local*
   player's context — at offset `0x2080`, confirmed via disasm (`mov eax, ms_pInstance...;
   mov esi, [eax+2080h]`). Sue's entire purpose is identifying an *accused* character; every
   confirmed GMS sue send-site encodes either the target's id (legacy) or a target/subcommand
   string (v92/v95) — never the sender's own id, since the server already knows the sender
   from the socket session. A "report a player" packet that embeds only the reporter's own id
   and never the target's is not sue.
2. **The field count and conditionality don't match.** Sue is invariantly 3 fields, always
   sent. `sub_575186` is 6–7 fields (four `Encode1`s, an id, a string, and a conditionally-gated
   second string) — a strictly larger, differently-shaped packet. The trailing-conditional-string
   pattern is structurally closer to *claim*'s `bChatClaim`-gated trailing string (§7.3), except
   claim has no `Encode4` field at all and only two `Encode1`s, so it doesn't match either.

Checked all 7 call sites (`0x567836`, `0x567979`, `0x567b35`, `0x567d1a`, `0x567ea0`,
`0x568019`, `0x5681db`) for whether any passes a *different* character's id as an argument
(which would contradict the self-id reading above): none does — every site either passes
literal `0`/`1` constants or locally-derived flags for `a1`/`n`/`a3`/`a4`/`a5`, and `s`/`nType`
are populated from parsed command-argument strings or `StringPool` lookups (e.g. index `0x109`
feeding a `CUtilDlg::Notice` confirmation dialog at the `0x567979` site), never from a
target-character lookup. `sub_575186` was annotated `task-30: NOT SueCharacter...` in the IDB
(not renamed — its actual purpose, most likely a JMS-specific self-report/petition/feedback
slash command, was not identified and should not be guessed) and the IDB saved.

**Conclusion: `sub_575186` is not `SUE_CHARACTER`.** It is a different, currently-unidentified
JMS slash-command feature that happens to share sue's superficial "id + reason string" body
shape while diverging on every field that matters (whose id, how many fields, whether the
trailing string is ever conditional).

**Step 3 — the remaining 81 direct `SendChatMsgSlash` send-sites don't match sue's shape
either.** §7.3's Search 3 already enumerated all 81 `COutPacket` constructions made directly
inside `SendChatMsgSlash` (not via a callee) and found 7 distinct opcodes: `0x83`, `0x78`,
`0x89`, `0xdd`, `0x84`, `0xe1`, `0x29`. One representative site per opcode was decompiled here:

| opcode | representative site | shape |
|---|---|---|
| `0x83` (131) | `0x567681` | `Encode1(0x26)` → `EncodeStr(s)` — 2 fields, no id at all |
| `0x78` (120) | `0x5685b0` | `Encode1(esi)` → `Encode1(4)` → `EncodeStr(s)` — 2 `Encode1`s, matches claim's family better than sue's single-`Encode1` shape |
| `0x89` (137) | `0x5687f8` | `Encode1(esi)` only, then falls through to a *different* opcode-`0x78` construction — an early-exit variant, not a self-contained sue-shaped packet |
| `0xdd` (221) | `0x56968c` | shares the `sub_56BC92` trailer helper with several other branches; no standalone id+flag+reason shape |
| `0x84` (132) | `0x56a838` | `EncodeStr` only (via `sub_42A9E9`) then `sub_56BC92` — no `Encode1` at all |
| `0xe1` (225) | `0x56b971` | single `Encode1` derived from a boolean, then `sub_56BC92` — no id field |
| `0x29` (41) | `0x56bbef` | `Encode4(get_update_time())` → `EncodeStr(s)` — the `Encode4` here is a **timestamp**, not a character id |

None reproduces the invariant 3-field `id/name + flag + reason` shape. Combined with §7.3's
Search 4 (every `SendPacket` call site in the binary, 504 total, all 35 unnamed ones
individually decompiled), every code path capable of constructing and sending an outgoing
packet from the JMS binary has now been examined for the sue shape specifically, not just the
claim shape.

**Step 4 — the clientbound dispatcher has no slot for `SUE_CHARACTER_RESULT` either.**
`CWvsContext::OnPacket` @ `0xaebfe7` was decompiled in full: a genuine compiled
`switch (nType)` spanning cases `0x1B`–`0x7A` (~74 explicit cases; everything else, including
gaps, falls to `default: return`). The case list runs `... case 0x28: OnAntiMacroResult; case
0x2A: OnClaimResult; ...` — **case `0x37` does not exist**, jumping directly from
`case 0x36: OnPartyResult` to `case 0x38: OnExpedtionResult`. The conclusion rests on the
fully-enumerated switch itself (every case from `0x1B`–`0x7A` was read, not sampled) plus the
zero-hit name search (`func_query name_regex "(?i)suecharacter"` returns nothing anywhere in
the JMS IDB) — that pair is sufficient on its own. Worth noting only as a coincidental footnote,
not corroborating evidence: `0x37` (55) happens to be the same opcode GMS v83/v84/v87/v95 use
for `SUE_CHARACTER_RESULT` (§1), and `docs/packets/registry/jms_v185.yaml` independently has a
matching gap at opcode 55 between `PARTY_OPERATION` (54) and `IDA_0X038`/`OnExpedtionResult`
(56). But JMS's opcode numbering is demonstrably unrelated to GMS's in general — the §7.4
near-miss `sub_575186` sits at `0x10F` (271) versus GMS's 114–126 for the same
`SUE_CHARACTER`/`CLAIM_REQUEST` family — and this switch has several other unremarkable gaps
(`0x29`, `0x33`, `0x34`, `0x3A`, `0x4A`) that don't line up with anything, so the `0x37` match is
not evidence of shared numbering and should not be read as such.

**Step 5 — string-table sweep, covering both sue and claim (closes §7.3's stated residual
gap in the same pass).** `find_regex` over the JMS binary's string table:

| pattern | hits | notes |
|---|---|---|
| `sue` | 0 | matches GMS v92's own 0-hit baseline for the same pattern — both clients keep this text in an external `StringPool`/`.nx` resource, not the binary, so a 0-hit result here is uninformative about presence/absence rather than confirmatory either way |
| `claim` | 0 | closes §7.3's explicitly-stated residual gap: no claim/report UI text embedded in the binary either |
| `abuse\|harass\|petition` | 0 | |
| `report` | 1 (`report25.mod`, an unrelated filename literal) | not report/claim UI text |

**Conclusion.** Both `SUE_CHARACTER` (serverbound) and `SUE_CHARACTER_RESULT` (clientbound)
are genuinely absent from jms_185: the clientbound dispatcher has no case at the opcode GMS
uses for the result (corroborated by an independent registry gap at the same opcode), no
function matching the result handler's name exists anywhere in the IDB, no serverbound send
site anywhere in the binary (81 direct `SendChatMsgSlash` sites, 504 total `SendPacket` call
sites, and the specific near-miss `sub_575186`) reproduces the fixed 3-field shape verified
fresh across three independent GMS versions, and the string table has no sue-related text
(uninformative on its own, but consistent with everything else). `sub_575186` — the lead this
task started from — is a different, unidentified JMS-specific slash-command feature; it embeds
the *sender's own* character id rather than an accused target's, and its 6–7-field conditional
shape does not match sue (or claim) in any confirmed GMS version.

**Registry/matrix disposition:** no `SUE_CHARACTER` or `SUE_CHARACTER_RESULT` rows are added to
`docs/packets/registry/jms_v185.yaml`. The JMS185 matrix cells for both ops (`STATUS.md` lines
81 and 640) stay `⬜` and must not be "corrected" to an opcode — this is a verified absence in
the same sense as §7.1's v48 sue finding, not an unresolved gap.
