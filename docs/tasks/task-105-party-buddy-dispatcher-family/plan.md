# Party + Buddy Dispatcher Family — Complete Migration & De-baseline — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the **party** (`CWvsContext::OnPartyResult`) and **buddy** (`CWvsContext::OnFriendResult`) packet dispatcher families to the canonical discrete-per-mode pattern, drive every supported arm to ✅ across `gms_v83/v84/v87/v95/jms_v185`, populate the empty v87/v95/jms operations tables, migrate all atlas-channel call sites, and remove **both** families from the dispatcher-lint baseline (→ empty) — completing the dispatcher-family campaign.

**Architecture:** Copy the executed task-103 guild exemplar (`libs/atlas-packet/guild/clientbound/operation.go` discrete structs + `guild/operation_body.go` fixed-key body funcs + `docs/packets/dispatchers/guild.yaml`). Split each shared `Error` catch-all struct into one discrete struct per IDA-enumerated mode (mode-only or name/extra-bearing, per the decompile); replace the caller-selectable `PartyErrorBody`/`BuddyErrorBody` with per-mode fixed-key body functions; author `party.yaml`/`buddy.yaml`; rewire `run.go` to one `#`-entry per mode; populate v87/v95/jms operations tables from the yamls; migrate the channel call sites (literal → direct body func; runtime error-code → call-site dispatch map); verify all five versions with byte fixtures; de-baseline.

