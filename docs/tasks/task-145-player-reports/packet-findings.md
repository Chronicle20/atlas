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

**Registry disposition:** `docs/packets/registry/jms_v185.yaml`'s `CLAIM_REQUEST` row is left
unchanged — still `provenance: csv-import`, opcode 101 / `0x065`, **unverified**. It is not
promoted (no verified send-site exists to cite) and not deleted (it is not a proven-absent
op the way §7.1/§7.2 are for GMS v48/v61 — a JMS-specific 6th independent search, e.g. a
string-table sweep for the JMS-localized claim/report UI text, could still in principle
surface something this address/opcode-based sweep would miss). The matrix's jms185 cell for
`CLAIM_REQUEST` stays `⬜` and should not be "corrected" to an opcode on the strength of the
CSV alone.
