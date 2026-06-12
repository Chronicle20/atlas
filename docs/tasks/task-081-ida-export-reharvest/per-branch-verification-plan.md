# Per-Branch Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `packet-audit` positively verify each *branch* of a dispatching client handler against its matching Atlas `#Mode` writer, collapsing the ~450 "per-mode shape not extractable" unverifiable entries and reporting client cases with no Atlas writer (missing-mode) and Atlas modes with no client case (extra-mode).

**Architecture:** Extend the existing flat read-order model with (a) `if/else` guard emission in `ParseDecompile` mirroring the switch state machine, (b) a default/else `Selector`, (c) full case-label-set enumeration on `Fields`, then add a stateful `resolve-dispatch` subcommand (inference + agent-confirmation, writes selectors into the committed baseline) and bijection buckets in `validate`, gated by a per-version allowlist.

**Tech Stack:** Go (`tools/packet-audit`), IDA-MCP over HTTP (`internal/idasrc`), `flag`-based subcommands, table-driven tests with real Hex-Rays `.c` fixtures.

**Context doc:** `per-branch-verification-context.md` (read it first — file/symbol map, ports, gates).
**Design doc:** `per-branch-verification-design.md`.

---

## File Structure

- **Modify** `internal/idasrc/extract.go` — add `Selector.Default`; teach matcher the default arm. (Task 1)
- **Create** `internal/idasrc/testdata/ifelse_*.c`, `default_arm.c` — real harvested fixtures. (Task 2)
- **Modify** `internal/idasrc/parse.go` — if/else dispatch guard emission + default-arm guard token. (Task 2)
- **Modify** `internal/idasrc/parse.go` + `idasrc.go` — `Fields.CaseLabels` enumeration. (Task 3)
- **Create** `internal/idasrc/baseline_write.go` + test — typed load/save that persists `dispatch`+`notes`. (Task 4)
- **Create** `cmd/resolve_dispatch.go` + test, **modify** `cmd/root.go` — `resolve-dispatch` subcommand. (Task 5)
- **Modify** `cmd/validate.go` + `internal/idasrc/bijection.go` (new) + tests — missing/extra-mode buckets. (Task 6)
- **Create** `internal/idasrc/allowlist.go` + test, allowlist files `docs/packets/audits/<v>/_unimplemented.json`. (Task 7)
- **End-to-end** re-run on four IDBs; record results. (Task 8)

Each task ends green (`go test -race ./... && go vet ./... && go build ./...` from `tools/packet-audit/`) and is committed. Run all `git`/`go` commands from `tools/packet-audit/` unless stated. Verify the branch after each commit: `git rev-parse --abbrev-ref HEAD` must print `task-081-ida-export-reharvest`.

---

## Task 1: `Selector.Default` + default-arm extraction

**Files:**
- Modify: `internal/idasrc/extract.go`
- Test: `internal/idasrc/extract_test.go`

Adds a default/else selector. The parser (Task 2) will tag default-arm reads with a fixed guard token `<default>`; a `Selector{Default:true}` matches reads carrying that token (and only those). A non-default selector must NOT match a `<default>`-guarded read, and a default selector must NOT match a normal `disc == N` read.

- [ ] **Step 1: Write the failing test**

Add to `internal/idasrc/extract_test.go`:

```go
func TestExtractShape_DefaultArm(t *testing.T) {
	// disc==N reads plus a trailing default-arm read.
	f := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},               // common header (pre-branch)
		{Op: Decode2, Guard: "mode == 1"},      // case 1
		{Op: Decode4, Guard: "mode == 2"},      // case 2
		{Op: DecodeStr, Guard: "<default>"},    // else/default arm
	}}

	// Default selector: header + the default-arm read only.
	got := ExtractShape(f, []Selector{{Discriminator: "mode", Default: true}})
	wantOps := []Primitive{Decode1, DecodeStr}
	if len(got) != len(wantOps) {
		t.Fatalf("default: got %d reads, want %d (%v)", len(got), len(wantOps), got)
	}
	for i := range wantOps {
		if got[i].Op != wantOps[i] {
			t.Fatalf("default[%d]=%s want %s", i, got[i].Op, wantOps[i])
		}
	}

	// A concrete case selector must NOT pick up the default-arm read.
	got2 := ExtractShape(f, []Selector{{Discriminator: "mode", Case: 1}})
	for _, c := range got2 {
		if c.Guard == "<default>" {
			t.Fatalf("case selector wrongly matched default-arm read: %v", got2)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/idasrc/ -run TestExtractShape_DefaultArm -v`
Expected: FAIL — `Selector` has no `Default` field (compile error).

- [ ] **Step 3: Implement**

In `extract.go`, add the field and a default-arm constant, and extend matching. Replace the `Selector` struct and `clauseMatches`:

```go
// DefaultGuardToken is the guard text the parser stamps on a switch `default:` /
// trailing `else` arm's reads. A Selector{Default:true} matches exactly these,
// and a normal case Selector never matches them.
const DefaultGuardToken = "<default>"

type Selector struct {
	Discriminator string `json:"discriminator,omitempty"` // "" matches any discriminator
	Case          int64  `json:"case"`
	Default       bool   `json:"default,omitempty"` // matches the default/else arm
}
```

In `clauseMatches`, handle the default selector and reject default-guarded reads for normal selectors. Replace the function body's start:

```go
func clauseMatches(guard string, sel Selector) bool {
	if sel.Default {
		// A default selector matches iff the guard is exactly the default token.
		return strings.TrimSpace(guard) == DefaultGuardToken
	}
	if strings.TrimSpace(guard) == DefaultGuardToken {
		// A normal case selector never matches a default-arm read.
		return false
	}
	for _, clause := range strings.Split(guard, "&&") {
		// ... unchanged existing clause-matching loop ...
```

