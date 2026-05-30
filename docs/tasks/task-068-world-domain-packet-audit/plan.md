# World-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline shipped in task-027 (login) and task-028 (character) to the 57 packets in `libs/atlas-packet/field/`, `portal/`, and `npc/`, ship wire-bug + template fixes against GMS v95 IDA, and re-verify across v83/v87/JMS v185.

**Architecture:** Phase 0 is a regression-only re-run of the existing v95 audit to confirm login (28) + character (52) verdicts are byte-identical before any registry change lands. Phase 1 lands the predicted `TypeRegistry` additions (NPC shop item entry, NPC conversation per-type sub-encoders, optional clock/set_field sub-structs) with one fixture each. Phase 2 audits the world domain in 8 tracking sub-phases ordered easy → hard (portal → field/serverbound → field/clientbound non-effect → field effect cluster → npc/clientbound non-conversation → npc/clientbound conversation → npc/serverbound → `_pending.md` sweep). Phase 3 re-runs the audit against v83 / v87 / JMS v185. Phase 4 ships `post-phase-b.md`, full verification, and code review. No analyzer code changes are anticipated; if one is forced, STOP and escalate per design §3.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast` for AST analysis), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer for round-trip tests, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies; this task ships audit reports + targeted code/template fixes only.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-068-world-domain-packet-audit/` on branch `task-068-world-domain-packet-audit`. Before *every* commit run `git rev-parse --show-toplevel` and `git branch --show-current`; if either disagrees, stop.
- **TDD cadence.** For analyzer/registry work: test first → run-to-fail → minimal implementation → run-to-pass → commit. For encoder fixes: 4-variant `pt.Variants` sweep test first → fix encoder → run-to-pass → commit.
- **Verification cadence (registry changes).** `go test -race ./tools/packet-audit/...` clean before commit. Then re-run the v95 audit and confirm login + character `SUMMARY.md` rows are byte-identical (Phase 0 gate; repeated after every Phase 1 commit).
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with a 4-variant test sweep covering GMS v28 / v83 / v87 / v95 + JMS v185 (use the existing `pt.Variants` pattern in `libs/atlas-packet/test/context.go`).
- **No `*_testhelpers.go` files.** Use the project's Builder pattern.
- **No `reflect`, no new `interface{}` params, no benchmarks** in atlas-packet edits.
- **No gitleaks bait.** Absolute paths like `/home/<user>/` must not appear in any audit report under `docs/packets/audits/gms_v95/`. Pre-PR check is mandatory (Task 15 step 4). Invoke the audit CLI from the worktree root with relative paths (`--atlas-packet libs/atlas-packet`, etc.) — never absolute.
- **No analyzer changes.** Per design §3, this task does not touch `tools/packet-audit/internal/atlaspacket/analyzer.go`. If the audit panics or surfaces a new cycle, STOP and spin a sibling task; do NOT inline-fix the analyzer.
- **Cross-domain regression gate.** Every Phase 1 registry commit re-runs the audit and confirms `SUMMARY.md` rows for login (28) and character (52) are byte-identical to pre-task state. Any drift → STOP, roll back the registry entry, investigate.
- **Tracking sub-tasks vs PR-sized commits.** Phase 2 sub-tasks (Tasks 7–14) and Phase 3 sub-tasks (Tasks 15–17) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every packet in its bucket has a verdict and every ❌ has either a fix commit or a `_pending.md` row.
- **Audit-report ack footer policy.** The ack footer ("Ack: <verifier> on <date>") is the LAST line written to each report. If a re-run is needed for a single report, `git checkout HEAD -- docs/packets/audits/gms_v95/<Report>.md` before re-execution.
- **Nesting cap.** Every encoder stays under **2 nested region/version guards** *except* `set_field.go`, which is allowed **3 deep** per PRD §4.6 and design §5. 4+ → STOP, defer to `_pending.md`.

---

## Phase 0 — Regression baseline (gate)

One task. Exit when login (28) + character (52) SUMMARY rows are byte-identical to pre-task state.

### Task 1: Re-run v95 audit, confirm prior-domain verdicts unchanged

**Files:**
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (only if pipeline output changes; expected to be a no-op)
- Modify: any per-packet `.md`/`.json` whose mtime updates (expected: pipeline is deterministic; no semantic diffs)

- [ ] **Step 1: Snapshot the prior login + character SUMMARY rows**

```
git show HEAD:docs/packets/audits/gms_v95/SUMMARY.md > /tmp/summary-pre-task068.md
wc -l /tmp/summary-pre-task068.md
grep -c '❌' /tmp/summary-pre-task068.md
grep -c '⚠️' /tmp/summary-pre-task068.md
grep -c '✅' /tmp/summary-pre-task068.md
```

Record the verdict counts. Expected from task-028 closeout: roughly `28 login + 52 character = 80` rows total before world entries are appended. Exact numbers from `task-028-character-domain-audit/post-phase-b.md` if disagreement.

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

Expected runtime: ≤ 30 s. Exit code 1 is normal if any prior ❌ rows persist.

- [ ] **Step 3: Diff SUMMARY against the snapshot**

```
diff /tmp/summary-pre-task068.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff. If any login or character row changed verdict, the analyzer drifted since task-028 closed — STOP and triage before continuing. If only ordering changed (e.g. registry order shifted), commit a "sort SUMMARY" pre-task cleanup commit and re-snapshot.

- [ ] **Step 4: Commit (no-op or audit-only)**

If no files changed:

```
git status --short docs/packets/audits/gms_v95/
```

Expected: empty. Then write a single empty marker commit:

```
git commit --allow-empty -m "audit(world): phase-0 regression baseline — prior-domain verdicts unchanged"
```

If files changed without semantic drift (whitespace, timestamps), commit them:

```
git add docs/packets/audits/gms_v95/
git commit -m "audit(world): phase-0 regression baseline — refresh prior-domain reports"
```

- [ ] **Step 5: Exit gate**

Verdict counts in `docs/packets/audits/gms_v95/SUMMARY.md` match the snapshot. Branch state clean. Proceed to Phase 1.

---

## Phase 1 — TypeRegistry sub-struct coverage

Two tasks. Each registry addition lands with one fixture in `registry_test.go` and a cross-domain regression check that login + character SUMMARY rows stay byte-identical.

Per design §4, the high-confidence additions for this domain are:

| Sub-struct | Source type | Consumed by | Confidence |
|---|---|---|---|
| NPC shop item entry | `libs/atlas-packet/npc/.../shop_item.go` (or inline in `shop_list.go`) | `npc/clientbound/shop_list.go` commodity loop | High |
| NPC conversation text-type sub-encoders | per-type encoder blocks inside `npc/clientbound/conversation.go` | `npc/clientbound/conversation.go` self | High |

Medium-confidence additions (`SetField` map-header, `WarpToMap` coord block, Clock per-mode sub-encoders) are deferred to Phase 2 when the analyzer surfaces an unresolved type. The plan registers them on the first sub-phase that needs them.

### Task 2: Registry fixture for NPC shop item entry

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Identify the NPC shop item sub-struct symbol**

```
grep -RnE '(ShopItem|ShopEntry|CommodityEntry|NpcShopItem|NpcShopEntry)' libs/atlas-packet/npc/ libs/atlas-packet/model/ | head -20
```

If a named type exists (likely in `libs/atlas-packet/npc/` or `libs/atlas-packet/model/`), record its name. If it's inline in `shop_list.go` (no named type), the fixture below tests the loop body's call set is non-empty rather than a typed entry.

- [ ] **Step 2: Write the fixture (typed-entry case)**

If a named type exists (e.g. `NpcShopItem`):

```go
func TestRegistryRegistersNpcShopItem(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    name := "NpcShopItem" // replace with the actual symbol if different
    if !reg.HasType(name) {
        t.Fatalf("registry missing type %s", name)
    }
    calls, ok := reg.Calls(name)
    if !ok || len(calls) == 0 {
        t.Fatalf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
    }
}
```

If the type is inline (no named struct), skip this fixture and fold the verification into the shop-list audit report manually during Task 11.

- [ ] **Step 3: Run the fixture**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersNpcShopItem -v
```

