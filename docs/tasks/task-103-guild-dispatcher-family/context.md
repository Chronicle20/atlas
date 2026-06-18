# Guild Dispatcher Family — Implementation Context

Companion to `plan.md`. Captures the grounded current-state facts, key file
locations, decisions, and dependencies an executor needs before starting. Every
fact below is cited to a file:line read during planning; anything not cited is
flagged as execution-time IDA work.

## 1. What this task is

Migrate the **guild** packet dispatcher family (`CWvsContext::OnGuildResult`
clientbound, `GUILD_OPERATION` serverbound, `CUIGuildBBS::OnGuildBBSPacket` BBS
sub-dispatcher, and the `CUserRemote::OnGuild{Name,Mark}Changed` foreign
broadcasts) to the canonical discrete-per-mode pattern in
`docs/packets/DISPATCHER_FAMILY.md`, drive every supported arm to ✅ across
`gms_v83/v84/v87/v95/jms_v185`, remove guild from
`docs/packets/dispatcher-lint-baseline.yaml`, and patch live tenant config.

`matrix ✅` already shows mostly green — but green means **codec byte-correct
only**. The footguns (caller-selectable mode, catch-all error struct, phantom
roots, the AgreementResponse wire bug, v84 carryover) are separate requirements
proven by the §8/§10 gates, not by the matrix.

## 2. The canonical exemplar to copy

- `libs/atlas-packet/field/clientbound/mts_operation.go` — one discrete struct
  per mode in ONE consolidated file.
- `libs/atlas-packet/field/mts_operation_body.go` — one fixed-key body func per
  mode; `WithResolvedCode("operations", FIXED_KEY, func(mode byte) …)`.
- `docs/packets/dispatchers/mts_operation.yaml` — per-version mode table
  (`writer`, `fname`, `op`, `direction`, `operations:` list of
  `{ key, modes: { gms_v83, … } }`).

## 3. Current guild code — grounded findings

### Clientbound (`libs/atlas-packet/guild/clientbound/operation.go`)
Discrete structs ALREADY EXIST for: `RequestAgreement`, `EmblemChange`,
`MemberStatusUpdate`, `MemberTitleUpdate`, `NoticeChange`, `MemberLeft`,
`MemberExpel`, `MemberJoined`, `Invite`, `TitleChange`, `Disband`,
`CapacityChange`. (`Info` lives in `info.go`; BBS in `bbs.go`.)

Two **catch-all** structs front many modes (the AP-1 violation to split):
- `ErrorMessage` (`operation.go:59-84`, `struct { mode byte }`) — run.go comment
  (`cmd/run.go:1383`) lists v95 cases **30,33,35,37,38,40,42,43,44,47,50,54,58,61**
  (14 mode-only arms).
- `ErrorMessageWithTarget` (`operation.go:89-117`, `struct { mode byte; target
  string }`) — run.go comment (`cmd/run.go:1388`) lists v95 cases **55,56,57**
  (3 target-bearing arms).

Latent AP-2 hard-coded mode bytes:
- `info.go:70` — `Info.Encode` writes `WriteByte(0x1A)` literal.
- `bbs.go:54,167` — `BBSThreadList`/`BBSThread` hard-code mode `0x06`/`0x07`.

### Body functions (`libs/atlas-packet/guild/operation_body.go`)
The **INV-3 footguns to remove**:
- `GuildErrorBody(code string)` (`:64`) — caller picks the operation key.
- `GuildErrorBody2(code string, target string)` (`:70`) — same + target.
- `RequestGuildNameBody` (`:50`) / `RequestGuildEmblemBody` (`:54`) delegate
  through `GuildErrorBody` — rewrite to own fixed-key bodies.
- `GuildInfoBody` (`:146`) bypasses `WithResolvedCode` entirely (calls
  `clientbound.NewInfo(...).Encode`) — fold into the resolved pattern.

The 35 operation-key consts already exist (`operation_body.go:13-47`). These are
the canonical `operations`-table keys; reuse them, do not invent new ones.

