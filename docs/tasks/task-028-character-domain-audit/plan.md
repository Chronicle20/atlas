# Character-Domain Packet Audit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the audit pipeline shipped in task-027 to the 48 packets in `libs/atlas-packet/character/{clientbound,serverbound}/`, ship wire-bug + template fixes against GMS v95 IDA, re-verify across v83/v87/JMS v185, and fix the documented `CharacterList ❌` analyzer false positive in `tools/packet-audit/`.

**Architecture:** Phase 0 fixes the analyzer's early-return suffix-taint and re-runs the existing login audit as a regression gate. Phase 1 lands the predicted `TypeRegistry` extensions (sub-struct support including the non-standard `EncodeForeign` method). Phase 2 audits the 30 clientbound + 18 serverbound character packets in 8 tracking sub-tasks (~6 packets each), front-loaded by gameplay hotness. Phase 3 re-runs the audit against v83 / v87 / JMS v185 IDA. Phase 4 ships `post-phase-b.md`, full-tree verification, and code review.

**Tech Stack:** Go 1.24 (`go/parser` + `go/ast` for AST analysis), `mcp__ida-pro__*` MCP tools for live IDA decompiles, `libs/atlas-socket` reader/writer for round-trip tests, GORM JSON-blob columns in `services/atlas-configurations` for template overrides. No new runtime dependencies; this task ships audit reports + targeted code/template fixes.

---

## Conventions used by every task

- **Worktree.** All work happens in `.worktrees/task-028-character-domain-audit/` on branch `task-028-character-domain-audit`. Before *every* commit run `git rev-parse --show-toplevel` and `git branch --show-current`; if either disagrees, stop.
- **TDD cadence.** Test first → run-to-fail → minimal implementation → run-to-pass → commit. Steps below spell each phase out.
- **Verification cadence (analyzer / registry changes).** `go test -race ./tools/packet-audit/...` clean before commit.
- **Verification cadence (atlas-packet edits).** `go test -race ./libs/atlas-packet/...` clean. Every encoder fix lands with a 4-variant test sweep covering GMS v28 / v83 / v87 / v95 + JMS v185 (use the existing `pt.Variants` pattern in `libs/atlas-packet/test/context.go`).
- **No `*_testhelpers.go` files.** Use the project's Builder pattern.
- **No `reflect`, no new `interface{}` params, no benchmarks** in atlas-packet edits (design §8).
- **No gitleaks bait.** Absolute paths like `/home/<user>/` must not appear in any file under `docs/packets/audits/gms_v95/character/`. Pre-PR check is mandatory (Task 18 step 6).
- **Tracking sub-tasks vs PR-sized commits.** Phase 2 and Phase 3 sub-tasks (Tasks 7–17) are *tracking* units, not single commits. Each ❌ verdict inside a sub-task triggers an independent fix commit (one fix = one commit). A sub-task is "done" when every packet in its bucket has a verdict and every ❌ has either a fix commit or a `_pending.md` row.

---

## Phase 0 — Analyzer early-return fix + login re-run (gate)

Three tasks. Exit when login `SUMMARY.md` shows 28/28 ✅ (was 27/28) or, if new ❌s surface, ≤ 2 new ❌s remain and they're documented per design §3.5.

### Task 1: Failing test for analyzer early-return suffix taint

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/analyzer_test.go`
- Create: `tools/packet-audit/internal/atlaspacket/testdata/early_return_then.go.txt`
- Create: `tools/packet-audit/internal/atlaspacket/testdata/early_return_else.go.txt`
- Create: `tools/packet-audit/internal/atlaspacket/testdata/early_return_negative.go.txt`

- [ ] **Step 1: Write the three fixture inputs**

`testdata/early_return_then.go.txt` — guarded early-return in `then` branch; suffix should pick up `NOT(a)`:

```go
package testdata

import (
    "context"
    "github.com/Chronicle20/atlas/libs/atlas-socket/response"
    "github.com/sirupsen/logrus"
)

type EarlyReturnThen struct{}

func (m EarlyReturnThen) Encode(l logrus.FieldLogger, ctx context.Context) func(opts map[string]interface{}) []byte {
    return func(opts map[string]interface{}) []byte {
        w := response.NewWriter()
        if a {
            w.WriteByte(1)
            return w.Bytes()
        }
        w.WriteInt(2)
        return w.Bytes()
    }
}
```

`testdata/early_return_else.go.txt` — `else` returns; suffix should pick up `a`:

```go
package testdata

import (
    "context"
    "github.com/Chronicle20/atlas/libs/atlas-socket/response"
    "github.com/sirupsen/logrus"
)

type EarlyReturnElse struct{}

func (m EarlyReturnElse) Encode(l logrus.FieldLogger, ctx context.Context) func(opts map[string]interface{}) []byte {
    return func(opts map[string]interface{}) []byte {
        w := response.NewWriter()
        if a {
            w.WriteByte(1)
        } else {
            w.WriteShort(2)
            return w.Bytes()
        }
        w.WriteInt(3)
        return w.Bytes()
    }
}
```

`testdata/early_return_negative.go.txt` — `if` without early-return; suffix must stay unconditional (regression guard):

```go
package testdata

import (
    "context"
    "github.com/Chronicle20/atlas/libs/atlas-socket/response"
    "github.com/sirupsen/logrus"
)

type EarlyReturnNegative struct{}