Expected: PASS *if a named type exists*. If FAIL because the struct uses a pointer receiver the registry strips, inspect `registry.go:152-163` `receiverIdent` and document the gap in a one-line comment in the test; do NOT modify `registry.go` (this task is registry-fixture-only — analyzer/registry code changes are out of scope).

- [ ] **Step 4: Re-run the full audit, confirm prior-domain rows unchanged**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
diff /tmp/summary-pre-task068.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff. If any login/character row flips, STOP — registry fixture-only commits should not cascade. Investigate; likely the fixture's `NewTypeRegistry` instantiation is interacting with module state.

- [ ] **Step 5: Commit**

```
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert NpcShopItem registry coverage"
```

If the type is inline and the fixture was skipped, instead write a one-line note to context.md and commit:

```
git add docs/tasks/task-068-world-domain-packet-audit/context.md
git commit -m "docs(task-068): NPC shop item is inline — no registry fixture"
```

---

### Task 3: Registry fixture for NPC conversation text-type sub-encoders

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Identify the per-type conversation sub-encoder symbols**

```
grep -nE '^func \(.*NpcConversation.*\) .*Encode|^type .*Conversation.*struct' libs/atlas-packet/npc/clientbound/conversation.go | head -30
```

Per PRD §4.5 and design §6 the file defines 8 dialog-type encoder blocks: `say`, `askText`, `askYesNo`, `askMenu`, `askNumber`, `askAvatar`, `askPet`, `askBoxText`. They may be modeled as constructor methods on a single `NpcConversation` struct (no separate type per branch) — in that case the registry will only know `NpcConversation`, not per-branch types. Confirm by reading the file once and record the actual shape in a one-line comment on the new test (used to decide whether per-type registration is even possible).

- [ ] **Step 2: Write the fixture (single-type case)**

```go
func TestRegistryRegistersNpcConversation(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    // NpcConversation is a single struct with multiple constructor methods —
    // one per dialog type. The registry surfaces one Encode call list; per-branch
    // verification lives in the audit report (Task 14, per-section breakdown).
    if !reg.HasType("NpcConversation") {
        t.Fatalf("registry missing NpcConversation")
    }
    calls, ok := reg.Calls("NpcConversation")
    if !ok || len(calls) == 0 {
        t.Fatalf("NpcConversation.Encode produced no calls (ok=%v len=%d)", ok, len(calls))
    }
}
```

If the file ships per-type structs (e.g. `NpcConversationSay`, `NpcConversationAskText`, …) — uncommon, but verify in step 1 — replace the fixture with a per-type loop similar to `TestRegistryRegistersMovementElements` from task-028.

- [ ] **Step 3: Run the fixture**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersNpcConversation -v
```

Expected: PASS.

- [ ] **Step 4: Cross-domain regression**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
diff /tmp/summary-pre-task068.md docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: no diff. Same STOP rule as Task 2 step 4.

- [ ] **Step 5: Commit**

```
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert NpcConversation registry coverage"
```

---

## Phase 2 — World v95 audit

Eight tracking sub-tasks (Tasks 4–11 for sub-phases 2a–2h plus tasks 12–14 for the deeper packets). Each sub-task is a tracking unit, NOT a single PR commit:

1. Add the world-domain operation entries to the v95 template if missing (one-shot for the first packet in each sub-domain).
2. Run the audit against the sub-task's packet bucket.
3. Triage each report: ✅ (no fix needed), ⚠️ (tolerable mismatch — annotate report), ❌ (real wire bug OR template drift OR analyzer descent gap).
4. Ship **one fix commit per ❌** with a 4-variant test sweep and an IDA citation in the commit message.
5. Each `_pending.md` deferral lands as a separate row + a one-line commit.
6. Bucket commit closes the sub-task — audit reports + SUMMARY + IDA-export updates batched.

The audit command for all of Phase 2 is the same:

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

It produces per-packet reports flat under `docs/packets/audits/gms_v95/<PacketName>.{md,json}` (design §10 — flat layout, world rows interleave with login + character) and updates `SUMMARY.md`. Run it once per sub-task after each fix commit; commit reports alongside the bucket commit.

Before starting Phase 2, the user must have v95 IDA loaded so `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the bucket audit reports.

### World-domain template seeding — one-time scaffolding

The v95 template `template_gms_95_1.json` currently lists only login + character writer entries (verified at plan-task time: `grep -o '"writer": ..."' template_gms_95_1.json | sort -u` shows 19 entries, none with `Field`/`Npc`/`Portal` prefixes). World-domain `Operation()` strings include `"SetField"`, `"NPCConversation"`, etc. (see `libs/atlas-packet/field/clientbound/warp_to_map.go:16`, `libs/atlas-packet/npc/clientbound/conversation.go:13`).

Each Phase 2 sub-task that adds a new writer to the template lists the new entries in the bucket commit. Reuse the existing entry schema:

```json
{
  "opCode": "0x6B",
  "validator": "InMapValidator",
  "handler": "FieldChangeHandle",
  "writer": "SetField"
}
```

Opcode values come from IDA case-statement decompile in the `CWvsApp::OnPacket`-equivalent dispatcher. Do not invent opcodes — `mcp__ida-pro__decompile_function` against the dispatcher is the source of truth.

### Task 4: Sub-phase 2a — `portal/serverbound` bucket (2 files)

**Packets:** `script.go` plus its test companion.

Smallest sub-domain — audit-and-go. Expected outcome: 2 ✅ rows. If anything else, document the surprise in the bucket commit.

- [ ] **Step 1: Add the `PortalScript` writer to the v95 template if missing**

```
grep -E '"writer": *"PortalScript"' services/atlas-configurations/seed-data/templates/template_gms_95_1.json
```

If empty: locate the serverbound portal opcode via IDA (`mcp__ida-pro__get_function_by_name("CField::OnPortalScriptRequest")` or equivalent) and append a handler entry. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register PortalScript handler for v95

