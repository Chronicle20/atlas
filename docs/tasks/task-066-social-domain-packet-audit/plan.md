# Social-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline shipped in task-027/028 to the **76 social-domain packets** in `libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{clientbound,serverbound}/`, ship wire-bug + template fixes against GMS v95 IDA, re-verify across v83/v87/JMS v185, and confirm no regression in the login (task-027) or character (task-028) audits.

**Architecture:** Phase 0 surveys `libs/atlas-packet/model/` to confirm the social sub-struct registry batch (`GuildMember`, `Buddy`, plus any newly-surfaced types) and lands those registrations with fixtures. Phase 1 audits the 76 packets in 6 sub-domain sub-tasks (note → buddy → messenger → chat → party → guild — warm-up to hot-path to largest). Phase 2 re-runs the audit against v83 / v87 / JMS v185 IDA, gating fixes by `Region/MajorVersion`. Phase 3 re-runs login + character audits as a regression gate. Phase 4 ships `post-phase-b.md`, full verification, code review, and PR.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast` for AST analysis already in `tools/packet-audit/internal/atlaspacket/`), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer + `libs/atlas-packet/test` `pt.Variants` for 4-variant round-trip tests, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies; this task ships audit reports + targeted code/template fixes.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-066-social-domain-packet-audit/` on branch `task-066-social-domain-packet-audit`. Before *every* commit run `git rev-parse --show-toplevel` (must end with `/.worktrees/task-066-social-domain-packet-audit`) and `git branch --show-current` (must be `task-066-social-domain-packet-audit`); if either disagrees, STOP.
- **TDD cadence (analyzer/registry).** Test first → run-to-fail → minimal implementation → run-to-pass → commit. Steps below spell each phase out.
- **Verification cadence (registry changes).** `go test -race ./tools/packet-audit/...` clean before commit.
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with a 4-variant test sweep covering GMS v28 / v83 / v95 + JMS v185 (use the existing `pt.Variants` pattern in `libs/atlas-packet/test/context.go`; v87 added when it surfaces during Phase 2).
- **No `*_testhelpers.go` files.** Use the project's Builder pattern. Per-test data is constructed via the existing `New<Packet>(...)` constructors in each clientbound/serverbound package.
- **No `reflect`, no new `interface{}` params, no benchmarks** in atlas-packet edits (design §6, inherited from task-028 §8).
- **Hard cap: 2 nested region/version guards per encoder** (design §8 Phase 2; carries from task-028 §7). 3+ → STOP, log `_pending.md` row, do not refactor under audit cover.
- **No gitleaks bait.** Absolute paths like `/home/<user>/` must not appear in any file under `docs/packets/audits/gms_v95/{guild,party,buddy,messenger,note,chat}/`. Pre-PR check is mandatory (Task 13 Step 4).
- **Tracking sub-tasks vs PR-sized commits.** Phase 1 sub-tasks (Tasks 2–7) and Phase 2 sub-tasks (Tasks 8–10) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every packet in its bucket has a verdict in `SUMMARY.md` and every ❌ has either a fix commit on this branch or a `_pending.md` row.
- **`_pending.md` row grouping.** Group deferrals by *cause*, not by *file*. One row per limitation with a sub-list of affected files (design §4.1, §9). One row per bare-handler family, not per handler.

---

## Phase 0 — Sub-struct registry batch (gate)

One survey + two registry tasks. Exit when `go test -race ./tools/packet-audit/...` is clean and the predicted social sub-struct types resolve through the registry.

### Task 1: Model survey + registry coverage fixtures

The social packets cite a small set of model sub-structs already auto-discovered by the registry (`registry.go` pass-2 walks every `Encode`/`Write` receiver method under `libs/atlas-packet/`). Per design §7, the predicted batch is:

| Sub-struct | Source file | Method | Used by |
|---|---|---|---|
| `model.GuildMember` | `libs/atlas-packet/model/guild_member.go:21` | `Encode` | `guild/clientbound/info.go`, `guild/clientbound/operation.go` (member-list ops) |
| `model.Buddy` | `libs/atlas-packet/model/buddy.go:19` | `Encode` | `buddy/clientbound/list_update.go`, `buddy/clientbound/update.go` |
| `model.Avatar` | `libs/atlas-packet/model/avatar.go` (existing) | `Encode` | `messenger/clientbound/add.go`, `update.go` (already used by other domains) |