**Tech Stack:** Go (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`), `packet-audit` tooling (dispatcher-lint / matrix / fname-doc / operations checks), ida-pro-mcp (`select_instance` per version), JSON seed templates.

---

## Grounding & honesty contract (read before Task 1)

This is a packet-family migration: **the codec/mode bytes are evidence, not guesswork.** The full
grounded current-state map, enumerated arm tables, and key findings live in `context.md` — keep it
open alongside this plan.

- Every byte, mode value, field width, and per-version presence MUST trace to a decompile line
  (function + address) or a checked-in export entry, cited in the struct/test comment. **No values
  from MapleStory general knowledge or memory.**
- Resolve the IDB by `select_instance(port)` and confirm the version before reading: gms_v83 :13342,
  gms_v84 :13337, gms_v87 :13341, gms_v95 :13340, jms_v185 :13339. v84 **has a live IDB** — read it,
  do not assume v84≡v83. Gate version divergence as `>=87`, never `>83`.
- v95 is the non-uniform shift family: read each v95 arm from the v95 switch + StringPool
  cross-check; never fold from v83.
- An unresolved packet-audit fname is a **stop-and-ask** — never auto-re-export, substitute an
  fname, or fake a hash.
- No `// TODO` / stubbed handler / 501 in any landed commit.
- All work happens in the task worktree on branch `task-105-party-buddy-dispatcher-family`. After
  every commit run `git rev-parse --show-toplevel` (must end
  `/.worktrees/task-105-party-buddy-dispatcher-family`) and `git branch --show-current` (must be
  `task-105-party-buddy-dispatcher-family`).
- Run all `packet-audit` commands from the worktree root.

---

## File structure (created / modified)

**libs/atlas-packet/party/**
- Modify `clientbound/error.go` — split `Error{mode,name}` into one discrete struct per enumerated
  OnPartyResult error/notice arm (mode-only or `{mode,name}` per IDA); delete `Error`/`NewError`
  once every arm is split.
- Modify `clientbound/operation_body.go` — remove `PartyErrorBody`; add one fixed-key body func per
  error arm (+ key consts).
- Modify `clientbound/error_test.go` (and a new consolidated test file if cleaner) — per-arm byte
  fixtures with `// packet-audit:verify` markers + IDA citations; drop the old `Error` fixtures.

**libs/atlas-packet/buddy/**
- Modify `clientbound/error.go` — split `Error{mode,hasExtra}` into one discrete struct per
  enumerated OnFriendResult error/unknown arm (mode-only or `{mode}`+extra-byte per IDA); delete
  `Error`/`NewBuddyError` once every arm is split. Drop the `BuddyErrorWriter` const if unused.
- Modify `operation_body.go` (parent package `buddy`) — remove `BuddyErrorBody`; add one fixed-key
  body func per error arm (+ key consts; reuse the existing `BuddyOperationError*` consts).
- Modify `clientbound/error_test.go` — per-arm byte fixtures + markers + citations.

**tools/packet-audit/**
- Modify `cmd/run.go` — replace `#Error` catch-alls (party run.go:1373, buddy run.go:1130) with one
  `#<Mode>` entry per arm.

**docs/packets/**
- Create `dispatchers/party.yaml`, `dispatchers/buddy.yaml` — per-version mode tables (source of
  truth), complete (error + non-error keys).
- Modify `dispatcher-lint-baseline.yaml` — remove **both** families (→ empty `exempt_families`).
- Regenerate `audits/STATUS.md` + `audits/status.json`.
- Evidence records / synthetic export entries as the verify flow requires.

**services/atlas-configurations/seed-data/templates/** — populate `PartyOperation`/`BuddyOperation`
`operations` maps for `template_gms_{87_1,95_1,jms_185_1}.json` (generated from the yamls); v83/v84
unchanged.

**services/atlas-channel/atlas.com/channel/**
- Modify `socket/handler/party_operation.go` (lines 97, 106) — literal → direct body func.
- Modify `kafka/consumer/invite/consumer.go` (line 171) — literal → direct body func.
- Modify `kafka/consumer/party/consumer.go` (line 452 region) — runtime `errorType` → call-site
  dispatch map.
- Modify `kafka/consumer/buddylist/consumer.go` (line 238 region) — runtime `errorCode` → call-site
  dispatch map.

**docs/tasks/task-105-party-buddy-dispatcher-family/** — `live-config-runbook.md` (RECORDED, NOT
executed — design §9 / PRD §6: live patching is operational and out of scope for this task).

---

## Task 0: Baseline — confirm the tree is green before changing anything

**Files:** none (read-only verification).

- [ ] **Step 1: Confirm worktree + branch**

```bash
git rev-parse --show-toplevel    # must end /.worktrees/task-105-party-buddy-dispatcher-family
git branch --show-current        # must be task-105-party-buddy-dispatcher-family
```
Expected: both match. If not, `cd` into the worktree before continuing.

- [ ] **Step 2: Confirm the four packet-audit gates pass on the untouched tree**

```bash
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```
Expected: all exit 0 (party + buddy still baselined, so dispatcher-lint passes them as exempt).
Record any pre-existing noise so it is not later mistaken for a regression.

- [ ] **Step 3: Confirm builds/tests are green**

```bash
( cd libs/atlas-packet && go build ./... && go vet ./... && go test ./... )
( cd tools/packet-audit && go build ./... && go test ./... )
( cd services/atlas-channel && go build ./... )
```
Expected: clean. If not, STOP and report — do not build on a red baseline.

---

## Task 1: Enumerate both switches from IDA; author `party.yaml` + `buddy.yaml`

Resolves the open questions (supported-arm set, per-version mode bytes, version-absent arms,
per-arm wire shape, buddy `UNKNOWN_ERROR` extra byte). **Its output parameterizes every later
per-arm task.** No Atlas code changes here.

**Files:**
- Create: `docs/packets/dispatchers/party.yaml`
- Create: `docs/packets/dispatchers/buddy.yaml`

- [ ] **Step 1: Resolve and confirm the IDBs**

Use `mcp__ida-pro__list_instances`; `select_instance` each of v83/v84/v87/v95/jms. Confirm each
loaded IDB's version matches before reading. If an instance is missing, STOP and ask which port to
use — do not proceed on a guessed instance.

- [ ] **Step 2: Decompile `CWvsContext::OnPartyResult` per version**

For each of v83, v84, v87, v95, jms: `func_query` (name_regex `OnPartyResult`), decompile, and record
the complete `switch` — every `case <mode>:` with its read order
(`Decode1/Decode2/Decode4/DecodeStr/DecodeBuf`). Capture the function address per version for
citations. For **each error/notice arm** record whether the client reads a trailing name string
(`%s` StringPool) or is mode-only (resolves the design §8 open item). For v95 read each case value
off the v95 switch and cross-check the StringPool message (non-uniform shift).

- [ ] **Step 3: Decompile `CWvsContext::OnFriendResult` per version**

Same procedure. For each error/unknown arm record mode-only vs trailing-extra-int; in particular
confirm **which** mode(s) carry the trailing int (the current `hasExtra` gates it for `UNKNOWN_ERROR`
= 0x10) — model each carrier as its own discrete struct that writes the extra byte (no `hasExtra`
flag).

- [ ] **Step 4: Map each switch case to an operation-key const**

Cross-reference the IDA cases against the v83 seed-template tables (`context.md` §4 — the
authoritative key list) and the existing key consts (`party/clientbound/operation_body.go:13-23`,
`buddy/operation_body.go:13-30`). Produce, per family, a table: `key | struct-name |
shape(mode-only / name / extra-byte) | mode byte per version | present?(✅/⬜)`. Append both tables
to `context.md` under "## 10. Enumerated arm tables (IDA)".

- A key present in the v83 table but **absent** from the IDA switch → flag, do NOT invent a struct.
- A switch case with **no** existing key → stop-and-ask before naming a new key.
- Version-absent arm (no case in a version's switch) → omit that version's mode (matrix ⬜), mirror
  the guild jms `SET_SKILL_RESPONSE` precedent.

- [ ] **Step 5: Write `party.yaml`**

Mirror `docs/packets/dispatchers/guild.yaml` exactly (header documents fname + per-version function
addresses + the v95-shift note + any version-absent arm). Body:
```yaml
writer: PartyOperation
fname: CWvsContext::OnPartyResult
op: PARTY_OPERATION
direction: clientbound
operations:
  - { key: INVITE,        modes: { gms_v83: 4,  gms_v84: <ida>, gms_v87: <ida>, gms_v95: <ida>, jms_v185: <ida> } }
  - { key: UPDATE,        modes: { gms_v83: 7,  gms_v84: <ida>, gms_v87: <ida>, gms_v95: <ida>, jms_v185: <ida> } }
  # … one line per IDA-enumerated key (ALL arms — error AND non-error, per context.md §5),
  #   decimal mode bytes, each value read from THAT version's switch (NOT copied across
  #   versions unless that version's IDA shows the same byte). Omit a version key only when
  #   the arm is genuinely version-absent (⬜).
```
The table MUST be **complete** (every OnPartyResult arm) so Task 7's generate populates the full
v87/v95/jms table (context.md §5). The v83/v84 values must match the existing v83/v84 seed tables.

- [ ] **Step 6: Write `buddy.yaml`**

Same format, `writer: BuddyOperation`, `fname: CWvsContext::OnFriendResult`, `op: BUDDYLIST`,
`direction: clientbound`, with every OnFriendResult arm key and per-version mode bytes. Mark the
`UNKNOWN_ERROR` (and any other extra-byte) arm in a comment.

- [ ] **Step 7: Sanity-check against the operations checker**

```bash
go run ./tools/packet-audit operations --check
```
Expected: still exit 0 (the yamls are additive source-of-truth; if the checker reconciles them
against the v87/v95/jms seed tables and now reports MISSING for the new keys, that is the EXPECTED
`bug_operations_mode_tables_missing_v87_v95_jms` gap — record it; it is fixed in Task 7, not papered
over here).

- [ ] **Step 8: Commit**

```bash
git add docs/packets/dispatchers/party.yaml docs/packets/dispatchers/buddy.yaml docs/tasks/task-105-party-buddy-dispatcher-family/context.md
git commit -m "task-105: party/buddy dispatcher mode tables (IDA-enumerated, 5 versions)"
```
Verify toplevel + branch.

---

## Task 2: Party — split the `Error` catch-all into discrete structs

Apply the canonical pattern to **each** mode-only and name-bearing party error/notice arm in the
Task-1 table. Work one arm at a time (TDD). The struct name derives from the key semantics
(design D1), e.g. `UnableToFindCharacter`, `RequestDenied`, `AlreadyJoined1`, `BeginnerCannotCreate`.

**Files:**
- Modify: `libs/atlas-packet/party/clientbound/error.go`
- Modify: `libs/atlas-packet/party/clientbound/error_test.go`

### Worked example — a mode-only error arm (`A_BEGINNER_CANT_CREATE_A_PARTY`, v83 mode 0x0A)

Repeat this cycle for every mode-only arm in the Task-1 table. (First read
`party/clientbound/created_test.go` to copy the exact byte-fixture helper API — do NOT invent helper
names.)

- [ ] **Step 1: Write the failing byte fixture**

In `error_test.go` (helper names per the existing tests):
```go
// packet-audit:verify packet=party/clientbound/BeginnerCannotCreate version=gms_v83 ida=CWvsContext::OnPartyResult@<addr from Task 1>
func TestBeginnerCannotCreate_v83(t *testing.T) {
    // Mode-only arm: client reads ONLY the mode byte (OnPartyResult case 0x0A, no DecodeStr).
    m := NewBeginnerCannotCreate(0x0A) // mode from party.yaml gms_v83
    got := <encode-helper>(t, m)
    want := []byte{0x0A}
    <assert-bytes-helper>(t, want, got)
}
```

- [ ] **Step 2: Run it — verify it fails to compile**

```bash
cd libs/atlas-packet && go test ./party/clientbound/ -run TestBeginnerCannotCreate_v83
```
Expected: FAIL — `NewBeginnerCannotCreate` undefined.

- [ ] **Step 3: Add the discrete struct**

In `error.go` (one consolidated file — AP-8), append (mirror guild `CreateErrorNameInUse`):
```go
// BeginnerCannotCreate — A_BEGINNER_CANT_CREATE_A_PARTY (case 0x0A). Mode-only
// StringPool notice (OnPartyResult@<addr>, case 0x0A → notice, no Decode* after mode).
// packet-audit:fname CWvsContext::OnPartyResult#BeginnerCannotCreate
type BeginnerCannotCreate struct{ mode byte }

func NewBeginnerCannotCreate(mode byte) BeginnerCannotCreate { return BeginnerCannotCreate{mode: mode} }
func (m BeginnerCannotCreate) Operation() string { return PartyOperationWriter }
func (m BeginnerCannotCreate) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m BeginnerCannotCreate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
    w := response.NewWriter(l)
    return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *BeginnerCannotCreate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
    return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}
```
(`PartyOperationWriter` is the existing writer const — confirm its definition; reuse it.)

- [ ] **Step 4: Run the fixture — verify it passes**

```bash
cd libs/atlas-packet && go test ./party/clientbound/ -run TestBeginnerCannotCreate_v83
```
Expected: PASS.

- [ ] **Step 5: Add the remaining per-version fixtures for this arm**

Add `_v84/_v87/_v95/_jms` test funcs + `packet-audit:verify` markers, each using that version's mode
byte from `party.yaml` and that version's IDA address. Omit a version only if the table marks the arm
version-absent (⬜). Run `go test ./party/clientbound/ -run TestBeginnerCannotCreate`. Expected: PASS.

### Name-bearing error arms (the arms IDA shows reading a trailing `DecodeStr`)

- [ ] **Step 6: Repeat Steps 1-5 for each name-bearing arm**, but the struct carries
  `mode byte; name string` and `Encode` writes `WriteByte(mode) + WriteAsciiString(name)` (mirror
  guild `InviteDenied`, `operation.go:414-439`). Per context.md §4 these are the
  character/invite-target arms — confirm the exact set from IDA (`UNABLE_TO_FIND_THE_CHARACTER`,
  `…IN_THIS_CHANNEL`, `HAVE_DENIED_REQUEST_TO_THE_PARTY`, the two `IS_*` invite arms, and any other
  the switch shows reading `DecodeStr`). Fixture `want` = `mode byte` ++ `WriteAsciiString(name)`
  wire bytes (read `response.Writer.WriteAsciiString` for the exact length-prefix width).

  **Regression guard (NFR / D8):** for the three arms Atlas currently emits via a literal call site
  (`UNABLE_TO_FIND_THE_CHARACTER`, `…IN_THIS_CHANNEL`, `HAVE_DENIED_REQUEST_TO_THE_PARTY`), add a
  byte-comparison test asserting the new discrete-struct output equals the old `Error{mode,name}`
  output for the same `(mode, name)`. If IDA shows an arm the current code wrote a name for is
  actually **mode-only**, IDA wins (mode-only struct) and the regression test is scoped to the bytes
  the client consumes — document the divergence in the struct comment; do not preserve a latent
  over-write.

- [ ] **Step 7: Delete the catch-all struct**

Remove `Error`/`NewError` (`error.go:13-47`) and its `// packet-audit:fname …#Error` marker **only
after** every arm it fronted has its own struct + fixtures. Remove the old `Error` fixtures from
`error_test.go`.

- [ ] **Step 8: Build + vet the package**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test ./party/...
```
Expected: clean if `operation_body.go` no longer references `Error` — but `PartyErrorBody` still does
until Task 3. Sequence: keep `Error` until Task 3 swaps the body func, OR do Task 3 immediately after
this step before the standalone build. (Mirror guild plan Task 2 Step 8.)

- [ ] **Step 9: Commit (per arm-group)**

```bash
git add libs/atlas-packet/party/clientbound/error.go libs/atlas-packet/party/clientbound/error_test.go
git commit -m "task-105: discrete structs for party error/notice arms (<group>)"
```
Verify toplevel + branch.

---

## Task 3: Party — per-mode fixed-key body functions; remove `PartyErrorBody`

**Files:**
- Modify: `libs/atlas-packet/party/clientbound/operation_body.go`

- [ ] **Step 1: Add key consts for the error arms**

In `operation_body.go`, extend the const block (`:13-23`) with one const per error arm, value = the
operations key string (from context.md §4), e.g.:
```go
const (
    // … existing PartyOperationCreated … PartyOperationTownPortal …
    PartyOperationUnableToFindCharacter   = "UNABLE_TO_FIND_THE_CHARACTER"
    PartyOperationUnableToFindInChannel   = "UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL"
    PartyOperationRequestDenied           = "HAVE_DENIED_REQUEST_TO_THE_PARTY"
    PartyOperationBeginnerCannotCreate    = "A_BEGINNER_CANT_CREATE_A_PARTY"
    // … one per error arm in the Task-1 table …
)
```

- [ ] **Step 2: Add one fixed-key body func per error arm**

Mode-only arm (mirror guild `GuildCreateErrorNameInUseBody`):
```go
func PartyBeginnerCannotCreateBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", PartyOperationBeginnerCannotCreate, func(mode byte) packet.Encoder {
        return NewBeginnerCannotCreate(mode)
    })
}
```
Name-bearing arm (mirror guild `GuildInviteDeniedBody`):
```go
func PartyUnableToFindCharacterBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", PartyOperationUnableToFindCharacter, func(mode byte) packet.Encoder {
        return NewUnableToFindCharacter(mode, name)
    })
}
```
Every error arm gets a body func (FR-3.4: future-feature arms too — every discrete struct is
constructed by ≥1 body func, INV-5). No body func takes an `op/code/mode/key` selector or any param
that flows into the `WithResolvedCode` key (INV-3).

- [ ] **Step 3: Delete `PartyErrorBody`**

Remove `PartyErrorBody` (`:78-82`). The compile now breaks at the channel call sites (Task 8) — that
is expected; keep `atlas-packet` building (it has no internal caller of `PartyErrorBody`).

- [ ] **Step 4: Build + vet + test atlas-packet**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./party/...
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/party/clientbound/operation_body.go
git commit -m "task-105: per-mode fixed-key party body funcs; remove PartyErrorBody selector"
```
Verify toplevel + branch.

---

## Task 4: Buddy — split the `Error` catch-all into discrete structs

Same canonical pattern for **each** OnFriendResult error/unknown arm (Task-1 table). Struct names
derive from the existing key consts, e.g. `ListFull`, `OtherListFull`, `AlreadyBuddy`,
`CannotBuddyGm`, `CharacterNotFound`, `UnknownError`, `UnknownError2/3/4`, `Unknown1`.

**Files:**
- Modify: `libs/atlas-packet/buddy/clientbound/error.go`
- Modify: `libs/atlas-packet/buddy/clientbound/error_test.go`

### Worked example — a mode-only error arm (`BUDDY_LIST_FULL`, v83 mode 0x0B)

(First read `buddy/clientbound/update_test.go` to copy the byte-fixture helper API.)

- [ ] **Step 1: Write the failing byte fixture**

```go
// packet-audit:verify packet=buddy/clientbound/ListFull version=gms_v83 ida=CWvsContext::OnFriendResult@<addr from Task 1>
func TestBuddyListFull_v83(t *testing.T) {
    // Mode-only arm: client reads ONLY the mode byte (OnFriendResult case 0x0B, StringPool notice).
    m := NewListFull(0x0B) // mode from buddy.yaml gms_v83
    got := <encode-helper>(t, m)
    want := []byte{0x0B}
    <assert-bytes-helper>(t, want, got)
}
```

- [ ] **Step 2: Run it — verify it fails to compile**

```bash
cd libs/atlas-packet && go test ./buddy/clientbound/ -run TestBuddyListFull_v83
```
Expected: FAIL — `NewListFull` undefined.

- [ ] **Step 3: Add the discrete struct**

```go
// ListFull — BUDDY_LIST_FULL (case 0x0B). Mode-only StringPool notice
// (OnFriendResult@<addr>, case 0x0B → notice, no Decode* after mode).
// packet-audit:fname CWvsContext::OnFriendResult#ListFull
type ListFull struct{ mode byte }

func NewListFull(mode byte) ListFull { return ListFull{mode: mode} }
func (m ListFull) Operation() string { return BuddyOperationWriter } // confirm writer const name
func (m ListFull) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m ListFull) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
    w := response.NewWriter(l)
    return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *ListFull) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
    return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}
```
(Confirm the writer-const name the buddy structs use for `Operation()` — the current `Error` uses a
local `BuddyErrorWriter = "BuddyError"`; the non-error buddy structs and the channel call site use
`BuddyOperationWriter`. Use the **same** writer const the non-error arms use so all arms share one
writer identity; drop the now-orphan `BuddyErrorWriter` if nothing else references it.)

- [ ] **Step 4: Run the fixture — verify it passes**

```bash
cd libs/atlas-packet && go test ./buddy/clientbound/ -run TestBuddyListFull_v83
```
Expected: PASS.

- [ ] **Step 5: Add the remaining per-version fixtures for this arm**

Add `_v84/_v87/_v95/_jms` fixtures + markers from `buddy.yaml`. Run
`go test ./buddy/clientbound/ -run TestBuddyListFull`. Expected: PASS.

### The extra-byte arm(s) (`UNKNOWN_ERROR` = 0x10, and any other IDA shows reading a trailing int)

- [ ] **Step 6: Repeat Steps 1-5 for the extra-byte arm(s)**, with a struct whose `Encode` writes
  `WriteByte(mode)` then the trailing byte exactly as the current `hasExtra` path does
  (`error.go:36-38` writes `WriteByte(0)`). NO `hasExtra` flag — the arm identity is the struct
  (design D1). Fixture `want` = `[]byte{0x10, 0x00}`. Cite the IDA case that reads the extra int.

- [ ] **Step 7: Delete the catch-all struct**

Remove `Error`/`NewBuddyError` (`error.go:15-50`) + its `#Error` marker only after every arm is
split. Remove old `Error` fixtures. Remove `BuddyErrorWriter` if unused.

- [ ] **Step 8: Build + vet the package**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test ./buddy/...
```
Expected: clean (sequence with Task 5 like Task 2/3 — `BuddyErrorBody` still references nothing in
`clientbound` since it lives in the parent `buddy` package; keep ordering so each commit builds).

- [ ] **Step 9: Commit (per arm-group)**

```bash
git add libs/atlas-packet/buddy/clientbound/error.go libs/atlas-packet/buddy/clientbound/error_test.go
git commit -m "task-105: discrete structs for buddy error/unknown arms (<group>)"
```
Verify toplevel + branch.

---

## Task 5: Buddy — per-mode fixed-key body functions; remove `BuddyErrorBody`

**Files:**
- Modify: `libs/atlas-packet/buddy/operation_body.go`

- [ ] **Step 1: Add one fixed-key body func per error arm**

Reuse the existing `BuddyOperationError*` / `BuddyOperationUnknown*` key consts (`:13-30`). Mode-only:
```go
func BuddyListFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorListFull, func(mode byte) packet.Encoder {
        return clientbound.NewListFull(mode)
    })
}
```
Extra-byte arm:
```go
func BuddyUnknownErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorUnknownError, func(mode byte) packet.Encoder {
        return clientbound.NewUnknownError(mode)
    })
}
```
One body func per arm (incl. the future-feature `UNKNOWN_1/UNKNOWN_ERROR_2/UNKNOWN_2/UNKNOWN_ERROR_3/
UNKNOWN_ERROR_4` arms — INV-5). No selector param (INV-3).

- [ ] **Step 2: Delete `BuddyErrorBody`**

Remove `BuddyErrorBody` (`:50-55`) and its `hasExtra := errorCode == …` line. Compile breaks at the
channel call site (Task 8) — expected.

- [ ] **Step 3: Build + vet + test atlas-packet**

```bash
cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./buddy/...
```
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/buddy/operation_body.go
git commit -m "task-105: per-mode fixed-key buddy body funcs; remove BuddyErrorBody selector"
```
Verify toplevel + branch.

---

## Task 6: Rewire `run.go` — per-mode `#`-entries; remove both `#Error` catch-alls

**Files:**
- Modify: `tools/packet-audit/cmd/run.go`

- [ ] **Step 1: Add one party `#`-entry per split arm**

Replace `case "CWvsContext::OnPartyResult#Error":` (run.go:1373) with one case per arm (mirror guild
run.go:1479-1540):
```go
case "CWvsContext::OnPartyResult#BeginnerCannotCreate":
    // case 0x0A (IDA OnPartyResult@<addr>): Decode1(mode) only. Atlas BeginnerCannotCreate writes mode(1). ✓ v83/84/87/95/jms.
    return []candidate{{name: "BeginnerCannotCreate", pkg: "party", dir: csvpkg.DirClientbound}}
// … name-bearing arm:
case "CWvsContext::OnPartyResult#UnableToFindCharacter":
    // case 0x21: Decode1(mode) + DecodeStr(name). Atlas writes mode(1) + name(str). ✓
    return []candidate{{name: "UnableToFindCharacter", pkg: "party", dir: csvpkg.DirClientbound}}
// … one per error arm …
```
Each comment reflects the current struct + per-version verdict.

- [ ] **Step 2: Add one buddy `#`-entry per split arm**

Replace `case "CWvsContext::OnFriendResult#Error":` (run.go:1130) with one case per buddy arm:
```go
case "CWvsContext::OnFriendResult#ListFull":
    // case 0x0B (IDA OnFriendResult@<addr>): Decode1(mode) only. Atlas ListFull writes mode(1). ✓
    return []candidate{{name: "ListFull", pkg: "buddy", dir: csvpkg.DirClientbound}}
case "CWvsContext::OnFriendResult#UnknownError":
    // case 0x10: Decode1(mode) + Decode1(extra). Atlas UnknownError writes mode(1) + 0x00(1). ✓
    return []candidate{{name: "UnknownError", pkg: "buddy", dir: csvpkg.DirClientbound}}
// … one per buddy error arm …
```

- [ ] **Step 3: Confirm no `#`-entry points at a deleted struct**

Grep run.go for `pkg: "party"` / `pkg: "buddy"` clientbound entries; every `name:` must resolve to a
`type <name> struct` now in `error.go` (INV-4). No `Error` representative remains for either family.

- [ ] **Step 4: Confirm `_pending.md` no longer references party/buddy Error**

Grep `docs/packets/audits/_pending.md` (or wherever the deferral notes live) for the party/buddy
`#Error` "deferred to _pending.md" entries; remove the now-resolved ones.

- [ ] **Step 5: Build + test packet-audit**

```bash
cd tools/packet-audit && go build ./... && go test ./...
```
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/run.go docs/packets/audits/_pending.md
git commit -m "task-105: run.go per-mode party/buddy #-entries; remove both #Error catch-alls"
```
Verify toplevel + branch.

---

## Task 7: Populate the v87/v95/jms operations tables; reconcile + regenerate matrix

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{87_1,95_1,jms_185_1}.json`
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Generate the operations tables from the yamls**

Per context.md §5, the v87/v95/jms `PartyOperation`/`BuddyOperation` tables are empty/near-empty.
Run the `packet-audit operations` generate command (read `go run ./tools/packet-audit operations
--help` for the exact generate invocation — guild task-103 Task 9 used it to populate the empty
tables) to write the full per-version tables from `party.yaml`/`buddy.yaml` into the v87/v95/jms
templates. v83/v84 already match and stay unchanged.

- [ ] **Step 2: Run `operations --check`**

```bash
go run ./tools/packet-audit operations --check
```
Expected: exit 0 for both families across all applicable versions (yamls ↔ seed tables reconciled).
If a key is still MISSING, the generate did not cover it — fix the yaml/template, do not hand-edit a
single key in isolation.

- [ ] **Step 3: Regenerate the matrix**

Run the matrix regeneration command (`go run ./tools/packet-audit matrix` — confirm via
`--help` / VERIFYING_A_PACKET.md) to refresh `STATUS.md`/`status.json` with the new toolSha stamp and
the per-arm `#`-entries.

- [ ] **Step 4: Verify the matrix**

```bash
go run ./tools/packet-audit matrix --check
```
Expected: exit 0 — no orphan/dangling/stale/drift, no conflict-count increase. Confirm the
`PARTY_OPERATION` and `BUDDYLIST` op-rows are ✅ on every applicable version (version-absent → ⬜),
aggregating worst-of all arms (FIELD_EFFECT model; neither family added to `families.yaml`).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-105: populate v87/v95/jms party/buddy operations tables; regenerate matrix"
```
Verify toplevel + branch.

---

## Task 8: Migrate atlas-channel call sites to the per-mode bodies

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/party_operation.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/invite/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buddylist/consumer.go`

- [ ] **Step 1: Migrate the three literal party call sites (direct body func)**

- `party_operation.go:97` `PartyErrorBody("UNABLE_TO_FIND_THE_CHARACTER", sp.Name())` →
  `partycb.PartyUnableToFindCharacterBody(sp.Name())`.
- `party_operation.go:106`
  `PartyErrorBody("UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL", sp.Name())` →
  `partycb.PartyUnableToFindInChannelBody(sp.Name())`.
- `invite/consumer.go:171` `PartyErrorBody("HAVE_DENIED_REQUEST_TO_THE_PARTY", targetName)` →
  `partycb.PartyRequestDeniedBody(targetName)`.

- [ ] **Step 2: Migrate the runtime party `errorType` call site (call-site dispatch map)**

In `kafka/consumer/party/consumer.go` (the `partyError` func, ~:447-456), replace
`PartyErrorBody(errorType, name)` with a map from each `atlas-parties`
`EventPartyStatusErrorType*` string (context.md §4) to its body func, with a logged default
(mirror guild plan Task 5 Step 1):
```go
// partyErrorBodies maps a status-event error type to the discrete fixed-key body func for that
// party error arm. A type with no entry is logged and dropped (never sent as the wrong mode).
// Name-bearing arms take `name`; mode-only arms ignore it.
func partyErrorBody(errorType string, name string) (func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, bool) {
    switch errorType {
    case party2.EventPartyStatusErrorTypeCannotFindCharacter:
        return partycb.PartyUnableToFindCharacterBody(name), true
    case party2.EventPartyStatusErrorTypeInviteDenied:
        return partycb.PartyRequestDeniedBody(name), true
    case party2.EventPartyStatusErrorTypeAlreadyJoined1:
        return partycb.PartyAlreadyJoined1Body(), true
    // … one case per EventPartyStatusErrorType* (context.md §4) → its body func …
    default:
        return nil, false
    }
}
```
At the call site, look up; on `!ok` log
`l.WithField("error_type", errorType).Warn("unmapped party error type; dropping")` and return a
no-op operator. `ERROR_UNEXPECTED` (no operations key) falls through to the logged default — do NOT
invent a mode for it.

- [ ] **Step 3: Migrate the runtime buddy `errorCode` call site (call-site dispatch map)**

In `kafka/consumer/buddylist/consumer.go` (the `buddyError` func, ~:234-242), replace
`BuddyErrorBody(errorCode)` with a switch over the 6 `StatusEventError*` strings (context.md §4) →
their body funcs, logged default:
```go
func buddyErrorBody(errorCode string) (func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, bool) {
    switch errorCode {
    case buddylist2.StatusEventErrorListFull:        return buddypkt.BuddyListFullBody(), true
    case buddylist2.StatusEventErrorOtherListFull:   return buddypkt.BuddyOtherListFullBody(), true
    case buddylist2.StatusEventErrorAlreadyBuddy:    return buddypkt.BuddyAlreadyBuddyBody(), true
    case buddylist2.StatusEventErrorCannotBuddyGm:   return buddypkt.BuddyCannotBuddyGmBody(), true
    case buddylist2.StatusEventErrorCharacterNotFound: return buddypkt.BuddyCharacterNotFoundBody(), true
    case buddylist2.StatusEventErrorUnknownError:    return buddypkt.BuddyUnknownErrorBody(), true
    default:
        return nil, false
    }
}
```
On `!ok` log + drop (keep the existing `l.WithError…Errorf` shape adapted to the unmapped case).

- [ ] **Step 4: Confirm no string error selector remains (FR-6.3)**

```bash
grep -rn 'PartyErrorBody\|BuddyErrorBody' services/ libs/
```
Expected: zero hits.

- [ ] **Step 5: Build + vet + test atlas-channel**

```bash
( cd services/atlas-channel && go build ./... && go vet ./... && go test -race ./... )
```
Expected: clean. Update any test referencing the removed funcs in the same commit.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/party_operation.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/invite/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/buddylist/consumer.go
git commit -m "task-105: migrate atlas-channel party/buddy call sites to per-mode bodies"
```
Verify toplevel + branch.

---

## Task 9: De-baseline + full gate sweep

**Files:**
- Modify: `docs/packets/dispatcher-lint-baseline.yaml`

- [ ] **Step 1: Remove both families from the baseline**

Delete `- CWvsContext::OnPartyResult` and `- CWvsContext::OnFriendResult` from `exempt_families`.
The list is now **empty** (`exempt_families: []`). Update the explanatory comment to record that the
campaign is complete (the baseline only ever shrank to empty).

- [ ] **Step 2: Run dispatcher-lint**

```bash
go run ./tools/packet-audit dispatcher-lint
```
Expected: exit 0 — both families now scanned, INV-1..INV-5 all satisfied (no >1-mapped struct, no
`mode: 0x` literal, no `func(_ byte)`, no caller-selector body, every struct constructed by a body
func, every `#`-entry resolves). Any flag is a real violation — fix it before continuing.

- [ ] **Step 3: Run all four packet-audit gates**

```bash
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```
Expected: all exit 0.

- [ ] **Step 4: Full build/vet/test sweep on every changed module**

```bash
( cd libs/atlas-packet     && go build ./... && go vet ./... && go test -race ./... )
( cd tools/packet-audit    && go build ./... && go vet ./... && go test -race ./... )
( cd services/atlas-channel && go build ./... && go vet ./... && go test -race ./... )
```
Expected: all clean.

- [ ] **Step 5: docker buildx bake the touched service**

```bash
docker buildx bake atlas-channel
```
Expected: build success. (`libs/atlas-packet` / `tools/packet-audit` are not bake targets — confirm
no other service `go.mod` was touched via `git diff --name-only main -- '**/go.mod'`.)

- [ ] **Step 6: redis-key-guard (no Redis change expected)**

```bash
GOWORK=off tools/redis-key-guard.sh
```
Expected: clean. Skip with a note if no Redis code changed.

- [ ] **Step 7: Record the live-config runbook (NOT executed)**

Create `docs/tasks/task-105-party-buddy-dispatcher-family/live-config-runbook.md` documenting, per
live v87/v95/jms tenant, the `PartyOperation`/`BuddyOperation` `operations` entries to PATCH + the
channel restart needed for the newly-split arms to resolve (mirror the guild runbook content). Per
design §9 / PRD §6 this is **recorded, not executed** — live patching is operational and out of this
task's scope.

- [ ] **Step 8: Commit**

```bash
git add docs/packets/dispatcher-lint-baseline.yaml docs/tasks/task-105-party-buddy-dispatcher-family/live-config-runbook.md
git commit -m "task-105: remove party+buddy from dispatcher-lint baseline (campaign complete)"
```
Verify toplevel + branch.

---

## Task 10: Code review + PR

- [ ] **Step 1: Run the modular code review (before PR)**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` +
`backend-guidelines-reviewer` (Go files changed). Each writes to
`docs/tasks/task-105-party-buddy-dispatcher-family/audit.md`. Address findings via
`superpowers:receiving-code-review` (verify each before implementing).

- [ ] **Step 2: Re-run the full gate sweep after addressing findings**

Repeat Task 9 Steps 3-5 (four packet-audit gates + build/vet/test + bake). Expected: all green.

- [ ] **Step 3: Open the PR**

PR description mirrors the `DISPATCHER_FAMILY.md` "family complete" checklist for BOTH families (one
discrete struct per mode; full-body Encodes; zero `mode: 0x`/`func(_ byte)`; no caller-selector; no
dangling `#`-entry/orphan; per-mode fixtures+markers; all four gates exit 0; party+buddy
de-baselined → **empty baseline, campaign complete**; build/vet/test clean; bake clean). Use the `gh`
auth pattern from project memory (`env -u GH_TOKEN -u GITHUB_TOKEN gh …`).

