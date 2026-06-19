# Guild Dispatcher Family — Complete Implementation & De-baseline — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the guild packet dispatcher family to the canonical discrete-per-mode pattern, drive every supported arm to ✅ across `gms_v83/v84/v87/v95/jms_v185`, remove guild from the dispatcher-lint baseline, and patch live tenant config so the completed family is usable.

**Architecture:** Copy the migrated exemplar (`field/clientbound/mts_operation.go` + `field/mts_operation_body.go` + `docs/packets/dispatchers/mts_operation.yaml`). Split the `ErrorMessage`/`ErrorMessageWithTarget` catch-all structs into one discrete struct per IDA-enumerated mode; replace the caller-selectable `GuildErrorBody`/`GuildErrorBody2` with per-mode fixed-key body functions; route every arm's mode byte through `WithResolvedCode("operations", FIXED_KEY, func(mode byte)…)`; remove the phantom dispatcher roots from `run.go`; fix the serverbound `AgreementResponse` wire bug; migrate the atlas-channel call sites (notably the dynamic-error-code consumer) to the new bodies; verify all five versions with byte fixtures; de-baseline and patch live config.

**Tech Stack:** Go (libs/atlas-packet, tools/packet-audit, services/atlas-channel, services/atlas-guilds), `packet-audit` tooling (dispatcher-lint / matrix / fname-doc / operations checks), ida-pro-mcp (`select_instance` per version), JSON seed templates, k8s/Grafana MCP for the live patch.

---

## Grounding & honesty contract (read before Task 1)

This is a packet-family migration: **the codec/mode bytes are evidence, not guesswork.**

- Every byte, mode value, field width, and per-version presence MUST trace to a
  decompile line (function + address) or a checked-in export entry, cited in the
  struct/test comment. **No values from MapleStory general knowledge or memory.**
- Resolve the IDB by `select_instance(port)` for v83/v87/v95/jms and confirm the
  version matches before reading. v84 has **no IDB** → treat as v83 unless the
  gms_84 template/registry proves a shift (task-100 carryover pattern).
- An unresolved packet-audit fname is a **stop-and-ask** — never auto-re-export,
  substitute an fname, or fake a hash. Surface it and wait.
- Gate version divergence as `>=87`, never `>83`; v84..86 == v83 unless IDA
  proves otherwise.
- No `// TODO` / stubbed handler / 501 in any landed commit.
- All work happens in the task worktree on branch
  `task-103-guild-dispatcher-family`. After every commit run
  `git rev-parse --show-toplevel` (must end `/.worktrees/task-103-guild-dispatcher-family`)
  and `git branch --show-current` (must be `task-103-guild-dispatcher-family`).
- Run all `packet-audit` commands from the worktree root (not via `../../`).

`context.md` holds the grounded current-state map (file:line) and the call-site
inventory — keep it open alongside this plan.

---

## File structure (created / modified)

