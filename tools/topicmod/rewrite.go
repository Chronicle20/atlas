// Package topicmod implements an AST-based codemod that retypes
// Kafka topic-token constants and message-buffer keys from string to
// topic.Token (github.com/Chronicle20/atlas/libs/atlas-kafka/topic).
//
// It applies two type rules:
//
//   - R1: retypes topic-token constant declarations (`const Env... = "..."`)
//     whose string value matches the topic-token shape.
//   - R2: retypes the per-service message Buffer's map[string]... keys
//     (the field, GetAll's result, and Put's first parameter) to
//     topic.Token.
//
// Sites the rules cannot safely rewrite are reported as residue
// findings rather than silently skipped.
package topicmod

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// topicImportPath is the import path of the topic.Token type this codemod
// introduces at call sites.
const topicImportPath = "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

// topicTokenShape is the shape of a topic-token constant's string value,
// applied identically by the generator, the analyzer, and this codemod.
var topicTokenShape = regexp.MustCompile(`^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`)

// Finding describes a site the rewriter could not safely apply, or (in a
// future extension) any other diagnostic worth surfacing to the caller.
type Finding struct {
	Pos    token.Position
	Rule   string
	Reason string
}

// Rewrite applies R1 and R2 to f in place, returning whether it changed
// anything and any residue findings it could not safely rewrite.
func Rewrite(fset *token.FileSet, f *ast.File, path string) (changed bool, residue []Finding) {
	c1, r1 := rewriteDecls(fset, f)
	c2, r2 := rewriteBuffer(fset, f)

	changed = c1 || c2
	residue = append(residue, r1...)
	residue = append(residue, r2...)

	if changed {
		astutil.AddImport(fset, f, topicImportPath)
	}

	return changed, residue
}

// topicTokenType builds the `topic.Token` type expression this codemod
// substitutes for `string` at retyped sites.
func topicTokenType() *ast.SelectorExpr {
	return &ast.SelectorExpr{X: ast.NewIdent("topic"), Sel: ast.NewIdent("Token")}
}

// rewriteDecls implements R1: retype topic-token constant declarations.
func rewriteDecls(fset *token.FileSet, f *ast.File) (changed bool, residue []Finding) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if vs.Type != nil {
				// Already explicitly typed; nothing for R1 to do.
				continue
			}
			topicShaped, otherShaped := classifyConstValues(vs)
			switch {
			case len(vs.Values) == 0:
				// Value comes from iota/implicit repetition; not a
				// topic-token literal shape R1 targets.
				continue
			case len(vs.Names) == 1 && len(vs.Values) == 1:
				if topicShaped {
					vs.Type = topicTokenType()
					changed = true
				}
			case topicShaped && otherShaped:
				// A single ValueSpec carrying several names where only
				// some values are topic-shaped cannot be split without
				// changing the surrounding const block's grouping.
				residue = append(residue, Finding{
					Pos:    fset.Position(vs.Pos()),
					Rule:   "R1",
					Reason: "mixed const spec",
				})
			case topicShaped:
				vs.Type = topicTokenType()
				changed = true
			}
		}
	}
	return changed, residue
}

// classifyConstValues reports whether any of a ValueSpec's values is a
// topic-token-shaped string literal (topicShaped), and whether any value is
// not (otherShaped).
func classifyConstValues(vs *ast.ValueSpec) (topicShaped, otherShaped bool) {
	for _, v := range vs.Values {
		if isTopicTokenLiteral(v) {
			topicShaped = true
		} else {
			otherShaped = true
		}
	}
	return topicShaped, otherShaped
}

// isTopicTokenLiteral reports whether expr is a single string BasicLit
// whose unquoted value matches the topic-token shape.
func isTopicTokenLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return false
	}
	return topicTokenShape.MatchString(value)
}

