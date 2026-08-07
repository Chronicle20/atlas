# v95 Skill `common` Formula Nodes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `services/atlas-data` derive per-level skill effects from `Skill.wz` `common` formula subtrees, so the 635 GMS v95.1 skills that currently serve `{"maxLevel": 0, "effects": []}` serve a correct level cap and a correct effects array.

**Architecture:** A new pure package `skill/formula` tokenizes and parses the `common` expression grammar into a reusable `Expr`, evaluated once per level with the *client's* arithmetic semantics (see Global Constraints). A declared key table in `skill/common.go` classifies each `common` child leaf (expression / vector pass-through / opaque / skill-level metadata) and carries the target field's integer range. `expandCommon` evaluates each parsed expression at levels `1..maxLevel` and synthesizes one `xml.Node` per level, which is fed through the **existing, unmodified** `getEffect` — so both read paths share one post-processing implementation. Failures are scoped (skill / key / key-at-level), logged at ERROR with structured fields, counted into a `Stats` value that flows out through `Read` → `Register` → `RegisterSkill` to a per-run summary.

**Tech Stack:** Go 1.x (module `atlas-data`, workspace `go.work`), `logrus` structured logging, the repo's `xml.Node` WZ-serialization model, TypeScript/React for the atlas-ui surfacing task.

## Global Constraints

These apply to **every** task. Values are copied verbatim from `design.md` / `prd.md`.

- **Detection is structural, never version-gated (FR-1.4).** `common` handling keys off `ChildByName("common")` only. No region/major/minor predicate anywhere in this change.
- **COMMON wins unconditionally (FR-1.2).** When `common` is present the `level` subtree is not read. Skills 2211002 and 2211006 therefore lose `mad`/`mastery` and cap at level 20 instead of 30 — an accepted, test-pinned regression.
- **Client arithmetic semantics, not textbook semantics.** Sourced from IDA session `79906a1e`, `GMS_v95.0_U_DEVM.exe.i64`, `SKILLLEVELDATA::GetParsedCommonData` (`0x6fe560`) and `SKILLLEVELDATA::GetArithmeticData` (`0x6f9300`):
  - Precedence, loosest first: `+` → `-` → `/` → `*`. (i.e. `*` binds tightest, then `/`, then `-`, then `+`.)
  - **Every binary operation truncates its result toward zero before the next operation consumes it.**
  - `d(v)` is `trunc(v)`, **not** `math.Floor`. `u(v)` is `trunc(v + 0.999999)`, **not** `math.Ceil`.
  - The `u()` ceiling is applied **in place of** the truncation of the argument's outermost binary operation — not after it. `u(x/2)` at `x=1` must be **1**; a naive "truncate then ceil" yields 0 and is wrong.
  - Division is real float division (`float64`), never Go integer division.
  - Expressions are lowercased (`_strlwr`) and `TrimSpace`d before parsing.
- **The variable `x` inside an expression is the skill level (FR-3.1/FR-4.1).** The `common` child key *named* `x` is an unrelated skill parameter. It is never bound as the variable.
- **No expression value may be routed through `xml.GetIntegerWithDefault` (FR-7.5).** Expressions are evaluated first; only canonical `strconv.FormatInt` output is written into a synthesized node.
- **A formula failure never aborts the job image or the run (FR-7.2).** It is scoped to the skill, the key, or the key-at-level, logged at ERROR, and counted.
- **Additive only (FR-6.5).** No existing `effect.RestModel` field is renamed, retyped, or removed.
- **Determinism (NFR-4).** Synthesized nodes are populated in declared-table order. No Go map iteration may reach the serialized output.
- **No `// TODO`, stubs, or dead code in landed commits** (project `CLAUDE.md`).
- Repo-relative paths only in committed files — never `/home/<user>/…`.

**Deviations from `design.md`, decided while planning** (rationale in `context.md`):

1. `formula.Expr.EvaluateFloat` from design §3.1 is **not implemented**. Per-operation truncation is intrinsic to the client's machine, so a float-faithful evaluator is a *second* semantics (the client's `GetParsedCommonDataFloat` sibling), not a `float64` variant of this one. Shipping it now would be dead code with a misleading name.
2. Design §3.2's rounding rule is restated precisely: the `u()` ceiling **replaces** the argument's outermost truncation rather than being applied on top of it. Written literally, §3.2 fails the design's own FR-3.4 acceptance case.
3. Run stats are accumulated through a `skill.StatsAccumulator` used by **both** ingest entry points (`data/workers/skill.go` and `data/processor.go`), not only the worker — the design named only the worker, and the second call site would otherwise silently drop the summary.

---

## File Structure

| File | Responsibility |
|---|---|
| `services/atlas-data/atlas.com/data/skill/formula/formula.go` (new) | Tokenizer, recursive-descent parser, evaluator, bounds. Pure; no tenant/context/logger. |
| `services/atlas-data/atlas.com/data/skill/formula/formula_test.go` (new) | Grammar/semantics table tests. |
| `services/atlas-data/atlas.com/data/skill/formula/testdata/common_corpus.csv` (new) | Archive-derived `(expression, level, expected)` regression corpus. |
| `services/atlas-data/atlas.com/data/skill/common.go` (new) | Declared `common` key table, `expandCommon`, per-level `xml.Node` synthesis, failure logging. |
| `services/atlas-data/atlas.com/data/skill/common_test.go` (new) | Expansion, precedence, pass-through, failure-scope, range-check tests. |
| `services/atlas-data/atlas.com/data/skill/stats.go` (new) | `Stats`, `Derivation`, `StatsAccumulator`. |
| `services/atlas-data/atlas.com/data/skill/reader.go` | `produceSkill` gains `common` precedence + stats; unchanged `getEffect` gains one read line per new key. |
| `services/atlas-data/atlas.com/data/skill/effect/model.go`, `effect/rest.go` | 33 new fields + 2 reuses, with builder setters. |
| `services/atlas-data/atlas.com/data/skill/processor.go`, `skill/mock/processor.go` | `Register`/`RegisterSkill` return `Stats`. |
| `services/atlas-data/atlas.com/data/data/workers/skill.go`, `data/processor.go` | Accumulate and log the run summary. |
| `services/atlas-ui/src/services/api/skills.service.ts`, `src/lib/skills/level-table.ts` | Surface the new attributes in the skill level table. |
| `docs/tasks/task-192-v95-skill-common-formulas/archive-census.md` (new) | OQ-5 cross-archive `common` census. |

---

### Task 1: `skill/formula` — the expression evaluator

**Files:**
- Create: `services/atlas-data/atlas.com/data/skill/formula/formula.go`
- Test: `services/atlas-data/atlas.com/data/skill/formula/formula_test.go`

**Interfaces:**
- Consumes: nothing (leaf package; stdlib only).
- Produces:
  - `formula.Parse(src string) (formula.Expr, error)`
  - `func (e formula.Expr) Evaluate(level int) (int64, error)`
  - `func (e formula.Expr) Source() string`

All work happens inside `services/atlas-data/atlas.com/data` (the module root). Run every `go` command from there.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-data/atlas.com/data/skill/formula/formula_test.go`:

```go
package formula

import "testing"

func TestEvaluate(t *testing.T) {
	testCases := []struct {
		name  string
		src   string
		level int
		want  int64
	}{
		// FR-3.4 — division is real, not integer.
		{"ceil half at level 1", "u(x/2)", 1, 1},
		{"floor quarter at level 1", "d(x/4)", 1, 0},
		{"floor quarter at level 4", "d(x/4)", 4, 1},
		// design §1.3 — u/d are trunc-based, not ceil/floor. The 0.999999
		// fudge must not push an exact integer up.
		{"exact integer does not round up", "u(x/5)", 5, 1},
		{"negative floor truncates toward zero", "d(-x/2)", 1, 0},
		{"negative ceiling truncates toward zero", "u(-x/2)", 3, 0},
		// design §1.4 — precedence is + -> - -> / -> *, so `*` binds
		// tighter than `/`. Left-to-right would give 30, not 3.
		{"star binds tighter than slash", "x/2*3", 20, 3},
		// FR-3.5 — unary minus.
		{"bare negative literal", "-2", 1, -2},
		{"leading negative expression", "-10-1*x", 3, -13},
		// FR-3.6 — the single decimal literal in the archive.
		{"decimal truncates at level 1", "0.5*x", 1, 0},
		{"decimal truncates at level 3", "0.5*x", 3, 1},
		{"decimal with addition", "5+0.5*x", 5, 7},
		// FR-3.7 — the one whitespace-bearing value (skill 2111002 damage).
		{"leading space", " 375+5*x", 1, 380},
		// FR-3.9 — maximum observed complexity and longest value.
		{"four operators at level 1", "-1-1*u(x/10)", 1, -2},
		{"four operators at level 20", "-1-1*u(x/10)", 20, -3},
		{"longest value at level 1", "150+50*u(x/10)", 1, 200},
		{"longest value at level 20", "150+50*u(x/10)", 20, 250},
		// FR-3.8 / design §4.4 — deliberate superset of the client.
		{"nested calls", "u(d(x/2))", 5, 2},
		{"bare parentheses truncate", "(x/2)*3", 5, 6},
		// design §1.1 — evaluation is case-insensitive (_strlwr).
		{"uppercase source", "U(X/2)", 1, 1},
		// Constant sources: an int-typed leaf reaches Parse as decimal text.
		{"plain integer", "0", 7, 0},
		{"plain integer one", "1", 7, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tc.src, err)
			}
			got, err := e.Evaluate(tc.level)
			if err != nil {
				t.Fatalf("Evaluate(%d) error = %v, want nil", tc.level, err)
			}
			if got != tc.want {
				t.Fatalf("Parse(%q).Evaluate(%d) = %d, want %d", tc.src, tc.level, got, tc.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	testCases := []struct {
		name string
		src  string
	}{
		{"empty", ""},
		{"blank", "   "},
		{"unknown identifier", "y+1"},
		{"unknown multi-letter identifier", "level+1"},
		{"unknown function", "f(x)"},
		{"function arity two", "u(x,1)"},
		{"unbalanced open paren", "u(x/2"},
		{"unbalanced close paren", "x/2)"},
		{"trailing operator", "x+"},
		{"double operator", "x**2"},
		{"bare function name", "u"},
		{"over length", strings.Repeat("1+", 200) + "1"},
		{"too many tokens", strings.Repeat("1+", 40) + "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.src); err == nil {
				t.Fatalf("Parse(%q) error = nil, want a parse error", tc.src)
			}
		})
	}
}

func TestEvaluateRejectsNonFinite(t *testing.T) {
	e, err := Parse("x/0")
	if err != nil {
		t.Fatalf("Parse error = %v, want nil", err)
	}
	if _, err := e.Evaluate(1); err == nil {
		t.Fatal("Evaluate error = nil, want a non-finite result error")
	}
}

func TestParseOnceEvaluateMany(t *testing.T) {
	e, err := Parse("6+2*u(x/5)")
	if err != nil {
		t.Fatalf("Parse error = %v, want nil", err)
	}
	want := map[int]int64{1: 8, 5: 8, 6: 10, 20: 14}
	for level, w := range want {
		got, err := e.Evaluate(level)
		if err != nil {
			t.Fatalf("Evaluate(%d) error = %v", level, err)
		}
		if got != w {
			t.Fatalf("Evaluate(%d) = %d, want %d", level, got, w)
		}
	}
	if e.Source() != "6+2*u(x/5)" {
		t.Fatalf("Source() = %q, want %q", e.Source(), "6+2*u(x/5)")
	}
}
```

Add `"strings"` to that file's import block (it is used by `TestParseRejects`).

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-data/atlas.com/data`:

```bash
go test ./skill/formula/...
```