### Operations table (seed templates)
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json:1577`
holds the `GuildOperation` writer with an `operations` map keyed by the const
strings → hex mode bytes (e.g. `REQUEST_NAME:0x01`, `REQUEST_AGREEMENT:0x03`,
`INVITE:0x05`, `JOIN_SUCCESS:0x27`, … `MEMBER_TITLE_CHANGE:0x40`,
`EMBLEM_CHANGE:0x42`, `NOTICE_CHANGE:0x44`, `SHOW_TITLES:0x49`, quest errors at
`0x4A`+). The table is ALREADY POPULATED for guild on v83; the task verifies it
per version and fills any gaps, esp. v84.

There is **no** `docs/packets/dispatchers/guild.yaml` yet — this task creates it
(+ `guild_bbs.yaml`) as the per-version source of truth.

### run.go (`tools/packet-audit/cmd/run.go`)
- `CWvsContext::OnGuildResult` (`:1369`) returns `RequestAgreement` as a
  **phantom representative** ("deferred to _pending.md") — INV-4 violation.
- `CWvsContext::OnGuildBBSPacket` (`:1462`) and `CUIFadeYesNo::OnButtonClicked`
  (`:1480`) are top-level catch-all roots, also "deferred to _pending.md".
- `#ErrorMessage` (`:1383`) and `#ErrorMessageWithTarget` (`:1388`) each map ONE
  struct to MANY cases — the catch-all to split into per-mode `#`-entries.
- Per-mode `#`-entries already exist for the structural arms (`#Invite`,
  `#EmblemChange`, `#MemberLeft`, etc.) — these stay, comments get freshened.
- Serverbound guild ops use `prefixName: "Operation", prefixPkg: "guild"` (the op
  byte is the dispatcher prefix). BBS serverbound uses `prefixName: "BBS"`.

### Serverbound AgreementResponse wire bug
`libs/atlas-packet/guild/serverbound/operation_agreement_response.go:27-40`
writes/reads `unk uint32` + `agreed bool`. IDA `CField::SendCreateGuildAgreeMsg`
(run.go:1488) = `Encode1(op) + Encode1(agreed)` — op is the dispatcher prefix,
so the body is **`agreed` (1 byte) only**; the `unk uint32` is the extra field to
drop. **Confirm per version in IDA before editing** (D4).

### STATUS.md matrix rows (`docs/packets/audits/STATUS.md`)
- `GUILD_OPERATION` clientbound row 85: v83/v84/v87/v95/jms all ✅ (aggregate;
  masks the catch-all/footgun debt).
- `GUILD_BBS_PACKET` clientbound row 80: **all ❌** (rows 823-824 BBSThread/List:
  v84 ❌).
- Serverbound rows 625 (`GUILD_OPERATION`), 662 (`BBS_OPERATION`), 667
  (`NEW_YEAR_CARD_REQUEST`): **v84 ❌**.
- Serverbound per-struct rows 825-839: **v84 ❌ across the board**; v83/v87 ❌ for
  `GuildInviteRequest/Join/Kick/RequestCreate/SetMemberTitle/SetNotice/SetTitleNames/Withdraw`.
- `AgreementResponse` row 825: v84 ❌ (and the wire bug above is latent on the
  passing versions).

## 4. Call-site wiring (Explore-verified)

All guild clientbound bodies are emitted by **atlas-channel Kafka consumers**;
atlas-guilds only produces the upstream status events.

- `services/atlas-channel/atlas.com/channel/kafka/consumer/guild/consumer.go`
  — 17 emit sites. **The critical footgun caller:** line **143**
  `GuildErrorBody(errCode)` where `errCode` is a **dynamic string** off the Kafka
  event (`StatusEventErrorBody.Error`). Migration requires a
  **string→fixed-key-body dispatch map** here (an error code → the matching
  per-mode body func). Line 578 also calls `GuildErrorBody(GuildOperationCreateError)`
  (a const — straightforward).
- `…/kafka/consumer/invite/consumer.go:181` —
  `GuildErrorBody2(GuildOperationInviteDenied, targetName)` (const key + target).
  Line 118 `GuildInviteBody(...)`.
- `…/kafka/consumer/session/consumer.go:238` — `GuildInfoBody(...)`.
- Writers: `…/socket/writer/guild_bbs.go` calls `NewBBSThreadList`/`NewBBSThread`
  directly (lines 57/74) — route through the new resolved BBS body funcs.