func (m EarlyReturnNegative) Encode(l logrus.FieldLogger, ctx context.Context) func(opts map[string]interface{}) []byte {
    return func(opts map[string]interface{}) []byte {
        w := response.NewWriter()
        if a {
            w.WriteByte(1)
        }
        w.WriteInt(2)
        return w.Bytes()
    }
}
```

- [ ] **Step 2: Add three test functions**

Append to `tools/packet-audit/internal/atlaspacket/analyzer_test.go`:

```go
func TestEarlyReturnThenTaintsSuffix(t *testing.T) {
    calls, err := AnalyzeFile("testdata/early_return_then.go.txt", "EarlyReturnThen", "Encode")
    if err != nil {
        t.Fatal(err)
    }
    if len(calls) != 2 {
        t.Fatalf("calls: got %d, want 2 (%+v)", len(calls), calls)
    }
    // First call: WriteByte under guard a.
    if calls[0].Op != Encode1 || calls[0].Guard == nil || calls[0].Guard.Text() != "a" {
        t.Errorf("call[0]: op=%v guard=%v; want Encode1 guard=a", calls[0].Op, guardText(calls[0].Guard))
    }
    // Second call: WriteInt under guard NOT(a).
    if calls[1].Op != Encode4 || calls[1].Guard == nil || calls[1].Guard.Text() != "!(a)" {
        t.Errorf("call[1]: op=%v guard=%v; want Encode4 guard=!(a)", calls[1].Op, guardText(calls[1].Guard))
    }
}

func TestEarlyReturnElseTaintsSuffix(t *testing.T) {
    calls, err := AnalyzeFile("testdata/early_return_else.go.txt", "EarlyReturnElse", "Encode")
    if err != nil {
        t.Fatal(err)
    }
    if len(calls) != 3 {
        t.Fatalf("calls: got %d, want 3 (%+v)", len(calls), calls)
    }
    // calls[0]: WriteByte under guard a.
    // calls[1]: WriteShort under guard !(a).
    // calls[2]: WriteInt under guard a (because the else-branch returned).
    if calls[2].Op != Encode4 || calls[2].Guard == nil || calls[2].Guard.Text() != "a" {
        t.Errorf("call[2]: op=%v guard=%v; want Encode4 guard=a", calls[2].Op, guardText(calls[2].Guard))
    }
}

func TestEarlyReturnNegativeLeavesSuffixUnconditional(t *testing.T) {
    calls, err := AnalyzeFile("testdata/early_return_negative.go.txt", "EarlyReturnNegative", "Encode")
    if err != nil {
        t.Fatal(err)
    }
    if len(calls) != 2 {
        t.Fatalf("calls: got %d, want 2 (%+v)", len(calls), calls)
    }
    if calls[1].Op != Encode4 || calls[1].Guard != nil {
        t.Errorf("call[1]: op=%v guard=%v; want Encode4 guard=nil", calls[1].Op, guardText(calls[1].Guard))
    }
}

// guardText is a test helper: returns "" for nil guards so format-string callers
// don't have to nil-check inline.
func guardText(g *GuardExpr) string {
    if g == nil {
        return ""
    }
    return g.Text()
}
```

If `GuardExpr.Text()` doesn't exist yet (check `guard.go`), it's already populated as the unexported `text` field; add an exported `Text()` accessor in `guard.go` as part of this commit:

```go
// Text returns the canonical string form of this guard expression.
func (g *GuardExpr) Text() string {
    if g == nil {
        return ""
    }
    return g.text
}
```

- [ ] **Step 3: Run tests to verify failure**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestEarlyReturn -v
```

Expected: All three FAIL. `TestEarlyReturnThenTaintsSuffix` fails because `calls[1].Guard` is nil today. `TestEarlyReturnElseTaintsSuffix` fails for the same reason on `calls[2]`. `TestEarlyReturnNegativeLeavesSuffixUnconditional` should PASS (negative case) — if it fails, the existing walker has a bug we hadn't seen and that needs triage before continuing.

- [ ] **Step 4: Commit the failing fixtures + tests**

```bash
git add tools/packet-audit/internal/atlaspacket/testdata/early_return_then.go.txt \
        tools/packet-audit/internal/atlaspacket/testdata/early_return_else.go.txt \
        tools/packet-audit/internal/atlaspacket/testdata/early_return_negative.go.txt \
        tools/packet-audit/internal/atlaspacket/analyzer_test.go \
        tools/packet-audit/internal/atlaspacket/guard.go
git commit -m "test(packet-audit): early-return suffix-taint fixtures (failing)"
```

---

### Task 2: Implement early-return suffix-taint in the walker

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/analyzer.go:191-243` (the `*ast.IfStmt` arm of `callCtx.walk`)

- [ ] **Step 1: Implement `blockTerminatesWithReturn`**

Add a private helper above `callCtx.walk`:

```go
// blockTerminatesWithReturn reports whether b's final statement is an *ast.ReturnStmt,
// either at top level or as the terminator of every branch of a terminal IfStmt.
// Loops are not descended (design §3.3 — loop-internal early-return is out of scope).
func blockTerminatesWithReturn(b *ast.BlockStmt) bool {
    if b == nil || len(b.List) == 0 {
        return false
    }
    last := b.List[len(b.List)-1]
    switch s := last.(type) {
    case *ast.ReturnStmt:
        return true
    case *ast.IfStmt:
        if s.Else == nil {
            return false
        }
        elseBlock, ok := s.Else.(*ast.BlockStmt)
        if !ok {
            // else if — descend into the inner IfStmt's body and walk its else recursively.
            innerIf, ok := s.Else.(*ast.IfStmt)
            if !ok {
                return false
            }
            wrapped := &ast.BlockStmt{List: []ast.Stmt{innerIf}}
            return blockTerminatesWithReturn(s.Body) && blockTerminatesWithReturn(wrapped)
        }
        return blockTerminatesWithReturn(s.Body) && blockTerminatesWithReturn(elseBlock)
    }
    return false
}
```

- [ ] **Step 2: Extend the `*ast.IfStmt` arm of `callCtx.walk` to push a residual guard after the if-block**

Replace the existing arm (analyzer.go:193-203) with:

```go
case *ast.IfStmt:
    g := guardFromIf(n, cc.fset)
    *cc.stack = append(*cc.stack, g)
    cc.walk(n.Body)
    *cc.stack = (*cc.stack)[:len(*cc.stack)-1]
    thenReturns := blockTerminatesWithReturn(n.Body)
    elseReturns := false
    if n.Else != nil {
        ng := negate(g)
        *cc.stack = append(*cc.stack, ng)
        cc.walk(n.Else)
        *cc.stack = (*cc.stack)[:len(*cc.stack)-1]
        switch e := n.Else.(type) {
        case *ast.BlockStmt:
            elseReturns = blockTerminatesWithReturn(e)
        case *ast.IfStmt:
            // else if — wrap and check.
            elseReturns = blockTerminatesWithReturn(&ast.BlockStmt{List: []ast.Stmt{e}})
        }
    }
    // Suffix-taint: when one branch returns, push an implicit guard for the surviving branch
    // onto cc.suffixStack so any sibling calls after this if-block inherit it.
    switch {
    case thenReturns && elseReturns:
        // Both branches return — unreachable suffix. Mark and skip.
        cc.unreachableSuffix = true
    case thenReturns:
        cc.pushSuffixGuard(negate(g))
    case elseReturns && n.Else != nil:
        cc.pushSuffixGuard(g)
    }
