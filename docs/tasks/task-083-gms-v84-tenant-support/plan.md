# GMS v84 Tenant Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add GMS v84.1 as a new supported tenant version running alongside the existing v83.1 tenant, verified end-to-end by a real v84 client completing login → channel → map → move/chat.

**Architecture:** No new entity, service, or REST endpoint. Five cooperating efforts: (A) IDA-sourced opcode/packet delta discovery → a source-of-truth doc; (B) `tenant.Model` version-predicate helpers + a corrected audit of every version-gated branch; (C) a new `template_gms_84_1.json` seed file authored from v83 plus the delta; (D) operational v84 WZ ingest; (E) a provisioning runbook + live playthrough. Hard dependency order: **A precedes B and C; B+C+D are independent; E requires A+B+C+D all landed.**

**Tech Stack:** Go 1.x microservices, `go.work` workspace, JSON socket-config templates, atlas-configurations seeder, atlas-data WZ-on-MinIO ingest Jobs, IDA-MCP (`select_instance` per IDB), k8s/Grafana MCP for live diagnosis.

---

## Conventions used throughout this plan

- All paths are **relative to the worktree root** (the `task-083-gms-v84-tenant-support` worktree) unless absolute.
- Go commands run with the workspace `go.work` (default). For the redis guard, set `GOWORK=off` per `reference_rediskeyguard_invariant`.
- **The delta doc** `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` is the single source of truth (FR-1.4). Tasks A0–A4 build it; Tasks B/C consume it. Every opcode/predicate decision in code or template must cite a row in this doc.
- IDA harvest tasks use the IDA-MCP server. Per `reference_ida_harvest_subagents`, **only one IDB is loaded per instance** and the user switches instances; use `mcp__ida-pro__list_instances` / `select_instance(port)` to target the v83 / v84 / v95 IDBs. If a required IDB is not loaded, **stop and ask the user to load it** rather than guessing (OQ-7).
- The four implementation phases B/C/D are independent and may be executed in any order or in parallel once Phase A's deliverables exist. Phase E is the final blocking gate.

---

## Phase A — Opcode & Packet Delta Discovery

> Output of this phase is documentation, not code. Each task has a concrete written deliverable and an objective verification step. Per FR-1.3, "same as v83" is a **finding with cited evidence**, never a default.

### Task A0: Scaffold the delta document

**Files:**
- Create: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`

- [ ] **Step 1: Create the doc skeleton with all required sections**

Write the file with exactly these headers (content filled by later tasks):

```markdown
# v83 → v84 Packet / Opcode / Version-Branch Delta

Source of truth for task-083 (FR-1.4). Every code/template change cites a row here.

## 0. IDB inventory & dispatch-table anchors
| IDB | port | dispatch table (inbound) addr | dispatch table (outbound) addr | naming density |
|---|---|---|---|---|

## 1. Inbound (handler) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence (IDB fn/addr or ref version) |
|---|---|---|---|---|

## 2. Outbound (writer) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence |
|---|---|---|---|---|

## 3. Packet-structure delta (FR-1.2)
### 3.1 In-scope flows (exhaustive): login handshake, auth, world/channel list, character list, character select / PIC-PIN, enter-channel, map load (spawn/field), movement, chat
### 3.2 Spot-checked elsewhere (what was checked, what was assumed)

## 4. usesPin determination (OQ-1)

## 5. Version-branch audit table (FR-3.1, FR-3.3)
| branch site (file:line) | predicate | v83 result | v84 result | correct for v84? | action | delta evidence |
|---|---|---|---|---|---|---|

## 6. Provisioning runbook (FR-5.1) + restart sequence (OQ-6)
```

- [ ] **Step 2: Verify the file exists and parses as Markdown**

Run: `test -f docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md && grep -c '^## ' docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`
Expected: prints `7` (seven `## ` sections).

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): scaffold v84 packet/opcode delta document"
```

### Task A1: Confirm IDB availability and locate dispatch tables

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 0)

- [ ] **Step 1: Enumerate loaded IDA instances**

Call `mcp__ida-pro__list_instances`. Confirm the v83, v84, and v95 GMS IDBs are reachable (by port). If any is missing, STOP and ask the user to load it (the v84 IDB is the hard requirement; v83 is the primary anchor; v95 is the tie-breaker).

- [ ] **Step 2: Locate the inbound (recv) and outbound (send) dispatch sites in each IDB**

For each IDB (anchor on v83 where naming is densest), find:
- **Inbound** = the client's recv/`ProcessPacket` opcode switch (server→client packets the client *parses*; these correspond to Atlas **writers**).
- **Outbound** = the client's send sites (client→server packets; these correspond to Atlas **handlers**).

Use `mcp__ida-pro__survey_binary`, `search_text`, `xrefs_to`, and `func_query` to find the dispatch switch. Record each table's address.

- [ ] **Step 3: Fill Section 0 of the delta doc**

Add one row per IDB: port, inbound dispatch addr, outbound dispatch addr, naming density (qualitative: dense / partial / sparse). Note in prose which v84 functions are unnamed `sub_XXXX` (OQ-7 evidence).

- [ ] **Step 4: Verify**

Run: `grep -A4 '## 0. IDB inventory' docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`
Expected: a table with at least the v84 row populated (v83/v95 rows populated if those IDBs were available).

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): record IDB dispatch-table anchors"
```

