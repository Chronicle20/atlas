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

## 10. Enumerated arm tables (IDA) — task-105 Task 1

All values below are read from each version's decompiled `CWvsContext::OnPartyResult`
/ `OnFriendResult` switch. Mode bytes are shown in DECIMAL (= the wire byte = the
`case` value); v83 decimal matches the gms_83 seed hex (e.g. 0x25 = 37).

### 10.0 CORRECTED IDA port table (the documented ports are STALE)
`list_instances` (verified by binary name, task-105 Task 1) — the running set:

| version | port (ACTUAL) | port (docs, WRONG) | binary |
|---|---|---|---|
| gms_v83  | 13341 | 13342 | MapleStory_dump.exe (path v83_Me) |
| gms_v84  | 13337 | 13337 | GMS_v84.1_U_DEVM.exe |
| gms_v87  | 13340 | 13341 | GMSv87_4GB.exe |
| gms_v95  | 13339 | 13340 | GMS_v95.0_U_DEVM.exe |
| jms_v185 | 13338 | 13339 | MapleStory_dump_SCY.exe |

Port 13342 does not exist. guild.yaml's header ports are stale and were NOT copied.

### 10.1 OnPartyResult function addresses
| version | addr | note |
|---|---|---|
| gms_v83  | **0xa3e31c** | TRUE v83: clean switch, named SP_* symbols, TOWN_PORTAL case 0x25. **NOTE: the v83 IDB ALSO has a duplicate `OnPartyResult` at 0xa89cf3 (v84-layout, TOWN_PORTAL 0x28) that func_query returns FIRST — do NOT read it for v83.** |
| gms_v84  | 0xa89cf3 | if-chain; TOWN_PORTAL case 0x28 (town_portal.go-confirmed) |
| gms_v87  | 0xad697a | if-chain w/ CHATLOG_ADD; CHANGE_LEADER 0x1F, TOWN_PORTAL 0x29 |
| gms_v95  | 0xa10ab0 | clean switch; CHANGE_LEADER 0x1F (case31), TOWN_PORTAL 0x2E (case46) |
| jms_v185 | 0xb297e7 | if-chain; TOWN_PORTAL case 0x28 |

### 10.2 OnFriendResult function addresses
| version | addr |
|---|---|
| gms_v83  | 0xa8ada2 |
| gms_v84  | 0xa8ada2 |
| gms_v87  | 0xad7ae5 |
| gms_v95  | 0xa12630 |
| jms_v185 | 0xb2a873 |

### 10.3 Party `PartyOperation` — key | struct-name | shape | per-version mode (dec) | present
"shape" is the CLIENT read order at that arm (IDA wins over the current Go Error).
`name` = trailing DecodeStr; `mode-only` = no body after the mode byte.

