# IDA Export Re-Harvest Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a real automated `packet-audit` exporter (decompiler-parser over MCP-HTTP with struct-helper descent and an honest `unresolved`-over-guess invariant), re-export all four client baselines, re-audit, and triage + fix every genuine wire divergence — turning the four-version packet audit from "trusted but input-limited" into genuinely verified.

**Architecture:** A standalone Go binary path (`packet-audit export`) drives IDA-MCP over HTTP, parses Hex-Rays decompile text into ordered `Decode*` reads, emits `Delegate`/`Ref` entries for packet-reading helpers (descent), and writes deterministic per-version JSON. The existing audit consumer (`resolveWithVisited` splicing, `candidatesFromFName` mapping, diff/report) is reused unchanged except for an additive `Unresolved` marker. Discovery phases (re-export, verdict-delta triage, wire fixes, opaque decomposition, template completeness, ledger) follow with hard, evidence-backed gates.

**Tech Stack:** Go 1.24 (`tools/packet-audit` module), `net/http` MCP-JSON-RPC client, table-driven Go tests against checked-in Hex-Rays fixtures, `libs/atlas-packet` byte-level wire tests (`pt.RoundTrip`/`pt.Variants`), JSON:API-agnostic doc artifacts under `docs/packets/`.

**Read first:** `context.md` in this folder — it locks the design-ambiguity resolutions (Delegate-emission over inline, `Harvest` as the descent driver, the `unresolved` marker) and the verified file:line references this plan depends on.

---

## Phase 0 — Rebase + verdict snapshot (HARD PREREQUISITE)

> No exporter code is written until this phase's gate passes. The corrected
> task-080 baseline does not exist in this worktree (it forks pre-080 `main`).

### Task 0.1: Rebase onto the task-080 baseline

**Files:** none (git history only)

- [ ] **Step 1: Confirm the current base is pre-080**

Run (from this task worktree root, `.worktrees/task-081-ida-export-reharvest/`):
```bash
git merge-base HEAD main | xargs git log --oneline -1     # expect 40af0c80f (pre-080)
ls docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md 2>&1 # expect: No such file (proves stale base)
git log --oneline main | grep -i "packet-audit-closeout"  # empty ⇒ 080 not on main yet
```

- [ ] **Step 2: Choose the rebase target**

If PR #678 (task-080) has merged to `main`, target `main`. Otherwise target the
branch `task-080-packet-audit-closeout`. Verify which:
```bash
git log --oneline main | grep -i "task-080\|packet-audit-closeout" | head
git log --oneline task-080-packet-audit-closeout | head -1
```
Pick `main` if the grep is non-empty, else `task-080-packet-audit-closeout`.

- [ ] **Step 3: Rebase the two task-081 doc commits onto the chosen target**

Run (substitute `<target>`):
```bash
git rebase --onto <target> $(git merge-base HEAD main) task-081-ida-export-reharvest
```
Resolve any conflicts (expected only in `docs/tasks/task-081-*` — keep ours). The
`docs/packets/**` files come wholesale from `<target>`.

- [ ] **Step 4: Verify the corrected baseline is now present**

Run:
```bash
ls docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md     # now EXISTS
ls docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md
git log --oneline -3                                       # task-081 commits sit atop 080
git branch --show-current                                  # task-081-ida-export-reharvest
```
Expected: `STARTING_A_NEW_VERSION_PASS.md` exists; all four SUMMARYs present; branch unchanged.

- [ ] **Step 5: Commit nothing (rebase already rewrote history); note in run log**

No commit. If a conflict resolution touched a task-081 doc, that is folded into the
existing doc commit by the rebase.

### Task 0.2: Snapshot task-080 per-packet verdicts (FR-7.1)

**Files:**
- Create: `docs/tasks/task-081-ida-export-reharvest/verdict-snapshot-080.md`

- [ ] **Step 1: Regenerate the task-080 audit to confirm the base reproduces**

Run from worktree root (paths per memory `reference_packet_audit_tool_mechanics`: `--output` is the PARENT):
```bash
go run ./tools/packet-audit \
  --csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --ida-source docs/packets/ida-exports/gms_v95.json \
  --output docs/packets/audits
git status --short docs/packets/audits/gms_v95/   # expect: no changes (base reproduces)
```
Expected: regenerating over the checked-in export produces no diff. If it does, the
base is not intact — STOP and report BLOCKED.

- [ ] **Step 2: Extract per-packet verdicts for all four versions into the snapshot**

For each version's `SUMMARY.md`, capture every packet's verdict line. Build the
snapshot file with one table per version: `| Packet | Verdict |` rows, copied from
the four `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md`.
Use a deterministic extraction so the post-re-export delta is mechanical:
```bash
for v in gms_v83 gms_v87 gms_v95 jms_v185; do
  echo "## $v"
  grep -E '^\| ' "docs/packets/audits/$v/SUMMARY.md" | grep -E '✅|⚠️|❌|🔍'
done
```
Paste the output into `verdict-snapshot-080.md` under a heading per version, with a
preamble: "Per-packet verdict baseline captured from task-080 (commit `<sha>`) for
the §4.7 verdict-delta triage gate. DO NOT EDIT after Phase 0."

- [ ] **Step 3: Commit the snapshot**

```bash
git add docs/tasks/task-081-ida-export-reharvest/verdict-snapshot-080.md
git commit -m "task-081(P0): snapshot task-080 per-packet verdicts for delta triage"
```

**Phase 0 GATE:** rebase clean; `STARTING_A_NEW_VERSION_PASS.md` + four SUMMARYs
present; audit re-run reproduces the task-080 SUMMARYs byte-identical; snapshot
committed. Do not proceed until all four hold.

---

## Phase 1 — Build the exporter (strict TDD, CI needs no IDA)

> All Phase 1 work is in the `tools/packet-audit` module. Every task is
> test-first. Fixtures are synthetic-but-realistic Hex-Rays text committed under
> `internal/idasrc/testdata/`, so CI never touches a live IDB.

### Task 1.1: Add the `Unresolved` primitive + schema field

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/idasrc.go:12-21`
- Modify: `tools/packet-audit/internal/idasrc/export.go:10-36` (add field), `:87-137` (handle op)
- Test: `tools/packet-audit/internal/idasrc/export_test.go`

- [ ] **Step 1: Write the failing test**

Add to `export_test.go`:
```go
func TestResolveUnresolvedCall(t *testing.T) {
	// An export entry with an Unresolved op resolves to a single Unresolved
	// FieldCall (a known gap), NOT an error.
	src := mustLoadExport(t, "testdata/unresolved_mini.json")
	got, err := src.Resolve(context.Background(), "Foo::OnBar")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(got.Calls) != 2 {
		t.Fatalf("want 2 calls, got %d", len(got.Calls))
	}
	if got.Calls[1].Op != idasrc.Unresolved {
		t.Errorf("call[1].Op = %v, want Unresolved", got.Calls[1].Op)
	}
}
```
Create `tools/packet-audit/internal/idasrc/testdata/unresolved_mini.json`:
```json
{
 "binary": "x", "md5": "x", "generated_at": "2026-01-01T00:00:00Z",
 "functions": {
  "Foo::OnBar": {
   "address": "0x1", "direction": "clientbound",
   "calls": [
    {"op": "Decode4", "comment": "id"},
    {"op": "Unresolved", "comment": "indirect dispatch via vtable; hand-trace"}
   ]
  }
 }
}
```
(If `mustLoadExport` does not exist, use the loader already used in `export_test.go`; match its existing helper name.)

- [ ] **Step 2: Run it to verify it fails**

Run: `cd tools/packet-audit && go test ./internal/idasrc/ -run TestResolveUnresolvedCall -v`
Expected: FAIL — `parsePrim` rejects `"Unresolved"` (unknown op) or `Unresolved` const undefined.

- [ ] **Step 3: Add the primitive**

In `idasrc.go`, append to the `Primitive` const block (append last to preserve existing ordinals):
```go
const (
	Decode1 Primitive = iota // ReadByte/WriteByte
	Decode2                  // ReadShort/WriteShort
	Decode4                  // ReadInt/WriteInt
	Decode8                  // ReadLong/WriteLong
	DecodeStr                // ReadAsciiString/WriteAsciiString
	DecodeBuf                // ReadBytes/WriteBytes
	Unresolved               // parser could not prove this element; audit treats as a known gap
)
```

- [ ] **Step 4: Handle the op in resolve + add the schema field**

In `export.go`, add to `exportFn`:
```go
	// Unresolved marks a function the parser could not faithfully trace.
	// The audit treats it as a known gap, never a false verdict.
	Unresolved bool `json:"unresolved,omitempty"`
```
In `parsePrim` (the op-string switch in `export.go`), add:
```go
	case "Unresolved":
		return Unresolved, nil