Expected: FAIL — the package has no non-test source, so it fails to build with `undefined: Parse`.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-data/atlas.com/data/skill/formula/formula.go`:

```go
// Package formula evaluates the arithmetic expressions stored in Skill.wz
// `common` skill nodes, e.g. mpCon = "6+2*u(x/5)". The only free variable is
// `x`, which is the skill level (1-based).
//
// The semantics implemented here are the GMS v95.0 CLIENT's, read from IDA
// session 79906a1e, GMS_v95.0_U_DEVM.exe.i64:
// SKILLLEVELDATA::GetParsedCommonData (0x6fe560) and
// SKILLLEVELDATA::GetArithmeticData (0x6f9300). They are deliberately NOT the
// textbook semantics. Do not "fix" any of the following:
//
//   - Precedence, loosest binding first, is  +  ->  -  ->  /  ->  * . The
//     client resolves every `*` first, then every `/`, then every `-`, then
//     every `+`, one operator per rewrite pass. So "x/2*3" at x=20 is
//     20/(2*3) = 3, NOT (20/2)*3 = 30.
//   - The client formats each intermediate result with "%d", so EVERY binary
//     operation truncates toward zero before the next one consumes it.
//   - d(v) is trunc(v), not math.Floor(v). u(v) is trunc(v + 0.999999), not
//     math.Ceil(v). They agree for v >= 0 and diverge for negatives, which is
//     why the current archive cannot distinguish them.
//   - The u() ceiling REPLACES the truncation of its argument's outermost
//     operation rather than being applied after it (the client passes
//     bCeiling into the arithmetic pass that spans the whole argument). This
//     is what makes u(x/2) at x=1 equal 1 rather than 0.
//
// Nesting (u(d(...))) and bare parentheses parse correctly here, which is a
// deliberate superset: the client mis-slices nested calls, and the archive
// contains none.
package formula

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Bounds against a malformed or hostile archive (NFR-5). The observed maxima
// are 14 bytes, ~9 tokens and zero nesting; these sit far above that.
const (
	maxSourceLen = 256
	maxTokens    = 64
	maxDepth     = 16
)

// Expr is a parsed, level-independent expression. It is immutable and safe for
// repeated evaluation: parse once per common key, evaluate maxLevel times.
type Expr struct {
	root node
	src  string
}

// Source returns the verbatim source the expression was parsed from, for error
// reporting.
func (e Expr) Source() string { return e.src }

// Evaluate computes the expression with the free variable x bound to level,
// and truncates the result toward zero.
func (e Expr) Evaluate(level int) (int64, error) {
	if e.root == nil {
		return 0, fmt.Errorf("formula: evaluate the zero Expr")
	}
	v := e.root.eval(float64(level), false)
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("formula: %q at level %d is not finite", e.src, level)
	}
	v = math.Trunc(v)
	if v > math.MaxInt64 || v < math.MinInt64 {
		return 0, fmt.Errorf("formula: %q at level %d overflows int64", e.src, level)
	}
	return int64(v), nil
}

// Parse tokenizes and parses src. It is pure: no tenant, no context, no logger.
func Parse(src string) (Expr, error) {
	normalized := strings.ToLower(strings.TrimSpace(src))
	if normalized == "" {
		return Expr{}, fmt.Errorf("formula: empty expression")
	}
	if len(normalized) > maxSourceLen {
		return Expr{}, fmt.Errorf("formula: expression of %d bytes exceeds the %d byte limit", len(normalized), maxSourceLen)
	}
	toks, err := tokenize(normalized)
	if err != nil {
		return Expr{}, err
	}
	p := &parser{toks: toks}
	root, err := p.parseAdd()
	if err != nil {
		return Expr{}, err
	}
	if p.pos != len(p.toks) {
		return Expr{}, fmt.Errorf("formula: unexpected trailing token at offset %d in %q", p.toks[p.pos].pos, src)
	}
	return Expr{root: root, src: src}, nil
}

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokVar
	tokFunc
	tokLParen
	tokRParen
	tokPlus
	tokMinus
	tokStar
	tokSlash
)

type token struct {
	kind tokenKind
	num  float64
	fn   byte // 'u' or 'd', only for tokFunc
	pos  int
}

func tokenize(src string) ([]token, error) {
	toks := make([]token, 0, 16)
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t':
			i++
			continue
		case c == '+':
			toks, i = append(toks, token{kind: tokPlus, pos: i}), i+1
		case c == '-':
			toks, i = append(toks, token{kind: tokMinus, pos: i}), i+1
		case c == '*':
			toks, i = append(toks, token{kind: tokStar, pos: i}), i+1
		case c == '/':
			toks, i = append(toks, token{kind: tokSlash, pos: i}), i+1
		case c == '(':
			toks, i = append(toks, token{kind: tokLParen, pos: i}), i+1
		case c == ')':
			toks, i = append(toks, token{kind: tokRParen, pos: i}), i+1
		case c >= '0' && c <= '9':
			j := i
			for j < len(src) && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			if j < len(src) && src[j] == '.' {
				j++
				for j < len(src) && src[j] >= '0' && src[j] <= '9' {
					j++
				}
			}
			v, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("formula: bad number %q at offset %d", src[i:j], i)
			}
			toks, i = append(toks, token{kind: tokNumber, num: v, pos: i}), j
		case c == 'x':
			toks, i = append(toks, token{kind: tokVar, pos: i}), i+1
		case c == 'u' || c == 'd':
			// A function name is legal only immediately before '('. There is
			// no symbol table: any other identifier is a parse error.
			if i+1 >= len(src) || src[i+1] != '(' {
				return nil, fmt.Errorf("formula: unknown identifier %q at offset %d", string(c), i)
			}
			toks, i = append(toks, token{kind: tokFunc, fn: c, pos: i}), i+1
		default:
			return nil, fmt.Errorf("formula: unexpected character %q at offset %d", string(c), i)
		}
		if len(toks) > maxTokens {
			return nil, fmt.Errorf("formula: more than %d tokens", maxTokens)
		}
	}
	if len(toks) == 0 {
		return nil, fmt.Errorf("formula: empty expression")
	}
	return toks, nil
}

// round mirrors the client's per-operation rounding. ceiling is true only for
// the operation that spans an entire u() argument.
func round(v float64, ceiling bool) float64 {
	if ceiling {
		return math.Trunc(v + 0.999999)
	}
	return math.Trunc(v)
}

// node is an AST node. eval receives the level as the binding for x, and
// ceiling reporting whether THIS node's own outermost operation applies the
// u() ceiling instead of truncation.
type node interface {
	eval(level float64, ceiling bool) float64
}

// numNode and varNode are atoms: the client's arithmetic pass finds no
// operator in a bare atom and returns it via atoi, so no rounding applies.
type numNode float64

func (n numNode) eval(_ float64, _ bool) float64 { return float64(n) }

type varNode struct{}

func (varNode) eval(level float64, _ bool) float64 { return level }

type negNode struct{ operand node }

func (n negNode) eval(level float64, ceiling bool) float64 {
	return -n.operand.eval(level, ceiling)
}

type binNode struct {
	op       tokenKind
	lhs, rhs node
}

func (n binNode) eval(level float64, ceiling bool) float64 {
	l := n.lhs.eval(level, false)
	r := n.rhs.eval(level, false)
	var v float64
	switch n.op {
	case tokPlus:
		v = l + r
	case tokMinus:
		v = l - r
	case tokStar:
		v = l * r
	case tokSlash:
		v = l / r
	}
	return round(v, ceiling)
}

// callNode is u(...), d(...) or a bare parenthesized group. u() evaluates its
// argument with the ceiling flag set; d() and bare parentheses truncate.
type callNode struct {
	ceiling bool
	arg     node
}

func (n callNode) eval(level float64, ceiling bool) float64 {
	return round(n.arg.eval(level, n.ceiling), ceiling)
}

type parser struct {
	toks  []token
	pos   int
	depth int
}

func (p *parser) peek() (token, bool) {
	if p.pos >= len(p.toks) {
		return token{}, false
	}
	return p.toks[p.pos], true
}

// Precedence tiers, loosest first: + then - then / then *. See the package
// doc comment; this ordering is the client's and is not a typo.
func (p *parser) parseAdd() (node, error) { return p.parseTier(tokPlus, p.parseSub) }
func (p *parser) parseSub() (node, error) { return p.parseTier(tokMinus, p.parseDiv) }
func (p *parser) parseDiv() (node, error) { return p.parseTier(tokSlash, p.parseMul) }
func (p *parser) parseMul() (node, error) { return p.parseTier(tokStar, p.parseUnary) }

func (p *parser) parseTier(op tokenKind, next func() (node, error)) (node, error) {
	lhs, err := next()
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.peek()
		if !ok || t.kind != op {
			return lhs, nil
		}
		p.pos++
		rhs, err := next()
		if err != nil {
			return nil, err
		}
		lhs = binNode{op: op, lhs: lhs, rhs: rhs}
	}
}

func (p *parser) parseUnary() (node, error) {
	if t, ok := p.peek(); ok && t.kind == tokMinus {
		p.pos++
		operand, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		return negNode{operand: operand}, nil
	}
	return p.parseAtom()
}

func (p *parser) parseAtom() (node, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("formula: unexpected end of expression")
	}
	switch t.kind {
	case tokNumber:
		p.pos++
		return numNode(t.num), nil
	case tokVar:
		p.pos++
		return varNode{}, nil
	case tokFunc:
		p.pos++
		arg, err := p.parseGroup()
		if err != nil {
			return nil, err
		}
		return callNode{ceiling: t.fn == 'u', arg: arg}, nil
	case tokLParen:
		arg, err := p.parseGroup()
		if err != nil {
			return nil, err
		}
		return callNode{ceiling: false, arg: arg}, nil
	default:
		return nil, fmt.Errorf("formula: unexpected token at offset %d", t.pos)
	}
}

