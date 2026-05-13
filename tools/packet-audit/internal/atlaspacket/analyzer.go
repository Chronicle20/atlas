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

// Call represents a single writer/reader primitive call found inside an Encode/Decode method.
type Call struct {
	Op    Primitive
	Line  int
	Guard *GuardExpr // nil for unconditional; populated in Task 8
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
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}
			if p, ok := primFromName(sel.Sel.Name); ok {
				out = append(out, Call{
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
				if ce, ok := c.(*ast.CallExpr); ok {
					if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
						if p, ok := primFromName(sel.Sel.Name); ok {
							out = append(out, Call{
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

// guardFromIf extracts and compiles the condition expression of an if statement.
// If parsing fails, returns a GuardExpr with text "<unparsed:...>" that always evaluates false.
func guardFromIf(n *ast.IfStmt, fset *token.FileSet) *GuardExpr {
	var buf strings.Builder
	printer.Fprint(&buf, fset, n.Cond)
	g, err := ParseGuard(buf.String())
	if err != nil {
		return &GuardExpr{eval: func(GuardContext) bool { return false }, text: "<unparsed:" + buf.String() + ">"}
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
	}
	return 0, false
}