```
Confirm `resolveWithVisited` falls through to the generic `parsePrim` branch for a
non-Delegate op — it already does (`export.go:130-134`), so `Unresolved` flows
into a `FieldCall{Op: Unresolved, ...}` automatically.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestResolveUnresolvedCall -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/idasrc.go tools/packet-audit/internal/idasrc/export.go \
        tools/packet-audit/internal/idasrc/export_test.go tools/packet-audit/internal/idasrc/testdata/unresolved_mini.json
git commit -m "task-081(P1): add Unresolved primitive + export schema marker"
```

### Task 1.2: `ParseDecompile` — linear read extraction

**Files:**
- Create: `tools/packet-audit/internal/idasrc/parse.go`
- Create: `tools/packet-audit/internal/idasrc/parse_test.go`
- Create: `tools/packet-audit/internal/idasrc/testdata/linear.c`
- Modify: `tools/packet-audit/internal/idasrc/mcp.go:49-54` (remove the old stub — `ParseDecompile` now lives in `parse.go`)

- [ ] **Step 1: Write the fixture**

`testdata/linear.c` (synthetic Hex-Rays for a simple clientbound packet):
```c
int __thiscall CLogin::OnFoo(CLogin *this, CInPacket *a2)
{
  unsigned __int8 result = CInPacket::Decode1(a2);   // resultCode
  int accountId = CInPacket::Decode4(a2);            // accountId
  CInPacket::DecodeStr(a2, &name);                    // name
  return result;
}
```

- [ ] **Step 2: Write the failing test**

`parse_test.go`:
```go
package idasrc

import (
	"os"
	"testing"
)

func mustFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func TestParseLinearReads(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "linear.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	want := []struct {
		op      string
		comment string
	}{
		{"Decode1", "resultCode"},
		{"Decode4", "accountId"},
		{"DecodeStr", "name"},
	}
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i].Op != w.op {
			t.Errorf("call[%d].Op = %q, want %q", i, calls[i].Op, w.op)
		}
		if calls[i].Comment != w.comment {
			t.Errorf("call[%d].Comment = %q, want %q", i, calls[i].Comment, w.comment)
		}
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseLinearReads -v`
Expected: FAIL — `ParseDecompile` returns the not-implemented error.

- [ ] **Step 4: Implement the linear scanner**

Create `parse.go`. `ParseDecompile` returns `[]rawCall`. The scanner:
1. Identifies the packet variable: the first arg passed to any `CInPacket::Decode*`
   / `COutPacket::Encode*` call (capture group from the call regex).
2. Walks lines in source order, matching `CInPacket::Decode{1,2,4,8}`, `DecodeStr`,
   `DecodeBuffer`, and the `COutPacket::Encode*` duals, emitting a `rawCall` per
   match with `Op` mapped to the canonical name and `Comment` taken from a trailing
   `// comment` if present (best-effort; correctness never gates on the label — PRD §9).
```go
package idasrc

import (
	"regexp"
	"strings"
)

var (
	reDecode = regexp.MustCompile(`C(?:In|Out)Packet::(Decode|Encode)(1|2|4|8|Str|Buffer)\s*\(`)
	reComment = regexp.MustCompile(`//\s*(.+?)\s*$`)
)

// opName maps the raw "<verb><width>" capture to the canonical export op string.
func opName(width string) string {
	switch width {
	case "1":
		return "Decode1"
	case "2":
		return "Decode2"
	case "4":
		return "Decode4"
	case "8":
		return "Decode8"
	case "Str":
		return "DecodeStr"
	case "Buffer":
		return "DecodeBuf"
	}
	return ""
}

// ParseDecompile extracts the ordered packet read/write primitives from one
// function's Hex-Rays decompile text. It is pure (no MCP access) and emits
// rawCall entries: direct reads, Delegate refs for packet-reading helpers
// (Task 1.3), and Unresolved markers where it cannot prove an element
// (Task 1.5). Label/comment capture is best-effort; only op width + order are
// load-bearing.
func ParseDecompile(text string) ([]rawCall, error) {
	var out []rawCall
	for _, line := range strings.Split(text, "\n") {
		m := reDecode.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		op := opName(m[2])
		if op == "" {
			continue
		}
		comment := ""
		if c := reComment.FindStringSubmatch(line); c != nil {
			comment = c[1]
		}
		out = append(out, rawCall{Op: op, Comment: comment})
	}
	return out, nil
}
```
Delete the old `ParseDecompile` stub from `mcp.go` (and its now-unused `errors`
import if nothing else uses it).

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestParseLinearReads -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/parse.go tools/packet-audit/internal/idasrc/parse_test.go \
        tools/packet-audit/internal/idasrc/testdata/linear.c tools/packet-audit/internal/idasrc/mcp.go
git commit -m "task-081(P1): ParseDecompile linear read extraction"
```

### Task 1.3: Descent — emit `Delegate` for packet-reading helpers; loop-vs-struct

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/parse.go`
- Modify: `tools/packet-audit/internal/idasrc/parse_test.go`
- Create: `tools/packet-audit/internal/idasrc/testdata/struct_helper.c`
- Create: `tools/packet-audit/internal/idasrc/testdata/count_loop.c`
- Create: `tools/packet-audit/internal/idasrc/testdata/nonpacket_skip.c`

- [ ] **Step 1: Write the fixtures**

`testdata/struct_helper.c` — a fixed-struct read via a packet-passing helper (the BuddyInvite shape):
```c
int __thiscall CWvsContext::OnFriendResult(CWvsContext *this, CInPacket *a2)
{
  int characterId = CInPacket::Decode4(a2);     // friendId
  CInPacket::DecodeStr(a2, &name);              // name
  CFriend::Insert(&friendRec, a2);             // GW_Friend struct
  unsigned __int8 inShop = CInPacket::Decode1(a2); // inShop
  return characterId;
}
```
`testdata/count_loop.c` — a genuine count-prefixed loop:
```c
int __thiscall CParty::OnList(CParty *this, CInPacket *a2)
{
  int count = CInPacket::Decode4(a2);            // memberCount
  for ( i = 0; i < count; ++i )
  {
    CInPacket::Decode4(a2);                       // member id
    CInPacket::DecodeStr(a2, &name);             // member name
  }
  return count;
}
```
`testdata/nonpacket_skip.c` — a helper that does NOT take the packet var (must be skipped) plus a denylisted UI call that does:
```c
int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)
{
  int id = CInPacket::Decode4(a2);              // id
  CUIFadeYesNo::Create(a2);                      // denylisted UI helper (skip)
  StringPool::GetString(&s, id);                 // does not take a2 (skip)
  return id;
}
```

- [ ] **Step 2: Write the failing tests**

Add to `parse_test.go`:
```go
func TestParseStructHelperDelegate(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "struct_helper.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// Expect: Decode4 friendId, DecodeStr name, Delegate->CFriend::Insert, Decode1 inShop
	if len(calls) != 4 {
		t.Fatalf("got %d calls, want 4: %+v", len(calls), calls)
	}
	if calls[2].Op != "Delegate" || calls[2].Ref != "CFriend::Insert" {
		t.Errorf("call[2] = %+v, want Delegate ref=CFriend::Insert", calls[2])
	}
	if calls[3].Op != "Decode1" {
		t.Errorf("call[3].Op = %q, want Decode1 (trailing inShop not truncated)", calls[3].Op)
	}
}

func TestParseCountLoop(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "count_loop.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// count read, then loop-body reads guarded "loop <count>"
	if calls[0].Op != "Decode4" {
		t.Fatalf("call[0] = %+v, want count Decode4", calls[0])
	}
	if !strings.HasPrefix(calls[1].Guard, "loop ") {
		t.Errorf("call[1].Guard = %q, want 'loop ...' prefix", calls[1].Guard)
	}
	if calls[1].Op != "Decode4" || calls[2].Op != "DecodeStr" {
		t.Errorf("loop body ops = %q,%q want Decode4,DecodeStr", calls[1].Op, calls[2].Op)
	}
}

func TestParseSkipsNonPacketHelpers(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "nonpacket_skip.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	for _, c := range calls {
		if c.Op == "Delegate" {
			t.Errorf("unexpected Delegate %q — non-packet/denylisted helpers must be skipped", c.Ref)
		}
	}
	if len(calls) != 1 || calls[0].Op != "Decode4" {
		t.Fatalf("got %+v, want only [Decode4 id]", calls)
	}
}
```

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./internal/idasrc/ -run 'TestParseStructHelperDelegate|TestParseCountLoop|TestParseSkipsNonPacketHelpers' -v`
Expected: FAIL — no Delegate emission, no loop guard, helpers not skipped.

- [ ] **Step 4: Implement descent + loop + denylist**