**libs/atlas-packet/guild/**
- Modify `clientbound/operation.go` — split `ErrorMessage`/`ErrorMessageWithTarget`
  into one discrete struct per enumerated mode; keep existing structural structs.
- Modify `clientbound/info.go` — drop the `0x1A` literal; mode injected via ctor.
- Modify `clientbound/bbs.go` — drop the `0x06`/`0x07` literals; mode injected.
- Modify `operation_body.go` — remove `GuildErrorBody`/`GuildErrorBody2`; add one
  fixed-key body func per enumerated clientbound arm; fold `GuildInfoBody`,
  `RequestGuildNameBody`, `RequestGuildEmblemBody` into `WithResolvedCode`.
- Create `bbs_body.go` — per-mode BBS body funcs (mode resolved, not literal).
- Modify `serverbound/operation_agreement_response.go` — wire fix.
- Modify/create `clientbound/*_test.go`, `serverbound/*_test.go` — per-arm byte
  fixtures with `// packet-audit:verify` markers + IDA citations.

**tools/packet-audit/**
- Modify `cmd/run.go` — one `#`-entry per enumerated mode; remove the three
  phantom roots; freshen comments to current verdicts.

**docs/packets/**
- Create `dispatchers/guild.yaml`, `dispatchers/guild_bbs.yaml` — per-version mode
  tables (source of truth).
- Modify `dispatcher-lint-baseline.yaml` — remove `CWvsContext::OnGuildResult`.
- Regenerate `audits/STATUS.md` + `audits/status.json`.
- Evidence records / synthetic export entries as the verify flow requires.

**services/atlas-channel/atlas.com/channel/**
- Modify `kafka/consumer/guild/consumer.go` — error-code→fixed-key-body dispatch
  map (replaces the dynamic `GuildErrorBody(errCode)`); const-key call sites.
- Modify `kafka/consumer/invite/consumer.go` — `GuildErrorBody2` → fixed-key body.
- Modify `socket/writer/guild_bbs.go` — route through resolved BBS body funcs.
- Verify (no change expected) serverbound handler validators in `main.go`.

**services/atlas-guilds/** — only if a producer must emit a newly-split packet.

**Seed templates** (`services/atlas-configurations/seed-data/templates/template_gms_{83_1,84_1,87_1,95_1,jms_185_1}.json`) — per-version `operations`/opcode/validator entries reconciled with the yamls.

**Live-config runbook** — `docs/tasks/task-103-guild-dispatcher-family/live-config-runbook.md` (created + executed).

---

## Task 0: Baseline — confirm the tree is green before changing anything

**Files:** none (read-only verification).

- [ ] **Step 1: Confirm worktree + branch**

Run (from the task worktree root):
```bash
git rev-parse --show-toplevel    # must end /.worktrees/task-103-guild-dispatcher-family
git branch --show-current        # must be task-103-guild-dispatcher-family
```
Expected: both match.

- [ ] **Step 2: Confirm the four packet-audit gates pass on the untouched tree**

Run from the worktree root:
```bash
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```
Expected: all exit 0 (guild still baselined, so dispatcher-lint passes it as
exempt). Record any pre-existing noise so it is not later mistaken for a
regression.

- [ ] **Step 3: Confirm builds/tests are green**

Run:
```bash
( cd libs/atlas-packet && go build ./... && go vet ./... && go test ./... )
( cd tools/packet-audit && go build ./... && go test ./... )
```
Expected: clean. If not, STOP and report — do not build on a red baseline.

---

## Task 1: Enumerate the switches from IDA; author `guild.yaml` + `guild_bbs.yaml`

This task resolves the open questions (supported-arm set, per-version mode bytes,
version-absent arms, v84 scope, RequestAgreement sharing). **Its output
parameterizes every later per-arm task.** No Atlas code changes here.

**Files:**
- Create: `docs/packets/dispatchers/guild.yaml`
- Create: `docs/packets/dispatchers/guild_bbs.yaml`

- [ ] **Step 1: Resolve and confirm the IDBs**

Use `mcp__ida-pro__list_instances`; `select_instance` each of v83/v87/v95/jms.
Confirm each loaded IDB's version matches before reading. If an instance is
missing, STOP and ask which port to use — do not proceed on a guessed instance.

- [ ] **Step 2: Decompile `CWvsContext::OnGuildResult` per version**

For each of v83, v87, v95, jms: `func_query` (name_regex `OnGuildResult`),
decompile, and record the complete `switch` — every `case <mode>:` with its read
order (`Decode1/Decode2/Decode4/DecodeStr/DecodeBuf`). Capture function address
per version for citations.

For v84: there is no IDB. Compare the gms_84 seed template `operations` table and
the opcode registry against v83; per task-083 v84≡v83 structurally. Record whether
v84 needs (a) operations-table population only, (b) opcode-registry reshift, or
(c) both — **grounded in the template/registry diff, not assumed**.

- [ ] **Step 3: Decompile `CUIGuildBBS::OnGuildBBSPacket` per version**

Same procedure. It dispatches on `(Decode1 - 6)` (run.go:1461): record each arm
(list/view/not-found + any others) and its read order per version.

- [ ] **Step 4: Map each switch case to an operation-key const**

Cross-reference the IDA cases against the 35 existing keys in
`operation_body.go:13-47` and the seed-template `operations` map
(`template_gms_83_1.json:1577`). Produce a table: `key | struct-name |
shape(mode-only / target / structured) | mode byte per version | present?(✅/⬜)`.

- Keys present in the table but **absent** from the IDA switch → flag, do NOT
  invent a struct (design §4.1).
- Switch cases with **no** existing key → stop-and-ask before naming a new key.
- Confirm whether `#RequestAgreement` (run.go:1373) and `#AgreementResponse`
  (run.go:1495) are the same mode or two distinct modes; if two, each needs its
  own struct (design D3).

- [ ] **Step 5: Write `guild.yaml`**

Mirror `docs/packets/dispatchers/mts_operation.yaml` exactly. Header documents the
fname + per-version function addresses. Body:
```yaml
writer: GuildOperation
fname: CWvsContext::OnGuildResult
op: GUILD_OPERATION
direction: clientbound
operations:
  - { key: REQUEST_NAME,       modes: { gms_v83: 1,  gms_v84: 1,  gms_v87: 1,  gms_v95: 1,  jms_v185: <ida> } }
  - { key: REQUEST_AGREEMENT,  modes: { gms_v83: 3,  gms_v84: 3,  gms_v87: 3,  gms_v95: 3,  jms_v185: <ida> } }
  # … one line per IDA-enumerated key, decimal mode bytes, per version,
  #    each value taken from the version's switch (NOT copied across versions
  #    unless that version's IDA shows the same byte). Omit a version key only
  #    when the arm is genuinely version-absent (⬜).
```
Every mode value must be the one read from that version's decompiled switch.

- [ ] **Step 6: Write `guild_bbs.yaml`**

Same format, `writer: GuildBBS` (confirm the actual writer name used by
`bbs.go`/the BBS writer), `fname: CUIGuildBBS::OnGuildBBSPacket`,
`op: GUILD_BBS_PACKET`, `direction: clientbound`, with the BBS arm keys and
per-version `(Decode1 - 6)`-derived mode bytes.

- [ ] **Step 7: Sanity-check against the operations checker**

Run:
```bash
go run ./tools/packet-audit operations --check
```
Expected: still exit 0 (the yamls are additive source-of-truth; if the checker
now reconciles them against the seed table and reports a mismatch, that mismatch
is real input for Task 8/9 — record it, do not paper over it).

- [ ] **Step 8: Commit**

```bash
git add docs/packets/dispatchers/guild.yaml docs/packets/dispatchers/guild_bbs.yaml
git commit -m "task-103: guild/guild_bbs dispatcher mode tables (IDA-enumerated)"
```
Then verify toplevel + branch (see Task 0 Step 1).

**Output artifact:** the key→struct→mode table from Step 4. Append it to
`context.md` under a new "## 10. Enumerated arm table" heading so later tasks
(and reviewers) reference one grounded list.

---

## Task 2: Split the clientbound error/notice catch-alls into discrete structs

Apply the canonical pattern to **each mode-only and target-bearing arm** in the
Task-1 table that is currently fronted by `ErrorMessage` / `ErrorMessageWithTarget`.
Work one arm at a time (TDD), committing per small group.

**Files:**
- Modify: `libs/atlas-packet/guild/clientbound/operation.go`
- Modify: `libs/atlas-packet/guild/clientbound/operation_test.go`

### Worked example — a mode-only error arm (`THE_..._MAX_NUMBER_OF_USERS`)

Repeat this five-step cycle for every mode-only arm in the table
(`THE_NAME_IS_ALREADY_IN_USE`, `SOMEBODY_HAS_DISAGREED`, `THE_PROBLEM_..._FORMING`,
`ALREADY_JOINED`, `MAX_NUMBER_OF_USERS`, `CHARACTER_CANNOT_BE_FOUND`,
`MEMBER_QUIT_ERROR_NOT_IN_GUILD`, `MEMBER_EXPELLED_ERROR_NOT_IN_GUILD`,
`THE_PROBLEM_..._DISBANDING`, `ADMIN_CANNOT_MAKE_A_GUILD`,
`THE_PROBLEM_..._INCREASING`, the two quest errors, `REQUEST_NAME`,
`REQUEST_EMBLEM`, `SHOW_TITLES`, and any other mode-only arm the IDA switch shows).
Use the struct name from the Task-1 table (derived from key semantics, e.g.
`GuildJoinErrorMaxMembers`).

- [ ] **Step 1: Write the failing byte fixture**

In `operation_test.go`, add (struct/version names per the table):
```go
// packet-audit:verify packet=guild/clientbound/GuildJoinErrorMaxMembers version=gms_v83 ida=<addr from Task 1>
func TestGuildJoinErrorMaxMembers_v83(t *testing.T) {
    // Mode-only arm: client reads ONLY the mode byte (IDA OnGuildResult case <n>).
    m := NewGuildJoinErrorMaxMembers(0x29) // mode from guild.yaml gms_v83
    got := pt.Encode(t, m)                 // use the same helper pattern as existing tests
    want := []byte{0x29}
    pt.AssertBytes(t, want, got)
}
```
(Match the exact helper API the existing `operation_test.go` / `info_test.go` use —
read one first; do not invent helper names.)

- [ ] **Step 2: Run it — verify it fails to compile**

Run: `cd libs/atlas-packet && go test ./guild/clientbound/ -run TestGuildJoinErrorMaxMembers_v83`
Expected: FAIL — `NewGuildJoinErrorMaxMembers` undefined.

- [ ] **Step 3: Add the discrete struct**

In `operation.go` (one consolidated file — AP-8), append:
```go
// GuildJoinErrorMaxMembers — mode-only notice (OnGuildResult case <n>, IDA <fn>@<addr>).
// packet-audit:fname CWvsContext::OnGuildResult#GuildJoinErrorMaxMembers
type GuildJoinErrorMaxMembers struct {
    mode byte
}

func NewGuildJoinErrorMaxMembers(mode byte) GuildJoinErrorMaxMembers {
    return GuildJoinErrorMaxMembers{mode: mode}
}

func (m GuildJoinErrorMaxMembers) Operation() string { return GuildOperationWriter }
func (m GuildJoinErrorMaxMembers) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }

func (m GuildJoinErrorMaxMembers) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
    w := response.NewWriter(l)
    return func(options map[string]interface{}) []byte {
        w.WriteByte(m.mode)
        return w.Bytes()
    }
}

func (m *GuildJoinErrorMaxMembers) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
    return func(r *request.Reader, options map[string]interface{}) {
        m.mode = r.ReadByte()
    }
}
```

- [ ] **Step 4: Run the fixture — verify it passes**

Run: `cd libs/atlas-packet && go test ./guild/clientbound/ -run TestGuildJoinErrorMaxMembers_v83`
Expected: PASS.

- [ ] **Step 5: Add the remaining per-version fixtures for this arm**

Add `_v84/_v87/_v95/_jms` test funcs + `packet-audit:verify` markers, each using
that version's mode byte from `guild.yaml` and that version's IDA address. Omit a
version only if the table marks the arm version-absent (⬜). Run the arm's tests:
`go test ./guild/clientbound/ -run TestGuildJoinErrorMaxMembers`. Expected: PASS.

### Target-bearing arms (`ErrorMessageWithTarget` cases — IS_NOT_ACCEPTING, IS_TAKING_CARE, HAS_DENIED)

- [ ] **Step 6: Repeat Steps 1-5 for each target-bearing arm**, but the struct
  carries `mode byte; target string` and Encode writes
  `WriteByte(mode) + WriteAsciiString(target)` (mirror the existing
  `ErrorMessageWithTarget` body, `operation.go:103-109`). Fixture `want` =
  `mode byte` ++ length-prefixed ascii string (match `WriteAsciiString`'s wire
  format — read `response.Writer.WriteAsciiString` to get the exact length prefix
  width).

- [ ] **Step 7: Delete the catch-all structs**

Remove `ErrorMessage` (`operation.go:56-84`) and `ErrorMessageWithTarget`
(`operation.go:86-117`) **only after** every arm they fronted has its own struct +
fixtures. Remove their now-dead fixtures (`GuildErrorMessage` /
`GuildErrorMessageWithTarget` markers in `operation_test.go:19,22,…`).

- [ ] **Step 8: Build + vet the package**

Run: `cd libs/atlas-packet && go build ./... && go vet ./... && go test ./guild/...`
Expected: clean. (Body funcs/run.go still reference the old structs → fix in
Tasks 3-4; if the package won't build standalone yet, keep the catch-all structs
until Task 3 swaps the body funcs, then delete — sequence so each commit builds.)

- [ ] **Step 9: Commit (per arm-group)**

```bash
git add libs/atlas-packet/guild/clientbound/operation.go libs/atlas-packet/guild/clientbound/operation_test.go
git commit -m "task-103: discrete structs for guild error/notice arms (<group>)"
```
Verify toplevel + branch.

---

## Task 3: Per-mode fixed-key body functions; remove the selectors

**Files:**
- Modify: `libs/atlas-packet/guild/operation_body.go`

- [ ] **Step 1: Add one fixed-key body func per error/notice arm**

For each arm split in Task 2 (using its key const + new struct):
```go
func GuildJoinErrorMaxMembersBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", GuildOperationJoinErrorMaxMembers, func(mode byte) packet.Encoder {
        return clientbound.NewGuildJoinErrorMaxMembers(mode)
    })
}
```
Target-bearing arms take only `target string` (NO op/code/key selector):
```go
func GuildInviteDeniedBody(target string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", GuildOperationInviteDenied, func(mode byte) packet.Encoder {
        return clientbound.NewGuildInviteDenied(mode, target)
    })
}
```

- [ ] **Step 2: Rewrite the delegating bodies to fixed-key**

`RequestGuildNameBody` (`:50`) and `RequestGuildEmblemBody` (`:54`) currently call
`GuildErrorBody(...)`. Rewrite each to its own `WithResolvedCode("operations",
GuildOperationRequestName/Emblem, func(mode byte) … NewGuildRequestName(mode))`
form, constructing its discrete struct.

- [ ] **Step 3: Fold `GuildInfoBody` into `WithResolvedCode`**

`GuildInfoBody` (`:146`) bypasses resolution. Change `Info` (`info.go`) so its
constructor takes `mode byte` and Encode writes `m.mode` (not `0x1A`); then:
```go
func GuildInfoBody(inGuild bool, /* …existing args… */) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", GuildOperationInfo, func(mode byte) packet.Encoder {
        return clientbound.NewInfo(mode, inGuild, /* …existing args… */)
    })
}
```
Add the `GuildOperationInfo` key const (value `"GUILD_INFO"` or the existing key
the table uses for sub-op `0x1A` — confirm against the seed table; the Info arm's
key MUST exist in `operations`). If the table has no Info key, add it to the seed
templates + `guild.yaml` in Task 8 and stop-and-note here.

- [ ] **Step 4: Delete `GuildErrorBody` and `GuildErrorBody2`**

Remove both (`:64`, `:70`). The compile will now break at the channel call sites
(Task 5) — that is expected; keep the commits ordered so atlas-packet builds even
if atlas-channel temporarily doesn't, then fix channel in Task 5 in the same
push-set. (If you prefer a single green sweep, do Task 5 immediately after this
step before running the channel build.)

- [ ] **Step 5: Build + vet + test atlas-packet**

Run: `cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./guild/...`
Expected: clean (atlas-packet has no caller of the deleted funcs).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/guild/operation_body.go libs/atlas-packet/guild/clientbound/info.go
git commit -m "task-103: per-mode fixed-key guild body funcs; remove GuildErrorBody selectors"
```
Verify toplevel + branch.

