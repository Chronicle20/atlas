# IDA Packet Findings — Sue / Claim Report System

Verified 2026-07-09 via ida-pro-mcp against:

- **v83**: `MapleStory_dump.exe` (`v83_Me` IDB, port 13342)
- **v95**: `GMS_v95.0_U_DEVM.exe` (port 13341)

These are spec-phase findings to ground the PRD. Implementation must still produce
byte-fixture tests + evidence records per the packet-verifier flow; this file is not
a substitute for that.

## 1. SUE_CHARACTER_RESULT (clientbound, 0x037 in v83/v84/v87/v95)

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

## 2. CLAIM_REQUEST (serverbound; v83 0x06A, v84 0x06A, v87 0x06D, v95 0x076)

- v95: `CWvsContext::SendClaimRequest(ZXString asChatLog, ZXString sCharacterName)` @ `0xa05fb0`.
  Decompile shows `COutPacket::COutPacket(&oPacket, 118)` — 118 = 0x76, matching the matrix.
- v83: send-site **unnamed** in the v83_Me IDB (registry fname `CWvsContext::SendClaimRequest`
  is `csv-import` provenance). `CUIClaim::OnCreate` exists @ `0x7db17d`, and the v83
  `OnClaimResult` mode set matches v95 exactly, so the same body is expected — but naming
  the v83 send-site and byte-verifying it is required implementation work (PRD FR-6.2).

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

## 3. CLAIM_RESULT (clientbound; v83/v84/v87 0x02D, v95 0x02C, jms 0x02A)

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

## 4. CLAIM_AVAILABLE_TIME (clientbound; v83/v84/v87 0x02E, v95 0x02D, jms 0x02B)

- v83 @ `0xa27b38`, v95 @ `0x9f1620` — identical:

```
byte openHour   -> m_nClaimSvrOpenTime
byte closeHour  -> m_nClaimSvrCloseTime
```

## 5. CLAIM_STATUS_CHANGED (clientbound; v83/v84/v87 0x02F, v95 0x02E, jms 0x02C)

- v83 @ `0xa27b61`, v95 @ `0x9f1650` — identical:

```
byte connected  -> m_bClaimSvrConnected (nonzero = true)
```

Without `connected = 1` the client refuses to open the claim dialog, so Atlas must send
this (plus an availability window, or 0/0 for always-open) for the feature to exist
client-side.

## 6. Already-implemented context

- Serverbound `SUE_CHARACTER` (`CField::SendChatMsgSlash#SueCharacter`) is T1-verified for
  all four GMS versions in `libs/atlas-packet/field/serverbound/sue_character.go`:
  v83/v84/v87 lead with accused character id (int32); v95 leads with a sub-command string;
  then `byte flag`, `string reason`. jms is version-absent (no send-site).
- The atlas-channel handler for it is a decode-and-log stub
  (`services/atlas-channel/atlas.com/channel/socket/handler/sue_character.go`).