IDA case-statement at <CField::OnPortalScriptRequest>@<addr>."
```

- [ ] **Step 2: Append the FName to `docs/packets/ida-exports/gms_v95.json`**

Add the matching `Decode*`-op list (use `mcp__ida-pro__decompile_function` to read the per-handler decoder).

- [ ] **Step 3: Run the audit (full command above)**

Inspect the new `PortalScript.md` (or whatever the FName resolves to) under `docs/packets/audits/gms_v95/`.

- [ ] **Step 4: Triage**

Expected: ✅. If ❌:
- Width/order/missing field → fix `libs/atlas-packet/portal/serverbound/script.go` with a 4-variant test sweep.
- Opcode drift → fix `template_gms_95_1.json`.
- Bare handler (no atlas-packet decoder) → defer to `_pending.md`.

- [ ] **Step 5: For each Atlas wire-bug fix, add a 4-variant sweep test**

Reference shape from `libs/atlas-packet/login/clientbound/auth_success_test.go:9-37`:

```go
func TestPortalScriptDecode(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), v.Tenant)
            raw, _ := hex.DecodeString(v.PortalScriptHex) // hex captured from IDA
            r := request.NewReader(raw)
            m := DecodePortalScript(testLogger(), ctx)(r)
            if r.Available() != 0 {
                t.Fatalf("leftover bytes after decode: %d", r.Available())
            }
            _ = m
        })
    }
}
```

If `pt.Variants` doesn't carry per-test hex fixtures, hard-code the four hex strings inline.

- [ ] **Step 6: Run the affected package's tests**

```
go test -race ./libs/atlas-packet/portal/...
```

Expected: clean.

- [ ] **Step 7: Commit each fix individually**

For atlas-packet fixes:

```
git add libs/atlas-packet/portal/serverbound/script.go libs/atlas-packet/portal/serverbound/script_test.go
git commit -m "fix(atlas-packet,portal/script): <one-line summary>

Cites IDA <CField::OnPortalScriptRequest>@<addr>: <one-line evidence>."
```

For template fixes:

```
git add services/atlas-configurations/seed-data/templates/template_*.json
git commit -m "fix(configurations,templates): portal opcode <old>→<new>

IDA case-statement value at <dispatcher>@<addr>."
```

For `_pending.md` deferrals:

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(portal/script): defer — <one-line reason>"
```

- [ ] **Step 8: Bucket commit — audit reports + SUMMARY + IDA export**

```
git add docs/packets/audits/gms_v95/PortalScript.md docs/packets/audits/gms_v95/PortalScript.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(world): sub-phase 2a portal/serverbound bucket (2 files)"
```

(Adjust filenames to actual FNames produced.) Append the ack footer to each `.md` report as the LAST step before this commit:

```
echo "" >> docs/packets/audits/gms_v95/PortalScript.md
echo "Ack: world-audit Phase 2a on $(date -I)" >> docs/packets/audits/gms_v95/PortalScript.md
```

- [ ] **Step 9: Exit gate**

`grep -E '^\| \[(Portal)' docs/packets/audits/gms_v95/SUMMARY.md` must show 1 row (the `script` packet — its `_test.go` companion is not audited). The row must have a verdict ✅, ⚠️, or ❌, and any ❌ must have a matching fix commit OR `_pending.md` deferral.

---

### Task 5: Sub-phase 2b — `field/serverbound` bucket (1 file)

**Packets:** `change.go` plus its test companion.

Single-file pass. The serverbound field-change packet is consumed during map transition acks.

- [ ] **Step 1: Add `FieldChange` (or actual handler-name) entry to the v95 template if missing**

Same pattern as Task 4 step 1. The handler is dispatched from `CWvsContext::OnUserTransferFieldRequest` (or equivalent — confirm via IDA).

- [ ] **Step 2: Append the FName to `gms_v95.json`** (Decode-op list from IDA).

- [ ] **Step 3: Run the audit (full command above)**

- [ ] **Step 4: Triage** (same flavours as Task 4 step 4).

- [ ] **Step 5: For each fix, add a 4-variant Decode round-trip sweep**

```go
func TestFieldChangeDecode(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), v.Tenant)
            raw, _ := hex.DecodeString(v.FieldChangeHex)
            r := request.NewReader(raw)
            m := DecodeChange(testLogger(), ctx)(r)
            if r.Available() != 0 {
                t.Fatalf("leftover bytes after decode: %d", r.Available())
            }
            _ = m
        })
    }
}
```

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/field/serverbound/...
```

- [ ] **Step 7: Commit each fix individually** (same format as Task 4 step 7).

- [ ] **Step 8: Bucket commit**

```
git add docs/packets/audits/gms_v95/FieldChange.md docs/packets/audits/gms_v95/FieldChange.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(world): sub-phase 2b field/serverbound bucket (1 file)"
```

(Adjust filenames to actual FNames; append ack footer per Task 4 step 8.)

- [ ] **Step 9: Exit gate** — `grep -E 'FieldChange' docs/packets/audits/gms_v95/SUMMARY.md` shows 1 row with a verdict.

---

### Task 6: Sub-phase 2c — `field/clientbound` non-effect bucket (8 files)

**Packets:** `affected_area_created`, `affected_area_removed`, `kite_destroy`, `kite_error`, `kite_spawn`, `set_field`, `transport`, `warp_to_map`.

This is the **version-density hotspot**. `set_field.go` is the canonical 4-sibling-guard envelope (4 sibling `Region/MajorVersion` checks at depth 1; design §5 grants up to 3-deep nesting for this file only). `warp_to_map.go` and `transport.go` are also version-sensitive.

- [ ] **Step 1: Add world-clientbound writer entries to the v95 template if missing**

The relevant `Operation()` strings are:
- `SetField` (used by both `SetField` and `WarpToMap` — see `warp_to_map.go:16-40`).
- `Transport` (per `field/clientbound/transport.go`).
- `Kite*` writers, `AffectedArea*` writers (read each file's `Operation()`).

For each writer not already in `template_gms_95_1.json`, append a handler entry with the opcode from IDA. Commit:

```
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json
git commit -m "feat(configurations,templates): register field/clientbound writers for v95

Opcodes from IDA <CWvsApp::OnSendPacket> dispatcher cases."
```

- [ ] **Step 2: Append the FNames to `gms_v95.json`**

For each clientbound writer, locate the matching IDA function (`CClientSocket::Send*` or `CField::Send*`) and add a `Decode*`-op list.

For `set_field.go` specifically: per design §3 last paragraph, audit the **envelope only**. The embedded `m.characterData.Encode(...)` call descends into `CharacterData`, which is already audited under the character domain (task-028). The envelope-only audit verifies the bytes *around* the `CharacterData.Encode` call; the inner shape is reused. If the analyzer surfaces an unresolved type for the `CharacterData` sub-call, that is expected and gets a `🔍 envelope-only` annotation in the audit report header — NOT a registry addition.

- [ ] **Step 3: Run the audit (full command above)**

- [ ] **Step 4: Triage per-packet**

For each ❌:
- **Atlas wire bug** — fix `libs/atlas-packet/field/clientbound/<pkt>.go` + 4-variant test sweep.
- **Template opcode drift** — fix every affected `template_gms_*_1.json` and `template_jms_185_1.json`.
- **Analyzer descent gap** (e.g. unresolved sub-struct type) → register the sub-struct (one-line analyzer registry addition with a new fixture in `registry_test.go`) THEN re-run. If it's `CharacterData` for `set_field` envelope, annotate `🔍 envelope-only` and move on.
- **Bare handler** — defer to `_pending.md` under a new `## Still pending — world domain` heading (create the heading on first deferral).