Note: `guardSatisfies` already requires every selector to match; the pre-branch empty-guard prefix logic in `ExtractShape` is unchanged and still prepends the common header.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/idasrc/ -run TestExtractShape_DefaultArm -v`
Expected: PASS

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/extract.go internal/idasrc/extract_test.go
git commit -m "feat(task-081): Selector.Default + default-arm extraction"
git rev-parse --abbrev-ref HEAD   # must be task-081-ida-export-reharvest
```

---

## Task 2: if/else dispatch guard emission in `ParseDecompile`

**Files:**
- Create: `internal/idasrc/testdata/ifelse_chain.c`, `internal/idasrc/testdata/ifelse_else.c`
- Modify: `internal/idasrc/parse.go`
- Test: `internal/idasrc/parse_test.go`

The riskiest component. The Phase-1.5 lesson (see `project_packet_audit_exporter_real_decompile_gaps`): **fixtures MUST be real Hex-Rays text**, not synthesized. So Step 1 harvests real decompile from a live IDB; Steps derive the expected read-order from the *captured* text.

### Harvest the real fixtures (do this first, record the truth)

- [ ] **Step 1: Capture two real if/else-dispatched handlers**

The canonical if/else dispatcher is `CLogin::OnCheckPasswordResult` (dispatches on a result
code via `if/else if/... else`). From the v83 baseline, get its address:

```bash
python3 -c "import json; d=json.load(open('docs/packets/ida-exports/gms_v83.json'))['functions']; \
print({k:v['address'] for k,v in d.items() if k.startswith('CLogin::OnCheckPasswordResult')})"
```

Decompile that base address on the v83 IDB (port 13337) via the IDA-MCP `decompile` tool
(the same server `--ida-url http://192.168.20.3:13337/mcp` uses; call `select_instance` port
13337 then `decompile` the address). Save the **verbatim** decompile text to
`internal/idasrc/testdata/ifelse_chain.c`. Pick a second handler that has a trailing bare
`else` arm (no condition) and save it to `internal/idasrc/testdata/ifelse_else.c`. If
`OnCheckPasswordResult` has a trailing `else`, one fixture can cover both — still save two
files capturing (a) an if/else-if chain and (b) a chain ending in bare `else`.

Then **read the captured `.c` files** and write down, per file, the true ordered reads and
which `if (x == N)` arm each read sits under. This recorded read-order is the test oracle for
Step 2 — do not invent it; transcribe it from the fixture.

> Illustrative shape (what the captured text looks like — replace expected values with the
> real transcription):
> ```c
> result = CInPacket::Decode1(a2);            // result code  -> guard ""  (pre-branch)
> if ( result == 2 ) { CInPacket::Decode1(a2); /*ban kind*/ CInPacket::Decode8(a2); /*until*/ }
> else if ( result == 5 ) { /* no reads */ }
> else { CInPacket::Decode2(a2); /*generic*/ }
> ```

### Now TDD the parser change

- [ ] **Step 2: Write the failing test (expected values transcribed from the fixtures)**

Add to `internal/idasrc/parse_test.go`. Fill `wantGuards`/`wantOps` from the Step-1
transcription of the REAL fixture (the values below match the illustrative shape; correct them
to the captured text):

```go
func TestParseDecompile_IfElseDispatch(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_chain.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	// Each guarded read must carry "<disc> == <N>" for the arm it sits under;
	// the pre-branch discriminator read carries "".  (Transcribe from fixture.)
	type rc struct{ op, guard string }
	var got []rc
	for _, c := range calls {
		if c.Op == Decode1 || c.Op == Decode2 || c.Op == Decode4 ||
			c.Op == Decode8 || c.Op == DecodeStr || c.Op == DecodeBuf {
			got = append(got, rc{c.Op.RawOp(), c.Guard})
		}
	}
	want := []rc{
		{"Decode1", ""},               // result code, pre-branch
		{"Decode1", "result == 2"},    // arm result==2
		{"Decode8", "result == 2"},
		{"Decode2", "<default>"},      // bare else -> default token
	}
	if len(got) != len(want) {
		t.Fatalf("got %d guarded reads, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("read[%d]=%+v want %+v", i, got[i], want[i])
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseDecompile_IfElseDispatch -v`
Expected: FAIL — guards are empty (parser ignores if/else), so the `want` guards don't match.

- [ ] **Step 4: Implement if/else tracking in `ParseDecompile`**

Mirror the switch state machine. Add regexes near `reCase`:

