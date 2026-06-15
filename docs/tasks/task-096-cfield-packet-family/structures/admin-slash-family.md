# task-096 — Admin/Slash serverbound family (`CField::SendChatMsgSlash`)

Six serverbound ops emitted by the client `/`-command parser
`CField::SendChatMsgSlash`. This is a **dispatcher-family** send-site (like
WHISPER but serverbound): the one function builds several distinct
`COutPacket(opcode)` packets — one opcode per command class — so each maps to a
distinct op via a `#suffix` export entry (`CField::SendChatMsgSlash#AdminChat`,
etc.). The base `CField::SendChatMsgSlash` export entry is left untouched.

## Send-site bases (export `CField::SendChatMsgSlash` address)

| version | port | base address |
|---|---|---|
| gms_v83 | 13342 | 0x52c40c |
| gms_v84 | 13337 | 0x5383ce |
| gms_v87 | 13341 | 0x552c6c |
| gms_v95 (authoritative) | 13340 | 0x5408e0 |
| jms_v185 | 13339 | 0x564ad3 |

## Per-op opcode + layout (read from IDA `COutPacket(opcode)` + `Encode*` order)

`COutPacket::COutPacket(&pkt, <decimal opcode>)` then ordered `Encode4/2/1/Str`
then `SendPacket`. Opcodes confirmed against the registry rows. v84 opcodes were
**READ from the v84 IDB send-sites** (not derived) — they differ from the stale
v83-seeded csv-import registry values.

| Op | #Suffix | Layout (wire) | v83 | v84 | v87 | v95 | jms |
|----|---------|---------------|-----|-----|-----|-----|-----|
| ADMIN_CHAT | AdminChat | byte, byte, string | 0x076 | **0x078** | 0x07C | 0x08B | 0x078 |
| ADMIN_COMMAND | AdminCommand | byte (sub-command; variable tail) | 0x080 | **0x084** | 0x088 | 0x097 | 0x083 |
| ADMIN_LOG | AdminLog | string | 0x081 | **0x085** | 0x089 | 0x098 | 0x084 |
| MATCH_TABLE | MatchTable | byte | 0x0D6 | **0x0DC** | 0x0E3 | 0x102 | 0x0E1 |
| SLIDE_REQUEST | SlideRequest | byte | — | — | — | 0x09E | 0x089 |
| SUE_CHARACTER | SueCharacter | (v83/v84/v87) int32, byte, string · (v95) string, byte, string | 0x072 | 0x072 | 0x075 | 0x07E | — |

Applicability per the task table:
- **SLIDE_REQUEST**: v95 + jms only (absent v83/v84/v87 — no `COutPacket(opcode)` site).
- **SUE_CHARACTER**: v83/v84/v87/v95 (jms-absent — no site).
- ADMIN_CHAT / ADMIN_COMMAND / ADMIN_LOG / MATCH_TABLE: all 5.

### Layout notes

- **ADMIN_CHAT** — uniform `Encode1, Encode1, EncodeStr` across every site and
  every version (12 sites in v83/v84/v87/v95, 22 in jms). Modeled as
  `byte1, byte2, message string`.
- **ADMIN_COMMAND** — true dispatcher: every site leads with `Encode1` (a
  per-`/`-command sub-command byte), then a *variable* per-sub-command payload
  (`Encode1`/`Encode2`/`Encode4`/`EncodeStr` in arbitrary combos across 35+ sites).
  The only stable wire field is the leading sub-command byte; the codec models
  that single byte (decode-and-log). Export `calls = [Encode1]`.
- **ADMIN_LOG** — single site, `EncodeStr` (the log message string).
- **MATCH_TABLE** — single site, `Encode1` (a bool flag).
- **SLIDE_REQUEST** — single site, `Encode1` (a byte; v95 sends 0).
- **SUE_CHARACTER** — single site per version. v83/v84/v87 lead with
  `Encode4` (the accused character id) then `Encode1`, `EncodeStr`. v95 leads
  with `EncodeStr` (a sub-command string) then `Encode1`, `EncodeStr`. Codec
  version-branches on `IsRegion("GMS") && MajorAtLeast(95)`.

## Representative send-site addresses (for the `#suffix` export entry `address`)

| Op | v83 | v84 | v87 | v95 | jms |
|----|-----|-----|-----|-----|-----|
| ADMIN_CHAT | 0x52de5a | 0x539f6a | 0x554e3b | 0x541d57 | 0x5685b0 |
| ADMIN_COMMAND | 0x52c958 | 0x53891a | 0x5531b8 | 0x540fbe | 0x568ac2 |
| ADMIN_LOG | 0x52e297 | 0x53a38e | 0x55524f | 0x54298b | 0x56a838 |
| MATCH_TABLE | 0x52ec6c | 0x53ad6d | 0x555dff | 0x5445eb | 0x56b971 |
| SLIDE_REQUEST | — | — | — | 0x542439 | 0x5687f8 |
| SUE_CHARACTER | 0x52cb7c | 0x538c80 | 0x553526 | 0x5413e5 | — |

## v84 registry opcode corrections (read from the v84 IDB)

The v84 serverbound registry rows for these ops were **stale csv-import** values
seeded from the v83 column (the CSVs have no v84 column; v84's serverbound opcode
table shifted vs v83 — memory `bug_v84_opcode_table_shifted_vs_v83`). The real
v84 opcodes read from the v84 `SendChatMsgSlash` send-sites:

| Op | stale (csv) | real (v84 IDB) | site |
|----|-------------|-----------------|------|
| SUE_CHARACTER | 114 (0x72) | 114 (0x72) — already correct | 0x538c80 |
| ADMIN_CHAT | 118 (0x76) | **120 (0x78)** | 0x539f6a |
| ADMIN_COMMAND | 128 (0x80) | **132 (0x84)** | 0x53891a |
| ADMIN_LOG | 129 (0x81) | **133 (0x85)** | 0x53a38e |
| MATCH_TABLE | 214 (0xD6) | **220 (0xDC)** | 0x53ad6d |

### v84 collision analysis (task-3 gate)

Unlike the WEDDING case in `serverbound-r-sb.md`, the real v84 opcodes here are
**free of template-handler collisions**:

- The v84 seed template has **no serverbound handler** at any of 0x72/0x78/0x84/
  0x85/0xDC (the real opcodes) nor at the stale 0x76/0x80/0x81/0xD6.
- The stale `WHISPER` serverbound registry row sits at opcode 120 (0x78) but the
  actual v84 template handler `CharacterChatWhisperHandle` is wired at **0x7A
  (122)**, and the serverbound WHISPER matrix row is already ❌ (unrouted) — not a
  ✅ that routing ADMIN_CHAT at 0x78 could regress. The stale `UNNAMED_R221`
  serverbound row at 132 (0x84) has an empty fname (a known phantom), no handler,
  and no matrix cell to flip.

Therefore all five applicable v84 cells (ADMIN_CHAT, ADMIN_COMMAND, ADMIN_LOG,
MATCH_TABLE, SUE_CHARACTER) get the real opcode (provenance `ida-discovered`) AND
an explicit v84 template handler route. **No v84 collision deferred.**