**Nesting policy specific to this bucket**:
- `set_field.go` may grow to 3-deep nesting if IDA confirms three orthogonal axes (region × major × tertiary). 4+ → STOP, defer.
- Every other file in this bucket stays at 2-deep max. 3+ → STOP, defer.
- For `set_field.go`, prepend a one-line header to the audit report after Step 8:

```
echo "Nesting policy: 3-deep exception per PRD §4.6 / design §5." >> docs/packets/audits/gms_v95/SetField.md
```

- [ ] **Step 5: For each Atlas wire-bug fix, add a 4-variant Encode sweep test**

Reference shape (per task-028 `TestSpawnByteForByte`):

```go
func TestSetFieldEncode(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), v.Tenant)
            got := NewSetField(channelId, characterData).Encode(testLogger(), ctx)(nil)
            // Compare against per-version expected hex captured from IDA wire shape.
            want, _ := hex.DecodeString(v.SetFieldHex)
            if !bytes.Equal(got, want) {
                t.Fatalf("encode mismatch\n got %x\nwant %x", got, want)
            }
        })
    }
}
```

For files where Encode-byte-equality is impractical (random damage seeds in `SetField`), substitute a sub-segment comparison: assert the version-gated prefix/suffix matches and skip the random-seed range.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/field/clientbound/...
```

- [ ] **Step 7: Commit each fix individually** (same format as Task 4 step 7).

- [ ] **Step 8: Bucket commit**

```
git add docs/packets/audits/gms_v95/{SetField,WarpToMap,Transport,AffectedArea*,Kite*,FieldEffect*}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(world): sub-phase 2c field/clientbound non-effect bucket (8 files)"
```

Append ack footers to each `.md` per Task 4 step 8.

- [ ] **Step 9: Exit gate** — `grep -E '\| \[(SetField|WarpToMap|Transport|AffectedArea|Kite)' docs/packets/audits/gms_v95/SUMMARY.md` shows ≥ 8 rows (some packets emit > 1 writer; see `effect.go` for 5 sub-structs in the next bucket). Every ❌ has a fix commit or `_pending.md` row.

---

### Task 7: Sub-phase 2d — `field/clientbound` effect cluster (3 files)

**Packets:** `effect.go`, `effect_weather.go`, `clock.go`.

`effect.go` is the **good form** of sub-op dispatch (5+ separate Go encoder structs: `EffectSummon`, `EffectTremble`, `EffectString`, `EffectBossHp`, `EffectRewardRullet`). Each gets its own audit row.

`effect_weather.go` and `clock.go` are the **bad form** (one struct, mode byte set in constructor, single `Encode` method). Per design §8 these expect a ❌ tool-limitation verdict and a manual sub-op annotation block, no refactor.

- [ ] **Step 1: Add effect/clock writer entries to the v95 template if missing** (same pattern as Task 4 step 1; opcodes from `CField::OnFieldEffect`, `CField::OnFieldWeather`, `CField::OnClock`).

- [ ] **Step 2: Append the FNames to `gms_v95.json`**

For `effect.go`, each of the 5+ effect-type variants gets a separate IDA-export entry keyed by the field-effect type discriminator value (read from `CField::OnFieldEffect` case-statement).

For `effect_weather.go` and `clock.go`, register the per-mode IDA cases under entries annotated `"sub_op": <value>` in `gms_v95.json` (use the existing schema's optional fields if present; otherwise embed the sub-op in the function comment field).

- [ ] **Step 3: Run the audit**

- [ ] **Step 4: Triage**

For each ❌:
- `effect.go` per-struct ❌ → fix the matching encoder + 4-variant sweep.
- `effect_weather.go` / `clock.go` ❌ with `tool-limitation` flavour → annotate the audit report with a per-mode sub-op table (mode-byte → IDA case-statement value → expected payload):

```
## Sub-op enum drift (tool limitation)
| Mode byte | Constructor | IDA case | Expected payload |
|---|---|---|---|
| 0x00 | `NewEffectWeatherInactive` | `CField::OnFieldWeather@<addr>:case 0` | `Decode4 itemId` |
| 0x01 | `NewEffectWeatherActive` | `CField::OnFieldWeather@<addr>:case 1` | `Decode4 itemId` + `DecodeStr message` |
```

- Field-effect types in IDA with no atlas struct → defer to `_pending.md` under "Sub-op enum drift — world domain" (create heading on first hit; mirror task-028 pattern).
- Real wire bug (independent of sub-op limitation) → fix encoder + 4-variant sweep.

- [ ] **Step 5: For each Atlas wire-bug fix, add a 4-variant sweep test** (Encode-side; same shape as Task 6 step 5).

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/field/clientbound/...
```

- [ ] **Step 7: Commit each fix individually**

