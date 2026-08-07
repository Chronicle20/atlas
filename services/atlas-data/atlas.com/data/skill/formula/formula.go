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