// parseGroup consumes '(' expr ')'. Arity is fixed at 1: a ',' never
// tokenizes, so u(x,1) fails as an unexpected character.
func (p *parser) parseGroup() (node, error) {
	p.depth++
	if p.depth > maxDepth {
		return nil, fmt.Errorf("formula: nesting deeper than %d", maxDepth)
	}
	defer func() { p.depth-- }()

	t, ok := p.peek()
	if !ok || t.kind != tokLParen {
		return nil, fmt.Errorf("formula: expected '(' at offset %d", p.pos)
	}
	p.pos++
	inner, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	t, ok = p.peek()
	if !ok || t.kind != tokRParen {
		return nil, fmt.Errorf("formula: unbalanced parenthesis")
	}
	p.pos++
	return inner, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./skill/formula/...
```

Expected: PASS (`ok atlas-data/skill/formula`).

If `TestEvaluate/star_binds_tighter_than_slash` fails with 30, the tier ordering was written textbook-style — re-read the package doc comment. If `ceil half at level 1` yields 0, the ceiling flag is being applied *after* the argument truncated rather than replacing it — check `binNode.eval`'s use of the `ceiling` parameter.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/formula
git commit -m "feat(atlas-data): add skill common formula evaluator"
```

---

### Task 2: `effect` model — the 35 `common` keys

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/effect/model.go`
- Modify: `services/atlas-data/atlas.com/data/skill/effect/rest.go`
- Test: `services/atlas-data/atlas.com/data/skill/effect/model_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: 35 builder setters on `*effect.ModelBuilder`, each returning `*ModelBuilder` — `SetRange`, `SetMastery`, `SetZ`, `SetDot`, `SetCr`, `SetDotInterval`, `SetDotTime`, `SetDamR`, `SetCriticaldamageMin`, `SetMHPRRate`, `SetV`, `SetIgnoreMobpdpR`, `SetEpad`, `SetW`, `SetU`, `SetEpdd`, `SetEmdd`, `SetSelfDestruction`, `SetAsrR`, `SetMMPRRate`, `SetT`, `SetEr`, `SetPddR`, `SetTerR`, `SetMadX`, `SetSubProp`, `SetEmhp`, `SetCriticaldamageMax`, `SetExpR`, `SetEmmp`, `SetConsumeItemId`, `SetMddR`, `SetSubTime`, `SetPadX`, `SetMesoR`. All take `int32` except `SetMHPRRate`/`SetMMPRRate`, which take `uint16`.

Naming rule (design §5.1): **Go field = PascalCase of the wz key; JSON tag = the wz key verbatim.** No key gets a "descriptive" name — their meanings are unverified (OQ-4) and a guessed name becomes permanent. The single exception is `itemConsume` (see Step 3).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-data/atlas.com/data/skill/effect/model_test.go`:

```go
package effect

import (
	"encoding/json"
	"testing"
)

// TestCommonKeyJSONTags pins the JSON attribute name of every field added for
// a Skill.wz `common` key. The tag is the wz key verbatim (design §5.1) so
// atlas-ui and any other consumer can address it by its archive name.
func TestCommonKeyJSONTags(t *testing.T) {
	rm := NewModelBuilder().
		SetRange(1).SetMastery(2).SetZ(3).SetDot(4).SetCr(5).
		SetDotInterval(6).SetDotTime(7).SetDamR(8).SetCriticaldamageMin(9).
		SetMHPRRate(10).SetV(11).SetIgnoreMobpdpR(12).SetEpad(13).SetW(14).
		SetU(15).SetEpdd(16).SetEmdd(17).SetSelfDestruction(18).SetAsrR(19).
		SetMMPRRate(20).SetT(21).SetEr(22).SetPddR(23).SetTerR(24).
		SetMadX(25).SetSubProp(26).SetEmhp(27).SetCriticaldamageMax(28).
		SetExpR(29).SetEmmp(30).SetConsumeItemId(31).SetMddR(32).
		SetSubTime(33).SetPadX(34).SetMesoR(35).
		Build()

	b, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	want := map[string]float64{
		"range": 1, "mastery": 2, "z": 3, "dot": 4, "cr": 5,
		"dotInterval": 6, "dotTime": 7, "damR": 8, "criticaldamageMin": 9,
		"MHPRRate": 10, "v": 11, "ignoreMobpdpR": 12, "epad": 13, "w": 14,
		"u": 15, "epdd": 16, "emdd": 17, "selfDestruction": 18, "asrR": 19,
		"MMPRRate": 20, "t": 21, "er": 22, "pddR": 23, "terR": 24,
		"madX": 25, "subProp": 26, "emhp": 27, "criticaldamageMax": 28,
		"expR": 29, "emmp": 30, "consumeItemId": 31, "mddR": 32,
		"subTime": 33, "padX": 34, "mesoR": 35,
	}
	for key, w := range want {
		v, ok := got[key]
		if !ok {
			t.Fatalf("marshalled effect has no %q attribute", key)
		}
		if v != w {
			t.Fatalf("attribute %q = %v, want %v", key, v, w)
		}
	}
}

// TestExistingItemConsumeUnchanged pins FR-6.4: the pre-existing
// `itemConsume` attribute is fed by wz `itemCon` and must NOT be repurposed
// for wz `common/itemConsume`, which lands on `consumeItemId`.
func TestExistingItemConsumeUnchanged(t *testing.T) {
	rm := NewModelBuilder().SetItemConsume(2000000).SetConsumeItemId(2331000).Build()
	if rm.ItemConsume != 2000000 {
		t.Fatalf("ItemConsume = %d, want 2000000", rm.ItemConsume)
	}
	if rm.ConsumeItemId != 2331000 {
		t.Fatalf("ConsumeItemId = %d, want 2331000", rm.ConsumeItemId)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./skill/effect/...
```

Expected: FAIL to build — `rm.SetRange undefined (type *ModelBuilder has no field or method SetRange)` and similar.

- [ ] **Step 3: Add the fields to `rest.go`**

In `services/atlas-data/atlas.com/data/skill/effect/rest.go`, append these fields to `RestModel` immediately **after** `CardStats` (keeping every existing field in place, FR-6.5):

```go
	// Fields below are Skill.wz `common` keys (task-192). Go name is the
	// PascalCase of the wz key and the JSON tag is the wz key verbatim; the
	// semantics of most of them are unverified, so no key is given an
	// invented descriptive name. Populated from both the `common` and
	// `level` read paths.
	Range             int32 `json:"range"`
	Mastery           int32 `json:"mastery"`
	Z                 int32 `json:"z"`
	Dot               int32 `json:"dot"`
	Cr                int32 `json:"cr"`
	DotInterval       int32 `json:"dotInterval"`
	DotTime           int32 `json:"dotTime"`
	DamR              int32 `json:"damR"`
	CriticaldamageMin int32 `json:"criticaldamageMin"`
	V                 int32 `json:"v"`
	IgnoreMobpdpR     int32 `json:"ignoreMobpdpR"`
	Epad              int32 `json:"epad"`
	W                 int32 `json:"w"`
	U                 int32 `json:"u"`
	Epdd              int32 `json:"epdd"`
	Emdd              int32 `json:"emdd"`
	SelfDestruction   int32 `json:"selfDestruction"`
	AsrR              int32 `json:"asrR"`
	T                 int32 `json:"t"`
	Er                int32 `json:"er"`
	PddR              int32 `json:"pddR"`
	TerR              int32 `json:"terR"`
	MadX              int32 `json:"madX"`
	SubProp           int32 `json:"subProp"`
	Emhp              int32 `json:"emhp"`
	CriticaldamageMax int32 `json:"criticaldamageMax"`
	ExpR              int32 `json:"expR"`
	Emmp              int32 `json:"emmp"`
	// ConsumeItemId is wz `common/itemConsume`. It is deliberately NOT the
	// `itemConsume` JSON attribute above, which is wz `itemCon` — the two are
	// distinct keys that never co-occur, and folding them would silently
	// merge two differently-sourced values (FR-6.4, design §5.4).
	ConsumeItemId int32 `json:"consumeItemId"`
	MddR          int32 `json:"mddR"`
	SubTime       int32 `json:"subTime"`
	PadX          int32 `json:"padX"`
	MesoR         int32 `json:"mesoR"`
```

`mhpR` and `mmpR` reuse the existing `MHPRRate` / `MMPRRate` fields (design §5.3) — do **not** add fields for them.

- [ ] **Step 4: Add the builder fields, setters and `Build()` lines to `model.go`**

In `ModelBuilder`, append these private fields after `monsterStatus`:

```go
	rangeValue        int32
	mastery           int32
	z                 int32
	dot               int32
	cr                int32
	dotInterval       int32
	dotTime           int32
	damR              int32
	criticaldamageMin int32
	v                 int32
	ignoreMobpdpR     int32
	epad              int32
	w                 int32
	u                 int32
	epdd              int32
	emdd              int32
	selfDestruction   int32
	asrR              int32
	t                 int32
	er                int32
	pddR              int32
	terR              int32
	madX              int32
	subProp           int32
	emhp              int32
	criticaldamageMax int32
	expR              int32
	emmp              int32
	consumeItemId     int32
	mddR              int32
	subTime           int32
	padX              int32
	mesoR             int32
```

(`rangeValue` rather than `range`, which is a Go keyword. `mhprRate` / `mmprRate` already exist.)

Append the setters at the end of the file:

```go
func (b *ModelBuilder) SetRange(v int32) *ModelBuilder             { b.rangeValue = v; return b }
func (b *ModelBuilder) SetMastery(v int32) *ModelBuilder           { b.mastery = v; return b }
func (b *ModelBuilder) SetZ(v int32) *ModelBuilder                 { b.z = v; return b }
func (b *ModelBuilder) SetDot(v int32) *ModelBuilder               { b.dot = v; return b }
func (b *ModelBuilder) SetCr(v int32) *ModelBuilder                { b.cr = v; return b }
func (b *ModelBuilder) SetDotInterval(v int32) *ModelBuilder       { b.dotInterval = v; return b }
func (b *ModelBuilder) SetDotTime(v int32) *ModelBuilder           { b.dotTime = v; return b }
func (b *ModelBuilder) SetDamR(v int32) *ModelBuilder              { b.damR = v; return b }
func (b *ModelBuilder) SetCriticaldamageMin(v int32) *ModelBuilder { b.criticaldamageMin = v; return b }
func (b *ModelBuilder) SetMHPRRate(v uint16) *ModelBuilder         { b.mhprRate = v; return b }
func (b *ModelBuilder) SetV(v int32) *ModelBuilder                 { b.v = v; return b }
func (b *ModelBuilder) SetIgnoreMobpdpR(v int32) *ModelBuilder     { b.ignoreMobpdpR = v; return b }
func (b *ModelBuilder) SetEpad(v int32) *ModelBuilder              { b.epad = v; return b }
func (b *ModelBuilder) SetW(v int32) *ModelBuilder                 { b.w = v; return b }
func (b *ModelBuilder) SetU(v int32) *ModelBuilder                 { b.u = v; return b }
func (b *ModelBuilder) SetEpdd(v int32) *ModelBuilder              { b.epdd = v; return b }
func (b *ModelBuilder) SetEmdd(v int32) *ModelBuilder              { b.emdd = v; return b }
func (b *ModelBuilder) SetSelfDestruction(v int32) *ModelBuilder   { b.selfDestruction = v; return b }
func (b *ModelBuilder) SetAsrR(v int32) *ModelBuilder              { b.asrR = v; return b }
func (b *ModelBuilder) SetMMPRRate(v uint16) *ModelBuilder         { b.mmprRate = v; return b }
func (b *ModelBuilder) SetT(v int32) *ModelBuilder                 { b.t = v; return b }
func (b *ModelBuilder) SetEr(v int32) *ModelBuilder                { b.er = v; return b }
func (b *ModelBuilder) SetPddR(v int32) *ModelBuilder              { b.pddR = v; return b }
func (b *ModelBuilder) SetTerR(v int32) *ModelBuilder              { b.terR = v; return b }
func (b *ModelBuilder) SetMadX(v int32) *ModelBuilder              { b.madX = v; return b }
func (b *ModelBuilder) SetSubProp(v int32) *ModelBuilder           { b.subProp = v; return b }
func (b *ModelBuilder) SetEmhp(v int32) *ModelBuilder              { b.emhp = v; return b }
func (b *ModelBuilder) SetCriticaldamageMax(v int32) *ModelBuilder { b.criticaldamageMax = v; return b }
func (b *ModelBuilder) SetExpR(v int32) *ModelBuilder              { b.expR = v; return b }
func (b *ModelBuilder) SetEmmp(v int32) *ModelBuilder              { b.emmp = v; return b }

// SetConsumeItemId sets wz `common/itemConsume`. See RestModel.ConsumeItemId:
// this is NOT the same key as `itemCon`, which SetItemConsume carries.
func (b *ModelBuilder) SetConsumeItemId(v int32) *ModelBuilder { b.consumeItemId = v; return b }

func (b *ModelBuilder) SetMddR(v int32) *ModelBuilder    { b.mddR = v; return b }
func (b *ModelBuilder) SetSubTime(v int32) *ModelBuilder { b.subTime = v; return b }
func (b *ModelBuilder) SetPadX(v int32) *ModelBuilder    { b.padX = v; return b }
func (b *ModelBuilder) SetMesoR(v int32) *ModelBuilder   { b.mesoR = v; return b }
```

In `Build()`'s returned `RestModel` literal, append after `RB: rbPtr,`:

```go
		Range:             b.rangeValue,
		Mastery:           b.mastery,
		Z:                 b.z,
		Dot:               b.dot,
		Cr:                b.cr,
		DotInterval:       b.dotInterval,
		DotTime:           b.dotTime,
		DamR:              b.damR,
		CriticaldamageMin: b.criticaldamageMin,
		V:                 b.v,
		IgnoreMobpdpR:     b.ignoreMobpdpR,
		Epad:              b.epad,
		W:                 b.w,
		U:                 b.u,
		Epdd:              b.epdd,
		Emdd:              b.emdd,
		SelfDestruction:   b.selfDestruction,
		AsrR:              b.asrR,
		T:                 b.t,
		Er:                b.er,
		PddR:              b.pddR,
		TerR:              b.terR,
		MadX:              b.madX,
		SubProp:           b.subProp,
		Emhp:              b.emhp,
		CriticaldamageMax: b.criticaldamageMax,
		ExpR:              b.expR,
		Emmp:              b.emmp,
		ConsumeItemId:     b.consumeItemId,
		MddR:              b.mddR,
		SubTime:           b.subTime,
		PadX:              b.padX,
		MesoR:             b.mesoR,
```

`MHPRRate` and `MMPRRate` are already wired in `Build()` — leave those lines alone.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test ./skill/...
```

Expected: PASS. The pre-existing `skill` package tests must still pass unchanged — no existing attribute changed name, type, or value.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/effect
git commit -m "feat(atlas-data): model the 35 Skill.wz common effect keys"
```

---

### Task 3: `getEffect` reads the new keys

**Files:**
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go` (in `getEffect`, after the existing `SetMoveTo(...)` chain ending at line ~247)
- Test: `services/atlas-data/atlas.com/data/skill/reader_test.go` (append one test)

**Interfaces:**
- Consumes: the Task 2 setters.
- Produces: nothing new — but this is the change that makes FR-6.1 ("populated on **both** read paths") true, because Task 4 routes `common` through this same function.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/skill/reader_test.go`:

```go
// TestLevelPathPopulatesCommonKeys pins FR-6.1 from the `level` side: the
// keys added for `common` are read by the one shared getEffect, so a `level`
// node that happens to carry them populates them too.
func TestLevelPathPopulatesCommonKeys(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="112.img">
  <imgdir name="skill">
    <imgdir name="1121000">
      <imgdir name="level">
        <imgdir name="1">
          <int name="mastery" value="40"/>
          <int name="range" value="150"/>
          <int name="dot" value="12"/>
          <int name="dotInterval" value="2"/>
          <int name="dotTime" value="8"/>
          <int name="mhpR" value="5"/>
          <int name="mmpR" value="6"/>
          <int name="itemConsume" value="2331000"/>
          <int name="itemCon" value="2000000"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _ := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), tn)

	d, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(d.Models))
	}
	ef := d.Models[0].Effects[0]
	if ef.Mastery != 40 {
		t.Fatalf("Mastery = %d, want 40", ef.Mastery)
	}
	if ef.Range != 150 {
		t.Fatalf("Range = %d, want 150", ef.Range)
	}
	if ef.Dot != 12 || ef.DotInterval != 2 || ef.DotTime != 8 {
		t.Fatalf("dot triple = (%d,%d,%d), want (12,2,8)", ef.Dot, ef.DotInterval, ef.DotTime)
	}
	if ef.MHPRRate != 5 || ef.MMPRRate != 6 {
		t.Fatalf("(MHPRRate,MMPRRate) = (%d,%d), want (5,6)", ef.MHPRRate, ef.MMPRRate)
	}
	if ef.ConsumeItemId != 2331000 {
		t.Fatalf("ConsumeItemId = %d, want 2331000", ef.ConsumeItemId)
	}
	if ef.ItemConsume != 2000000 {
		t.Fatalf("ItemConsume = %d, want 2000000", ef.ItemConsume)
	}
}
```

This test uses `d.Models`, which does not exist until Task 5 changes `Read`'s return type. **Until Task 5 lands, write it against today's shape** — replace the two `Read(...)()` lines with:

```go
	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
	models, err := rms()
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(models))
	}
	ef := models[0].Effects[0]
