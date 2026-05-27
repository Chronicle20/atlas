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
- **No gitleaks bait.** Absolute paths like `/home/<user>/` must not appear in any social-domain audit report (`docs/packets/audits/<version>/{Guild,Party,Buddy,Messenger,Note,Chat}*.md`). Pre-PR check is mandatory (Task 13 Step 4 — flat layout, prefix-glob).
- **Tracking sub-tasks vs PR-sized commits.** Phase 1 sub-tasks (Tasks 2–7) and Phase 2 sub-tasks (Tasks 8–10) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every packet in its bucket has a verdict in `SUMMARY.md` and every ❌ has either a fix commit on this branch or a `_pending.md` row.
- **`_pending.md` row grouping.** Group deferrals by *cause*, not by *file*. One row per limitation with a sub-list of affected files (design §4.1, §9). One row per bare-handler family, not per handler.

---

## Pipeline wiring & report layout — corrections applied at execute-time

The plan as originally written contained three inaccuracies about how `tools/packet-audit` actually produces social-domain reports. These corrections are *binding* for every audit invocation in Phase 1 / Phase 2 / Phase 3 below. Where the literal text of a downstream task disagrees, follow this section.

### A. `candidatesFromFName` wiring is a prerequisite, not an automatic step

`tools/packet-audit/cmd/run.go` (`Run` function, lines 96–110) only audits an FName when **both** of these are true:

1. The FName has an entry in `docs/packets/ida-exports/gms_v95.json` (or the appropriate per-version export).
2. The hardcoded switch statement `candidatesFromFName(fname)` in `tools/packet-audit/cmd/run.go` returns a non-empty `[]candidate` that maps the FName to an atlas writer/handler name **with `pkg: "<sub-domain>"`** for social packets (mirroring the `pkg: "monster"`/`"pet"`/`"drop"`/`"reactor"` rows that task-065 added).

At branch-base, `candidatesFromFName` has **zero social-domain entries**. Running the audit CLI with the plan's original command therefore produces zero new social reports — the tool silently iterates an empty candidate set.

**Implication for every Phase 1 sub-task:** before "Run the audit", the implementer must:

1. Identify the FName for each packet in the bucket (use `docs/packets/MapleStory Ops - ClientBound.csv` / `ServerBound.csv` — the `FName` column).
2. Decompile each FName via `mcp__ida-pro__get_function_by_name` + `mcp__ida-pro__decompile_function` and append wire layouts to `gms_v95.json` in the existing `Decode1/2/4/Str/Buffer/Loop` schema.
3. Add one `candidatesFromFName` switch case per packet in `tools/packet-audit/cmd/run.go`, using `pkg: "<sub-domain>"` so `qualifiedWriterName` produces a domain-prefixed report filename (see §C below).
4. Run `go test -race ./tools/packet-audit/...` — confirm green (no regression in the existing fixture suite, including the social-sub-struct fixture from Task 1).
5. Commit the wiring + IDA-export changes in **one commit per sub-domain** (commit message: `feat(packet-audit): wire <domain> FName candidates + v95 IDA exports`).
6. Then proceed with the per-sub-task "Run the audit" step.

### B. The `--output` flag value

`tools/packet-audit/cmd/run.go:42` constructs the report directory as `filepath.Join(opts.Output, "<region>_v<major>")`. So:

- Plan-original (wrong): `--output docs/packets/audits/gms_v95` → produces `docs/packets/audits/gms_v95/gms_v95/*.md`.
- Correct: `--output docs/packets/audits` → produces `docs/packets/audits/gms_v95/*.md`.

Apply the corrected value to **every** audit-CLI invocation downstream (Phase 1 preamble, Tasks 8/9/10, Task 11). Per-version, the flag stays `--output docs/packets/audits` — the tool derives `gms_v83/`, `gms_v87/`, `gms_v95/`, `jms_v185/` itself from the `--template` file's `Region`/`MajorVersion`.

### C. Report layout is flat, not per-sub-domain

The tool writes one report per writer struct directly under `docs/packets/audits/<region>_v<major>/`. There are no `note/`, `buddy/`, etc. subdirectories. `qualifiedWriterName(pkg, name)` returns `Titlecase(pkg) + name` (e.g. `MonsterSpawn` / `NoteOperation` / `BuddyError`), which becomes the report filename. So:

- Plan-original (wrong): reports under `docs/packets/audits/gms_v95/note/Display.md`, `docs/packets/audits/gms_v95/note/Operation.md`, etc.
- Correct: `docs/packets/audits/gms_v95/NoteDisplay.md`, `docs/packets/audits/gms_v95/NoteSendSuccess.md`, `docs/packets/audits/gms_v95/NoteOperation.md` (cb dispatcher), etc.

There is **no cb/sb filename collision** to manage manually — `pkg: "note"` already prefixes both directions, and each clientbound/serverbound file in a sub-domain holds a distinct top-level struct (verified by `grep ^type` across `libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{clientbound,serverbound}/*.go`).

### D. Per-sub-task exit gate adjustments

Wherever a downstream sub-task says `ls docs/packets/audits/gms_v95/<domain>/*.md | wc -l`, read it as:

```bash
ls docs/packets/audits/gms_v95/<TitleDomain>*.md 2>/dev/null | wc -l
```

(e.g. `Note*.md`, `Buddy*.md`, `Messenger*.md`, `Chat*.md`, `Party*.md`, `Guild*.md`.) The expected count is the **struct-level audit-report count** for that sub-domain, which may differ from the file count if a single source file (e.g. `note/clientbound/operation.go`) declares multiple writer structs. Determine the expected count by enumerating top-level `type X struct` declarations under `libs/atlas-packet/<domain>/{clientbound,serverbound}/` (excluding `_test.go`) **and** intersecting with the FName set the implementer wired up in step A.5. The `SUMMARY.md` `grep -c` checks similarly use the writer name (e.g. `grep -c "atlas-packet/note/" SUMMARY.md`) — these survive because `SUMMARY.md` records the atlas-file path, not the report filename.

### E. Bucket commit composition

The plan's bucket-commit recipe lists:

```bash
git add docs/packets/audits/gms_v95/<domain>/ \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
```

Replace the first path with a glob over the per-sub-domain prefixed reports:

```bash
git add docs/packets/audits/gms_v95/<TitleDomain>*.md \
        docs/packets/audits/gms_v95/<TitleDomain>*.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
```

The `gms_v95.json` append is already done in step A.2 (often as part of the wiring commit); if it lands in the wiring commit instead of the bucket commit, omit it from the bucket-commit add list. Only one of the two commits should touch `gms_v95.json` per sub-task — pick whichever flow is cleaner (default: wiring commit owns the IDA-export append; bucket commit only adds reports + SUMMARY).

### F. The wiring-commit budget

Each Phase 1 sub-task now produces:

- **1 wiring commit** — `candidatesFromFName` additions + `gms_v95.json` appends.
- **0–N per-fix commits** — each ❌ atlas-wire-bug fix (one fix = one commit) per the existing per-fix recipe.
- **0–N deferral commits** — each `_pending.md` row landed (one row = one commit, or grouped if 2+ rows land together for a clear reason).
- **1 bucket commit** — audit reports + SUMMARY.md (and optionally the IDA-export append if it didn't go in the wiring commit).

Total commits per sub-task therefore: 1 wiring + 0–N fixes + 0–N deferrals + 1 bucket. The plan-original budgeted only 0–N fixes + 1 bucket; budget for one extra commit per sub-task.

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
  --output           docs/packets/audits
```

(`--output` is the *base* path; the tool appends `/gms_v95/` itself — see "Pipeline wiring & report layout" §B.) It produces per-packet reports under `docs/packets/audits/gms_v95/<TitleDomain><Name>.{md,json}` (flat layout, prefixed by `qualifiedWriterName`) and updates `docs/packets/audits/gms_v95/SUMMARY.md`. Run once per sub-task; commit the report files alongside the fix commits.

**Per §A above:** before invoking the audit CLI, the implementer must add `candidatesFromFName` switch cases for every social FName in the sub-task bucket and append IDA decompiles to `gms_v95.json`. Without that wiring the CLI produces zero new reports for this domain.

Before starting Phase 1, the user must have v95 IDA loaded so MCP `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the audit report bucket commit.

### Verdict triage rules (apply within every sub-task)

For each report under `docs/packets/audits/gms_v95/<TitleDomain>*.md` (per "Pipeline wiring & report layout" §C above — reports are flat, prefixed by `qualifiedWriterName`):

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

After all per-fix commits land, commit the audit reports + SUMMARY in one bucket commit (the IDA-export append landed in the wiring commit per §A above; only re-add it here if a per-fix recipe extended it beyond the wiring-commit snapshot):

```bash
git add docs/packets/audits/gms_v95/<TitleDomain>*.md \
        docs/packets/audits/gms_v95/<TitleDomain>*.json \
        docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(<domain>): v95 audit (<n> packets)"
```

### Exit gate (every Phase 1 sub-task)

```bash
ls docs/packets/audits/gms_v95/<TitleDomain>*.md 2>/dev/null | wc -l
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
- Modify: `tools/packet-audit/cmd/run.go` (add `candidatesFromFName` cases for note FNames — see §A)
- Append: `docs/packets/ida-exports/gms_v95.json` (note IDA decompiles — see §A)
- Modify (per fix): `libs/atlas-packet/note/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/note/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/Note*.{md,json}` (created by audit run; flat layout per §C)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v95. If not, ask the user to swap before continuing.

- [ ] **Step 1b: Wire candidatesFromFName + populate gms_v95.json (per §A).**

For each note struct (cb: `Display`, `SendSuccess`, `SendError`, `Refresh`; sb: `Operation` dispatcher, `OperationDiscard`, `OperationSend`) — note structs without an FName in the CSV (e.g. pure body-decorator helpers) are skipped:

1. Look up the FName in `docs/packets/MapleStory Ops - {Client,Server}Bound.csv` (search keywords `MEMO`, `NOTE`).
2. `mcp__ida-pro__get_function_by_name("<FName>")` → `mcp__ida-pro__decompile_function(<addr>)`.
3. Append an entry to `docs/packets/ida-exports/gms_v95.json` in the existing `Decode1/2/4/Str/Buffer/Loop` schema.
4. Add a switch case to `candidatesFromFName` in `tools/packet-audit/cmd/run.go`: `return []candidate{{name: "<StructName>", pkg: "note", dir: csvpkg.Dir{Client,Server}bound}}`.

After all note FNames are wired:

```
go test -race ./tools/packet-audit/...
go vet ./tools/packet-audit/...
```

Both must be clean. Commit:

```bash
git add tools/packet-audit/cmd/run.go docs/packets/ida-exports/gms_v95.json
git commit -m "feat(packet-audit): wire note FName candidates + v95 IDA exports"
```

- [ ] **Step 2: Run the audit (full pipeline; tool now emits `Note*.{md,json}` per §C).**

Use the audit command from the Phase 1 preamble. Expected runtime: ≤ 60 s.

- [ ] **Step 3: Triage each `Note*.md` report.**

Apply the verdict triage rules from the Phase 1 preamble. `note/serverbound/operation.go` is the operation-dispatcher (op-byte only) — record an OP-FAMILY-note row in `_pending.md` if any ❌ triggers it; verdict ⚠️ on the file. `note/clientbound/operation_body.go` is a body-decorator file (no top-level audit-target struct); skip.

- [ ] **Step 4: Per-fix loop — for each ❌, follow the per-fix recipe (5 sub-steps) from the Phase 1 preamble.**

- [ ] **Step 5: Bucket commit (audit reports + SUMMARY).**

```bash
git add docs/packets/audits/gms_v95/Note*.md \
        docs/packets/audits/gms_v95/Note*.json \
        docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(note): v95 audit"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Note*.md 2>/dev/null | wc -l
```

Expected: equals the count of note FNames wired in Step 1b. There is no cb/sb collision — `qualifiedWriterName` already prefixes both directions with `Note`, and each cb/sb file holds a distinct struct.

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
- Modify: `tools/packet-audit/cmd/run.go` (add `candidatesFromFName` cases for buddy FNames — see §A)
- Append: `docs/packets/ida-exports/gms_v95.json` (buddy IDA decompiles — see §A)
- Modify (per fix): `libs/atlas-packet/buddy/clientbound/<pkt>.go` + matching `_test.go`
- Modify (per fix): `libs/atlas-packet/buddy/serverbound/<pkt>.go` + matching `_test.go`
- Modify (per template fix): `services/atlas-configurations/seed-data/templates/template_gms_*_1.json`
- Modify: `docs/packets/audits/gms_v95/Buddy*.{md,json}` (audit-generated; flat layout per §C)
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`
- Append (if deferrals): `docs/packets/ida-exports/_pending.md`

- [ ] **Step 1: Confirm v95 IDA is still loaded** (`mcp__ida-pro__get_metadata`).
- [ ] **Step 1b: Wire candidatesFromFName + gms_v95.json entries (per §A) for buddy structs. Commit as `feat(packet-audit): wire buddy FName candidates + v95 IDA exports`.**
- [ ] **Step 2: Run the audit (Phase 1 preamble command). Tool emits `Buddy*.{md,json}` per §C.**
- [ ] **Step 3: Triage each `Buddy*.md` report per Phase 1 preamble triage rules.** `error.go` → likely sub-op enum row in `_pending.md` (group under Task 1's deferral heading).
- [ ] **Step 4: Per-fix loop — Phase 1 per-fix recipe.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/Buddy*.md \
        docs/packets/audits/gms_v95/Buddy*.json \
        docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(buddy): v95 audit"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Buddy*.md 2>/dev/null | wc -l   # Expected: equals buddy FName count wired in Step 1b
grep -c "atlas-packet/buddy/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: same count
```

---

### Task 4: Phase 1c — messenger (13 packets)

**Packets — clientbound (8):** `add.go`, `chat.go`, `invite_declined.go`, `invite_sent.go`, `join.go`, `remove.go`, `request_invite.go`, `update.go`
**Packets — serverbound (5):** `operation.go`, `operation_answer_invite.go`, `operation_chat.go`, `operation_decline_invite.go`, `operation_invite.go`

`messenger/clientbound/add.go` and `update.go` consume `model.Avatar` (already registered, design §7). `messenger/serverbound/operation.go` is the dispatcher (op-byte only) — OP-FAMILY-messenger row in `_pending.md`; verdict ⚠️ on the file. `messenger/clientbound/invite_declined.go` carries decline-reason sub-op enum — `_pending.md` row.

**Files:**
- Same as Task 3 substituting `buddy` → `messenger`.

- [ ] **Step 1: Confirm v95 IDA is loaded.**
- [ ] **Step 1b: Wire candidatesFromFName + gms_v95.json entries (per §A) for messenger structs. Commit as `feat(packet-audit): wire messenger FName candidates + v95 IDA exports`.**
- [ ] **Step 2: Run the audit (tool emits `Messenger*.{md,json}` per §C).**
- [ ] **Step 3: Triage each `Messenger*.md` report.** OP-FAMILY-messenger row in `_pending.md` for `serverbound/operation.go`. Sub-op enum row covers `invite_declined.go` decline reasons.
- [ ] **Step 4: Per-fix loop.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/Messenger*.md \
        docs/packets/audits/gms_v95/Messenger*.json \
        docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(messenger): v95 audit"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Messenger*.md 2>/dev/null | wc -l   # Expected: equals messenger FName count wired in Step 1b
grep -c "atlas-packet/messenger/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: same count
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
- [ ] **Step 1b: Wire candidatesFromFName + gms_v95.json entries (per §A) for chat structs. Commit as `feat(packet-audit): wire chat FName candidates + v95 IDA exports`.**
- [ ] **Step 2: Run the audit (tool emits `Chat*.{md,json}` per §C).**
- [ ] **Step 3: Per-file sub-mode classification.** Apply the case-1/case-2 dichotomy above to each of the 8 chat files. For case-2 files, append a single bullet to the existing `_pending.md` heading "Sub-op enum / sub-struct deferrals — social domain (task-066)" naming all parameterised chat files in one row:

  ```markdown
  - **Chat sub-mode enum modelling** — `chat/{clientbound,serverbound}/<file>.go` accept a `mode` byte parameter (NORMAL=0, WHISPER, MEGAPHONE, SMEGA, ITEM_POP, AVATAR_MEGAPHONE, HEART, SKULL). The audit pipeline cannot statically check the value space; verdict ⚠️ on each file. Sub-mode mapping captured in IDA dispatcher case-statements; cross-version verification deferred. Affected files: <list>.
  ```

  Cap: if the parameterised sub-mode space exceeds 5 distinct values per file (per design §4), defer enum verification entirely; otherwise the audit-report footer for that file inlines the verdict.

- [ ] **Step 4: Per-fix loop.** For case-1 files (hard-coded sub-mode literal), if the literal disagrees with IDA's dispatcher case-statement value, follow the Phase 1 per-fix recipe.
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/Chat*.md \
        docs/packets/audits/gms_v95/Chat*.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(chat): v95 audit + sub-mode enum deferral"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Chat*.md 2>/dev/null | wc -l   # Expected: equals chat FName count wired in Step 1b
grep -c "atlas-packet/chat/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: same count
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
- [ ] **Step 1b: Wire candidatesFromFName + gms_v95.json entries (per §A) for party structs. Commit as `feat(packet-audit): wire party FName candidates + v95 IDA exports`.**
- [ ] **Step 2: Run the audit (tool emits `Party*.{md,json}` per §C).**
- [ ] **Step 3: Triage each `Party*.md` report. Add OP-FAMILY-party + party operation-result enum rows to `_pending.md` if not already present.**
- [ ] **Step 4: Per-fix loop. Hot-path discipline (member_hp/update) — 4-variant byte-output sweep is mandatory; cite IDA dispatcher offset in the fix comment.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/Party*.md \
        docs/packets/audits/gms_v95/Party*.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(party): v95 audit"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Party*.md 2>/dev/null | wc -l   # Expected: equals party FName count wired in Step 1b
grep -c "atlas-packet/party/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: same count
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
- [ ] **Step 1b: Wire candidatesFromFName + gms_v95.json entries (per §A) for guild structs. Note: 19 serverbound packets — this is the largest wiring step. Commit as `feat(packet-audit): wire guild FName candidates + v95 IDA exports`.**
- [ ] **Step 2: Run the audit (tool emits `Guild*.{md,json}` per §C).**
- [ ] **Step 3: Triage each `Guild*.md` report. Add OP-FAMILY-guild and OP-FAMILY-bbs rows to `_pending.md` (one row per family). Add guild operation-result enum row if not already present from earlier sub-tasks.**

  Sanity-check the dispatcher reading: re-read `libs/atlas-packet/guild/serverbound/operation.go` and `bbs_operation.go`. If either dispatcher's `Encode` body emits more than one `WriteByte(...)` (i.e. carries payload beyond the op byte), design §10's "Low likelihood" risk has materialised — record the actual dispatcher shape in the audit report instead of treating as op-byte-only, and audit accordingly. Do not rewrite the design.

- [ ] **Step 4: Per-fix loop. Pay special attention to `bbs.go` clientbound — it author/replyer entries are likely a sub-struct list; verify against `model.GuildMember` (Phase 0 registration) or an inline struct.**
- [ ] **Step 5: Bucket commit.**

```bash
git add docs/packets/audits/gms_v95/Guild*.md \
        docs/packets/audits/gms_v95/Guild*.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(guild): v95 audit"
```

- [ ] **Step 6: Exit gate.**

```bash
ls docs/packets/audits/gms_v95/Guild*.md 2>/dev/null | wc -l   # Expected: equals guild FName count wired in Step 1b
grep -c "atlas-packet/guild/" docs/packets/audits/gms_v95/SUMMARY.md  # Expected: same count
```

**Phase 1 exit:** `SUMMARY.md` contains rows for every social-domain FName the implementer wired across Tasks 2–7 (originally targeted as 76 from a file-count survey; the actual struct/FName-level count may differ — see §A & §D). Every ❌ has a fix commit on this branch OR a `_pending.md` row.

```bash
grep -c "atlas-packet/\(guild\|party\|buddy\|messenger\|note\|chat\)/" docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: equals the sum of FName counts from Tasks 2–7 Step 1b commits. If 76 was the original target and the actual is below, this gap should be reconciled — either by adding the missing FName wirings or by documenting why specific structs have no corresponding IDA function (e.g. body-decorator helpers like `note/clientbound/operation_body.go`). If less than originally targeted, identify the missing files via:

```bash
diff <(ls libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{clientbound,serverbound}/*.go 2>/dev/null | grep -v _test.go | xargs -n1 basename | sort -u) \
     <(grep -oE "atlas-packet/(guild|party|buddy|messenger|note|chat)/(clientbound|serverbound)/[a-z_]+\.go" docs/packets/audits/gms_v95/SUMMARY.md | xargs -n1 basename | sort -u)
```

Investigate every missing file before exiting Phase 1.

---

## Phase 2 — Cross-version pass (v83 → v87 → JMS v185)

Three tracking sub-tasks (Tasks 8–10), one per version. Each requires a user-driven IDA binary swap before starting (PRD §4.6, design §8 Phase 2). A sub-task is "done" when:

- `docs/packets/ida-exports/gms_{v83,v87}.json` or `gms_jms_185.json` contains social-domain entries for every FName resolved during Phase 1.
- The audit has been re-run against the version's template + IDA export, producing flat reports under `docs/packets/audits/<version>/<TitleDomain>*.md` (per §C above).
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
- Create or modify: `docs/packets/audits/gms_v83/<TitleDomain>*.{md,json}` (tool creates the directory; flat layout per §C)
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
  --output           docs/packets/audits
```

(`--output` is the *base* path; the tool derives `gms_v83/` from the template's Region/MajorVersion — see "Pipeline wiring & report layout" §B.)

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

Final bucket commit (flat-layout glob — pick up Note/Buddy/Messenger/Chat/Party/Guild prefixed reports the audit run produced):

```bash
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/{Note,Buddy,Messenger,Chat,Party,Guild}*.md \
        docs/packets/audits/gms_v83/{Note,Buddy,Messenger,Chat,Party,Guild}*.json \
        docs/packets/audits/gms_v83/SUMMARY.md
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
- Create or modify: `docs/packets/audits/gms_v87/<TitleDomain>*.{md,json}` (flat layout per §C)
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
  --output           docs/packets/audits
```

(`--output` is the *base* path; the tool derives `gms_v87/` from the template's Region/MajorVersion.)

- [ ] **Step 4: Triage divergences** (per Task 8 Step 4).
- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_v87.json \
        docs/packets/audits/gms_v87/{Note,Buddy,Messenger,Chat,Party,Guild}*.md \
        docs/packets/audits/gms_v87/{Note,Buddy,Messenger,Chat,Party,Guild}*.json \
        docs/packets/audits/gms_v87/SUMMARY.md
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
- Create or modify: `docs/packets/audits/jms_v185/<TitleDomain>*.{md,json}` (flat layout per §C)
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
  --output           docs/packets/audits
```

(`--output` is the *base* path; the tool derives `jms_v185/` from the template's Region/MajorVersion.)

- [ ] **Step 4: Triage per design §8 / task-028 §7.1 in-scope rules:**
  - In scope: atlas-packet writes bytes the JMS client decodes wrong.
  - Out of scope: JMS-specific feature the service doesn't wire through (JMS-only alliance tier, JMS BBS structure differences) → `_pending.md` row + sibling-task suggestion.
  - In scope: width mismatch on a field both versions decode.
  - Out of scope: JMS template opcode wrong when v95 is right → fix the template, atlas-packet untouched.

- [ ] **Step 5: Per-fix commits + bucket commit.**

```bash
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/{Note,Buddy,Messenger,Chat,Party,Guild}*.md \
        docs/packets/audits/jms_v185/{Note,Buddy,Messenger,Chat,Party,Guild}*.json \
        docs/packets/audits/jms_v185/SUMMARY.md
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
  --output           docs/packets/audits
```

(`--output` is the *base* path; the tool derives `gms_v95/` from the template.)

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

Reports are flat (per §C) — there are no per-sub-domain subdirs. Filter by the `qualifiedWriterName` prefixes instead:

```bash
for d in gms_v95 gms_v83 gms_v87 jms_v185; do
    for prefix in Note Buddy Messenger Chat Party Guild; do
        grep -l '/home/' docs/packets/audits/$d/${prefix}*.md 2>/dev/null
    done
done
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