---

## Task 4: BBS clientbound — drop literal modes; resolved body funcs

**Files:**
- Modify: `libs/atlas-packet/guild/clientbound/bbs.go`
- Create: `libs/atlas-packet/guild/bbs_body.go`
- Modify: `libs/atlas-packet/guild/clientbound/bbs_test.go`

- [ ] **Step 1: Make `BBSThreadList`/`BBSThread` take `mode byte`**

Add `mode byte` as the first struct field + first ctor param; Encode writes
`m.mode` instead of the `0x06`/`0x07` literal (`bbs.go:54,167`). Update Decode to
read the mode byte first if the wire includes it (confirm against IDA from Task 1
— the BBS dispatcher consumes `Decode1 - 6`, so the mode byte IS on the wire).

- [ ] **Step 2: Add resolved BBS body funcs in `bbs_body.go`**

```go
package guild

import ( /* atlas_packet, clientbound, packet, context, logrus */ )

const (
    GuildBBSThreadList = "BBS_THREAD_LIST" // key per guild_bbs.yaml
    GuildBBSThread     = "BBS_THREAD"
)

func GuildBBSThreadListBody(/* existing list args */) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", GuildBBSThreadList, func(mode byte) packet.Encoder {
        return clientbound.NewBBSThreadList(mode, /* existing args */)
    })
}
// + GuildBBSThreadBody similarly.
```
(Confirm whether the BBS writer resolves modes from the same `operations` table or
a BBS-specific table; if BBS uses its own table name, use that string in
`WithResolvedCode`. Read the BBS writer registration to confirm.)