- [ ] **Step 4: Confirm CI green on the actual PR HEAD**

Watch the check job specifically (not just local green). Address any CI-only failures (e.g.
path-nesting, missing `COPY libs/...`) on the same branch.

---

## Self-Review (completed during planning)

**Spec coverage** — every PRD §4 FR and §10 acceptance item maps to a task:
- FR-1 (IDA enumeration, all 5 versions, v95 shift, version-absent, citations) → Task 1 Steps 1-4
  + grounding contract.
- FR-2 (discrete structs, full client-switch enumeration, full-body Encode, delete shared `Error`) →
  Task 2 (party) + Task 4 (buddy), Step 7 deletions.
- FR-3 (fixed-key body funcs, no selector, `mode byte` ctor, every struct has a body func) → Task 3
  (party) + Task 5 (buddy).
- FR-4 (run.go per-mode `#`-entries, per-arm export/report/fixture/evidence, worst-of op-row ✅) →
  Task 6 (`#`-entries) + Tasks 2/4 (fixtures+markers) + Task 7 (matrix).
- FR-5 (author party.yaml/buddy.yaml, generate operations tables, `operations --check` 0) → Task 1
  (yamls) + Task 7 (generate/check).
- FR-6 (migrate all call sites; runtime → call-site map; no string selector remains) → Task 8.
- FR-7 (remove both from baseline; dispatcher-lint 0 no suppressed notes) → Task 9.
- §8 NFR (determinism byte-comparison; CI gates; multi-tenancy; no hard-coded bytes) → Task 2 Step 6
  regression guard + Task 9 gate sweep + the `WithResolvedCode` pattern throughout.
