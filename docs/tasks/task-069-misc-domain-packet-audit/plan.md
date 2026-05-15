# Misc-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline established by task-027 (login) and refined by tasks 028 / 065-068 to the ~21 misc-domain packets in `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest,tool}/`, ship wire-bug + template fixes against GMS v95 IDA, re-verify across v83 / v87 / JMS v185, and produce `docs/packets/audits/gms_v95/TOTAL.md` — the canonical cross-task ledger covering 027 + 028 + 065 + 066 + 067 + 068 + 069.

**Architecture:** Phase 0 re-runs the existing v95 audit unchanged to confirm the 28 login-domain rows are byte-identical to pre-task state. Phase 1 lands three TypeRegistry fixtures for the body sub-structs (`fame/response_body`, `ui/ui_open_body`, `merchant/operation_body`). Phase 2 audits the misc domain in 10 sub-phases ordered easy → hard: tool → stat → channel → ui → fame → merchant → quest → account → socket → `_pending.md` sweep. Phase 3 re-runs the audit against v83 / v87 / JMS v185. Phase 4 ships `TOTAL.md`, `post-phase-b.md`, runs the full verification matrix, and dispatches code review. No analyzer code changes are anticipated; if one is forced, STOP and escalate per design §1.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast` for AST analysis), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer for round-trip tests, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies; this task ships audit reports + targeted code/template fixes only.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-069-misc-domain-packet-audit/` on branch `task-069-misc-domain-packet-audit`. Before *every* commit run `git rev-parse --show-toplevel` and `git branch --show-current`; if either disagrees, STOP.
- **TDD cadence (registry tests).** Test first → run-to-fail → minimal addition → run-to-pass → commit. Registry additions land in `tools/packet-audit/internal/atlaspacket/registry.go` and `registry_test.go` only — no analyzer changes.
- **TDD cadence (encoder fixes).** 4-variant `pt.Variants` sweep test first → fix encoder → run-to-pass → commit. The sweep covers GMS v28 / v83 / v95 + JMS v185 via the helper at `libs/atlas-packet/test/context.go:18-25`.
- **Verification cadence (registry changes).** `go test -race ./tools/packet-audit/...` clean before commit. Then re-run the v95 audit and confirm the login-domain SUMMARY rows are byte-identical (Phase 0 gate; repeated after every Phase 1 commit).
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with the 4-variant sweep test using the project Builder pattern — NO `*_testhelpers.go` files.
- **No `reflect`, no new `interface{}` params, no benchmarks** in atlas-packet edits.
- **No analyzer changes.** Per design §1, this task does not touch `tools/packet-audit/internal/atlaspacket/analyzer.go`. If the audit panics or surfaces a new cycle, STOP and spin a sibling task.
- **Cross-domain regression gate.** Every Phase 1 registry commit re-runs the audit and confirms the 28 login-domain rows are byte-identical to the pre-task baseline. Any drift → STOP, roll back the registry entry, investigate.
- **Tracking sub-tasks vs PR-sized commits.** Phase 2 sub-tasks (Tasks 5–14) and Phase 3 sub-tasks (Tasks 15–17) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every packet in its bucket has a verdict and every ❌ has either a fix commit or a `_pending.md` row.
- **Audit-report ack footer policy.** The ack footer (`Ack: misc-audit Phase 2<sub> on YYYY-MM-DD`) is the LAST line written to each report. If a re-run is needed for a single report, `git checkout HEAD -- docs/packets/audits/gms_v95/<Report>.md` before re-execution.
- **Nesting cap.** Every misc-domain encoder stays under **2 nested region/version guards**. 3+ → STOP, defer to `_pending.md`. The world-domain `set_field.go` 3-deep carve-out from task-068 does NOT apply to misc.
- **Socket extra caution.** Every socket fix (Task 13) ships with 4-variant tests AND a manual re-verification against all six version templates (`template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json`). `atlas-login` + `atlas-channel` build clean before the bucket commit. Per design §4.
- **Quest cross-task discipline.** Before touching any file under `libs/atlas-packet/quest/`, run `git log --oneline -- libs/atlas-packet/quest/` on main and read task-014, task-015, task-023 commits. Existing `Region/MajorVersion` gates are load-bearing — do not widen/narrow without IDA evidence from the same version context the prior task used. Per design §6.
- **Audit pipeline command.** Always invoked from worktree root with relative paths. The full command is in `context.md` §"Audit pipeline command".
- **Pre-PR rebase.** Squash repetitive `audit(misc):` audit-report commits per sub-phase before opening PR; fix commits and registry commits stay individual.

---

## Phase 0 — Regression baseline (gate)

One task. Exit when the 28 login-domain SUMMARY rows are byte-identical to pre-task state.

### Task 1: Re-run v95 audit, confirm login-domain verdicts unchanged

**Files:**
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (only if pipeline output changes; expected to be a no-op)
- Modify: any per-packet `.md`/`.json` whose mtime updates (expected: pipeline is deterministic; no semantic diffs)

- [ ] **Step 1: Snapshot the prior SUMMARY**

```
git show HEAD:docs/packets/audits/gms_v95/SUMMARY.md > /tmp/summary-pre-task069.md
wc -l /tmp/summary-pre-task069.md
grep -c '❌' /tmp/summary-pre-task069.md
grep -c '⚠️' /tmp/summary-pre-task069.md
grep -c '✅' /tmp/summary-pre-task069.md
```

Expected: 32 lines (header + 4 blanks/dividers + 28 rows). ❌ 1, ⚠️ 0, ✅ 27 (per context.md baseline). If counts differ, the branch has drifted from the documented baseline — investigate before proceeding.

- [ ] **Step 2: Run the audit unchanged**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Expected runtime: ≤ 30 s. Exit code 1 is normal because `CharacterList` is ❌ in the baseline.

- [ ] **Step 3: Diff SUMMARY against snapshot**

```
diff /tmp/summary-pre-task069.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff. Any login-row verdict change → STOP. Investigate the analyzer state vs the snapshot before continuing.

- [ ] **Step 4: Commit (no-op or audit-only)**

If `git status --short docs/packets/audits/gms_v95/` is empty:

```
git commit --allow-empty -m "audit(misc): phase-0 regression baseline — login verdicts unchanged"
```

If files changed without semantic drift (whitespace, timestamps):

```
git add docs/packets/audits/gms_v95/
git commit -m "audit(misc): phase-0 regression baseline — refresh login reports"
```

- [ ] **Step 5: Exit gate**

Verdict counts in `docs/packets/audits/gms_v95/SUMMARY.md` match the snapshot. Branch state clean. Proceed to Phase 1.

---

## Phase 1 — TypeRegistry sub-struct coverage

Three tasks, one per body file. Each registry addition lands with a fixture in `registry_test.go` and a cross-domain regression check that login SUMMARY rows stay byte-identical. Quest/socket/channel sub-struct candidates from design §5 stay deferred until Phase 2 surfaces evidence — do not pre-emptively register.

### Task 2: Registry fixture for `FameResponseBody`

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Identify the symbol**

```
grep -nE '^type .*Body.*struct|^func \(.*Body\) .*Encode' libs/atlas-packet/fame/response_body.go
```

Record the type name (likely `ResponseBody` or `FameResponseBody`). The file lives in the package-root of `libs/atlas-packet/fame/` rather than under `clientbound/` because it's shared between `clientbound/response.go`'s three response variants.

- [ ] **Step 2: Write the fixture**

Append to `tools/packet-audit/internal/atlaspacket/registry_test.go`:

```go
func TestRegistryRegistersFameResponseBody(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    // The body type defined in libs/atlas-packet/fame/response_body.go.
    // Symbol name resolved from Step 1; replace if needed.
    name := "ResponseBody"
    if !reg.HasType(name) {
        t.Fatalf("registry missing type %s", name)
    }
    calls, ok := reg.Calls(name)
    if !ok || len(calls) == 0 {
        t.Fatalf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
    }
}
```

If the actual symbol name differs (e.g. `FameResponseBody` to disambiguate across packages), substitute in `name :=`.

- [ ] **Step 3: Run the fixture**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersFameResponseBody -v
```

Expected: PASS. If FAIL with "registry missing type", the registry walker may be stripping pointer receivers; document the gap in a one-line `t.Skip(...)` comment and add a `_pending.md` row for the body file. Do NOT modify `registry.go` — registry/analyzer code changes are out of scope for this task.

- [ ] **Step 4: Cross-domain regression**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
diff /tmp/summary-pre-task069.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff. If any login row flips, STOP — registry fixture-only commits should not cascade.

- [ ] **Step 5: Commit**

```
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert FameResponseBody registry coverage"
```

---

### Task 3: Registry fixture for `UiOpenBody`

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Identify the symbol**

```
grep -nE '^type .*Body.*struct|^func \(.*Body\) .*Encode' libs/atlas-packet/ui/ui_open_body.go
```

Record the type name.

- [ ] **Step 2: Write the fixture**

Append to `registry_test.go`:

```go
func TestRegistryRegistersUiOpenBody(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    name := "UiOpenBody"
    if !reg.HasType(name) {
        t.Fatalf("registry missing type %s", name)
    }
    calls, ok := reg.Calls(name)
    if !ok || len(calls) == 0 {
        t.Fatalf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
    }
}
```