- [ ] **Step 3: Update the BBS fixtures** to construct with the resolved mode and
  assert the same bytes (mode now first param). Run:
  `cd libs/atlas-packet && go test ./guild/clientbound/ -run BBS`. Expected: PASS.

- [ ] **Step 4: Build + vet + test, then commit**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./guild/...
git add libs/atlas-packet/guild/clientbound/bbs.go libs/atlas-packet/guild/bbs_body.go libs/atlas-packet/guild/clientbound/bbs_test.go
git commit -m "task-103: guild BBS clientbound config-driven mode resolution"
```
Verify toplevel + branch.

---

## Task 5: Migrate atlas-channel call sites to the per-mode bodies

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/guild/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/invite/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/guild_bbs.go`

- [ ] **Step 1: Build a dynamic-error-code → fixed-key-body dispatch map**

`guild/consumer.go:143` calls `GuildErrorBody(errCode)` with a **dynamic** string
off the Kafka event. Replace with an explicit map from each supported error-code
string (the `GuildOperation*` key consts) to its new per-mode body func:
```go
// guildErrorBodies maps a status-event error code to the discrete, fixed-key
// body func for that guild error arm. A code with no entry is logged and
// dropped (never sent as the wrong mode) — the AP-4 footgun is gone.
var guildErrorBodies = map[string]func() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte{
    guild.GuildOperationJoinErrorMaxMembers:  guild.GuildJoinErrorMaxMembersBody,
    guild.GuildOperationAlreadyJoined:        guild.GuildAlreadyJoinedBody,
    // … one entry per error arm split in Task 2 (mode-only) …
}
```
At the call site:
```go
bodyFn, ok := guildErrorBodies[errCode]
if !ok {
    l.WithField("error_code", errCode).Warn("unmapped guild error code; dropping")
    return
}
// announce via bodyFn() exactly as the old GuildErrorBody(errCode) result was used
```
For the target-bearing error path in `invite/consumer.go:181`
(`GuildErrorBody2(GuildOperationInviteDenied, targetName)`), call the new
`guild.GuildInviteDeniedBody(targetName)` directly (it is a const-key arm — no map
needed).