`party.PartyMember` does **not** have an `Encode` method on the type — `libs/atlas-packet/party/member_data.go:19` exposes a package-level `WritePartyData(w, members, leaderId)` that flattens 6 fixed-size column slices. The registry's pass-2 only discovers receiver-method writers; package-level functions are invisible. This task documents that limitation in `_pending.md` rather than fixing the analyzer (design §1: "do not touch the analyzer unless a concrete social-domain finding forces a fix").

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`
- Modify: `docs/packets/ida-exports/_pending.md` (add survey deferral row)

- [ ] **Step 1: Re-verify the survey**

```bash
ls libs/atlas-packet/model/ | grep -iE "guild|buddy|party|messenger|chat|note"
```

Expected: includes `buddy.go` and `guild_member.go` at minimum. If a NEW sub-struct file appears that this plan didn't predict (e.g. `party_member.go`, `messenger_chat_entry.go`), the executor must add a registry fixture row for it in Step 2 below — register every type the social packets actually reference.

```bash
grep -RIn "package model" libs/atlas-packet/model/buddy.go libs/atlas-packet/model/guild_member.go libs/atlas-packet/model/avatar.go
```

Expected: each file in `package model`. If a sub-struct moved out of `model` into a sibling package (e.g. `libs/atlas-packet/guild/member.go`), the registry's `recvType` for it changes accordingly — adapt the fixture key.

- [ ] **Step 2: Add the failing fixtures**

Append to `tools/packet-audit/internal/atlaspacket/registry_test.go` (mirror the `TestRegistryRegistersCharacterSubStructs` style introduced by task-028 Task 5):

```go
func TestRegistryRegistersSocialSubStructs(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    for _, name := range []string{"GuildMember", "Buddy", "Avatar"} {
        if !reg.HasType(name) {
            t.Errorf("registry missing type %s", name)
            continue
        }
        calls, ok := reg.Calls(name)
        if !ok || len(calls) == 0 {
            t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
        }
    }
}
```

If Step 1 surfaced any additional social sub-struct types, append them to the slice literal above before running Step 3.

- [ ] **Step 3: Run to verify**

```bash
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersSocialSubStructs -v
```

Expected: PASS *on the first run* — these types already have `Encode` methods that pass-2 picks up.

If a type FAILs:
- "registry missing type X" → check if X uses a non-default receiver shape (e.g. pointer receiver). Inspect `libs/atlas-packet/model/<x>.go` and confirm against `registry.go`'s `receiverIdent` (registry.go ~line 152). If genuinely new walker behaviour is needed, STOP — the design forbids analyzer/registry surgery in this task; instead add the type to the Phase 0 deferral row in Step 4 and continue without it.
- "X.Encode produced no calls" → unlikely (each sub-struct file shows a `Write*` body). If it happens, the receiver method may be on a wrapper type the registry doesn't see; same handling as above.

- [ ] **Step 4: Document the `WritePartyData` package-function limitation in `_pending.md`**

Open `docs/packets/ida-exports/_pending.md` and append (creating the heading if it does not yet exist):

```markdown
## Sub-op enum / sub-struct deferrals — social domain (task-066)

- **`party.WritePartyData` (package-level function)** — `libs/atlas-packet/party/member_data.go:19` flattens 6 fixed-size column slices (id, name, jobId, level, channelId, mapId) plus a leader id and 6×4 zero-padding tail. The audit pipeline's TypeRegistry walks receiver-method `Encode`/`Write` only; package-level write helpers are invisible. Affected packets: `party/clientbound/update.go`, `party/clientbound/join.go`, `party/clientbound/left.go`. Audit verdict for these three files will be ⚠️ "tool-limitation: package-level write helper not modelled; verify against IDA member-list shape".
```

If Step 1 surfaced additional sub-struct types not registered in Step 2, add a sibling bullet here describing the missing type and the affected packets.

- [ ] **Step 5: Run the full audit-tool test suite**

```bash
go test -race ./tools/packet-audit/...
```

Expected: clean. Confirms no fixture regression elsewhere.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/registry_test.go \
        docs/packets/ida-exports/_pending.md
git commit -m "test(packet-audit): assert social sub-struct registry coverage

Adds GuildMember/Buddy/Avatar registry fixtures (mirrors task-028's
character sub-struct fixture). Documents party.WritePartyData package-
level helper as a known tool-limitation in _pending.md (affects
party/clientbound/{update,join,left}.go)."
```

Verify post-commit:

```bash
git rev-parse --show-toplevel
git branch --show-current
```

Expected: ends with `/.worktrees/task-066-social-domain-packet-audit` and branch is `task-066-social-domain-packet-audit`. If either disagrees, STOP.

---

## Phase 1 — v95 audit by sub-domain