- [ ] **Step 3: Run the fixture**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersUiOpenBody -v
```

Expected: PASS. STOP semantics same as Task 2 step 3.

- [ ] **Step 4: Cross-domain regression**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
diff /tmp/summary-pre-task069.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff.

- [ ] **Step 5: Commit**

```
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert UiOpenBody registry coverage"
```

---

### Task 4: Registry fixture for `MerchantOperationBody`

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Identify the symbol**

```
grep -nE '^type .*Body.*struct|^func \(.*Body\) .*Encode' libs/atlas-packet/merchant/operation_body.go
```

The body is referenced by BOTH directions (`merchant/clientbound/operation.go` AND `merchant/serverbound/operation.go`). The registry entry must be usable from each direction's Encode paths — a single fixture asserting the type exists and produces calls is sufficient.

- [ ] **Step 2: Write the fixture**

Append to `registry_test.go`:

```go
func TestRegistryRegistersMerchantOperationBody(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    name := "OperationBody"
    if !reg.HasType(name) {
        t.Fatalf("registry missing type %s", name)
    }
    calls, ok := reg.Calls(name)
    if !ok || len(calls) == 0 {
        t.Fatalf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
    }
}
```

If `OperationBody` collides with another package's symbol of the same name (the registry is symbol-name-keyed), substitute the package-qualified name surfaced by the registry walker.

- [ ] **Step 3: Run the fixture**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersMerchantOperationBody -v
```

Expected: PASS. If FAIL with "symbol collision", the registry needs disambiguation logic that's out of scope here — `t.Skip(...)` with a `_pending.md` row instead.

- [ ] **Step 4: Cross-domain regression**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
diff /tmp/summary-pre-task069.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff.

- [ ] **Step 5: Commit**

```
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert MerchantOperationBody registry coverage"
```

---

## Phase 2 — v95 misc audit

Ten tracking sub-tasks (Tasks 5–14, sub-phases 2a–2j). Each sub-task is a *tracking* unit, not a single PR commit:

1. Add the misc-domain operation entries to `template_gms_95_1.json` if missing (one-shot for the first packet in each sub-domain).
2. Run the audit against the sub-task's packet bucket.
3. Triage each report: ✅ (no fix needed), ⚠️ (tolerable mismatch — annotate report), ❌ (real wire bug OR template drift OR analyzer descent gap).
4. Ship **one fix commit per ❌** with a 4-variant test sweep and an IDA citation in the commit message.
5. Each `_pending.md` deferral lands as a separate row + a one-line commit.
6. Bucket commit closes the sub-task — audit reports + SUMMARY + IDA-export updates batched.

The audit command for all of Phase 2 is the same (see `context.md` §"Audit pipeline command"). Run it once per sub-task after each fix commit; commit reports alongside the bucket commit.

Before starting Phase 2, the user must have v95 IDA loaded so `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the bucket audit reports.

Sub-phase ordering rationale (per design §11 Phase 2): tool → stat → channel → ui → fame → merchant → quest → account → socket — easy wins first, cross-task / dispatcher-offset / critical-path packets last when context is maximal.

---

### Task 5: Sub-phase 2a — `tool/` confirmation (0 packets)

**Goal:** confirm `libs/atlas-packet/tool/` is utility-only and document the finding. No audit reports, no SUMMARY rows.

- [ ] **Step 1: Enumerate**

```
find libs/atlas-packet/tool -name '*.go' ! -name '*_test.go' | sort
```

Expected output:

```
libs/atlas-packet/tool/uint128.go
```

- [ ] **Step 2: Confirm `uint128.go` is type-only**

```
grep -E '^func ' libs/atlas-packet/tool/uint128.go | head -20
grep -E '\bOperation\(\)|\bEncode\(|\bDecode\(' libs/atlas-packet/tool/uint128.go
```

Expected: no `Operation()` method and no Encode/Decode method matching the atlas-packet writer/handler interface. The file defines a 128-bit unsigned integer utility type used by socket/channel handshake encoders for hash fields.

- [ ] **Step 3: Add `_pending.md` confirmation row**

Edit `docs/packets/ida-exports/_pending.md`. Under a new `## Tool domain — utility-only (task-069)` heading, append:

```markdown
## Tool domain — utility-only (task-069)

`libs/atlas-packet/tool/` contains only `uint128.go` — a 128-bit unsigned
integer utility type consumed by socket/channel handshake encoders for hash
fields. It is NOT a packet domain; zero `Operation()`/`Encode()`/`Decode()`
methods, zero audit rows. Confirmed at audit time via `find … ! -name
'*_test.go'`.

See TOTAL.md for the cross-task coverage matrix where `tool/` is listed
under "no packets; utility-only".
```

- [ ] **Step 4: Commit**

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(misc,tool): confirm utility-only — no packet rows"
```

- [ ] **Step 5: Exit gate**

`docs/packets/ida-exports/_pending.md` contains the tool-domain heading. No new SUMMARY rows. Proceed to Task 6.

---

### Task 6: Sub-phase 2b — `stat/clientbound` bucket (1 file)

**Packets:** `stat/clientbound/changed.go` (`StatChangedWriter` = `"StatChanged"`).

Smallest packet domain — audit-and-go. Expected outcome: 1 verdict row.

- [ ] **Step 1: Add the `StatChanged` writer to the v95 template if missing**

```
grep -E '"writer": *"StatChanged"' services/atlas-configurations/seed-data/templates/template_gms_95_1.json
```

If empty: locate the clientbound stat-change opcode via IDA. Run:

```
mcp__ida-pro__get_function_by_name "CWvsContext::OnStatChanged"
```

(or the equivalent symbol surfaced by the search). Decompile the dispatcher case-statement for the writer's opcode. Append an entry to `template_gms_95_1.json` following the schema documented in task-068 plan §"World-domain template seeding":

```json
{
  "opCode": "0x1F",
  "validator": "InGameValidator",
  "handler": "StatChangedHandle",
  "writer": "StatChanged"
}
```

Use the opcode value from IDA — do not invent. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register StatChanged writer for v95

IDA case-statement at <dispatcher>@<addr>."
```

- [ ] **Step 2: Append the FName to `docs/packets/ida-exports/gms_v95.json`**

Use `mcp__ida-pro__decompile_function` against the resolved address. Append a top-level entry under `functions` matching the existing schema in `gms_v95.json:5-26`:

```json
"CWvsContext::OnStatChanged": {
  "address": "0x...",
  "direction": "clientbound",
  "calls": [
    {"op": "Decode4", "comment": "statMask"},
    ...
  ]
}
```

Do NOT reorder existing login entries.

- [ ] **Step 3: Run the audit (pipeline command from `context.md`)**

Inspect the new `StatChanged.md` and `StatChanged.json` under `docs/packets/audits/gms_v95/`.

- [ ] **Step 4: Triage**

Expected: ✅. If ❌:
- Width/order/missing field → fix `libs/atlas-packet/stat/clientbound/changed.go` with a 4-variant test sweep (Step 5).
- Opcode drift → fix `template_gms_95_1.json`.
- Bare handler (no atlas-packet decoder) → defer to `_pending.md`.

- [ ] **Step 5: For each Atlas wire-bug fix, write a 4-variant round-trip test**

Create or modify `libs/atlas-packet/stat/clientbound/changed_test.go`:

```go
package clientbound

import (
    "testing"

    pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestChangedRoundTrip(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            input := NewChanged(/* fields per Changed constructor */)
            output := Changed{}
            pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
            // assert field-by-field via the getter methods (e.g. output.Mask(), output.Stats(), ...).
        })
    }
}
```

If `Changed` exposes no public constructor, use the Builder pattern conventions in `libs/atlas-packet/login/` for reference (e.g. `NewChanged(...).WithMask(0x01)`).

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/stat/...
```

Expected: clean.

- [ ] **Step 7: Commit each fix individually**

For atlas-packet fixes:

```
git add libs/atlas-packet/stat/clientbound/changed.go libs/atlas-packet/stat/clientbound/changed_test.go
git commit -m "fix(atlas-packet,stat/changed): <one-line summary>

Cites IDA <CWvsContext::OnStatChanged>@<addr>: <one-line evidence>."
```

For template fixes:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "fix(configurations,templates): StatChanged opcode <old>→<new>

IDA case-statement value at <dispatcher>@<addr>."
```

For `_pending.md` deferrals:

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(stat/changed): defer — <one-line reason>"
```

- [ ] **Step 8: Append ack footer and bucket commit**

```
echo "" >> docs/packets/audits/gms_v95/StatChanged.md
echo "Ack: misc-audit Phase 2b on $(date -I)" >> docs/packets/audits/gms_v95/StatChanged.md

git add docs/packets/audits/gms_v95/StatChanged.md docs/packets/audits/gms_v95/StatChanged.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2b stat/clientbound bucket (1 file)"
```

- [ ] **Step 9: Exit gate**

`grep -E 'StatChanged' docs/packets/audits/gms_v95/SUMMARY.md` shows 1 row with a verdict. Any ❌ has a matching fix commit OR `_pending.md` row.

---

### Task 7: Sub-phase 2c — `channel/` bucket (2 files)

**Packets:** `channel/clientbound/change.go` (`ChannelChangeWriter` = `"ChannelChange"`) + `channel/serverbound/channel_change.go` (`ChannelChangeRequestHandle` = `"ChannelChangeHandle"`).

**Dispatcher offset boundary (design §7.2):** the clientbound packet encodes host (4 bytes) + port (2 bytes); the endianness of the host field has historically been a bug surface. Verify against the v95 IDA `CClientSocket::OnSocketDisconnect` (or equivalent) before assuming layout.

- [ ] **Step 1: Add the `ChannelChange` writer + `ChannelChangeHandle` handler entries to the v95 template if missing**

```
grep -E '"writer": *"ChannelChange"|"handler": *"ChannelChangeHandle"' services/atlas-configurations/seed-data/templates/template_gms_95_1.json
```

If empty: resolve opcodes via IDA and append entries. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register Channel{Change,Handle} for v95

IDA case-statements at <dispatcher>@<addr>."
```

- [ ] **Step 2: Append FNames to `gms_v95.json`**

Run `mcp__ida-pro__decompile_function` against each resolved address and add a `Decode*`-op list to `gms_v95.json`. For the clientbound `ChannelChange`, annotate the IPv4 byte-order in the call comment (`{"op": "Decode4", "comment": "host (big-endian per CClientSocket::OnSocketDisconnect)"}`).