```go
// reIfEq matches an if/else-if header that dispatches on an equality against a
// constant: `if ( x == 5 )`, `else if ( v7 == 0xA )`. Captures discriminator and
// constant. The optional leading `else` distinguishes chain continuation.
reIfEq = regexp.MustCompile(`^\s*(?:(else)\s+)?if\s*\(\s*([A-Za-z_]\w*)\s*==\s*(0[xX][0-9A-Fa-f]+|[0-9]+)[uUlL]*\s*\)\s*{?\s*$`)
// reElse matches a bare `else` / `else {` (no condition) — the default arm.
reElse = regexp.MustCompile(`^\s*else\s*{?\s*$`)
```

Add an if/else scope tracker alongside `switches`. An if/else *chain* shares one
discriminator; each arm is a scope on `stack` keyed by brace depth. Use this state:

```go
// ifChain tracks one active if/else-if/else dispatch chain. discrim is the shared
// equality variable; armIdx is the index into `stack` of the current arm's scope
// (-1 when between arms / chain closed). seen guards the "same discriminator"
// invariant: an `if (y == ...)` with y != discrim starts a NEW chain, not an arm.
type ifChainEntry struct {
	bodyDepth int
	discrim   string
	armIdx    int
}
var ifChains []ifChainEntry
pendingArmFrag := "" // guard fragment for an arm whose body brace opens next
```

In the per-line loop, BEFORE the `opensBrace` handling and AFTER the switch/case block, add
arm detection (an arm header stages its fragment to bind to the body brace, exactly like
`pendingSwitchVar`):

```go
// if/else-if dispatch arm.
if m := reIfEq.FindStringSubmatch(line); m != nil {
	isElse, disc, lit := m[1] == "else", m[2], m[3]
	if !isElse {
		// `if (...)` starts a fresh chain (close any arm of the innermost chain).
		pendingArmFrag = disc + " == " + lit
		ifChains = append(ifChains, ifChainEntry{bodyDepth: braceDepth + 1, discrim: disc, armIdx: -1})
	} else if len(ifChains) > 0 && ifChains[len(ifChains)-1].discrim == disc {
		// `else if (disc == lit)` continues the innermost chain on the same disc.
		clearActiveArm()
		pendingArmFrag = disc + " == " + lit
	} else {
		// `else if (other == lit)` — discriminator switch: treat as a new chain.
		pendingArmFrag = disc + " == " + lit
		ifChains = append(ifChains, ifChainEntry{bodyDepth: braceDepth + 1, discrim: disc, armIdx: -1})
	}
} else if reElse.MatchString(line) && len(ifChains) > 0 {
	// bare `else` — the default arm of the innermost chain.
	clearActiveArm()
	pendingArmFrag = DefaultGuardToken
}
```

Bind a pending arm to the body brace (next to the pending-switch binding):

```go
if pendingArmFrag != "" && opensBrace {
	ic := &ifChains[len(ifChains)-1]
	stack = append(stack, scope{depth: braceDepth + 1, frag: pendingArmFrag})
	ic.armIdx = len(stack) - 1
	pendingArmFrag = ""
}
```

Add `clearActiveArm` (mirror `clearActiveCase`) and pop chains on brace-exit (mirror the
`switches` pop loop):

```go
clearActiveArm := func() {
	if len(ifChains) == 0 {
		return
	}
	ic := &ifChains[len(ifChains)-1]
	if ic.armIdx >= 0 && ic.armIdx == len(stack)-1 {
		stack = stack[:ic.armIdx]
	}
	ic.armIdx = -1
}
```

And in the brace-exit popping section, after the `switches` pop:

```go
for len(ifChains) > 0 && braceDepth < ifChains[len(ifChains)-1].bodyDepth {
	ifChains = ifChains[:len(ifChains)-1]
}
if len(ifChains) > 0 {
	ic := &ifChains[len(ifChains)-1]
	if ic.armIdx >= len(stack) {
		ic.armIdx = -1
	}
}
```

**Bail rule (design):** only `x == const` arms get a guard. An arm whose condition is a range,
inequality, compound, or non-discriminator predicate matches neither `reIfEq` nor `reElse`, so
its reads carry whatever outer guard is active (no fabricated `==`). Those entries stay honestly
unverifiable rather than mis-extracted — that is the intended behavior, not a bug.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/idasrc/ -run TestParseDecompile_IfElseDispatch -v`
Expected: PASS. Then run the bare-`else` fixture coverage:

```go
func TestParseDecompile_IfElseTrailingElse(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_else.c"), DirClientbound)
	if err != nil { t.Fatal(err) }
	sawDefault := false
	for _, c := range calls {
		if c.Guard == DefaultGuardToken { sawDefault = true }
	}
	if !sawDefault {
		t.Fatalf("expected a <default> guard on the bare-else arm; got %+v", calls)
	}
}
```

Run: `go test ./internal/idasrc/ -run TestParseDecompile_IfElse -v`
Expected: PASS (both).

- [ ] **Step 6: Regression — switch fixtures still pass**

Run: `go test ./internal/idasrc/ -run TestParseDecompile -v`
Expected: PASS for ALL existing `mode_switch`, `loop_in_case`, `switch_usuffix`,
`real_onfriendresult_v83`, etc. If a switch test broke, the if/else block is matching switch
lines — tighten `reIfEq`/`reElse` (they must not match `case`/`switch` lines).

- [ ] **Step 7: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/parse.go internal/idasrc/parse_test.go internal/idasrc/testdata/ifelse_chain.c internal/idasrc/testdata/ifelse_else.c
git commit -m "feat(task-081): if/else dispatch guard emission (real-Hex-Rays fixtures)"
git rev-parse --abbrev-ref HEAD
```

---

## Task 3: full case-label-set enumeration on `Fields`

**Files:**
- Modify: `internal/idasrc/idasrc.go` (add field), `internal/idasrc/parse.go` (populate)
- Test: `internal/idasrc/parse_test.go`

For bijection (Task 6) the tool must know EVERY dispatch case the client has, including cases
that read nothing. `enumerateCases` (infer.go) only sees cases that guard a read. Populate a
`CaseLabels` set as the parser walks switch `case N:` / if `== N` headers and `default:`/bare
`else` — independent of whether the arm reads.

- [ ] **Step 1: Write the failing test**

Add to `internal/idasrc/parse_test.go`. Use a fixture with an empty case (no reads) — capture
one, or extend `mode_switch.c` with a `case 3: break;` and save as
`testdata/switch_emptycase.c` (this fixture may be hand-written for the structural case-label
test since it asserts labels, not read fidelity):

```go
func TestParseDecompile_CaseLabelSet(t *testing.T) {
	f, err := ParseDecompileFields(mustFixture(t, "switch_emptycase.c"), DirClientbound)
	if err != nil { t.Fatal(err) }
	got := f.CaseLabels["mode"]            // map[discriminator]*CaseSet
	if got == nil { t.Fatal("no case labels for 'mode'") }
	for _, want := range []int64{1, 2, 3} {
		if !got.Has(want) {
			t.Fatalf("missing case label %d; have %v", want, got.Values())
		}
	}
}
```

`switch_emptycase.c` (structural fixture):

```c
result = CInPacket::Decode1(a2);
switch ( result )
{
  case 1: CInPacket::Decode2(a2); break;
  case 2: CInPacket::Decode4(a2); break;
  case 3: break;
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseDecompile_CaseLabelSet -v`
Expected: FAIL — `ParseDecompileFields`, `Fields.CaseLabels`, `CaseSet` undefined.

- [ ] **Step 3: Implement**

In `idasrc.go`, add the type and field:

```go
// CaseSet is the ordered set of dispatch case labels seen for one discriminator,
// plus whether a default/else arm exists.
type CaseSet struct {
	cases   []int64
	seen    map[int64]bool
	Default bool
}

func (c *CaseSet) add(v int64) {
	if c.seen == nil { c.seen = map[int64]bool{} }
	if !c.seen[v] { c.seen[v] = true; c.cases = append(c.cases, v) }
}
func (c *CaseSet) Has(v int64) bool { return c.seen[v] }
func (c *CaseSet) Values() []int64  { return append([]int64(nil), c.cases...) }
```

Add `CaseLabels map[string]*CaseSet` to `Fields`.

`ParseDecompile` currently returns `[]rawCall`. Add a sibling that also returns the label set
without disturbing existing callers:

```go
// ParseDecompileFields runs ParseDecompile and additionally collects the full
// dispatch case-label set (every switch case / if `== N` arm and default/else),
// independent of whether the arm reads. Existing callers of ParseDecompile are
// unchanged.
func ParseDecompileFields(text string, dir Direction) (Fields, error) {
	calls, err := ParseDecompile(text, dir)
	if err != nil { return Fields{}, err }
	labels := collectCaseLabels(text)
	return Fields{Direction: dir, Calls: toFieldCalls(calls), CaseLabels: labels}, nil
}
```

Implement `collectCaseLabels(text)` as a focused second pass: brace-depth-track the innermost
`switch ( disc )` / if-chain discriminator (reuse `reSwitch`/`reBareVar`/`reCase`/`reIfEq`/
`reElse`), and for each `case N:` / `if (disc == N)` add `N` to `CaseLabels[disc]`, and set
`.Default=true` on a `default:` / bare `else`. (A second pass keeps `ParseDecompile`'s
load-bearing read-emission logic untouched.) Implement `toFieldCalls([]rawCall)` by resolving
each rawCall's op via `parsePrim` (skip `Delegate` — labels don't need descent).

> The case-label collector reuses the SAME header regexes and the SAME brace-depth/switch-nesting
> bookkeeping as `ParseDecompile`; factor the shared scanner skeleton into a small helper if it
> reduces duplication, but do not change `ParseDecompile`'s emitted reads.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/idasrc/ -run TestParseDecompile_CaseLabelSet -v`
Expected: PASS

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/idasrc.go internal/idasrc/parse.go internal/idasrc/parse_test.go internal/idasrc/testdata/switch_emptycase.c
git commit -m "feat(task-081): full dispatch case-label-set enumeration"
git rev-parse --abbrev-ref HEAD
```

---

## Task 4: persisted dispatch writer

**Files:**
- Create: `internal/idasrc/baseline_write.go`, `internal/idasrc/baseline_write_test.go`

`resolve-dispatch` (Task 5) must write confirmed selectors into the committed baseline. The
`exportFile`/`exportFn` types are unexported; add a focused exported writer that loads a
baseline, sets `dispatch` + a provenance `notes` on named entries, and writes it back
deterministically (preserving every other field).

- [ ] **Step 1: Write the failing test**

Create `internal/idasrc/baseline_write_test.go`:

```go
package idasrc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDispatch_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "base.json")
	const in = `{"binary":"x","md5":"y","generated_at":"z","functions":{` +
		`"A::B#One":{"address":"0x1","direction":"clientbound","calls":[{"op":"Decode1","comment":"c"}]}}}`
	if err := os.WriteFile(p, []byte(in), 0o644); err != nil { t.Fatal(err) }

	updates := map[string]DispatchUpdate{
		"A::B#One": {Dispatch: []Selector{{Discriminator: "mode", Case: 9}}, Note: "agent-confirmed @0x1 mode==9"},
	}
	if err := WriteDispatch(p, updates); err != nil { t.Fatal(err) }

	src, err := NewExportSource(p)
	if err != nil { t.Fatal(err) }
	var found *BaselineEntry
	for _, e := range src.Entries() {
		if e.FName == "A::B#One" { ee := e; found = &ee }
	}
	if found == nil { t.Fatal("entry lost after write") }
	if len(found.Dispatch) != 1 || found.Dispatch[0].Case != 9 || found.Dispatch[0].Discriminator != "mode" {
		t.Fatalf("dispatch not persisted: %+v", found.Dispatch)
	}
	// Original calls preserved.
	if len(found.HandCalls) != 1 || found.HandCalls[0].Op != Decode1 {
		t.Fatalf("calls mutated: %+v", found.HandCalls)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestWriteDispatch_RoundTrip -v`
Expected: FAIL — `WriteDispatch`, `DispatchUpdate` undefined.

- [ ] **Step 3: Implement**

Create `internal/idasrc/baseline_write.go`:

```go
package idasrc

import (
	"encoding/json"
	"fmt"
	"os"
)

// DispatchUpdate is one entry's confirmed dispatch selector plus a provenance note
// (e.g. "agent-confirmed @0x.. mode==9" / "inferred-high-confidence").
type DispatchUpdate struct {
	Dispatch []Selector
	Note     string
}

// WriteDispatch loads the baseline at path, sets the dispatch selectors and
// provenance notes for each named function, and writes the file back with a
// deterministic (sorted-key, 2-space) marshal. Unknown FNames are an error (the
// caller's map should only name real entries). Every other field is preserved.
func WriteDispatch(path string, updates map[string]DispatchUpdate) error {
	b, err := os.ReadFile(path)
	if err != nil { return err }
	var ef exportFile
	if err := json.Unmarshal(b, &ef); err != nil { return err }
	for fname, up := range updates {
		fn, ok := ef.Functions[fname]
		if !ok { return fmt.Errorf("idasrc: WriteDispatch: unknown FName %q", fname) }
		fn.Dispatch = up.Dispatch
		if up.Note != "" { fn.Notes = up.Note }
		ef.Functions[fname] = fn
	}
	out, err := json.MarshalIndent(ef, "", "  ")
	if err != nil { return err }
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/idasrc/ -run TestWriteDispatch_RoundTrip -v`
Expected: PASS

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/baseline_write.go internal/idasrc/baseline_write_test.go
git commit -m "feat(task-081): WriteDispatch — persist confirmed selectors into baseline"
git rev-parse --abbrev-ref HEAD
```

---

## Task 5: `resolve-dispatch` subcommand (inference + agent-confirmation gate)

**Files:**
- Create: `cmd/resolve_dispatch.go`, `cmd/resolve_dispatch_test.go`
- Modify: `cmd/root.go` (dispatch the new subcommand)

Runs `InferDispatchJoint` per base handler, **auto-accepts** picks at/above `--min-confidence`,
**writes** them into the baseline via `WriteDispatch`, and emits a markdown + JSON
**confirmation worklist** of the low-confidence picks for the agent to resolve in IDA. Mirrors
`infer`'s structure but is stateful (writes the baseline) and splits high vs low confidence.

- [ ] **Step 1: Write the failing test (core, MCP injected via a fake)**

Existing tests inject a fake `idasrc.MCPClient`. Mirror the infer/validate test setup. Create
`cmd/resolve_dispatch_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDispatch_AutoAcceptsHighConfidence(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	// One base handler @0x10 with two #Mode siblings whose hand shapes map
	// one-to-one onto two distinct, well-separated switch cases (high confidence).
	writeFixtureBaseline(t, base) // helper: see below
	worklist := filepath.Join(dir, "worklist.md")

	opts := resolveDispatchOpts{Baseline: base, Worklist: worklist, MinConfidence: 0.6}
	var out bytes.Buffer
	code := resolveDispatchRun(opts, fakeDispatchClient(t), &out)
	if code != 0 { t.Fatalf("exit %d: %s", code, out.String()) }

	// High-confidence picks were written into the baseline.
	got, _ := os.ReadFile(base)
	if !strings.Contains(string(got), `"dispatch"`) {
		t.Fatalf("expected dispatch written to baseline; got:\n%s", got)
	}
	// No low-confidence entries -> worklist notes "0 to confirm".
	if !strings.Contains(out.String(), "auto-accepted") {
		t.Fatalf("roll-up missing: %s", out.String())
	}
}
```

`writeFixtureBaseline` and `fakeDispatchClient` follow the existing fake-client pattern in
`cmd/infer_test.go` / `cmd/validate_test.go` — copy that scaffolding (a fake whose
`decompile` returns a switch body with two cases). Read those test files and reuse their
helper shape; do not invent a new MCP fake protocol.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/ -run TestResolveDispatch -v`
Expected: FAIL — `resolveDispatchOpts`, `resolveDispatchRun` undefined.

- [ ] **Step 3: Implement the core driver**

Create `cmd/resolve_dispatch.go`:

```go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

type resolveDispatchOpts struct {
	Baseline      string  // baseline export JSON (mutated in place for high-confidence picks)
	Worklist      string  // markdown confirmation worklist output path
	MinConfidence float64 // auto-accept threshold (default 0.6)
	DescentDepth  int
}

// worklistItem is one low-confidence pick the agent must confirm in IDA.
type worklistItem struct {
	FName      string            `json:"fname"`
	Address    string            `json:"address"`
	Proposed   []idasrc.Selector `json:"proposed"`
	Confidence float64           `json:"confidence"`
	Candidates []int64           `json:"candidates,omitempty"`
}

// resolveDispatchRun infers per-base dispatch selectors, auto-accepts high-confidence
// picks (writing them into the baseline), and emits a confirmation worklist (markdown
// + sibling .json) of the low-confidence picks. Deterministic for identical (opts,client).
func resolveDispatchRun(opts resolveDispatchOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: load baseline:", err)
		return 3
	}
	entries := src.Entries()

	byAddr := map[string][]int{}
	var addrOrder []string
	for i := range entries {
		a := entries[i].Address
		if _, ok := byAddr[a]; !ok { addrOrder = append(addrOrder, a) }
		byAddr[a] = append(byAddr[a], i)
	}
	sort.Strings(addrOrder)

	ctx := context.Background()
	accepted := map[string]idasrc.DispatchUpdate{}
	var worklist []worklistItem
	undecompilable := 0

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		// Only #Mode entries need a selector; skip flat entries.
		var modeIdxs []int
		for _, i := range idxs {
			if strings.Contains(entries[i].FName, "#") { modeIdxs = append(modeIdxs, i) }
		}
		if len(modeIdxs) == 0 { continue }

		dir := entries[modeIdxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil { undecompilable += len(modeIdxs); continue }

		shapes := make([]idasrc.EntryShape, len(modeIdxs))
		for k, i := range modeIdxs {
			shapes[k] = idasrc.EntryShape{FName: entries[i].FName, Hand: entries[i].HandCalls}
		}
		for _, a := range idasrc.InferDispatchJoint(f, shapes) {
			if len(a.Dispatch) > 0 && a.Confidence >= opts.MinConfidence && len(a.Candidates) < 2 {
				accepted[a.FName] = idasrc.DispatchUpdate{
					Dispatch: a.Dispatch,
					Note:     fmt.Sprintf("inferred-high-confidence (%.2f) @%s", a.Confidence, addr),
				}
			} else {
				worklist = append(worklist, worklistItem{
					FName: a.FName, Address: addr, Proposed: a.Dispatch,
					Confidence: a.Confidence, Candidates: a.Candidates,
				})
			}
		}
	}

	if len(accepted) > 0 {
		if err := idasrc.WriteDispatch(opts.Baseline, accepted); err != nil {
			fmt.Fprintln(stdout, "resolve-dispatch: write baseline:", err)
			return 3
		}
	}
	if code := writeWorklist(opts, worklist, stdout); code != 0 { return code }

	sort.Slice(worklist, func(i, j int) bool { return worklist[i].FName < worklist[j].FName })
	fmt.Fprintf(stdout, "resolve-dispatch: %d auto-accepted (>=%.2f), %d to confirm, %d undecompilable\n",
		len(accepted), opts.MinConfidence, len(worklist), undecompilable)
	return 0
}

// writeWorklist writes the markdown confirmation worklist and a sibling .json. The
// agent reads the .json, confirms each pick against the IDA decompile at Address,
// and applies the confirmed selector via a follow-up WriteDispatch (or by hand).
func writeWorklist(opts resolveDispatchOpts, items []worklistItem, stdout io.Writer) int {
	sort.Slice(items, func(i, j int) bool { return items[i].FName < items[j].FName })
	var b strings.Builder
	fmt.Fprintf(&b, "# resolve-dispatch confirmation worklist\n\n%d low-confidence picks to confirm in IDA.\n\n", len(items))
	for _, it := range items {
		fmt.Fprintf(&b, "- `%s` @%s — proposed %v (conf %.2f) candidates %v\n",
			it.FName, it.Address, it.Proposed, it.Confidence, it.Candidates)
	}
	if dir := filepath.Dir(opts.Worklist); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil { fmt.Fprintln(stdout, "resolve-dispatch: mkdir:", err); return 3 }
	}
	if err := os.WriteFile(opts.Worklist, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: write worklist:", err); return 3
	}
	jsonPath := strings.TrimSuffix(opts.Worklist, filepath.Ext(opts.Worklist)) + ".json"
	jb, _ := json.MarshalIndent(items, "", "  ")
	jb = append(jb, '\n')
	if err := os.WriteFile(jsonPath, jb, 0o644); err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: write worklist json:", err); return 3
	}
	return 0
}
```

- [ ] **Step 4: Wire the subcommand in `cmd/root.go`**

Add the flag wrapper (mirror `runInfer`) and dispatch. In `Run`, add near the other prefixes:

```go
	if len(args) > 0 && args[0] == "resolve-dispatch" {
		return runResolveDispatch(args[1:], stderr)
	}
