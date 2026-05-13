package atlaspacket

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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

// GuardExpr is a stub for Task 8; declared here so callers can reference it.
type GuardExpr struct{}

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

// collectCalls walks a block and collects all w.Write*/r.Read* primitive calls in order.
func collectCalls(b *ast.BlockStmt, fset *token.FileSet) []Call {
	var out []Call
	ast.Inspect(b, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if p, ok := primFromName(sel.Sel.Name); ok {
			out = append(out, Call{Op: p, Line: fset.Position(ce.Pos()).Line})
		}
		return true
	})
	return out
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
