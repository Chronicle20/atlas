# Systematic Off-By-One Divergent Remediation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `diff-shape` diagnostic that surfaces hand-vs-live read lists for divergent entries, then use it to characterize and remediate the ~175 systematic off-by-one divergences (dispatcher annotation / baseline correction / flag-as-real) — without changing `ValidateShape`.

**Architecture:** A new read-only `diff-shape` subcommand mirrors `validate`'s per-address loop but, for divergent entries, emits hand vs live read sequences side by side with the differing position classified leading/trailing/interior. Remediation is data (dispatcher annotations / baseline `calls` corrections via the existing lossless surgical writer) plus, only when a shared prefix is found, one new `dispatcherPrefix` kind. `ValidateShape` is untouched (option 3 — no byte-equivalence absorption).

**Tech Stack:** Go (`tools/packet-audit`), table-driven tests with the existing `validateFakeMCP` fake and hand-crafted `.c` fixtures.

**Design:** `divergent-offbyone-design.md`. **Context (file/symbol map, ports, gates):** `per-branch-verification-context.md`.

---

## File Structure

- **Create** `cmd/diff_shape.go` — the `diff-shape` subcommand (diagnostic only, no verdict change). (Task 1, 2)
- **Modify** `cmd/root.go` — dispatch `diff-shape`. (Task 2)
- **Modify** `internal/idasrc/export.go` — a new `dispatcherPrefix` kind, ONLY if characterization finds a shared prefix. (Task 5, conditional)
- **Docs:** `divergent-characterization.md`, `divergent-findings.md` (Tasks 4, 5, 6).

Run all `go`/`git` from `tools/packet-audit/`. After each commit: `git rev-parse --abbrev-ref HEAD`
→ `task-081-ida-export-reharvest`. Gate every code task with
`go test -race ./... && go vet ./... && go build ./...` (tool, not a service → no docker bake / no redis).

**IDA-gated tasks (need MCP ports 13337–13340): Tasks 4, 5 (live parts), 6. If the MCP is busy, do Tasks 1–3 offline and defer.** Tasks 1–3 build the diagnostic and run fully offline with the fake.

---

## Task 1: diff-shape core — read-list diff classifier

**Files:**
- Create: `cmd/diff_shape.go`
- Test: `cmd/diff_shape_test.go`

A pure function `classifyDiff(hand, live []idasrc.FieldCall) shapeDiff` that locates where two read
lists differ via longest-common-prefix (LCP) and longest-common-suffix (LCS), and classifies the
divergence position. Op identity for prefix/suffix matching uses `Primitive.RawOp()` string equality
(EXACT — this is a diagnostic, NOT the width-tolerant verdict; we want to see the literal shapes).

- [ ] **Step 1: Write the failing test**

Create `cmd/diff_shape_test.go`:

```go
package cmd

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

func fc(ops ...idasrc.Primitive) []idasrc.FieldCall {
	out := make([]idasrc.FieldCall, len(ops))
	for i, o := range ops {
		out[i] = idasrc.FieldCall{Op: o}
	}
	return out
}

func TestClassifyDiff(t *testing.T) {
	cases := []struct {
		name       string
		hand, live []idasrc.FieldCall
		wantPos    string // "leading" | "trailing" | "interior" | "none"
		wantDelta  int    // len(live) - len(hand)
	}{
		{"leading-extra-live", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode1, idasrc.Decode4, idasrc.Decode2), "leading", 1},
		{"trailing-extra-live", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode2, idasrc.Decode1), "trailing", 1},
		{"interior", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode1, idasrc.Decode2), "interior", 1},
		{"identical", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode2), "none", 0},
	}
	for _, tc := range cases {
		d := classifyDiff(tc.hand, tc.live)
		if d.position != tc.wantPos || d.delta != tc.wantDelta {
			t.Errorf("%s: got {pos:%q delta:%d}, want {pos:%q delta:%d}",
				tc.name, d.position, d.delta, tc.wantPos, tc.wantDelta)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/ -run TestClassifyDiff -v`
Expected: FAIL — `classifyDiff` / `shapeDiff` undefined.

- [ ] **Step 3: Implement**

Create `cmd/diff_shape.go`:

```go
package cmd

import (
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// shapeDiff describes where a hand read-list and a live read-list diverge.
// position is "none" (identical), "leading", "trailing", or "interior" depending
// on where the shared prefix/suffix leaves the differing span. delta is
// len(live)-len(hand). prefix/suffix are the matched-on-both lengths.
type shapeDiff struct {
	position string
	delta    int
	prefix   int
	suffix   int
}

// classifyDiff locates the divergence between two read lists via longest common
// prefix + longest common suffix (by exact op identity). Diagnostic only — it
// never affects a verdict. A list whose shorter side is fully covered by the
// shared prefix+suffix yields "leading"/"trailing" by which side the extra reads
// sit on; otherwise "interior".
func classifyDiff(hand, live []idasrc.FieldCall) shapeDiff {
	d := shapeDiff{delta: len(live) - len(hand)}
	if eqOps(hand, live) {
		d.position = "none"
		return d
	}
	n := min2(len(hand), len(live))
	p := 0
	for p < n && hand[p].Op == live[p].Op {
		p++
	}
	s := 0
	for s < n-p && hand[len(hand)-1-s].Op == live[len(live)-1-s].Op {
		s++
	}
	d.prefix, d.suffix = p, s
	switch {
	case p == 0 && s > 0:
		d.position = "leading"
	case s == 0 && p > 0:
		d.position = "trailing"
	case p+s >= n:
		// the shorter list is entirely shared prefix+suffix; the extra reads sit
		// on one side — leading if the prefix ran to the shorter end, else trailing.
		if p >= len(hand) || p >= len(live) {
			d.position = "trailing"
		} else {
			d.position = "leading"
		}
	default:
		d.position = "interior"
	}
	return d
}

func eqOps(a, b []idasrc.FieldCall) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Op != b[i].Op {
			return false
		}
	}
	return true
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./cmd/ -run TestClassifyDiff -v`
Expected: PASS

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add cmd/diff_shape.go cmd/diff_shape_test.go
git commit -m "feat(task-081): diff-shape read-list divergence classifier"
git rev-parse --abbrev-ref HEAD
```

---

## Task 2: diff-shape driver + subcommand wiring

**Files:**
- Modify: `cmd/diff_shape.go`
- Modify: `cmd/root.go`
- Test: `cmd/diff_shape_test.go`, `cmd/testdata/diffshape_mini.json`

The driver mirrors `validate`'s loop (`cmd/validate.go`): group entries by address, `ResolveLive`
once per address, `ExtractShape` per entry, run `ValidateShape` to find DIVERGENT entries, and for
each emit a side-by-side row using `classifyDiff`. It NEVER writes a verdict — output is a report
of divergent entries only.

- [ ] **Step 1: Write the failing test**

Add to `cmd/diff_shape_test.go`. `fooDecomp` (0x100, a 2-case switch) already exists in
`validate_test.go` (same package). Create a baseline whose `#B` hand shape is one read short of
case 2's live shape so it is divergent:

```go
func TestDiffShapeRun_EmitsDivergentRows(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": fooDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "d.md")
	code := diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	// Foo::OnBar#Short is divergent (hand 1 vs live 2) and must appear with a delta + position.
	if !strings.Contains(s, "Foo::OnBar#Short") {
		t.Fatalf("divergent entry missing from diff-shape report:\n%s", s)
	}
	if !strings.Contains(s, "delta") {
		t.Fatalf("report missing delta annotation:\n%s", s)
	}
	// A verified entry (#A) must NOT appear (report is divergent-only).
	if strings.Contains(s, "Foo::OnBar#A") {
		t.Fatalf("verified entry wrongly included:\n%s", s)
	}
}
```