```

Add `runResolveDispatch` (copy `runInfer`'s flag/client wiring; flags: `--version`,
`--baseline` default `docs/packets/ida-exports/<version>.json`, `--worklist` (required),
`--min-confidence` default 0.6, `--ida-url`, `--ida-timeout`, `--ida-port`, `--descent-depth`).
Delegate to `resolveDispatchRun`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./cmd/ -run TestResolveDispatch -v`
Expected: PASS

- [ ] **Step 6: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add cmd/resolve_dispatch.go cmd/resolve_dispatch_test.go cmd/root.go
git commit -m "feat(task-081): resolve-dispatch subcommand (infer + agent-confirmation gate)"
git rev-parse --abbrev-ref HEAD
```

---

## Task 6: bijection / completeness buckets in `validate`

**Files:**
- Create: `internal/idasrc/bijection.go`, `internal/idasrc/bijection_test.go`
- Modify: `cmd/validate.go`
- Test: `cmd/validate_test.go`

Per base handler, diff the client case-set `C` (from `Fields.CaseLabels`, Task 3) against the
Atlas `#Mode` set `M` (the baseline selectors). Report `C \ M` as **missing-mode** and `M \ C`
as **extra-mode**, minus the allowlist (Task 7). New report buckets join verified/divergent/
unverifiable.

