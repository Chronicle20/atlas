# Leaf Flat-Validation + Verbatim-Guard Dispatch — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut the 373 "per-mode shape not extractable" unverifiable entries by flat-validating leaf `#Mode` entries and matching verbatim non-equality branch conditions as dispatch selectors.

**Architecture:** Add a `HasMultiwayDispatch` signal + verbatim-guard emission to the parser, a `Selector.Guard` verbatim-clause match to extraction, verbatim-arm enumeration to inference, and a leaf-vs-dispatcher branch to `validate`. The surgical writer and the bijection are unchanged.

**Tech Stack:** Go (`tools/packet-audit`), table-driven tests with hand-crafted Hex-Rays-style `.c` fixtures.

**Design:** `non-equality-dispatch-design.md`. **Context (file/symbol map, ports, gates):** `per-branch-verification-context.md`.

---

## File Structure

- **Modify** `internal/idasrc/extract.go` — `Selector.Guard` + verbatim clause match. (Task 1)
- **Modify** `internal/idasrc/parse.go` — verbatim-guard arm emission. (Task 2)
- **Modify** `internal/idasrc/parse.go` + `idasrc.go` — `Fields.HasMultiwayDispatch`. (Task 3)
- **Modify** `internal/idasrc/infer.go` — `enumerateArms` + verbatim selector proposals. (Task 4)
- **Modify** `cmd/validate.go` — leaf flat-validation. (Task 5)
- **Fixtures:** `internal/idasrc/testdata/ifelse_noneq.c`, `leaf_linear.c`, `multiway_*.c`.

Run all `go`/`git` from `tools/packet-audit/`. After each commit verify the branch:
`git rev-parse --abbrev-ref HEAD` → `task-081-ida-export-reharvest`. Gate each task with
`go test -race ./... && go vet ./... && go build ./...` (tool, not a service → no docker bake / no redis).

**IDA-gated tasks (need MCP ports 13337–13340): Task 0 and Task 6. If the MCP is busy, do Tasks 1–5 offline and defer 0/6.** Tasks 1–5 use hand-crafted structural fixtures and run fully offline.

---

## Task 0: Live characterization of the 251 shared-address handlers — IDA-GATED

**Purpose:** confirm how much of the 251 Approach-1 covers (verbatim single-predicate arms) vs the residual that stays unverifiable (indirect/vtable). No code — produces a doc that informs nothing structurally but bounds expectations. **Skip/defer if the MCP is busy.**

- [ ] **Step 1: List the shared-address no-selector `#Mode` handlers**

```bash
for v in gms_v83 gms_v87 gms_v95 gms_jms_185; do python3 -c "
import json,collections
fns=json.load(open(f'docs/packets/ida-exports/$v.json'))['functions']
addrcount=collections.Counter(e['address'] for k,e in fns.items() if '#' in k)
shared=[(k,e['address']) for k,e in fns.items() if '#' in k and not e.get('dispatch') and not e.get('absent') and addrcount[e['address']]>1]
import collections as c; byaddr=c.defaultdict(list)
for k,a in shared: byaddr[a].append(k)
print('$v shared-address dispatcher addrs:', len(byaddr))
for a in list(byaddr)[:50]: print(' ',a, byaddr[a][0].split('#')[0])
"; done
```

- [ ] **Step 2: Decompile a sample per distinct base handler and classify**

For each distinct base handler address, decompile it on its IDB (`select_instance <port>` → `decompile <addr>` via the same IDA-MCP the tool uses) and classify the dispatch: `switch` / `if ==` / `if non-eq single predicate` (`<`,`>`,`>=`,`<=`,`&`,`!=`) / `flag` / `indirect (*…)` / `nested`. Tally per class.

- [ ] **Step 3: Write the characterization**

Create `docs/tasks/task-081-ida-export-reharvest/non-equality-characterization.md` with the per-class tally and the expected coverage (single-predicate arms → Approach 1 handles; indirect/vtable → residual unverifiable). Commit.

```bash
git add docs/tasks/task-081-ida-export-reharvest/non-equality-characterization.md
git commit -m "docs(task-081): characterize the 251 shared-address dispatch patterns"
git rev-parse --abbrev-ref HEAD
```