- [ ] **Step 3: Run the audit (pipeline command)**

- [ ] **Step 4: Triage**

Triage flavours per Task 6 step 4. Particular attention to:
- Host endianness (clientbound). Atlas uses `WriteInt32` / similar — confirm byte order matches IDA's read.
- Port width (clientbound). Should be `Decode2` per v95 IDA.

- [ ] **Step 5: For each fix, add a 4-variant round-trip test (same shape as Task 6 step 5)**

Tests land in `libs/atlas-packet/channel/clientbound/change_test.go` and/or `libs/atlas-packet/channel/serverbound/channel_change_test.go`.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/channel/...
```

- [ ] **Step 7: Commit each fix individually (format per Task 6 step 7)**

- [ ] **Step 8: Append ack footers + bucket commit**

```
for f in docs/packets/audits/gms_v95/ChannelChange.md docs/packets/audits/gms_v95/ChannelChangeHandle.md; do
  [ -f "$f" ] || continue
  echo "" >> "$f"
  echo "Ack: misc-audit Phase 2c on $(date -I)" >> "$f"
done

git add docs/packets/audits/gms_v95/ChannelChange*.md docs/packets/audits/gms_v95/ChannelChange*.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2c channel bucket (2 files)"
```

(Adjust filenames to actual FNames.)

- [ ] **Step 9: Exit gate**

`grep -E 'Channel(Change|ChangeRequest)' docs/packets/audits/gms_v95/SUMMARY.md` shows 2 rows with verdicts. Every ❌ resolved.

---

### Task 8: Sub-phase 2d — `ui/clientbound` bucket (3 files + body)

**Packets:** `ui/clientbound/{disable,lock,open}.go` (writers: `UiDisable`, `UiLock`, `UiOpen`). Body (registered in Phase 1 Task 3): `ui/ui_open_body.go`.

These are UI notification packets — expect minimal version branching per PRD §9.

- [ ] **Step 1: Add `UiDisable`, `UiLock`, `UiOpen` writer entries to the v95 template if missing**

Check each writer string with `grep -E '"writer": *"Ui<Name>"'`. For each missing entry, resolve opcode via IDA and append. Commit (per writer or batched):

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register Ui{Disable,Lock,Open} writers for v95

IDA case-statements at <dispatcher>@<addr>."
```

- [ ] **Step 2: Append FNames to `gms_v95.json`**

For each writer, locate the matching client-side decoder in IDA (e.g. `CUIDlg::OnUiOpen`) and add a `Decode*`-op list. For `UiOpen` specifically, the call list will descend through `ui/ui_open_body.go` — the registry entry from Phase 1 Task 3 covers that descent.

- [ ] **Step 3: Run the audit**

- [ ] **Step 4: Triage per Task 6 step 4**

- [ ] **Step 5: For each fix, write a 4-variant Encode-side test** (these are clientbound — Atlas only encodes; the round-trip helper covers Encode→Decode parity when a `Decode` exists, otherwise use the Encode-byte-equality form documented in `context.md` §"Test patterns"):

```go
func TestUiOpenEncode(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            got := NewOpen(/* fields */).Encode(testLogger(), ctx)(nil)
            want, _ := hex.DecodeString(v.UiOpenHex)  // per-variant expected bytes from IDA
            if !bytes.Equal(got, want) {
                t.Fatalf("encode mismatch\n got %x\nwant %x", got, want)
            }
        })
    }
}
```

If `pt.Variants` does not yet carry per-test hex fixtures, hard-code four hex strings inline in the test. `testLogger()` follows the helper pattern in `libs/atlas-packet/login/clientbound/*_test.go`.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/ui/...
```

- [ ] **Step 7: Commit each fix individually (format per Task 6 step 7)**

- [ ] **Step 8: Append ack footers + bucket commit**

```
for f in docs/packets/audits/gms_v95/UiDisable.md docs/packets/audits/gms_v95/UiLock.md docs/packets/audits/gms_v95/UiOpen.md; do
  [ -f "$f" ] || continue
  echo "" >> "$f"
  echo "Ack: misc-audit Phase 2d on $(date -I)" >> "$f"
done

git add docs/packets/audits/gms_v95/Ui{Disable,Lock,Open}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2d ui/clientbound bucket (3 files + body)"
```

- [ ] **Step 9: Exit gate**

`grep -E 'Ui(Disable|Lock|Open)' docs/packets/audits/gms_v95/SUMMARY.md` shows 3 rows. Every ❌ resolved.

---

### Task 9: Sub-phase 2e — `fame/` bucket (2 files + body)

**Packets:** `fame/clientbound/response.go` (multi-struct: `ReceiveResponse`, `GiveResponse`, `ErrorResponse` — all return `FameResponseWriter` = `"FameResponse"`) + `fame/serverbound/change.go` (`FameChangeHandle` = `"FameChangeHandle"`). Body (Phase 1 Task 2): `fame/response_body.go`.

The three clientbound structs share one writer FName, so the audit pipeline emits ONE `FameResponse.md` report. Per-variant verdicts go in per-section sub-headings inside the report (mirror task-068 Task 9 multi-struct pattern).

- [ ] **Step 1: Add `FameResponse` + `FameChangeHandle` entries to the v95 template if missing**

Same pattern as Task 8 step 1. Commit per the same convention.

- [ ] **Step 2: Append FNames to `gms_v95.json`**

For `FameResponse`, the analyzer surfaces one flat call list per struct (`ReceiveResponse.Encode`, `GiveResponse.Encode`, `ErrorResponse.Encode`). Annotate each in `gms_v95.json` via the `comment` field if separate FNames are not surfaced by IDA — otherwise add three entries keyed by the per-variant FName.

- [ ] **Step 3: Run the audit**

- [ ] **Step 4: Triage per struct**

For the `FameResponse.md` report, expect ONE row in SUMMARY (worst-of-three verdict). The report body must include per-struct sub-sections (`### ReceiveResponse`, `### GiveResponse`, `### ErrorResponse`) each with its own verdict, atlas-line range citation, and IDA wire comparison. If any sub-section is ❌, ship a fix:
- Atlas wire bug → fix the matching encoder.
- Width/order on the shared `response_body.go` body → fix the body, sweep test all three variants.
- Real `Region/MajorVersion` gate missing → add it (2-deep cap).

For `FameChangeHandle.md` (serverbound), standard per-Task-6 triage.

- [ ] **Step 5: For each fix, add a 4-variant round-trip test (clientbound) or Decode sweep (serverbound)**

Tests land in `libs/atlas-packet/fame/clientbound/response_test.go` (per struct) and `libs/atlas-packet/fame/serverbound/change_test.go`. Per-struct tests are named `TestReceiveResponseRoundTrip`, `TestGiveResponseRoundTrip`, `TestErrorResponseRoundTrip`.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/fame/...
```

- [ ] **Step 7: Commit each fix individually**

For a per-struct fix:

```
git add libs/atlas-packet/fame/clientbound/response.go libs/atlas-packet/fame/clientbound/response_test.go
git commit -m "fix(atlas-packet,fame/response): ReceiveResponse <one-line summary>

Cites IDA <function>@<addr>: <one-line evidence>."
```

For a body-file fix:

```
git add libs/atlas-packet/fame/response_body.go
git commit -m "fix(atlas-packet,fame/response_body): <one-line summary>

Cites IDA <function>@<addr>: <evidence>. All three response variants share the body, so this widens/narrows the shape for ReceiveResponse, GiveResponse, and ErrorResponse."
```

- [ ] **Step 8: Append per-struct breakdown + ack footer, then bucket commit**

After the auto-generated `FameResponse.md` lands, append per-struct sub-sections manually:

```markdown
## Per-struct breakdown

### ReceiveResponse
- **Verdict:** ✅ / ⚠️ / ❌
- **Atlas encoder:** `libs/atlas-packet/fame/clientbound/response.go:<startLine>-<endLine>`
- **Body delegation:** `libs/atlas-packet/fame/response_body.go` (verified via TypeRegistry)
- **IDA wire shape:** `<function>@<addr>` — `Decode1, Decode4, ...`
- **Notes:** <fix summary or "no fix needed">

### GiveResponse
…

### ErrorResponse
…
```

Then:

```
echo "" >> docs/packets/audits/gms_v95/FameResponse.md
echo "Ack: misc-audit Phase 2e on $(date -I); per-struct breakdown verified across 3 variants." >> docs/packets/audits/gms_v95/FameResponse.md
[ -f docs/packets/audits/gms_v95/FameChangeHandle.md ] && echo "" >> docs/packets/audits/gms_v95/FameChangeHandle.md && echo "Ack: misc-audit Phase 2e on $(date -I)" >> docs/packets/audits/gms_v95/FameChangeHandle.md

git add docs/packets/audits/gms_v95/Fame*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2e fame bucket (2 files + body, 3 client structs)"
```

- [ ] **Step 9: Exit gate**

`grep -E 'Fame(Response|ChangeHandle)' docs/packets/audits/gms_v95/SUMMARY.md` shows 2 rows. `FameResponse.md` contains 3 per-struct sub-sections, each with a verdict.

---

### Task 10: Sub-phase 2f — `merchant/` bucket (2 files + body, 7 employee-shop variants)

**Packets:** `merchant/clientbound/operation.go` (7 structs: `OpenShop`, `ErrorSimple`, `ShopSearch`, `ShopRename`, `RemoteShopWarp`, `ConfirmManage`, `FreeFormNotice` — all return `HiredMerchantOperationWriter`) + `merchant/serverbound/operation.go` (`HiredMerchantOperationHandle`). Body (Phase 1 Task 4): `merchant/operation_body.go`.

**Scope clarification (design §8):** these 7 structs are the **employee-shop** subset. The **hire-merchant** mode bytes are task-067's responsibility (commerce-domain `interaction/`). If audit reveals any of the 7 are hire-merchant variants, partition the report header and cross-link.

- [ ] **Step 1: Read the file once and map each struct to its mode byte**

```
sed -n '1,300p' libs/atlas-packet/merchant/clientbound/operation.go
```

Record each struct's mode-byte constant (the first byte each Encode writes) in a scratch table. The audit-report header (Step 8) embeds this table.

- [ ] **Step 2: Add `HiredMerchantOperation` writer + `HiredMerchantOperationHandle` handler entries to the v95 template if missing**

Same pattern as prior bucket tasks. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register HiredMerchantOperation{,Handle} for v95

IDA case-statements at <dispatcher>@<addr>."
```