- [ ] **Step 1: Write the failing unit test for the pure diff**

Create `internal/idasrc/bijection_test.go`:

```go
package idasrc

import "testing"

func TestBijection_MissingAndExtra(t *testing.T) {
	cs := &CaseSet{}
	cs.add(1); cs.add(2); cs.add(9) // client has cases 1,2,9
	// Atlas modes (selectors) cover cases 1 and 9.
	modes := []ModeBinding{
		{FName: "X#A", Case: 1},
		{FName: "X#B", Case: 9},
		{FName: "X#Ghost", Case: 7}, // case 7 not in client -> extra
	}
	res := Bijection(cs, modes)
	if len(res.Missing) != 1 || res.Missing[0] != 2 {
		t.Fatalf("missing=%v want [2]", res.Missing)
	}
	if len(res.Extra) != 1 || res.Extra[0].FName != "X#Ghost" {
		t.Fatalf("extra=%v want [X#Ghost]", res.Extra)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestBijection -v`
Expected: FAIL — `Bijection`, `ModeBinding` undefined.

- [ ] **Step 3: Implement the pure diff**

Create `internal/idasrc/bijection.go`:

```go
package idasrc

import "sort"

// ModeBinding is one Atlas #Mode entry's case assignment within a base handler.
type ModeBinding struct {
	FName string
	Case  int64
}

// BijectionResult: client cases with no Atlas writer (Missing) and Atlas modes
// with no client case (Extra). Both sorted deterministically.
type BijectionResult struct {
	Missing []int64       // client case labels with no bound Atlas mode
	Extra   []ModeBinding // Atlas modes whose case is absent from the client
}

// Bijection diffs a client case-label set against the bound Atlas modes.
func Bijection(client *CaseSet, modes []ModeBinding) BijectionResult {
	bound := map[int64]bool{}
	for _, m := range modes { bound[m.Case] = true }
	var res BijectionResult
	if client != nil {
		for _, c := range client.Values() {
			if !bound[c] { res.Missing = append(res.Missing, c) }
		}
	}
	clientHas := map[int64]bool{}
	if client != nil {
		for _, c := range client.Values() { clientHas[c] = true }
	}
	for _, m := range modes {
		if !clientHas[m.Case] { res.Extra = append(res.Extra, m) }
	}
	sort.Slice(res.Missing, func(i, j int) bool { return res.Missing[i] < res.Missing[j] })
	sort.Slice(res.Extra, func(i, j int) bool { return res.Extra[i].FName < res.Extra[j].FName })
	return res
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/idasrc/ -run TestBijection -v`
Expected: PASS