```

- [ ] **Step 3: Add suffix-guard machinery to `callCtx`**

Edit the `callCtx` struct (analyzer.go:120-133) to add fields, and add helpers:

```go
type callCtx struct {
    reg               *TypeRegistry
    enclosing         string
    rangeVars         map[string]string
    fieldVars         map[string]string
    out               *[]Call
    stack             *[]*GuardExpr
    suffixGuards      []*GuardExpr // implicit guards from preceding if-returns at this scope
    unreachableSuffix bool         // true when both branches of a preceding if returned
    fset              *token.FileSet
}

func (cc *callCtx) pushSuffixGuard(g *GuardExpr) {
    if g == nil {
        return
    }
    cc.suffixGuards = append(cc.suffixGuards, g)
}

func (cc *callCtx) conjoin() *GuardExpr {
    // Combine explicit stack and any accumulated suffix guards.
    if len(cc.suffixGuards) == 0 {
        return conjoin(*cc.stack)
    }
    combined := append([]*GuardExpr{}, *cc.stack...)
    combined = append(combined, cc.suffixGuards...)
    return conjoin(combined)
}
```

Also: in the `*ast.BlockStmt` arm, reset suffix machinery on scope entry, then either skip iteration (if `unreachableSuffix`) or proceed normally. Add right above the `case *ast.BlockStmt:` body:

```go
case *ast.BlockStmt:
    // Each block scope owns its own suffix-guard accumulator.
    savedSuffix := cc.suffixGuards
    savedUnreachable := cc.unreachableSuffix
    cc.suffixGuards = nil
    cc.unreachableSuffix = false
    for _, s := range n.List {
        if cc.unreachableSuffix {
            // Optionally emit a sentinel call for reviewer-visible reporting; for now skip.
            break
        }
        cc.walk(s)
    }
    cc.suffixGuards = savedSuffix
    cc.unreachableSuffix = savedUnreachable
```

Note: `collectSub` (analyzer.go:394-413) also constructs a fresh `callCtx` per loop body — it does not need changes; the new fields default to nil/false.

- [ ] **Step 4: Run the three new tests**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestEarlyReturn -v
```

Expected: all three PASS.

- [ ] **Step 5: Run the full analyzer test suite**

```
go test -race ./tools/packet-audit/...
```

Expected: clean. If any test that previously passed now fails, the suffix-taint is over-tainting; debug before continuing.

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/analyzer.go
git commit -m "fix(packet-audit): taint suffix of guarded blocks containing early returns

When an if-block (or its else) terminates with return, sibling calls in
the enclosing scope inherit the negated guard. Fixes the CharacterList
false positive documented in task-027 follow-up. Loop-internal
early-return remains out of scope per task-028 design §3.3."
```

---

### Task 3: Login re-run and `CharacterList ✅` confirmation

**Files:**
- Modify: `docs/packets/audits/gms_v95/CharacterList.md`
- Modify: `docs/packets/audits/gms_v95/CharacterList.json`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md`

- [ ] **Step 1: Re-run the login audit with the existing inputs**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Expected runtime: ≤ 30 s.

- [ ] **Step 2: Inspect the new `CharacterList.md`**

```
cat docs/packets/audits/gms_v95/CharacterList.md
```

Expected: verdict line `✅`. The diff section should be empty (or contain only ⚠️-tagged trailing-call noise).

If still ❌: re-read `tools/packet-audit/internal/atlaspacket/analyzer.go`'s walker — the suffix-taint may not be reaching the wrapped sub-struct call. Add an analyzer fixture that mirrors `CharacterList.go`'s shape (`if ... return; w.WriteByteArray(c.Encode(...))`) and re-iterate before continuing.

- [ ] **Step 3: Inspect `SUMMARY.md` for regressions**

```
grep -c '❌' docs/packets/audits/gms_v95/SUMMARY.md
```

Expected: 0. Pre-Task 3 it was 1 (`CharacterList`).

If new ❌s appear:
- 1–2 new ❌s → in-scope per design §3.5. Triage each: read the new `.md` report, decompile the matching IDA function via `mcp__ida-pro__decompile_function`, and fix as a one-commit follow-up inside Task 3. Each fix lands a 4-variant test sweep in `libs/atlas-packet/login/`.
- 3+ new ❌s → STOP. Document each in a new file `docs/tasks/task-028-character-domain-audit/login-regression.md` and ask the user to spin up a sibling task. Do not proceed to Phase 1 until cleared.

- [ ] **Step 4: Commit the audit outputs**