```

Task 5 converts it, along with every other call site, in one mechanical pass.

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./skill/ -run TestLevelPathPopulatesCommonKeys
```

Expected: FAIL — `Mastery = 0, want 40`.

- [ ] **Step 3: Add the read lines**

In `getEffect` (`skill/reader.go`), immediately after the chain ending `SetMoveTo(node.GetIntegerWithDefault("moveTo", -1))` and before `ms := make(map[string]uint32)`, insert:

```go
	// Skill.wz `common` keys (task-192). Read here, in the single shared
	// getEffect, so they populate on BOTH the `common` and `level` paths
	// (FR-6.1). Absent keys default to 0, matching the `level` path's rule
	// for every key it does not find.
	e.SetRange(node.GetIntegerWithDefault("range", 0)).
		SetMastery(node.GetIntegerWithDefault("mastery", 0)).
		SetZ(node.GetIntegerWithDefault("z", 0)).
		SetDot(node.GetIntegerWithDefault("dot", 0)).
		SetCr(node.GetIntegerWithDefault("cr", 0)).
		SetDotInterval(node.GetIntegerWithDefault("dotInterval", 0)).
		SetDotTime(node.GetIntegerWithDefault("dotTime", 0)).
		SetDamR(node.GetIntegerWithDefault("damR", 0)).
		SetCriticaldamageMin(node.GetIntegerWithDefault("criticaldamageMin", 0)).
		SetMHPRRate(uint16(node.GetIntegerWithDefault("mhpR", 0))).
		SetV(node.GetIntegerWithDefault("v", 0)).
		SetIgnoreMobpdpR(node.GetIntegerWithDefault("ignoreMobpdpR", 0)).
		SetEpad(node.GetIntegerWithDefault("epad", 0)).
		SetW(node.GetIntegerWithDefault("w", 0)).
		SetU(node.GetIntegerWithDefault("u", 0)).
		SetEpdd(node.GetIntegerWithDefault("epdd", 0)).
		SetEmdd(node.GetIntegerWithDefault("emdd", 0)).
		SetSelfDestruction(node.GetIntegerWithDefault("selfDestruction", 0)).
		SetAsrR(node.GetIntegerWithDefault("asrR", 0)).
		SetMMPRRate(uint16(node.GetIntegerWithDefault("mmpR", 0))).
		SetT(node.GetIntegerWithDefault("t", 0)).
		SetEr(node.GetIntegerWithDefault("er", 0)).
		SetPddR(node.GetIntegerWithDefault("pddR", 0)).
		SetTerR(node.GetIntegerWithDefault("terR", 0)).
		SetMadX(node.GetIntegerWithDefault("madX", 0)).
		SetSubProp(node.GetIntegerWithDefault("subProp", 0)).
		SetEmhp(node.GetIntegerWithDefault("emhp", 0)).
		SetCriticaldamageMax(node.GetIntegerWithDefault("criticaldamageMax", 0)).
		SetExpR(node.GetIntegerWithDefault("expR", 0)).
		SetEmmp(node.GetIntegerWithDefault("emmp", 0)).
		SetConsumeItemId(node.GetIntegerWithDefault("itemConsume", 0)).
		SetMddR(node.GetIntegerWithDefault("mddR", 0)).
		SetSubTime(node.GetIntegerWithDefault("subTime", 0)).
		SetPadX(node.GetIntegerWithDefault("padX", 0)).
		SetMesoR(node.GetIntegerWithDefault("mesoR", 0))
```

Note `SetConsumeItemId` reads the wz key `itemConsume`, while the pre-existing `SetItemConsume` a few lines above reads `itemCon`. That asymmetry is deliberate (FR-6.4).

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./skill/...
```

Expected: PASS, including every pre-existing reader test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/reader.go services/atlas-data/atlas.com/data/skill/reader_test.go
git commit -m "feat(atlas-data): read the new common keys in getEffect"
```

---

### Task 4: `common` detection, the key table, and per-level expansion

**Files:**
- Create: `services/atlas-data/atlas.com/data/skill/common.go`
- Create: `services/atlas-data/atlas.com/data/skill/common_test.go`
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go` (`produceSkill`, `Read`)

**Interfaces:**
- Consumes: `formula.Parse` / `Expr.Evaluate` (Task 1); `getEffect` (unmodified).
- Produces:
  - `func expandCommon(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, buff bool, common *xml.Node) ([]effect.RestModel, uint8, Stats)`
  - `type Stats struct { Processed, FromCommon, FromLevel, Neither, SkillsWithFailures, Failures int }` with `func (s *Stats) Add(o Stats)` — Task 5 moves this type to `stats.go` and plumbs it outward; define it here now so this task compiles and is testable on its own.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-data/atlas.com/data/skill/common_test.go`:

```go
package skill

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"atlas-data/xml"
)

func commonTestContext(t *testing.T) (logrus.FieldLogger, *test.Hook, context.Context) {
	t.Helper()
	l, hook := test.NewNullLogger()
	tn, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	return l, hook, tenant.WithContext(context.Background(), tn)
}

func readSkills(t *testing.T, l logrus.FieldLogger, ctx context.Context, data string) []RestModel {
	t.Helper()
	models, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(data)))()
	if err != nil {
		t.Fatal(err)
	}
	return models
}

// TestCommonExpansion covers FR-5.1/FR-5.2: a common node expands to exactly
// maxLevel effects, evaluated at x = 1..maxLevel, and MaxLevel is the declared
// value. Values are skill 1001003's real v95.1 expressions.
func TestCommonExpansion(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1001003">
      <imgdir name="common">
        <int name="maxLevel" value="20"/>
        <string name="mpCon" value="6+2*u(x/5)"/>
        <string name="time" value="100+10*x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	models := readSkills(t, l, ctx, xmlData)
	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(models))
	}
	m := models[0]
	if m.MaxLevel != 20 {
		t.Fatalf("MaxLevel = %d, want 20", m.MaxLevel)
	}
	if len(m.Effects) != 20 {
		t.Fatalf("len(Effects) = %d, want 20", len(m.Effects))
	}
	// 6+2*u(1/5) = 6 + 2*1 = 8; time 100+10*1 = 110 s -> 110000 ms.
	if m.Effects[0].MPConsume != 8 {
		t.Fatalf("Effects[0].MPConsume = %d, want 8", m.Effects[0].MPConsume)
	}
	if m.Effects[0].Duration != 110000 {
		t.Fatalf("Effects[0].Duration = %d, want 110000", m.Effects[0].Duration)
	}
	// 6+2*u(20/5) = 6 + 2*4 = 14; time 100+10*20 = 300 s -> 300000 ms.
	if m.Effects[19].MPConsume != 14 {
		t.Fatalf("Effects[19].MPConsume = %d, want 14", m.Effects[19].MPConsume)
	}
	if m.Effects[19].Duration != 300000 {
		t.Fatalf("Effects[19].Duration = %d, want 300000", m.Effects[19].Duration)
	}
}

// TestCommonXKeyIsNotTheVariable pins FR-4.1: the common child named `x` is a
// skill parameter, never the expression variable. Modelled on skill 1101004,
// whose common/x is the literal "-2".
func TestCommonXKeyIsNotTheVariable(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="110.img">
  <imgdir name="skill">
    <imgdir name="1101004">
      <imgdir name="common">
        <int name="maxLevel" value="3"/>
        <string name="x" value="-2"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	for i, ef := range m.Effects {
		if ef.X != -2 {
			t.Fatalf("Effects[%d].X = %d, want -2 at every level", i, ef.X)
		}
		if ef.MPConsume != uint16(i+1) {
			t.Fatalf("Effects[%d].MPConsume = %d, want %d (the level)", i, ef.MPConsume, i+1)
		}
	}
}

// TestCommonWinsOverLevel pins FR-1.2 and its accepted regression: skills
// 2211002/2211006 carry both subtrees; common wins, so maxLevel is 20 (not
// 30) and the level-only keys mad/mastery are lost.
func TestCommonWinsOverLevel(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2211.img">
  <imgdir name="skill">
    <imgdir name="2211002">
      <imgdir name="common">
        <int name="maxLevel" value="20"/>
        <string name="mpCon" value="20+6*d(x/4)"/>
        <string name="damage" value="100+2*x"/>
      </imgdir>
      <imgdir name="level">
        <imgdir name="1">
          <int name="mpCon" value="21"/>
          <int name="mad" value="120"/>
          <int name="mastery" value="40"/>
        </imgdir>
        <imgdir name="2">
          <int name="mpCon" value="22"/>
          <int name="mad" value="125"/>
          <int name="mastery" value="45"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	if m.MaxLevel != 20 {
		t.Fatalf("MaxLevel = %d, want 20 (common wins over level's 2)", m.MaxLevel)
	}
	if len(m.Effects) != 20 {
		t.Fatalf("len(Effects) = %d, want 20", len(m.Effects))
	}
	// 20+6*d(1/4) = 20 + 6*0 = 20, NOT the level subtree's 21.
	if m.Effects[0].MPConsume != 20 {
		t.Fatalf("Effects[0].MPConsume = %d, want 20", m.Effects[0].MPConsume)
	}
	// Accepted regression: mad/mastery live only under `level` and are lost.
	if m.Effects[0].MagicAttack != 0 {
		t.Fatalf("Effects[0].MagicAttack = %d, want 0 (accepted FR-1.2 regression)", m.Effects[0].MagicAttack)
	}
	if m.Effects[0].Mastery != 0 {
		t.Fatalf("Effects[0].Mastery = %d, want 0 (accepted FR-1.2 regression)", m.Effects[0].Mastery)
	}
}

// TestCommonMaxLevelNodeTypes covers FR-2.3: maxLevel is an int node 632x and
// a string node 3x (13100004, 1320005, 32120001). Both must parse.
func TestCommonMaxLevelNodeTypes(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="1310.img">
  <imgdir name="skill">
    <imgdir name="13100004">
      <imgdir name="common">
        <string name="maxLevel" value="10"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
    <imgdir name="13100005">
      <imgdir name="common">
        <int name="maxLevel" value="5"/>
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	got := map[string]uint8{}
	for _, m := range readSkills(t, l, ctx, xmlData) {
		got[strconv.Itoa(int(m.Id))] = m.MaxLevel
	}
	if got["13100004"] != 10 {
		t.Fatalf("13100004 MaxLevel = %d, want 10 (string-typed node)", got["13100004"])
	}
	if got["13100005"] != 5 {
		t.Fatalf("13100005 MaxLevel = %d, want 5 (int-typed node)", got["13100005"])
	}
}

// TestCommonIntLeavesAndPassThrough covers FR-2.4/2.5/2.6: int-typed leaves
// parse as constants, lt/rb vectors pass through unevaluated, and `action` is
// never evaluated.
func TestCommonIntLeavesAndPassThrough(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="311.img">
  <imgdir name="skill">
    <imgdir name="3111003">
      <imgdir name="common">
        <int name="maxLevel" value="2"/>
        <int name="time" value="0"/>
        <int name="x" value="1"/>
        <string name="action" value="slashStorm2"/>
        <vector name="lt" x="-150" y="-100"/>
        <vector name="rb" x="150" y="100"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	for i, ef := range m.Effects {
		if ef.X != 1 {
			t.Fatalf("Effects[%d].X = %d, want 1 (int leaf)", i, ef.X)
		}
		if ef.LT == nil || ef.LT.X != -150 || ef.LT.Y != -100 {
			t.Fatalf("Effects[%d].LT = %+v, want (-150,-100)", i, ef.LT)
		}
		if ef.RB == nil || ef.RB.X != 150 || ef.RB.Y != 100 {
			t.Fatalf("Effects[%d].RB = %+v, want (150,100)", i, ef.RB)
		}
		// time 0 is > -1, so getEffect converts seconds -> ms and flags overTime.
		if ef.Duration != 0 || !ef.OverTime {
			t.Fatalf("Effects[%d] duration/overTime = (%d,%v), want (0,true)", i, ef.Duration, ef.OverTime)
		}
	}
}

// TestCommonNeitherSubtree covers FR-1.3: a skill with neither subtree yields
// zero effects and does not panic.
func TestCommonNeitherSubtree(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1000000">
      <int name="skillType" value="0"/>
    </imgdir>
  </imgdir>
</imgdir>`

	l, _, ctx := commonTestContext(t)
	m := readSkills(t, l, ctx, xmlData)[0]
	if m.MaxLevel != 0 || len(m.Effects) != 0 {
		t.Fatalf("(MaxLevel,len(Effects)) = (%d,%d), want (0,0)", m.MaxLevel, len(m.Effects))
	}
}