### Task A2: Harvest + diff inbound (handler) opcodes

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 1)

- [ ] **Step 1: Dump the v84 outbound (client-send = Atlas handler) opcode → function map**

On the v84 IDB, walk the outbound dispatch identified in A1. For each opcode value, capture the opcode and the function (named symbol or `sub_XXXX` + address) via `mcp__ida-pro__decompile` / `callees` / `func_query`.

- [ ] **Step 2: Name-anchor each v84 entry against v83**

For each v84 opcode, find the v83 handler occupying the same logical slot (by structural/byte similarity of the decompiled body, not by trusting the opcode value). Where v83 is ambiguous or the slot shifted, use the v95 IDB as a tie-breaker.

- [ ] **Step 3: Classify and fill Section 1**

For every opcode write one row: `logical name | v83 opcode | v84 opcode | classification | evidence`. Classification is exactly one of:
- `SAME` — v84 opcode == v83 opcode, same handler (cite the v83/v84 fn proving identity).
- `SHIFTED` — same handler, different opcode value.
- `ADDED` — v84-only (no v83 equivalent).
- `REMOVED` — present in v83, absent in v84.

Cross-reference the Atlas handler **name constants** (the strings the template will use) by matching each logical handler to its constant in `libs/atlas-packet/*/serverbound/` (e.g. `LoginHandle = "LoginHandle"`). Record that string in the logical-name column so Component C can wire it.

- [ ] **Step 4: Verify completeness against v83 template**