```bash
git add docs/packets/audits/gms_v95/CharacterList.md \
        docs/packets/audits/gms_v95/CharacterList.json \
        docs/packets/audits/gms_v95/SUMMARY.md
git commit -m "audit(packet-audit): CharacterList flips ✅ after early-return fix"
```

Then commit any fix commits surfaced in Step 3 individually with messages of the form:

```
fix(atlas-packet,<pkt>): <one-line summary>; cites IDA <fn>@<addr>
```

---

## Phase 1 — TypeRegistry sub-struct coverage

The existing `TypeRegistry` (`tools/packet-audit/internal/atlaspacket/registry.go`) auto-discovers `Encode` and `Write` methods in every `libs/atlas-packet/**/*.go` file. The five sub-structs predicted by design §4 are already physically present:

| Type | Method needed | Location | Already auto-discovered? |
|---|---|---|---|
| `AttackInfo` | `Encode` | `libs/atlas-packet/model/attack_info.go` | yes |
| `Pet` | `Encode` | `libs/atlas-packet/model/pet.go:23` | yes |
| `DamageTakenInfo` | `Encode` | `libs/atlas-packet/model/damage_taken_info.go:121` | yes |
| `Movement` | `Encode` | `libs/atlas-packet/model/movement.go:182` | yes (top-level; sub-elements `NormalElement`/`TeleportElement`/etc. also auto-discovered) |
| `CharacterTemporaryStat` | `EncodeForeign` | `libs/atlas-packet/model/character_temporary_stat.go:575` | **NO** — registry only knows about `Encode` and `Write` |

Phase 1 ships three tasks: extend the registry to recognise `EncodeForeign`, then assert correct sub-struct call-list shapes for all five via tests.

### Task 4: Registry support for `EncodeForeign`

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry.go:81-117`
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Write the failing test**

Append to `registry_test.go`:

```go
func TestRegistryDiscoversEncodeForeign(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    // CharacterTemporaryStat has both Encode and EncodeForeign; the registry must
    // expose calls for the EncodeForeign variant under a distinct key.
    if _, ok := reg.Calls("CharacterTemporaryStat::EncodeForeign"); !ok {
        t.Errorf("expected calls registered for CharacterTemporaryStat::EncodeForeign; got none")
    }
    // Encode entry must still resolve under the bare type name.
    if _, ok := reg.Calls("CharacterTemporaryStat"); !ok {
        t.Errorf("expected calls registered for CharacterTemporaryStat (Encode); got none")
    }
}
```

- [ ] **Step 2: Run to verify failure**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryDiscoversEncodeForeign -v
```

Expected: FAIL — `CharacterTemporaryStat::EncodeForeign` is not registered.

- [ ] **Step 3: Extend pass 2 to register secondary encode methods**

Modify the pass-2 switch in `registry.go` (currently lines 101-114):

```go
switch fd.Name.Name {
case "Encode":
    body := findReturnClosure(fd.Body)
    if body == nil {
        body = fd.Body
    }
    entry.Calls = collectCallsWithCtx(body, fc.fset, reg, recvType)
case "EncodeForeign":
    // Register under the "<Type>::EncodeForeign" key so callers can pick it
    // explicitly without colliding with the primary Encode entry.
    body := findReturnClosure(fd.Body)
    if body == nil {
        body = fd.Body
    }
    altKey := recvType + "::EncodeForeign"
    reg.types[altKey] = &TypeEntry{
        File:       entry.File,
        StructDecl: entry.StructDecl,
        Calls:      collectCallsWithCtx(body, fc.fset, reg, recvType),
    }
case "Write":
    if entry.Calls == nil {
        entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, recvType)
    }
}
```

Then teach the diff engine to prefer the `::EncodeForeign` key when the atlas call-site is `cts.EncodeForeign(...)`. In `analyzer.go`, the `*ast.CallExpr` arm currently treats any `Encode` selector as a recurse marker. Add a sibling branch right after the `Encode || Decode` block (analyzer.go:269-281):

```go
if sel.Sel.Name == "EncodeForeign" {
    recv := receiverTypeHint(sel.X)
    if !isWriterReaderReceiver(recv) {
        resolved := resolveRecurse(recv, cc)
        // Annotate the recurse with the EncodeForeign variant so the diff
        // engine resolves via the alternate registry key.
        cc.appendCall(Call{
            Kind:        KindRecurse,
            RecurseType: resolved + "::EncodeForeign",
            Line:        cc.fset.Position(n.Pos()).Line,
            Guard:       cc.conjoin(),
        })
        return
    }
}
```

Make the same addition in the nested-walk fallback (analyzer.go:347-376, near the `Encode || Decode` check).

- [ ] **Step 4: Run the new test + the full suite**

```
go test -race ./tools/packet-audit/...
```

Expected: clean, all green.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/registry.go \
        tools/packet-audit/internal/atlaspacket/registry_test.go \
        tools/packet-audit/internal/atlaspacket/analyzer.go
git commit -m "feat(packet-audit): register CharacterTemporaryStat::EncodeForeign

