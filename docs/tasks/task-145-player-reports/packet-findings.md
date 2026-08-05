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