- [ ] **Step 3: Append FNames to `gms_v95.json`**

For `HiredMerchantOperation`, the IDA dispatcher (likely `CHiredMerchantDlg::OnPacket` or similar) sub-dispatches on a mode byte. Add the per-mode call list under one FName entry annotated `"sub_ops": { "0xXX": { "comment": "OpenShop", "calls": [...] }, … }` per the schema documented in task-068 plan Task 9 step 2. If the existing schema doesn't carry sub-ops, embed in the function `comment` field per the same task-068 fallback — do NOT extend the analyzer to consume them.

- [ ] **Step 4: Run the audit**

- [ ] **Step 5: Triage per struct**

The auto-generated `HiredMerchantOperation.md` is the skeleton. Manually append per-struct sub-sections (one per the 7 variants):

```markdown
## Per-operation breakdown — employee-shop scope only

> Hire-merchant operation modes (player-driven shops) are audited under task-067.

### OpenShop (mode 0xXX)
- **Verdict:** ✅ / ⚠️ / ❌
- **IDA dispatcher branch:** `<function>@<addr>:case <mode>`
- **Atlas encoder:** `libs/atlas-packet/merchant/clientbound/operation.go:<startLine>-<endLine>`
- **Body delegation:** via `OperationBody` (TypeRegistry)
- **Wire-shape comparison:**
  - Atlas: `WriteByte(0xXX), WriteInt(shopId), ...`
  - IDA:   `Decode1 mode, Decode4 shopId, ...`
- **Notes:** <fix summary>

### ErrorSimple (mode 0xXX) … etc for the remaining 6 structs
```

For unresolvable mode bytes (e.g. `FreeFormNotice` mode set via constructor parameter), mark ⚠️ with a one-line rationale and a `_pending.md` row:

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(merchant/operation,FreeFormNotice): defer — runtime-resolved mode byte"
```

For real wire bugs, ship per-struct fixes with 4-variant tests.

- [ ] **Step 6: For each fix, add a 4-variant round-trip / Encode sweep test**

Tests in `libs/atlas-packet/merchant/clientbound/operation_test.go` (per struct, e.g. `TestOpenShopRoundTrip`) and `libs/atlas-packet/merchant/serverbound/operation_test.go`.

- [ ] **Step 7: Run tests**

```
go test -race ./libs/atlas-packet/merchant/...
```

- [ ] **Step 8: Commit each fix individually**

```
git add libs/atlas-packet/merchant/clientbound/operation.go libs/atlas-packet/merchant/clientbound/operation_test.go
git commit -m "fix(atlas-packet,merchant/operation): <Struct> <one-line summary>

Cites IDA <function>@<addr>:case <mode>: <one-line evidence>. Employee-shop scope per task-069 design §8."
```

- [ ] **Step 9: Append ack footer + bucket commit**

```
echo "" >> docs/packets/audits/gms_v95/HiredMerchantOperation.md
echo "Ack: misc-audit Phase 2f on $(date -I); per-struct breakdown verified across 7 employee-shop variants. Hire-merchant variants: see task-067." >> docs/packets/audits/gms_v95/HiredMerchantOperation.md
[ -f docs/packets/audits/gms_v95/HiredMerchantOperationHandle.md ] && echo "" >> docs/packets/audits/gms_v95/HiredMerchantOperationHandle.md && echo "Ack: misc-audit Phase 2f on $(date -I)" >> docs/packets/audits/gms_v95/HiredMerchantOperationHandle.md

git add docs/packets/audits/gms_v95/HiredMerchantOperation*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(misc): sub-phase 2f merchant bucket (2 files + body, 7 employee-shop structs)"
```

- [ ] **Step 10: Exit gate**

`grep -E 'HiredMerchantOperation' docs/packets/audits/gms_v95/SUMMARY.md` shows ≥ 2 rows (clientbound + serverbound). `HiredMerchantOperation.md` contains 7 per-struct sub-sections each with a verdict + IDA citation, and an explicit "employee-shop scope per task-067 cross-reference" header line.

---

### Task 11: Sub-phase 2g — `quest/` bucket (7 files)

**Packets:** `quest/clientbound/script_progress.go` + `quest/serverbound/{action,action_complete,action_restore_lost_item,action_script_end,action_script_start,action_start}.go`.

**Cross-task discipline (design §6):** tasks 014, 015, 023 touched quest reward / quest start / skill-gate. Read those commits first.

- [ ] **Step 1: Read task-014/015/023 quest history**

```
git log --oneline -- libs/atlas-packet/quest/ 2>/dev/null | head -40
git log --all --oneline --grep='task-01[4|5]\|task-023' -- libs/atlas-packet/quest/ 2>/dev/null
```

For each commit listed, run `git show <sha> -- libs/atlas-packet/quest/` and record the `Region/MajorVersion` gates each touched. Save to a scratch note (do NOT commit) at `/tmp/task-069-quest-prior-gates.md`. The Phase 2g triage uses this as the "do not silently widen/narrow" gate-tracking sheet.

- [ ] **Step 2: Check for existing reward sub-struct registry entries**

```
grep -E 'Reward|QuestReward|RewardInfo' tools/packet-audit/internal/atlaspacket/registry.go tools/packet-audit/internal/atlaspacket/registry_test.go
```

If task-014/015 already registered an equivalent type, do NOT add a duplicate (per design §6 last paragraph). Use the existing entry. If the analyzer surfaces an unresolved `QuestReward` type during Step 5 audit, add a per-Phase-1 registry fixture for it under a `task-069`-namespaced test name and re-run regression.

- [ ] **Step 3: Add quest writer / handler entries to the v95 template if missing**

Operation strings: `ScriptProgress`, `QuestActionHandle`, `ActionStart`, `ActionComplete`, `ActionRestoreLostItem`, `ActionScriptStart`, `ActionScriptEnd`. (Note: the seven serverbound files do NOT share one handler — each has its own; verify by reading each `Operation()` body.) Resolve opcodes via IDA and append. Commit per the standard pattern.

- [ ] **Step 4: Append FNames to `gms_v95.json`**

For `action_complete.go` and `action_start.go` specifically, the reward sub-struct is the analyzer descent point. Cite the v95 IDA dispatcher (`CUserBase::OnQuestRequest` or equivalent) and decompile to the per-handler decoder. If the quest reward sub-struct surfaces as unresolved, add a `QuestReward` registry entry now (mirror Phase 1 Task 2 fixture pattern) and re-run the audit.

- [ ] **Step 5: Run the audit**

- [ ] **Step 6: Triage per packet**

Standard flavours per Task 6 step 4. Plus quest-specific:
- **Gate overlap with task-014/015/023.** If the audit flags an "atlas gate wrong" verdict on a line range one of those tasks touched, STOP and re-read the prior task's IDA evidence before changing. Document any retained gate with a "matches task-NNN" line in the audit-report header. If the gate genuinely needs to widen/narrow, include "supersedes task-NNN gate (cites <prior-IDA-evidence>; this audit cites <new-IDA-evidence>)" in the fix commit message.
- **Sub-struct descent into `QuestReward`.** Verdict ❌ "unresolved type" → register, re-run.
- **Bare handlers** (e.g. `action_script_*` may be handled by `services/atlas-quest/` without an atlas-packet decoder). Defer to `_pending.md` with one row each.

- [ ] **Step 7: For each fix, add a 4-variant round-trip test**

For serverbound packets, use the Decode-sweep shape from Task 6 step 5. For clientbound (`script_progress.go`), use the round-trip pattern.

- [ ] **Step 8: Run tests**

```
go test -race ./libs/atlas-packet/quest/...
```

- [ ] **Step 9: Commit each fix individually**

```
git add libs/atlas-packet/quest/serverbound/action_complete.go libs/atlas-packet/quest/serverbound/action_complete_test.go
git commit -m "fix(atlas-packet,quest/action_complete): <one-line summary>

Cites IDA <function>@<addr>: <evidence>. Supersedes task-NNN gate at <line range> per <prior-IDA-evidence-comparison>."
```

(Drop the "Supersedes" line if no prior-task gate is touched.)

- [ ] **Step 10: Append ack footers + bucket commit**

```
for f in docs/packets/audits/gms_v95/ScriptProgress.md \
         docs/packets/audits/gms_v95/QuestActionHandle.md \
         docs/packets/audits/gms_v95/Action*.md; do
  [ -f "$f" ] || continue
  echo "" >> "$f"
  echo "Ack: misc-audit Phase 2g on $(date -I); cross-task lineage with task-014/015/023 verified." >> "$f"
done

git add docs/packets/audits/gms_v95/ScriptProgress.{md,json} \
        docs/packets/audits/gms_v95/QuestActionHandle.{md,json} \
        docs/packets/audits/gms_v95/Action*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(misc): sub-phase 2g quest bucket (7 files)"