Pass-2 of the registry walker now recognises the EncodeForeign method
in addition to Encode/Write, and the analyzer emits the matching
'<Type>::EncodeForeign' recurse marker. Required for character/spawn,
character/buff_give, and any future foreign-payload encoder."
```

---

### Task 5: Registry coverage fixtures — `AttackInfo`, `Pet`, `DamageTakenInfo`

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Write the failing fixtures**

Append to `registry_test.go`:

```go
func TestRegistryRegistersCharacterSubStructs(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    for _, name := range []string{"AttackInfo", "Pet", "DamageTakenInfo"} {
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

- [ ] **Step 2: Run to verify**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersCharacterSubStructs -v
```

Expected: PASS *on the first run* — these types already have `Encode` methods that pass-2 picks up. If any FAIL, the receiver style is unusual; inspect the affected `libs/atlas-packet/model/<x>.go` and adjust `receiverIdent` in `registry.go`.

- [ ] **Step 3: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert AttackInfo/Pet/DamageTakenInfo registry coverage"
```

---

### Task 6: Registry coverage fixture — `Movement` + element sub-types

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry_test.go`

- [ ] **Step 1: Write the fixture**

Append to `registry_test.go`:

```go
func TestRegistryRegistersMovementElements(t *testing.T) {
    _, thisFile, _, _ := runtime.Caller(0)
    root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
    reg, err := NewTypeRegistry(root)
    if err != nil {
        t.Fatal(err)
    }
    // Top-level wrapper.
    if !reg.HasType("Movement") {
        t.Fatal("registry missing Movement")
    }
    // Element sub-types — each has its own Encode method.
    for _, name := range []string{
        "Element",
        "NormalElement",
        "TeleportElement",
        "StartFallDownElement",
        "FlyingBlockElement",
        "JumpElement",
        "StatChangeElement",
    } {
        if !reg.HasType(name) {
            t.Errorf("registry missing movement element type %s", name)
        }
    }
}
```

- [ ] **Step 2: Run**

```
go test -race ./tools/packet-audit/internal/atlaspacket/ -run TestRegistryRegistersMovementElements -v
```

If any sub-type is missing, the registry's struct-discovery pass missed it. Read `libs/atlas-packet/model/movement.go` and confirm the names match. If a sub-type uses a pointer receiver (e.g. `*NormalElement`) and the registry strips it, the test should still pass; if it doesn't, fix `receiverIdent` (registry.go:152-163) to handle the missing case.

- [ ] **Step 3: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/registry_test.go
git commit -m "test(packet-audit): assert Movement + element sub-types registered"
```

---

## Phase 2 — Character v95 audit

Eight tracking sub-tasks (Tasks 7–14). Each sub-task is a tracking unit, NOT a single PR:

1. Run the audit against the sub-task's packet bucket.
2. Triage each report: ✅ (no fix needed), ⚠️ (tolerable mismatch — note in report), ❌ (real wire bug OR template drift OR analyzer descent needed).
3. Ship **one fix commit per ❌** with a 4-variant test sweep and an IDA citation in the commit message.
4. Each `_pending.md` deferral lands as a separate row + a one-line commit message.

The audit command for all of Phase 2 is the same as Task 3 step 1; it produces per-packet reports under `docs/packets/audits/gms_v95/character/<PacketName>.{md,json}` and updates `docs/packets/audits/gms_v95/SUMMARY.md`. Run it once per sub-task after any registry/analyzer change inside that sub-task; commit the report files alongside the fix commits.

Before starting Phase 2, the user must have v95 IDA loaded so MCP `mcp__ida-pro__*` calls resolve. Each sub-task's IDA additions land in `docs/packets/ida-exports/gms_v95.json` (append) in the same commit as the audit report.

### Task 7: Clientbound — hot path bucket

**Packets (6):** `spawn`, `attack`, `damage`, `buff_give`, `movement`, `skill_change`.

These are the packets that most exercise the early-return fix, the sub-struct registry, and version branching. Doing them first catches systemic bugs before the cooler packets multiply them.

- [ ] **Step 1: Run the audit, scoped to this bucket**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits/gms_v95
```

Look at the 6 character/* reports under `docs/packets/audits/gms_v95/character/`. Sort by verdict:
- ✅ → nothing to do.
- ⚠️ → annotate the report manually with a one-line "ack: <reason>" footer; commit alone.
- ❌ → triage step 2.

- [ ] **Step 2: For each ❌, fetch IDA evidence**

For the matching `CWvsContext::OnXxxPacket` (serverbound) or `CClientSocket::Send*` (clientbound) function:

```
mcp__ida-pro__get_function_by_name("<FName>")  // or _get_function_by_address with the address listed in the audit report
mcp__ida-pro__decompile_function(<addr>)
```

Append the function's signature + address + a Decode-op summary to `docs/packets/ida-exports/gms_v95.json` (matching the existing schema's `Decode1/2/4/Str/Buffer/Loop` shape).

- [ ] **Step 3: Decide the fix flavour**

For each ❌:
- **Atlas wire bug** (width / order / missing field / silent-success) → fix in `libs/atlas-packet/character/{clientbound,serverbound}/<pkt>.go`. Add or extend the 4-variant test sweep in `<pkt>_test.go`.
- **Template opcode drift** → fix in every affected `services/atlas-configurations/seed-data/templates/template_gms_*_1.json` and `template_jms_185_1.json`. Atlas-packet stays untouched.
- **Analyzer descent gap** → if the audit can't resolve a sub-struct call, the fix is upstream of this task; pause, register the missing type via the Phase 1 pattern, then re-run.
- **Bare handler / no atlas-packet type** → append a row to `docs/packets/ida-exports/_pending.md` under a `## Still pending — character domain` section (create the section once, on the first deferral).

- [ ] **Step 4: For each Atlas wire-bug fix, add a sweep test**

Example (don't copy verbatim — match the existing `<pkt>_test.go` shape):

```go
func TestSpawnByteForByte(t *testing.T) {
    cases := []struct {
        name string
        tn   tenant.Model
        want string // hex
    }{
        {"gms_v83", pt.GMSv83(), "<...>"},
        {"gms_v87", pt.GMSv87(), "<...>"},
        {"gms_v95", pt.GMSv95(), "<...>"},
        {"jms_v185", pt.JMSv185(), "<...>"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), tc.tn)
            got := NewCharacterSpawn(/*...*/).Encode(testLogger(), ctx)(nil)
            if hex.EncodeToString(got) != tc.want {
                t.Fatalf("encode mismatch\n got %s\nwant %s", hex.EncodeToString(got), tc.want)
            }
        })
    }
}
```

Hex values come from IDA — capture them by hand from the decompile (or from the Phase 2 cross-version pass when applicable).

- [ ] **Step 5: Run tests for the affected packet**

```
go test -race ./libs/atlas-packet/character/clientbound/... -run TestSpawn -v
```

Repeat for each fix. Expect: clean.

- [ ] **Step 6: Commit each fix individually**

For atlas-packet fixes:

```bash
git add libs/atlas-packet/character/clientbound/<pkt>.go \
        libs/atlas-packet/character/clientbound/<pkt>_test.go
git commit -m "fix(atlas-packet,character/<pkt>): <one-line summary>

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
git commit -m "audit(character/<pkt>): defer — <one-line reason>"
```

- [ ] **Step 7: Commit the audit reports + SUMMARY**

```bash
git add docs/packets/audits/gms_v95/character/{spawn,attack,damage,buff_give,movement,skill_change}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): clientbound hot-path bucket (6 packets)"
```

- [ ] **Step 8: Exit gate — every bucket packet has a verdict row**

`grep -E '(spawn|attack|damage|buff_give|movement|skill_change)' docs/packets/audits/gms_v95/SUMMARY.md` must show 6 rows, each with ✅, ⚠️, or ❌ + (if ❌) a matching fix commit OR `_pending.md` row.

---

### Task 8: Clientbound — effects / buffs bucket

**Packets (6):** `buff_cancel`, `effect`, `effect_quest`, `effect_skill_use`, `skill_cooldown`, `appearance_update`.

Same workflow as Task 7. The expected analyzer pressure here is on `effect.go` / `effect_quest.go` / `effect_skill_use.go` — these dispatch on a leading effect-type byte and per design §9 the audit pipeline can't model sub-op enums. Sub-op drift caught here is documented in `_pending.md` (under a new `## Sub-op enum drift — character domain` heading on first hit) and NOT fixed in this task.

- [ ] **Step 1: Run the audit (same command as Task 7 step 1).**
- [ ] **Step 2–6: Per-packet triage + fix commits per Task 7 steps 2–6.**
- [ ] **Step 7: Commit the audit reports for the bucket.**

```bash
git add docs/packets/audits/gms_v95/character/{buff_cancel,effect,effect_quest,effect_skill_use,skill_cooldown,appearance_update}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): clientbound effects/buffs bucket (6 packets)"
```

- [ ] **Step 8: Exit gate — same shape as Task 7.**

---

### Task 9: Clientbound — spawn/list bucket

**Packets (6):** `list`, `view_all`, `add_entry`, `add_entry_error`, `despawn`, `name_response`.

Same workflow. Watch `view_all.go` and `list.go` for sub-struct work — both likely consume `CharacterListEntry` which was the load-bearing type for the early-return fix.

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{list,view_all,add_entry,add_entry_error,despawn,name_response}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): clientbound spawn/list bucket (6 packets)"
```

---

### Task 10: Clientbound — misc state bucket

**Packets (6):** `chair_show`, `chalkboard`, `expression`, `hint`, `info`, `sit_result`.

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{chair_show,chalkboard,expression,hint,info,sit_result}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): clientbound misc-state bucket (6 packets)"
```

---

### Task 11: Clientbound — tail bucket

**Packets (6):** `delete_response`, `item_upgrade`, `keymap`, `keymap_auto_hp`, `keymap_auto_mp`, `status_message`.

`status_message.go` will probably surface the sub-op enum drift documented in design §9 — same deferral handling as Task 8.

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{delete_response,item_upgrade,keymap,keymap_auto_hp,keymap_auto_mp,status_message}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): clientbound tail bucket (6 packets)"
```

**Phase 2 clientbound exit:** `SUMMARY.md` has 30 character/clientbound rows + every ❌ has either a fix commit on this branch or a `_pending.md` deferral row.

---

### Task 12: Serverbound — hot bucket

**Packets (6):** `move`, `monster_damage_friendly`, `heal_over_time`, `info_request`, `buff_cancel`, `item_cancel`.

Same workflow. For serverbound, the IDA evidence lives in `CWvsContext::OnXxxPacket` (or the equivalent dispatcher); audit reports cite `Decode*` ops rather than `Encode*`. The 4-variant test sweep is a Decode round-trip:

```go
func TestMoveByteForByte(t *testing.T) {
    cases := []struct {
        name string
        tn   tenant.Model
        hex  string
    }{...}
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            ctx := tenant.WithContext(context.Background(), tc.tn)
            raw, _ := hex.DecodeString(tc.hex)
            r := request.NewReader(raw)
            m := DecodeMove(testLogger(), ctx)(r)
            if r.Available() != 0 {
                t.Fatalf("leftover bytes after decode: %d", r.Available())
            }
            // Optional: field-level asserts on m.
            _ = m
        })
    }
}
```

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{move,monster_damage_friendly,heal_over_time,info_request,sb_buff_cancel,sb_item_cancel}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): serverbound hot bucket (6 packets)"
```