---

## Task 1: `Selector.Guard` + verbatim clause matching

**Files:**
- Modify: `internal/idasrc/extract.go`
- Test: `internal/idasrc/extract_test.go`

A selector with `Guard` set matches a read whose composed guard contains that exact clause
(one of the `&&`-split, paren-trimmed clauses equals `Guard`). Sets exactly one of
`{Case+Discriminator, Default, Guard}`.

- [ ] **Step 1: Write the failing test**

Add to `internal/idasrc/extract_test.go`:

```go
func TestExtractShape_VerbatimGuard(t *testing.T) {
	f := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},                  // pre-branch discriminator read
		{Op: Decode2, Guard: "v5 < 5"},            // non-equality arm
		{Op: Decode4, Guard: "v5 < 5 && loop n"},  // composed (arm + loop)
		{Op: DecodeStr, Guard: "v5 >= 5"},         // sibling arm
	}}
	got := ExtractShape(f, []Selector{{Guard: "v5 < 5"}})
	wantOps := []Primitive{Decode1, Decode2, Decode4} // header + both reads under "v5 < 5"
	if len(got) != len(wantOps) {
		t.Fatalf("got %d reads, want %d: %v", len(got), len(wantOps), got)
	}
	for i := range wantOps {
		if got[i].Op != wantOps[i] {
			t.Fatalf("[%d]=%s want %s", i, got[i].Op, wantOps[i])
		}
	}
	// A verbatim selector must NOT match the sibling arm.
	for _, c := range got {
		if c.Guard == "v5 >= 5" {
			t.Fatalf("verbatim selector matched sibling arm: %v", got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestExtractShape_VerbatimGuard -v`
Expected: FAIL — `Selector` has no `Guard` field (compile error).

- [ ] **Step 3: Implement**

In `extract.go`, add the field:

```go
type Selector struct {
	Discriminator string `json:"discriminator,omitempty"`
	Case          int64  `json:"case"`
	Default       bool   `json:"default,omitempty"`
	// Guard, when set, matches a read whose composed guard contains this exact
	// branch-condition clause (a non-equality dispatch arm, e.g. "v5 < 5").
	Guard string `json:"guard,omitempty"`
}
```

In `clauseMatches`, handle the verbatim guard BEFORE the equality scan. Insert at the top of
`clauseMatches` (after the existing `Default` handling):