- [ ] **Step 5: Wire bijection into `validate` and extend the report**

`validateRun` currently resolves each base by address via `ResolveLive` and extracts per
entry. Two changes:

1. `ResolveLive` must populate `Fields.CaseLabels`. Check `live.go`: if it builds `Fields`
   from `ParseDecompile`, switch that internal call to `ParseDecompileFields` so the case
   labels ride along. (Add a unit assertion in `live_test.go` that a switch handler yields
   non-empty `CaseLabels`.)
2. After the per-entry verification loop for an address, collect the `ModeBinding`s for that
   address's `#Mode` entries (FName + their `Dispatch[0].Case`, skipping `Default` selectors),
   call `Bijection(f.CaseLabels[disc], bindings)` for the discriminator the bindings use,
   subtract the allowlist (Task 7), and append `missing-mode` / `extra-mode` results.

Extend `shapeResult` with a `Kind` (reuse the FName/Detail fields; add a `Bucket` string:
`"missing-mode"`, `"extra-mode"`). In `writeReport`, add the two buckets to the roll-up line
and as `## missing-mode` / `## extra-mode` sections:

```go
fmt.Fprintf(&b, "verified %d / divergent %d / missing-mode %d / extra-mode %d / unverifiable %d\n\n",
	len(verified), len(divergent), len(missing), len(extra), len(unverifiable))
```