// stringSliceMapType reports whether t is `map[string][]<expr>` for any
// element type expr; used to recognize the Buffer's `buffer` field and the
// type returned by GetAll.
func stringSliceMapType(t ast.Expr) (*ast.MapType, bool) {
	mt, ok := t.(*ast.MapType)
	if !ok {
		return nil, false
	}
	key, ok := mt.Key.(*ast.Ident)
	if !ok || key.Name != "string" {
		return nil, false
	}
	if _, ok := mt.Value.(*ast.ArrayType); !ok {
		return nil, false
	}
	return mt, true
}

// rewriteBuffer implements R2: retype the per-service message Buffer's
// map[string]... keys to topic.Token wherever the file declares
// `type Buffer struct`.
func rewriteBuffer(fset *token.FileSet, f *ast.File) (changed bool, residue []Finding) {
	if !declaresBufferStruct(f) {
		return false, nil
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != "Buffer" {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					if mt, ok := stringSliceMapType(field.Type); ok {
						mt.Key = topicTokenType()
						changed = true
					}
				}
			}
		case *ast.FuncDecl:
			if !isBufferMethod(d, "GetAll") && !isBufferMethod(d, "Put") {
				continue
			}
			switch d.Name.Name {
			case "GetAll":
				if d.Type.Results != nil {
					for _, res := range d.Type.Results.List {
						if mt, ok := stringSliceMapType(res.Type); ok {
							mt.Key = topicTokenType()
							changed = true
						}
					}
				}
			case "Put":
				if params := d.Type.Params; params != nil && len(params.List) > 0 {
					first := params.List[0]
					if ident, ok := first.Type.(*ast.Ident); ok && ident.Name == "string" {
						first.Type = topicTokenType()
						changed = true
					}
				}
			}
		}
	}

	// Any top-level Emit/EmitWithResult ranging over b.GetAll() with an
	// explicit `var t string` shadow is residue; the inferred range
	// variable itself needs no edit.
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil {
			continue
		}
		if fd.Name.Name != "Emit" && fd.Name.Name != "EmitWithResult" {
			continue
		}
		residue = append(residue, explicitStringBindingOverGetAllRange(fset, fd)...)
	}

	return changed, residue
}

// declaresBufferStruct reports whether f declares `type Buffer struct`.
func declaresBufferStruct(f *ast.File) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if ts.Name.Name != "Buffer" {
			return true
		}
		if _, ok := ts.Type.(*ast.StructType); ok {
			found = true
		}
		return true
	})
	return found
}

// isBufferMethod reports whether d is a method on *Buffer (or Buffer)
// named name.
func isBufferMethod(d *ast.FuncDecl, name string) bool {
	if d.Name.Name != name || d.Recv == nil || len(d.Recv.List) != 1 {
		return false
	}
	recvType := d.Recv.List[0].Type
	if star, ok := recvType.(*ast.StarExpr); ok {
		recvType = star.X
	}
	ident, ok := recvType.(*ast.Ident)
	return ok && ident.Name == "Buffer"
}

// explicitStringBindingOverGetAllRange scans fn for a `for _, _ := range
// b.GetAll()`-shaped range clause (or `for t, ... := range b.GetAll()`)
// carrying an explicit `var t string` shadow, which would defeat the
// inferred topic.Token range type. It reports residue when found; no edit
// is needed for the ordinary inferred case.
func explicitStringBindingOverGetAllRange(fset *token.FileSet, fn *ast.FuncDecl) (residue []Finding) {
	ast.Inspect(fn, func(n ast.Node) bool {
		rs, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		call, ok := rs.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "GetAll" {
			return true
		}

		// Look for an explicit `var t string` declaration shadowing the
		// range key inside the range body.
		ast.Inspect(rs.Body, func(bn ast.Node) bool {
			gd, ok := bn.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				return true
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || vs.Type == nil {
					continue
				}
				ident, ok := vs.Type.(*ast.Ident)
				if !ok || ident.Name != "string" {
					continue
				}
				residue = append(residue, Finding{
					Pos:    fset.Position(vs.Pos()),
					Rule:   "R2",
					Reason: "explicit string binding over GetAll range",
				})
			}
			return true
		})
		return true
	})
	return residue
}