// TestSynthesizedNodesAreCanonicalIntegers pins FR-7.5 / design §2.2: no
// expression text ever reaches xml.GetIntegerWithDefault, because synthesis
// writes only canonical strconv output.
func TestSynthesizedNodesAreCanonicalIntegers(t *testing.T) {
	common := xml.Node{
		Name: "common",
		IntegerNodes: []xml.IntegerNode{
			{Name: "maxLevel", Value: "3"},
		},
		StringNodes: []xml.StringNode{
			{Name: "mpCon", Value: "6+2*u(x/5)"},
			{Name: "damage", Value: " 375+5*x"},
			{Name: "x", Value: "-2"},
		},
	}
	l, _, ctx := commonTestContext(t)
	tn := tenant.MustFromContext(ctx)
	nodes, maxLevel, failures := synthesizeCommonNodes(l, tn, 100, 1001003, &common)
	if maxLevel != 3 {
		t.Fatalf("maxLevel = %d, want 3", maxLevel)
	}
	if failures != 0 {
		t.Fatalf("failures = %d, want 0", failures)
	}
	if len(nodes) != 3 {
		t.Fatalf("len(nodes) = %d, want 3", len(nodes))
	}
	for _, n := range nodes {
		if _, err := strconv.Atoi(n.Name); err != nil {
			t.Fatalf("node name %q is not a level number", n.Name)
		}
		for _, in := range n.IntegerNodes {
			if _, err := strconv.ParseInt(in.Value, 10, 64); err != nil {
				t.Fatalf("synthesized %s = %q does not round-trip through ParseInt", in.Name, in.Value)
			}
		}
		if len(n.StringNodes) != 0 {
			t.Fatalf("synthesized node carries %d string nodes, want 0", len(n.StringNodes))
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./skill/ -run 'TestCommon|TestSynthesized'
```

Expected: FAIL to build — `undefined: synthesizeCommonNodes`.

- [ ] **Step 3: Write `skill/common.go`**

```go
package skill

import (
	"fmt"
	"math"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"atlas-data/skill/effect"
	"atlas-data/skill/formula"
	"atlas-data/xml"
)

// commonKind classifies a `common` child leaf.
type commonKind int

const (
	// commonExpr is evaluated once per level with formula.Evaluate.
	commonExpr commonKind = iota
	// commonVector passes through unevaluated (FR-2.4).
	commonVector
	// commonOpaque is a client animation name, never evaluated (FR-2.5).
	commonOpaque
	// commonMeta is skill-level, not per-effect (maxLevel).
	commonMeta
)

// commonKeyDef declares how one `common` child key is handled and, for
// expressions, the inclusive range its target effect field admits.
//
// The range check exists because every key lands in a narrow Go type: an
// unchecked narrowing turns an evaluated -2 into uint16(65534). A violation is
// a loud FR-7.1 failure, never a silent wrap. It applies ONLY to the `common`
// path — the `level` path's conversions are untouched (FR-6.5).
type commonKeyDef struct {
	name string
	kind commonKind
	min  int64
	max  int64
}

const (
	minInt16  = math.MinInt16
	maxInt16  = math.MaxInt16
	maxUint16 = math.MaxUint16
	minInt32  = math.MinInt32
	maxInt32  = math.MaxInt32
	maxUint32 = math.MaxUint32
	// getEffect multiplies a non-negative `time` by 1000 to convert wz
	// seconds to milliseconds; bound it so that product still fits int32.
	maxTimeSeconds = math.MaxInt32 / 1000
)

// commonKeys is the declared table: the single source of truth for `common`
// handling, and the iteration order for synthesis (NFR-4 determinism — never
// range a Go map into serialized output). It covers all 65 distinct keys the
// GMS v95.1 archive uses.
var commonKeys = []commonKeyDef{
	// Skill-level metadata and pass-throughs.
	{name: "maxLevel", kind: commonMeta},
	{name: "action", kind: commonOpaque},
	{name: "lt", kind: commonVector},
	{name: "rb", kind: commonVector},

	// Keys the effect model already carried before task-192.
	{name: "time", kind: commonExpr, min: -1, max: maxTimeSeconds},
	{name: "hp", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mp", kind: commonExpr, min: 0, max: maxUint16},
	{name: "hpCon", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mpCon", kind: commonExpr, min: 0, max: maxUint16},
	{name: "prop", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mobCount", kind: commonExpr, min: 0, max: maxUint32},
	{name: "cooltime", kind: commonExpr, min: 0, max: maxUint32},
	{name: "morph", kind: commonExpr, min: 0, max: maxUint32},
	{name: "pad", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "pdd", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "mad", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "mdd", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "acc", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "eva", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "speed", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "jump", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "x", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "y", kind: commonExpr, min: minInt16, max: maxInt16},
	{name: "damage", kind: commonExpr, min: 0, max: maxUint32},
	{name: "attackCount", kind: commonExpr, min: 0, max: maxUint32},
	{name: "bulletCount", kind: commonExpr, min: 0, max: maxUint16},
	{name: "bulletConsume", kind: commonExpr, min: 0, max: maxUint16},
	{name: "moneyCon", kind: commonExpr, min: 0, max: maxUint32},
	{name: "itemCon", kind: commonExpr, min: 0, max: maxUint32},
	{name: "itemConNo", kind: commonExpr, min: 0, max: maxUint32},

	// Keys task-192 added to the effect model.
	{name: "mhpR", kind: commonExpr, min: 0, max: maxUint16},
	{name: "mmpR", kind: commonExpr, min: 0, max: maxUint16},
	{name: "range", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mastery", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "z", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dot", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "cr", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dotInterval", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "dotTime", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "damR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "criticaldamageMin", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "v", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "ignoreMobpdpR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "epad", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "w", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "u", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "epdd", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emdd", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "selfDestruction", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "asrR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "t", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "er", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "pddR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "terR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "madX", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "subProp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emhp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "criticaldamageMax", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "expR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "emmp", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "itemConsume", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mddR", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "subTime", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "padX", kind: commonExpr, min: minInt32, max: maxInt32},
	{name: "mesoR", kind: commonExpr, min: minInt32, max: maxInt32},
}

var commonKeyIndex = func() map[string]commonKeyDef {
	m := make(map[string]commonKeyDef, len(commonKeys))
	for _, k := range commonKeys {
		m[k.name] = k
	}
	return m
}()

// unknownCommonKey is the fallback for a key the table does not declare. It is
// carried (it costs nothing) but logged at WARN so a future archive that
// introduces a key is noticed rather than silently dropped.
func unknownCommonKey(name string) commonKeyDef {
	return commonKeyDef{name: name, kind: commonExpr, min: minInt32, max: maxInt32}
}

// commonLeaf is one classified, already-parsed `common` child.
type commonLeaf struct {
	def  commonKeyDef
	expr formula.Expr
	src  string
}

// lookupCommonLeaf returns the raw text of a `common` child leaf. `common` is
// exactly one level deep archive-wide (FR-2.1), and its leaves are string, int
// or vector only (FR-2.2) — an int-typed leaf's decimal text parses as a
// constant expression (FR-2.6).
func lookupCommonLeaf(node *xml.Node, name string) (string, bool) {
	for _, c := range node.IntegerNodes {
		if c.Name == name {
			return c.Value, true
		}
	}
	for _, c := range node.StringNodes {
		if c.Name == name {
			return c.Value, true
		}
	}
	return "", false
}

// commonMaxLevel reads the declared maxLevel (FR-2.3, FR-5.2). A missing or
// non-positive-integer value is an FR-7.4 failure for the whole skill: with no
// level count there is nothing to expand.
func commonMaxLevel(node *xml.Node) (int, error) {
	raw, ok := lookupCommonLeaf(node, "maxLevel")
	if !ok {
		return 0, fmt.Errorf("common node has no maxLevel")
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("maxLevel %q is not an integer: %w", raw, err)
	}
	if v <= 0 || v > 255 {
		return 0, fmt.Errorf("maxLevel %d is out of the 1..255 range", v)
	}
	return v, nil
}

// logCommonFailure emits the FR-7.1 ERROR. Fields are structured, not a
// formatted sentence, so a run is greppable by skill id (NFR-3).
func logCommonFailure(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, key string, expression string, level int, err error) {
	l.WithFields(logrus.Fields{
		"tenant":     t.Id().String(),
		"jobId":      jobId,
		"skillId":    uint32(skillId),
		"key":        key,
		"expression": expression,
		"level":      level,
	}).WithError(err).Errorf("Unable to derive skill effect from Skill.wz common node.")
}

// synthesizeCommonNodes expands a `common` subtree into one xml.Node per level
// (design §2.1 option C). The synthetic node's Name is the level number so
// levelFromNode keeps working, and it carries no child nodes — matching a
// `level` node that has no mob/0 subtree, which is what the archive's
// one-level-deep `common` means (FR-2.1).
//
// Only canonical strconv.FormatInt output is written, so no expression text
// can reach xml.GetIntegerWithDefault's silent default (FR-7.5).
//
// It returns the synthesized nodes, the DECLARED maxLevel (FR-5.2 — the
// authoritative value, not a derived count), and the failure count.
func synthesizeCommonNodes(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, common *xml.Node) ([]xml.Node, int, int) {
	failures := 0

	maxLevel, err := commonMaxLevel(common)
	if err != nil {
		raw, _ := lookupCommonLeaf(common, "maxLevel")
		logCommonFailure(l, t, jobId, skillId, "maxLevel", raw, 0, err)
		return nil, 0, failures + 1
	}

	// Classify and parse ONCE per key (FR-8.3): maxLevel is at most 255, so
	// this replaces up to 255 parses per key with one.
	leaves := make([]commonLeaf, 0, len(common.IntegerNodes)+len(common.StringNodes))
	seen := make(map[string]bool, len(common.IntegerNodes)+len(common.StringNodes))
	appendLeaf := func(name string, value string) {
		if seen[name] {
			return
		}
		seen[name] = true
		def, known := commonKeyIndex[name]
		if !known {
			def = unknownCommonKey(name)
			l.WithFields(logrus.Fields{
				"tenant":  t.Id().String(),
				"jobId":   jobId,
				"skillId": uint32(skillId),
				"key":     name,
			}).Warnf("Skill.wz common node carries an undeclared key; carrying it as a plain expression.")
		}
		if def.kind != commonExpr {
			return
		}
		e, err := formula.Parse(value)
		if err != nil {
			logCommonFailure(l, t, jobId, skillId, name, value, 0, err)
			failures++
			return
		}
		leaves = append(leaves, commonLeaf{def: def, expr: e, src: value})
	}

	// Declared-table order first, then any undeclared keys in archive order.
	for _, def := range commonKeys {
		if v, ok := lookupCommonLeaf(common, def.name); ok {
			appendLeaf(def.name, v)
		}
	}
	for _, c := range common.IntegerNodes {
		appendLeaf(c.Name, c.Value)
	}
	for _, c := range common.StringNodes {
		appendLeaf(c.Name, c.Value)
	}

	nodes := make([]xml.Node, 0, maxLevel)
	for level := 1; level <= maxLevel; level++ {
		synth := xml.Node{Name: strconv.Itoa(level)}
		for _, leaf := range leaves {
			v, err := leaf.expr.Evaluate(level)
			if err != nil {
				logCommonFailure(l, t, jobId, skillId, leaf.def.name, leaf.src, level, err)
				failures++
				continue
			}
			if v < leaf.def.min || v > leaf.def.max {
				logCommonFailure(l, t, jobId, skillId, leaf.def.name, leaf.src, level,
					fmt.Errorf("value %d is outside the target field range [%d,%d]", v, leaf.def.min, leaf.def.max))
				failures++
				continue
			}
			synth.IntegerNodes = append(synth.IntegerNodes, xml.IntegerNode{
				Name:  leaf.def.name,
				Value: strconv.FormatInt(v, 10),
			})
		}
		// lt/rb vectors pass through verbatim (FR-2.4).
		synth.PointNodes = append(synth.PointNodes, common.PointNodes...)
		nodes = append(nodes, synth)
	}
	return nodes, maxLevel, failures
}

// expandCommon derives a skill's effects from its `common` subtree. Every
// effect goes through the SAME getEffect the `level` path uses, so the two
// paths share one post-processing implementation (FR-5.3) and one set of
// defaults for absent keys (FR-5.4).
func expandCommon(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, buff bool, common *xml.Node) ([]effect.RestModel, uint8, Stats) {
	stats := Stats{}
	nodes, maxLevel, failures := synthesizeCommonNodes(l, t, jobId, skillId, common)
	stats.Failures = failures
	if failures > 0 {
		stats.SkillsWithFailures = 1
	}

	es := make([]effect.RestModel, 0, len(nodes))
	for _, n := range nodes {
		es = append(es, getEffect(t, skillId, buff, n))
	}
	// FR-5.2: the declared maxLevel is authoritative. commonMaxLevel already
	// bounded it to 1..255, and a failed parse yields 0 with zero effects.
	return es, uint8(maxLevel), stats
}
```

Add `Stats` at the bottom of `common.go` for now (Task 5 moves it to `stats.go` and extends the plumbing):

```go
// Stats counts one ingest pass's skill derivation outcomes. Task-192 FR-7.3
// requires a run summary; the counters are an explicit return value rather
// than a package-level registry so they stay per-run and order-independent.
type Stats struct {
	Processed          int
	FromCommon         int
	FromLevel          int
	Neither            int
	SkillsWithFailures int
	Failures           int
}

func (s *Stats) Add(o Stats) {
	s.Processed += o.Processed
	s.FromCommon += o.FromCommon
	s.FromLevel += o.FromLevel
	s.Neither += o.Neither
	s.SkillsWithFailures += o.SkillsWithFailures
	s.Failures += o.Failures
}
```

- [ ] **Step 4: Wire `produceSkill` to prefer `common`**

In `skill/reader.go`, replace the `es := make(...)` / `level, err := xml.ChildByName("level")` block in `produceSkill` (lines ~126-130) and the `maxLevel` block (lines ~139-144). New `produceSkill` signature and body fragment:

```go
func produceSkill(l logrus.FieldLogger, t tenant.Model, jobId uint32, skillId skill.Id, xml xml.Node) (RestModel, Stats, error) {
	// ... element/action/buff derivation above is unchanged ...

	stats := Stats{Processed: 1}
	es := make([]effect.RestModel, 0)
	maxLevel := uint8(0)

	// FR-1.2: COMMON WINS UNCONDITIONALLY. When `common` is present the
	// `level` subtree is not read at all. Detection is structural — never
	// gated on region/major/minor (FR-1.4).
	if common, err := xml.ChildByName("common"); err == nil {
		var commonStats Stats
		es, maxLevel, commonStats = expandCommon(l, t, jobId, skillId, buff, common)
		stats.Add(commonStats)
		stats.FromCommon = 1
	} else if level, err := xml.ChildByName("level"); err == nil {
		es = getEffects(t, skillId, buff, level.ChildNodes)
		if n := len(es); n > 255 {
			maxLevel = 255
		} else {
			maxLevel = uint8(n)
		}
		stats.FromLevel = 1
	} else {
		// FR-1.3: neither subtree. Zero effects, maxLevel 0, no panic.
		stats.Neither = 1
	}

	// ... name/desc lookup unchanged ...

	m := RestModel{
		Id:            uint32(skillId),
		Name:          name,
		Description:   desc,
		Action:        action,
		Element:       element,
		AnimationTime: 0,
		MaxLevel:      maxLevel,
		Effects:       es,
	}
	return m, stats, nil
}
```

Note the shadowed parameter name `xml` (the existing code names the node parameter `xml`, shadowing the package). Keep it — renaming it is out of scope — and reference `xml.ChildByName` on the parameter as the existing code already does.

In `Read`, adapt the caller and accumulate:

```go
			ms := make([]RestModel, 0)
			stats := Stats{}
			for _, sxml := range ssxml.ChildNodes {
				skillId, err := strconv.Atoi(sxml.Name)
				if err != nil {
					return model.ErrorProvider[[]RestModel](err)
				}
				l.Debugf("Processing skill [%d] for job [%d].", skillId, jobId)

				m, s, err := produceSkill(l, t, jobId, skill.Id(skillId), sxml)
				if err != nil {
					return model.ErrorProvider[[]RestModel](err)
				}
				stats.Add(s)
				ms = append(ms, m)
			}
			return model.FixedProvider[[]RestModel](ms)
```

`stats` is accumulated but not yet returned — Task 5 changes `Read`'s return type to carry it. Leave it assigned to `_ = stats` only if the compiler complains; prefer to land Task 5 immediately after so the value is used. (If `go vet` flags the unused write in this intermediate state, add `l.Debugf("Derived %d skills for job [%d].", stats.Processed, jobId)` — a line Task 5 keeps.)

Add `"github.com/sirupsen/logrus"` to `reader.go`'s imports if not already present (it is — `Read` takes a `logrus.FieldLogger`).

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test ./skill/...
```

Expected: PASS, including all pre-existing reader tests (they exercise `level`-only fixtures, which must be untouched).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/skill/common.go services/atlas-data/atlas.com/data/skill/common_test.go services/atlas-data/atlas.com/data/skill/reader.go
git commit -m "feat(atlas-data): derive skill effects from Skill.wz common nodes"
```

---

### Task 5: Failure stats plumbing and the run summary

**Files:**
- Create: `services/atlas-data/atlas.com/data/skill/stats.go` (move `Stats` here from `common.go`, add `Derivation` and `StatsAccumulator`)
- Modify: `services/atlas-data/atlas.com/data/skill/reader.go` (`Read` returns `Derivation`)
- Modify: `services/atlas-data/atlas.com/data/skill/processor.go`
- Modify: `services/atlas-data/atlas.com/data/skill/mock/processor.go`
- Modify: `services/atlas-data/atlas.com/data/data/workers/skill.go`
- Modify: `services/atlas-data/atlas.com/data/data/processor.go` (the `WorkerSkill` branch, ~line 153)
- Modify: `services/atlas-data/atlas.com/data/skill/reader_test.go`, `skill/rest_test.go`, `skill/common_test.go` (call-site shape)

**Interfaces:**
- Consumes: `Stats` (Task 4).
- Produces:
  - `type Derivation struct { Models []RestModel; Stats Stats }`
  - `func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[Derivation]`
  - `Processor.Register(s *document.Storage[string, RestModel], r model.Provider[Derivation]) (Stats, error)`
  - `Processor.RegisterSkill(path string) (Stats, error)`
  - `type StatsAccumulator struct{ … }` with `func (a *StatsAccumulator) Wrap(rf func(path string) (Stats, error)) func(path string) error` and `func (a *StatsAccumulator) Log(l logrus.FieldLogger)`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/skill/common_test.go`:

```go
// TestCommonFailureIsScopedAndCounted covers FR-7.1/7.2/7.3: a malformed
// expression logs exactly one ERROR with the required fields, does not abort
// the job image, leaves the key at getEffect's default, and is counted.
func TestCommonFailureIsScopedAndCounted(t *testing.T) {
	const xmlData = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="100.img">
  <imgdir name="skill">
    <imgdir name="1001003">
      <imgdir name="common">
        <int name="maxLevel" value="2"/>
        <string name="mpCon" value="6+2*u(x/5)"/>
        <string name="damage" value="100+bogus(x)"/>
      </imgdir>
    </imgdir>
    <imgdir name="1001004">
      <imgdir name="common">
        <string name="mpCon" value="x"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

	l, hook, ctx := commonTestContext(t)
	d, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))()
	if err != nil {
		t.Fatalf("Read error = %v, want nil (a formula failure must not abort the job image)", err)
	}
	if len(d.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(d.Models))
	}

	byId := map[uint32]RestModel{}
	for _, m := range d.Models {
		byId[m.Id] = m
	}

	// 1001003: the bad `damage` key is dropped; the good `mpCon` survives and
	// `damage` falls to getEffect's default of 100 (FR-5.4).
	good := byId[1001003]
	if len(good.Effects) != 2 {
		t.Fatalf("1001003 len(Effects) = %d, want 2", len(good.Effects))
	}
	if good.Effects[0].MPConsume != 8 {
		t.Fatalf("1001003 Effects[0].MPConsume = %d, want 8", good.Effects[0].MPConsume)
	}
	if good.Effects[0].Damage != 100 {
		t.Fatalf("1001003 Effects[0].Damage = %d, want the default 100", good.Effects[0].Damage)
	}

	// 1001004: no maxLevel — FR-7.4 scopes the failure to the whole skill.
	bad := byId[1001004]
	if bad.MaxLevel != 0 || len(bad.Effects) != 0 {
		t.Fatalf("1001004 (MaxLevel,len(Effects)) = (%d,%d), want (0,0)", bad.MaxLevel, len(bad.Effects))
	}

	if d.Stats.Processed != 2 {
		t.Fatalf("Stats.Processed = %d, want 2", d.Stats.Processed)
	}
	if d.Stats.FromCommon != 2 {
		t.Fatalf("Stats.FromCommon = %d, want 2", d.Stats.FromCommon)
	}
	if d.Stats.SkillsWithFailures != 2 {
		t.Fatalf("Stats.SkillsWithFailures = %d, want 2", d.Stats.SkillsWithFailures)
	}
	if d.Stats.Failures != 2 {
		t.Fatalf("Stats.Failures = %d, want 2 (one bad key, one missing maxLevel)", d.Stats.Failures)
	}

	errs := 0
	for _, entry := range hook.AllEntries() {
		if entry.Level != logrus.ErrorLevel {
			continue
		}
		errs++
		for _, field := range []string{"tenant", "jobId", "skillId", "key", "expression"} {
			if _, ok := entry.Data[field]; !ok {
				t.Fatalf("ERROR entry is missing the %q field: %+v", field, entry.Data)
			}
		}
	}
	if errs != 2 {
		t.Fatalf("ERROR log count = %d, want 2", errs)
	}
}

// TestStatsAccumulatorSummary covers FR-7.3: the run summary escalates to
// ERROR when any skill had a derivation failure.
func TestStatsAccumulatorSummary(t *testing.T) {
	l, hook := test.NewNullLogger()
	var acc StatsAccumulator

	clean := acc.Wrap(func(string) (Stats, error) {
		return Stats{Processed: 3, FromLevel: 3}, nil
	})
	if err := clean("a.img.xml"); err != nil {
		t.Fatalf("wrapped register error = %v", err)
	}
	dirty := acc.Wrap(func(string) (Stats, error) {
		return Stats{Processed: 2, FromCommon: 2, SkillsWithFailures: 1, Failures: 4}, nil
	})
	if err := dirty("b.img.xml"); err != nil {
		t.Fatalf("wrapped register error = %v", err)
	}
	acc.Log(l)

	var summary, escalation bool
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.InfoLevel && strings.Contains(entry.Message, "processed=5") {
			summary = true
		}
		if entry.Level == logrus.ErrorLevel {
			escalation = true
		}
	}
	if !summary {
		t.Fatalf("no INFO run summary with processed=5: %+v", hook.AllEntries())
	}
	if !escalation {
		t.Fatal("failures > 0 did not escalate the summary to ERROR")
	}
}
```

Add `"strings"` to `common_test.go`'s imports.

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./skill/ -run 'TestCommonFailure|TestStatsAccumulator'
```

Expected: FAIL to build — `d.Models undefined`, `undefined: StatsAccumulator`.

- [ ] **Step 3: Create `skill/stats.go`**

Move the `Stats` type out of `common.go` (delete it there) into a new `services/atlas-data/atlas.com/data/skill/stats.go`:

```go
package skill

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Stats counts one ingest pass's skill derivation outcomes (FR-7.3). The
// counters are an explicit return value rather than a package-level registry:
// they are per-run, not per-tenant-lifetime, and hidden global mutable state
// makes unit tests order-dependent.
type Stats struct {
	Processed          int
	FromCommon         int
	FromLevel          int
	Neither            int
	SkillsWithFailures int
	Failures           int
}

func (s *Stats) Add(o Stats) {
	s.Processed += o.Processed
	s.FromCommon += o.FromCommon
	s.FromLevel += o.FromLevel
	s.Neither += o.Neither
	s.SkillsWithFailures += o.SkillsWithFailures
	s.Failures += o.Failures
}

// Derivation is what reading one Skill.wz job image produces: the skill
// documents and the counters describing how they were derived.
type Derivation struct {
	Models []RestModel
	Stats  Stats
}

// StatsAccumulator sums Stats across the job images of one ingest run and
// emits the run summary. Both ingest entry points (the SKILL worker and the
// legacy data processor) use it, so neither can silently drop the summary.
type StatsAccumulator struct {
	mu    sync.Mutex
	total Stats
}

// Wrap adapts a stats-returning register function to the plain
// func(path) error shape the directory walkers expect, accumulating as it
// goes. A failing register contributes nothing to the totals.
func (a *StatsAccumulator) Wrap(rf func(path string) (Stats, error)) func(path string) error {
	return func(path string) error {
		s, err := rf(path)
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.total.Add(s)
		a.mu.Unlock()
		return nil
	}
}

// Total returns a copy of the accumulated counters.
func (a *StatsAccumulator) Total() Stats {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.total
}

// Log emits the FR-7.3 run summary. The separate ERROR line is what makes the
// per-failure permissiveness safe: one ERROR can scroll past in a 954-skill
// run, an aggregate at the end cannot.
func (a *StatsAccumulator) Log(l logrus.FieldLogger) {
	s := a.Total()
	l.Infof("skills: processed=%d fromCommon=%d fromLevel=%d neither=%d failures=%d",
		s.Processed, s.FromCommon, s.FromLevel, s.Neither, s.Failures)
	if s.SkillsWithFailures > 0 {
		l.WithFields(logrus.Fields{
			"skillsWithFailures": s.SkillsWithFailures,
			"failures":           s.Failures,
		}).Errorf("Skill.wz ingest had skills with common-node derivation failures.")
	}
}
```

- [ ] **Step 4: Change `Read` to return `Derivation`**

In `skill/reader.go`, change the signature and the two return sites:

```go
func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[Derivation] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[Derivation] {
		t := tenant.MustFromContext(ctx)
		return func(np model.Provider[xml.Node]) model.Provider[Derivation] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[Derivation](err)
			}
			// ... jobId + ssxml lookups unchanged, but every
			// model.ErrorProvider[[]RestModel] becomes
			// model.ErrorProvider[Derivation] ...

			return model.FixedProvider(Derivation{Models: ms, Stats: stats})
		}
	}
}
```

- [ ] **Step 5: Update the processor, mock and both ingest entry points**

`skill/processor.go`:

```go
type Processor interface {
	Register(s *document.Storage[string, RestModel], r model.Provider[Derivation]) (Stats, error)
	RegisterSkill(path string) (Stats, error)
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[Derivation]) (Stats, error) {
	d, err := r()
	if err != nil {
		return Stats{}, err
	}
	for _, m := range d.Models {
		_, err = s.Add(p.ctx)(m)()
		if err != nil {
			return Stats{}, err
		}
	}
	return d.Stats, nil
}

func (p *ProcessorImpl) RegisterSkill(path string) (Stats, error) {
	var stats Stats
	err := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		s, err := p.Register(NewStorage(p.l, tx), Read(p.l)(p.ctx)(xml.FromPathProvider(path)))
		stats = s
		return err
	})
	return stats, err
}
```

`skill/mock/processor.go`:

```go
type ProcessorMock struct {
	RegisterFunc      func(s *document.Storage[string, skill.RestModel], r model.Provider[skill.Derivation]) (skill.Stats, error)
	RegisterSkillFunc func(path string) (skill.Stats, error)
}

func (m *ProcessorMock) Register(s *document.Storage[string, skill.RestModel], r model.Provider[skill.Derivation]) (skill.Stats, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return skill.Stats{}, nil
}

func (m *ProcessorMock) RegisterSkill(path string) (skill.Stats, error) {
	if m.RegisterSkillFunc != nil {
		return m.RegisterSkillFunc(path)
	}
	return skill.Stats{}, nil
}
```

`data/workers/skill.go` — replace the `registerAllInDirectory(... .RegisterSkill)` call at line 50:

```go
	// Accumulate the FR-7.3 run summary across every per-job image.
	var skillStats skill.StatsAccumulator
	if err := registerAllInDirectory(l, ctx, filepath.Join(root, "Skill.wz"), skillStats.Wrap(skill.NewProcessor(l, ctx, db).RegisterSkill)); err != nil {
		return err
	}
	skillStats.Log(l)
```

`data/processor.go` — the `WorkerSkill` branch (~line 153):

```go
		var skillStats skill.StatsAccumulator
		err = p.RegisterAllData(path, "Skill.wz", skillStats.Wrap(skill.NewProcessor(p.l, p.ctx, p.db).RegisterSkill))()
		skillStats.Log(p.l)
		_ = skill.GetSkillStringRegistry().Clear(t)
```

- [ ] **Step 6: Update the test call sites**

`Read(...)` is called at 14 sites in `skill/reader_test.go` and 1 in `skill/rest_test.go`. At each, replace

```go
	rms := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))