- [ ] **Step 2: Fix the const-key call site**

`guild/consumer.go:578` calls `GuildErrorBody(GuildOperationCreateError)` — replace
with the new `guild.GuildCreateErrorBody()` fixed-key func directly.

- [ ] **Step 3: Route the BBS writer through the resolved bodies**

`socket/writer/guild_bbs.go:57,74` call `NewBBSThreadList`/`NewBBSThread`
directly. Replace with `guild.GuildBBSThreadListBody(...)` /
`guild.GuildBBSThreadBody(...)` so the mode resolves from config.

- [ ] **Step 4: Verify every serverbound guild handler has a validator**

Read `services/atlas-channel/atlas.com/channel/main.go` guild handler
registrations (Explore found `GuildOperationHandle`/`GuildInviteRejectHandle`/
`GuildBBSHandle` all registered with `LoggedInValidator`). Confirm each handler
entry in the seed templates has a non-empty `validator` field (a missing validator
is silently dropped by `BuildHandlerMap`). Record the confirmation; only edit if a
gap is found.

- [ ] **Step 5: Build + vet + test atlas-channel**

Run:
```bash
( cd services/atlas-channel && go build ./... && go vet ./... && go test -race ./... )
```
Expected: clean. Update any test that referenced the removed funcs in the same
commit.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/guild/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/invite/consumer.go \
        services/atlas-channel/atlas.com/channel/socket/writer/guild_bbs.go
git commit -m "task-103: migrate atlas-channel guild call sites to per-mode bodies"
```
Verify toplevel + branch.

---

## Task 6: Fix the serverbound `AgreementResponse` wire mismatch

**Files:**
- Modify: `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- Modify: `libs/atlas-packet/guild/serverbound/operation_agreement_response_test.go`