- §10 acceptance A-H → Tasks 1/2/4 (yamls, structs, fixtures), Task 3/5 (body funcs), Task 6
  (run.go), Task 7 (operations + matrix), Task 8 (call sites), Task 9 (de-baseline + gates), Task 10
  (review/PR/CI).

**Placeholder scan** — `<addr>`/`<group>`/`<encode-helper>`/`<assert-bytes-helper>` are deliberate
execution-time values: `<addr>` is the per-version IDA function address (inventing it would fabricate
evidence — the grounding contract, not a placeholder gap), each paired with the exact IDA step
(Task 1) that produces it; the helper placeholders are resolved by reading one existing `*_test.go`
in the package first (instructed in Tasks 2/4) because the plan must not invent helper names
(reference_atlas_ui… analog: copy the real API). The repeated per-arm cycle is parameterized over the
**grounded** Task-1 key list (the existing v83 seed-table keys in context.md §4), not an unknown set.

**Type consistency** — struct names (`BeginnerCannotCreate`, `UnableToFindCharacter`, `ListFull`,
`UnknownError`), body-func names (`PartyBeginnerCannotCreateBody`, `PartyUnableToFindCharacterBody`,
`BuddyListFullBody`, `BuddyUnknownErrorBody`), key consts
(`PartyOperationBeginnerCannotCreate`/`BuddyOperationErrorListFull`), and `#`-entry names line up
across Tasks 2/3/4/5/6/8 (design D1 naming rule: struct = body-func stem = key-const stem). Every
`New<Arm>(mode byte[, name])` ctor signature is consistent between its definition (Task 2/4) and its
body-func call (Task 3/5). The buddy `Operation()` writer-const ambiguity (`BuddyErrorWriter` vs
`BuddyOperationWriter`) is explicitly flagged for resolution in Task 4 Step 3.
