package atlaspacket

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// GuardContext holds the version/region parameters used to evaluate a guard expression.
type GuardContext struct {
	Region       string
	MajorVersion uint16
	MinorVersion uint16
}

// GuardExpr is a compiled version-conditional guard expression.
type GuardExpr struct {
	eval func(GuardContext) bool
	text string
}

// Eval evaluates the guard against a GuardContext.
func (g *GuardExpr) Eval(c GuardContext) bool { return g.eval(c) }

// String returns the original source text of the guard.
func (g *GuardExpr) String() string { return g.text }

// Text returns the canonical string form of this guard expression.
func (g *GuardExpr) Text() string {
	if g == nil {
		return ""
	}
	return g.text
}

// ParseGuard parses a Go expression string into a GuardExpr that can be evaluated
// against a GuardContext. Supports:
//   - t.Region() == "X" / != "X"
//   - t.MajorVersion() >/>=/</<=/==/!= N
//   - t.MinorVersion() >/>=/</<=/==/!= N
//   - && / || combinations
//   - !( ... ) negation
func ParseGuard(text string) (*GuardExpr, error) {
	e, err := parser.ParseExpr(text)
	if err != nil {
		return nil, err
	}
	fn, err := compileExpr(e)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", text, err)
	}
	return &GuardExpr{eval: fn, text: text}, nil
}

func compileExpr(e ast.Expr) (func(GuardContext) bool, error) {
	switch v := e.(type) {
	case *ast.ParenExpr:
		return compileExpr(v.X)
	case *ast.BinaryExpr:
		return compileBinary(v)
	case *ast.UnaryExpr:
		if v.Op == token.NOT {
			inner, err := compileExpr(v.X)
			if err != nil {
				return nil, err
			}
			return func(c GuardContext) bool { return !inner(c) }, nil
		}
	}
	return nil, fmt.Errorf("unsupported expression %T", e)
}

func compileBinary(b *ast.BinaryExpr) (func(GuardContext) bool, error) {
	switch b.Op {
	case token.LAND:
		l, err := compileExpr(b.X)
		if err != nil {
			return nil, err
		}
		r, err := compileExpr(b.Y)
		if err != nil {
			return nil, err
		}
		return func(c GuardContext) bool { return l(c) && r(c) }, nil
	case token.LOR:
		l, err := compileExpr(b.X)
		if err != nil {
			return nil, err
		}
		r, err := compileExpr(b.Y)
		if err != nil {
			return nil, err
		}
		return func(c GuardContext) bool { return l(c) || r(c) }, nil
	}
	lhs, err := callKey(b.X)
	if err != nil {
		return nil, err
	}
	switch lhs {
	case "Region":
		s, err := stringLit(b.Y)
		if err != nil {
			return nil, err
		}
		switch b.Op {
		case token.EQL:
			return func(c GuardContext) bool { return c.Region == s }, nil
		case token.NEQ:
			return func(c GuardContext) bool { return c.Region != s }, nil
		}
	case "MajorVersion":
		n, err := intLit(b.Y)
		if err != nil {
			return nil, err
		}
		return cmpUint(b.Op, n, func(c GuardContext) uint16 { return c.MajorVersion })
	case "MinorVersion":
		n, err := intLit(b.Y)
		if err != nil {
			return nil, err
		}
		return cmpUint(b.Op, n, func(c GuardContext) uint16 { return c.MinorVersion })
	}
	return nil, fmt.Errorf("unsupported binary lhs %q", lhs)
}

func cmpUint(op token.Token, rhs uint16, lhs func(GuardContext) uint16) (func(GuardContext) bool, error) {
	switch op {
	case token.GTR:
		return func(c GuardContext) bool { return lhs(c) > rhs }, nil
	case token.GEQ:
		return func(c GuardContext) bool { return lhs(c) >= rhs }, nil
	case token.LSS:
		return func(c GuardContext) bool { return lhs(c) < rhs }, nil
	case token.LEQ:
		return func(c GuardContext) bool { return lhs(c) <= rhs }, nil
	case token.EQL:
		return func(c GuardContext) bool { return lhs(c) == rhs }, nil
	case token.NEQ:
		return func(c GuardContext) bool { return lhs(c) != rhs }, nil
	}
	return nil, fmt.Errorf("unsupported numeric op %v", op)
}

func callKey(e ast.Expr) (string, error) {
	ce, ok := e.(*ast.CallExpr)
	if !ok {
		return "", fmt.Errorf("expected call, got %T", e)
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", fmt.Errorf("expected selector call, got %T", ce.Fun)
	}
	name := sel.Sel.Name
	if _, ok := sel.X.(*ast.Ident); ok {
		return name, nil
	}
	return "", fmt.Errorf("unsupported lhs receiver %T", sel.X)
}

func stringLit(e ast.Expr) (string, error) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", fmt.Errorf("expected string literal, got %T", e)
	}
	return strings.Trim(bl.Value, `"`), nil
}

func intLit(e ast.Expr) (uint16, error) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return 0, fmt.Errorf("expected int literal, got %T", e)
	}
	n, err := strconv.ParseUint(bl.Value, 10, 16)
	return uint16(n), err
}