Create `cmd/testdata/diffshape_mini.json` (case 1 reads Decode4; #A matches → verified; case 2
reads Decode2 but #Short hand is empty-after-header → divergent):

```json
{
 "binary":"x","md5":"x","generated_at":"t",
 "functions":{
  "Foo::OnBar#A":{"address":"0x100","direction":"clientbound",
    "dispatch":[{"discriminator":"switch","case":1}],
    "calls":[{"op":"Decode4","guard":""}]},
  "Foo::OnBar#Short":{"address":"0x100","direction":"clientbound",
    "dispatch":[{"discriminator":"switch","case":2}],
    "calls":[{"op":"Decode4","guard":""}]}
 }
}
```

(For case 2 the live shape is `[Decode2]`; `#Short` hand is `[Decode4]` → `ValidateShape` returns
divergent. Op identity differs so `classifyDiff` reports an interior/positional diff with delta 0.
Adjust the fixture if you want a length delta — the test only asserts presence + "delta".)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/ -run TestDiffShapeRun -v`
Expected: FAIL — `diffShapeOpts` / `diffShapeRun` undefined.

- [ ] **Step 3: Implement the driver**

Append to `cmd/diff_shape.go`:

```go
import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type diffShapeOpts struct {
	Baseline     string
	Report       string
	DescentDepth int
}

// diffShapeRun emits a side-by-side hand-vs-live read-list report for every
// DIVERGENT baseline entry, with the divergence position classified. It is a
// pure diagnostic: it loads the baseline, resolves each base live once, extracts
// each entry's shape, and reports — it NEVER mutates the baseline or a verdict.
func diffShapeRun(opts diffShapeOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "diff-shape: load baseline:", err)
		return 3
	}
	entries := src.Entries()

	byAddr := map[string][]int{}
	var addrOrder []string
	for i := range entries {
		a := entries[i].Address
		if _, ok := byAddr[a]; !ok {
			addrOrder = append(addrOrder, a)
		}
		byAddr[a] = append(byAddr[a], i)
	}
	sort.Strings(addrOrder)

	ctx := context.Background()
	type row struct {
		fname string
		diff  shapeDiff
		hand  []idasrc.FieldCall
		live  []idasrc.FieldCall
	}
	var rows []row

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		dir := entries[idxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			continue // diagnostic skips unresolvable bases (validate reports them unverifiable)
		}
		for _, i := range idxs {
			e := entries[i]
			live := idasrc.ExtractShape(f, e.Dispatch)
			verdict, _ := idasrc.ValidateShape(e.HandCalls, live)
			if verdict != idasrc.ShapeDivergent {
				continue
			}
			rows = append(rows, row{fname: e.FName, diff: classifyDiff(e.HandCalls, live), hand: e.HandCalls, live: live})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].fname < rows[j].fname })

	var b strings.Builder
	fmt.Fprintf(&b, "# diff-shape report\n\n%d divergent entries\n\n", len(rows))
	for _, r := range rows {
		fmt.Fprintf(&b, "## %s — %s (delta %+d, prefix %d, suffix %d)\n",
			r.fname, r.diff.position, r.diff.delta, r.diff.prefix, r.diff.suffix)
		fmt.Fprintf(&b, "- hand: %s\n", opsLine(r.hand))
		fmt.Fprintf(&b, "- live: %s\n\n", opsLine(r.live))
	}

	if dir := filepath.Dir(opts.Report); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "diff-shape: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Report, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "diff-shape: write report:", err)
		return 3
	}
	fmt.Fprintf(stdout, "diff-shape: %d divergent entries\n", len(rows))
	return 0
}

func opsLine(cs []idasrc.FieldCall) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.Op.RawOp()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
```

- [ ] **Step 4: Wire the subcommand in `cmd/root.go`**

Add the dispatch line next to the others in `Run`:

```go
	if len(args) > 0 && args[0] == "diff-shape" {
		return runDiffShape(args[1:], stderr)
	}
```

Add `runDiffShape` (copy `runResolveDispatch`'s flag/client wiring at `cmd/root.go:228`; flags:
`--version`, `--baseline` default `docs/packets/ida-exports/<version>.json`, `--report` (required),
`--ida-url`, `--ida-timeout`, `--ida-port`, `--descent-depth`). Delegate to `diffShapeRun`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./cmd/ -run TestDiffShapeRun -v`
Expected: PASS

