# Party + Buddy Dispatcher Family — Context

Task: task-105-party-buddy-dispatcher-family
Companion to: `prd.md` (approved), `design.md` (approved), `plan.md`
Governing pattern: `docs/packets/DISPATCHER_FAMILY.md`
Executed exemplar to copy: **task-103 guild** (`docs/tasks/task-103-guild-dispatcher-family/plan.md`,
`libs/atlas-packet/guild/clientbound/operation.go`, `libs/atlas-packet/guild/operation_body.go`,
`docs/packets/dispatchers/guild.yaml`) and task-104 message.

---

## 1. What "done" means (one line)

Both error catch-alls (`party/clientbound.Error`, `buddy/clientbound.Error`) split into discrete
per-mode structs + fixed-key body funcs; `party.yaml`/`buddy.yaml` authored (IDA-verified, 5
versions); v87/v95/jms operations tables populated; all atlas-channel call sites migrated; both
families removed from `dispatcher-lint-baseline.yaml` (→ **empty**); all four packet-audit gates +
build/vet/test/bake/redis-guard exit 0.

## 2. Grounding & honesty contract (read before any IDA task)

Identical to the guild plan's contract:

- Every byte, mode value, field width, and per-version presence MUST trace to a decompile line
  (function + address) or a checked-in export entry, cited in the struct/test comment. **No values
  from MapleStory general knowledge or memory** (CLAUDE.md "Verification Over Memory", "No Inventing").
- Resolve the IDB by `select_instance(port)` and confirm the version matches before reading:
  gms_v83 :13342, gms_v84 :13337, gms_v87 :13341, gms_v95 :13340, jms_v185 :13339.
- v84 **has a live IDB** (port 13337, used in task-103) — read it, do not assume v84≡v83. Gate
  version divergence as `>=87`, never `>83`.
- v95 is the **non-uniform shift** family (same as the opcode-table / guild bug): read each v95 arm
  from the v95 switch and cross-check the decrypted StringPool message; never fold from v83.
- An unresolved packet-audit fname is a **stop-and-ask** — never auto-re-export, substitute an fname,
  or fake a hash (`feedback_unresolved_fname_escalate`).
- No `// TODO` / stubbed handler / 501 in any landed commit.
- All work happens in the task worktree on branch `task-105-party-buddy-dispatcher-family`. After
  every commit: `git rev-parse --show-toplevel` (must end
  `/.worktrees/task-105-party-buddy-dispatcher-family`) and `git branch --show-current` (must be
  `task-105-party-buddy-dispatcher-family`).
- Run all `packet-audit` commands from the worktree root.

## 3. Current-state map (file:line, grounded)

### Party (`CWvsContext::OnPartyResult`, writer `PartyOperation`, op `PARTY_OPERATION`)
- `libs/atlas-packet/party/clientbound/error.go:13-47` — shared `Error{mode, name}` struct;
  `Encode` writes `WriteByte(mode) + WriteAsciiString(name)`. `// packet-audit:fname
  CWvsContext::OnPartyResult#Error` (line 12).
- `libs/atlas-packet/party/clientbound/operation_body.go:78-82` — `PartyErrorBody(code string, name
  string)` — the AP-4/INV-3 footgun (caller selects the mode via `code`).
- Non-error arms already discrete (out of scope to rewrite): `created.go`, `disband.go`, `left.go`
  (also serves EXPEL), `join.go`, `update.go`, `change_leader.go`, `invite.go`, `town_portal.go`,
  `member_hp.go` (separate `CUserRemote::OnReceiveHP`, NOT an OnPartyResult arm).
- Non-error body funcs + key consts: `operation_body.go:13-23` (`PartyOperationCreated`…
  `PartyOperationTownPortal`).

### Buddy (`CWvsContext::OnFriendResult`, writer `BuddyOperation`, op `BUDDYLIST`)
- `libs/atlas-packet/buddy/clientbound/error.go:15-50` — shared `Error{mode, hasExtra}` struct;
  `Encode` writes `WriteByte(mode)` + (if `hasExtra`) `WriteByte(0)`. `// packet-audit:fname
  CWvsContext::OnFriendResult#Error` (line 14). Const `BuddyErrorWriter = "BuddyError"` (line 12).
- `libs/atlas-packet/buddy/operation_body.go:50-55` — `BuddyErrorBody(errorCode string)` — the
  semantic INV-3 footgun (`errorCode` flows into the `operations` key; escaped by-name check,
  caught by task-101 hardening). `hasExtra := errorCode == BuddyOperationErrorUnknownError`.