```

(Adjust filenames to actual FNames.)

- [ ] **Step 11: Exit gate**

`grep -E 'ScriptProgress|QuestAction|Action(Complete|Start|RestoreLostItem|Script(Start|End))' docs/packets/audits/gms_v95/SUMMARY.md` shows ≥ 7 rows. Every ❌ resolved. Task-014/015/023 gate review noted in each affected audit report.

---

### Task 12: Sub-phase 2h — `account/serverbound` bucket (2 new files)

**Packets:** `register_pin.go` (`RegisterPinHandle`) + `set_gender.go` (`SetGenderHandle`). `accept_tos.go` is **already audited** in baseline (`AcceptTos.md` present from task-027) — do not re-audit.

**Dispatcher offset boundary (design §7.1):** the login-server dispatcher (`CLogin::OnPacket` or equivalent) MAY prepend an `accountId`/`sessionId` before the per-handler payload. Atlas-side decoders must consistently either include the field at offset 0 OR treat it as already-consumed by the dispatcher. Verify before changing.

- [ ] **Step 1: Verify dispatcher offset**

```
mcp__ida-pro__get_function_by_name "CLogin::OnPacket"
```

Decompile, find the `RegisterPin` and `SetGender` case-statement branches, confirm whether `accountId` is consumed pre-handler or per-handler. Document in `context.md` (one line each) and reference from each report header.

```
git add docs/tasks/task-069-misc-domain-packet-audit/context.md
git commit -m "docs(task-069): account-domain dispatcher-offset finding from v95 IDA"
```

(Skip the docs commit if no offset annotation is added to context.md.)

- [ ] **Step 2: Add `RegisterPinHandle` + `SetGenderHandle` entries to the v95 template if missing**

Same pattern as prior tasks. `AcceptTosHandle` is already registered (audited under task-027).

- [ ] **Step 3: Append FNames to `gms_v95.json`**

For each handler, locate the per-handler decoder in IDA and add a `Decode*`-op list with the dispatcher-offset annotation in the `comment` or `dispatcher_offset` field.

- [ ] **Step 4: Run the audit**

- [ ] **Step 5: Triage per Task 6 step 4 + per design §7.1**

Possible findings:
- Atlas decoder includes `accountId` at offset 0 AND IDA per-handler starts after the prepend → ✅ with header-line ack.
- Atlas decoder omits `accountId` AND atlas-login handler reads it from dispatcher context → ✅ with header-line ack.
- Inconsistency between `register_pin` and `set_gender` (one includes, one omits) → real bug; fix.
- Width / order on post-`accountId` payload → standard fix.

- [ ] **Step 6: For each fix, add a 4-variant Decode round-trip test (serverbound)**

Tests land in `libs/atlas-packet/account/serverbound/register_pin_test.go` and `set_gender_test.go`. Follow the shape from Task 6 step 5.

- [ ] **Step 7: Run tests**

```
go test -race ./libs/atlas-packet/account/...
```

- [ ] **Step 8: Commit each fix individually**

- [ ] **Step 9: Append ack footers + bucket commit**

```
for f in docs/packets/audits/gms_v95/RegisterPinHandle.md docs/packets/audits/gms_v95/SetGenderHandle.md; do
  [ -f "$f" ] || continue
  echo "" >> "$f"
  echo "Ack: misc-audit Phase 2h on $(date -I); dispatcher-offset finding documented in context.md." >> "$f"
done

git add docs/packets/audits/gms_v95/RegisterPin*.{md,json} \
        docs/packets/audits/gms_v95/SetGender*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2h account/serverbound bucket (2 new files; AcceptTos already audited)"
```

- [ ] **Step 10: Exit gate**

`grep -E 'RegisterPin|SetGender' docs/packets/audits/gms_v95/SUMMARY.md` shows 2 new rows. `AcceptTos` row is unchanged (verify via diff against `/tmp/summary-pre-task069.md`). Every ❌ resolved. Dispatcher-offset finding identical across both new reports (cross-packet consistency check).

---

### Task 13: Sub-phase 2i — `socket/` bucket (5 files; CRITICAL PATH)

**Packets:** `socket/clientbound/{hello,ping}.go` + `socket/serverbound/{channel_connect,pong,start_error}.go`. Writer/handler constants per `context.md`.

**Operational policy (design §4) — extra caution because socket fixes break every client if wrong:**

1. Every fix lands with a 4-variant `pt.Variants` round-trip test. NO exceptions.
2. Before commit, manually re-verify the fix against **all six version templates** (`template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json`). The widening past task-028's v95 baseline is because socket handshake formats are version-history-sensitive.
3. `atlas-login` AND `atlas-channel` build clean before the bucket commit. Socket encoders are constructed by both services.
4. The `hello` packet's `sendIv` and `recvIv` are 4-byte AES-OFB session seeds — verify byte order against IDA `CClientSocket::OnPacket` (or equivalent inbound-hello decoder symbol) before any change.
5. Any 3-deep nesting requirement → STOP, defer to `_pending.md`.

- [ ] **Step 1: Add `Hello`, `Ping` writer entries + `PongHandle`, `StartErrorHandle`, `CharacterLoggedInHandle` handler entries to the v95 template if missing**

Same pattern. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register socket writers/handlers for v95

IDA case-statements at <dispatchers>@<addr>."
```

- [ ] **Step 2: Append FNames to `gms_v95.json`**

For `Hello`, cite the inbound-hello decoder symbol. For `Pong`/`StartError`/`CharacterLoggedIn`, cite the per-handler decoder. For `Ping`, cite the keep-alive sender.

- [ ] **Step 3: Run the audit**

- [ ] **Step 4: Triage per packet**

Standard flavours per Task 6 step 4. Plus socket-specific:
- **`hello` IV byte order**. Atlas at `libs/atlas-packet/socket/clientbound/hello.go:52-58` writes `recvIv` then `sendIv` (4 bytes each). IDA evidence must confirm which order the v95 client reads. If reversed, fix + 4-variant test.
- **`hello` locale-byte position**. Atlas writes locale at end; v83 vs v95 may differ. If audit flips a verdict between v83 and v95, gate the difference with `Region/MajorVersion` (≤ 2-deep cap).
- **`pong` payload**. Currently `Pong{}` is payload-empty (see `libs/atlas-packet/socket/serverbound/pong.go`). IDA may show a 4-byte tick-count — if so, fix.
- **`start_error` enum**. 1-byte error code; verify enum values.

- [ ] **Step 5: For each fix, add a 4-variant round-trip test**

For `hello` (clientbound — Atlas encodes, client decodes):

```go
func TestHelloRoundTrip(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            input := NewHello(v.MajorVersion, v.MinorVersion, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 0x08)
            output := Hello{}
            pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
            if output.MajorVersion() != input.MajorVersion() {
                t.Errorf("majorVersion: got %v, want %v", output.MajorVersion(), input.MajorVersion())
            }
            // assert sendIv, recvIv, locale similarly
        })
    }
}
```

Tests land in `libs/atlas-packet/socket/clientbound/hello_test.go`, `ping_test.go`, and `libs/atlas-packet/socket/serverbound/{channel_connect,pong,start_error}_test.go`.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/socket/...
```

- [ ] **Step 7: Six-template re-verification**

For each socket fix, manually re-verify against the six version templates:

```
for t in template_gms_12_1.json template_gms_28_1.json template_gms_83_1.json \
         template_gms_87_1.json template_gms_92_1.json template_gms_95_1.json \
         template_jms_185_1.json; do
  echo "== $t =="
  grep -E '"writer": *"(Hello|Ping)"|"handler": *"(PongHandle|StartErrorHandle|CharacterLoggedInHandle)"' \
    services/atlas-configurations/seed-data/templates/$t || echo "  (no entry)"
done
```

For any template where the entry exists but the opcode differs from v95, verify via that version's IDA (or defer cross-version reconciliation to Phase 3). For any template where the entry is missing on a version that should have it, defer the addition to Phase 3.

- [ ] **Step 8: Verify atlas-login + atlas-channel build clean**

```
go build ./services/atlas-login/...
go build ./services/atlas-channel/...
```

Both must be clean. If either fails because an encoder constructor signature changed, the fix has rippled — patch the service handler in the same commit batch.

- [ ] **Step 9: Commit each fix individually**

```
git add libs/atlas-packet/socket/clientbound/hello.go libs/atlas-packet/socket/clientbound/hello_test.go
git commit -m "fix(atlas-packet,socket/hello): <one-line summary>

Cites IDA <CClientSocket::OnPacket>@<addr>: <evidence>. Critical-path encoder — six-template re-verification per task-069 design §4; atlas-login + atlas-channel build clean."
```

- [ ] **Step 10: Append ack footers + bucket commit**

```
for f in docs/packets/audits/gms_v95/Hello.md \
         docs/packets/audits/gms_v95/Ping.md \
         docs/packets/audits/gms_v95/PongHandle.md \
         docs/packets/audits/gms_v95/StartErrorHandle.md \
         docs/packets/audits/gms_v95/CharacterLoggedInHandle.md; do
  [ -f "$f" ] || continue
  echo "" >> "$f"
  echo "Ack: misc-audit Phase 2i on $(date -I); six-template re-verification + atlas-login/atlas-channel build clean per task-069 design §4." >> "$f"
done

git add docs/packets/audits/gms_v95/{Hello,Ping,PongHandle,StartErrorHandle,CharacterLoggedInHandle}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(misc): sub-phase 2i socket bucket (5 files, CRITICAL PATH)"
```

- [ ] **Step 11: Exit gate**

`grep -E 'Hello|Ping|PongHandle|StartError|CharacterLoggedIn' docs/packets/audits/gms_v95/SUMMARY.md` shows 5 rows. Every ❌ has a fix commit. Six-template re-verification noted in each report. `go build ./services/atlas-login/... ./services/atlas-channel/...` clean.

---

### Task 14: Sub-phase 2j — `_pending.md` sweep

Phase 2 closes with a consolidation pass on `docs/packets/ida-exports/_pending.md`. Goal: every misc-domain deferral lives under a clean per-domain heading; the file is a one-glance ledger for Task 18's reviewer.

- [ ] **Step 1: Inventory all misc-domain deferrals**

```
grep -nE '^\| (account|fame|stat|ui|socket|channel|merchant|quest|tool)/' docs/packets/ida-exports/_pending.md
```

Expected: rows added during Tasks 5–13. Count by sub-domain.

- [ ] **Step 2: Add or confirm section headings**

Open `_pending.md` and ensure these headings exist in order, AFTER the existing login + (eventually) world headings (do NOT touch prior-domain sections — regression risk):

```markdown
## Still pending — misc domain
| Packet | Reason | Reference |
|---|---|---|