Extend `parse.go`. Track the packet variable (first arg of the first Decode/Encode
call). Track brace depth and `for`/`while`/`do` to tag loop-body reads with a
`Guard: "loop <countComment>"`. For each call line that is NOT a Decode/Encode:
classify it.
```go
var (
	rePktVar  = regexp.MustCompile(`C(?:In|Out)Packet::(?:Decode|Encode)\w+\s*\(\s*([A-Za-z_]\w*)`)
	reCall    = regexp.MustCompile(`([A-Za-z_]\w*(?:::[A-Za-z_]\w*)+)\s*\(([^;]*)\)`)
	reForCount = regexp.MustCompile(`<\s*([A-Za-z_]\w*)`)
)

// denylist: helpers that take a packet pointer but never read the wire
// (UI/dialog/alloc). Matched by "Class::" prefix.
var helperDenylist = []string{
	"CUIFadeYesNo::", "CUIDlg", "StringPool::", "operator new", "CWnd::",
	"ZAllocEx", "ZArray", "free", "malloc",
}

func isDenylisted(name string) bool {
	for _, d := range helperDenylist {
		if strings.Contains(name, d) {
			return true
		}
	}
	return false
}
```
In the line walk:
- Determine `pktVar` once (first `rePktVar` match).
- Maintain a loop-guard string: on entering a `for (...)` whose condition matches
  `reForCount`, set `loopGuard = "loop " + <var>`; clear it at the matching closing brace.
- A Decode/Encode line → emit `rawCall{Op: op, Comment: comment, Guard: loopGuard}`.
- A non-Decode call line matching `reCall`: if the arg list contains `pktVar` as a
  standalone argument AND the name is not denylisted → emit
  `rawCall{Op: "Delegate", Ref: name, Comment: comment, Guard: loopGuard}`. Otherwise skip.

Keep it a brace-depth + keyword scanner (not a full C parser) per design §3.2. A
helper that passes the packet var is a packet-reading helper → Delegate (fixed
struct). A genuine inline loop body reads directly → loop-guarded reads. This is
the loop-vs-fixed-struct disambiguation: a helper descent (Delegate) is a struct;
an in-body `Decode*` under a count `for` is a loop.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run 'TestParseStructHelperDelegate|TestParseCountLoop|TestParseSkipsNonPacketHelpers' -v`
Expected: PASS. Also run the whole package: `go test ./internal/idasrc/ -v` (Task 1.2 still green).

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/parse.go tools/packet-audit/internal/idasrc/parse_test.go \
        tools/packet-audit/internal/idasrc/testdata/struct_helper.c \
        tools/packet-audit/internal/idasrc/testdata/count_loop.c \
        tools/packet-audit/internal/idasrc/testdata/nonpacket_skip.c
git commit -m "task-081(P1): ParseDecompile struct-helper descent + loop-vs-struct"
```

### Task 1.4: Mode-switch sub-cases

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/parse.go`
- Modify: `tools/packet-audit/internal/idasrc/parse_test.go`
- Create: `tools/packet-audit/internal/idasrc/testdata/mode_switch.c`

- [ ] **Step 1: Write the fixture**

`testdata/mode_switch.c` — a discriminator `switch` with per-case reads:
```c
int __thiscall CField::OnPacket(CField *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);  // mode
  switch ( mode )
  {
    case 0:
      CInPacket::Decode4(a2);                       // case0 id
      break;
    case 1:
      CInPacket::DecodeStr(a2, &s);                // case1 name
      CInPacket::Decode2(a2);                       // case1 qty
      break;
  }
  return mode;
}
```

- [ ] **Step 2: Write the failing test**

```go
func TestParseModeSwitch(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "mode_switch.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// mode read unguarded; case bodies guarded "mode == N"
	if calls[0].Op != "Decode1" || calls[0].Guard != "" {
		t.Fatalf("call[0] = %+v, want unguarded mode Decode1", calls[0])
	}
	byGuard := map[string][]string{}
	for _, c := range calls[1:] {
		byGuard[c.Guard] = append(byGuard[c.Guard], c.Op)
	}
	if got := byGuard["mode == 0"]; len(got) != 1 || got[0] != "Decode4" {
		t.Errorf("case 0 = %v, want [Decode4]", got)
	}
	if got := byGuard["mode == 1"]; len(got) != 2 || got[0] != "DecodeStr" || got[1] != "Decode2" {
		t.Errorf("case 1 = %v, want [DecodeStr Decode2]", got)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseModeSwitch -v`
Expected: FAIL — case reads currently emitted unguarded.

- [ ] **Step 4: Implement switch/case guard tracking**

In the scanner, track the most recent `switch ( <var> )` discriminator and the
current `case N:` label; tag reads inside a case with `Guard: "<var> == N"`. Reset
the case guard at `break;`/`default:`/closing brace of the switch. Compose with the
loop guard if both are active (AND them, e.g. `"mode == 1 && loop count"`).

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestParseModeSwitch -v` → PASS.
Run full package: `go test ./internal/idasrc/` → all green.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/parse.go tools/packet-audit/internal/idasrc/parse_test.go \
        tools/packet-audit/internal/idasrc/testdata/mode_switch.c
git commit -m "task-081(P1): ParseDecompile mode-switch sub-case guards"
```

### Task 1.5: `Unresolved` fallback (the anti-BuddyInvite invariant)

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/parse.go`
- Modify: `tools/packet-audit/internal/idasrc/parse_test.go`
- Create: `tools/packet-audit/internal/idasrc/testdata/indirect_dispatch.c`

- [ ] **Step 1: Write the fixture**

`testdata/indirect_dispatch.c` — an unprovable indirect read:
```c
int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)
{
  int id = CInPacket::Decode4(a2);             // id
  (*(void (__thiscall **)(CFoo *, CInPacket *))(*this + 4 * id))(this, a2); // vtable dispatch
  return id;
}
```

- [ ] **Step 2: Write the failing test**

```go
func TestParseUnresolvedIndirect(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "indirect_dispatch.c"))
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	last := calls[len(calls)-1]
	if last.Op != "Unresolved" {
		t.Fatalf("last call = %+v, want Unresolved (indirect dispatch must not be guessed)", last)
	}
	if last.Comment == "" {
		t.Errorf("Unresolved must carry a reason comment")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseUnresolvedIndirect -v`
Expected: FAIL — indirect call silently dropped.

- [ ] **Step 4: Implement the unresolved fallback**

In the scanner, detect a call through a function pointer / vtable on a line that
also passes the packet var (regex for `(*(...))(... a2 ...)` or `(*v\d+)(` patterns
passing `pktVar`). Emit `rawCall{Op: "Unresolved", Comment: "indirect dispatch via vtable; hand-trace"}`.
General rule (design §1.1): when a packet-consuming construct cannot be proven a
read, a loop, or a named helper → `Unresolved`, never a guess.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestParseUnresolvedIndirect -v` → PASS.
Full package green: `go test ./internal/idasrc/`.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/parse.go tools/packet-audit/internal/idasrc/parse_test.go \
        tools/packet-audit/internal/idasrc/testdata/indirect_dispatch.c
git commit -m "task-081(P1): ParseDecompile unresolved fallback (anti-BuddyInvite invariant)"
```

### Task 1.6: Extend `MCPClient` (`GetCallees`, `StructInfo`) + result types

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/mcp.go:10-15`
- Modify: `tools/packet-audit/internal/idasrc/mcp_test.go`

- [ ] **Step 1: Write the failing test (fake client satisfies the wider interface)**

In `mcp_test.go`:
```go
type fakeClient struct {
	addrs    map[string]string            // name -> addr
	decomp   map[string]string            // addr -> text
	callees  map[string][]Callee          // addr -> callees
	structs  map[string]StructLayout      // name -> layout
}

func (f *fakeClient) GetFunctionByName(_ context.Context, n string) (string, bool, error) {
	a, ok := f.addrs[n]
	return a, ok, nil
}
func (f *fakeClient) DecompileFunction(_ context.Context, a string) (string, error) {
	return f.decomp[a], nil
}
func (f *fakeClient) GetCallees(_ context.Context, a string) ([]Callee, error) {
	return f.callees[a], nil
}
func (f *fakeClient) StructInfo(_ context.Context, n string) (StructLayout, error) {
	return f.structs[n], nil
}

func TestFakeClientSatisfiesInterface(t *testing.T) {
	var _ MCPClient = (*fakeClient)(nil)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestFakeClientSatisfiesInterface -v`
Expected: FAIL — `Callee`/`StructLayout` undefined; interface lacks the two methods.

- [ ] **Step 3: Extend the interface + add result types**

In `mcp.go`:
```go
type Callee struct {
	Name string
	Addr string
}

type StructField struct {
	Name   string
	Offset int
	Size   int // bytes
}

type StructLayout struct {
	Name   string
	Size   int
	Fields []StructField
}

type MCPClient interface {
	GetFunctionByName(ctx context.Context, name string) (addr string, ok bool, err error)
	DecompileFunction(ctx context.Context, addr string) (text string, err error)
	GetCallees(ctx context.Context, addr string) ([]Callee, error)
	StructInfo(ctx context.Context, name string) (StructLayout, error)
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestFakeClientSatisfiesInterface -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/idasrc/mcp.go tools/packet-audit/internal/idasrc/mcp_test.go
git commit -m "task-081(P1): extend MCPClient with GetCallees + StructInfo"
```

### Task 1.7: Real MCP-HTTP client (`mcphttp.go`)

**Files:**
- Create: `tools/packet-audit/internal/idasrc/mcphttp.go`
- Create: `tools/packet-audit/internal/idasrc/mcphttp_test.go`

- [ ] **Step 1: Write the failing test with a fake `http.RoundTripper`**

`mcphttp_test.go`:
```go
package idasrc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)),
		Header: http.Header{"Content-Type": []string{"application/json"}}}
}