- Serverbound handlers `socket/handler/guild_operation.go`, `guild_bbs.go`,
  `guild_invite_reject.go` are registered in `main.go` **with `LoggedInValidator`**
  (validators already present — verify, don't assume).
- atlas-guilds producers: `…/guilds/guild/producer.go` status-event providers
  (RequestAgreement/Created/Disbanded/EmblemUpdated/MemberStatusUpdated/
  MemberTitleUpdated/NoticeUpdated/CapacityUpdated/MemberLeft/MemberJoined/
  TitlesUpdated/Error). The `Error` provider is what feeds the dynamic `errCode`.

## 5. Test fixture format
`libs/atlas-packet/guild/clientbound/operation_test.go` already uses the
`// packet-audit:verify packet=<pkg/dir/PacketName> version=<v> ida=<addr>`
marker convention (lines 9-70) and `pt` (`libs/atlas-packet/test`) helpers. Each
new discrete struct needs its own marker lines per supported version.

## 6. Open questions resolved at execution (NOT now) — from PRD §9 / design §9
1. Exact supported-arm set + per-version mode bytes behind the catch-alls — IDA
   enumeration of the `OnGuildResult` / `OnGuildBBSPacket` switches (Task 1).
   The candidate key set is the existing operations-table keys (§3 above); IDA
   confirms membership, mode byte, and per-version presence.
2. Version-absent (⬜) vs unimplemented (must reach ✅) per arm — per-arm vs each
   IDB.
3. v84 fix scope — operations table only vs registry opcode reshift vs both —
   confirm vs the gms_84 template + registry (task-100 carryover pattern).
4. Live tenant/version set for the FR-22 patch+restart — determined at execution
   via k8s/Grafana MCP.
5. `RequestAgreement` shared by `#RequestAgreement` and `#AgreementResponse`
   (run.go:1373 and :1495) — both currently return the same clientbound struct;
   confirm and split if the two `#`-entries represent distinct modes.

## 7. Hard rules (from CLAUDE.md / project memory)
- **Grounding:** every byte/mode/opcode traces to a decompile line (fn+addr) or a
  checked-in export entry, cited in the struct/test comment. No MapleStory-memory
  values. An unresolved packet-audit fname is **stop-and-ask** — never
  auto-re-export or fake a hash.
- **IDA:** resolve the IDB by `select_instance(port)` for v83/v87/v95/jms; confirm
  version match before reading; v84 has no IDB → treat as v83 unless the
  registry/template proves a shift. Use `func_query` with `name_regex`.
- **`MajorVersion` gate:** `>=87` not `>83`; v84..86 == v83 unless IDA proves
  otherwise (the off-by-one).
- **No deferral / no TODO stubs** in landed commits; finish bounded work.
- **Worktree discipline:** all work in
  `.worktrees/task-103-guild-dispatcher-family`; verify branch after each commit.
- **Path robustness:** run all packet-audit/path logic from the repo (worktree)
  root, not relying on `../../` nesting (the INV-4 CI bug).

## 8. Verification gates (every must exit 0 before "done")
- `go run ./tools/packet-audit dispatcher-lint` — clean; guild removed from
  `dispatcher-lint-baseline.yaml`.
- `go run ./tools/packet-audit matrix --check` — no orphan/dangling/stale/drift,
  no conflict-count increase; `STATUS.md`/`status.json` regenerated (toolSha).
- `go run ./tools/packet-audit fname-doc --check` — clean.
- `go run ./tools/packet-audit operations --check` — clean.
- `go build ./...`, `go vet ./...`, `go test -race ./...` clean in every changed
  module (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`,
  `services/atlas-guilds` if touched).
- `docker buildx bake atlas-channel` (+ `atlas-guilds` if its go.mod changed) from
  the worktree root.
- `tools/redis-key-guard.sh` (only if Redis touched — not expected).

## 9. Changed-module / go.mod map
- `libs/atlas-packet` — primary (structs, bodies, fixtures, codec fix).
- `tools/packet-audit` — run.go rewire, regen STATUS.
- `services/atlas-channel` — call-site migration + validator audit.
- `services/atlas-guilds` — only if a producer must emit a newly-split packet.
- `docs/packets/...`, seed templates, live-config runbook — non-go.
Touching a service `go.mod` ⇒ that service needs a `docker buildx bake`.

## 10. Enumerated arm table (Task 1 — IDA-grounded)

All values below are read directly from each version's decompiled switch (no
MapleStory-knowledge / inference). OnGuildResult addresses: gms_v83 `0xa37490`,
gms_v84 `0xa82e2b`, gms_v87 `0xacf7d3`, gms_v95 `0xa0d3b0`, jms_v185 `0xb22518`.
Mode bytes shown in hex. `shape`: mode-only = ErrorMessage `{mode}`; target =
ErrorMessageWithTarget `{mode,target}` (NOTE: none of the GuildOperation error
arms actually read a trailing string in any version — they are ALL mode-only on
the wire; the client substitutes the StringPool message locally, so
ErrorMessageWithTarget is NOT used by any OnGuildResult arm — see flag F4);
structured = arm with its own body struct.

### GuildOperation (CWvsContext::OnGuildResult)

| key | struct | shape | v83 | v84 | v87 | v95 | jms | present |
|---|---|---|---|---|---|---|---|---|
| REQUEST_NAME | (RequestGuildNameBody→ErrorMessage) | mode-only | 0x01 | 0x01 | 0x01 | 0x01 | 0x01 | all |
| REQUEST_AGREEMENT | RequestAgreement | structured (partyId+leader+guild) | 0x03 | 0x03 | 0x03 | 0x03 | 0x03 | all |
| INVITE | Invite | structured (guildId+name[+v87+:unk+skill]) | 0x05 | 0x05 | 0x05 | 0x05 | 0x05 | all |
| REQUEST_EMBLEM | (RequestGuildEmblemBody→ErrorMessage) | mode-only | 0x11 | 0x11 | 0x11 | 0x11 | 0x11 | all |
| THE_NAME_IS_ALREADY_IN_USE_PLEASE_TRY_OTHER_ONES | ErrorMessage | mode-only | 0x1C | 0x1C | 0x1C | 0x1E | 0x1C | all |
| SOMEBODY_HAS_DISAGREED_TO_FORM_A_GUILD | ErrorMessage | mode-only | 0x24 | 0x24 | 0x24 | 0x26 | 0x24 | all |
| THE_PROBLEM…FORMING_THE_GUILD…TRY_AGAIN | ErrorMessage | mode-only | 0x26 | 0x26 | 0x26 | 0x28 | 0x26 | all |
| JOIN_SUCCESS | MemberJoined | structured (guildId+charId+GUILDMEMBER 37B) | 0x27 | 0x27 | 0x27 | 0x29 | 0x27 | all |
| ALREADY_JOINED_THE_GUILD | ErrorMessage | mode-only | 0x28 | 0x28 | 0x28 | 0x2A | 0x28 | all |
| THE_GUILD…MAX_NUMBER_OF_USERS | ErrorMessage | mode-only | 0x29 | 0x29 | 0x29 | 0x2B | 0x29 | all |
| THE_CHARACTER_CANNOT_BE_FOUND_IN_THE_CURRENT_CHANNEL | ErrorMessage | mode-only | 0x2A | 0x2A | 0x2A | 0x2C | 0x2A | all |
| MEMBER_QUIT_SUCCESS | MemberLeft | structured (guildId+charId+name) | 0x2C | 0x2C | 0x2C | 0x2E | 0x2C | all |
| MEMBER_QUIT_ERROR_NOT_IN_GUILD | ErrorMessage | mode-only | 0x2D | 0x2D | 0x2D | 0x2F | 0x2D | all |
| MEMBER_EXPELLED_SUCCESS | MemberExpel | structured (guildId+charId+name) | 0x2F | 0x2F | 0x2F | 0x31 | 0x2F | all |
| MEMBER_EXPELLED_ERROR_NOT_IN_GUILD | ErrorMessage | mode-only | 0x30 | 0x30 | 0x30 | 0x32 | 0x30 | all |
| DISBAND_SUCCESS | Disband | structured (guildId) | 0x32 | 0x32 | 0x32 | 0x34 | 0x32 | all |
| THE_PROBLEM…DISBANDING…TRY_AGAIN | ErrorMessage | mode-only | 0x34 | 0x34 | 0x34 | 0x36 | 0x34 | all |
| IS_CURRENTLY_NOT_ACCEPTING_GUILD_INVITE_MESSAGE | ErrorMessage | mode-only | 0x35 | 0x35 | 0x35 | 0x37 | 0x35 | all |
| IS_TAKING_CARE_OF_ANOTHER_INVITATION | ErrorMessage | mode-only | 0x36 | 0x36 | 0x36 | 0x38 | 0x36 | all |
| HAS_DENIED_YOUR_GUILD_INVITATION | ErrorMessage | mode-only | 0x37 | 0x37 | 0x37 | 0x39 | 0x37 | all |
| ADMIN_CANNOT_MAKE_A_GUILD | ErrorMessage | mode-only | 0x38 | 0x38 | 0x38 | 0x3A | 0x38 | all |
| CONGRATULATION…INCREASED_TO (CapacityChange) | CapacityChange | structured (guildId+capacity byte) | 0x3A | 0x3A | 0x3A | 0x3C | 0x3A | all |
| THE_PROBLEM…INCREASING…TRY_AGAIN | ErrorMessage | mode-only | 0x3B | 0x3B | 0x3B | 0x3D | 0x3B | all |
| MEMBER_UPDATE | (no body func yet) | structured (guildId+charId+level+job) | 0x3C | 0x3C | 0x3C | 0x3E | 0x3C | all |
| MEMBER_ONLINE | MemberStatusUpdate | structured (guildId+charId+online byte) | 0x3D | 0x3D | 0x3D | 0x3F | 0x3D | all |
| TITLE_UPDATE | TitleChange | structured (guildId+5×str) | 0x3E | 0x3E | 0x3E | 0x40 | 0x3E | all |
| MEMBER_TITLE_CHANGE | MemberTitleUpdate | structured (guildId+charId+title byte) | 0x40 | 0x40 | 0x40 | 0x42 | 0x40 | all |
| EMBLEM_CHANGE | EmblemChange | structured (guildId+markBg+bgColor+mark+color) | 0x42 | 0x42 | 0x42 | 0x45 | 0x42 | all |
| NOTICE_CHANGE | NoticeChange | structured (guildId+notice str) | 0x44 | 0x44 | 0x44 | 0x47 | 0x44 | all |
| SHOW_TITLES | (no body func yet) | structured (guildId+count+name+5×int) | 0x49 | 0x49 | 0x49 | 0x4C | 0x49 | all |
| THERE_ARE_LESS_THAN_6_MEMBERS… | ErrorMessage | mode-only | 0x4A | 0x4A | 0x4A | 0x4D | 0x4A | all |
| THE_USER_THAT_REGISTERED_HAS_DISCONNECTED… | ErrorMessage | mode-only | 0x4B | 0x4B | 0x4B | 0x4E | 0x4B | all |
| QUEST_WAITING_NOTICE | (no body func yet) | structured (Decode1+Decode4) | 0x4C | 0x4C | 0x4C | 0x4F | 0x4C | all |
| BOARD_AUTH_KEY_UPDATE | (no body func yet) | structured (DecodeStr) | 0x4D | 0x4D | 0x4D | 0x50 | 0x4D | all |
| SET_SKILL_RESPONSE | (no body func yet) | structured (guildId+skillId+SKILLENTRY) | 0x4E | 0x4E | 0x4E | 0x51 | ⬜ | gms only |

### GuildBBS (CUIGuildBBS::OnGuildBBSPacket — dispatch on `Decode1 - 6`)

Addresses: gms_v83 `0x816c32`, gms_v84 `0x841ec9` (UNNAMED sub_841EC9 via thunk
`0xa5c77c`), gms_v87 `0x87a5df`, gms_v95 `0x7c8260`, jms_v185 = absent. Mode
bytes are STRUCT LITERALS (bbs.go `WriteByte(0x06)`/`(0x07)`), NOT config-resolved.

| arm (handler) | struct | shape | mode | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|---|---|
| OnLoadListResult | BBSThreadList | structured (notice + thread page) | 6 | 0x06 | 0x06 | 0x06 | 0x06 | ⬜ |
| OnViewEntryResult | BBSThread | structured (thread + replies) | 7 | 0x07 | 0x07 | 0x07 | 0x07 | ⬜ |
| OnEntryNotFound | (none — mode-only) | mode-only | 8 | 0x08 | 0x08 | 0x08 | 0x08 | ⬜ |

### Flags / findings (real input for later tasks)

- **F1 — v95 mode-byte SHIFT (non-uniform):** v95 GuildOperation modes are shifted
  vs v83/v84/v87/jms by 0 (≤0x11), +2 (0x1C..0x44 range), +3 (EMBLEM/NOTICE and the
  0x49+ quest/skill range). Mapped per arm by client read-order AND by decrypting
  the StringPool message each arm shows (ms_aKey @0xb98830, rotl(seed)^cipher;
  decryptor reproduced in-process). This is the SAME family as the opcode-table
  drift bug — v95 modes are NOT a copy of v83.
- **F2 — v83/v84/v87/jms modes are BYTE-IDENTICAL.** v84 confirmed from the LIVE
  v84 IDB (port 13337), not folded from v83. The two-switch (`v4 > 0x32`) +
  low-value if-chain shape and every case value match v83.
- **F3 — Invite body shape, NOT a mode-byte issue:** the INVITE mode byte is 0x05
  in every version, but the BODY differs. v83 reads only `guildId + name`;
  **v84, v87, v95, jms ALL read `guildId + name + unk(4) + skillId(4)`** (the 2
  trailing ints). The existing `Invite` struct gates the trailing ints on
  `(GMS && MajorAtLeast(87)) || JMS`, i.e. it treats v84 like v83 — **this is WRONG
  per the live v84 IDB** (v84 case 5 @0xa82e2b L1212-1216 reads the 2 ints). Later
  task must widen the gate to include v84 (or `>=84`). Records contradict the
  struct comment "v84..86 == v83".
- **F4 — No OnGuildResult arm uses ErrorMessageWithTarget.** Every error arm in
  every version is mode-only on the wire (client looks up the StringPool message;
  the IS_CURRENTLY_NOT_ACCEPTING / ANOTHER_INVITATION / DENIED messages embed the
  name via local `%s` formatting of a value the client already has, NOT a wire
  string). `ErrorMessageWithTarget`/`GuildErrorBody2` is therefore an orphan codec
  with no matching OnGuildResult arm — flag for the migration (it currently maps
  to no enumerated arm).
- **F5 — RequestAgreement vs AgreementResponse (the run.go question):** there is
  exactly ONE clientbound mode here — **REQUEST_AGREEMENT = 0x03** (case 3 in every
  version: `Decode4(partyId) + DecodeStr(leaderName) + DecodeStr(guildName)`, the
  create-guild-agree-dialog request). run.go has BOTH
  `#RequestAgreement` (L1373) and `#AgreementResponse` (L1495→L1511) pointing at the
  SAME clientbound `RequestAgreement` struct. The serverbound `AgreementResponse`
  (CField::SendCreateGuildAgreeMsg, L1487) is a DIFFERENT, serverbound packet (the
  member's yes/no reply). So: clientbound side = ONE struct (RequestAgreement,
  mode 0x03); the two clientbound run.go `#`-entries are aliases of one mode, NOT
  two distinct clientbound modes. No second clientbound struct is needed.
- **F6 — keys absent from the IDA switch:** none. Every GuildOperation key in
  operation_body.go and the gms_83/gms_84 templates resolves to a real switch case
  in v83/v84/v87 (and to its shifted case in v95). No invented structs.
- **F7 — switch cases with NO existing key (NOT named — flagged, not invented):**
  v83 has NPCsay/CHATLOG-only arms with no Atlas key (e.g. 0x1A guild-info-set /
  GUILDDATA::Decode→0x1C in v95; v83 0x1F/0x20/0x21/0x23 minlevel/already-joined
  NPCsay; v83 0x35 first-switch is the WAIT_AND_SEE NPCSay). v95 ADDS NEW upper
  arms with no v83 analog and no Atlas key: **0x4B** (guildId+nPoint+nLevel guild
  points/level update) and **0x52** ("guild request not accepted, unknown reason"
  default path). These are NOT in the Atlas key set; per the stop-and-ask rule they
  are NOT named/added here — surfaced for a design decision in a later task.
- **F8 — `operations --check` RECORDED result (Step 7):** exit 1, **0 drift, 0
  extra, 104 missing.** All 104 MISSING are the gms_87 / gms_95 / jms GuildOperation
  keys whose seed templates have an EMPTY operations map (ops_count=0 —
  bug_operations_mode_tables_missing_v87_v95_jms). gms_83/gms_84 = 0 drift/extra
  (their templates already match these IDA values exactly). Not papered over: a
  later task runs `packet-audit operations` (generate) to populate v87/v95/jms from
  guild.yaml. GuildBBS produced 0 entries (correctly tool-ignored — no `writer:`).
- **F9 — stale run.go comments to freshen later:** the `#`-entry narrative for
  guild arms is point-in-time. The Invite comment claims v84..86==v83 (disproven by
  F3). CapacityChange/Invite "[Prior ❌ … stale]" notes are accurate that the structs
  are fixed, but the v84 Invite gate itself is wrong (F3).