## Sub-op enum drift — misc domain
| Packet | Sub-op | Constructor | IDA case |
|---|---|---|---|

## Tool limitations — misc domain
| Packet | Limitation | Workaround |
|---|---|---|

## Tool domain — utility-only (task-069)
(already populated in Task 5)
```

Reorganize Task-5-through-13 rows under the right heading. Each row carries a one-line rationale and a reference (audit report filename or commit SHA).

- [ ] **Step 3: Cross-reference deferrals against audit reports**

For every `_pending.md` row, the matching audit report under `docs/packets/audits/gms_v95/` must reference the deferral by name (e.g. "Sub-op enum modeling deferred — see `_pending.md` §Sub-op enum drift — misc domain"). Add the reference line to any report missing it.

- [ ] **Step 4: Commit**

```
git add docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/*.md
git commit -m "audit(misc): consolidate _pending.md sweep for sub-phase 2j"
```

- [ ] **Step 5: Exit gate — Phase 2 complete**

```
grep -c '❌' docs/packets/audits/gms_v95/SUMMARY.md
grep -c '⚠️' docs/packets/audits/gms_v95/SUMMARY.md
grep -c '✅' docs/packets/audits/gms_v95/SUMMARY.md
```

Total misc rows expected: ~20 new rows on top of the 28 login baseline (multi-struct files may push the count higher). Every ❌ has a fix commit on this branch (`git log --oneline | grep '^[a-f0-9]* fix(atlas-packet'`) OR a `_pending.md` row.

Login rows still byte-identical to baseline:

```
diff /tmp/summary-pre-task069.md <(grep -E '\[(All|AcceptTos|After|Auth|Character|Pin|Request|SelectWorld|Server)' docs/packets/audits/gms_v95/SUMMARY.md)
```

Expected: no diff. If anything drifts, STOP and investigate before Phase 3.

---

## Phase 3 — Cross-version pass

Three tracking sub-tasks (Tasks 15–17). One IDA binary at a time, user-driven swap. Each sub-task is "done" when:
- `docs/packets/ida-exports/<version>.json` has misc-domain entries for every FName from the v95 audit.
- The audit has been re-run against the version's template + IDA export.
- Every divergence vs v95 atlas-packet behaviour has either:
  - A `Region/MajorVersion` gate that already handles it (audit report captures evidence; no code change),
  - A gate fix on this branch with a 4-variant test sweep, OR
  - A template fix.

If a packet on a non-v95 version needs structural rewriting (>2 nested region/version guards), STOP, log to `_pending.md`, and continue.

Expected churn per design §9: v83 = socket-heavy; v87 = light; JMS v185 = moderate (channel port-width).

### Task 15: GMS v83 cross-version pass

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (exists; append misc entries)
- Modify (per fix): `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest}/**/*.go` + matching `_test.go`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Create or modify: `docs/packets/audits/gms_v83/` per-packet misc reports + `SUMMARY.md` (the directory exists from task-027 baseline; append misc rows to its SUMMARY)

- [ ] **Step 1: Confirm v83 IDA is loaded**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` matches GMS v83. If not, ask user to swap before continuing.

- [ ] **Step 2: For each misc FName resolved during Phase 2, populate `gms_v83.json`**

Per FName:
1. `mcp__ida-pro__get_function_by_name("<FName>")` → resolve address.
2. `mcp__ida-pro__decompile_function(<addr>)` → read Decode op list.
3. Translate to the existing `gms_v83.json` schema with the same op shape used during Phase 2.

Do not reorder existing login entries.

- [ ] **Step 3: Re-run the audit against v83**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v83.json \
  --output           docs/packets/audits/gms_v83
```

- [ ] **Step 4: Triage divergences**

For each ❌ in the v83 misc audit:
- **v95 fix gated on `MajorVersion() >= 95`** → no v83 regression. Audit-report-only.
- **v95 fix gated on `Region() == "GMS"` only** (no major-version filter) → check v83 IDA. If v83 confirms the same behaviour, tighten gate to preserve v83 shape. If not, leave gate as-is and document.
- **New v83-only mismatch the v95 audit didn't surface** → genuine cross-version bug. Fix with 4-variant test sweep + `Region/MajorVersion` gate.

Socket-specific (design §9): expect the highest churn here. Each socket fix repeats the six-template re-verification + atlas-login/atlas-channel build clean from Task 13 Step 7-8.

- [ ] **Step 5: For each fix, add a 4-variant sweep test that proves no regression on v28/v87/v95/JMS-185**

(The test exists from Phase 2 if the v95 audit caught a sibling bug; in that case extend the existing test's per-variant assertions. Otherwise create new.)

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/...
```

- [ ] **Step 7: Commit each fix individually**

```
fix(atlas-packet,<domain>/<pkt>): widen/narrow v83 gate for <field>

Cites IDA v83 <function>@<addr>: <one-line evidence>.
```

- [ ] **Step 8: Hard-cap check**

If any single misc-domain encoder now contains 3+ nested `Region`/`MajorVersion` levels — STOP per design §1. Append to `_pending.md`:

```markdown
## Nesting hard-cap (deferred) — misc domain
| Encoder | Trigger | Defer reason |
|---|---|---|
| `<file>.go` | <which version chain triggered it> | 3-deep nesting exceeded; structural rewrite out of scope per task-069 design §1 |
```

Do not refactor in this task.

- [ ] **Step 9: Bucket commit**

```
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/ \
        services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "audit(misc): GMS v83 cross-version pass"
```

- [ ] **Step 10: Exit gate**

`docs/packets/audits/gms_v83/SUMMARY.md` has misc-domain rows for every Phase 2 FName. Every ❌ has a fix commit or `_pending.md` row. atlas-login + atlas-channel build clean (if any socket fix landed).

---

### Task 16: GMS v87 cross-version pass

**Files:**
- Create: `docs/packets/ida-exports/gms_v87.json` (does not exist yet on this branch — create with the same schema as `gms_v95.json`, login entries can be added on-demand if any login row is touched).
- Create: `docs/packets/audits/gms_v87/` (does not exist yet on this branch — pipeline creates on first run).
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` (exists)
- Modify (per fix): `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest}/**/*.go` + matching `_test.go`

Identical workflow to Task 15. Replace `v83` with `v87` everywhere. Note that the `gms_v87.json` file must be created before the audit run; seed it with the minimum schema (top-level `binary`, `md5`, `generated_at`, `functions: {}`) and populate misc-domain functions during Step 2.

- [ ] **Step 1: Confirm v87 IDA is loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Create `gms_v87.json` skeleton if missing, then populate misc FNames**

```
[ -f docs/packets/ida-exports/gms_v87.json ] || cat > docs/packets/ida-exports/gms_v87.json <<'EOF'
{
  "binary": "<filename-from-IDA-metadata>",
  "md5": "<md5-from-IDA-metadata>",
  "generated_at": "<ISO-8601 timestamp>",
  "functions": {}
}
EOF
```

Then append per-FName entries via `mcp__ida-pro__decompile_function`.

- [ ] **Step 3: Re-run the audit against v87**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v87.json \
  --output           docs/packets/audits/gms_v87
```

- [ ] **Steps 4–8:** Same shape as Task 15 steps 4–8.

- [ ] **Step 9: Bucket commit**

```
git add docs/packets/ida-exports/gms_v87.json \
        docs/packets/audits/gms_v87/ \
        services/atlas-configurations/seed-data/templates/template_gms_87_1.json
git commit -m "audit(misc): GMS v87 cross-version pass"
```

- [ ] **Step 10: Exit gate** — same as Task 15 step 10.

---

### Task 17: JMS v185 cross-version pass

**Files:**
- Create: `docs/packets/ida-exports/gms_jms_185.json` (does not exist yet on this branch — create per Task 16 Step 2 pattern, naming aligns with the existing `gms_v83.json` convention).
- Create: `docs/packets/audits/jms_v185/` (does not exist yet on this branch — pipeline creates on first run).
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` (exists)
- Modify (per fix): `libs/atlas-packet/{account,fame,stat,ui,socket,channel,merchant,quest}/**/*.go` + matching `_test.go`

**Highest-attention JMS divergences expected (per design §9):**
- `channel/clientbound/change.go` port-width — JMS sometimes encoded port as u32 vs u16.
- `socket/clientbound/hello.go` IV-byte ordering / locale-byte position.

- [ ] **Step 1: Confirm JMS v185 IDA is loaded** (`mcp__ida-pro__get_metadata`).

- [ ] **Step 2: Create `gms_jms_185.json` skeleton if missing, then populate misc FNames**

```
[ -f docs/packets/ida-exports/gms_jms_185.json ] || cat > docs/packets/ida-exports/gms_jms_185.json <<'EOF'
{
  "binary": "<filename-from-IDA-metadata>",
  "md5": "<md5-from-IDA-metadata>",
  "generated_at": "<ISO-8601 timestamp>",
  "functions": {}
}
EOF
```

For FNames with no JMS equivalent (different opcode space, or different code-path entry), record the JMS-side FName + address as a separate entry annotated `"region": "JMS"`. Do NOT reuse GMS FNames for unrelated JMS functions.

- [ ] **Step 3: Re-run the audit against JMS v185**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage divergences per Task 15 step 4**

Plus JMS-specific:
- **In scope:** atlas-packet writes bytes the JMS client decodes wrong.
- **Out of scope:** JMS-specific feature the service doesn't wire through.
- **In scope:** width mismatch on a field both versions decode (e.g. channel port).
- **Out of scope:** JMS template opcode wrong when GMS is right — fix the template, atlas-packet untouched.

- [ ] **Step 5: Hard-cap check (same as Task 15 step 8)**

If any encoder hits 3-deep nesting because the JMS branch adds a third axis on top of `Region` × `MajorVersion`, STOP and defer to `_pending.md`. Misc-domain encoders do NOT get a `set_field`-style 3-deep carve-out.

- [ ] **Step 6: For each in-scope fix, add a 4-variant sweep test + gate**

- [ ] **Step 7: Run tests**

```
go test -race ./libs/atlas-packet/...
```

- [ ] **Step 8: Commit each fix individually**

- [ ] **Step 9: Bucket commit**

```
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/ \
        services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "audit(misc): JMS v185 cross-version pass"