For `_pending.md` deferrals on sub-op limitations:

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(field/effect_weather): defer sub-op enum modeling — tool limitation"
```

- [ ] **Step 8: Bucket commit**

```
git add docs/packets/audits/gms_v95/{FieldEffect,Effect*,Clock}*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(world): sub-phase 2d field/clientbound effect cluster (3 files)"
```

Append ack footers.

- [ ] **Step 9: Exit gate** — ≥ 5 effect rows (one per `effect.go` sub-struct) + 1 each for `effect_weather`, `clock` in SUMMARY.

---

### Task 8: Sub-phase 2e — `npc/clientbound` non-conversation bucket (6 files)

**Packets:** `action`, `guide_talk`, `shop_list`, `shop_operation`, `shop_operation_body`, `spawn`, `spawn_request_controller`.

`shop_list.go` is the **loop-count limitation case**. The commodity-array loop flattens incorrectly through the analyzer (same shape as task-028's `KeyMapChange`). Expected verdict: ❌ with a "loop-count tool limitation" annotation; manually verify per-iteration bytes + the loop bound against IDA's `CUserShopDlg::SendShopList` (or equivalent).

- [ ] **Step 1: Add NPC clientbound writer entries to the v95 template if missing**

Operation strings: `NPCAction`, `NPCSpawn`, `NPCShopList`, `NPCShopOperation`, etc. (read `Operation()` per file). Opcodes from IDA `CClientSocket::SendNpc*` or `CUserPool::SendNpc*` dispatcher.

- [ ] **Step 2: Append FNames to `gms_v95.json`**

- [ ] **Step 3: Run the audit**

- [ ] **Step 4: Triage**

For `shop_list.go` specifically:
- Read `shop_list.go` against IDA `CUserShopDlg::SendShopList` decompile.
- Confirm the commodity-count limit (likely 16 or 32 per shop). Document expected count in the audit report.
- Verify the per-item entry sub-struct (NpcShopItem) bytes match IDA per-iteration shape.
- The analyzer's flat verdict will be ❌ on the loop body; the audit report MUST contain a manual "Loop bounds verified against IDA" annotation:

```
## Loop bounds (tool limitation)
Verified against `CUserShopDlg::SendShopList@<addr>`:
- Max commodity entries: <N> (atlas: `len(m.items)` capped at <M>).
- Per-entry shape: `Decode4 itemId, Decode4 price, ...` (matches atlas's per-item Encode).
```

For other packets, standard triage flavours from Task 4 step 4.

- [ ] **Step 5: For each Atlas wire-bug fix, add a 4-variant sweep test**

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/npc/clientbound/...
```

- [ ] **Step 7: Commit each fix individually**

- [ ] **Step 8: Bucket commit**

```
git add docs/packets/audits/gms_v95/{NPCAction,NPCGuideTalk,NPCShop*,NPCSpawn*}*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(world): sub-phase 2e npc/clientbound non-conversation bucket (6 files)"
```

Append ack footers.

- [ ] **Step 9: Exit gate** — ≥ 6 NPC clientbound rows in SUMMARY (one per file), every ❌ resolved.

---

### Task 9: Sub-phase 2f — `npc/clientbound/conversation.go` (1 file, 8 sub-encoders)

**Packet:** `conversation.go` (360 lines, 8 dialog-type sub-encoder branches).

Per PRD §4.5 and design §6: produce ONE audit report `docs/packets/audits/gms_v95/NPCConversation.md` containing per-dialog-type sub-sections, each with its own verdict. SUMMARY uses the worst verdict across sub-sections. Do NOT refactor `conversation.go` into per-type files (PRD §3 non-goal).

The 8 dialog types per design §6:
1. `say`
2. `askText`
3. `askYesNo`
4. `askMenu`
5. `askNumber`
6. `askAvatar`
7. `askPet`
8. `askBoxText`

- [ ] **Step 1: Read `npc/clientbound/conversation.go` end-to-end and map per-type encoder blocks**

```
sed -n '1,360p' libs/atlas-packet/npc/clientbound/conversation.go
```

Record the line-range of each per-type encoder block in the audit report (post-Step 4 manual annotation).

- [ ] **Step 2: Append `NPCConversation` FName to `gms_v95.json` with per-case sub-functions**

For each dialog type, locate the matching IDA branch (likely all under one `CUser::OnQuestionAsk` or `CUserPool::OnNpcConversation`-equivalent function — read the dispatcher to confirm). Record each per-case Decode-op list with a `"sub_op": <value>` annotation matching the leading text-type byte:

```json
{
  "fname": "CUser::OnQuestionAsk",
  "address": "0x...",
  "direction": "clientbound",
  "calls": [...],
  "sub_ops": {
    "0x00": {"comment": "say", "calls": [...]},
    "0x01": {"comment": "askYesNo", "calls": [...]},
    "0x02": {"comment": "askText", "calls": [...]},
    "0x03": {"comment": "askNumber", "calls": [...]},
    "0x04": {"comment": "askMenu", "calls": [...]},
    "0x05": {"comment": "askAvatar", "calls": [...]},
    "0x06": {"comment": "askPet", "calls": [...]},
    "0x07": {"comment": "askBoxText", "calls": [...]}
  }
}
```

If the existing `gms_v95.json` schema doesn't support `sub_ops`, embed the sub-op tables in the function `comment` field as a plain-text block and document the limitation in the report. Do NOT extend the analyzer/diff engine to consume them — Phase 2f's verification is human-driven.

- [ ] **Step 3: Run the audit**

The analyzer will produce one flat verdict for `NPCConversation` (almost certainly ❌ — mode-byte sub-op dispatch). That's expected. The auto-generated report becomes the skeleton.

- [ ] **Step 4: Append the per-dialog-type breakdown to `docs/packets/audits/gms_v95/NPCConversation.md`**

```markdown
## Per-dialog-type breakdown

### say (text-type 0x00)
- **Verdict:** ✅ / ⚠️ / ❌
- **IDA dispatcher branch:** `CUser::OnQuestionAsk@<addr>:case 0`
- **Atlas encoder block:** `conversation.go:<startLine>-<endLine>`
- **Wire-shape comparison:**
  - Atlas: `WriteByte(npcType), WriteInt(npcId), WriteByte(0x00), WriteCString(message), WriteByte(prev), WriteByte(next)`
  - IDA: `Decode1 npcType, Decode4 npcId, Decode1 sub_op, DecodeStr message, Decode1 prev, Decode1 next`
- **Fix block:** none / (if ❌) summary of what changed.

### askText (text-type 0x02)
- **Verdict:** …
…
```

Each per-type sub-section follows this skeleton. For unresolvable branches (text-type byte assembled from a runtime parameter that the analyzer can't statically resolve), defer that branch to `_pending.md` and mark the sub-section ⚠️ with a one-line rationale:

```
- **Verdict:** ⚠️ — text-type byte set at construction time via `WithType(...)`. Static resolution not feasible; per-case bytes verified manually against IDA.
```

- [ ] **Step 5: For each ❌ sub-section, ship a per-branch fix**

Fix lands in `conversation.go` (the per-type encoder block); test lands in `conversation_test.go` as an independent 4-variant sweep for that branch:

```go
func TestNpcConversationSayEncode(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), v.Tenant)
            got := NewNpcConversationSay(npcId, message, hasPrev, hasNext).Encode(testLogger(), ctx)(nil)
            want, _ := hex.DecodeString(v.ConvSayHex)
            if !bytes.Equal(got, want) {
                t.Fatalf("encode mismatch\n got %x\nwant %x", got, want)
            }
        })
    }
}
```

Tests are named `TestNpcConversation<Type>Encode` per dialog type. A fix on `askMenu` does NOT blanket-test `say`.

- [ ] **Step 6: Run tests**

```
go test -race ./libs/atlas-packet/npc/clientbound/...
```

- [ ] **Step 7: Commit each fix individually**

```
git add libs/atlas-packet/npc/clientbound/conversation.go libs/atlas-packet/npc/clientbound/conversation_test.go
git commit -m "fix(atlas-packet,npc/conversation): <dialog-type> wire shape — <one-line>

Cites IDA <CUser::OnQuestionAsk>@<addr>:case <subOp>: <evidence>."
```

For `_pending.md` deferrals on unresolvable branches:

```
git add docs/packets/ida-exports/_pending.md
git commit -m "audit(npc/conversation,<type>): defer — unresolvable text-type byte"
```

- [ ] **Step 8: Bucket commit — audit report + SUMMARY**

```
git add docs/packets/audits/gms_v95/NPCConversation.md docs/packets/audits/gms_v95/NPCConversation.json \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(world): sub-phase 2f npc/conversation per-dialog-type (1 file, 8 sub-encoders)"
```

Append the ack footer LAST:

```
echo "" >> docs/packets/audits/gms_v95/NPCConversation.md
echo "Ack: world-audit Phase 2f on $(date -I); per-type breakdown verified across 8 dialog types." >> docs/packets/audits/gms_v95/NPCConversation.md
git commit -am "audit(world): ack footer for npc/conversation"
```

- [ ] **Step 9: Exit gate**

`grep -E 'NPCConversation' docs/packets/audits/gms_v95/SUMMARY.md` shows 1 row with verdict = worst-of-8. The report file contains 8 per-type sub-sections, each with its own verdict + IDA citation + wire comparison.

If more than 2 of 8 branches required deferral to `_pending.md`, the file's row in SUMMARY is ⚠️ overall with the unresolved branches enumerated (per PRD §4.5).

---

### Task 10: Sub-phase 2g — `npc/serverbound` bucket (9 files)

**Packets:** `action`, `continue_conversation`, `continue_conversation_selection`, `continue_conversation_text`, `shop`, `shop_buy`, `shop_recharge`, `shop_sell`, `start_conversation`.

**NPC dispatcher offset** (design §7): `CUserPool::OnPacket` (or equivalent) prepends `characterId` (4 bytes) before per-handler decoders for NPC actions. Atlas-side decoders must either include `characterId` at offset 0 OR consistently treat it as already-consumed. The audit's job is to **document the boundary** (audit report header per packet) and verify the post-`characterId` payload against IDA.

- [ ] **Step 1: Identify the dispatcher offset**

```
mcp__ida-pro__get_function_by_name("CUserPool::OnPacket")
mcp__ida-pro__decompile_function(<addr>)
```

Confirm whether `characterId` is consumed at the dispatcher layer or per-handler. Record the finding (one line) in `context.md` for the rest of the bucket to reference.

- [ ] **Step 2: Add NPC serverbound handler entries to the v95 template if missing**

Operation strings come from per-file `Operation()`. Opcodes from the dispatcher case-statements (`CUserPool::OnPacket` switch).

- [ ] **Step 3: Append FNames to `gms_v95.json`**

For each NPC serverbound packet, locate `CWvsContext::OnUserNpc<Action>` or equivalent per-handler function and add a `Decode*`-op list. Annotate the dispatcher offset in each entry's comment:

```json
{
  "fname": "CWvsContext::OnUserScriptMessageAnswer",
  "address": "0x...",
  "direction": "serverbound",
  "dispatcher_offset": "characterId consumed by CUserPool::OnPacket; decoder starts at offset 4 conceptually",
  "calls": [{"op": "Decode1", "comment": "endType"}, ...]
}
```

- [ ] **Step 4: Run the audit**

- [ ] **Step 5: Triage per packet**

Possible findings:
- **Atlas decoder consistently includes `characterId` at offset 0** AND IDA per-handler starts after the prepend → ✅ with a header-line ack of the dispatcher offset.
- **Atlas decoder consistently omits `characterId`** AND atlas-channel handler reads `characterId` from the dispatcher context → ✅ with a header-line ack.
- **Inconsistency between two packets** in this bucket (one includes, one omits) → real bug; fix the inconsistent one + 4-variant Decode sweep.
- **Width / order / missing field on the post-`characterId` payload** → standard wire-bug fix.
- **Bare handler with no atlas-packet decoder** → defer to `_pending.md`.

- [ ] **Step 6: For each Atlas wire-bug fix, add a 4-variant Decode round-trip sweep** (same shape as Task 5 step 5).

- [ ] **Step 7: Run tests**

```
go test -race ./libs/atlas-packet/npc/serverbound/...
```

- [ ] **Step 8: Commit each fix individually**

- [ ] **Step 9: Bucket commit**

```
git add docs/packets/audits/gms_v95/{NPCAction,NPCStartConversation,NPCContinueConversation*,NPCShop*}*.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json \
        docs/packets/ida-exports/_pending.md
git commit -m "audit(world): sub-phase 2g npc/serverbound bucket (9 files)"
```

Append ack footers. For each packet's report, prepend a one-line header documenting the dispatcher-offset finding from Step 1.

- [ ] **Step 10: Exit gate** — ≥ 9 NPC serverbound rows in SUMMARY (test companion files are not audited). Every ❌ resolved. The dispatcher-offset finding is documented identically across every report in the bucket (cross-packet consistency check).

---

### Task 11: Sub-phase 2h — `_pending.md` sweep

Phase 2 closes with a consolidation pass on `docs/packets/ida-exports/_pending.md`. The goal: every deferred row from sub-phases 2a–2g lives under a clean per-domain heading; the file is a one-glance ledger reviewers can scan in Task 22.

- [ ] **Step 1: Inventory all world-domain deferrals**

```
grep -nE '^\| (field|portal|npc)/' docs/packets/ida-exports/_pending.md
```

Expected: rows added during Tasks 4–10. Count by sub-domain.

- [ ] **Step 2: Add or confirm section headings**

Open `_pending.md` and ensure these headings exist in order:

```markdown
## Still pending — world domain
| Packet | Reason | Reference |
|---|---|---|

## Sub-op enum drift — world domain
| Packet | Sub-op | Constructor | IDA case |
|---|---|---|---|

## Tool limitations — world domain
| Packet | Limitation | Workaround |
|---|---|---|
```

Reorganize existing rows under the right heading. Do NOT touch login or character sections (regression risk).

- [ ] **Step 3: Cross-reference deferrals against audit reports**

For every `_pending.md` row, the matching audit report under `docs/packets/audits/gms_v95/` must reference the deferral by name (e.g. "Sub-op enum modeling deferred — see `_pending.md` §Sub-op enum drift — world domain"). Add the reference line to any report missing it.

- [ ] **Step 4: Commit**

```
git add docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/*.md
git commit -m "audit(world): consolidate _pending.md sweep for sub-phase 2h"
```

- [ ] **Step 5: Exit gate — Phase 2 complete**

```
grep -c '❌' docs/packets/audits/gms_v95/SUMMARY.md
grep -c '⚠️' docs/packets/audits/gms_v95/SUMMARY.md
grep -c '✅' docs/packets/audits/gms_v95/SUMMARY.md
```

Total world rows: 57 (sub-phases 2a + 2b + 2c + 2d + 2e + 2f + 2g; some files emit multiple writer rows so the count is a floor, not a ceiling). Every ❌ has either a fix commit on this branch (`git log --oneline | grep '^[a-f0-9]* fix(atlas-packet'`) OR a `_pending.md` row.

Prior login (28) + character (52) rows are still byte-identical to pre-task state:

```
diff /tmp/summary-pre-task068.md <(grep -vE '^\| \[(SetField|WarpToMap|Transport|AffectedArea|Kite|FieldEffect|FieldChange|PortalScript|Clock|NPC)' docs/packets/audits/gms_v95/SUMMARY.md)
```

Expected: no diff. If anything drifts, STOP and investigate before Phase 3.

---

## Phase 3 — Cross-version pass

Three tracking sub-tasks (Tasks 12–14). One IDA binary at a time, user-driven swap. Each sub-task is "done" when:
- `docs/packets/ida-exports/<version>.json` has world-domain entries for every FName from the v95 audit.
- The audit has been re-run against the version's template + IDA export.
- Every divergence vs v95 atlas-packet behaviour has either:
  - A `Region/MajorVersion` gate that already handles it (audit report captures evidence; no code change),
  - A gate fix on this branch with a 4-variant test sweep, OR
  - A template fix.

If a packet on a non-v95 version needs structural rewriting (>2 nested region/version guards, except `set_field.go` capped at 3-deep), STOP, log to `_pending.md`, and continue.

### Task 12: GMS v83 cross-version pass

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (exists from task-027/028; append world entries)
- Modify (per fix): `libs/atlas-packet/{field,portal,npc}/**/*.go` + matching `_test.go`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Create or modify: `docs/packets/audits/gms_v83/` per-packet world reports + `SUMMARY.md` (follow whatever layout task-028 chose — likely a sibling `gms_v83/` directory; if reviewers prefer flat, restructure on first run and be consistent)

- [ ] **Step 1: Confirm v83 IDA is loaded**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v83. If not, ask user to swap before continuing.

- [ ] **Step 2: For each world FName resolved during Phase 2, populate `gms_v83.json`**

Workflow per FName:
1. `mcp__ida-pro__get_function_by_name("<FName>")` → resolve address.
2. `mcp__ida-pro__decompile_function(<addr>)` → read Decode op list.
3. Translate to the existing `gms_v83.json` schema with the same op shape used during Phase 2.

Do not reorder existing login/character entries.

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

If the `gms_v83/` audit directory doesn't exist, the tool creates it. Match the existing per-version directory choice (task-028 used `gms_v83/` if its Phase 3 created one; otherwise this task originates the directory).

- [ ] **Step 4: Triage divergences**

For each ❌ in the v83 audit:
- **v95 fix was already gated on `MajorVersion() >= 95`** → no v83 regression. Audit-report-only.
- **v95 fix was gated on `Region() == "GMS"`** (no major-version filter) → check whether v83 IDA confirms the same behaviour. If yes: tighten the gate so v83 keeps its old shape. If no: leave as-is and document.
- **New v83-only mismatch the v95 audit didn't surface** → genuine cross-version bug. Fix with a 4-variant test sweep + `Region/MajorVersion` gate.

- [ ] **Step 5: For each fix, add a 4-variant sweep test** that proves the fix doesn't regress v87/v95/JMS-185.

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

If any single world-domain encoder now contains 3+ nested `Region`/`MajorVersion` levels — except `set_field.go` capped at 3-deep — STOP per design §5. Append a row to `_pending.md` describing the encoder + which version chain triggered it. Do not refactor in this task.

- [ ] **Step 9: Bucket commit**

```
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/ \
        services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "audit(world): GMS v83 cross-version pass (world domain)"
```

- [ ] **Step 10: Exit gate** — `docs/packets/audits/gms_v83/SUMMARY.md` has 57+ world-domain rows; every ❌ has a fix commit or `_pending.md` row.

---

### Task 13: GMS v87 cross-version pass

Identical shape to Task 12. Replace `v83` with `v87` everywhere. Templates: `template_gms_87_1.json`. Export file: `docs/packets/ida-exports/gms_v87.json` (created during task-028 Phase 3 — append world entries).

- [ ] **Steps 1–10: Same shape as Task 12.**

Bucket commit message:

```
audit(world): GMS v87 cross-version pass (world domain)
```

---

### Task 14: JMS v185 cross-version pass

JMS v185 is the **highest-attention version** per design §9. `set_field.go` already has explicit JMS branches at depth 1; v185 IDA may reveal JMS sub-divides further (early-JMS pre-185 vs 185+). If so, the v185 pass produces the first known case of `Region() == "JMS"` requiring a major-version split.

**Files:**
- Modify: `docs/packets/ida-exports/gms_jms_185.json` (created in task-028; append world entries)
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Possibly modify: `libs/atlas-packet/{field,portal,npc}/**/*.go` per the §5 + §7 policy
- Create or modify: `docs/packets/audits/jms_v185/` per-packet world reports + `SUMMARY.md`

- [ ] **Step 1: Confirm JMS v185 IDA is loaded**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches JMS v185. If not, ask user to swap.

- [ ] **Step 2: Populate `gms_jms_185.json` for the world FNames from Phase 2**

If a FName has no JMS equivalent (different opcode space or different code-path entry), record the JMS-side FName + address as a separate entry annotated `"region": "JMS"`. Do NOT reuse GMS FNames for unrelated JMS functions.

- [ ] **Step 3: Re-run the audit**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage per design §7.1 (task-028 pattern)**

- **In scope:** atlas-packet writes bytes the JMS client decodes wrong.
- **Out of scope:** JMS-specific feature the service doesn't wire through.
- **In scope:** width mismatch on a field both versions decode.
- **Out of scope:** JMS template opcode wrong when v95 is right (fix the template, atlas-packet untouched).

- [ ] **Step 5: If `set_field.go` needs a structural change (4+ nesting)**

Per design §5 hard cap: STOP. Append to `_pending.md`:

```
## set_field JMS sub-major-version split (deferred)
| Encoder | Trigger | Defer reason |
|---|---|---|
| `set_field.go` | JMS v185 IDA reveals early-JMS vs 185+ field-limits divergence inside an already-`Region() == "JMS"` block | Hard-cap 3-deep nesting exceeded; structural rewrite out of scope per design §5 |
```

Do not refactor. The follow-up task picks this up.

- [ ] **Step 6: For each in-scope fix, add a 4-variant sweep test + gate**

- [ ] **Step 7: Run tests**

```
go test -race ./libs/atlas-packet/...
```

- [ ] **Step 8: Commit each fix individually**

- [ ] **Step 9: Hard-cap check** (same as Task 12 step 8 — 2-deep elsewhere, 3-deep set_field.go).

- [ ] **Step 10: Bucket commit**

```
git add docs/packets/ida-exports/gms_jms_185.json \
        docs/packets/audits/jms_v185/ \
        services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "audit(world): JMS v185 cross-version pass (world domain)"
```

- [ ] **Step 11: Exit gate** — `docs/packets/audits/jms_v185/SUMMARY.md` has 57+ world-domain rows; every ❌ has a fix commit, `_pending.md` row, or set_field-deferral row.

---

## Phase 4 — Closeout

One task. `post-phase-b.md` + full verification + code review + PR.

### Task 15: `post-phase-b.md`, verification, code review, PR

**Files:**
- Create: `docs/tasks/task-068-world-domain-packet-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tallies row)
- Modify: `docs/packets/ida-exports/_pending.md` (final world-domain section state)

- [ ] **Step 1: Write `post-phase-b.md`**

Mirror task-028's structure. Five sections:

```markdown
# Task-068 Post-Phase-B — World-Domain Audit Closeout

## Final state
- Packets audited: 57 (21 field/clientbound + 2 field/serverbound + 2 portal/serverbound + 14 npc/clientbound + 18 npc/serverbound).
- Verdicts (v95): ✅ <n_pass> / ⚠️ <n_warn> / ❌ <n_fail> / 🔍 <n_review> / pending <n_pending>.
- IDA-export coverage: v83 / v87 / v95 / JMS v185 — world FNames populated.

## Real wire bugs fixed
| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit; enumerate via `git log --oneline main..HEAD | grep '^[a-f0-9]* fix(atlas-packet'`)

## Template opcode / enum fixes
| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements
- TypeRegistry fixtures for `NpcShopItem` (Phase 1 Task 2) and `NpcConversation` (Phase 1 Task 3).
- (No analyzer changes — design §3 mandate honoured.)

## Remaining work
| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` `## Still pending — world domain`, `## Sub-op enum drift — world domain`, `## Tool limitations — world domain`, plus any §5 hard-cap stops)

## Cross-version notes
- **v83:** <notable findings, gate changes>
- **v87:** <notable findings>
- **JMS v185:** <set_field split status, NPC dispatcher offset confirmation>
```

Fill in actual numbers and rows from the commit history.

- [ ] **Step 2: Run the full verification matrix**

```
go build ./...
go vet ./libs/atlas-packet/... ./tools/packet-audit/...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All four must be clean.

- [ ] **Step 3: Decide whether `docker build` is required**

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

- [ ] **Step 4: gitleaks scrub**

```
grep -r '/home/' docs/packets/audits/gms_v95/ docs/packets/audits/gms_v83/ docs/packets/audits/gms_v87/ docs/packets/audits/jms_v185/ 2>/dev/null
```

Expected: no output. If any user-home path appears in an audit report, scrub it:

```
sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>
```

Commit:

```
git commit -am "audit(world): scrub absolute user-home paths from world-domain reports"
```

- [ ] **Step 5: Verify login + character SUMMARY rows still byte-identical**

```
diff /tmp/summary-pre-task068.md <(grep -vE '^\| \[(SetField|WarpToMap|Transport|AffectedArea|Kite|FieldEffect|FieldChange|PortalScript|Clock|NPC)' docs/packets/audits/gms_v95/SUMMARY.md)
```

Expected: no diff. If any prior-domain row drifted, STOP and investigate before opening the PR.

- [ ] **Step 6: Commit `post-phase-b.md`**

```
git add docs/tasks/task-068-world-domain-packet-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "docs(task-068): post-phase-b closeout"
```

- [ ] **Step 7: Run code review**

Invoke `superpowers:requesting-code-review`. Allow the orchestration skill to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/`, `tools/packet-audit/`, and `services/atlas-configurations/seed-data/templates/` changes.

Read the resulting `audit.md` and act on every BLOCKER / MAJOR finding before opening a PR. Re-run reviews after fix commits land.

- [ ] **Step 8: Open the PR**

Title: `task-068: world-domain packet audit (v83/v87/v95/JMS185) — field/portal/npc`

Body: short summary + link to `post-phase-b.md` for the full bug ledger. Use `superpowers:finishing-a-development-branch` to drive the PR creation.

---

## Self-review notes

Run through the plan once more with fresh eyes before committing it.

- **Spec coverage** — every PRD §4 functional requirement is covered by an explicit task above:
  - §4.1 coverage matrix → Phase 2 Tasks 4–11 (sub-phases 2a–2h, all 57 files enumerated by sub-domain).
  - §4.2 IDA exports → Phase 2 Task 4–10 step 2 (v95 append) + Phase 3 Tasks 12–14 (v83/v87/JMS185 append).
  - §4.3 wire bug fixes → embedded in every Phase 2/3 task (`fix(atlas-packet,...)` commits with 4-variant sweeps).
  - §4.4 template fixes → embedded in every Phase 2/3 task (`fix(configurations,templates)` commits).
  - §4.5 conversation per-dialog-type → Task 9 (Phase 2f).
  - §4.6 set_field 3-deep exception → Task 6 step 4 nesting policy block + header annotation; rest of the world stays 2-deep.
  - §4.7 TypeRegistry extensions → Tasks 2–3 (Phase 1 high-confidence batch) + per-task additions on-demand during Phase 2.
  - §4.8 cross-version cadence → Tasks 12–14 (one task per version, v95-complete-first).
  - §10 acceptance criteria → Task 15 (Phase 4 closeout, all four verification commands + gitleaks + plan-adherence review).
- **No placeholders** — every step contains either an exact command, an exact code block, or an exact file path. Where IDA addresses or hex strings can't be known until execution (the bulk of this task), the placeholder is annotated with `<addr>` / `<one-line evidence>` / `v.SetFieldHex` and the surrounding step makes the IDA-lookup workflow explicit.
- **Type consistency** — registry-test names (`TestRegistryRegistersNpcShopItem`, `TestRegistryRegistersNpcConversation`) match Phase 1 task numbers. Encoder test names follow `Test<PacketName><Direction>` shape (Encode for clientbound, Decode for serverbound). Sub-encoder tests (`TestNpcConversation<Type>Encode`) follow the per-type shape from Task 9.
- **No analyzer changes** — design §3 mandate honoured throughout. Tasks 2–3 are registry-fixture-only (no `analyzer.go` edits). Task 4 step 4 explicitly defers analyzer descent gaps to one-line registry additions, not analyzer rewrites.
- **No `reflect`, no `interface{}`, no benchmarks** — none of the code in the plan uses `reflect.*` or adds an `interface{}` parameter to an encoder.
- **Cross-domain regression gate** — Phase 0 (Task 1), every Phase 1 task (Tasks 2 step 4, Task 3 step 4), and Phase 4 (Task 15 step 5) each diff against `/tmp/summary-pre-task068.md`. Prior-domain verdicts are protected at every commit point.
- **Nesting cap** — `set_field.go` allowed 3-deep per §4.6/§5; documented in Task 6 step 4 + Task 14 step 5. All other encoders capped at 2-deep — enforced in Task 12/13/14 step 8 "hard-cap check" gates.
- **NPC dispatcher offset** — design §7 — verified once in Task 10 step 1, applied uniformly across the bucket, ack-line in each report header (Task 10 step 9).
- **Gitleaks** — Task 15 step 4 is the mandatory pre-PR scrub. Steps consistently invoke the audit CLI with relative paths.
- **Conversation per-type audit** — Task 9 ships ONE report file with 8 per-type sub-sections; SUMMARY row = worst-of-8 verdict; per-section fix commits + per-section 4-variant tests.
- **Phase 2 sub-phase ordering** — easy → hard per design §16 recommendation (portal → field/serverbound → field/clientbound non-effect → effect cluster → npc/clientbound non-conversation → conversation.go → npc/serverbound → `_pending.md` sweep). Easy wins build pipeline confidence before the high-cognitive-load packets.
- **set_field envelope-only decision** — design §3 last paragraph — Task 6 step 2 documents the envelope-only audit policy for `set_field.go` so the embedded `CharacterData.Encode` does not re-audit character work.
