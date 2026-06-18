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