```

- [ ] **Step 10: Exit gate** — `docs/packets/audits/jms_v185/SUMMARY.md` has misc-domain rows for every Phase 2 FName. Every ❌ has a fix commit, `_pending.md` row, or hard-cap-deferral row. atlas-login + atlas-channel build clean (if any socket fix landed).

---

## Phase 4 — TOTAL.md + post-phase-b.md + closeout

One task. Phase 4 produces the unique `TOTAL.md` deliverable, the closing-memo `post-phase-b.md`, runs the full verification matrix, performs the coverage-completeness sweep, and dispatches code review before PR.

### Task 18: TOTAL.md, post-phase-b.md, verification, code review, PR

**Files:**
- Create: `docs/packets/audits/gms_v95/TOTAL.md`
- Create: `docs/tasks/task-069-misc-domain-packet-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tallies row if used)
- Modify: `docs/packets/ida-exports/_pending.md` (final misc-domain section state)

- [ ] **Step 1: Run the coverage-completeness sweep**

```
find libs/atlas-packet -maxdepth 1 -type d | sort
```

Cross-reference against the 7-task coverage matrix (027 login, 028 character, 065 combat, 066 social, 067 commerce, 068 world, 069 misc). For each unclaimed directory:
- If it contains `.go` files that look like packets (have `Operation()` / `Encode()` methods), add a TOTAL.md gap row with "defer to new task" recommendation + a one-line rationale.
- If it's a utility / model / test directory, add a gap row with "out-of-scope-permanently" + rationale.
- If it's empty or already documented, no entry needed.

Save the sweep output to a scratch note for Step 2's TOTAL.md draft.

- [ ] **Step 2: Pull verdict roll-ups from sibling-task branches**

For each sibling task in flight (028, 065, 066, 067, 068), the sibling worktree's HEAD has its current SUMMARY:

```
for branch in task-028-character-domain-audit \
              task-065-combat-domain-audit \
              task-066-social-domain-packet-audit \
              task-067-commerce-domain-packet-audit \
              task-068-world-domain-packet-audit; do
  echo "=== $branch ==="
  git -C ../$branch show $branch:docs/packets/audits/gms_v95/SUMMARY.md 2>/dev/null | head -5 || echo "(no SUMMARY yet)"
  echo "  ✅ count: $(git -C ../$branch show $branch:docs/packets/audits/gms_v95/SUMMARY.md 2>/dev/null | grep -c '| ✅ |')"
  echo "  ⚠️ count: $(git -C ../$branch show $branch:docs/packets/audits/gms_v95/SUMMARY.md 2>/dev/null | grep -c '| ⚠️ |')"
  echo "  ❌ count: $(git -C ../$branch show $branch:docs/packets/audits/gms_v95/SUMMARY.md 2>/dev/null | grep -c '| ❌ |')"
done
```

Note any sibling without a final `post-phase-b.md` commit — those rows in TOTAL.md get `(draft)` annotations per design §10.4.

- [ ] **Step 3: Write `docs/packets/audits/gms_v95/TOTAL.md`**

Layout per design §10.2:

```markdown
# Atlas Packet Library — Cross-Task Audit Ledger (GMS v95 baseline)

> **Last updated:** 2026-MM-DD
> **Maintenance:** To add a domain, append a row to §2 with task-id, file count, and verdict roll-up. Update last-updated. Recompute coverage-completeness statement if §3 gaps section becomes empty.

## 1. Contributing tasks

| Task | Domain(s) | post-phase-b commit | Status |
|---|---|---|---|
| task-027 | login | `<sha>` | shipped |
| task-028 | character | `<sha>` | shipped / (draft) |
| task-065 | combat (monster, drop, reactor, ...) | `<sha>` | shipped / (draft) |
| task-066 | social (buddy, messenger, note, chat) | `<sha>` | shipped / (draft) |
| task-067 | commerce (inventory, pet, storage, cash, interaction/hire-merchant) | `<sha>` | shipped / (draft) |
| task-068 | world (field, portal, npc) | `<sha>` | shipped / (draft) |
| task-069 | misc (account, fame, stat, ui, socket, channel, merchant/employee-shop, quest, tool) | `<sha>` | shipped |

## 2. Coverage matrix — `libs/atlas-packet/`

| Directory | Owning task | Packet files | ✅ | ⚠️ | ❌ | Notes |
|---|---|---|---|---|---|---|
| account/ | task-069 | 3 | <n> | <n> | <n> | AcceptTos audited under task-027 |
| buddy/ | task-066 | <n> | <n> | <n> | <n> | (draft) |
| cash/ | task-066 or task-067 | <n> | <n> | <n> | <n> | (draft) — verify task-067 scope at finalization |
| channel/ | task-069 | 2 | <n> | <n> | <n> | |
| character/ | task-028 | <n> | <n> | <n> | <n> | (draft) |
| chat/ | task-066 | <n> | <n> | <n> | <n> | (draft) |
| drop/ | task-065 | <n> | <n> | <n> | <n> | (draft) |
| fame/ | task-069 | 2 + 1 body | <n> | <n> | <n> | |
| field/ | task-068 | <n> | <n> | <n> | <n> | (draft) |
| guild/ | task-066 or new | <n> | <n> | <n> | <n> | verify task-066 scope; otherwise gap |
| interaction/ | task-067 | <n> | <n> | <n> | <n> | (draft) — hire-merchant subset |
| inventory/ | task-067 | <n> | <n> | <n> | <n> | (draft) |
| login/ | task-027 | 28 | 27 | 0 | 1 | from baseline |
| merchant/ | task-069 | 2 + 1 body | <n> | <n> | <n> | employee-shop scope only; hire-merchant → task-067 |
| messenger/ | task-066 | <n> | <n> | <n> | <n> | (draft) |
| model/ | — | 0 packets | — | — | — | shared types; not wire-bound |
| monster/ | task-065 | <n> | <n> | <n> | <n> | (draft) |
| note/ | task-066 | <n> | <n> | <n> | <n> | (draft) |
| npc/ | task-068 | <n> | <n> | <n> | <n> | (draft) |
| party/ | task-066 or new | <n> | <n> | <n> | <n> | verify task-066 scope; otherwise gap |
| pet/ | task-067 | <n> | <n> | <n> | <n> | (draft) |
| portal/ | task-068 | <n> | <n> | <n> | <n> | (draft) |
| quest/ | task-069 | 7 | <n> | <n> | <n> | |
| reactor/ | task-065 | <n> | <n> | <n> | <n> | (draft) |
| socket/ | task-069 | 5 | <n> | <n> | <n> | critical path |
| stat/ | task-069 | 1 | <n> | <n> | <n> | |
| storage/ | task-067 | <n> | <n> | <n> | <n> | (draft) |
| test/ | — | 0 packets | — | — | — | test harness; not wire-bound |
| tool/ | — | 0 packets | — | — | — | utility (uint128); not wire-bound |
| ui/ | task-069 | 3 + 1 body | <n> | <n> | <n> | |

Top-level files (`packet.go`, `resolve.go`, `resolve_test.go`) are library plumbing, not domains.

## 3. Gaps

| Directory | Reason | Rationale |
|---|---|---|
| (populated only if §1 coverage sweep finds an unclaimed packet directory) | | |

Expected after the §1 sweep: **empty**.

## 4. Verdict roll-up arithmetic

Each task's per-domain counts in §2 are computed by:

```
grep -c '| ✅ |' <task>/docs/packets/audits/gms_v95/SUMMARY.md  # rows scoped to that task's domains
```

Sibling-task tables that use different verdict markers are normalized to ✅ / ⚠️ / ❌ here.

## 5. Coverage-completeness statement

Coverage of `libs/atlas-packet/` is complete as of <task-069 closing commit ref>.
```

Fill in actual numbers and SHAs. Sibling-task rows with `(draft)` annotations carry the sibling branch HEAD SHA.

- [ ] **Step 4: Write `post-phase-b.md`**

Mirror task-028's structure (most-recent shipped pattern). Six sections:

```markdown
# Task-069 Post-Phase-B — Misc-Domain Audit Closeout

## Final state

- Packets audited: ~20 new misc-domain rows (account 2 new; fame 2; stat 1; ui 3; socket 5; channel 2; merchant 2; quest 7; tool 0; AcceptTos pre-existing from task-027).
- Verdicts (v95): ✅ <n_pass> / ⚠️ <n_warn> / ❌ <n_fail> / pending <n_pending>.
- IDA-export coverage: v83 / v87 / v95 / JMS v185 — misc FNames populated.
- TOTAL.md: shipped at `docs/packets/audits/gms_v95/TOTAL.md` with cross-task ledger covering 027 + 028 + 065-069. (draft) rows revised pre-PR for any sibling that landed during the task.

## Real wire bugs fixed

| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit; enumerate via `git log --oneline main..HEAD | grep '^[a-f0-9]* fix(atlas-packet'`)

## Template opcode / enum fixes

| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements

- TypeRegistry fixtures for `FameResponseBody` (Phase 1 Task 2), `UiOpenBody` (Phase 1 Task 3), `MerchantOperationBody` (Phase 1 Task 4).
- (No analyzer changes — design §1 mandate honoured.)

## Remaining work

| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` `## Still pending — misc domain`, `## Sub-op enum drift — misc domain`, `## Tool limitations — misc domain`, plus any §1 hard-cap stops)