```go
func clauseMatches(guard string, sel Selector) bool {
	if sel.Default {
		return strings.TrimSpace(guard) == DefaultGuardToken
	}
	if strings.TrimSpace(guard) == DefaultGuardToken {
		return false
	}
	if sel.Guard != "" {
		// Verbatim clause match: some &&-clause equals sel.Guard exactly.
		for _, clause := range strings.Split(guard, "&&") {
			clause = strings.TrimSpace(clause)
			clause = strings.TrimPrefix(clause, "(")
			clause = strings.TrimSuffix(clause, ")")
			if strings.TrimSpace(clause) == sel.Guard {
				return true
			}
		}
		return false
	}
	for _, clause := range strings.Split(guard, "&&") {
		// ... unchanged existing equality scan ...
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/idasrc/ -run TestExtractShape_VerbatimGuard -v`
Expected: PASS

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/extract.go internal/idasrc/extract_test.go
git commit -m "feat(task-081): Selector.Guard verbatim non-equality clause matching"
git rev-parse --abbrev-ref HEAD
```

---

## Task 2: Parser — verbatim-guard emission for non-equality arms

**Files:**
- Create: `internal/idasrc/testdata/ifelse_noneq.c`
- Modify: `internal/idasrc/parse.go`
- Test: `internal/idasrc/parse_test.go`

`reIfEq` (parse.go:70) matches only `disc == N`. Add a fallback `reIfCond` for an `if`/`else if`
whose condition is a single predicate (no `&&`/`||`, not indirect), emitting the **verbatim**
condition as the arm guard. Equality keeps flowing through `reIfEq` (normalized `disc == N`).

- [ ] **Step 1: Create the structural fixture**

`internal/idasrc/testdata/ifelse_noneq.c`:

```c
int __thiscall CFoo::OnNonEq(CFoo *this, CInPacket *a2)
{
  unsigned __int8 v5 = CInPacket::Decode1(a2);  // discriminator
  if ( v5 < 5 )
  {
    CInPacket::Decode2(a2);                       // small payload
  }
  else if ( v5 & 0x10 )
  {
    CInPacket::Decode4(a2);                       // flag payload
  }
  else
  {
    CInPacket::Decode8(a2);                        // default payload
  }
  return v5;
}
```

- [ ] **Step 2: Write the failing test**

Add to `internal/idasrc/parse_test.go`:

```go
func TestParseDecompile_NonEqVerbatimGuards(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_noneq.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[string]string{}
	for _, c := range calls {
		byOp[c.Op] = c.Guard
	}
	if byOp["Decode2"] != "v5 < 5" {
		t.Errorf("Decode2 guard = %q, want \"v5 < 5\"", byOp["Decode2"])
	}
	if byOp["Decode4"] != "v5 & 0x10" {
		t.Errorf("Decode4 guard = %q, want \"v5 & 0x10\"", byOp["Decode4"])
	}
	if byOp["Decode8"] != DefaultGuardToken {
		t.Errorf("Decode8 guard = %q, want %q", byOp["Decode8"], DefaultGuardToken)
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseDecompile_NonEqVerbatimGuards -v`
Expected: FAIL — non-equality arms currently get no guard (empty).

- [ ] **Step 4: Implement the verbatim fallback**

In `parse.go`, add a regex near `reIfEq`:

```go
// reIfCond matches an if / else-if header with ANY parenthesized condition,
// capturing the optional leading "else" and the full condition text. Used as a
// fallback to reIfEq: a single-predicate non-equality condition (e.g. "v5 < 5",
// "v5 & 0x10") is emitted as a verbatim arm guard. Compound/indirect conditions
// are rejected by isSinglePredicate so they bail to no-guard.
reIfCond = regexp.MustCompile(`^\s*(?:(else)\s+)?if\s*\(\s*(.+?)\s*\)\s*{?\s*$`)
```

Add a helper:

```go
// isSinglePredicate reports whether a condition is a single readable predicate
// suitable for a verbatim arm guard: it has no boolean combinator (&&/||) and is
// not an indirect/function-pointer expression. (Equality "x == N" is handled by
// reIfEq before this is consulted, so it never reaches here.)
func isSinglePredicate(cond string) bool {
	if strings.Contains(cond, "&&") || strings.Contains(cond, "||") {
		return false
	}
	if strings.Contains(cond, "(*") || strings.HasPrefix(strings.TrimSpace(cond), "*") {
		return false
	}
	return strings.TrimSpace(cond) != ""
}
```

In the arm-detection block (the `if m := reIfEq.FindStringSubmatch(line); m != nil { … } else if reElse.MatchString(line) … {` chain from Task 2), add a `reIfCond` branch BETWEEN the `reIfEq` and `reElse` cases. The branch mirrors the `reIfEq` chain-tracking logic but uses the verbatim condition as the fragment:

```go
} else if m := reIfCond.FindStringSubmatch(line); m != nil && isSinglePredicate(m[2]) {
	isElse, cond := m[1] == "else", strings.TrimSpace(m[2])
	if isElse && len(ifChains) > 0 && ifChains[len(ifChains)-1].discrim == cond {
		clearActiveArm()
		pendingArmFrag = cond
	} else {
		pendingArmFrag = cond
		ifChains = append(ifChains, ifChainEntry{startDepth: braceDepth, discrim: cond, armIdx: -1})
	}
} else if reElse.MatchString(line) && len(ifChains) > 0 {
	clearActiveArm()
	pendingArmFrag = DefaultGuardToken
}
```

> Note: a verbatim chain stores the condition text as the chain's `discrim` (each non-equality
> arm is its own predicate; the `else if (same cond)` continuation case is rare but handled).
> The pending-arm binding, `clearActiveArm`, and chain-pop logic from Task 2 are reused as-is.

- [ ] **Step 5: Run to verify it passes + regression**

Run: `go test ./internal/idasrc/ -run TestParseDecompile -v`
Expected: PASS for the new test AND every existing parser test (`TestParseModeSwitch`,
`TestParseDecompile_IfElseDispatch`, `TestParseLoopBreakInsideCase`, the real-fixture tests).
If an equality test broke, `reIfCond` is shadowing `reIfEq` — confirm `reIfEq` is tried first.

- [ ] **Step 6: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/parse.go internal/idasrc/parse_test.go internal/idasrc/testdata/ifelse_noneq.c
git commit -m "feat(task-081): emit verbatim guards for non-equality dispatch arms"
git rev-parse --abbrev-ref HEAD
```

---

## Task 3: `Fields.HasMultiwayDispatch`

**Files:**
- Create: `internal/idasrc/testdata/leaf_linear.c`, `internal/idasrc/testdata/multiway_if.c`
- Modify: `internal/idasrc/idasrc.go`, `internal/idasrc/parse.go`
- Test: `internal/idasrc/parse_test.go`

A function multi-way-dispatches when it has a `switch` with ≥2 `case` labels OR an if/else chain
with ≥2 arms on one discriminator. A lone optional-field `if`, or a single-case switch, is NOT
multi-way. `ParseDecompileFields` sets `Fields.HasMultiwayDispatch`.

- [ ] **Step 1: Create fixtures**

`internal/idasrc/testdata/leaf_linear.c` (lone optional `if` → NOT multi-way):

```c
int __thiscall CFoo::OnLeaf(CFoo *this, CInPacket *a2)
{
  CInPacket::Decode4(a2);          // id
  unsigned __int8 has = CInPacket::Decode1(a2);
  if ( has )
  {
    CInPacket::DecodeStr(a2, &s);  // optional name
  }
  return 0;
}
```

`internal/idasrc/testdata/multiway_if.c` (2-arm chain on one discriminator → multi-way):

```c
int __thiscall CFoo::OnMulti(CFoo *this, CInPacket *a2)
{
  unsigned __int8 v5 = CInPacket::Decode1(a2);
  if ( v5 == 1 )
  {
    CInPacket::Decode4(a2);
  }
  else if ( v5 == 2 )
  {
    CInPacket::Decode2(a2);
  }
  return v5;
}
```

- [ ] **Step 2: Write the failing test**

Add to `internal/idasrc/parse_test.go`:

```go
func TestParseDecompileFields_HasMultiwayDispatch(t *testing.T) {
	cases := []struct {
		fixture string
		want    bool
	}{
		{"leaf_linear.c", false},       // lone optional if
		{"multiway_if.c", true},        // 2-arm equality chain
		{"mode_switch.c", true},        // switch with 2 cases
		{"switch_emptycase.c", true},   // switch with 3 cases
		{"linear.c", false},            // no branches at all
	}
	for _, tc := range cases {
		f, err := ParseDecompileFields(mustFixture(t, tc.fixture), DirClientbound)
		if err != nil {
			t.Fatalf("%s: %v", tc.fixture, err)
		}
		if f.HasMultiwayDispatch != tc.want {
			t.Errorf("%s: HasMultiwayDispatch=%v want %v", tc.fixture, f.HasMultiwayDispatch, tc.want)
		}
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestParseDecompileFields_HasMultiwayDispatch -v`
Expected: FAIL — `Fields.HasMultiwayDispatch` undefined.

- [ ] **Step 4: Implement**

In `idasrc.go`, add the field to `Fields`:

```go
type Fields struct {
	Function           string
	Address            string
	Direction          Direction
	Calls              []FieldCall
	CaseLabels         map[string]*CaseSet
	HasMultiwayDispatch bool
}
```

In `parse.go`, compute it in `collectCaseLabels`'s caller or as a sibling pass and set it in
`ParseDecompileFields`. Simplest: have `collectCaseLabels` ALSO return whether any discriminator
has ≥2 case labels, and detect ≥2-arm if/else chains in the same scan. Add a `multiway` return:

```go
func collectCaseLabels(text string) (labels map[string]*CaseSet, multiway bool) {
	// ... existing scan, plus: track per-chain arm counts and per-switch case counts ...
	// After building labels: a switch discriminator with len(CaseSet.Values()) >= 2 is multiway.
	for _, cs := range labels {
		if len(cs.Values()) >= 2 {
			multiway = true
		}
		if len(cs.Values()) >= 1 && cs.Default {
			// 1 case + default = 2-way
			multiway = true
		}
	}
	// if/else chains: track armsByChainDisc[disc]++ when an arm opens; >=2 => multiway.
	return labels, multiway
}
```

Track if/else-chain arm counts during the scan: increment a `chainArms[disc]` counter each time
an arm header (`reIfEq`/`reIfCond`) opens for discriminator `disc`; if any reaches ≥2, set
`multiway`. (The existing `collectCaseLabels` already tracks `chains` by discriminator — extend
it with the per-disc arm counter.) Update `ParseDecompileFields`:

```go
func ParseDecompileFields(text string, dir Direction) (Fields, error) {
	calls, err := ParseDecompile(text, dir)
	if err != nil {
		return Fields{}, err
	}
	labels, multiway := collectCaseLabels(text)
	return Fields{
		Direction:           dir,
		Calls:               toFieldCalls(calls),
		CaseLabels:          labels,
		HasMultiwayDispatch: multiway,
	}, nil
}
```

Update `ResolveLive` (live.go) where it sets `f.CaseLabels = collectCaseLabels(baseText)` — it
now must capture both returns:

```go
labels, multiway := collectCaseLabels(baseText)
f.CaseLabels = labels
f.HasMultiwayDispatch = multiway
```

- [ ] **Step 5: Run to verify it passes + regression**

Run: `go test ./internal/idasrc/ -run 'TestParseDecompile|TestResolveLive' -v`
Expected: PASS (new test + existing `TestParseDecompile_CaseLabelSet`, `TestResolveLiveCaseLabels`).

- [ ] **Step 6: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/idasrc.go internal/idasrc/parse.go internal/idasrc/live.go internal/idasrc/parse_test.go internal/idasrc/testdata/leaf_linear.c internal/idasrc/testdata/multiway_if.c
git commit -m "feat(task-081): Fields.HasMultiwayDispatch (switch>=2 / chain>=2 arms)"
git rev-parse --abbrev-ref HEAD
```

---

## Task 4: Inference — verbatim selector proposals

**Files:**
- Modify: `internal/idasrc/infer.go`
- Test: `internal/idasrc/infer_test.go`

`enumerateCases` (infer.go:279) collects numeric `disc == N` cases only. Generalize to
`enumerateArms`, returning each distinct dispatch **arm** as a `Selector` (equality →
`{Discriminator, Case}`; verbatim → `{Guard}`; default → `{Default}`). `InferDispatchJoint`
scores hand shapes against each arm's `ExtractShape(base, []Selector{arm})` and assigns one-to-one
as today, carrying the arm's `Selector` into `Assignment.Dispatch`.

- [ ] **Step 1: Write the failing test**

Add to `internal/idasrc/infer_test.go`:

```go
func TestInferDispatchJoint_VerbatimArm(t *testing.T) {
	// base: discriminator read, then a non-equality arm "v5 < 5" reading Decode2,
	// and an equality arm "v5 == 9" reading Decode4.
	base := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},
		{Op: Decode2, Guard: "v5 < 5"},
		{Op: Decode4, Guard: "v5 == 9"},
	}}
	entries := []EntryShape{
		{FName: "Foo#Small", Hand: []FieldCall{{Op: Decode1}, {Op: Decode2}}}, // -> v5 < 5
		{FName: "Foo#Nine", Hand: []FieldCall{{Op: Decode1}, {Op: Decode4}}},  // -> v5 == 9
	}
	got := map[string]Selector{}
	for _, a := range InferDispatchJoint(base, entries) {
		if len(a.Dispatch) == 1 {
			got[a.FName] = a.Dispatch[0]
		}
	}
	if got["Foo#Small"].Guard != "v5 < 5" {
		t.Errorf("#Small dispatch = %+v, want Guard \"v5 < 5\"", got["Foo#Small"])
	}
	if got["Foo#Nine"].Case != 9 || got["Foo#Nine"].Discriminator != "v5" {
		t.Errorf("#Nine dispatch = %+v, want v5==9", got["Foo#Nine"])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/idasrc/ -run TestInferDispatchJoint_VerbatimArm -v`
Expected: FAIL — verbatim arm not enumerated, `#Small` gets no/​wrong dispatch.

- [ ] **Step 3: Implement `enumerateArms`**

Add to `infer.go` (keep `enumerateCases` for any other caller; `InferDispatchJoint`/`InferDispatch`
switch to `enumerateArms`):

```go
// enumerateArms collects the distinct dispatch arms present across the base's
// guards as Selectors, in first-seen order: an equality clause "disc == N" yields
// {Discriminator: disc, Case: N}; the default token yields {Default: true}; any
// other single non-loop clause yields a verbatim {Guard: clause}. Loop clauses
// ("loop ...") are not dispatch arms and are skipped.
func enumerateArms(base Fields) []Selector {
	seen := map[string]bool{}
	var arms []Selector
	addEq := func(disc string, v int64) {
		key := "eq:" + disc + ":" + strconv.FormatInt(v, 10)
		if !seen[key] {
			seen[key] = true
			arms = append(arms, Selector{Discriminator: disc, Case: v})
		}
	}
	addVerbatim := func(clause string) {
		key := "g:" + clause
		if !seen[key] {
			seen[key] = true
			arms = append(arms, Selector{Guard: clause})
		}
	}
	for _, call := range base.Calls {
		g := strings.TrimSpace(call.Guard)
		if g == "" {
			continue
		}
		if g == DefaultGuardToken {
			if !seen["default"] {
				seen["default"] = true
				arms = append(arms, Selector{Default: true})
			}
			continue
		}
		for _, clause := range strings.Split(g, "&&") {
			clause = strings.TrimSpace(clause)
			clause = strings.TrimPrefix(clause, "(")
			clause = strings.TrimSuffix(clause, ")")
			clause = strings.TrimSpace(clause)
			if clause == "" || strings.HasPrefix(clause, "loop ") {
				continue
			}
			if parts := strings.SplitN(clause, "==", 2); len(parts) == 2 {
				if v, ok := parseIntLit(strings.TrimSpace(parts[1])); ok {
					addEq(strings.TrimSpace(parts[0]), v)
					continue
				}
			}
			addVerbatim(clause)
		}
	}
	return arms
}
```

In `InferDispatchJoint`, replace the `disc, cases := enumerateCases(base)` + per-case
`ExtractShape(base, []Selector{{Discriminator: disc, Case: cases[j]}})` machinery with
`arms := enumerateArms(base)` and `ExtractShape(base, []Selector{arms[j]})`; the assignment's
`Dispatch` becomes `[]Selector{arms[j]}`. The greedy one-to-one assignment, scoring, and
joint-confidence logic are otherwise unchanged (they operate on column index `j`, now indexing
`arms` instead of `cases`). Apply the same substitution in `InferDispatch`.

> Keep `enumerateCases` in the file (it may have other callers / tests); `enumerateArms` is the
> superset. If `go vet`/build flags `enumerateCases` as unused after the switch, delete it and
> its test in the same commit.

- [ ] **Step 4: Run to verify it passes + regression**

Run: `go test ./internal/idasrc/ -run 'TestInfer' -v`
Expected: PASS (new verbatim test + existing `TestInferDispatchJoint*` equality tests — the
canonical OnFriendResult 8-vs-9 joint case must still resolve).

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add internal/idasrc/infer.go internal/idasrc/infer_test.go
git commit -m "feat(task-081): enumerateArms — infer verbatim non-equality selectors"
git rev-parse --abbrev-ref HEAD
```

---

## Task 5: Validate — leaf flat-validation

**Files:**
- Modify: `cmd/validate.go`
- Test: `cmd/validate_test.go`

Replace the isMode short-circuit (cmd/validate.go:123-126). A `#Mode` entry with empty dispatch
flat-validates when the live function is NOT a multi-way dispatcher.

- [ ] **Step 1: Write the failing tests (fixtures via the fake MCP)**

Add to `cmd/validate_test.go` two decompiles + a baseline:

```go
// leafDecomp: a linear leaf handler (no dispatch) — Decode4 then Decode2.
const leafDecomp = "void __thiscall Foo::OnLeaf(Foo *this, CInPacket *a2)\n{\n" +
	"  CInPacket::Decode4(a2);\n  CInPacket::Decode2(a2);\n}\n"

func TestValidate_LeafModeFlatValidated(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x300": leafDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/leaf_mode.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	// #Solo has no selector but its function is a leaf -> flat-validated -> verified.
	if got := sectionOf(s, "Foo::OnLeaf#Solo"); got != "verified" {
		t.Fatalf("Foo::OnLeaf#Solo in %q, want verified\n%s", got, s)
	}
}
```

Create `cmd/testdata/leaf_mode.json`:

```json
{
 "binary":"x","md5":"x","generated_at":"t",
 "functions":{
  "Foo::OnLeaf#Solo":{"address":"0x300","direction":"clientbound",
    "calls":[{"op":"Decode4","guard":""},{"op":"Decode2","guard":""}]}
 }
}
```

Also assert a multi-way solo entry stays unverifiable — reuse `authDecomp` (0x200, a 2-case
switch) with a no-dispatch `#Mode` baseline:

```go
func TestValidate_MultiwayModeStaysUnverifiable(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": authDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/multiway_nosel.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if got := sectionOf(s, "Auth::OnCheckPasswordResult#NoSel"); got != "unverifiable" {
		t.Fatalf("multiway no-selector entry in %q, want unverifiable\n%s", got, s)
	}
}
```

Create `cmd/testdata/multiway_nosel.json`:

```json
{
 "binary":"x","md5":"x","generated_at":"t",
 "functions":{
  "Auth::OnCheckPasswordResult#NoSel":{"address":"0x200","direction":"clientbound",
    "calls":[{"op":"Decode1","guard":""},{"op":"Decode4","guard":""}]}
 }
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./cmd/ -run 'TestValidate_LeafModeFlatValidated|TestValidate_MultiwayModeStaysUnverifiable' -v`
Expected: FAIL — `#Solo` currently lands in unverifiable (the blanket isMode rule).

- [ ] **Step 3: Implement**

In `cmd/validate.go`, replace the isMode block:

```go
			isMode := strings.Contains(e.FName, "#")
			switch {
			case isMode && len(e.Dispatch) > 0:
				live := idasrc.ExtractShape(f, e.Dispatch)
				if len(live) == 0 {
					verdict = idasrc.ShapeUnverifiable
					detail = "per-mode selector matched no reads"
				} else {
					verdict, detail = idasrc.ValidateShape(e.HandCalls, live)
				}
			case isMode && !f.HasMultiwayDispatch:
				// Leaf handler: the whole function IS this entry's wire shape.
				verdict, detail = idasrc.ValidateShape(e.HandCalls, f.Calls)
			case isMode:
				// Multi-way dispatcher with no selector: genuinely not extractable.
				verdict = idasrc.ShapeUnverifiable
				detail = "per-mode shape not extractable (no usable dispatch selector)"
			default:
				live := idasrc.ExtractShape(f, e.Dispatch)
				verdict, detail = idasrc.ValidateShape(e.HandCalls, live)
			}
```

(Remove the now-redundant `live := idasrc.ExtractShape(f, e.Dispatch)` that preceded the old
block; each branch computes what it needs. The non-`#` `default` branch preserves the prior
flat behavior.)

- [ ] **Step 4: Run to verify they pass + regression**

Run: `go test ./cmd/ -run TestValidate -v`
Expected: PASS for the two new tests AND existing `TestValidateRunReport`,
`TestValidateRunUndispatchable`, `TestValidate_BijectionMissingExtra`,
`TestValidate_BijectionMultiAddressNoFalseMissing`, `TestValidate_AllowlistSuppressesMissing`.

> Watch `TestValidateRunUndispatchable`: its `#AuthLoginFailed` has no dispatch and its base
> (`authDecomp`, 0x200) is a 2-case switch → `HasMultiwayDispatch=true` → stays unverifiable.
> Confirm that assertion still holds (it should — the multiway branch covers it).

- [ ] **Step 5: Full gate + commit**

```bash
go test -race ./... && go vet ./... && go build ./...
git add cmd/validate.go cmd/validate_test.go cmd/testdata/leaf_mode.json cmd/testdata/multiway_nosel.json
git commit -m "feat(task-081): validate flat-validates leaf #Mode entries"
git rev-parse --abbrev-ref HEAD
```

---

## Task 6: E2E re-validate on four IDBs + results — IDA-GATED

**Purpose:** measure the per-mode-not-extractable collapse. No code. **Defer if the MCP is busy.**

- [ ] **Step 1: (Optional) re-run resolve-dispatch to harvest verbatim selectors**

Building the tool: `go build -o /tmp/packet-audit .`. For each `(version, port)` —
`(gms_v83,13337) (gms_v87,13338) (gms_v95,13339) (gms_jms_185,13340)`:

```bash
/tmp/packet-audit resolve-dispatch --version <version> --ida-port <port> --worklist /tmp/ne/<version>.md
```

This now proposes verbatim `{Guard}` selectors for non-equality arms; auto-accept writes them
via the surgical writer (additive diff). Agent-confirm low-confidence picks in IDA as in the
per-branch run.

- [ ] **Step 2: Re-validate all four, capture counts**

```bash
for vp in "gms_v83 13337" "gms_v87 13338" "gms_v95 13339" "gms_jms_185 13340"; do
  set -- $vp
  /tmp/packet-audit validate --version "$1" --ida-port "$2" --report /tmp/ne/$1.md
done
grep -H 'verified' /tmp/ne/*.md
```

Expected: the `unverifiable` "per-mode shape not extractable" sub-bucket shrinks — leaf entries
move to verified/divergent, and non-equality dispatchers gain verbatim selectors. Residual:
indirect/vtable handlers (honestly unverifiable). Remember the jms audit-dir is `jms_v185`.

- [ ] **Step 2b: Commit any newly-written verbatim selectors**

```bash
git add docs/packets/ida-exports/
git commit -m "feat(task-081): persist verbatim non-equality dispatch selectors"
git rev-parse --abbrev-ref HEAD
```

- [ ] **Step 3: Write the results doc**

Create `docs/tasks/task-081-ida-export-reharvest/non-equality-dispatch-results.md` with the
before→after table (against the post-per-branch baseline: verified 352 / unverifiable 434, of
which 373 per-mode-not-extractable). State how many leaf entries flat-validated, how many
non-equality selectors landed, and the residual + why. Commit.

- [ ] **Step 4: Code review before any PR**

Per CLAUDE.md, run `superpowers:requesting-code-review` (backend-guidelines + plan-adherence)
before opening a PR. Address findings.

---

## Self-Review (completed by plan author)

- **Spec coverage:** design component 1→Task 2, 2→Task 1, 3→Task 4, 4→Task 5, leaf-detection
  signal→Task 3, characterization→Task 0, bijection-unchanged→noted (no task needed), E2E→Task 6.
  All covered.
- **Type consistency:** `Selector.Guard` (Task 1) is read by `clauseMatches` (Task 1), proposed
  by `enumerateArms` (Task 4), persisted unchanged by the surgical writer; `HasMultiwayDispatch`
  (Task 3) is set by `ParseDecompileFields`/`ResolveLive` and consumed by `validate` (Task 5);
  `collectCaseLabels` signature change (→ returns `multiway`) is propagated to both call sites
  (`ParseDecompileFields` and `ResolveLive`) in Task 3.
- **Placeholder scan:** the only deferred specifics are the IDA-gated Task 0 / Task 6 procedures
  (concrete commands, gated by MCP availability — not TODOs) and the standing real-fixture
  hardening owed from Task 2 (exercised in the live E2E). Offline Tasks 1–5 are fully concrete.
- **Ordering note:** Task 1 (Selector.Guard) before Task 4 (proposes it) before Task 5 (extracts
  it); Task 2/3 (parser) before Task 5 (consumes HasMultiwayDispatch). Dependencies satisfied.