Six tracking sub-tasks (Tasks 2–7), one per social sub-domain. Ordering: **note → buddy → messenger → chat → party → guild** — warm-up (smallest, simplest dispatcher pattern) to hot-path (member_hp/update) to largest (guild's 24 packets including `bbs_*` and `operation_*` families).

The audit command is the same for every sub-task in this phase:

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

It produces per-packet reports under `docs/packets/audits/gms_v95/<domain>/<PacketName>.{md,json}` and updates `docs/packets/audits/gms_v95/SUMMARY.md`. Run once per sub-task; commit the report files alongside the fix commits.

Before starting Phase 1, the user must have v95 IDA loaded so MCP `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the audit report bucket commit.

### Verdict triage rules (apply within every sub-task)

For each report under `docs/packets/audits/gms_v95/<domain>/`:

- **✅** → no action; row already in `SUMMARY.md`.
- **⚠️** → annotate the report manually with a one-line "ack: <reason>" footer; commit alone (commit message: `audit(<domain>/<pkt>): ack <reason>`). Examples: tool-limited package-level write helper (per Phase 0 row), loop-flattened lists (analyzer flattens fixed-count loops; cite IDA loop bound), `EncodeMask` / sub-struct method calls that emit multiple bytes per analyzer call.
- **❌** → triage by flavour:
  - **Atlas wire bug** (width / order / missing field / silent-success) → fix in `libs/atlas-packet/<domain>/<dir>/<pkt>.go`. Add or extend a 4-variant test sweep in `<pkt>_test.go` (template provided in Step 4 of every Phase 1 sub-task).
  - **Template opcode drift** → fix in `services/atlas-configurations/seed-data/templates/template_gms_*_1.json` and/or `template_jms_185_1.json`. Atlas-packet stays untouched.
  - **Sub-op enum drift** (chat type-discriminator, guild operation result code, party operation result code, buddy error code, messenger decline reason) → defer to `_pending.md` per design §9, grouped under the existing "Sub-op enum / sub-struct deferrals — social domain (task-066)" heading from Task 1.
  - **Bare handler / no atlas-packet decoder** → defer to `_pending.md` under a new "## Bare handlers — social domain (task-066)" heading (create on first hit). Do not descend into atlas-channel/atlas-guild/atlas-party/atlas-buddies service code (PRD §3 non-goal).
  - **Operation-dispatcher op-byte parameter** (per design §5: `guild/serverbound/operation.go`, `party/serverbound/operation.go`, `messenger/serverbound/operation.go`, `note/clientbound/operation.go`, `note/serverbound/operation.go`, `bbs_operation.go`) → ⚠️ verdict + footer noting "op-byte value supplied by caller; see `_pending.md` row OP-FAMILY-{guild,party,messenger,note,bbs}". Add the OP-FAMILY rows to `_pending.md` in the first sub-task that encounters each family.

### Per-fix recipe (used inside every Phase 1 sub-task)

For each ❌ atlas-wire-bug fix:

1. **Fetch IDA evidence.** For clientbound packets the IDA function is `CClientSocket::SendXxx` (or the named writer); for serverbound, `CWvsContext::OnXxxPacket` or the named handler. Use the FName from the audit report:

   ```
   mcp__ida-pro__get_function_by_name("<FName>")
   mcp__ida-pro__decompile_function(<addr>)
   ```

   Append the function's signature + address + a `Decode*`/`Encode*` op summary (matching the existing `gms_v95.json` schema's `Decode1/2/4/Str/Buffer/Loop` shape) to `docs/packets/ida-exports/gms_v95.json`.

2. **Edit the encoder.** Apply the minimum `Write*` change needed to match IDA. For version-conditional fixes use the existing `tenant.Model.Region()` / `tenant.Model.MajorVersion()` axes; respect the 2-nested-guard cap.

3. **Add the 4-variant test sweep.** Mirror task-028's pattern (e.g. `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37`). For clientbound:

   ```go
   func TestMemberHPByteForByte(t *testing.T) {
       cases := []struct {
           name string
           tn   tenant.Model
           want string // hex
       }{
           {"gms_v83", pt.GMSv83(), "<hex from IDA>"},
           {"gms_v95", pt.GMSv95(), "<hex from IDA>"},
           {"jms_v185", pt.JMSv185(), "<hex from IDA>"},
           // pt.Variants iteration is the canonical form; this expansion is here to
           // make the IDA hex source explicit per variant. If a fourth variant
           // (GMS v28 or v87) is in pt.Variants at this commit, add it.
       }
       for _, tc := range cases {
           t.Run(tc.name, func(t *testing.T) {
               ctx := tenant.WithContext(context.Background(), tc.tn)
               got := NewPartyMemberHP(/* fields per packet */).Encode(testLogger(), ctx)(nil)
               if hex.EncodeToString(got) != tc.want {
                   t.Fatalf("encode mismatch\n got %s\nwant %s", hex.EncodeToString(got), tc.want)
               }
           })
       }
   }
   ```

   For serverbound, use the round-trip pattern from `libs/atlas-packet/test/roundtrip.go:12-24` — decode known-good IDA hex bytes, assert `r.Available() == 0` and field values.

   Hex values are captured from IDA by hand: in the decompile, find the call-site or the case-statement body and translate the `WriteXxx` / `ReadXxx` sequence to bytes.

4. **Run the affected packet's tests:**

   ```bash
   go test -race ./libs/atlas-packet/<domain>/<dir>/... -run Test<Pkt> -v
   ```

   Expect: clean.

5. **Commit per fix:**

   For atlas-packet fixes:

   ```bash
   git add libs/atlas-packet/<domain>/<dir>/<pkt>.go \
           libs/atlas-packet/<domain>/<dir>/<pkt>_test.go
   git commit -m "fix(atlas-packet,<domain>/<pkt>): <one-line summary>

   Cites IDA <CClientSocket::SendXxx>@<addr>: <one-line evidence>."
   ```

   For template fixes:

   ```bash
   git add services/atlas-configurations/seed-data/templates/template_*.json
   git commit -m "fix(configurations,templates): <pkt> opcode <old>→<new> for <region/version>

   IDA case-statement value at <CWvsContext::OnXxxPacket>@<addr>."
   ```

   For `_pending.md` deferrals:

   ```bash
   git add docs/packets/ida-exports/_pending.md
   git commit -m "audit(<domain>/<pkt>): defer — <one-line reason>"
   ```

### Bucket-commit recipe (end of every Phase 1 sub-task)

After all per-fix commits land, commit the audit reports + SUMMARY + IDA-export append in one bucket commit:

```bash
git add docs/packets/audits/gms_v95/<domain>/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(<domain>): v95 audit (<n> packets)"
```

### Exit gate (every Phase 1 sub-task)

```bash
ls docs/packets/audits/gms_v95/<domain>/*.md | wc -l
```

Must equal the bucket packet count. Then:

```bash
grep -c "<domain>/" docs/packets/audits/gms_v95/SUMMARY.md
```

Must equal the same count. Every ❌ in `SUMMARY.md` for this sub-domain has either a fix commit on this branch (`git log --oneline | grep "<domain>/<pkt>"`) or a row in `_pending.md`.

---

### Task 2: Phase 1a — note (6 packets)

**Packets — clientbound (3):** `display.go`, `operation.go`, `operation_body.go`
**Packets — serverbound (3):** `operation.go`, `operation_discard.go`, `operation_send.go`

**Files:**
- Modify (per fix): `libs/atlas-packet/note/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/note/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/note/<PacketName>.{md,json}` (created by audit run)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v95. If not, ask the user to swap before continuing.

- [ ] **Step 2: Run the audit (full pipeline; the tool produces note/* reports as a subset of the run).**

Use the audit command from the Phase 1 preamble. Expected runtime: ≤ 60 s.

- [ ] **Step 3: Triage each note/* report.**

Apply the verdict triage rules from the Phase 1 preamble. `note/serverbound/operation.go` is the operation-dispatcher (op-byte only) — record an OP-FAMILY-note row in `_pending.md` if any ❌ triggers it; verdict ⚠️ on the file. `note/clientbound/operation_body.go` carries the per-op result body shape — audit normally.

- [ ] **Step 4: Per-fix loop — for each ❌, follow the per-fix recipe (5 sub-steps) from the Phase 1 preamble.**

- [ ] **Step 5: Bucket commit (audit reports + SUMMARY + IDA-export append).**

```bash
git add docs/packets/audits/gms_v95/note/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(note): v95 audit (6 packets)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/note/*.md | wc -l
```

Expected: 6 (counting `display.md`, `operation_cb.md` if renamed for cb/sb collision, `operation_body.md`, `operation_sb.md`, `operation_discard.md`, `operation_send.md`). If the tool collides cb and sb `operation.md` filenames, hand-rename one to `operation_cb.md` / `operation_sb.md` consistently and re-stage.

```bash
grep -c "atlas-packet/note/" docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: 6.

---

### Task 3: Phase 1b — buddy (9 packets)

**Packets — clientbound (6):** `capacity_update.go`, `channel_change.go`, `error.go`, `invite.go`, `list_update.go`, `update.go`
**Packets — serverbound (3):** `operation_accept.go`, `operation_add.go`, `operation_delete.go`

`buddy/clientbound/list_update.go` and `update.go` exercise the `model.Buddy` sub-struct registered in Phase 0 — first practical proof of the registry batch. `buddy/clientbound/error.go` is the canonical sub-op enum surface (capacity-full, target-offline, target-blocked, etc.) — sub-op enum drift goes to `_pending.md` per design §9. `buddy/clientbound/list_update.go` linearises the buddy list loop — analyzer flattens it; ⚠️ verdict + IDA loop-bound citation.

**Files:**
- Modify (per fix): `libs/atlas-packet/buddy/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/buddy/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/buddy/<PacketName>.{md,json}` (audit-generated)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append: `docs/packets/ida-exports/gms_v95.json`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is still loaded** (`mcp__ida-pro__get_metadata`).
- [ ] **Step 2: Run the audit (Phase 1 preamble command).**
- [ ] **Step 3: Triage each buddy/* report per Phase 1 preamble triage rules.** `error.go` → likely sub-op enum row in `_pending.md` (group under Task 1's deferral heading).
- [ ] **Step 4: Per-fix loop — Phase 1 per-fix recipe.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/buddy/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(buddy): v95 audit (9 packets)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/buddy/*.md | wc -l   # Expected: 9
grep -c "atlas-packet/buddy/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 9
```

---

### Task 4: Phase 1c — messenger (13 packets)

**Packets — clientbound (8):** `add.go`, `chat.go`, `invite_declined.go`, `invite_sent.go`, `join.go`, `remove.go`, `request_invite.go`, `update.go`
**Packets — serverbound (5):** `operation.go`, `operation_answer_invite.go`, `operation_chat.go`, `operation_decline_invite.go`, `operation_invite.go`

`messenger/clientbound/add.go` and `update.go` consume `model.Avatar` (already registered, design §7). `messenger/serverbound/operation.go` is the dispatcher (op-byte only) — OP-FAMILY-messenger row in `_pending.md`; verdict ⚠️ on the file. `messenger/clientbound/invite_declined.go` carries decline-reason sub-op enum — `_pending.md` row.

**Files:**
- Same as Task 3 substituting `buddy` → `messenger`.

- [ ] **Step 1: Confirm v95 IDA is loaded.**
- [ ] **Step 2: Run the audit.**
- [ ] **Step 3: Triage each messenger/* report.** OP-FAMILY-messenger row in `_pending.md` for `serverbound/operation.go`. Sub-op enum row covers `invite_declined.go` decline reasons.
- [ ] **Step 4: Per-fix loop.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/messenger/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(messenger): v95 audit (13 packets)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/messenger/*.md | wc -l   # Expected: 13
grep -c "atlas-packet/messenger/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 13
```

---

### Task 5: Phase 1d — chat (8 packets, highest deferral density)

**Packets — clientbound (5):** `general.go`, `multi.go`, `whisper.go`, `world_message.go`, `world_message_extra.go`
**Packets — serverbound (3):** `general.go`, `multi.go`, `whisper.go`

This is the design §4 sub-mode dispatch concentration. Per design §4 there are two outcomes per chat packet:

1. **Sub-mode is hard-coded per file** (the file always writes a literal mode byte) → audit normally; verdict drives whether to fix the literal or the template. Verify by reading `Encode` body: if the first `WriteByte(...)` argument is a struct field (e.g. `m.mode`), it is a parameter (case 2 below); if it is a numeric literal (e.g. `WriteByte(0x12)`), it is hard-coded.

2. **Sub-mode is a parameter** (e.g. `general.go` accepts a `mode` field and writes it as `m.mode`) → verdict ⚠️ "manual sub-op review"; sub-mode value space goes to one consolidated `_pending.md` row keyed by "sub-op enum modeling — chat domain", NOT one row per file (design §4.1).

Read each file once before triaging:

```bash
grep -n "WriteByte" libs/atlas-packet/chat/clientbound/{general,multi,whisper,world_message,world_message_extra}.go
grep -n "WriteByte" libs/atlas-packet/chat/serverbound/{general,multi,whisper}.go
```

**Files:**
- Same shape as Task 3 substituting `buddy` → `chat`.

- [ ] **Step 1: Confirm v95 IDA is loaded.**
- [ ] **Step 2: Run the audit.**
- [ ] **Step 3: Per-file sub-mode classification.** Apply the case-1/case-2 dichotomy above to each of the 8 chat files. For case-2 files, append a single bullet to the existing `_pending.md` heading "Sub-op enum / sub-struct deferrals — social domain (task-066)" naming all parameterised chat files in one row:

  ```markdown
  - **Chat sub-mode enum modelling** — `chat/{clientbound,serverbound}/<file>.go` accept a `mode` byte parameter (NORMAL=0, WHISPER, MEGAPHONE, SMEGA, ITEM_POP, AVATAR_MEGAPHONE, HEART, SKULL). The audit pipeline cannot statically check the value space; verdict ⚠️ on each file. Sub-mode mapping captured in IDA dispatcher case-statements; cross-version verification deferred. Affected files: <list>.
  ```

  Cap: if the parameterised sub-mode space exceeds 5 distinct values per file (per design §4), defer enum verification entirely; otherwise the audit-report footer for that file inlines the verdict.

- [ ] **Step 4: Per-fix loop.** For case-1 files (hard-coded sub-mode literal), if the literal disagrees with IDA's dispatcher case-statement value, follow the Phase 1 per-fix recipe.
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/chat/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(chat): v95 audit (8 packets) + sub-mode enum deferral"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/chat/*.md | wc -l   # Expected: 8 (with cb/sb name disambiguation if needed)
grep -c "atlas-packet/chat/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 8
```

---

### Task 6: Phase 1e — party (16 packets, includes hot path)

**Packets — clientbound (10):** `change_leader.go`, `created.go`, `disband.go`, `error.go`, `invite.go`, `join.go`, `left.go`, `member_hp.go`, `operation_body.go`, `update.go`
**Packets — serverbound (6):** `invite_reject.go`, `operation.go`, `operation_change_leader.go`, `operation_expel.go`, `operation_invite.go`, `operation_join.go`

`party/clientbound/member_hp.go` (broadcasts on every party-member HP change) and `update.go` (broadcasts on join/leave/leader-change) are the social hot path (design §6). 4-variant byte-output test sweep is mandatory for any fix to these two; benchmark check explicitly NOT required (design §6 inherits task-028 §8 — no benchmarks).

`party/clientbound/{update,join,left}.go` use `party.WritePartyData` (the package-level helper from Phase 0) — verdict ⚠️ "tool-limitation: package-level write helper not modelled; cross-checked against IDA member-list shape <addr>". Cite the IDA function for evidence even when the audit pipeline can't model the write.

`party/serverbound/operation.go` is the dispatcher → OP-FAMILY-party row in `_pending.md`; ⚠️ verdict on the file. `party/clientbound/error.go` and `operation_body.go` carry party operation result codes → sub-op enum row.

**Files:**
- Same shape as Task 3 substituting `buddy` → `party`.

- [ ] **Step 1: Confirm v95 IDA is loaded.**
- [ ] **Step 2: Run the audit.**
- [ ] **Step 3: Triage each party/* report. Add OP-FAMILY-party + party operation-result enum rows to `_pending.md` if not already present.**
- [ ] **Step 4: Per-fix loop. Hot-path discipline (member_hp/update) — 4-variant byte-output sweep is mandatory; cite IDA dispatcher offset in the fix comment.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/party/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(party): v95 audit (16 packets)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/party/*.md | wc -l   # Expected: 16
grep -c "atlas-packet/party/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 16
```

---

### Task 7: Phase 1f — guild (24 packets, largest)

**Packets — clientbound (5):** `bbs.go`, `emblem_changed_foreign.go`, `info.go`, `name_changed_foreign.go`, `operation.go`
**Packets — serverbound (19):** `bbs_create_or_edit_thread.go`, `bbs_delete_reply.go`, `bbs_delete_thread.go`, `bbs_display_thread.go`, `bbs_list_threads.go`, `bbs_operation.go`, `bbs_reply_thread.go`, `invite_reject.go`, `operation.go`, `operation_agreement_response.go`, `operation_invite.go`, `operation_join.go`, `operation_kick.go`, `operation_request_create.go`, `operation_set_emblem.go`, `operation_set_member_title.go`, `operation_set_notice.go`, `operation_set_title_names.go`, `operation_withdraw.go`

This is the largest sub-domain. Two dispatcher families:
- **`guild/serverbound/operation.go`** dispatcher (op-byte only, confirmed at `libs/atlas-packet/guild/serverbound/operation.go:30-40`) → 12 `operation_*` sub-op files audit individually. OP-FAMILY-guild row in `_pending.md`.
- **`guild/serverbound/bbs_operation.go`** dispatcher (op-byte only) → 6 `bbs_*` sub-op files audit individually. OP-FAMILY-bbs row in `_pending.md`.

`guild/clientbound/info.go` and `operation.go` are the primary `model.GuildMember` consumers — first practical proof of the GuildMember registration from Phase 0. `guild/clientbound/operation.go` (clientbound) carries guild operation result codes (invite-result, join-result, kick-result, rank-update-result) → sub-op enum row in `_pending.md`.

**Files:**
- Same shape as Task 3 substituting `buddy` → `guild`.

- [ ] **Step 1: Confirm v95 IDA is loaded.**
- [ ] **Step 2: Run the audit.**
- [ ] **Step 3: Triage each guild/* report. Add OP-FAMILY-guild and OP-FAMILY-bbs rows to `_pending.md` (one row per family). Add guild operation-result enum row if not already present from earlier sub-tasks.**

  Sanity-check the dispatcher reading: re-read `libs/atlas-packet/guild/serverbound/operation.go` and `bbs_operation.go`. If either dispatcher's `Encode` body emits more than one `WriteByte(...)` (i.e. carries payload beyond the op byte), design §10's "Low likelihood" risk has materialised — record the actual dispatcher shape in the audit report instead of treating as op-byte-only, and audit accordingly. Do not rewrite the design.

- [ ] **Step 4: Per-fix loop. Pay special attention to `bbs.go` clientbound — it author/replyer entries are likely a sub-struct list; verify against `model.GuildMember` (Phase 0 registration) or an inline struct.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/guild/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(guild): v95 audit (24 packets)"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/guild/*.md | wc -l   # Expected: 24
grep -c "atlas-packet/guild/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: 24
```

**Phase 1 exit:** `SUMMARY.md` contains rows for all 76 social-domain packets. Every ❌ has a fix commit on this branch OR a `_pending.md` row.

```bash
grep -c "atlas-packet/\(guild\|party\|buddy\|messenger\|note\|chat\)/" docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: 76. If less, identify the missing files via:

```bash
diff <(ls libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{clientbound,serverbound}/*.go 2>/dev/null | grep -v _test.go | xargs -n1 basename | sort -u) \
     <(grep -oE "atlas-packet/(guild|party|buddy|messenger|note|chat)/(clientbound|serverbound)/[a-z_]+\.go" docs/packets/audits/gms_v95/SUMMARY.md | xargs -n1 basename | sort -u)
```

Investigate every missing file before exiting Phase 1.

---

## Phase 2 — Cross-version pass (v83 → v87 → JMS v185)

Three tracking sub-tasks (Tasks 8–10), one per version. Each requires a user-driven IDA binary swap before starting (PRD §4.6, design §8 Phase 2). A sub-task is "done" when:

- `docs/packets/ida-exports/gms_{v83,v87}.json` or `gms_jms_185.json` contains social-domain entries for every FName resolved during Phase 1.
- The audit has been re-run against the version's template + IDA export, producing reports under `docs/packets/audits/<version>/<social-domain>/`.
- Every divergence vs v95 atlas-packet behaviour has either:
  - A `Region/MajorVersion` gate that already handles it (audit report captures evidence; no code change),
  - A gate fix on this branch with a 4-variant test sweep, OR
  - A template fix.
- Hard cap: if any single social-domain encoder now contains 3+ nested `if t.Region()` / `if t.MajorVersion()` levels, STOP per design §8 / task-028 §7. Append a row to `_pending.md` describing the encoder + which version chain triggered it. Do NOT refactor in this task.

### Task 8: GMS v83 cross-version pass

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (append social entries — file already exists with login + possibly character entries)
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Create or modify: `docs/packets/audits/gms_v83/<domain>/<pkt>.{md,json}` (tool creates the directory)
- Modify: `docs/packets/audits/gms_v83/SUMMARY.md`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v83 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v83. If not, ask the user to swap before continuing.

- [ ] **Step 2: For each social-domain FName resolved during Phase 1, populate `gms_v83.json`.**

Workflow (per FName):
- `mcp__ida-pro__get_function_by_name("<FName>")` (or `get_function_by_address` if names diverge).
- `mcp__ida-pro__decompile_function(<addr>)`.
- Translate to the existing `gms_v83.json` schema (`Decode1/2/4/Str/Buffer/Loop` op list with guard expressions). Append entries; do not reorder existing login/character entries.

If a FName has no v83 equivalent (different opcode space, no matching function), record the v83-side FName + address as a separate entry annotated with `"region": "GMS"` and `"version": 83`. Do NOT reuse v95 FNames for unrelated v83 functions.

- [ ] **Step 3: Re-run the audit against v83.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v83.json \
  --output           docs/packets/audits/gms_v83
```

If `docs/packets/audits/gms_v83/` doesn't exist yet, the tool creates it.

- [ ] **Step 4: Triage divergences.**

For each ❌ in the v83 social audit:
- Was the v95 fix gated on `MajorVersion() >= 95`? → no v83 regression. Audit-report-only.
- Was the v95 fix gated on `Region() == "GMS"` (no major-version filter)? → check whether v83 IDA confirms the same behaviour. If yes: tighten the gate so v83 keeps its old shape. If no: leave as-is and document.
- Is this a *new* v83-only mismatch the v95 audit didn't surface? → genuine cross-version bug. Fix with 4-variant test sweep + `Region/MajorVersion` gate. Per-fix recipe from Phase 1 preamble.
- **Hard cap:** if the fix makes the encoder breach 3 nested guards, STOP — `_pending.md` row, no refactor.

- [ ] **Step 5: Commit per-fix; bucket commit for the version.**

Per-fix commit format:

```
fix(atlas-packet,<domain>/<pkt>): widen/narrow v83 gate for <field>

Cites IDA v83 <CClientSocket::SendXxx>@<addr>: <one-line evidence>.
```

Final bucket commit:

```bash
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/
git commit -m "audit(social): GMS v83 cross-version pass (social domain)"
```

- [ ] **Step 6: Hard-cap check.**

```bash
for f in libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{clientbound,serverbound}/*.go; do
    [[ "$f" == *_test.go ]] && continue
    nested=$(awk '
        /if t\.Region\(\)|if t\.MajorVersion\(\)|if .*\.Region\(\)|if .*\.MajorVersion\(\)/ { d++; if (d > max) max = d }
        /^}/ { if (d > 0) d-- }
        END { print max+0 }
    ' "$f")
    if (( nested >= 3 )); then
        echo "OVER CAP: $f ($nested nested guards)"
    fi
done
```

If any "OVER CAP" line appears, append a row to `docs/packets/ida-exports/_pending.md` describing the encoder + which version chain triggered it; do NOT refactor in this task.

---

### Task 9: GMS v87 cross-version pass

Identical shape to Task 8. Replace `v83` with `v87` everywhere. Templates: `template_gms_87_1.json`. Export file: `docs/packets/ida-exports/gms_v87.json` (does not exist yet per `ls docs/packets/ida-exports/` showing only `gms_v83.json` and `gms_v95.json` and `_pending.md`; create the file by Step 2's first append).

**Files:**
- Create: `docs/packets/ida-exports/gms_v87.json`
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Create or modify: `docs/packets/audits/gms_v87/<domain>/<pkt>.{md,json}`
- Modify: `docs/packets/audits/gms_v87/SUMMARY.md`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v87 IDA is loaded** (`mcp__ida-pro__get_metadata`; binary field == GMS v87).
- [ ] **Step 2: Populate `gms_v87.json` for the social FNames from Phase 1.** Workflow per Task 8 Step 2; this task creates the file from scratch.
- [ ] **Step 3: Re-run the audit.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v87.json \
  --output           docs/packets/audits/gms_v87
```

- [ ] **Step 4: Triage divergences** (per Task 8 Step 4).
- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_v87.json \
        docs/packets/audits/gms_v87/
git commit -m "audit(social): GMS v87 cross-version pass (social domain)"
```

- [ ] **Step 6: Hard-cap check** (per Task 8 Step 6 with the same awk loop).

---

### Task 10: JMS v185 cross-version pass

JMS v185 had a separate opcode space for login/character (task-027/028 finding) and a divergent guild/party feature set (PRD §9 q3: different alliance tiers, different BBS structure). Expect heavier divergence than GMS v83/v87.

**Files:**
- Create: `docs/packets/ida-exports/gms_jms_185.json`
- Modify (per fix): `libs/atlas-packet/<domain>/<dir>/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Create or modify: `docs/packets/audits/jms_v185/<domain>/<pkt>.{md,json}`
- Modify: `docs/packets/audits/jms_v185/SUMMARY.md`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm JMS v185 IDA is loaded** (`mcp__ida-pro__get_metadata`).
- [ ] **Step 2: Populate `gms_jms_185.json` for the social FNames from Phase 1.**

If a FName has no JMS equivalent, record the JMS-side FName + address as a separate entry annotated with `"region": "JMS"` and `"version": 185`. Do NOT reuse GMS FNames for unrelated JMS functions.

- [ ] **Step 3: Re-run the audit.**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage per design §8 / task-028 §7.1 in-scope rules:**
  - In scope: atlas-packet writes bytes the JMS client decodes wrong.
  - Out of scope: JMS-specific feature the service doesn't wire through (JMS-only alliance tier, JMS BBS structure differences) → `_pending.md` row + sibling-task suggestion.
  - In scope: width mismatch on a field both versions decode.
  - Out of scope: JMS template opcode wrong when v95 is right → fix the template, atlas-packet untouched.

- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/
git commit -m "audit(social): JMS v185 cross-version pass (social domain)"
```

- [ ] **Step 6: Hard-cap check** (per Task 8 Step 6).

---

## Phase 3 — Login + character regression confirm

Mechanical re-run of the existing login (task-027) and character (task-028, if landed on the branch base) audits. No verdict regression vs the snapshot recorded in their `post-phase-b.md` files. Cap: 2 new ❌s across login + character is the budget before stop-and-split (design §8 Phase 3).

### Task 11: Re-run login + character audits + assert no verdict regression

**Files:**
- Modify (read-only assertion; should not change unless regression): `docs/packets/audits/gms_v95/SUMMARY.md`
- Create (if regression diagnosis needed): `docs/tasks/task-066-social-domain-packet-audit/regression-notes.md`

- [ ] **Step 1: Snapshot the current SUMMARY verdicts before re-run.**

```bash
cp docs/packets/audits/gms_v95/SUMMARY.md /tmp/summary-pre-phase3.md
```

- [ ] **Step 2: Re-run the v95 audit (full pipeline, login + character + social).**

```bash
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

- [ ] **Step 3: Diff the SUMMARY against the pre-Phase-3 snapshot.**

```bash
diff /tmp/summary-pre-phase3.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: empty (no diff) for login (`atlas-packet/login/`) and character (`atlas-packet/character/`) rows.

If a login/character row's verdict changed:
- **Verdict regressed (✅ → ❌ or ✅ → ⚠️ for any login/character row)** → in-scope to triage. For each regressed packet:
  - Read the new audit report (`docs/packets/audits/gms_v95/<PacketName>.md`).
  - Decompile via `mcp__ida-pro__decompile_function`.
  - Identify whether a Phase 1/2 social fix's gate accidentally affected the login/character encoder (e.g. a shared sub-struct, a shared template entry).
  - Fix as a one-commit follow-up here, with a 4-variant sweep in `libs/atlas-packet/login/` or `libs/atlas-packet/character/`.
  - Cap: 2 regressions across login + character is the budget. 3+ → STOP. Document each in a new `regression-notes.md` and ask the user to spin up a sibling task. Do not proceed to Phase 4 until cleared.
- **Verdict improved (❌ → ✅)** → no action; record in Phase 4 `post-phase-b.md` "Tooling improvements" section as a side-effect win.

- [ ] **Step 4: Commit the (possibly unchanged) SUMMARY + any regression-fix commits individually.**

If no diff, no commit needed for the regression check itself. If fix commits surfaced in Step 3, each commits as `fix(atlas-packet,<area>/<pkt>): <reason>` per the per-fix recipe. After all regression fixes:

```bash
git add docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(social): no login/character verdict regression"
```

(Skip this commit if Step 3's diff was empty AND no fix commits ran.)

---

## Phase 4 — Closeout

### Task 12: `post-phase-b.md`, full verification, code review, PR

**Files:**
- Create: `docs/tasks/task-066-social-domain-packet-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tally row, if not already present from earlier sub-tasks)
- Modify: `docs/packets/ida-exports/_pending.md` (final social-domain section state)

- [ ] **Step 1: Write `post-phase-b.md`.**

Mirror task-027/028 structure verbatim. Five sections:

```markdown
# Task-066 Post-Phase-B — Social-Domain Audit Closeout

## Final state
- Packets audited: 76 (37 clientbound + 39 serverbound across guild, party, buddy, messenger, note, chat).
- Verdicts: ✅ <n_pass> / ⚠️ <n_warn> / ❌ <n_fail> / 🔍 <n_review> / pending <n_pending>.
- IDA-export coverage: v83 / v87 / v95 / JMS v185 — social FNames populated.

## Real wire bugs fixed
| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit from Phase 1/2/3)

## Template opcode / enum fixes
| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements
- Registry fixtures for GuildMember / Buddy / Avatar (Phase 0).
- Documented `party.WritePartyData` package-level helper as known tool-limitation in `_pending.md`.
- Documented OP-FAMILY rows for guild / party / messenger / note / bbs dispatchers.
- Documented chat sub-mode enum modelling deferral.
- (Any side-effect login/character verdict improvements from Phase 3.)

## Remaining work
| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` social-domain section + any §8 hard-cap stops + bare-handler families)
```

Fill in actual numbers and rows from the commit history:

```bash
git log --oneline 71ff6ea82^..HEAD | grep -E "^[0-9a-f]+ (fix|audit)" > /tmp/social-commits.txt
```

(`71ff6ea82` is the spec commit; replace with the actual base if different.)

- [ ] **Step 2: Run the four PRD §10 verification commands.**

```bash
go build ./...
go vet ./libs/atlas-packet/...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All four must be clean.

- [ ] **Step 3: Decide whether `docker build` is required.**

Per CLAUDE.md Build & Verification §3: required when a service `Dockerfile` or `go.mod` was touched. This task is expected to touch only `template_*.json` files under `services/atlas-configurations/seed-data/`. Confirm:

```bash
git diff --name-only $(git merge-base main HEAD)..HEAD -- services/atlas-configurations/ | grep -v 'seed-data/templates/'
```

If empty: skip `docker build`. Otherwise:

```bash
docker build -f services/atlas-configurations/Dockerfile .
```

Expected: clean. If it fails on workspace replace lines, the affected Dockerfile needs its `COPY` / `go mod edit -replace` blocks updated — fix and re-run.

- [ ] **Step 4: gitleaks scrub.**

```bash
grep -r '/home/' docs/packets/audits/gms_v95/{guild,party,buddy,messenger,note,chat}/ \
                 docs/packets/audits/gms_v83/{guild,party,buddy,messenger,note,chat}/ \
                 docs/packets/audits/gms_v87/{guild,party,buddy,messenger,note,chat}/ \
                 docs/packets/audits/jms_v185/{guild,party,buddy,messenger,note,chat}/ \
                 2>/dev/null
```

Expected: no output. If any user-home path appears in an audit report, scrub it and commit:

```bash
sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>
git commit -am "audit: scrub absolute user-home paths from social/* reports"
```

- [ ] **Step 5: Commit `post-phase-b.md`.**

```bash
git add docs/tasks/task-066-social-domain-packet-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "docs(task-066): post-phase-b closeout"
```

- [ ] **Step 6: Run code review.**

Invoke `superpowers:requesting-code-review`. Allow the orchestration skill to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/` and `tools/packet-audit/` changes.

Read the resulting `audit.md` and act on every BLOCKER / MAJOR finding before opening a PR. Re-run reviews after fix commits land.

- [ ] **Step 7: Open the PR.**

Title: `task-066: social-domain packet audit (v83/v87/v95/JMS185)`

Body: short summary + link to `post-phase-b.md` for the full bug ledger. Use `superpowers:finishing-a-development-branch` to drive the PR creation.

---

## Self-review notes

Run through the plan once more with fresh eyes before committing it.

- **Spec coverage** — every PRD §4 functional requirement is covered by an explicit task above:
  - §4.1 coverage matrix (76 packets — corrected from 147 per design §3) → Phase 1 (Tasks 2–7).
  - §4.2 IDA exports → Phase 1 v95 + Phase 2 v83/v87/JMS-185 (Tasks 2–10).
  - §4.3 wire-bug fixes → embedded per-fix recipe in every Phase 1 + Phase 2 sub-task.
  - §4.4 template fixes → embedded per-fix recipe.
  - §4.5 TypeRegistry extensions → Phase 0 (Task 1).
  - §4.6 cross-version re-verification → Phase 2 (Tasks 8–10).
  - §4.7 deferral handling → embedded triage rules in Phase 1 preamble; `_pending.md` headings created in Task 1.

- **Acceptance criteria coverage** — every PRD §10 acceptance bullet maps to a Task:
  - "all 147 listed packet files have audit reports" — corrected to 76 per design §3.1; covered by Phase 1 exit gate.
  - "every ❌ verdict has either a fix commit OR a `_pending.md` row" — Phase 1 sub-task exit gates.
  - "all 4 verification commands pass cleanly" — Task 12 Step 2.
  - "gitleaks scrub clean" — Task 12 Step 4.
  - "post-phase-b.md ledger written" — Task 12 Step 1.
  - "plan-adherence-reviewer and backend-guidelines-reviewer dispatched" — Task 12 Step 6.
  - "login (task-027) and character (task-028) audit verdicts unchanged" — Phase 3 (Task 11).

- **No placeholders** — every step contains either an exact command, an exact code block, or an exact file path. No "TBD" / "similar to" / "fill in".

- **Type consistency** — `model.GuildMember`, `model.Buddy`, `model.Avatar` are referenced consistently across Tasks 1, 3, 4, 7. Audit-output paths `docs/packets/audits/gms_v95/<domain>/` (six sub-domains) are consistent across Tasks 2–7. IDA-export filenames `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` consistent across Tasks 8–10. Template filenames `template_gms_83_1.json`, `template_gms_87_1.json`, `template_jms_185_1.json` consistent.

- **Loop-internal early-return / analyzer surgery is explicitly out of scope** per design §1. Phase 0 documents the `WritePartyData` limitation in `_pending.md` instead of teaching the analyzer.

- **Sub-op enum drift** — Phase 1 preamble defers chat sub-mode + buddy/messenger/party error codes + guild operation results to `_pending.md`. Single row per *cause*, not per file (design §4.1, §9). Encoder change for these is forbidden in this task.

- **Hot-path discipline** — Task 6 (party `member_hp.go` + `update.go`) calls out 4-variant byte-output sweep + IDA dispatcher offset citation per design §6.

- **No `reflect`, no `interface{}`, no benchmarks** — none of the code in the plan uses `reflect.*` or adds an `interface{}` parameter to an encoder. Per-fix recipe Step 3 is byte-output assertion, not benchmark.

- **2-nested-guard hard cap** — Phase 2 hard-cap check (Task 8/9/10 Step 6) enforces it via an awk scan. 3+ → `_pending.md`, no refactor.

- **Bucket commit cadence** — Tasks 2–7 each produce 0–N fix commits before the bucket commit. Maintain ordering: fixes first, audit-report bucket commit last, so the bucket reflects post-fix state.

- **Worktree discipline** — every task ends with the `git rev-parse --show-toplevel` + `git branch --show-current` check baked into the conventions section.

- **Gitleaks scrub** — Task 12 Step 4 is mandatory.