| key | struct-name | shape (IDA) | v83 | v84 | v87 | v95 | jms | present |
|---|---|---|---|---|---|---|---|---|
| INVITE | Invite (exists) | id+name+job+lvl | 4 | 4 | 4 | 4 | 4 | ✅ |
| UPDATE | Update (exists) | PARTYDATA | 7 | 7 | 7 | 7 | 7 | ✅ |
| CREATED | Created (exists) | partyId(+door) | 8 | 8 | 8 | 8 | 8 | ✅ |
| ALREADY_HAVE_JOINED_A_PARTY_1 | new | mode-only | 9 | 9 | 9 | 9 | 9 | ✅ |
| A_BEGINNER_CANT_CREATE_A_PARTY | new | mode-only | 10 | 10 | 10 | 10 | 10 | ✅ |
| LEAVE / DISBAND / EXPEL | Left/Disband (exist) | members | 12 | 12 | 12 | 12 | 12 | ✅ |
| YOU_HAVE_YET_TO_JOIN_A_PARTY | new | mode-only | 13 | 13 | 13 | 13 | 13 | ✅ |
| JOIN | Join (exists) | name+members | 15 | 15 | 15 | 15 | 15 | ✅ |
| ALREADY_HAVE_JOINED_A_PARTY_2 | new | mode-only | 16 | 16 | **17** | **17** | **17** | ✅ (v87/v95/jms +1, see 10.3a) |
| THE_PARTY..FULL_CAPACITY | new | mode-only | 17 | 17 | **18** | **18** | **18** | ✅ (v87/v95/jms +1, see 10.3a) |
| UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL | new | **mode-only** (v83 case19; current Go writes a name the client does NOT read — IDA wins) | 19 | 19 | ⬜ absent | ⬜ absent | ⬜ absent | resolved (absent v87/v95/jms) |
| IS_CURRENTLY_BLOCKING_ANY_PARTY_INVITATIONS | new | **name** (v83 case21 DecodeStr) | 21 | 21 | ⬜ absent | ⬜ absent | ⬜ absent | resolved (absent) |
| IS_TAKING_CARE_OF_ANOTHER_INVITATION | new | **name** (v83 case22 DecodeStr) | 22 | 22 | ⬜ absent | ⬜ absent | ⬜ absent | resolved (absent) |
| HAVE_DENIED_REQUEST_TO_THE_PARTY | new | **name** (v83 case23 DecodeStr) | 23 | 23 | ⬜ absent | ⬜ absent | ⬜ absent | resolved (absent) |
| CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP | new | mode-only | 25 | 25 | **29** | **29** | **29** | ✅ |
| CHANGE_LEADER | ChangeLeader (exists) | targetId+disc | 27 | 27 | **31** | 31 | **31** | ✅ |
| THIS_CAN_ONLY_BE_GIVEN..VICINITY | new | mode-only | 28 | 28 | **32** | **32** | **32** | ✅ |
| UNABLE_TO_HAND_OVER_THE_LEADERSHIP.. | new | mode-only | 29 | 29 | **33** | **33** | **33** | ✅ |
| YOU_MAY_ONLY_CHANGE..SAME_CHANNEL | new | mode-only | 30 | 30 | **34** | **34** | **34** | ✅ |
| AS_A_GM_YOURE_FORBIDDEN..PARTY | new | mode-only | 32 | 32 | **36** | **36** | **36** | ✅ |
| UNABLE_TO_FIND_THE_CHARACTER | new | mode-only | 33 | 33 | **37** | ⬜ absent | ⬜ absent | resolved (v87 only) |
| TOWN_PORTAL | TownPortal (exists) | slot+maps(+skillId v95)+xy | 37 | 40 | 41 | 46 | 40 | ✅ |

NON-error / non-key arms seen in the IDA switch but with NO Atlas operation-key
(do NOT invent — recorded for completeness): v83 case 0x12 (no-op), case 0x13
(level/job member-update; v95 case 39, jms 0x1F-region), case 0x24 (member-leave
notice, name-or-mode), case 0x45 (expedition invite, sends 0x4D). v95 adds
PQReward arms (cases 40-43), expedition (case 78), case 22 (name notice), case 29.
These are gameplay arms outside the party error/notice key set — out of scope.