Run: `grep -o '"handler": "[^"]*"' services/atlas-configurations/seed-data/templates/template_gms_83_1.json | sort -u | wc -l`
Then confirm Section 1 of the delta doc accounts for every v83 handler opcode (each appears as SAME / SHIFTED / REMOVED) plus any ADDED. Note any v84 inbound opcode whose function is unnamed as **low-confidence** in the evidence column (OQ-7).

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): v84 inbound (handler) opcode map + v83 diff"
```

### Task A3: Harvest + diff outbound (writer) opcodes

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 2)

- [ ] **Step 1: Dump the v84 inbound (client-recv = Atlas writer) opcode → function map**

On the v84 IDB, walk the inbound dispatch from A1. Capture opcode → parsing function for every server→client packet the client handles.

- [ ] **Step 2: Name-anchor against v83 (v95 tie-breaker), same method as A2 Step 2.**

- [ ] **Step 3: Classify and fill Section 2** with the same `SAME/SHIFTED/ADDED/REMOVED` scheme. Match each logical writer to its Atlas writer **name** (the string in the template `writers[].writer`, registered in `produceWriters()` of atlas-login/atlas-channel — values equal the constant names, e.g. those in `libs/atlas-packet/*/clientbound/`).

- [ ] **Step 4: Verify completeness against v83 template**

Run: `grep -o '"writer": "[^"]*"' services/atlas-configurations/seed-data/templates/template_gms_83_1.json | sort -u | wc -l`
Confirm Section 2 accounts for every v83 writer opcode plus any ADDED. Flag unnamed v84 functions as low-confidence.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): v84 outbound (writer) opcode map + v83 diff"
```

### Task A4: Packet-structure delta for in-scope flows + usesPin

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Sections 3, 4)

- [ ] **Step 1: For each in-scope flow, diff packet structure v83 vs v84**

In-scope flows (Section 3.1, exhaustive): login handshake, auth, world/channel list, character list, character select / PIC-PIN, enter-channel, map load (spawn/field), movement, chat. For each packet, compare the v83 and v84 decompiled (de)serialization and document every field **added / removed / reordered / resized / conditional**, citing the IDA function/address. Where structure is identical, state "identical to v83" with the proving function (FR-1.3).

- [ ] **Step 2: Record spot-checks for out-of-scope flows (Section 3.2)**

For flows not in scope, record what was checked and what was assumed, so future work knows the confidence boundary (FR-1.2).

- [ ] **Step 3: Determine usesPin (Section 4, OQ-1)**

From the v84 login-flow analysis, determine whether v84 uses the PIN flow. Document the evidence (the login-sequence functions). Default expectation is `false` (every GMS template is `false`), but this must be confirmed, not assumed.

- [ ] **Step 4: Verify**

Run: `grep -A2 '### 3.1 In-scope flows' docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md && grep -A2 '## 4. usesPin' docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`
Expected: Section 3.1 has a documented entry per in-scope flow; Section 4 states a concrete `usesPin` value with evidence.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): v84 in-scope packet-structure delta + usesPin"
```

---

## Phase B — Version Helpers + Code Audit

> Depends on Phase A for the *evidence column* of the audit table, but B1 (helpers) and B2 (enumeration) can start immediately; the **corrections** (B3–B5) must cite A's findings.

### Task B1: Add `tenant.Model` version-predicate helpers (TDD)

**Files:**
- Modify: `libs/atlas-tenant/tenant.go`
- Test: `libs/atlas-tenant/tenant_test.go` (already exists; `package tenant`, internal — **append** to it)

> Construction API (verified): there is **no Builder**. Models are built with `tenant.Create(id uuid.UUID, region string, major, minor uint16) (Model, error)` — internally `package tenant` so the test calls `Create(...)` directly. The existing `tenant_test.go` already imports `github.com/google/uuid`. **The helpers use pointer receivers**, so a method call on a non-addressable function return (`mv(84).MajorAtLeast(...)`) will not compile — always assign to a local var first (`m := mv(84); m.MajorAtLeast(...)`).

- [ ] **Step 1: Append the failing tests to `libs/atlas-tenant/tenant_test.go`**

```go
func mv(major uint16) Model {
	t, _ := Create(uuid.New(), "GMS", major, 1)
	return t
}

func TestIsRegion(t *testing.T) {
	m := mv(84)
	if !m.IsRegion("GMS") {
		t.Fatalf("IsRegion(GMS) = false, want true")
	}
	if m.IsRegion("JMS") {
		t.Fatalf("IsRegion(JMS) = true, want false")
	}
}

func TestMajorAtLeast(t *testing.T) {
	cases := []struct {
		v, bound uint16
		want     bool
	}{{83, 84, false}, {84, 84, true}, {95, 84, true}, {84, 95, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorAtLeast(c.bound); got != c.want {
			t.Errorf("mv(%d).MajorAtLeast(%d) = %v, want %v", c.v, c.bound, got, c.want)
		}
	}
}

func TestMajorAtMost(t *testing.T) {
	cases := []struct {
		v, bound uint16
		want     bool
	}{{12, 12, true}, {28, 28, true}, {29, 28, false}, {84, 94, true}, {95, 94, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorAtMost(c.bound); got != c.want {
			t.Errorf("mv(%d).MajorAtMost(%d) = %v, want %v", c.v, c.bound, got, c.want)
		}
	}
}

func TestMajorInRange(t *testing.T) {
	// inclusive on both ends; encodes e.g. monster book GMS 28..87
	cases := []struct {
		v, lo, hi uint16
		want      bool
	}{{28, 28, 87, true}, {87, 28, 87, true}, {84, 28, 87, true}, {27, 28, 87, false}, {88, 28, 87, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorInRange(c.lo, c.hi); got != c.want {
			t.Errorf("mv(%d).MajorInRange(%d,%d) = %v, want %v", c.v, c.lo, c.hi, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail (compile error: undefined methods)**

Run: `cd libs/atlas-tenant && go test ./... -run 'TestIsRegion|TestMajorAtLeast|TestMajorAtMost|TestMajorInRange' -v`
Expected: FAIL — `m.IsRegion undefined`, etc.

- [ ] **Step 3: Implement the four helpers**

In `libs/atlas-tenant/tenant.go`, after the existing getters (after line 31), add:

```go
// IsRegion reports whether the tenant belongs to the given region.
func (m *Model) IsRegion(region string) bool {
	return m.region == region
}

// MajorAtLeast reports whether the tenant's major version is >= v.
func (m *Model) MajorAtLeast(v uint16) bool {
	return m.majorVersion >= v
}

// MajorAtMost reports whether the tenant's major version is <= v.
func (m *Model) MajorAtMost(v uint16) bool {
	return m.majorVersion <= v
}

// MajorInRange reports whether the tenant's major version is in [lo, hi] (inclusive both ends).
func (m *Model) MajorInRange(lo, hi uint16) bool {
	return m.majorVersion >= lo && m.majorVersion <= hi
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-tenant && go test -race ./... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-tenant/tenant.go libs/atlas-tenant/tenant_test.go
git commit -m "feat(atlas-tenant): add version-predicate helpers (IsRegion/MajorAtLeast/MajorAtMost/MajorInRange)"
```

### Task B2: Enumerate every version-gated branch → audit table

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 5)

- [ ] **Step 1: Enumerate all sites**

Run and capture full output:

```bash
grep -rn 'Region()\|MajorVersion()\|MinorVersion()' services/ libs/ --include='*.go' \
  | grep -v '_test.go' \
  | grep -E '==|!=|>=|<=|>|<'
```

- [ ] **Step 2: Fill Section 5 — one row per site**

For each site write: `file:line | predicate | v83 result | v84 result | correct for v84? | action | delta evidence`. Compute the v84 result by substituting `majorVersion=84` into the predicate. Mark **action** as one of:
- `unchanged (correct)` — v84 already evaluates correctly; leave the code as-is (§5.2 of design: do NOT migrate correct out-of-scope sites).
- `migrate+correct` — v84 evaluates wrong, OR the site is on an in-scope flow we verify against the delta. These become Tasks B3–B5.

Cross-reference each `migrate+correct` row's evidence column to a Section 1/2/3 finding (or "no packet/behavior difference observed").

- [ ] **Step 3: Verify every site is classified**

Run: count the grep hits from Step 1 and confirm Section 5 has the same number of rows.
Expected: row count == grep hit count (no site unclassified).

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): version-branch audit table (all sites classified)"
```

### Task B3: Correct + migrate the auto-AP boundary (atlas-character)

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:1336`
- Test: `services/atlas-character/atlas.com/character/character/processor_test.go` (or the nearest existing test file for this processor — confirm with `ls services/atlas-character/atlas.com/character/character/*_test.go`)

> This is the one unambiguous bug: `Region()=="GMS" && MajorVersion()==83` silently excludes v84 (inline TODO admits the range is undefined). Widen to the range confirmed by the Phase A behavior evidence (OQ-3). Expected resolution: auto-AP is a pre-Big-Bang behavior → `IsRegion("GMS") && MajorAtMost(94)`. **Use the range the delta doc actually supports; if Phase A proves a different bound, use that and note it.** Must not change the v83 result (true).

- [ ] **Step 1: Read the surrounding context**

Run: `sed -n '1320,1360p' services/atlas-character/atlas.com/character/character/processor.go`
Identify the function and what `p.t` is (the `tenant.Model`), and what the branch guards (Beginner/Noblesse/Legend auto-AP assignment).

- [ ] **Step 2: Write a failing behavior-preservation test**

Add a test that exercises the auto-AP decision through the smallest accessible seam. If the branch is reachable only via a large method, extract the predicate into a tiny pure helper first (e.g. `func appliesAutoAP(t tenant.Model) bool { return t.IsRegion("GMS") && t.MajorAtMost(94) }`) and test that:

```go
func TestAppliesAutoAP(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		{"GMS", 83, true},  // v83 unchanged
		{"GMS", 84, true},  // v84 now included (was the bug)
		{"GMS", 94, true},  // pre-Big-Bang upper edge
		{"GMS", 95, false}, // post-Big-Bang excluded
		{"JMS", 83, false}, // region-gated
	}
	for _, c := range cases {
		tm, _ := tenant.Create(uuid.New(), c.region, c.major, 1)
		if got := appliesAutoAP(tm); got != c.want {
			t.Errorf("appliesAutoAP(%s,%d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}
```

> Construction uses `tenant.Create(uuid.New(), region, major, minor) (Model, error)` (verified — there is no Builder). Add imports `"github.com/google/uuid"` and the tenant lib (`tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` — confirm the exact import path from a neighboring file in this package). `tm` is a local var (addressable), so the pointer-receiver helpers work. The five assertions are the contract.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestAppliesAutoAP -v`
Expected: FAIL — `appliesAutoAP` undefined (and/or the `==83` form would fail the v84 case if tested inline).

- [ ] **Step 4: Implement — extract helper + migrate the call site**

Replace the line `if p.t.Region() == "GMS" && p.t.MajorVersion() == 83 {` with `if appliesAutoAP(p.t) {` and add the helper near the top of the file:

```go
// appliesAutoAP reports whether Beginner/Noblesse/Legend auto-AP assignment applies.
// Pre-Big-Bang GMS behavior (28..94 era); v84 included. Evidence: v84-packet-delta.md §5.
func appliesAutoAP(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtMost(94)
}
```

Remove the now-resolved inline TODO comment about the undefined range.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd services/atlas-character/atlas.com/character && go test -race ./character/ -run TestAppliesAutoAP -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/character/processor_test.go
git commit -m "fix(atlas-character): include v84 in auto-AP boundary (==83 -> GMS MajorAtMost(94))"
```

### Task B4: Correct + migrate the default-gender boundary (atlas-account)

**Files:**
- Modify: `services/atlas-account/atlas.com/account/account/processor.go:165`
- Test: nearest existing processor test (confirm with `ls services/atlas-account/atlas.com/account/account/*_test.go`)

> Current `Region()=="GMS" && MajorVersion()>83` already fires for v84 (gender=10, UI-choose). The migration is behavior-identical (`MajorAtLeast(84)`), confirmed by the delta. v83 stays Male.

- [ ] **Step 1: Read context**

Run: `sed -n '155,180p' services/atlas-account/atlas.com/account/account/processor.go`
Identify the default-gender assignment (`10` = UI-choose) and the v83 fallthrough (Male).

- [ ] **Step 2: Write a failing behavior-preservation test**

Extract the predicate to a helper and test:

```go
func TestUsesChooseGender(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		{"GMS", 83, false}, // v83 -> Male (unchanged)
		{"GMS", 84, true},  // v84 -> UI-choose (unchanged from >83)
		{"GMS", 95, true},
		{"JMS", 84, false},
	}
	for _, c := range cases {
		tm, _ := tenant.Create(uuid.New(), c.region, c.major, 1)
		if got := usesChooseGender(tm); got != c.want {
			t.Errorf("usesChooseGender(%s,%d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}
```

> Same construction note as B3: `tenant.Create(uuid.New(), region, major, minor)`; import `github.com/Chronicle20/atlas/libs/atlas-tenant` (package `tenant`) and `github.com/google/uuid`.

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-account/atlas.com/account && go test ./account/ -run TestUsesChooseGender -v`
Expected: FAIL — `usesChooseGender` undefined.

- [ ] **Step 4: Implement**

Add the helper and replace the call site:

```go
// usesChooseGender reports whether default gender is UI-choose (10) rather than Male.
// GMS post-v83. Evidence: v84-packet-delta.md §5.
func usesChooseGender(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(84)
}
```

Replace `if p.t.Region() == "GMS" && p.t.MajorVersion() > 83 {` with `if usesChooseGender(p.t) {`.

- [ ] **Step 5: Run to verify it passes**

Run: `cd services/atlas-account/atlas.com/account && go test -race ./account/ -run TestUsesChooseGender -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-account/atlas.com/account/account/processor.go services/atlas-account/atlas.com/account/account/processor_test.go
git commit -m "refactor(atlas-account): migrate default-gender boundary to MajorAtLeast(84) (behavior-identical)"
```

### Task B5: Migrate remaining audit-flagged in-scope predicates

**Files:**
- Modify: only sites Task B2 marked `migrate+correct` that lie on in-scope flows. Known candidates (confirm each against the B2 table and Phase A evidence before touching):
  - `libs/atlas-packet/character/data.go` monster-book (`>28 && <=87`) → `MajorInRange(29,87)` if delta confirms v84 encodes the monster book.
  - `libs/atlas-packet/buddy/clientbound/invite.go` (`<=87`) → `MajorAtMost(87)` if delta confirms job-level present for v84.
  - `>=95` family (`character_cash_item_use.go`, `character_attack_common.go:180`, `model/damage_taken_info.go:66`, `libs/atlas-packet/chat/serverbound/whisper.go:60,75`, spawn) → `MajorAtLeast(95)` (do NOT fire for v84 — confirm v84 wants the pre-95 path).
  - `<=28` family (`login/main.go:277`, `channel/main.go:378`) → `MajorAtMost(28)`; `<=12` family (`login/session/model.go:35`, `channel/session/model.go:40`) → `MajorAtMost(12)`.

> Each of these is a **behavior-preserving** rename: the new helper must evaluate identically to the inequality it replaces for ALL of {12,28,83,84,87,94,95}. Do not change v83 behavior. A site B2 marked `unchanged (correct)` and **not** on an in-scope flow is left as-is.

- [ ] **Step 1: For each site, read context and confirm the equivalent helper**

For site `S` with predicate `P`, confirm the replacement helper `H` satisfies `P(v) == H(v)` for v ∈ {12,28,83,84,87,94,95}. Example for monster-book `t.MajorVersion() > 28 && t.MajorVersion() <= 87`: equivalent to `t.MajorInRange(29,87)` — note `>28` is `>=29`, so use `MajorInRange(29,87)`, NOT `MajorInRange(28,87)`. **Verify the exact inclusive/exclusive boundary every time.**

- [ ] **Step 2: Write a failing boundary table-test per migrated site**

For each migrated predicate, add a table test asserting the helper equals the original inequality across {12,28,29,83,84,87,94,95}. Pattern (adapt names/bounds per site):

```go
func TestMonsterBookPredicate(t *testing.T) {
	for _, v := range []uint16{12, 28, 29, 83, 84, 87, 94, 95} {
		old := v > 28 && v <= 87
		tm, _ := tenant.Create(uuid.New(), "GMS", v, 1)
		neu := tm.MajorInRange(29, 87)
		if old != neu {
			t.Errorf("v=%d: old=%v new=%v", v, old, neu)
		}
	}
}
```

> Same construction note as B3/B4. `tm` is a local var so the pointer-receiver helper resolves.

- [ ] **Step 3: Run to verify the test fails (or fails to compile until the migration is applied)**

Run the package test for the touched module. Expected: FAIL until Step 4.

- [ ] **Step 4: Apply the migration at the call site** (swap the raw inequality for the verified helper call). Keep `Region()` as a separate `IsRegion(...)` conjunct where present. Add an inline comment naming the capability the range encodes (e.g. `// monster book: GMS 29..87`).

- [ ] **Step 5: Run to verify it passes**

Run: `go test -race ./...` in each touched module.
Expected: PASS.

- [ ] **Step 6: Commit (one commit per module touched)**

```bash
git add <changed files in this module>
git commit -m "refactor(<module>): migrate in-scope version predicate to tenant helper (behavior-identical, v84 audited)"
```

- [ ] **Step 7: Update the B2 audit table**

Mark each migrated row's action as done and ensure its evidence column cites the delta finding.

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): mark migrated predicates resolved in audit table"
```

---

## Phase C — Socket Configuration Template

> Depends on Phase A Sections 1, 2, 4 (opcode maps + usesPin). Independent of Phase B/D.

### Task C1: Author `template_gms_84_1.json`

**Files:**
- Create: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Reference: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`

- [ ] **Step 1: Copy v83 as the base**

```bash
cp services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
   services/atlas-configurations/seed-data/templates/template_gms_84_1.json
```

- [ ] **Step 2: Set the version header + usesPin**

In the new file set `"majorVersion": 84`, keep `"minorVersion": 1`, keep `"region": "GMS"`, and set `"usesPin"` to the value determined in delta-doc Section 4 (OQ-1; default `false` unless the delta proved otherwise).

- [ ] **Step 3: Apply the socket opcode delta (delta-doc Sections 1 & 2)**

For each row:
- `SAME` → leave the entry unchanged.
- `SHIFTED` → change only the `"opCode"` value; keep the `validator`/`handler`/`writer` name.
- `ADDED` → add a new entry referencing an **existing** registered symbol name (handler/validator/writer). If the delta says the symbol does not yet exist in atlas-channel/atlas-login, **STOP and escalate** — that is a Go change, not config (do not invent a name).
- `REMOVED` → delete the entry.

- [ ] **Step 4: Apply the message-type table delta (FR-2.2)**

Locate the message-type / non-socket version-keyed sections:

```bash
python3 -c "import json,sys; d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_83_1.json')); print(sorted(d.keys())); print(sorted(d.get('socket',{}).keys()))"
```

If a `messageType` (or equivalent enum) table exists and delta-doc Section 3 found a v84 difference, encode the v84 values. **Do not hardcode enum bytes anywhere in Go** — inbound handlers reverse-resolve this table (`bug_npc_msgtype_hardcoded_vs_config`). If no difference, copy v83 as-is.

- [ ] **Step 5: Copy non-socket sections unchanged**

`characters` (templates/presets), `npcs`, `worlds`, `cashShop` are copied from v83 (parity sufficient for basic playthrough; `worlds` is operator-tunable). Confirm they are present and identical to v83 unless the delta required a change.

- [ ] **Step 6: Verify the file is valid JSON with the right version key**

Run:
```bash
python3 -c "import json; d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_84_1.json')); assert d['region']=='GMS' and d['majorVersion']==84 and d['minorVersion']==1, d; print('ok', len(d['socket']['handlers']),'handlers', len(d['socket']['writers']),'writers')"
```
Expected: prints `ok <N> handlers <M> writers` with no assertion error.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_84_1.json
git commit -m "feat(atlas-configurations): add GMS v84.1 socket-config seed template"
```

### Task C2: Symbol-resolution gate + seeder verification (FR-2.4, FR-2.3)

**Files:**
- Create: `tools/template-symbol-check.sh`
- Test: `services/atlas-configurations/atlas.com/configurations/seeder/seeder_test.go` (extend if present; confirm with `ls services/atlas-configurations/atlas.com/configurations/seeder/*_test.go`)

> The runtime resolves a template's `handler`/`validator`/`writer` **name strings** to Go funcs via `produceHandlers()`/`produceValidators()`/`produceWriters()` in `services/atlas-login/.../main.go` and `services/atlas-channel/.../main.go`. The name-string VALUES equal the constant names (e.g. `const LoginHandle = "LoginHandle"` in `libs/atlas-packet/login/serverbound/request.go`). A dangling template name is one that exists as a registered string **nowhere** in the login/channel/opcode source. The gate catches exactly that.

- [ ] **Step 1: Write the gate script**

Create `tools/template-symbol-check.sh`:

```bash
#!/usr/bin/env bash
# FR-2.4 gate: every handler/validator/writer name in a socket-config template
# must appear as a registered string literal in atlas-login / atlas-channel / libs/atlas-packet.
# Usage: tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_84_1.json
set -euo pipefail
TEMPLATE="${1:?usage: template-symbol-check.sh <template.json>}"
ROOT="$(git rev-parse --show-toplevel)"
SEARCH_PATHS=("$ROOT/services/atlas-login" "$ROOT/services/atlas-channel" "$ROOT/libs/atlas-packet")

names() { # extract distinct values for a JSON key
  python3 -c "import json,sys; d=json.load(open('$TEMPLATE')); s=d.get('socket',{}); \
print('\n'.join(sorted({h.get('$1','') for h in s.get('$2',[]) if h.get('$1')})))"
}

missing=0
check() {
  local name="$1"
  [ -z "$name" ] && return 0
  if ! grep -rqF "\"$name\"" "${SEARCH_PATHS[@]}" --include='*.go'; then
    echo "DANGLING: $name (no registered string literal found)"
    missing=1
  fi
}

while IFS= read -r n; do check "$n"; done < <(names validator handlers)
while IFS= read -r n; do check "$n"; done < <(names handler  handlers)
while IFS= read -r n; do check "$n"; done < <(names writer   writers)

if [ "$missing" -ne 0 ]; then
  echo "FAIL: template has dangling symbol references"
  exit 1
fi
echo "OK: all template symbols resolve"
```

Make it executable: `chmod +x tools/template-symbol-check.sh`.

- [ ] **Step 2: Run the gate against the v83 template first (sanity — must pass)**

Run: `tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
Expected: `OK: all template symbols resolve`. (If v83 reports DANGLING, the gate is too strict — fix the matcher before trusting it on v84.)

- [ ] **Step 3: Run the gate against the v84 template**

Run: `tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
Expected: `OK: all template symbols resolve`. Any `DANGLING:` line means a name in the template has no registered symbol → fix the template (or escalate the missing-symbol Go change per C1 Step 3).

- [ ] **Step 4: Add a seeder idempotency + distinct-version test (FR-2.3)**

Confirm/extend the seeder test so that (a) the v84 template parses into the seeder's template model and (b) `(GMS,84,1)` is treated as distinct from `(GMS,83,1)` and an already-seeded `(GMS,84,1)` is skipped on re-run. If the seeder already has a table test over `templates/`, assert it now discovers the 84_1 file. Read the existing test first:

Run: `sed -n '1,80p' services/atlas-configurations/atlas.com/configurations/seeder/seeder_test.go` (if it exists) and extend its fixture list / assertions accordingly. If no seeder test exists, add a minimal one that loads `template_gms_84_1.json` via the same `ConfigMetadata` decode the seeder uses and asserts `Region=="GMS", Major==84, Minor==1`.

- [ ] **Step 5: Run the seeder test**

Run: `cd services/atlas-configurations/atlas.com/configurations && go test -race ./seeder/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/template-symbol-check.sh services/atlas-configurations/atlas.com/configurations/seeder/seeder_test.go
git commit -m "test(task-083): template symbol-resolution gate + v84 seeder idempotency check"
```

---

## Phase D — WZ Game Data (operational)

> Independent of Phase B/C. No atlas-data code change expected. Output is data in MinIO + a verification record appended to the delta doc.

### Task D1: Ingest v84 WZ data

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 6, runbook — append ingest steps)

- [ ] **Step 1: Upload v84 WZ archives to the 84.1 MinIO path**

Place each v84 archive at `<scope>/regions/GMS/versions/84.1/<archive>` in the WZ bucket (`BucketWZ`). This is additive — no 83.1 key is touched. (Confirm the bucket/scope env the cluster uses; cf. `reference_atlas_data_wz_inspection` for the MinIO `atlas-wz` bucket layout.)

- [ ] **Step 2: Trigger the ingest Job for (GMS, 84, 1)**

Use the REST `JobCreator` path (renders the `atlas-data-ingest-job-template` ConfigMap with `MODE=ingest SCOPE=... REGION=GMS MAJOR_VERSION=84 MINOR_VERSION=1`). Record the exact trigger (endpoint or `kubectl` apply) in delta-doc Section 6.

- [ ] **Step 3: Verify the ingest Job completed**

Use `mcp__kubernetes__pods_list_in_namespace` / `pods_log` to confirm the ingest pod ran each registered worker without error. Record the Job name and outcome in Section 6.

- [ ] **Step 4: Commit the runbook update**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): record v84 WZ ingest steps + outcome"
```

### Task D2: Verify atlas-data serves 84.1 + clear spawn cache

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 6 — verification record)

- [ ] **Step 1: Fetch a representative 84.1 asset set**

With an 84.1 tenant header (`TENANT_ID`, `REGION=GMS`, `MAJOR_VERSION=84`, `MINOR_VERSION=1`), `GET` from atlas-data a representative **map, item, mob, and reactor** (per `reference_atlas_data_wz_inspection`, e.g. via a throwaway curl pod). Confirm each resolves from the 84.1 path.

- [ ] **Step 2: Confirm 83.1 still resolves (isolation, FR-4.2)**

Repeat one fetch with an 83.1 header; confirm it still returns 83.1 data unchanged.

- [ ] **Step 3: Clear the spawn cache (FR-4.3)**

Per `reference_atlas_maps_spawn_cache`: `DEL atlas:maps:spawn:*` and DELETE the affected map's monsters in atlas-monsters so v84 spawn data is observed and not masked by stale 83-era cache. Record the exact commands in Section 6.

- [ ] **Step 4: Record OQ-4 findings**

If any representative asset fails to parse (a reader gap like prior `consumeOnPickup` / snap-to-ground cases), document it as a finding. Small fixes land here (note them as a follow-up task in this plan); large structural reader work is escalated, not silently absorbed.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): verify atlas-data serves v84 84.1 assets; record spawn-cache clear"
```

---

## Phase E — Provisioning Runbook + Live E2E (blocking gate)

> Requires A + B + C + D all landed. **Per the approved decision, the task is not "done" until the live playthrough passes** (design §1, §7.2).

### Task E1: Write the provisioning runbook

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 6)

- [ ] **Step 1: Document the end-to-end provisioning sequence (FR-5.1)**

Write the ordered, repeatable steps:
1. Deploy + seed `template_gms_84_1.json` (seeder idempotent; skips if `(GMS,84,1)` already present).
2. Upload v84 WZ to `regions/GMS/versions/84.1/`; run the ingest Job; clear spawn cache (Tasks D1/D2).
3. Create the tenant `(region=GMS, major=84, minor=1)` via the existing atlas-tenants `CreateTenantHandler` (no schema change). Record the exact request body.
4. **Restart sequence (OQ-6):** determine and document whether channel/login pods must restart for the freshly seeded v84 tenant's handler/writer bindings to load (`bug_new_opcodes_not_in_live_tenant_config` — bindings are not hot-reloaded). Capture the exact sequence that works.

- [ ] **Step 2: Verify the runbook is self-contained**

Re-read Section 6: a fresh operator should be able to follow it without external context. Confirm each step has a concrete command or API call.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): v84 tenant provisioning runbook + restart sequence"
```

### Task E2: Live v84 playthrough (FR-5.2)

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 6 — E2E result)

- [ ] **Step 1: Provision the v84 tenant per the E1 runbook on the running stack.**

- [ ] **Step 2: Drive a real v84 client through the full flow**

connect to login → authenticate → world/channel select → character list → enter channel → load starting map → move + chat.

- [ ] **Step 3: Diagnose failures live (do not guess)**

Watch channel/login logs via `mcp__kubernetes__pods_log` / Grafana Loki (`reference_observability`). The canonical failure signature is `unhandled message op 0xXX` at info → a missing/wrong opcode in the template. Fix in the delta → template loop (re-run Phase C symbol gate, re-seed, restart per E1), never by hardcoding. Low-confidence (unnamed-IDB) opcodes from Phase A are the first suspects.

- [ ] **Step 4: Record the result**

Document the passing playthrough (each step reached) in Section 6. If a step fails and requires a code/template change, loop back to the owning phase, then re-run.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): record passing v84 live playthrough"
```

### Task E3: v83 regression check (FR-5.3)

**Files:**
- Modify: `docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md` (Section 6 — regression result)

- [ ] **Step 1: Re-run login → channel → map on the existing v83 tenant** after all changes are deployed. Confirm v83 behavior is unchanged (the version-helper migrations are behavior-preserving; this is the live confirmation).

- [ ] **Step 2: Record the v83 regression result in Section 6.**

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md
git commit -m "docs(task-083): record v83 regression pass"
```

---

## Phase F — Build & Verification Gates (mandatory before "done")

> Per CLAUDE.md "Build & Verification". Run after all code phases (B, C) land; before claiming the branch ready.

### Task F1: Full verification sweep

- [ ] **Step 1: Identify changed Go modules**

Run: `git diff --name-only main... | grep '\.go$' | sed 's#/atlas\.com/.*##; s#^libs/[^/]*#&#' | sort -u`
Expected changed modules include at least: `libs/atlas-tenant`, `services/atlas-character`, `services/atlas-account`, plus any module touched by Task B5 (e.g. `libs/atlas-packet`).

- [ ] **Step 2: `go test -race ./...` clean in every changed module**

Run per module, e.g.:
```bash
( cd libs/atlas-tenant && go test -race ./... )
( cd services/atlas-character/atlas.com/character && go test -race ./... )
( cd services/atlas-account/atlas.com/account && go test -race ./... )
```
Expected: all PASS.

- [ ] **Step 3: `go vet ./...` clean in every changed module** (same set as Step 2). Expected: no output.

- [ ] **Step 4: `go build ./...` clean in every changed service.** Expected: no errors.

- [ ] **Step 5: `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched**

`libs/atlas-tenant` is consumed by many services — if its `go.mod` consumers changed, bake every affected service. At minimum bake the services with touched `go.mod`:
```bash
docker buildx bake atlas-character
docker buildx bake atlas-account
# + any other service whose go.mod changed (and atlas-login/atlas-channel if touched)
```
Expected: all bake targets succeed (catches missing `COPY libs/...` in the shared Dockerfile that `go build` cannot).

- [ ] **Step 6: `tools/redis-key-guard.sh` clean**

Run: `GOWORK=off tools/redis-key-guard.sh` from the worktree root (`reference_rediskeyguard_invariant`).
Expected: clean (no banned raw keyed go-redis calls).

- [ ] **Step 7: Run the template symbol gate one final time**

Run: `tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
Expected: `OK: all template symbols resolve`.

- [ ] **Step 8: Commit any final fixes**

```bash
git add -A
git commit -m "chore(task-083): verification sweep fixes"
```

---

## Acceptance-criteria coverage map

| PRD acceptance criterion | Task(s) |
|---|---|
| `v84-packet-delta.md` exists w/ opcode maps, structure delta, audit table | A0–A4, B2 |
| `template_gms_84_1.json` exists, v84-correct, symbols resolve | C1, C2 |
| Seeding idempotent, v83 untouched | C2 (seeder test), seeder idempotent by construction |
| v84 WZ ingested at 84.1, representative asset set served, 83.1 unaffected | D1, D2 |
| Every `MajorVersion()` branch audited; 83/84 edges corrected w/ evidence; v83 unchanged | B2, B3, B4, B5 |
| v84.1 tenant provisioned via documented repeatable steps | E1 |
| Real v84 client completes login→channel→map→move+chat | E2 |
| v83 regression passes | E3 |
| go test -race / vet / build / bake / redis-key-guard clean | F1 |