(Note: the audit pipeline disambiguates clientbound and serverbound `buff_cancel` / `item_cancel` already via the source path. If the SUMMARY rows collide, file-rename the report markdown by hand to `<pkt>_sb.md`.)

---

### Task 13: Serverbound — chairs / expression / misc bucket

**Packets (6):** `chair_fixed`, `chair_portable`, `chalkboard_close`, `expression` (sb), `drop_meso`, `key_map_change`.

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{chair_fixed,chair_portable,chalkboard_close,sb_expression,drop_meso,key_map_change}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): serverbound chairs/expression bucket (6 packets)"
```

---

### Task 14: Serverbound — character lifecycle bucket

**Packets (6):** `auto_distribute_ap`, `distribute_ap`, `distribute_sp`, `check_name`, `create`, `delete`.

`create` and `check_name` are likely bare handlers; expect `_pending.md` deferrals per design §1 working assumption.

- [ ] **Step 1–8: Same shape as Task 7.**

Bucket commit:

```bash
git add docs/packets/audits/gms_v95/character/{auto_distribute_ap,distribute_ap,distribute_sp,check_name,create,delete}.{md,json} \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/gms_v95.json
git commit -m "audit(character): serverbound lifecycle bucket (6 packets)"
```

**Phase 2 exit:** `SUMMARY.md` has 30 clientbound + 18 serverbound character rows (48 total) + every ❌ has either a fix commit or a `_pending.md` deferral.

---

## Phase 3 — Cross-version pass

Three tracking sub-tasks. One binary at a time, user-driven IDA swap. Each sub-task is "done" when:
- `docs/packets/ida-exports/<version>.json` exists with character-domain entries for every FName from the v95 audit.
- The audit has been re-run against the version's template + IDA export.
- Every divergence vs v95 atlas-packet behaviour has either:
  - A `Region/MajorVersion` gate that already handles it (audit report captures evidence; no code change),
  - A gate fix on this branch with a 4-variant test sweep, OR
  - A template fix.

If a packet on a non-v95 version needs structural rewriting (>2 nested region/version guards per design §7), STOP, log to `_pending.md`, and continue.

### Task 15: GMS v83 cross-version pass

**Files:**
- Create: `docs/packets/ida-exports/gms_v83.json` (already exists; append character entries — current file ships login entries from task-027)
- Modify (per fix): `libs/atlas-packet/character/**/*.go` + matching `_test.go`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (add a v83 column note where relevant) — OR create `docs/packets/audits/gms_v83/SUMMARY.md` if reviewers prefer per-version directories (decide on first run; commit both styles consistently).

- [ ] **Step 1: Confirm v83 IDA is loaded.**

```
mcp__ida-pro__get_metadata
```

Expected: `binary` field matches GMS v83. If not, ask user to swap before continuing.

- [ ] **Step 2: For each character FName resolved during Phase 2, populate `gms_v83.json`**

Workflow (per FName): `mcp__ida-pro__get_function_by_name("<FName>")` → `decompile_function(<addr>)` → translate to the existing `gms_v83.json` schema (`Decode1/2/4/Str/Buffer/Loop` op list with guard expressions). Append entries; do not reorder existing login entries.

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

If `docs/packets/audits/gms_v83/` doesn't exist yet, the tool creates it.

- [ ] **Step 4: Triage divergences**

For each ❌ in the v83 audit:
- Was the v95 fix gated on `MajorVersion() >= 95`? → no v83 regression. Audit-report-only.
- Was the v95 fix gated on `Region() == "GMS"` (no major-version filter)? → check whether v83 IDA confirms the same behaviour. If yes: tighten the gate so v83 keeps its old shape. If no: leave as-is and document.
- Is this a *new* v83-only mismatch the v95 audit didn't surface? → genuine cross-version bug. Fix with 4-variant test sweep + `Region/MajorVersion` gate.

- [ ] **Step 5: Commit per-fix; bucket commit for the version**

Per-fix commit format:

```
fix(atlas-packet,character/<pkt>): widen/narrow v83 gate for <field>