```

with

```go
	d, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(xmlData)))()
	if err != nil {
		t.Fatal(err)
	}
	rms := model.FixedProvider(d.Models)
```

Everything downstream of `rms` in each test is unchanged. Where a test already declares `err` earlier in scope, use `=` instead of `:=` for the tuple as the compiler requires.

Two further conversions:

- `TestLevelPathPopulatesCommonKeys` (written in Task 3 Step 1 against the old shape) becomes the `d.Models` form given there.
- `common_test.go`'s `readSkills` helper is the single point of change for every Task 4 test — its body becomes:

```go
func readSkills(t *testing.T, l logrus.FieldLogger, ctx context.Context, data string) []RestModel {
	t.Helper()
	d, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(data)))()
	if err != nil {
		t.Fatal(err)
	}
	return d.Models
}
```

- [ ] **Step 7: Run the full module test suite**

```bash
go test -race ./...
```

Expected: PASS for every package in the `atlas-data` module.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data
git commit -m "feat(atlas-data): report skill common derivation failures and a run summary"
```

---

### Task 6: atlas-ui — surface the new attributes

**Files:**
- Modify: `services/atlas-ui/src/services/api/skills.service.ts` (the `SkillEffect` interface, lines ~14-48)
- Modify: `services/atlas-ui/src/lib/skills/level-table.ts` (`FIELD_LABELS`, lines ~19-53)