Add `cmd/validate_test.go` coverage: a fixture baseline + fake client where the client switch
has a case with no Atlas `#Mode` → assert a `missing-mode` row appears; an Atlas `#Mode` whose
case is absent from the client switch → assert an `extra-mode` row appears.

- [ ] **Step 6: Run validate tests**

Run: `go test ./cmd/ -run TestValidate -v`
Expected: PASS (existing + new missing/extra assertions).

- [ ] **Step 7: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/bijection.go internal/idasrc/bijection_test.go internal/idasrc/live.go internal/idasrc/live_test.go cmd/validate.go cmd/validate_test.go
git commit -m "feat(task-081): case<->mode bijection buckets in validate (missing/extra-mode)"
git rev-parse --abbrev-ref HEAD
```

---

## Task 7: per-version allowlist

**Files:**
- Create: `internal/idasrc/allowlist.go`, `internal/idasrc/allowlist_test.go`
- Modify: `cmd/validate.go` (load + apply)
- Create (data): `docs/packets/audits/<version>/_unimplemented.json` (one per version, as needed)

Suppress intentionally-unimplemented client cases so `missing-mode` doesn't re-surface them.

- [ ] **Step 1: Write the failing test**

Create `internal/idasrc/allowlist_test.go`:

```go
package idasrc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowlist_Suppress(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "_unimplemented.json")
	const j = `{"entries":[{"fname":"CWvsContext::OnPartyResult","case":12,"reason":"PQ feature not built"}]}`
	if err := os.WriteFile(p, []byte(j), 0o644); err != nil { t.Fatal(err) }

	al, err := LoadAllowlist(p)
	if err != nil { t.Fatal(err) }
	if !al.Suppressed("CWvsContext::OnPartyResult", 12) {
		t.Fatal("case 12 should be suppressed")
	}
	if al.Suppressed("CWvsContext::OnPartyResult", 3) {
		t.Fatal("case 3 should NOT be suppressed")
	}
	// Missing file -> empty allowlist, no error.
	empty, err := LoadAllowlist(filepath.Join(dir, "nope.json"))
	if err != nil { t.Fatal(err) }
	if empty.Suppressed("X", 1) { t.Fatal("empty allowlist suppressed nothing-expected") }
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestAllowlist -v`
Expected: FAIL — `LoadAllowlist`, `Allowlist` undefined.

- [ ] **Step 3: Implement**

Create `internal/idasrc/allowlist.go`:

```go
package idasrc

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

type allowEntry struct {
	FName  string `json:"fname"`
	Case   int64  `json:"case"`
	Reason string `json:"reason"`
}

// Allowlist records intentionally-unimplemented client cases so the bijection
// check counts but does not flag them as missing-mode.
type Allowlist struct {
	set map[string]map[int64]bool
}

// LoadAllowlist reads a per-version _unimplemented.json. A missing file yields an
// empty (suppress-nothing) allowlist with no error.
func LoadAllowlist(path string) (*Allowlist, error) {
	al := &Allowlist{set: map[string]map[int64]bool{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) { return al, nil }
		return nil, err
	}
	var doc struct{ Entries []allowEntry `json:"entries"` }
	if err := json.Unmarshal(b, &doc); err != nil { return nil, err }
	for _, e := range doc.Entries {
		if al.set[e.FName] == nil { al.set[e.FName] = map[int64]bool{} }
		al.set[e.FName][e.Case] = true
	}
	return al, nil
}