- [ ] **Step 1: Decompile `CField::SendCreateGuildAgreeMsg` per version**

Confirm the exact serverbound read order (run.go:1488 records
`Encode1(op) + Encode1(agreed)`; op is the dispatcher prefix). Cite fn+addr per
version. If the read order differs between versions, gate it.

- [ ] **Step 2: Write/adjust the failing round-trip + byte fixture**

```go
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v83 ida=<addr>
func TestAgreementResponse_v83(t *testing.T) {
    raw := []byte{0x01} // agreed=true, single byte (op stripped by dispatcher prefix)
    var m AgreementResponse
    pt.Decode(t, &m, raw)
    if !m.Agreed() { t.Fatalf("agreed not decoded") }
    got := pt.Encode(t, m)
    pt.AssertBytes(t, raw, got) // round-trip identity
}
```

- [ ] **Step 3: Run it — verify it fails against the current 5-byte body**

Run: `cd libs/atlas-packet && go test ./guild/serverbound/ -run TestAgreementResponse`
Expected: FAIL (current Encode writes `WriteInt(unk)` + `WriteBool` = 5 bytes).

- [ ] **Step 4: Correct the codec**

Drop the `unk uint32` field; Encode writes `WriteBool(m.agreed)` only; Decode reads
`m.agreed = r.ReadBool()` only. Remove the `Unk()` accessor and any caller of it
(grep `.Unk()` across services; update or remove). If IDA shows a real leading
field on some version, model that exactly instead — do not assume.

- [ ] **Step 5: Run — verify it passes; build/vet/test**

Run: `cd libs/atlas-packet && go test -race ./guild/serverbound/ && go build ./... && go vet ./...`
Then `( cd services/atlas-channel && go build ./... )` (in case the handler read
`Unk()`). Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/guild/serverbound/operation_agreement_response.go libs/atlas-packet/guild/serverbound/operation_agreement_response_test.go
git commit -m "task-103: fix guild AgreementResponse serverbound wire (drop extra Encode4 unk)"
```
Verify toplevel + branch.

---

## Task 7: Rewire `run.go` — per-mode `#`-entries; remove phantom roots

**Files:**
- Modify: `tools/packet-audit/cmd/run.go`

- [ ] **Step 1: Add one `#`-entry per split error/notice arm**

For each new struct from Task 2, add:
```go
case "CWvsContext::OnGuildResult#GuildJoinErrorMaxMembers":
    // case <n> (IDA <fn>@<addr>): Decode1(mode) only. Atlas GuildJoinErrorMaxMembers writes mode(1). ✓ gms_v83/87/95/jms; v84 == v83.
    return []candidate{{name: "GuildJoinErrorMaxMembers", pkg: "guild", dir: csvpkg.DirClientbound}}
```
Comment MUST reflect the current struct + the per-version verdict (FR-12). Do the
same for the target-bearing arms (→ their `mode + target` structs).

- [ ] **Step 2: Remove the catch-all `#`-entries**

Delete `case "CWvsContext::OnGuildResult#ErrorMessage":` (run.go:1382) and
`#ErrorMessageWithTarget` (run.go:1387) — they map deleted structs.

- [ ] **Step 3: Remove the phantom dispatcher roots**

For `CWvsContext::OnGuildResult` (run.go:1369), `CWvsContext::OnGuildBBSPacket`
(run.go:1462), `CUIFadeYesNo::OnButtonClicked` (run.go:1480): these return a
phantom stand-in "deferred to _pending.md". Per FR-11/INV-4, the bare root must
NOT return a representative. **Confirm against the exemplar** how `OnFieldEffect`
/ `CITC::OnNormalItemResult` handle their bare root in run.go (grep for those
fnames): match that pattern — either remove the case entirely (if the tool needs
no root entry once every arm is `#`-mapped) or return an empty candidate slice as
the exemplar does. Do whichever the exemplar does; do not invent a third option.

- [ ] **Step 4: Freshen the BBS + Info + Invite + CapacityChange comments**

Update the stale-comment cases (run.go:1369-1495) to the current verdict; remove
the `⚠️ OP-FAMILY-…-deferred-to-_pending.md` banners now that arms are enumerated.

- [ ] **Step 5: Confirm `_pending.md` no longer references guild**

Grep `docs/packets/audits/_pending.md` (or wherever the deferral notes live) for
guild OP-FAMILY entries; remove the now-resolved ones.

- [ ] **Step 6: Build + test packet-audit**

Run: `cd tools/packet-audit && go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/cmd/run.go docs/packets/audits/_pending.md
git commit -m "task-103: run.go per-mode guild #-entries; remove phantom roots"
```
Verify toplevel + branch.

---

## Task 8: Serverbound v83/v87 gaps + v84 fold; per-arm fixtures all five versions

**Files:**
- Modify: serverbound `*.go` + `*_test.go` for the rows STATUS.md marks ❌
- Modify: seed templates (operations/opcode tables) for v84 (+ any version gap)

- [ ] **Step 1: For each ❌ serverbound row, decompile and verify the codec**