**Interfaces:**
- Consumes: the JSON attribute names pinned by Task 2's `TestCommonKeyJSONTags`.
- Produces: nothing consumed by later tasks.

Without this, all 35 attributes are ingested, persisted and served — and render nowhere, because the level table is an explicit whitelist. `buildLevelTable` already omits all-zero columns, so tenants without `common` nodes see no new columns.

For keys whose meaning is unverified (`z`, `u`, `v`, `w`, `t`), the label **is the key** — no invented gloss (design §5.1, OQ-4).

- [ ] **Step 1: Extend the `SkillEffect` interface**

In `skills.service.ts`, append inside `SkillEffect` before `statups?: SkillEffectStatup[];`:

```ts
  // Skill.wz `common` keys (task-192). Keys are the Go JSON tags verbatim.
  range?: number;
  mastery?: number;
  z?: number;
  dot?: number;
  cr?: number;
  dotInterval?: number;
  dotTime?: number;
  damR?: number;
  criticaldamageMin?: number;
  criticaldamageMax?: number;
  v?: number;
  ignoreMobpdpR?: number;
  epad?: number;
  w?: number;
  u?: number;
  epdd?: number;
  emdd?: number;
  selfDestruction?: number;
  asrR?: number;
  t?: number;
  er?: number;
  pddR?: number;
  terR?: number;
  mddR?: number;
  madX?: number;
  padX?: number;
  subProp?: number;
  subTime?: number;
  emhp?: number;
  emmp?: number;
  expR?: number;
  mesoR?: number;
  consumeItemId?: number;
```

(`MHPRRate` and `MMPRRate` are already declared — they are the `mhpR`/`mmpR` reuse.)

- [ ] **Step 2: Extend `FIELD_LABELS`**

In `level-table.ts`, append to the `FIELD_LABELS` array, after the existing `["itemConsumeAmount", "Item Qty"]` entry so current column order is unchanged:

```ts
  // Skill.wz `common` keys (task-192). Where the key's meaning is not
  // verified from the client or the archive, the label IS the key — an
  // invented gloss would become permanent and load-bearing.
  ["range", "Range"],
  ["mastery", "Mastery"],
  ["dot", "DoT Damage"],
  ["dotInterval", "DoT Interval"],
  ["dotTime", "DoT Time"],
  ["cr", "cr"],
  ["damR", "damR"],
  ["criticaldamageMin", "Crit Damage Min"],
  ["criticaldamageMax", "Crit Damage Max"],
  ["ignoreMobpdpR", "ignoreMobpdpR"],
  ["epad", "epad"],
  ["epdd", "epdd"],
  ["emdd", "emdd"],
  ["emhp", "emhp"],
  ["emmp", "emmp"],
  ["pddR", "pddR"],
  ["mddR", "mddR"],
  ["padX", "padX"],
  ["madX", "madX"],
  ["asrR", "asrR"],
  ["terR", "terR"],
  ["er", "er"],
  ["expR", "expR"],
  ["mesoR", "mesoR"],
  ["subProp", "subProp"],
  ["subTime", "subTime"],
  ["selfDestruction", "selfDestruction"],
  ["consumeItemId", "Consume Item ID"],
  ["t", "t"],
  ["u", "u"],
  ["v", "v"],
  ["w", "w"],
  ["z", "z"],
```

- [ ] **Step 3: Verify**

From `services/atlas-ui`, with Node 22 active (`nvm use 22`):

```bash
npm run build
```

Expected: build succeeds. `npm run build` type-checks the test files too, which `vitest` alone does not — a per-task atlas-ui verification that skips it is not a verification.

Then:

```bash
npx vitest run src/lib/skills
```