func TestMCPHTTPGetFunctionByName(t *testing.T) {
	var lastMethod string
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		lastMethod = req.Method
		// MCP tools/call returns {result:{content:[{type:"text",text:"..."}]}}
		return jsonResp(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]any{
				"content": []map[string]any{{"type": "text", "text": "0x5dc600"}},
			},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	addr, ok, err := c.GetFunctionByName(context.Background(), "CLogin::OnFoo")
	if err != nil || !ok {
		t.Fatalf("GetFunctionByName err=%v ok=%v", err, ok)
	}
	if addr != "0x5dc600" {
		t.Errorf("addr = %q, want 0x5dc600", addr)
	}
	if lastMethod != "tools/call" {
		t.Errorf("method = %q, want tools/call", lastMethod)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestMCPHTTPGetFunctionByName -v`
Expected: FAIL — `NewMCPHTTPClient` undefined.

- [ ] **Step 3: Implement the MCP-HTTP client**

`mcphttp.go` — a JSON-RPC-over-HTTP client implementing `MCPClient`. Map each
method to its IDA-MCP tool (`get_function_by_name`, `decompile_function`,
`get_callees`, `analyze_struct_detailed`). Lifecycle: lazily `initialize` +
`notifications/initialized` once per client, then `tools/call` per request; reuse
one session. Parse the `result.content[].text` payload. Error loudly on non-200,
JSON-RPC `error`, or empty content (design §3.1 resilience — never silently
produce an empty export).
```go
type MCPHTTPClient struct {
	url    string
	http   *http.Client
	inited bool
	nextID int
}

func NewMCPHTTPClient(url string, hc *http.Client) *MCPHTTPClient {
	if hc == nil {
		hc = &http.Client{}
	}
	return &MCPHTTPClient{url: url, http: hc}
}

// callTool issues a tools/call and returns the concatenated text content.
func (c *MCPHTTPClient) callTool(ctx context.Context, tool string, args map[string]any) (string, error) {
	// initialize once (implement per MCP spec: initialize ->
	// notifications/initialized), then:
	// POST {jsonrpc,id,method:"tools/call",params:{name:tool,arguments:args}}
	// parse result.content[].text; error on rpc error / empty content.
	return "", nil // replace with real implementation
}

func (c *MCPHTTPClient) GetFunctionByName(ctx context.Context, name string) (string, bool, error) {
	out, err := c.callTool(ctx, "get_function_by_name", map[string]any{"name": name})
	if err != nil {
		return "", false, err
	}
	out = strings.TrimSpace(out)
	if out == "" || strings.Contains(strings.ToLower(out), "not found") {
		return "", false, nil
	}
	return out, true, nil
}
// DecompileFunction -> "decompile_function", GetCallees -> "get_callees",
// StructInfo -> "analyze_struct_detailed", each parsing the text payload.
```
Note: the exact IDA-MCP text-payload shape (e.g. whether `get_callees` returns JSON
or a formatted list) is confirmed against the live server in Phase 2; parse
defensively and unit-test the parsing of each shape with a fixture string.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestMCPHTTP -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/idasrc/mcphttp.go tools/packet-audit/internal/idasrc/mcphttp_test.go
git commit -m "task-081(P1): real MCP-HTTP client implementing MCPClient"
```

### Task 1.8: `Harvest` descent driver

**Files:**
- Create: `tools/packet-audit/internal/idasrc/harvest.go`
- Create: `tools/packet-audit/internal/idasrc/harvest_test.go`

- [ ] **Step 1: Write the failing test (single parent + one helper descent)**

`harvest_test.go`:
```go
func TestHarvestDescendsHelper(t *testing.T) {
	fc := &fakeClient{
		addrs: map[string]string{
			"CWvsContext::OnFriendResult": "0xA0",
			"CFriend::Insert":             "0xB0",
		},
		decomp: map[string]string{
			"0xA0": mustFixture(t, "struct_helper.c"),       // emits Delegate->CFriend::Insert
			"0xB0": "int CFriend::Insert(GW_Friend*r, CInPacket*a2){ CInPacket::Decode4(a2); CInPacket::Decode2(a2); }",
		},
	}
	ef, err := Harvest(context.Background(), fc,
		[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 4})
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	// Both parent and discovered helper are exported as their own entries.
	if _, ok := ef.Functions["CWvsContext::OnFriendResult"]; !ok {
		t.Fatal("parent missing from export")
	}
	helper, ok := ef.Functions["CFriend::Insert"]
	if !ok {
		t.Fatal("descended helper CFriend::Insert missing from export")
	}
	if len(helper.Calls) != 2 {
		t.Errorf("helper calls = %d, want 2", len(helper.Calls))
	}
	// Parent retains the Delegate ref (resolver inlines at audit time).
	parent := ef.Functions["CWvsContext::OnFriendResult"]
	foundDelegate := false
	for _, c := range parent.Calls {
		if c.Op == "Delegate" && c.Ref == "CFriend::Insert" {
			foundDelegate = true
		}
	}
	if !foundDelegate {
		t.Error("parent missing Delegate->CFriend::Insert")
	}
}

func TestHarvestCycleGuard(t *testing.T) {
	fc := &fakeClient{
		addrs:  map[string]string{"A::f": "0x1", "B::f": "0x2"},
		decomp: map[string]string{
			"0x1": "void A::f(X*x,CInPacket*a2){ CInPacket::Decode1(a2); B::f(x,a2); }",
			"0x2": "void B::f(X*x,CInPacket*a2){ CInPacket::Decode1(a2); A::f(x,a2); }",
		},
	}
	ef, err := Harvest(context.Background(), fc, []string{"A::f"}, HarvestOpts{DescentDepth: 8})
	if err != nil {
		t.Fatalf("Harvest must not loop forever / error on cycle: %v", err)
	}
	// Both exported once; the cycle terminates (resolver enforces cycle at resolve time).
	if _, ok := ef.Functions["A::f"]; !ok {
		t.Error("A::f missing")
	}
	if _, ok := ef.Functions["B::f"]; !ok {
		t.Error("B::f missing")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestHarvest -v`
Expected: FAIL — `Harvest`/`HarvestOpts` undefined.

- [ ] **Step 3: Implement `Harvest`**

`harvest.go` — BFS over the roster + discovered Delegate refs:
```go
type HarvestOpts struct {
	DescentDepth int    // max helper recursion depth; 0 => default 6
	Binary       string // provenance
	MD5          string
	GeneratedAt  string
}

// Harvest drives the MCP client over the roster, parsing each function and
// enqueuing every discovered packet-reading helper (Delegate ref) as its own
// export entry. Cycle-guarded (visited set) and depth-bounded; a function past
// the depth bound on a still-descending path is marked Unresolved rather than
// truncated.
func Harvest(ctx context.Context, c MCPClient, roster []string, opts HarvestOpts) (exportFile, error) {
	if opts.DescentDepth == 0 {
		opts.DescentDepth = 6
	}
	out := exportFile{Binary: opts.Binary, MD5: opts.MD5,
		GeneratedAt: opts.GeneratedAt, Functions: map[string]exportFn{}}
	type item struct {
		name  string
		depth int
	}
	queue := make([]item, 0, len(roster))
	for _, n := range roster {
		queue = append(queue, item{n, 0})
	}
	visited := map[string]bool{}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		if visited[it.name] {
			continue
		}
		visited[it.name] = true
		addr, ok, err := c.GetFunctionByName(ctx, it.name)
		if err != nil {
			return out, fmt.Errorf("harvest %s: %w", it.name, err)
		}
		if !ok {
			out.Functions[it.name] = exportFn{Unresolved: true,
				Calls: []rawCall{{Op: "Unresolved", Comment: "function not found in IDB"}}}
			continue
		}
		text, err := c.DecompileFunction(ctx, addr)
		if err != nil {
			return out, fmt.Errorf("harvest %s decompile: %w", it.name, err)
		}
		calls, err := ParseDecompile(text)
		if err != nil {
			return out, fmt.Errorf("harvest %s parse: %w", it.name, err)
		}
		fn := exportFn{Address: addr, Calls: calls}
		// enqueue discovered helpers (Delegate refs)
		for _, cl := range calls {
			if cl.Op == "Delegate" && cl.Ref != "" && !visited[cl.Ref] {
				if it.depth+1 > opts.DescentDepth {
					fn.Unresolved = true // descent too deep to prove
					continue
				}
				queue = append(queue, item{cl.Ref, it.depth + 1})
			}
		}
		out.Functions[it.name] = fn
	}
	return out, nil
}
```
Direction assignment: `Harvest` leaves `Direction` empty; `runExport` (Task 1.10)
sets each function's direction from the prior export's value (or `candidatesFromFName`
direction), preserving the established mapping.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestHarvest -v` → PASS. Full package green.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/idasrc/harvest.go tools/packet-audit/internal/idasrc/harvest_test.go
git commit -m "task-081(P1): Harvest descent driver (BFS, cycle/depth guard)"
```

### Task 1.9: Four-version BuddyInvite regression (the canonical anti-case)

**Files:**
- Modify: `tools/packet-audit/internal/idasrc/harvest_test.go`
- Create: `tools/packet-audit/internal/idasrc/testdata/friend_v83.c`, `friend_v87.c`, `friend_jms.c`, `friend_insert.c`

- [ ] **Step 1: Write the four-version fixtures**

The parent `OnFriendResult` case-9 (#Invite) per version. v83 has no jobId/level;
v87 + JMS add them; all end with `GW_Friend` (via `CFriend::Insert`) + `inShop`.
`friend_insert.c` is the shared helper reading the 39-byte struct as primitives
(synthetic breakdown summing to 39 bytes — the structural stand-in; the real layout
is captured live in Phase 2):
```c
// friend_insert.c — GW_Friend 39 bytes as primitives
int __thiscall CFriend::Insert(GW_Friend *r, CInPacket *a2)
{
  CInPacket::Decode4(a2);              // friendId (4)
  CInPacket::DecodeBuffer(a2, 13);     // name[13] (13)
  CInPacket::Decode1(a2);             // flag (1)
  CInPacket::Decode4(a2);             // ... (4)
  CInPacket::DecodeBuffer(a2, 17);     // group[17] (17)
  return 0;                            // total 39
}
```
`friend_v83.c` (no jobId/level):
```c
int __thiscall CWvsContext::OnFriendResult(CWvsContext *this, CInPacket *a2)
{
  unsigned __int8 mode = CInPacket::Decode1(a2);   // mode
  switch ( mode ) {
    case 9:
      CInPacket::Decode4(a2);          // friendId
      CInPacket::DecodeStr(a2, &name);// name
      CFriend::Insert(&rec, a2);       // GW_Friend(39)
      CInPacket::Decode1(a2);          // inShop
      break;
  }
  return mode;
}
```
`friend_v87.c` / `friend_jms.c` — identical but add `CInPacket::Decode4(a2); // jobId`
and `CInPacket::Decode4(a2); // level` between `name` and `CFriend::Insert`.

- [ ] **Step 2: Write the failing regression test**

```go
func resolveHarvested(t *testing.T, ef exportFile, fname string) Fields {
	t.Helper()
	src := newExportSourceFromFile(ef) // in-package constructor over exportFile
	f, err := src.Resolve(context.Background(), fname)
	if err != nil {
		t.Fatalf("resolve %s: %v", fname, err)
	}
	return f
}

func TestBuddyInviteFourVersion(t *testing.T) {
	cases := []struct {
		name     string
		fixture  string
		wantHead []Primitive // before GW_Friend
	}{
		{"v83", "friend_v83.c", []Primitive{Decode4, DecodeStr}},
		{"v87", "friend_v87.c", []Primitive{Decode4, DecodeStr, Decode4, Decode4}},
		{"jms", "friend_jms.c", []Primitive{Decode4, DecodeStr, Decode4, Decode4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeClient{
				addrs: map[string]string{"CWvsContext::OnFriendResult": "0xA", "CFriend::Insert": "0xB"},
				decomp: map[string]string{
					"0xA": mustFixture(t, tc.fixture),
					"0xB": mustFixture(t, "friend_insert.c"),
				},
			}
			ef, err := Harvest(context.Background(), fc,
				[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 4})
			if err != nil {
				t.Fatalf("Harvest: %v", err)
			}
			f := resolveHarvested(t, ef, "CWvsContext::OnFriendResult")
			// Filter to the case-9 (#Invite) reads via guard "mode == 9".
			var inv []FieldCall
			for _, c := range f.Calls {
				if strings.Contains(c.Guard, "mode == 9") {
					inv = append(inv, c)
				}
			}
			// head + 5 GW_Friend prims (4,13,1,4,17 → 5 calls) + inShop(1)
			wantLen := len(tc.wantHead) + 5 + 1
			if len(inv) != wantLen {
				t.Fatalf("%s: got %d invite calls, want %d: %+v", tc.name, len(inv), wantLen, inv)
			}
			for i, w := range tc.wantHead {
				if inv[i].Op != w {
					t.Errorf("%s head[%d] = %v, want %v", tc.name, i, inv[i].Op, w)
				}
			}
			// MUST NOT be a count-loop and MUST NOT truncate before inShop.
			last := inv[len(inv)-1]
			if last.Op != Decode1 {
				t.Errorf("%s: last invite call = %v, want Decode1 inShop (no truncation)", tc.name, last.Op)
			}
			for _, c := range inv {
				if strings.HasPrefix(c.Guard, "loop ") {
					t.Errorf("%s: GW_Friend mistraced as a loop (%+v)", tc.name, c)
				}
			}
		})
	}
}
```
(If an in-package `exportFile`→`ExportSource` constructor does not exist, add a tiny
unexported `newExportSourceFromFile(exportFile) *ExportSource` in `export.go` and
reuse it from `runExport` too.)

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestBuddyInviteFourVersion -v`
Expected: FAIL until `newExportSourceFromFile` exists and the scanner handles the
case-9 + helper composition.

- [ ] **Step 4: Add the in-package constructor; make it pass**

Add `newExportSourceFromFile`. No parser change should be needed if Tasks 1.3–1.4
are correct; if the test surfaces a real gap (e.g. `DecodeBuffer` width handling),
fix `parse.go` minimally and note it.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./internal/idasrc/ -run TestBuddyInviteFourVersion -v` → PASS for all three subtests.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/idasrc/harvest_test.go tools/packet-audit/internal/idasrc/export.go \
        tools/packet-audit/internal/idasrc/testdata/friend_*.c
git commit -m "task-081(P1): four-version BuddyInvite descent regression (anti-BuddyInvite invariant)"
```

### Task 1.10: `runExport` driver + CLI flags

**Files:**
- Rewrite: `tools/packet-audit/cmd/export.go`
- Modify: `tools/packet-audit/cmd/root.go` (export flag parsing)
- Create: `tools/packet-audit/cmd/export_test.go`

- [ ] **Step 1: Write the failing test (roster build + determinism, fake client)**

`export_test.go` — test the roster assembly + deterministic write via an injectable
client (refactor `runExport` so its core is `exportRun(opts, client, stdout, stderr)`
and the flag-parsing wrapper builds the real MCP-HTTP client):
```go
func TestExportRunDeterministic(t *testing.T) {
	fc := newFakeClient(/* two functions, one helper */)
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")
	opts := exportOpts{Version: "gms_v95", Output: out, PriorExport: "testdata/gms_v95_mini.json"}
	if code := exportRun(opts, fc, io.Discard, io.Discard); code != 0 {
		t.Fatalf("exportRun exit = %d", code)
	}
	a, _ := os.ReadFile(out)
	// re-run must be byte-identical (determinism, FR-1.5)
	_ = exportRun(opts, fc, io.Discard, io.Discard)
	b, _ := os.ReadFile(out)
	if !bytes.Equal(a, b) {
		t.Error("export not deterministic across runs")
	}
	// keys sorted: assert the function-name keys appear in sorted order in the raw bytes
	var ef struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	if err := json.Unmarshal(a, &ef); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
```
(The fake client lives in `idasrc`. Expose it to the `cmd` package via a tiny
`idasrc/idasrctest` helper package (constructor `New(...)`) referenced by both test
packages, OR keep the cmd test using the real `MCPHTTPClient` against a fake
`http.RoundTripper` like Task 1.7. Choose the lower-churn option at implementation
time and keep it consistent.)

- [ ] **Step 2: Run to verify it fails**

Run: `cd tools/packet-audit && go test ./cmd/ -run TestExportRunDeterministic -v`
Expected: FAIL — `exportRun`/`exportOpts` undefined.

- [ ] **Step 3: Implement the driver**

Rewrite `export.go`:
- Parse export flags (Step 4) into `exportOpts{Version, Output, PriorExport, IDAURL, IDATimeout, DescentDepth, GeneratedAt}`.
- **Roster** = union of (a) keys in the prior version JSON, (b) `candidatesFromFName`
  FName set, (c) FNames listed in `docs/packets/ida-exports/_pending.md`. De-dup, sort.
- Call `idasrc.Harvest(ctx, client, roster, HarvestOpts{...})`.
- **Direction backfill:** for each harvested fn, set `Direction` from the prior
  export entry if present, else from `candidatesFromFName(fname)[0].dir`.
- **Determinism:** marshal with sorted keys (Go's `encoding/json` sorts `map[string]`
  keys already; ensure call order is the parser's source order) and stable
  indentation. Re-runs must be byte-identical.
- **Provenance:** `Binary`, `MD5`, `GeneratedAt`. Pass `GeneratedAt` in via flag/env
  to keep tests deterministic — do NOT call `time.Now()` in the core; default to a
  fixed value when unset so re-runs are byte-identical.
- **Unresolved summary** to stderr: counts of resolved / descended-helper / unresolved
  + the unresolved FName list (observability NFR).

- [ ] **Step 4: Wire the export flags in `root.go`**

In `Run`, when `args[0] == "export"`, parse a dedicated flag set:
```go
fs.StringVar(&eo.IDAURL, "ida-url", "http://192.168.20.3:13337/mcp", "IDA-MCP HTTP endpoint")
fs.DurationVar(&eo.IDATimeout, "ida-timeout", 60*time.Second, "per-call IDA-MCP timeout")
fs.StringVar(&eo.Version, "version", "", "target version key, e.g. gms_v95 (required)")
fs.IntVar(&eo.DescentDepth, "descent-depth", 6, "max helper-descent recursion depth")
fs.StringVar(&eo.Output, "output", "", "output JSON path (required)")
```
Build the real client: `idasrc.NewMCPHTTPClient(eo.IDAURL, &http.Client{Timeout: eo.IDATimeout})`,
derive `PriorExport` from `--version` (`docs/packets/ida-exports/<version>.json`),
then call `exportRun(eo, client, os.Stdout, stderr)`.

- [ ] **Step 5: Run to verify pass**

Run: `go test ./cmd/ -run TestExportRunDeterministic -v` → PASS.
Run the whole tool: `go test ./...` in `tools/packet-audit` → all green.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/export.go tools/packet-audit/cmd/root.go tools/packet-audit/cmd/export_test.go
git commit -m "task-081(P1): runExport driver — roster, determinism, provenance, unresolved summary"
```

### Task 1.11: Audit consumer renders `Unresolved` distinctly

**Files:**
- Modify: `tools/packet-audit/internal/diff/diff.go:10-75`
- Modify: `tools/packet-audit/internal/diff/diff_test.go`
- Modify (if needed): `tools/packet-audit/internal/report/report.go`

- [ ] **Step 1: Write the failing test**

In `diff_test.go`:
```go
func TestDiffUnresolvedRow(t *testing.T) {
	atlas := []atlaspacket.Call{{Op: atlaspacket.Encode4}}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.Unresolved, Comment: "vtable"}}}
	rows := Diff(atlas, ida)
	if rows[0].Verdict != VerdictUnresolved {
		t.Errorf("verdict = %v (%s), want VerdictUnresolved", rows[0].Verdict, rows[0].Verdict.Symbol())
	}
}

func TestVerdictUnresolvedSymbol(t *testing.T) {
	if VerdictUnresolved.Symbol() != "🚫" {
		t.Errorf("symbol = %q, want 🚫", VerdictUnresolved.Symbol())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/diff/ -run 'TestDiffUnresolvedRow|TestVerdictUnresolvedSymbol' -v`
Expected: FAIL — `VerdictUnresolved` undefined.

- [ ] **Step 3: Implement**

In `diff.go`:
```go
const (
	VerdictMatch      Verdict = iota // ✅
	VerdictMinor                     // ⚠️
	VerdictBlocker                   // ❌
	VerdictDeferred                  // 🔍
	VerdictUnresolved                // 🚫 — IDA read-order unknown (export gap)
)

func (v Verdict) Symbol() string {
	return [...]string{"✅", "⚠️", "❌", "🔍", "🚫"}[v]
}
```
In the `Diff` switch, add a case BEFORE the width-mismatch case so an Unresolved IDA
op short-circuits to the known-gap verdict:
```go
		case i < len(ida.Calls) && ida.Calls[i].Op == idasrc.Unresolved:
			r.Verdict = VerdictUnresolved
			r.Note = "IDA read-order unresolved: " + ida.Calls[i].Comment
```
Confirm `report.go` SUMMARY tallies count `VerdictUnresolved` as its own bucket
(if it aggregates by symbol it already works; if it hardcodes the four old symbols,
add the fifth).

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/diff/ ./internal/report/ -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/diff/diff.go tools/packet-audit/internal/diff/diff_test.go tools/packet-audit/internal/report/report.go
git commit -m "task-081(P1): audit renders Unresolved (🚫) as a distinct known-gap verdict"
```

### Phase 1 GATE

- [ ] **Run the full module test suite, vet, build**

```bash
cd tools/packet-audit
go test -race ./...    # all green incl. BuddyInvite four-version regression
go vet ./...
go build ./...
```
Expected: all clean. The four-version `OnFriendResult#Invite` regression (Task 1.9)
proves descent + loop-disambiguation + no-truncation. Do not proceed to Phase 2
until this gate is green.

---

## Phase 2 — Full re-export (LIVE IDA, maintainer-cycled)

> Requires a maintainer to load each IDB one at a time (no MCP tool switches IDBs
> — memory `reference_ida_harvest_subagents`). CI never runs this phase.

### Task 2.1: Per-IDB re-export (×4)

**Files:**
- Replace: `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json`

For **each** version (`gms_v83`, `gms_v87`, `gms_v95`, `gms_jms_185`), in turn:

- [ ] **Step 1: Maintainer loads the matching IDB** (ask the user to confirm; verify reachability)

Verify the endpoint is live and serving the right binary:
```bash
go run ./tools/packet-audit export --version <ver> --output /tmp/<ver>.json --ida-url http://192.168.20.3:13337/mcp 2>/tmp/<ver>.stderr || true
head -5 /tmp/<ver>.json   # check "binary"/"md5" match the loaded IDB
cat /tmp/<ver>.stderr     # resolved/descended/unresolved summary
```
If the endpoint is unreachable or the binary/md5 is wrong, STOP and ask the
maintainer to load the correct IDB (the client errors loudly by design).

- [ ] **Step 2: Write the real export to the repo path**

```bash
go run ./tools/packet-audit export --version <ver> \
  --output docs/packets/ida-exports/<ver>.json --ida-url http://192.168.20.3:13337/mcp 2>/tmp/<ver>.stderr
```

- [ ] **Step 3: Determinism check (re-run must be byte-identical)**

```bash
cp docs/packets/ida-exports/<ver>.json /tmp/<ver>.a
go run ./tools/packet-audit export --version <ver> --output /tmp/<ver>.b --ida-url http://192.168.20.3:13337/mcp
diff /tmp/<ver>.a /tmp/<ver>.b   # expect no diff
```

- [ ] **Step 4: Hand-fill `unresolved` where feasible**

For each FName flagged `unresolved` in the stderr summary: read its decompile
(`decompile_function` via a focused subagent), and if it is genuinely a packet read
the parser missed, hand-author the entry (or a `Delegate`/`Ref` to a hand-traced
helper) per the existing schema. Leave a genuine `unresolved:true` marker where the
read truly cannot be statically determined (indirect/data-driven). Never replace an
`unresolved` with a guess (§1.1).

(Subagents may drive the per-IDB batch; one IDB is loaded at a time, so the four
runs are strictly serial.)

- [ ] **Step 5: GATE — all four re-exported**

All four `docs/packets/ida-exports/*.json` replaced; each re-run is byte-identical;
each stderr summary captured.

### Task 2.2: Structural-change summary + commit

**Files:**
- Create: `docs/packets/ida-exports/REEXPORT-SUMMARY-081.md`

- [ ] **Step 1: Summarize what changed structurally**

Diff each new export against its pre-re-export version and write a per-version
summary: count of struct descents resolved (new `Delegate` entries), truncations
recovered (functions that gained trailing calls), loops corrected (former count-loops
now fixed structs and vice-versa), and unresolved markers added. This lets a
reviewer audit the *exporter's* correctness, not just trust it (FR-2.2, reviewable
NFR).

- [ ] **Step 2: Commit the corrected exports + summary**

```bash
git add docs/packets/ida-exports/*.json docs/packets/ida-exports/REEXPORT-SUMMARY-081.md
git commit -m "task-081(P2): full four-version re-export via automated exporter + structural summary"
```

**Phase 2 GATE:** four exports replaced; deterministic re-run identical; structural
summary committed; unresolved counts recorded.

---

## Phase 3 — Re-audit + verdict-delta triage (§4.7)

### Task 3.1: Re-run the audit over corrected exports

**Files:**
- Regenerate: `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/**`

- [ ] **Step 1: Confirm the per-version template filenames**

```bash
ls services/atlas-configurations/seed-data/templates/template_*.json
```
Map each version key to its template file (substitute the real names below).

- [ ] **Step 2: Re-run for all four versions**

```bash
for pair in "gms_v83:template_gms_83_1" "gms_v87:template_gms_87_1" \
            "gms_v95:template_gms_95_1" "gms_jms_185:template_jms_185_1"; do
  ver=${pair%%:*}; tmpl=${pair##*:}
  go run ./tools/packet-audit \
    --csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
    --csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
    --template services/atlas-configurations/seed-data/templates/$tmpl.json \
    --ida-source docs/packets/ida-exports/$ver.json \
    --output docs/packets/audits
done
```
(`--output` is the parent dir — the tool appends `<region>_v<major>`.)

- [ ] **Step 3: Commit the regenerated audits**

```bash
git add docs/packets/audits/
git commit -m "task-081(P3): re-run audit over corrected exports"
```

### Task 3.2: Compute + disposition the per-packet verdict delta

**Files:**
- Create: `docs/tasks/task-081-ida-export-reharvest/verdict-delta-081.md`

- [ ] **Step 1: Compute the per-packet delta vs the Phase-0 snapshot (FR-7.2)**

For each version, extract the new per-packet verdicts (same `grep` as Task 0.2 Step 2)
and diff against `verdict-snapshot-080.md`. Record, per version, three sets:
`❌→✅` / `🔍→✅`, `✅→❌` (+ `✅→🔍`), and `new non-✅ on previously-unaudited`. Write
them into `verdict-delta-081.md` as a checklist (one row per flipped packet).

- [ ] **Step 2: Disposition every flip (FR-7.3, the gate)**

Walk the checklist. For each entry:
- **`❌`/`🔍`→`✅`:** spot-check a representative sample — confirm it flipped *because*
  the corrected read-order now matches (read both the old + new export entry), not by
  coincidence. Mark the row `accepted (corrected-read-order)`. Sample size ≥ 20% or 5
  rows, whichever is larger, per version.
- **`✅`→`❌`/`🔍` (DANGEROUS):** hand-decompile the function in IDA (focused subagent,
  `decompile_function` + `get_callees`) and compare to Atlas. Mark exactly one
  disposition: `(a) real-atlas-bug → Phase 4`, `(b) exporter-over-corrected → fix
  parser (back to Phase 1)`, or `(c) verified-equivalence → recorded exception (IDA
  evidence)`. NEVER act by trusting the new export alone (BuddyInvite lesson).
- **new non-`✅`:** same hand-investigation as `✅→❌`.
- **`🚫` (unresolved):** record as a known gap with the FName + reason; not a flip to
  fix unless it masks a real divergence (then treat as `new non-✅`).

- [ ] **Step 3: Commit the dispositioned delta**

```bash
git add docs/tasks/task-081-ida-export-reharvest/verdict-delta-081.md
git commit -m "task-081(P3): verdict-delta triage — every flip dispositioned"
```

**Phase 3 GATE (FR-7.5):** every `✅→❌`/new-non-`✅` flip carries a written
disposition (fixed-atlas / fixed-exporter / verified-equiv); zero rubber-stamped in
either direction. If any flip is `exporter-over-corrected`, loop back to Phase 1,
fix the parser, re-export the affected version (Phase 2), and re-run (Phase 3) before
this gate passes.

---

## Phase 4 — Fix surfaced wire bugs (task-080 discipline)

> Discovery-driven: the exact bugs come from Phase 3's `real-atlas-bug` rows. Each
> follows the identical per-bug loop below. No fix ships on analyzer verdict alone —
> a per-version byte-level test is the oracle (memory `reference_packet_audit_tool_mechanics`).

### Task 4.x (one per `real-atlas-bug` row): Fix + byte-test

**Files (per bug):**
- Modify: `libs/atlas-packet/<sub-domain>/<direction>/<packet>.go`
- Test: `libs/atlas-packet/<sub-domain>/<direction>/<packet>_test.go`
- Modify (if the field originates upstream): the surfacing `services/**` handler/producer

- [ ] **Step 1: Confirm the read-order in IDA** (hand-decompile; cite addr in the test comment — the disposition row already has this)

- [ ] **Step 2: Write the failing per-version byte test (the oracle)**

Model on `libs/atlas-packet/socket/clientbound/hello_test.go` `TestHelloWireShape`:
```go
func Test<Packet>WireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := New<Packet>(/* representative args */)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// Expected layout per <FName> @ <addr> (<version>):
			//   offset 0: Decode...
			// assert exact bytes/offsets per the IDA decompile, gated by version where it diverges.
			if v.MajorVersion >= <N> { /* assert the v≥N field present at offset X */ }
		})
	}
}
```
Also keep/add a `pt.RoundTrip` test for encode/decode symmetry. The WireShape test
(exact offsets) is what catches a wrong-but-symmetric bug a round-trip misses.

- [ ] **Step 3: Run to verify it fails** — `go test ./libs/atlas-packet/<sub>/<dir>/ -run Test<Packet>WireShape -v` → FAIL (current encoder wrong).

- [ ] **Step 4: Fix the encoder/decoder**

Apply version/region gates symmetrically in Encode AND Decode
(`t.Region() == "GMS" && t.MajorVersion() >= N`). For >2-version divergences use the
region-dispatch idiom (≤2 nested guards; analyzer-visible via task-080 A5). If the
divergent field originates upstream, thread it through the surfacing handler/producer.

- [ ] **Step 5: Run to verify pass** — WireShape + RoundTrip green for all variants.

- [ ] **Step 6: Re-audit the affected version; confirm the row is now `✅`**

```bash
go run ./tools/packet-audit --csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template services/atlas-configurations/seed-data/templates/<tmpl>.json \
  --ida-source docs/packets/ida-exports/<ver>.json --output docs/packets/audits
grep -A2 '<Packet>' docs/packets/audits/<ver>/SUMMARY.md   # now ✅
```

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/<sub>/<dir>/<packet>*.go services/... docs/packets/audits/
git commit -m "task-081(P4): fix <Packet> wire divergence (<ver>) + byte test"
```

### Task 4.Z: Register genuinely-too-large bugs as follow-ups (FR-3.4)

- [ ] If a surfaced bug is a multi-service protocol change too large for this task,
  do NOT park it in `_pending.md` as "accepted" (memory `feedback_no_todos_in_deliverables`).
  Register a dedicated follow-up task: verify the next number against `git log --all`
  (memory `reference_task_numbers_historical_gap`), then `/spec-task` it. Note the
  reference in `verdict-delta-081.md`.

**Phase 4 GATE:** every `real-atlas-bug` row is fixed-with-byte-test (and re-audits
to `✅`) OR registered as a follow-up; changed modules pass `go test -race`/`vet`/`build`;
`docker buildx bake` for every service whose `go.mod` changed.

---

## Phase 5 — Opaque register-boundary decomposition (FR-4)

> For each type in task-080's opaque set: `model.Asset`/`GW_ItemSlotBase`,
> `GW_CharacterStat`, monster stat blobs, `BuddyEntry`, pet bodies, and the
> ~31 A3-flagged types. Enumerate the exact set from task-080's `_pending.md` +
> SUMMARY "analyzer skipped" / 🔍 rows at execution start.

### Task 5.0: Enumerate the opaque set

**Files:**
- Create: `docs/tasks/task-081-ida-export-reharvest/opaque-set-081.md`

- [ ] List every opaque/skipped type from `docs/packets/audits/gms_v95/_pending.md`
  and the four SUMMARYs (🔍 sub-struct rows). One checklist row per type with its
  carrier packet(s) and current status.

### Task 5.x (one per opaque type): Decompose or verified-exception

- [ ] **Step 1: Determine decomposability in IDA** — read the client's read of the
  type (focused subagent). Decide: decomposable into known primitives, or
  mask/mode-driven variable layout.

- [ ] **Step 2a (decomposable): extend export with `Delegate` entries**

Author the type's read as its own FName entry (primitives in order) and reference it
via `Delegate`/`Ref` from each carrier packet's export entry (the same mechanism
Phase 1 automates). Re-audit → the type's fields now verify inline (real per-field
verdicts replace the 🔍).

- [ ] **Step 2b (undecomposable): byte-test-backed verified exception**

Write a per-version byte-level test (the §Phase-4 WireShape template) proving Atlas's
encoder matches the client for representative mask/mode states. Record a *verified
exception* in `opaque-set-081.md` (IDA evidence + the test name as oracle) — status
becomes "verified correct, analyzer can't model it", replacing "analyzer skipped it".

- [ ] **Step 3: Commit** (per type or small batch)

```bash
git add docs/packets/ida-exports/*.json docs/packets/audits/ libs/atlas-packet/... \
        docs/tasks/task-081-ida-export-reharvest/opaque-set-081.md
git commit -m "task-081(P5): decompose/verify <Type> opaque read"
```

**Phase 5 GATE:** no type remains in an unexamined "analyzer skipped it" state — each
is either decomposed-and-verified inline or carries a byte-test-backed verified
exception with IDA evidence.

---

## Phase 6 — Per-version template completeness (FR-5)

### Task 6.0: Enumerate unrouted families

**Files:**
- Create: `docs/tasks/task-081-ida-export-reharvest/template-gaps-081.md`

- [ ] Enumerate packet families with Atlas packet/handler code but no per-version
  routing — notably JMS NPC-shop (`NPCShopHandle`/`NPCShopOperation`) and the
  mini-room player-interaction family beyond the two ops task-080 wired. Grep
  `libs/atlas-packet` for the family structs and cross-check each version's
  `template_<region>_<major>_<minor>.json` for the op-byte routing. One row per
  (family, version) with present/absent routing.

### Task 6.x (one per gap): Wire or verify-client-absent

- [ ] **Step 1: Confirm the per-version op byte in IDA** (like task-080 B5.1f) — the
  client's handler dispatch byte for the family in that version.

- [ ] **Step 2a (present): wire the op-byte map**

Edit the version's seed template in
`services/atlas-configurations/seed-data/templates/template_<region>_<major>_<minor>.json`
to route the family's op byte(s). Validate it parses:
```bash
python3 -m json.tool services/atlas-configurations/seed-data/templates/template_<...>.json >/dev/null
```

- [ ] **Step 2b (absent): record verified client-absent verdict**

If the client of that version genuinely never sends/receives the family, record a
verified client-absent verdict (IDA evidence) in `template-gaps-081.md`.

- [ ] **Step 3 (threshold): split large families to follow-ups (PRD open-Q)**

If a family's full routing is itself a large effort, register a dedicated follow-up
task (FR-3.4 discipline) rather than bloating this task; note it in `template-gaps-081.md`.

- [ ] **Step 4: Re-audit; confirm newly-routed families appear**

Re-run the affected version's audit (Phase 3 Step 2 command) and confirm the family
now produces packet reports.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/*.json docs/packets/audits/ \
        docs/tasks/task-081-ida-export-reharvest/template-gaps-081.md
git commit -m "task-081(P6): wire/verify <family> routing for <version>"
```

**Phase 6 GATE:** every unrouted family is either wired (and audits per version) or
carries a verified client-absent verdict; all edited templates parse; no family left
without a verdict.

---

## Phase 7 — Ledger + guide (FR-6)

### Task 7.1: Re-curate both `_pending.md` registries

**Files:**
- Modify: `docs/packets/ida-exports/_pending.md`
- Modify: `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Remove the truncation/mistrace category entirely**

Every former "export read-order truncation/mistrace" row is now one of: fixed (Phase 4),
verified-exclusion (Phase 3c/5b), or an explicit `unresolved`/`🚫` marker. Delete the
category; reclassify each row to its real disposition with a one-line citation
(disposition source: `verdict-delta-081.md` / `opaque-set-081.md` / a byte-test name).

- [ ] **Step 2: Confirm zero "export was wrong" + zero unexamined opaque skips**

```bash
grep -niE "truncat|mistrace|export.*wrong|analyzer skipped" docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/_pending.md
```
Expected: no matches except where explicitly annotated as resolved/historical.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/_pending.md
git commit -m "task-081(P7): re-curate _pending registries — only verified exclusions remain"
```

### Task 7.2: Update `TOTAL.md` + `STARTING_A_NEW_VERSION_PASS.md`

**Files:**
- Modify: `docs/packets/audits/gms_v95/TOTAL.md`
- Modify: `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`

- [ ] **Step 1: `TOTAL.md`** — update the verdict roll-up (counts incl. the new 🚫
  bucket) and the completeness statement to reflect the corrected baseline.

- [ ] **Step 2: `STARTING_A_NEW_VERSION_PASS.md`** — document the automated exporter:
  the `packet-audit export --version <v> --ida-url <url> --output <path>` workflow,
  the struct-helper descent behavior, the `--descent-depth` flag, the maintainer-local
  IDB-cycling model, and the `unresolved`-over-guess invariant (§1.1) with the hand-fill
  (`Delegate`/`Ref`) procedure for `🚫` functions.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/audits/gms_v95/TOTAL.md docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md
git commit -m "task-081(P7): update TOTAL + new-version-pass guide for automated exporter"
```

**Phase 7 GATE:** zero "export was wrong" entries; both `_pending.md` contain only
verified exclusions; guide documents the new workflow + invariant.

---

## Phase 8 — Verify + code review

### Task 8.1: Full verification (CLAUDE.md gates)

- [ ] **Step 1: Per-module test/vet/build**

For every changed module (at minimum `tools/packet-audit`; plus `libs/atlas-packet`
and any `services/**` touched by wire fixes / templates):
```bash
go test -race ./...   # in each changed module
go vet ./...
go build ./...
```

- [ ] **Step 2: Docker bake for every service whose `go.mod` changed**

```bash
# from worktree root, for each touched service module:
docker buildx bake atlas-<svc>
# tools/packet-audit is a tool module (no service target). If only it + docs
# changed, the service bake set is empty; if libs/atlas-packet consumers changed,
# bake those services; if atlas-configurations seed templates changed, bake it.
```

- [ ] **Step 3: Redis key guard + nesting cap**

```bash
GOWORK=off tools/redis-key-guard.sh   # from repo root
```
Confirm the backend nesting-cap is clean (no new >2-level nested version guards).

- [ ] **Step 4: Final deterministic audit sanity (no live IDA needed)**

Re-run the Phase 3 audit over the committed exports and confirm no diff:
```bash
# re-run Phase 3 Task 3.1 Step 2, then:
git status --short docs/packets/audits/   # expect clean
```

### Task 8.2: Code review before PR

- [ ] **Step 1: Dispatch the reviewers**

Invoke `superpowers:requesting-code-review` (Go files changed → `plan-adherence-reviewer`
+ `backend-guidelines-reviewer`). No frontend files change.

- [ ] **Step 2: Address findings; write the audit**

Findings written to `docs/tasks/task-081-ida-export-reharvest/audit.md`. Resolve
blockers; re-run the relevant gate after fixes.

**Phase 8 GATE:** all CLAUDE.md gates green; code review run; `audit.md` present;
closed task-080 items still green. Ready for PR.

---

## Self-review notes (coverage map: FR → task)

- FR-1.1 struct descent → Tasks 1.3, 1.8, 1.9
- FR-1.2 no truncation → Tasks 1.3, 1.9 (trailing `inShop` assertion)
- FR-1.3 loop-vs-struct → Task 1.3 (count_loop vs struct_helper)
- FR-1.4 field semantics/width → Tasks 1.2–1.4 (op width load-bearing; labels best-effort)
- FR-1.5 determinism/provenance → Task 1.10
- FR-1.6 bounded recursion → Tasks 1.5 (depth→unresolved), 1.8 (cycle/depth guard)
- FR-2.1/2.2 full re-export + reviewable diff → Tasks 2.1, 2.2
- FR-2.3 unresolved markers → Tasks 1.1, 1.5, 1.11
- FR-3.1/3.2 re-audit + reconcile → Tasks 3.1, 3.2
- FR-3.3 fix with byte tests + gates → Task 4.x
- FR-3.4 too-large → follow-up → Task 4.Z
- FR-4.1/4.2/4.3 opaque decomposition → Tasks 5.0, 5.x
- FR-5.1/5.2/5.3 template completeness → Tasks 6.0, 6.x
- FR-6.1/6.2 ledger + guide → Tasks 7.1, 7.2
- FR-7.1 snapshot → Task 0.2; FR-7.2 delta → Task 3.2 Step 1; FR-7.3/7.4/7.5 disposition → Task 3.2 Step 2 + Phase 3 GATE
- NFR verify gates → Phase 8