- [ ] **Step 6: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add cmd/diff_shape.go cmd/root.go cmd/diff_shape_test.go cmd/testdata/diffshape_mini.json
git commit -m "feat(task-081): diff-shape subcommand (divergent hand-vs-live diagnostic)"
git rev-parse --abbrev-ref HEAD
```

---

## Task 3: diff-shape determinism + verdict-safety guard

**Files:**
- Test: `cmd/diff_shape_test.go`

Two safety properties the design demands: the report is byte-stable, and diff-shape never mutates
the baseline.

- [ ] **Step 1: Write the test**

```go
func TestDiffShape_DeterministicAndReadOnly(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": fooDecomp}}
	dir := t.TempDir()
	r1 := filepath.Join(dir, "a.md")
	r2 := filepath.Join(dir, "b.md")
	before, _ := os.ReadFile("testdata/diffshape_mini.json")
	_ = diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: r1, DescentDepth: 4}, fc, io.Discard)
	_ = diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: r2, DescentDepth: 4}, fc, io.Discard)
	a, _ := os.ReadFile(r1)
	b, _ := os.ReadFile(r2)
	if string(a) != string(b) {
		t.Fatal("diff-shape report not deterministic")
	}
	after, _ := os.ReadFile("testdata/diffshape_mini.json")
	if string(before) != string(after) {
		t.Fatal("diff-shape mutated the baseline")
	}
}
```

- [ ] **Step 2: Run + commit**

Run: `go test ./cmd/ -run TestDiffShape -v` → PASS.

```bash
go test -race ./... && go vet ./... && go build ./...
git add cmd/diff_shape_test.go
git commit -m "test(task-081): diff-shape determinism + read-only guards"
git rev-parse --abbrev-ref HEAD
```

---

## Task 4: Live characterization — IDA-GATED

**Purpose:** find the shared cause(s) of the off-by-one cluster. **Defer if the MCP is busy.**

- [ ] **Step 1: Build + run diff-shape on all four**

```bash
go build -o /tmp/packet-audit .
mkdir -p /tmp/ds
/tmp/packet-audit diff-shape --version gms_v83 --ida-port 13337 --report /tmp/ds/gms_v83.md
/tmp/packet-audit diff-shape --version gms_v87 --ida-port 13338 --report /tmp/ds/gms_v87.md
/tmp/packet-audit diff-shape --version gms_v95 --ida-port 13339 --report /tmp/ds/gms_v95.md
/tmp/packet-audit diff-shape --version gms_jms_185 --ida-port 13340 --report /tmp/ds/gms_jms_185.md
```

(Run each as its own command — zsh does not word-split unquoted vars, so avoid `set -- $pair` loops.)

- [ ] **Step 2: Cluster the off-by-one and identify shared causes**

From the reports, group `delta ±1` rows by base-handler family and by `position`
(leading/trailing/interior). Confirm the `CCashShop` family's shared extra read (expected: a
`leading` op — the wrapper/action byte) and look for other families with a consistent
leading/trailing op. Record the exact op and position per cluster.

- [ ] **Step 3: Write the characterization**

Create `docs/tasks/task-081-ida-export-reharvest/divergent-characterization.md`: per cluster, the
shared extra read (op + leading/trailing/interior), the count, and the remediation category
(shared-prefix → dispatcher; one-off → baseline correction; genuine → flag). Commit.

```bash
git add docs/tasks/task-081-ida-export-reharvest/divergent-characterization.md
git commit -m "docs(task-081): characterize the off-by-one divergent cluster"
git rev-parse --abbrev-ref HEAD
```

---

## Task 5: Remediation — IDA-gated / data + conditional code

**Purpose:** apply the approved blend, driven by Task 4's findings. **Defer the live re-verify if busy.**

- [ ] **Step 1: Shared-prefix → dispatcher annotation (conditional code)**

If a cluster shares a leading wrapper read that matches an existing `dispatcherPrefix` kind
(`internal/idasrc/export.go`: `per-mob` / `per-pet` / `per-pet-remote` / `per-user-remote`),
annotate each affected baseline entry with `"dispatcher": "<kind>"` (surgical edit; the field is
already in the schema). If the shared read is a NEW prefix (e.g. a cash-shop action `Decode1`), add
a new kind to `dispatcherPrefix` first, with a test:

```go
// in export_test.go, mirroring the existing per-mob/per-pet cases:
func TestDispatcherPrefix_CashShop(t *testing.T) {
	got := dispatcherPrefix("cash-shop")
	if len(got) != 1 || got[0].Op != Decode1 {
		t.Fatalf("cash-shop prefix = %+v, want one Decode1", got)
	}
}
```

```go
// in dispatcherPrefix's switch (export.go):
case "cash-shop":
	return []FieldCall{
		{Op: Decode1, Comment: "cash-shop action byte — auto-prepended via dispatcher: cash-shop"},
	}