// Suppressed reports whether (fname, case) is an allowlisted unimplemented case.
func (a *Allowlist) Suppressed(fname string, c int64) bool {
	return a.set[fname] != nil && a.set[fname][c]
}
```

- [ ] **Step 4: Apply in `validate`**

In `validateRun`, load the allowlist next to the audit dir (default
`filepath.Join(filepath.Dir(opts.Baseline)/..., audits, <version>, "_unimplemented.json")` —
simplest: add a `validateOpts.Allowlist` path, defaulted in `runValidate` to
`docs/packets/audits/<version>/_unimplemented.json`, and `jms_v185` for jms per the context
doc). When building `missing-mode` results, skip any `(baseFName, case)` where
`al.Suppressed` is true; tally suppressed counts in the roll-up (`/ allowlisted %d`).

- [ ] **Step 5: Run tests**

Run: `go test ./internal/idasrc/ -run TestAllowlist -v && go test ./cmd/ -run TestValidate -v`
Expected: PASS

- [ ] **Step 6: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/allowlist.go internal/idasrc/allowlist_test.go cmd/validate.go cmd/root.go
git commit -m "feat(task-081): per-version unimplemented-case allowlist for bijection"
git rev-parse --abbrev-ref HEAD
```

---

## Task 8: end-to-end run on four IDBs + record results

**Files:**
- Create: `docs/tasks/task-081-ida-export-reharvest/per-branch-verification-results.md`
- Modify (data): `docs/packets/ida-exports/gms_*.json` (selectors written by resolve-dispatch)

No new code — exercise the pipeline live, confirm the bucket collapse, commit the populated
baselines and the results writeup. Build the tool once: `go build -o /tmp/packet-audit .`

- [ ] **Step 1: Resolve dispatch + agent-confirm, per version**

For each `(version, port)` in `(gms_v83,13337) (gms_v87,13338) (gms_v95,13339) (gms_jms_185,13340)`:

```bash
/tmp/packet-audit resolve-dispatch --version <version> --ida-port <port> \
  --worklist /tmp/t081rd/<version>.md
```

Then open the emitted `/tmp/t081rd/<version>.json` worklist; for each low-confidence item,
decompile its `address` on that IDB (`select_instance <port>` → `decompile`), read the actual
`switch`/`if (disc == N)` discriminator + the case label whose body matches the entry's hand
shape, and confirm/correct the selector. Apply confirmed selectors by writing a small
`updates` map through `WriteDispatch` (a throwaway `go run` snippet) or by hand-editing the
baseline `dispatch` field. Escalate only genuinely ambiguous client cases to the user.

- [ ] **Step 2: Re-run validate on all four, capture counts**

```bash
mkdir -p /tmp/t081rd
for vp in "gms_v83 13337" "gms_v87 13338" "gms_v95 13339" "gms_jms_185 13340"; do
  set -- $vp
  ad="docs/packets/audits/$1"; [ "$1" = gms_jms_185 ] && ad="docs/packets/audits/jms_v185"
  /tmp/packet-audit validate --version "$1" --ida-port "$2" \
    --report /tmp/t081rd/$1.md
done
grep -H 'verified .* missing-mode' /tmp/t081rd/*.md
```

Expected: the `unverifiable` "per-mode shape not extractable" sub-bucket has collapsed
(toward 0 for switch handlers; remaining ones are if/else arms the bail-rule couldn't
represent or genuinely undecompilable bases). `missing-mode` / `extra-mode` now carry real
findings.

- [ ] **Step 3: Triage `extra-mode` and seed allowlists**

Any `missing-mode` that is an intentionally-unimplemented feature → add to that version's
`docs/packets/audits/<version>/_unimplemented.json` with a reason; re-run validate to confirm
it drops out of `missing-mode` into the allowlisted tally. Any `extra-mode` is a latent
Atlas bug (a `#Mode` writer with no client case) → record it as a finding (do NOT fix encoders
in this pass; that is correctness work for a follow-up).

- [ ] **Step 4: Write the results doc**

Create `per-branch-verification-results.md` with the before/after table:

```
| Version | verified | divergent | missing-mode | extra-mode | unverifiable |  (was unverifiable)
```

against the 2026-06-09 baseline (verified 293 / divergent 296 / unverifiable 508; 450 of those
"per-mode not extractable"). State how many of the 450 collapsed, and what remains and why.

- [ ] **Step 5: Final gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add docs/packets/ida-exports/ docs/packets/audits/ docs/tasks/task-081-ida-export-reharvest/per-branch-verification-results.md
git commit -m "feat(task-081): per-branch verification live run — confirmed selectors + results"
git rev-parse --abbrev-ref HEAD
```

- [ ] **Step 6: Code review before any PR**

Per CLAUDE.md, run `superpowers:requesting-code-review` (dispatches `backend-guidelines-reviewer`
for the Go changes + `plan-adherence-reviewer` against this plan) BEFORE opening a PR. Address
findings, then decide on finishing the branch.

---

## Self-Review (completed by plan author)

- **Spec coverage:** design components 1→Task 2, 1a→Task 1, 2→Task 3, 3→Task 4, 4→Task 5,
  5→Task 6, 6→Task 7; onboarding tie-in exercised in Task 8; out-of-scope items explicitly
  deferred. All covered.
- **Type consistency:** `Selector.Default` (Task 1) used by parser default token (Task 2) and
  bijection skips `Default` selectors (Task 6); `DispatchUpdate`/`WriteDispatch` (Task 4) used
  by `resolve-dispatch` (Task 5); `CaseSet`/`CaseLabels`/`ParseDecompileFields` (Task 3)
  consumed by `Bijection` + `ResolveLive` (Task 6); `ModeBinding`/`Bijection`/`Allowlist`
  names consistent across Tasks 6–7.
- **Placeholder scan:** the only deferred specifics are (a) Task 2's expected guard values,
  which are deliberately transcribed from REAL harvested fixtures (the design forbids synthetic
  fixtures) — the harvest procedure is concrete, not a TODO; (b) Task 5/6 reuse the existing
  MCP-fake test scaffolding rather than re-specifying it — pointer to the exact files given.
- **Risk note:** Task 2 (if/else parser) is the highest-risk unit; it is fully isolated behind
  table-driven tests and a regression gate on the existing switch fixtures.