#### 10.3a Name-bearing wire-shape verdict (3 invite-target arms)
RE-VERIFIED from the v83 clean switch (0xa3e31c): cases 21/22/23 each begin with
`CInPacket::DecodeStr(v4, &v152)` and then `ZXString<char>::Format(&arg0, m_pStr,
v152)` — the `%s` is filled from the WIRE-decoded string `v152`, not a client-
local. So all three ARE mode+name on the wire (contra guild's mode-only arms).
Decompile citations (v83 0xa3e31c):
  - IS_CURRENTLY_BLOCKING (case 21): `CInPacket::DecodeStr(v4, &v152); ... v21 =
    StringPool::GetString(..., SP_308_...); ZXString<char>::Format(&arg0,
    v21->_m_pStr, v152);`
  - IS_TAKING_CARE (case 22): `CInPacket::DecodeStr(v4, &v152); ... GetString(...,
    SP_2723_...); Format(&arg0, v23->_m_pStr, v152);`
  - HAVE_DENIED (case 23): `CInPacket::DecodeStr(v4, &v152); ... GetString(...,
    SP_309_...); Format(&arg0, v25->_m_pStr, v152);`
Downstream (Task 2): these three need `struct{mode, name}`. All other party
error/notice arms below are mode-only. NOTE: these three are VERSION-ABSENT in
v87/v95/jms, so the mode+name struct only emits for v83/v84.

#### 10.3b StringPool decryptor (reproduced in-process; how the modes were resolved)
Method mirrors task-103 guild v95. The StringPool stream cipher is identical
across versions; the 16-byte `ms_aKey` is the SAME constant in every binary:
`d6de75864664a371e8e67bd33330e72e`.
  - Key location: gms_v87 inline @0xBA43CC (len @0xBA43DC=0x10; the decompiler
    mislabels it as a pointer `dword_BA43CC`=garbage, but the bytes ARE the key);
    gms_v95 `ms_aKey` @0xb98830; jms inline @0xBEC954 (len @0xBEC964=0x10).
  - Per-string blob = `ms_aString[id]`. Seed = first BYTE of the blob (v87/v95/jms
    1-byte-seed layout: `Assign(entry+1, -1)` / `nKeySeed = *ms_aString[id]`);
    cipher = blob[1:] up to the first NUL. (v83's table layout reads a 4-byte seed
    + cipher@+4, but v83 has named SP_* symbols so no decrypt was needed.)
  - Rotate key LEFT by seed: byte-rotate `(seed>>3)%16`, then bit-rotate left
    `seed&7` (rotatel<unsigned char> @v95 0x746270). plaintext[i] = cipher[i] ^
    rotkey[i%16], with the NUL guard (if rotkey[i]==cipher[i], keep rotkey[i]).
  - Validation: decrypts gms_v95 0x143 → "You have created a new party.";
    gms_v87 320 → same; jms decrypts to correct Shift-JIS (e.g. 0x137 → "新しい
    グループを作りました。"). Sanity-checked English/JP before trusting each.

#### 10.3c Resolved upper-arm evidence (decrypted text per arm per version)
All read from each version's OnPartyResult switch; case = the mode byte.

| arm | v87 case→id→text | v95 case→id→text | jms case→id→text |
|---|---|---|---|
| CANNOT_KICK | 29→5070→"Cannot kick another user in this map" | 29→0x13D5→"Cannot kick another user in this map" | 29→0x12CB→"追放機能が制限されたマップです。" |
| CHANGE_LEADER | 31→4054/4055→"%s has become the leader…"/"Due to the party leader disconnecting…" | 31→0xFF7/0xFF8→same | 31→0xFF5/0xFF6→"%s様がグループ長になりました。"/disconnect |
| THIS_CAN_ONLY_BE_GIVEN | 32→4056→"This can only be given to a party member within the vicinity." | 32→0xFF9→same | 32→0xFF7→"同じマップにいるグループ長にのみ譲れます。" |
| UNABLE_TO_HAND_OVER | 33→4058→"Unable to hand over the leadership post; No party member is currently within the vicinity…" | 33→0xFFB→same | 33→0xFF9→"グループ長と同じマップにグループ員がいないため譲れません。" |
| YOU_MAY_ONLY_CHANGE | 34→4057→"You may only change with the party member that's on the same channel." | 34→0xFFA→same | 34→0xFF8→"チャンネルにいるグループ員にのみ譲渡可能です…" |
| AS_A_GM | 36→336→"As a GM, you're forbidden from creating a party." | 36→0x153→same | 36→0x151→"運用者キャラクターはグループを作れません。" |
| UNABLE_TO_FIND_THE_CHARACTER | 37→376→"Unable to find the character." | ABSENT (no case decrypts to it; enumerated @0xa10ab0) | ABSENT (enumerated @0xb297e7) |
| ALREADY_HAVE_JOINED_A_PARTY_2 | 17→329→"Already have joined a party." | 17→0x14C→same | 17→0x142→"既に参加しているグループがあります。" |
| THE_PARTY..FULL_CAPACITY | 18→332→"The party you're trying to join is already in full capacity." | 18→0x14F→same | 18→0x147→"加入しようとしたグループはいっぱいです。" |

#### 10.3d VERSION-ABSENT arms (proven by full case enumeration)
For each, every case of the version's switch was enumerated and decrypted; NONE
yields the arm's v83 text (legitimate per the guild jms SET_SKILL_RESPONSE
precedent — proven, not assumed):
  - UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL (v83 SP_330): v87 case
    19 / v95 case 19 / jms case 0x13 are all an empty `return`; no other case
    decrypts to the text. Absent in v87/v95/jms.
  - IS_CURRENTLY_BLOCKING (SP_308), IS_TAKING_CARE (SP_2723), HAVE_DENIED (SP_309):
    no v87/v95/jms case reads the corresponding name-bearing notice. Absent.
  - UNABLE_TO_FIND_THE_CHARACTER (SP_366): present v87 case 37 (id 376); ABSENT in
    v95 (enumerated @0xa10ab0) and jms (enumerated @0xb297e7).

#### 10.3e CORRECTION to the "byte-identical LOW arms" assumption
The prior table claimed cases ≤17 are byte-identical v83..jms. IDA text DISPROVES
this for two arms: ALREADY_HAVE_JOINED_A_PARTY_2 and FULL_CAPACITY shift +1 in
v87/v95/jms (v83/v84 case 16/17 → v87/v95/jms case 17/18). All arms at cases ≤15
remain identical. party.yaml rows corrected accordingly.

### 10.4 Buddy `BuddyOperation` — key | struct-name | shape | per-version mode (dec) | present
OnFriendResult is BYTE-IDENTICAL across all 5 versions (v95 NOT shifted; it only
ADDS a keyless case 0x17). Every key fully grounded.

| key | struct-name | shape (IDA) | v83 | v84 | v87 | v95 | jms | present |
|---|---|---|---|---|---|---|---|---|
| UPDATE | ListUpdate (exists) | list | 7 | 7 | 7 | 7 | 7 | ✅ |
| BUDDY_UPDATE | Update (exists) | one buddy | 8 | 8 | 8 | 8 | 8 | ✅ |
| INVITE | Invite (exists) | id+name+job+lvl | 9 | 9 | 9 | 9 | 9 | ✅ |
| UNKNOWN_1 | (list-reset, not error) | list (shares case 7/0xA/0x12) | 10 | 10 | 10 | 10 | 10 | ✅ |
| BUDDY_LIST_FULL | new | mode-only (StringPool) | 11 | 11 | 11 | 11 | 11 | ✅ |
| OTHER_BUDDY_LIST_FULL | new | mode-only | 12 | 12 | 12 | 12 | 12 | ✅ |
| ALREADY_BUDDY | new | mode-only | 13 | 13 | 13 | 13 | 13 | ✅ |
| CANNOT_BUDDY_GM | new | mode-only | 14 | 14 | 14 | 14 | 14 | ✅ |
| CHARACTER_NOT_FOUND | new | mode-only | 15 | 15 | 15 | 15 | 15 | ✅ |
| UNKNOWN_ERROR | new | **extra-byte** in GMS (Decode1); **mode-only** in jms | 16 | 16 | 16 | 16 | 16 | ✅ |
| UNKNOWN_ERROR_2 | new | **extra-byte** in GMS (case 0x11); mode-only jms | 17 | 17 | 17 | 17 | 17 | ✅ |
| UNKNOWN_2 | (list-reset, not error) | list (shares case 7/0xA/0x12) | 18 | 18 | 18 | 18 | 18 | ✅ |
| UNKNOWN_ERROR_3 | new | **extra-byte** in GMS (case 0x13); mode-only jms | 19 | 19 | 19 | 19 | 19 | ✅ |
| BUDDY_CHANNEL_CHANGE | ChannelChange (exists) | id+channel | 20 | 20 | 20 | 20 | 20 | ✅ |
| CAPACITY_CHANGE | CapacityUpdate (exists) | capacity | 21 | 21 | 21 | 21 | 21 | ✅ |
| UNKNOWN_ERROR_4 | new | **extra-byte** in GMS (case 0x16); mode-only jms | 22 | 22 | 22 | 22 | 22 | ✅ |

KEY FINDINGS (buddy):
1. The extra trailing byte (`if (CInPacket::Decode1())`) is read by FOUR cases —
   0x10 UNKNOWN_ERROR, 0x11 UNKNOWN_ERROR_2, 0x13 UNKNOWN_ERROR_3, 0x16
   UNKNOWN_ERROR_4 — in gms_v83/v84/v87/v95. The current Go `hasExtra :=
   errorCode == "UNKNOWN_ERROR"` gate (operation_body.go:51) covers only 0x10 and
   is WRONG for the other three. Each gets its own extra-byte struct (design D1).
2. In **jms_v185** those same four cases are MODE-ONLY (no Decode1; straight to
   StringPool 765 + Notice). The extra-byte structs must gate the trailing byte
   GMS-only.
3. UNKNOWN_1 (0x0A) and UNKNOWN_2 (0x12) are NOT errors — they share the case
   7/0xA/0x12 list-reset handler (CFriend::Reset). §4's "tbd extra byte" for them
   resolves to: list-reset shape, no trailing byte.
4. gms_v95 adds a keyless `case 0x17` (StringPool 384, mode-only) — no Atlas key,
   NEEDS_CONTEXT, not invented.

### 10.5 operations --check result (task-105 Task 1)
`go run ./tools/packet-audit operations --check` →
`0 drift, 86 missing, 0 extra` (exit 1). The 86 "missing" are the new v87/v95/jms
party/buddy keys this yaml declares but whose seed tables are still empty — the
EXPECTED `bug_operations_mode_tables_missing_v87_v95_jms` gap, fixed by a later
`packet-audit operations` (generate) task, NOT here. **0 drift / 0 extra** = none
of the authored values CONFLICT with the existing v83/v84 seed entries (no
contradiction). Seed templates were NOT edited in this task.