```

Then add `"dispatcher": "cash-shop"` to the affected `CCashShop` baseline entries (and document the
exact op/kind that Task 4 actually found — adjust `Decode1` if the real prefix differs). Gate +
commit:

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/export.go internal/idasrc/export_test.go docs/packets/ida-exports/
git commit -m "feat(task-081): dispatcher prefix annotation for shared off-by-one cluster"
git rev-parse --abbrev-ref HEAD
```

- [ ] **Step 2: One-off omissions → baseline `calls` correction**

For entries whose extra read is a genuine field the hand trace missed (not a shared prefix), add the
missing read to that entry's `calls` in `docs/packets/ida-exports/<version>.json` (surgical edit;
keep the file's 1-space + Python-escaping style — the lossless writer's format). One commit per
coherent batch:

```bash
git add docs/packets/ida-exports/
git commit -m "fix(task-081): correct hand baselines for one-off off-by-one omissions"
git rev-parse --abbrev-ref HEAD
```

- [ ] **Step 3: Genuine Atlas-vs-client differences → flag, do NOT fix**

For off-by-one entries that are a real difference between what the client reads and what Atlas
writes, record each in `docs/tasks/task-081-ida-export-reharvest/divergent-findings.md` (handler,
the missing/extra field, which side has it) as encoder work. Do not edit baselines or encoders here.
Commit the findings doc.

```bash
git add docs/tasks/task-081-ida-export-reharvest/divergent-findings.md
git commit -m "docs(task-081): flag genuine Atlas-vs-client off-by-one findings"
git rev-parse --abbrev-ref HEAD
```

---

## Task 6: Re-validate + results — IDA-GATED

**Purpose:** measure the divergent reduction. **Defer if the MCP is busy.**

- [ ] **Step 1: Re-validate all four**

```bash
/tmp/packet-audit validate --version gms_v83 --ida-port 13337 --report /tmp/ds/gms_v83.val.md
/tmp/packet-audit validate --version gms_v87 --ida-port 13338 --report /tmp/ds/gms_v87.val.md
/tmp/packet-audit validate --version gms_v95 --ida-port 13339 --report /tmp/ds/gms_v95.val.md
/tmp/packet-audit validate --version gms_jms_185 --ida-port 13340 --report /tmp/ds/gms_jms_185.val.md
grep -H 'verified' /tmp/ds/*.val.md
```

Expected: divergent drops by the remediated shared-prefix + one-off counts; flagged genuine diffs
remain divergent; width/opaque/loop diffs remain divergent (by design). The jms audit-dir is
`jms_v185` (the allowlist default path quirk) — irrelevant here since validate's allowlist only
affects missing-mode.

- [ ] **Step 2: Write results + code review**

Create `docs/tasks/task-081-ida-export-reharvest/divergent-offbyone-results.md` with the
before→after divergent counts and the remediation breakdown (dispatcher / baseline / flagged).
Commit. Then per CLAUDE.md run `superpowers:requesting-code-review` (backend-guidelines +
plan-adherence) before any PR.

---

## Self-Review (completed by plan author)

- **Spec coverage:** design component 1 (diagnostic)→Tasks 1–3; component 2 (characterization)→Task 4;
  component 3 (remediation blend)→Task 5 (dispatcher / baseline / flag); component 4 (re-validate)→Task 6.
  All covered. `ValidateShape` is never modified — no task touches it (option 3 honored).
- **Type consistency:** `shapeDiff{position,delta,prefix,suffix}` + `classifyDiff` (Task 1) consumed by
  `diffShapeRun` (Task 2); `diffShapeOpts{Baseline,Report,DescentDepth}` and `diffShapeRun` names stable
  across Tasks 2–3; `opsLine`/`Primitive.RawOp()` used consistently; `dispatcherPrefix` kind addition
  (Task 5) matches the existing switch signature in `export.go`.
- **Placeholder scan:** the conditional bits in Task 5 (new dispatcher kind / which op) are explicitly
  parameterized on Task 4's live findings — concrete procedures with example code, not TODOs. Tasks 1–3
  are fully concrete and offline. IDA-gated steps are flagged for deferral.
- **Risk note:** Task 5's exact remediation depends on Task 4's characterization; if the off-by-one is
  NOT systematic (no shared prefix), Steps 1 collapses and the lever yields little — the design already
  flags this contingency honestly.