STATUS.md (rows 831-839) marks v83/v87 ❌ for
`GuildInviteRequest/Join/Kick/RequestCreate/SetMemberTitle/SetNotice/SetTitleNames/Withdraw`.
For each: decompile the matching `CField::Send*` per version, confirm the read
order, and either (a) confirm the existing codec is byte-correct (the ❌ is a stale
matrix cell → re-pin) or (b) fix the codec. Add a per-version byte fixture +
`// packet-audit:verify` marker + IDA citation. **A row reaches ✅ only when its
arm is genuinely verified — never flip a cell without evidence.**

- [ ] **Step 2: Resolve the v84 carryover (per the Task-1 Step-2 finding)**

If v84 needs operations-table population: add the guild keys' v84 mode bytes to
`template_gms_84_1.json` (matching v83 unless the registry proves a shift). If it
needs an opcode-registry reshift: apply it to the gms_84 registry rows for the
guild handler/writer opcodes (task-100 pattern). Add v84 byte fixtures for every
guild arm (clientbound + serverbound + BBS). v84 ≡ v83 bytes unless proven.

- [ ] **Step 3: Run the serverbound tests**

Run: `cd libs/atlas-packet && go test -race ./guild/serverbound/...`
Expected: PASS for every added/edited fixture.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/guild/serverbound services/atlas-configurations/seed-data/templates/template_gms_84_1.json
git commit -m "task-103: guild serverbound v83/v87 verification + v84 reshift fold"
```
Verify toplevel + branch.

---

## Task 9: Reconcile seed templates + regenerate the matrix

**Files:**
- Modify: all five seed templates' guild `operations`/opcode/validator entries
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Reconcile every version's `operations` table against the yamls**

For each of `template_gms_{83_1,84_1,87_1,95_1,jms_185_1}.json`: ensure the
`GuildOperation` (and BBS) `operations` map contains every key in `guild.yaml` /
`guild_bbs.yaml` with the per-version mode byte from the yaml. Add the missing
error/notice/Info keys that the catch-all previously hid. Confirm handler/writer
opcodes + validators are present per version (the v87/v95/jms "operations table
missing → ResolveCode 99 → client crash" trap).

- [ ] **Step 2: Run `operations --check`**

Run: `go run ./tools/packet-audit operations --check`
Expected: exit 0 (yamls ↔ seed tables reconciled).

- [ ] **Step 3: Regenerate the matrix**

Run the matrix regeneration command (read `tools/packet-audit` help / the
VERIFYING_A_PACKET playbook for the exact subcommand — typically
`go run ./tools/packet-audit matrix` to regenerate `STATUS.md`/`status.json` with
the new toolSha stamp).

- [ ] **Step 4: Verify the matrix**

Run: `go run ./tools/packet-audit matrix --check`
Expected: exit 0 — no orphan/dangling/stale/drift, no conflict-count increase.
Confirm every guild + BBS arm is ✅ on all five versions (version-absent → ⬜) and
v84 is cleared.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-103: reconcile guild operations tables (5 versions); regenerate matrix"
```
Verify toplevel + branch.

---

## Task 10: De-baseline + full gate sweep

**Files:**
- Modify: `docs/packets/dispatcher-lint-baseline.yaml`

- [ ] **Step 1: Remove guild from the baseline**

Delete the `- CWvsContext::OnGuildResult` line from `exempt_families`
(`dispatcher-lint-baseline.yaml`). Keep `party` and `buddy`. The baseline only
shrinks.

- [ ] **Step 2: Run dispatcher-lint**

Run: `go run ./tools/packet-audit dispatcher-lint`
Expected: exit 0 — guild now scanned, INV-1..INV-5 all satisfied (no >1-mapped
struct, no `mode: 0x` literal, no `func(_ byte)`, no caller-selector body, every
struct constructed by a body func, every `#`-entry resolves).

If dispatcher-lint flags anything, fix it (it is a real violation — the whole
point of the migration) before continuing.

- [ ] **Step 3: Run all four packet-audit gates**

Run:
```bash
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```
Expected: all exit 0.

- [ ] **Step 4: Full build/vet/test sweep on every changed module**

Run:
```bash
( cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./... )
( cd tools/packet-audit && go build ./... && go vet ./... && go test -race ./... )
( cd services/atlas-channel && go build ./... && go vet ./... && go test -race ./... )
# only if atlas-guilds was touched:
( cd services/atlas-guilds && go build ./... && go vet ./... && go test -race ./... )
```
Expected: all clean.

- [ ] **Step 5: docker buildx bake the touched services**

Run from the worktree root (mandatory — catches missing `COPY libs/...`):
```bash
docker buildx bake atlas-channel
# only if atlas-guilds go.mod changed:
docker buildx bake atlas-guilds
```
Expected: build success.

- [ ] **Step 6: redis-key-guard (only if Redis touched — not expected)**

Run: `tools/redis-key-guard.sh` (from repo root, `GOWORK=off`). Expected: clean.
Skip with a note if no Redis code changed.

- [ ] **Step 7: Commit**