Expected: the existing level-table tests pass. `FIELD_LABELS` entries whose key is not in `SkillEffect` are a type error, so Step 1 and Step 2 must agree exactly.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/services/api/skills.service.ts services/atlas-ui/src/lib/skills/level-table.ts
git commit -m "feat(atlas-ui): surface the Skill.wz common effect attributes"
```

---

### Task 7: Archive verification — OQ-5 census and the differential corpus

**Files:**
- Create: `docs/tasks/task-192-v95-skill-common-formulas/archive-census.md`
- Create: `services/atlas-data/atlas.com/data/skill/formula/testdata/common_corpus.csv`
- Create: `services/atlas-data/atlas.com/data/skill/formula/corpus_test.go`

**Interfaces:**
- Consumes: `formula.Parse` / `Expr.Evaluate` (Task 1).
- Produces: a committed regression corpus; no new Go API.

This task is **not optional and not deferrable**. Design §8 puts OQ-5 in scope for the plan, and §10.3's differential check is what converts "the client's precedence and the textbook precedence happen to agree on this archive" from a premise into a checked fact.

The analysis program is a **scratch module outside the repo** (the precedent set by `wz-common-grammar.md`'s `wzscan/`), written under the session scratchpad with `replace` directives to the repo libs. Nothing outside the three files above is committed.

- [ ] **Step 1: Fetch the archives**

Port-forward MinIO and pull each region/version `Skill.wz` (credentials in Secret `atlas-minio-credentials`, namespace `atlas-main`):

```bash
kubectl -n minio port-forward svc/minio 19000:9000
# then, per archive, with the access/secret key from the secret:
curl --aws-sigv4 "aws:amz:us-east-1:s3" --user "$KEY:$SECRET" \
  -o Skill-<region>-<version>.wz \
  "http://127.0.0.1:19000/atlas-wz/shared/regions/<REGION>/versions/<VERSION>/Skill.wz"
```

Enumerate the available `regions/*/versions/*` prefixes rather than assuming the set. Record each file's size and md5.

- [ ] **Step 2: Census every archive**

In the scratch module, for each downloaded archive: `wz.Open`, assert `File.GameVersion()`, walk every root image, and count skill nodes under `/<numeric jobId>/skill/` with `common` only, `level` only, both, and neither.

Write `docs/tasks/task-192-v95-skill-common-formulas/archive-census.md` containing, per archive: region, version, `GameVersion()`, md5, and the four counts. For any archive **other than GMS 95.1** with a non-zero `common` count, additionally tokenize every one of its `common` string values and report whether the character classes and function names stay inside the FR-3 grammar. State plainly whether this task's fix applies to that tenant — do not assert it without the census row behind it.

- [ ] **Step 3: Extract and adjudicate the corpus**

From the GMS v95.1 archive (md5 `2d77583108eb928b65f2904196a894ef`), collect every distinct `(expression, maxLevel)` pair under every `common` node. For each pair and each level `1..maxLevel`, compute:

- **A** — this task's evaluator (`formula.Parse` + `Evaluate`, via the scratch module's `replace` to the repo copy);
- **B** — a naive reference: standard precedence (`*`/`/` tier above `+`/`-`, left-to-right), `math.Ceil` for `u`, `math.Floor` for `d`, and a single truncation at the end.

Any `(expression, level)` where A ≠ B is a **finding**, not a rounding detail: re-read `GetParsedCommonData`/`GetArithmeticData` in IDA session `79906a1e` and adjudicate it against the client before continuing. Record every disagreement (or the fact that there were none) in `archive-census.md`.

Write the surviving corpus to `services/atlas-data/atlas.com/data/skill/formula/testdata/common_corpus.csv` with a header and one row per `(expression, level)`:

```csv
expression,level,expected
6+2*u(x/5),1,8
6+2*u(x/5),20,14
```

Quote fields per RFC 4180 (`encoding/csv` handles the leading-space value ` 375+5*x` correctly when quoted).

- [ ] **Step 4: Write the corpus test**

Create `services/atlas-data/atlas.com/data/skill/formula/corpus_test.go`:

```go
package formula

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"
)

// TestArchiveCorpus replays every distinct (expression, level) pair found in
// the GMS v95.1 Skill.wz common nodes (md5 2d77583108eb928b65f2904196a894ef).
// The expected column was produced by this evaluator AND cross-checked against
// a standard-precedence reference implementation; every disagreement was
// adjudicated against the client (design §10.3, docs/tasks/task-192-.../
// archive-census.md). The 163 MB archive is not committed — this corpus is.
func TestArchiveCorpus(t *testing.T) {
	f, err := os.Open("testdata/common_corpus.csv")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	if len(records) < 2 {
		t.Fatal("corpus is empty")
	}

	cache := map[string]Expr{}
	for i, rec := range records[1:] {
		if len(rec) != 3 {
			t.Fatalf("row %d has %d fields, want 3", i+2, len(rec))
		}
		src, levelText, wantText := rec[0], rec[1], rec[2]
		level, err := strconv.Atoi(levelText)
		if err != nil {
			t.Fatalf("row %d: bad level %q", i+2, levelText)
		}
		want, err := strconv.ParseInt(wantText, 10, 64)
		if err != nil {
			t.Fatalf("row %d: bad expected %q", i+2, wantText)
		}
		e, ok := cache[src]
		if !ok {
			e, err = Parse(src)
			if err != nil {
				t.Fatalf("row %d: Parse(%q) error = %v", i+2, src, err)
			}
			cache[src] = e
		}
		got, err := e.Evaluate(level)
		if err != nil {
			t.Fatalf("row %d: Evaluate(%q, %d) error = %v", i+2, src, level, err)
		}
		if got != want {
			t.Fatalf("row %d: Parse(%q).Evaluate(%d) = %d, want %d", i+2, src, level, got, want)
		}
	}
}
```

- [ ] **Step 5: Run the corpus test**

```bash
go test ./skill/formula/...
```

Expected: PASS. A `Parse` error on any row means the FR-3 grammar is **not** complete for this archive — that is a real gap and must be fixed in Task 1, not worked around by deleting the row.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-192-v95-skill-common-formulas/archive-census.md \
        services/atlas-data/atlas.com/data/skill/formula/testdata/common_corpus.csv \
        services/atlas-data/atlas.com/data/skill/formula/corpus_test.go
git commit -m "test(atlas-data): pin the v95.1 common expression corpus and census the archives"
```

---

### Task 8: Gate sweep, end-to-end verification, and code review

**Files:**
- Modify: `docs/tasks/task-192-v95-skill-common-formulas/` (verification notes appended to `archive-census.md`)
- No source changes expected; any needed fix lands in the task it belongs to.

**Interfaces:**
- Consumes: everything above.
- Produces: the evidence the PR description cites.

- [ ] **Step 1: Run the Go gates**

From `services/atlas-data/atlas.com/data`:

```bash
go build ./...
go vet ./...
go test -race ./...
```

Expected: all three clean. From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: exit 0 each. `tools/lint.sh --check` needs Node 22 on PATH (`nvm use 22`) or it false-fails on the atlas-ui half. Fix mode is `tools/lint.sh` with no flags — run it before committing if the check reports formatting drift.

`go.mod` must be unchanged (the evaluator adds no dependency). Confirm with `git diff --stat -- services/atlas-data/atlas.com/data/go.mod`; only if it is non-empty is `docker buildx bake atlas-data` required from the worktree root.

- [ ] **Step 2: Merge main and re-verify**

```bash
git fetch origin main
git merge origin/main
```

Re-run Step 1 after the merge. Verifying against the branch tip alone is not verification — the merge result is what CI builds.

- [ ] **Step 3: Deploy and re-ingest (PRD §11 runbook)**

1. Deploy the atlas-data image built from this branch.
2. `POST /data/process` scoped to the v95.1 archive (`shared` scope requires the `X-Atlas-Operator: 1` header). This creates a Kubernetes Job.
3. Watch it with `GET /data/process` and read the pod logs for the FR-7.3 summary line. **A non-zero failure count blocks acceptance** — the grammar is claimed complete, so any failure is a real gap. Investigate before continuing.
4. Restart the atlas-data **serving** pods. The ingest Job pod is a separate process; its in-memory registry mirror is not the one serving traffic.

- [ ] **Step 4: End-to-end checks**

With the v95 tenant headers:

```bash
# Flagship: 20 levels, MPConsume 8 -> 14, duration 110000 -> 300000 ms.
GET /api/data/skills/1001003
```

Assert `maxLevel == 20`, `len(effects) == 20`, `effects[0].MPConsume == 8`, `effects[0].duration == 110000`, `effects[19].MPConsume == 14`, `effects[19].duration == 300000`.

Then census the served documents: page `GET /api/data/skills` for the v95 tenant and count documents with `maxLevel == 0` and an empty `effects` array. Expected: **0 of 954** (down from 635).

Finally, spot the non-v95 tenants: for at least one `level`-only tenant (v92), confirm no *existing* effect attribute changed name, type, or value. Per the design's amendment 1 to the PRD, byte-identity is **not** the criterion — every effect legitimately gains the new zero-valued attributes, and a `level` node that carries e.g. `mastery` now serves it.

Record the actual observed values (not "as expected") in `archive-census.md` under a "Verification" heading.

- [ ] **Step 5: Code review before the PR**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go changed) and `frontend-guidelines-reviewer` (atlas-ui TS changed); findings land in `docs/tasks/task-192-v95-skill-common-formulas/audit.md`. Pin the reviewer subagents to a standard model, not an expensive one. Ensure each subagent operates inside this worktree and leaves the tree clean.

Address the findings, then re-run Step 1.

- [ ] **Step 6: Commit the verification record**

```bash
git add docs/tasks/task-192-v95-skill-common-formulas
git commit -m "docs(task-192): record archive census and end-to-end verification"
```

The PR description must name, explicitly: the 2211002/2211006 accepted regression (FR-1.2), the OQ-5 census result, and the three PRD amendments from design §12.

---

## Requirement Coverage

| Requirement | Task |
|---|---|
| FR-1.1 / FR-1.2 / FR-1.3 / FR-1.4 | 4 (`produceSkill`), tests `TestCommonWinsOverLevel`, `TestCommonNeitherSubtree` |
| FR-2.1 / FR-2.2 | 4 (no recursion into `common`; leaf lookup over Integer/String/Point nodes) |
| FR-2.3 | 4 (`commonMaxLevel`), test `TestCommonMaxLevelNodeTypes` |
| FR-2.4 / FR-2.5 / FR-2.6 | 4, test `TestCommonIntLeavesAndPassThrough` |
| FR-3.1 – FR-3.10 | 1, tests `TestEvaluate` (each case names its FR) |
| FR-4.1 | 4, test `TestCommonXKeyIsNotTheVariable` |
| FR-5.1 / FR-5.2 | 4, test `TestCommonExpansion` |
| FR-5.3 / FR-5.4 | 4 (synthesis feeds the unmodified `getEffect`), tests `TestCommonExpansion`, `TestCommonFailureIsScopedAndCounted` (the `damage` default of 100) |
| FR-6.1 – FR-6.5 | 2 + 3, tests `TestCommonKeyJSONTags`, `TestExistingItemConsumeUnchanged`, `TestLevelPathPopulatesCommonKeys` |
| FR-7.1 / FR-7.2 / FR-7.4 | 4 + 5, test `TestCommonFailureIsScopedAndCounted` |
| FR-7.3 | 5, test `TestStatsAccumulatorSummary` |
| FR-7.5 | 4, test `TestSynthesizedNodesAreCanonicalIntegers` |
| FR-8.1 / FR-8.2 / FR-8.3 | 1 (pure package, `Parse` separate from `Evaluate`), test `TestParseOnceEvaluateMany` |
| NFR-1 | 1 + 4 (parse once, evaluate per level) |
| NFR-2 | unchanged; evaluator is tenant-agnostic (Task 1) |
| NFR-3 | 5 (structured log fields), test asserts the field set |
| NFR-4 | 4 (declared-table iteration order) |
| NFR-5 | 1 (length/token/depth bounds, no symbol table), test `TestParseRejects` |
| NFR-6 | 1 + 7 |
| OQ-1 (rounding for `t`) | resolved in design §1.5 — `t` is a truncated `int32` (Task 2) |
| OQ-2 (`itemConsume` vs `itemCon`) | resolved in design §5.4 — separate `consumeItemId` (Task 2) |
| OQ-3 (`MHPRRate`/`MMPRRate` reuse) | resolved in design §5.3 — reused (Tasks 2, 3) |
| OQ-4 (single-letter key semantics) | carried opaquely; labels are the keys (Tasks 2, 6) |
| OQ-5 (other regions' archives) | 7 |
| design §7 (atlas-ui surfacing) | 6 |
| PRD §10 build & verification gates | 8 |
| PRD §11 rollout runbook | 8 |