- Non-error arms already discrete (out of scope): `invite.go`, `list_update.go`, `update.go`,
  `channel_change.go`, `capacity_update.go`.
- All key consts (error + non-error): `operation_body.go:13-30` (`BuddyOperationUpdate`…
  `BuddyOperationErrorUnknownError4`).

### Audit wiring (`tools/packet-audit/cmd/run.go`)
- Party catch-all: `case "CWvsContext::OnPartyResult#Error":` **run.go:1373** → `{name:"Error",
  pkg:"party", dir:Clientbound}`. Comment: "mode=9,10,13,17,18,22,29,32–34,36 … Sub-op enum
  deferred to _pending.md."
- Buddy catch-all: `case "CWvsContext::OnFriendResult#Error":` **run.go:1130** → `{name:"Error",
  pkg:"buddy", dir:Clientbound}`. Comment: "mode=0x0B–0x17 … Sub-op enum deferred to _pending.md."
- Party non-error `#`-entries: run.go:1356-1394 (Created/Invite/Disband/ChangeLeader/Join/Left/
  Update) — unchanged.
- Buddy non-error `#`-entries: run.go:1122-1147 (CapacityUpdate/ChannelChange/Invite/ListUpdate/
  Update) — unchanged.

### Baseline (`docs/packets/dispatcher-lint-baseline.yaml`)
- `exempt_families:` currently `[CWvsContext::OnPartyResult, CWvsContext::OnFriendResult]` (guild
  already removed by task-103). After this task: **empty**.

### Call sites (`services/atlas-channel`)
- `socket/handler/party_operation.go:97` → `PartyErrorBody("UNABLE_TO_FIND_THE_CHARACTER", sp.Name())`
  — literal, direct.
- `socket/handler/party_operation.go:106` →
  `PartyErrorBody("UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL", sp.Name())` — literal,
  direct.
- `kafka/consumer/invite/consumer.go:171` → `PartyErrorBody("HAVE_DENIED_REQUEST_TO_THE_PARTY",
  targetName)` — literal, direct.
- `kafka/consumer/party/consumer.go:452` — `partyError(...)(errorType, name)` →
  `PartyErrorBody(errorType, name)`; `errorType` is **runtime** (`e.Body.Type`, the Kafka
  `party2.ErrorEventBody.Type`). Needs a call-site switch (D3).
- `kafka/consumer/buddylist/consumer.go:238` — `buddyError(...)(errorCode)` →
  `BuddyErrorBody(errorCode)`; `errorCode` is **runtime** (`c.Body.Error`, the Kafka
  `buddylist2.ErrorStatusEventBody.Error`). Needs a call-site switch (D3).

## 4. Enumerated arm tables (from the v83 seed templates — the grounded key set)

The v83/v84 `operations` tables are FULL; they are the authoritative key list to reconcile the IDA
switch against. The mode bytes below are the **v83** values (hex from the gms_83 template); per-
version modes (esp. v95 shift) come from IDA in Task 1.

### Party `PartyOperation` (`template_gms_83_1.json:1538`) — v83 modes
Non-error (already discrete, keys already wired): `INVITE`=0x04, `UPDATE`=0x07, `CREATED`=0x08,
`LEAVE`/`DISBAND`/`EXPEL`=0x0C, `JOIN`=0x0F, `CHANGE_LEADER`=0x1B, `TOWN_PORTAL`=0x25.

Error/notice arms (currently fronted by shared `Error`; **these get new discrete structs**):
| key | v83 | currently emitted? |
|---|---|---|
| `ALREADY_HAVE_JOINED_A_PARTY_1` | 0x09 | runtime (atlas-parties) |
| `A_BEGINNER_CANT_CREATE_A_PARTY` | 0x0A | runtime |
| `YOU_HAVE_YET_TO_JOIN_A_PARTY` | 0x0D | runtime |
| `ALREADY_HAVE_JOINED_A_PARTY_2` | 0x10 | runtime |
| `THE_PARTY_YOURE_TRYING_TO_JOIN_IS_ALREADY_IN_FULL_CAPACITY` | 0x11 | runtime |
| `UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL` | 0x13 | literal (handler:106) + runtime |
| `IS_CURRENTLY_BLOCKING_ANY_PARTY_INVITATIONS` | 0x15 | runtime |
| `IS_TAKING_CARE_OF_ANOTHER_INVITATION` | 0x16 | runtime |
| `HAVE_DENIED_REQUEST_TO_THE_PARTY` | 0x17 | literal (invite:171) + runtime |
| `CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP` | 0x19 | runtime |
| `THIS_CAN_ONLY_BE_GIVEN_TO_A_PARTY_MEMBER_WITHIN_THE_VICINITY` | 0x1C | runtime |
| `UNABLE_TO_HAND_OVER_THE_LEADERSHIP_POST_NO_PARTY_MEMBER_IS_CURRENTLY_WITHIN_THE` | 0x1D | runtime |
| `YOU_MAY_ONLY_CHANGE_WITH_THE_PARTY_MEMBER_THATS_ON_THE_SAME_CHANNEL` | 0x1E | runtime |
| `AS_A_GM_YOURE_FORBIDDEN_FROM_CREATING_A_PARTY` | 0x20 | runtime |
| `UNABLE_TO_FIND_THE_CHARACTER` | 0x21 | literal (handler:97) + runtime |

