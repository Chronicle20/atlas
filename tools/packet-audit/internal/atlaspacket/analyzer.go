package atlaspacket

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

type Primitive int

const (
	Encode1   Primitive = iota // byte / bool
	Encode2                    // int16
	Encode4                    // int32
	Encode8                    // int64
	EncodeStr                  // ascii string
	EncodeBuf                  // raw bytes
)

func (p Primitive) String() string {
	return [...]string{"byte", "int16", "int32", "int64", "string", "bytes"}[p]
}

// Kind classifies what a Call represents.
type Kind int

const (
	KindWrite   Kind = iota // a primitive write/read call (Op is valid)
	KindRecurse             // a sub-struct .Encode/.Decode call (RecurseType is valid)
	KindRepeat              // a for/range loop over a slice (Body is valid)
)

// Call represents a single writer/reader primitive call found inside an Encode/Decode method,
// or a recurse/repeat marker for sub-struct and loop encoding.
type Call struct {
	Kind        Kind
	Op          Primitive  // valid for KindWrite
	RecurseType string     // valid for KindRecurse — Go receiver/field name (best-effort)
	Body        []Call     // valid for KindRepeat
	Line        int
	Guard       *GuardExpr // nil for unconditional; populated in Task 8
}

// AnalyzeFile parses a single .go (or .go.txt) file and extracts the ordered
// sequence of w.Write*/r.Read* calls inside the named method's outer return closure.
func AnalyzeFile(path, typeName, methodName string) ([]Call, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var body *ast.BlockStmt
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != methodName || fd.Recv == nil || len(fd.Recv.List) != 1 {
			continue
		}
		recvType := ""
		switch rt := fd.Recv.List[0].Type.(type) {
		case *ast.Ident:
			recvType = rt.Name
		case *ast.StarExpr:
			if id, ok := rt.X.(*ast.Ident); ok {
				recvType = id.Name
			}
		}
		if recvType != typeName {
			continue
		}
		body = fd.Body
		break
	}
	if body == nil {
		return nil, fmt.Errorf("method %s.%s not found in %s", typeName, methodName, path)
	}
	inner := findReturnClosure(body)
	if inner == nil {
		inner = body
	}
	return collectCalls(inner, fset), nil
}

// findReturnClosure finds the first return func literal's body in a block statement.
func findReturnClosure(b *ast.BlockStmt) *ast.BlockStmt {
	var out *ast.BlockStmt
	ast.Inspect(b, func(n ast.Node) bool {
		if out != nil {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		fl, ok := ret.Results[0].(*ast.FuncLit)
		if !ok {
			return true
		}
		out = fl.Body
		return false
	})
	return out
}

// collectCalls walks a block and collects all w.Write*/r.Read* primitive calls in order,
// tagging each with a Guard derived from enclosing if-statement conditions.
// It also emits KindRecurse markers for .Encode/.Decode sub-struct calls and
// KindRepeat markers for for/range loops.
func collectCalls(b *ast.BlockStmt, fset *token.FileSet) []Call {
	var out []Call
	var stack []*GuardExpr
	var walk func(node ast.Node)
	walk = func(node ast.Node) {
		switch n := node.(type) {
		case *ast.IfStmt:
			g := guardFromIf(n, fset)
			stack = append(stack, g)
			walk(n.Body)
			stack = stack[:len(stack)-1]
			if n.Else != nil {
				ng := negate(g)
				stack = append(stack, ng)
				walk(n.Else)
				stack = stack[:len(stack)-1]
			}
		case *ast.BlockStmt:
			for _, s := range n.List {
				walk(s)
			}
		case *ast.ExprStmt:
			walk(n.X)
		case *ast.RangeStmt:
			sub := collectGuardedSub(n.Body, fset)
			out = append(out, Call{
				Kind:  KindRepeat,
				Body:  sub,
				Line:  fset.Position(n.Pos()).Line,
				Guard: conjoin(stack),
			})
		case *ast.ForStmt:
			sub := collectGuardedSub(n.Body, fset)
			out = append(out, Call{
				Kind:  KindRepeat,
				Body:  sub,
				Line:  fset.Position(n.Pos()).Line,
				Guard: conjoin(stack),
			})
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				// Handle chained calls like m.sub.Encode(l, ctx)(opts):
				// the outer CallExpr has Fun = inner CallExpr; recurse into Fun.
				if inner, ok := n.Fun.(*ast.CallExpr); ok {
					walk(inner)
				}
				return
			}
			// Recurse marker: x.Encode(l, ctx) or x.Decode(l, ctx)
			// Exclude common false-positives (writer/reader/logger receivers).
			if sel.Sel.Name == "Encode" || sel.Sel.Name == "Decode" {
				recv := receiverTypeHint(sel.X)
				if !isWriterReaderReceiver(recv) {
					out = append(out, Call{
						Kind:        KindRecurse,
						RecurseType: recv,
						Line:        fset.Position(n.Pos()).Line,
						Guard:       conjoin(stack),
					})
					return
				}
			}
			if p, ok := primFromName(sel.Sel.Name); ok {
				out = append(out, Call{
					Kind:  KindWrite,
					Op:    p,
					Line:  fset.Position(n.Pos()).Line,
					Guard: conjoin(stack),
				})
			}
		default:
			ast.Inspect(node, func(c ast.Node) bool {
				if c == node {
					return true
				}
				if _, ok := c.(*ast.IfStmt); ok {
					walk(c)
					return false
				}
				if _, ok := c.(*ast.RangeStmt); ok {
					walk(c)
					return false
				}
				if _, ok := c.(*ast.ForStmt); ok {
					walk(c)
					return false
				}
				if ce, ok := c.(*ast.CallExpr); ok {
					if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
						// Recurse marker check
						if sel.Sel.Name == "Encode" || sel.Sel.Name == "Decode" {
							recv := receiverTypeHint(sel.X)
							if !isWriterReaderReceiver(recv) {
								out = append(out, Call{
									Kind:        KindRecurse,
									RecurseType: recv,
									Line:        fset.Position(ce.Pos()).Line,
									Guard:       conjoin(stack),
								})
								return false
							}
						}
						if p, ok := primFromName(sel.Sel.Name); ok {
							out = append(out, Call{
								Kind:  KindWrite,
								Op:    p,
								Line:  fset.Position(ce.Pos()).Line,
								Guard: conjoin(stack),
							})
						}
					}
				}
				return true
			})
		}
	}
	walk(b)
	return out
}