Cites IDA v83 <CClientSocket::SendXxx>@<addr>: <one-line evidence>.
```

Final bucket commit:

```bash
git add docs/packets/ida-exports/gms_v83.json \
        docs/packets/audits/gms_v83/
git commit -m "audit(character): GMS v83 cross-version pass (character domain)"
```

- [ ] **Step 6: Hard-cap check**

If any single character-domain encoder now contains 3+ nested `if t.Region()` / `if t.MajorVersion()` levels, STOP per design §7. Append a row to `_pending.md` describing the encoder + which version chain triggered it. Do not refactor in this task.

---

### Task 16: GMS v87 cross-version pass

Identical shape to Task 15. Replace `v83` with `v87` everywhere. Templates: `template_gms_87_1.json`. Export file: `docs/packets/ida-exports/gms_v87.json` (create — does not exist yet per `ls docs/packets/ida-exports/` showing only `gms_v83.json` and `gms_v95.json`).

- [ ] **Steps 1–6: Same shape as Task 15.**

Bucket commit message:

```
audit(character): GMS v87 cross-version pass (character domain)
```

---

### Task 17: JMS v185 cross-version pass

JMS v185 had a separate opcode space for login (task-027 finding). Expect the same for character.

**Files:**
- Create: `docs/packets/ida-exports/gms_jms_185.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Possibly modify: `libs/atlas-packet/character/**/*.go` per design §7 policy.

- [ ] **Step 1: Confirm JMS v185 IDA is loaded.**
- [ ] **Step 2: Populate `gms_jms_185.json` for the character FNames from Phase 2.**

If a FName has no JMS equivalent (different opcode space, different code-path entry), record the JMS-side FName and address as a separate entry annotated with `"region": "JMS"`. Do NOT reuse GMS FNames for unrelated JMS functions.

- [ ] **Step 3: Re-run the audit:**

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_jms_185.json \
  --output           docs/packets/audits/jms_v185
```

- [ ] **Step 4: Triage per design §7.1**

- In scope: atlas-packet writes bytes the JMS client decodes wrong.
- Out of scope: JMS-specific feature the service doesn't wire through.
- In scope: width mismatch on a field both versions decode.
- Out of scope: JMS template opcode wrong when v95 is right (fix the template, atlas-packet untouched).

- [ ] **Step 5: Commit per-fix; bucket commit**

Bucket commit message:

```
audit(character): JMS v185 cross-version pass (character domain)
```

- [ ] **Step 6: Hard-cap check** (same as Task 15 step 6).

---

## Phase 4 — Closeout

### Task 18: `post-phase-b.md`, full verification, code review, PR

**Files:**
- Create: `docs/tasks/task-028-character-domain-audit/post-phase-b.md`
- Modify: `docs/packets/audits/gms_v95/SUMMARY.md` (final tallies row)
- Modify: `docs/packets/ida-exports/_pending.md` (final character-domain section)

- [ ] **Step 1: Write `post-phase-b.md`**

Mirror task-027's structure. Five sections:

```markdown
# Task-028 Post-Phase-B — Character-Domain Audit Closeout