Runtime error-type strings the consumer can receive (`atlas-parties` `party/kafka.go:22-39`,
`EventPartyStatusErrorType*`): the 15 keys above **plus** `ERROR_UNEXPECTED` (`ERROR_UNEXPECTED`
has no operations-table key — the call-site switch's logged default; do NOT invent a mode for it).

**Per-arm wire-shape OPEN ITEM (design §8):** the current shared `Error` always writes
`mode + AsciiString name`. Most party StringPool notices read **mode only** (no `%s` name); only the
character/invite-target arms (`UNABLE_TO_FIND_THE_CHARACTER`, `…IN_THIS_CHANNEL`,
`HAVE_DENIED_REQUEST_TO_THE_PARTY`, the two `IS_*` invite arms) read a trailing name. The IDA read
order per arm decides `struct{mode}` vs `struct{mode,name}` (FR-2.3). Where the current code writes
a name the client does NOT read, **IDA wins** (mode-only struct) and the "byte-identical regression"
guard (D8) is scoped to the bytes the client actually consumes — document the divergence per arm,
never paper over it.

### Buddy `BuddyOperation` (`template_gms_83_1.json:1570`) — v83 modes
Non-error (already discrete): `UPDATE`=0x07, `BUDDY_UPDATE`=0x08, `INVITE`=0x09,
`BUDDY_CHANNEL_CHANGE`=0x14, `CAPACITY_CHANGE`=0x15.

Error/unknown arms (currently fronted by shared `Error`; **these get new discrete structs**):
| key | v83 | trailing byte? | currently emitted? |
|---|---|---|---|
| `UNKNOWN_1` | 0x0A | tbd (IDA) | future-feature |
| `BUDDY_LIST_FULL` | 0x0B | no | runtime |
| `OTHER_BUDDY_LIST_FULL` | 0x0C | no | runtime |
| `ALREADY_BUDDY` | 0x0D | no | runtime |
| `CANNOT_BUDDY_GM` | 0x0E | no | runtime |
| `CHARACTER_NOT_FOUND` | 0x0F | no | runtime |
| `UNKNOWN_ERROR` | 0x10 | **yes** (`hasExtra`) | runtime |
| `UNKNOWN_ERROR_2` | 0x11 | tbd (IDA) | future-feature |
| `UNKNOWN_2` | 0x12 | tbd (IDA) | future-feature |
| `UNKNOWN_ERROR_3` | 0x13 | tbd (IDA) | future-feature |
| `UNKNOWN_ERROR_4` | 0x16 | tbd (IDA) | future-feature |

Runtime error strings the consumer can receive (`atlas-channel` `kafka/message/buddylist/kafka.go:
39-44`, `StatusEventError*`): `BUDDY_LIST_FULL`, `OTHER_BUDDY_LIST_FULL`, `ALREADY_BUDDY`,
`CANNOT_BUDDY_GM`, `CHARACTER_NOT_FOUND`, `UNKNOWN_ERROR` (6 values). The other 5 arms are
future-feature entry points (FR-3.4 — every discrete struct needs a body func even if no caller).

**`UNKNOWN_ERROR` trailing byte:** confirm from the `OnFriendResult` decompile **which** mode(s)
read the trailing int — model each as its own discrete struct whose `Encode` writes the extra byte
(NO `hasExtra` flag — the arm identity is the struct, design D1).

## 5. KEY FINDING — v87/v95/jms operations tables are empty/near-empty

Measured from the seed templates:
- `template_gms_87_1.json`: `PartyOperation` ops_count=**1**, `BuddyOperation` ops_count=**0**.
- `template_gms_95_1.json`: `PartyOperation` ops_count=**1**, `BuddyOperation` ops_count=**0**.
- `template_jms_185_1.json`: `PartyOperation` = `{TOWN_PORTAL: 0x28}` only, `BuddyOperation` has **no
  `options` block at all**.

Consequence: on v87/v95/jms today, virtually every party/buddy arm (error AND non-error) resolves to
the `99` fallback → client crash. This is the `bug_operations_mode_tables_missing_v87_v95_jms` gap,
amplified vs guild. **Resolution (guild precedent):** `party.yaml`/`buddy.yaml` enumerate the
**complete** per-version table (all keys, error + non-error), and `packet-audit operations` (generate)
populates the full v87/v95/jms tables from the yaml. This incidentally restores full v87/v95/jms
party/buddy functionality — a beneficial, design-consistent side effect (D6), not scope creep: the
non-error STRUCTS are untouched; only their KEYS appear in the yaml/table. v83/v84 tables already
match and stay unchanged.

`operations --check` reconciles yaml→seed (every yaml key present in the seed with the matching
per-version mode). It does not require the seed to have no extra keys. Default to the **complete**
yaml; if reading the full non-error switch per version on v87/v95/jms surfaces anything unexpected,
stop-and-note rather than guess.

## 6. The canonical shapes to copy (from guild)

- **Mode-only error struct** → `guild/clientbound/operation.go` `RequestName`/`CreateErrorNameInUse`
  (`struct{ mode byte }`, `Encode` writes `WriteByte(mode)` only).
- **Name/target-bearing error struct** → guild `InviteDenied` (`struct{ mode byte; target string }`,
  `Encode` writes `WriteByte(mode) + WriteAsciiString(target)`).
- **Mode-only body func** → guild `GuildCreateErrorNameInUseBody()` (no params) using
  `WithResolvedCode("operations", FIXED_KEY, func(mode byte) packet.Encoder { return New…(mode) })`.
- **Name-bearing body func** → guild `GuildInviteDeniedBody(target string)`.
- **`party.yaml`/`buddy.yaml`** → mirror `docs/packets/dispatchers/guild.yaml` header + per-key
  per-version `modes:` block (decimal mode bytes; per-version function addresses + v95-shift note in
  the header).
- **run.go per-mode `#`-entry** → guild `case "CWvsContext::OnGuildResult#CreateErrorNameInUse":`
  returning `{name:"CreateErrorNameInUse", pkg:"guild", dir:Clientbound}`.
- **Runtime-code → body dispatch map at the call site** → guild plan Task 5 Step 1 (`map[string]…`
  with a logged default for unmapped codes; AP-4 footgun gone).

**Package-layout asymmetry (preserve, do NOT normalize — design §3):** party body funcs live in
package `clientbound` (`party/clientbound/operation_body.go`); buddy body funcs live in the parent
package `buddy` (`buddy/operation_body.go`, calling `clientbound.New*`). Keep each where it is.

## 7. Test helper pattern

Read an existing `*_test.go` in each package first (`party/clientbound/created_test.go`,
`buddy/clientbound/update_test.go`) to copy the exact byte-fixture helper API — do NOT invent helper
names. Use the project Builder pattern for any model setup; no `*_testhelpers.go` (CLAUDE.md).

## 8. Verification gates (run from worktree root)

```bash
go run ./tools/packet-audit dispatcher-lint     # 0 with EMPTY baseline after de-baseline
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
( cd libs/atlas-packet   && go build ./... && go vet ./... && go test -race ./... )
( cd tools/packet-audit  && go build ./... && go vet ./... && go test -race ./... )
( cd services/atlas-channel && go build ./... && go vet ./... && go test -race ./... )
docker buildx bake atlas-channel                # only service go.mod touched
tools/redis-key-guard.sh                        # repo root, GOWORK=off (no redis change expected)
```

`libs/atlas-packet` and `tools/packet-audit` are NOT bake targets (confirm during execution);
`atlas-channel` is the only service whose `go.mod` is touched.

## 9. Scope guardrails (PRD §2 non-goals)

Out of scope: `pet` `PetDespawnBody(reason)` footgun (separate unenrolled family);
`CWvsContext::OnAllianceResult`; new party/buddy gameplay/error conditions/messages; rewriting the
already-discrete non-error arms beyond the operations-key reconciliation; normalizing the
party-vs-buddy package-layout asymmetry; **live-tenant config patching** (operational — record in a
runbook only, unlike guild task-103 which executed it; PRD §6 records live v87/v95/jms tenants need
the patch + channel restart but this task does not execute it).