// collectGuardedSub collects body calls fresh — the parent Repeat already carries the outer guard.
func collectGuardedSub(b *ast.BlockStmt, fset *token.FileSet) []Call {
	return collectCalls(b, fset)
}

// receiverTypeHint returns a best-effort static type name for the receiver of
// a .Encode/.Decode call. For x.Encode(...): returns "x". For m.sub.Encode(...):
// returns "sub". Real type resolution requires a full type-check pass; Phase A
// uses the identifier text as a placeholder for the diff engine to surface.
func receiverTypeHint(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.Ident:
		// Could be a local variable name OR a package-level type name (for static methods).
		name := v.Name
		if name == "" {
			return ""
		}
		return name
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.CallExpr:
		// e.g. someChain().Encode — chain too deep to resolve
		return ""
	case *ast.IndexExpr:
		// e.g. arr[i].Encode
		return receiverTypeHint(v.X)
	}
	return ""
}

// isWriterReaderReceiver returns true for common local variable names that are
// writer/reader/logger instances (not sub-encoders), to avoid false-positive
// KindRecurse classification.
func isWriterReaderReceiver(name string) bool {
	switch name {
	case "w", "r", "l", "log", "ctx", "t":
		return true
	}
	return false
}

// guardFromIf extracts and compiles the condition expression of an if statement.
// If parsing fails (e.g. field-presence checks like `m.ipAddr != ""`), returns a
// GuardExpr that always evaluates true: we assume the common/full-payload code path
// for audit purposes.
func guardFromIf(n *ast.IfStmt, fset *token.FileSet) *GuardExpr {
	var buf strings.Builder
	printer.Fprint(&buf, fset, n.Cond)
	g, err := ParseGuard(buf.String())
	if err != nil {
		return &GuardExpr{eval: func(GuardContext) bool { return true }, text: "<unparsed:" + buf.String() + ">"}
	}
	return g
}

// conjoin combines a stack of guards into a single AND-ed GuardExpr.
// Returns nil if the stack is empty (unconditional call).
func conjoin(s []*GuardExpr) *GuardExpr {
	if len(s) == 0 {
		return nil
	}
	if len(s) == 1 {
		return s[0]
	}
	parts := make([]string, len(s))
	for i, g := range s {
		parts[i] = "(" + g.text + ")"
	}
	combined, err := ParseGuard(strings.Join(parts, " && "))
	if err != nil {
		return s[len(s)-1]
	}
	return combined
}

// negate wraps a guard expression in logical NOT.
func negate(g *GuardExpr) *GuardExpr {
	if g == nil {
		return nil
	}
	ng, err := ParseGuard("!(" + g.text + ")")
	if err != nil {
		return g
	}
	return ng
}

// primFromName maps a method name to its Primitive encoding width.
func primFromName(name string) (Primitive, bool) {
	switch name {
	case "WriteByte", "WriteBool", "ReadByte", "ReadBool":
		return Encode1, true
	case "WriteShort", "ReadUint16":
		return Encode2, true
	case "WriteInt", "ReadUint32":
		return Encode4, true
	case "WriteLong", "ReadUint64":
		return Encode8, true
	case "WriteAsciiString", "ReadAsciiString":
		return EncodeStr, true
	case "WriteBytes", "ReadBytes":
		return EncodeBuf, true
	case "WriteByteArray", "ReadByteArray":
		return EncodeBuf, true
	}
	return 0, false
}