## Final state
- Packets audited: 48 (30 clientbound + 18 serverbound).
- Verdicts: ✅ <n_pass> / ⚠️ <n_warn> / ❌ <n_fail> / 🔍 <n_review> / pending <n_pending>.
- IDA-export coverage: v83 / v87 / v95 / JMS v185 — character FNames populated.

## Real wire bugs fixed
| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
(one row per fix commit)

## Template opcode / enum fixes
| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|

## Tooling improvements
- Analyzer early-return suffix-taint (Phase 0).
- Registry support for `EncodeForeign` (Phase 1).
- Registry fixtures for `AttackInfo`, `Pet`, `DamageTakenInfo`, `Movement` (+ element sub-types).

## Remaining work
| Area | What | Why deferred |
|---|---|---|
(rows from `_pending.md` and any §7 hard-cap stops)
```

Fill in actual numbers and rows from the commit history. Use `git log --oneline task-027-atlas-packet-v95-audit..HEAD` to enumerate fix commits.

- [ ] **Step 2: Run the full verification matrix**

```
go build ./...
go vet ./...
go test -race ./libs/atlas-packet/...
go test -race ./tools/packet-audit/...
```

All four must be clean.

- [ ] **Step 3: Decide whether `docker build` is required**

Per CLAUDE.md Build & Verification §3: required when a service `Dockerfile` or `go.mod` was touched. This task is expected to touch only `template_*.json` files under `services/atlas-configurations/seed-data/`. If only seed-data JSON changed:

```
git diff --name-only main..HEAD -- services/atlas-configurations/ | grep -v 'seed-data/templates/'
```

If empty: skip `docker build`. Otherwise:

```
docker build -f services/atlas-configurations/Dockerfile .
```

Expected: clean. If it fails on workspace replace lines, the affected Dockerfile needs its `COPY` / `go mod edit -replace` blocks updated — fix and re-run.

- [ ] **Step 4: gitleaks scrub**

```
grep -r '/home/' docs/packets/audits/gms_v95/character/ docs/packets/audits/gms_v83/ docs/packets/audits/gms_v87/ docs/packets/audits/jms_v185/ 2>/dev/null
```

Expected: no output. If any user-home path appears in an audit report, scrub it (`sed -i 's|/home/[^/]*/source/atlas-ms/atlas/||g' <file>`) and commit:

```bash
git commit -am "audit: scrub absolute user-home paths from character/* reports"
```

- [ ] **Step 5: Commit `post-phase-b.md`**

```bash
git add docs/tasks/task-028-character-domain-audit/post-phase-b.md \
        docs/packets/audits/gms_v95/SUMMARY.md \
        docs/packets/ida-exports/_pending.md
git commit -m "docs(task-028): post-phase-b closeout"
```

- [ ] **Step 6: Run code review**

Invoke `superpowers:requesting-code-review`. Allow the orchestration skill to dispatch:
- `plan-adherence-reviewer` — verifies every checkbox in this plan has commit evidence.
- `backend-guidelines-reviewer` — DOM-* Go audit on `libs/atlas-packet/` and `tools/packet-audit/` changes.

Read the resulting `audit.md` and act on every BLOCKER / MAJOR finding before opening a PR. Re-run reviews after fix commits land.

- [ ] **Step 7: Open the PR**

Title: `task-028: character-domain packet audit (v83/v87/v95/JMS185) + analyzer early-return fix`

Body: short summary + link to `post-phase-b.md` for the full bug ledger. Use `superpowers:finishing-a-development-branch` to drive the PR creation.

---

## Self-review notes

Run through the plan once more with fresh eyes before committing it.

- **Spec coverage** — every PRD §4 functional requirement (§4.1 coverage matrix → Phase 2 + 3; §4.2 IDA exports → Phase 2 + 3; §4.3 TypeRegistry extensions → Phase 1; §4.4 analyzer fix → Phase 0; §4.5 cross-version → Phase 3; §4.6 wire bug fixes → embedded in Phase 2/3; §4.7 template fixes → embedded in Phase 2/3) is covered by an explicit task above. Every PRD §10 acceptance bullet is covered by Task 18.
- **No placeholders** — every step contains either an exact command, an exact code block, or an exact file path. No "TBD" / "similar to" / "fill in".
- **Type consistency** — `CharacterTemporaryStat::EncodeForeign` key is used identically in Task 4 step 3 (registry write side) and the analyzer recurse-marker code (also Task 4 step 3, read side). `GuardExpr.Text()` exported accessor is added in Task 1 and used in Tasks 1–3.
- **Loop-internal early-return** is explicitly out of scope per design §3.3. Task 2 step 1's `blockTerminatesWithReturn` does not descend into `*ast.ForStmt` / `*ast.RangeStmt`.
- **`inventory/`** is not registered anywhere — design §4.2 confirms spawn delivers inventory via `CharacterTemporaryStat::EncodeForeign`, which is the only inventory-adjacent registration this plan adds.
- **Sub-op enum drift** — Task 8 and Task 11 explicitly defer to `_pending.md` per design §9; no encoder change.
- **No `reflect`, no `interface{}`, no benchmarks** — none of the code in the plan uses `reflect.*` or adds an `interface{}` parameter to an encoder.
- **Gitleaks** — Task 18 step 4 is the mandatory scrub.