## Cross-version notes

- **v83:** <notable findings; socket-heavy per design §9>.
- **v87:** <notable findings; expected light>.
- **JMS v185:** <notable findings; channel port-width and socket-handshake>.

## Tool-domain confirmation

`libs/atlas-packet/tool/` is utility-only (`uint128.go`). Zero packet rows. Documented in `_pending.md` §Tool domain — utility-only and in TOTAL.md §2.

## Coverage statement

`find libs/atlas-packet -maxdepth 1 -type d | sort` cross-referenced against
the 7-task coverage matrix in TOTAL.md §2. Every directory is either claimed
by a task or documented in TOTAL.md §2 notes column / §3 gaps section.
```

Fill in actual numbers, file paths, and commit SHAs from the branch history.

- [ ] **Step 5: Run the full verification matrix**

```
go build ./...
go vet ./libs/atlas-packet/... ./tools/packet-audit/...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All four must be clean. If any fails, fix on the branch and re-run before continuing.

- [ ] **Step 6: Verify socket-consumer services build clean**

```
go build ./services/atlas-login/...
go build ./services/atlas-channel/...
go build ./services/atlas-account/...
```

All three must be clean. These are the services whose handler-side encoder constructors ripple from socket/account fixes per design §4 / §7.1.

- [ ] **Step 7: Decide whether `docker build` is required**

Per CLAUDE.md Build & Verification §3: required when a service `Dockerfile` or `go.mod` was touched. This task is expected to touch only `template_*.json` files under `services/atlas-configurations/seed-data/` and audit reports under `docs/`. If only seed-data JSON + docs changed:

```
git diff --name-only main..HEAD -- services/atlas-configurations/ | grep -v 'seed-data/templates/'
git diff --name-only main..HEAD -- '**/Dockerfile' '**/go.mod' '**/go.sum'
```

If both empty: skip `docker build`. Otherwise:

```
docker build -f services/atlas-configurations/Dockerfile .
```

Expected: clean.

- [ ] **Step 8: gitleaks scrub**

```
grep -r '/home/' docs/packets/audits/gms_v95/ docs/packets/audits/gms_v83/ docs/packets/audits/gms_v87/ docs/packets/audits/jms_v185/ 2>/dev/null
```

Expected: no output. If any user-home path appears in an audit report, scrub:

```
sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>
git commit -am "audit(misc): scrub absolute user-home paths from misc-domain reports"
```

- [ ] **Step 9: Verify login SUMMARY rows still byte-identical to pre-task baseline**

```
diff /tmp/summary-pre-task069.md <(grep -E '\[(All|AcceptTos|After|Auth|Character|Pin|Request|SelectWorld|Server)' docs/packets/audits/gms_v95/SUMMARY.md)
```

Expected: no diff. If anything drifted, STOP and investigate before opening the PR.

- [ ] **Step 10: Pre-PR TOTAL.md sibling-task revision**

For each sibling task in flight at Phase 4 start, re-run Step 2's sibling SUMMARY pull. For any sibling whose `post-phase-b.md` has shipped since Step 3, update its TOTAL.md row from `(draft)` to final + replace the branch-HEAD SHA with the merge-commit SHA.

- [ ] **Step 11: Commit TOTAL.md + post-phase-b.md**

```
git add docs/packets/audits/gms_v95/TOTAL.md \
        docs/tasks/task-069-misc-domain-packet-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "docs(task-069): TOTAL.md cross-task ledger + post-phase-b closeout"
```

- [ ] **Step 12: Run code review**

Invoke `superpowers:requesting-code-review`. Allow the orchestration skill to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/`, `tools/packet-audit/`, and `services/atlas-configurations/seed-data/templates/` changes.

Read the resulting `audit.md` and act on every BLOCKER / MAJOR finding before opening a PR. Re-run reviews after fix commits land.

- [ ] **Step 13: Open the PR**

Title: `task-069: misc-domain packet audit (v83/v87/v95/JMS185) — account/fame/stat/ui/socket/channel/merchant/quest + TOTAL.md`

Body: short summary + link to `post-phase-b.md` for the full bug ledger + link to `TOTAL.md` for the cross-task coverage statement. Use `superpowers:finishing-a-development-branch` to drive the PR creation.

---

## Self-review notes

Run through the plan once more with fresh eyes before committing it.

- **Spec coverage** — every PRD §4 functional requirement is covered by an explicit task above:
  - §4.1 coverage matrix → Phase 2 Tasks 5–14 (sub-phases 2a–2j; ~20 new SUMMARY rows from the 8 non-tool sub-phases, ignoring multi-struct fan-out).
  - §4.2 inventory enumeration → context.md §"Inventory" + Task 5 step 1 (tool/ confirmation).
  - §4.3 IDA exports → Phase 2 Tasks 6–13 step 2 (v95 append) + Phase 3 Tasks 15–17 step 2 (v83/v87/JMS185 append; v87 + JMS185 files created from skeleton if missing).
  - §4.4 wire bug fixes → embedded in every Phase 2/3 task (`fix(atlas-packet,...)` commits with 4-variant sweeps).
  - §4.5 template fixes → embedded in every Phase 2/3 task (`fix(configurations,templates)` commits).
  - §4.6 TypeRegistry extensions → Phase 1 Tasks 2–4 (high-confidence batch) + Task 11 step 2 (quest reward sub-struct deferred-evidence registration).
  - §4.7 cross-version pass → Tasks 15–17 (one task per version, v95-complete-first).
  - §4.8 task-domain confirmation → Task 5 (sub-phase 2a tool/ confirmation).
  - §4.9 TOTAL.md → Task 18 step 3.
  - §4.10 post-audit coverage sweep → Task 18 step 1.
- **No placeholders** — every step contains either an exact command, an exact code block, or an exact file path. IDA addresses, hex strings, and per-handler decoder symbols can't be known until execution; placeholders are annotated with `<addr>` / `<one-line evidence>` / `v.<Name>Hex` and the surrounding step makes the IDA-lookup workflow explicit.
- **Type consistency** — registry-test names (`TestRegistryRegistersFameResponseBody`, `TestRegistryRegistersUiOpenBody`, `TestRegistryRegistersMerchantOperationBody`) match Phase 1 task numbers. Encoder test names follow `Test<PacketName>RoundTrip` for round-trip pairs (most misc packets have both Encode and Decode) or `Test<PacketName>Encode` for encode-only client-bound packets. Per-struct test names (`TestOpenShopRoundTrip`, `TestReceiveResponseRoundTrip`, etc.) follow the per-type shape from Tasks 9 and 10.
- **No analyzer changes** — design §1 mandate honoured throughout. Tasks 2–4 are registry-fixture-only (no `analyzer.go` edits). Task 11 step 2 explicitly defers analyzer descent gaps to one-line registry additions, not analyzer rewrites.
- **No `reflect`, no `interface{}`, no benchmarks** — none of the code in the plan uses `reflect.*` or adds an `interface{}` parameter to an encoder.
- **Cross-domain regression gate** — Phase 0 (Task 1), every Phase 1 task (Tasks 2–4 step 4), and Phase 4 (Task 18 step 9) each diff against `/tmp/summary-pre-task069.md`. Login verdicts are protected at every commit point.
- **Nesting cap** — 2-deep cap throughout. Hard-cap check explicit in Task 15/16/17 step 8 / 5 / 5. Misc-domain encoders do NOT get a `set_field`-style 3-deep carve-out.
- **Socket sensitivity** — Task 13 enforces the design §4 critical-path policy: 4-variant tests + six-template re-verification (Step 7) + atlas-login/atlas-channel build clean (Step 8). Phase 4 Task 18 step 6 repeats the service-build check.
- **Quest cross-task discipline** — Task 11 step 1 reads task-014/015/023 commits BEFORE any quest change. Task 11 step 2 checks for existing reward sub-struct registry entries before duplicating.
- **Account dispatcher offset** — Task 12 step 1 verifies offset against `CLogin::OnPacket`; the finding is documented in context.md (one-line each per packet) and referenced from each report header.
- **Channel migration endianness** — Task 7 step 4 calls out host endianness explicitly.
- **Merchant scope** — Task 10 step 8 ack footer + report header explicitly states "employee-shop scope; hire-merchant → task-067".
- **Gitleaks** — Task 18 step 8 is the mandatory pre-PR scrub. Steps consistently invoke the audit CLI with relative paths.
- **TOTAL.md** — Task 18 step 3 produces it; step 2 pulls sibling-task verdict counts via `git -C ../<branch> show`; step 10 revises `(draft)` rows pre-PR. Per design §10.4 the post-PR amend protocol is documented in TOTAL.md §1 maintenance line.
- **Coverage-completeness sweep** — Task 18 step 1 runs `find libs/atlas-packet -maxdepth 1 -type d | sort` and cross-references against the 7-task matrix. TOTAL.md §3 gaps section is the result.
- **Phase 2 sub-phase ordering** — tool → stat → channel → ui → fame → merchant → quest → account → socket per design §11. Easy wins build pipeline confidence before the high-cognitive-load packets; cross-task / dispatcher-offset / critical-path packets land last.
- **AcceptTos handling** — already audited in baseline (`AcceptTos.md` exists with ✅). Task 12 step 1 explicitly notes this and avoids re-auditing. Phase 4 Task 18 step 9 diff filters preserve the row.
- **Sibling tasks not in main** — context.md baseline note documents this. TOTAL.md draft pulls sibling-branch HEAD SHAs; pre-PR revision (Task 18 step 10) updates to merge commits for any that ship.