```bash
git add docs/packets/dispatcher-lint-baseline.yaml
git commit -m "task-103: remove guild from dispatcher-lint baseline (family migrated)"
```
Verify toplevel + branch.

---

## Task 11: Live tenant config patch + restart runbook (executed)

**Files:**
- Create: `docs/tasks/task-103-guild-dispatcher-family/live-config-runbook.md`

Seed templates apply only at tenant creation; existing tenants need a live patch.

- [ ] **Step 1: Determine the live tenant/version set**

Use the k8s/Grafana MCP (`mcp__kubernetes__*`) to list running tenants and their
versions (PRD open question 4). Record exactly which versions are live.

- [ ] **Step 2: Author the runbook**

Document, per live tenant/version: the `operations` table entries to PATCH (every
new guild/BBS key → its mode byte), the guild handler/writer opcode entries, and
the serverbound handler validators — exactly as reconciled in Task 9. Mirror the
existing live-config-patch pattern (project memory: "New opcodes missing from live
tenant config" — patch live config + restart channel; projection does not
hot-reload handlers/writers).

- [ ] **Step 3: Execute the patch + restart the affected channels**

Apply the PATCHes; restart the channel pods for the affected tenants.

- [ ] **Step 4: Verify in logs — no unhandled guild op**

Via `mcp__kubernetes__pods_log` / `mcp__grafana__query_loki_logs`, confirm no
`unhandled message op 0xXX` for the guild opcodes post-restart, and that a guild
operation (invite/notice/error) renders. Quote the actual log lines as evidence.

- [ ] **Step 5: Commit the runbook**

```bash
git add docs/tasks/task-103-guild-dispatcher-family/live-config-runbook.md
git commit -m "task-103: live guild config patch runbook (executed + log-verified)"
```
Verify toplevel + branch.

---

## Task 12: Code review + PR

- [ ] **Step 1: Run the modular code review (before PR)**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`
+ `backend-guidelines-reviewer` (Go files changed). Each writes to
`docs/tasks/task-103-guild-dispatcher-family/audit.md`. Address findings via
`superpowers:receiving-code-review` (verify each before implementing).

- [ ] **Step 2: Re-run the full gate sweep after addressing findings**

Repeat Task 10 Steps 3-5 (four packet-audit gates + build/vet/test + bake).
Expected: all green.

- [ ] **Step 3: Open the PR**

PR description mirrors the `DISPATCHER_FAMILY.md` "family complete" checklist
(one discrete struct per mode; full-body Encodes; zero `mode: 0x`/`func(_ byte)`;
no caller-selector; no dangling `#`-entry/orphan; per-mode fixtures+markers; all
four gates exit 0; guild de-baselined; build/vet/test clean; bake clean). Use the
`gh` auth pattern from project memory (`env -u GH_TOKEN -u GITHUB_TOKEN gh …`).

- [ ] **Step 4: Confirm CI green on the actual PR HEAD**

Watch the check job specifically (not just local green). Address any CI-only
failures (e.g. path-nesting, missing `COPY libs/...`) on the same branch.

---

## Self-Review (completed during planning)

**Spec coverage** — every PRD §4 FR and §10 acceptance item maps to a task:
- FR-1/2/3 grounding/IDA/stop-and-ask → grounding contract + Task 1 Steps 1-4, Task 6/8.
- FR-4 enumerate switches → Task 1. FR-5/6/7 discrete structs / full-body Encode /
  no >1-mode → Task 2 (+ catch-all deletion Step 7). FR-8/9/10 config-driven,
  fixed-key, no selector → Task 3 (+ Task 4 BBS). FR-11/12 run.go #-entries /
  freshened comments / no phantom → Task 7. FR-13/14/15 codec correctness /
  version gate / round-trip → Tasks 2/6/8. FR-16 AgreementResponse → Task 6.
  FR-17 fixtures+markers → Tasks 2/4/6/8. FR-18/19/20 usability / real caller /
  validators → Tasks 3/4/5. FR-21/22/23 seed templates / live patch / per-version
  tables → Tasks 9/11. FR-24 no TODO/deferral → grounding contract + per-commit.
- §10 A-H gates → Task 10 (build/test/gates), Task 11 (live), Task 12 (review/PR/CI).

**Placeholder scan** — `<ida>`/`<addr>`/`<n>`/`<group>` markers are deliberate
execution-time IDA values (the plan cannot invent per-version mode bytes without
fabricating evidence — that is the grounding contract, not a placeholder gap).
Each is paired with the exact IDA step that produces it. The repeated per-arm
cycle (Task 2/3) is parameterized over the **grounded** Task-1 key list (the
existing 35 `operation_body.go` consts), not an unknown set.

**Type consistency** — struct names (`GuildJoinErrorMaxMembers`), body-func names
(`GuildJoinErrorMaxMembersBody`), key consts (`GuildOperationJoinErrorMaxMembers`),
and `#`-entry names line up across Tasks 2/3/5/7 (the design D1 naming rule:
struct = body-func stem = key-const stem). `NewGuildJoinErrorMaxMembers(mode byte)`
ctor signature is consistent between Task 2 (definition) and Task 3 (call).
