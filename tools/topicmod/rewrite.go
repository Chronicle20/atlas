// Package topicmod implements an AST-based codemod that retypes
// Kafka topic-token constants and message-buffer keys from string to
// topic.Token (github.com/Chronicle20/atlas/libs/atlas-kafka/topic), and
// makes callers handle the error topic.EnvProvider now returns instead of
// discarding it.
//
// It applies two type rules and two error rules:
//
//   - R1: retypes topic-token constant declarations (`const Env... = "..."`)
//     whose string value matches the topic-token shape.
//   - R2: retypes the per-service message Buffer's map[string]... keys
//     (the field, GetAll's result, and Put's first parameter) to
//     topic.Token.
//   - R3: propagates the error topic.EnvProvider(...)(...)() now returns,
//     for every discarded-error assignment inside a function that returns
//     error.
//   - R4: retypes the `NewConfig` curried token wrapper's `token string`
//     parameter to `topic.Token` and turns its discarded EnvProvider error
//     into a fatal log.
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

// Rewrite applies R4, R3, R1, and R2 (in that order) to f in place,
// returning whether it changed anything and any residue findings it could
// not safely rewrite. R4 runs before R3 so that R3 does not re-handle the
// discarded-error assignment R4 already turned into a fatal log.
func Rewrite(fset *token.FileSet, f *ast.File, path string) (changed bool, residue []Finding) {
	c4, r4 := rewriteNewConfig(fset, f)
	c3, r3 := rewriteEnvProviderErrors(fset, f)
	c1, r1 := rewriteDecls(fset, f)
	c2, r2 := rewriteBuffer(fset, f)

	changed = c4 || c3 || c1 || c2
	residue = append(residue, r4...)
	residue = append(residue, r3...)
	residue = append(residue, r1...)
	residue = append(residue, r2...)

	if changed {
		astutil.AddImport(fset, f, topicImportPath)
	}

	return changed, residue
}

// isEnvProviderCall reports whether e is the shape
// topic.EnvProvider(<any>)(<any>)().
func isEnvProviderCall(e ast.Expr) bool {
	outer, ok := e.(*ast.CallExpr)
	if !ok || len(outer.Args) != 0 {
		return false
	}
	mid, ok := outer.Fun.(*ast.CallExpr)
	if !ok || len(mid.Args) != 1 {
		return false
	}
	inner, ok := mid.Fun.(*ast.CallExpr)
	if !ok || len(inner.Args) != 1 {
		return false
	}
	sel, ok := inner.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "topic" && sel.Sel.Name == "EnvProvider"
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

	retypedFields := map[string]bool{}

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
						for _, name := range field.Names {
							retypedFields[name.Name] = true
						}
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

	// Retype any make(map[string][]T) initializer of a retyped buffer
	// field, e.g. NewBuffer()'s `&Buffer{buffer: make(map[string][]T)}`.
	if len(retypedFields) > 0 && rewriteBufferInit(f, retypedFields) {
		changed = true
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

// rewriteBufferInit retypes make(map[string][]T) initializer sites that
// initialize a buffer field retyped by rewriteBuffer, wherever they appear
// inside a composite literal constructing Buffer (e.g. NewBuffer's
// `&Buffer{buffer: make(map[string][]T)}`).
func rewriteBufferInit(f *ast.File, retypedFields map[string]bool) (changed bool) {
	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok || !isBufferCompositeLitType(cl.Type) {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || !retypedFields[key.Name] {
				continue
			}
			if retypeMakeMapCall(kv.Value) {
				changed = true
			}
		}
		return true
	})
	return changed
}

// isBufferCompositeLitType reports whether t is the `Buffer` identifier, the
// type of a composite literal constructing a Buffer value.
func isBufferCompositeLitType(t ast.Expr) bool {
	ident, ok := t.(*ast.Ident)
	return ok && ident.Name == "Buffer"
}

// retypeMakeMapCall reports whether e is `make(map[string][]T, ...)` and, if
// so, retypes the map's key to topic.Token.
func retypeMakeMapCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || fn.Name != "make" || len(call.Args) == 0 {
		return false
	}
	mt, ok := stringSliceMapType(call.Args[0])
	if !ok {
		return false
	}
	mt.Key = topicTokenType()
	return true
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

// isEnvProviderBlankAssign reports whether as is `_ = topic.EnvProvider(...)(...)()`
// with a blank second LHS element, in either `:=` or `=` form.
func isEnvProviderBlankAssign(as *ast.AssignStmt) bool {
	if as.Tok != token.DEFINE && as.Tok != token.ASSIGN {
		return false
	}
	if len(as.Lhs) != 2 || len(as.Rhs) != 1 {
		return false
	}
	blank, ok := as.Lhs[1].(*ast.Ident)
	if !ok || blank.Name != "_" {
		return false
	}
	return isEnvProviderCall(as.Rhs[0])
}

// funcReturnsError reports whether n (a *ast.FuncDecl or *ast.FuncLit)
// has a non-empty result list whose last result is `error`.
func funcReturnsError(n ast.Node) bool {
	var results *ast.FieldList
	switch fn := n.(type) {
	case *ast.FuncDecl:
		results = fn.Type.Results
	case *ast.FuncLit:
		results = fn.Type.Results
	}
	if results == nil || len(results.List) == 0 {
		return false
	}
	last := results.List[len(results.List)-1]
	ident, ok := last.Type.(*ast.Ident)
	return ok && ident.Name == "error"
}

// declaresVarErrError reports whether s is `var err error` (or a var decl
// naming err among its specs).
func declaresVarErrError(s ast.Stmt) bool {
	ds, ok := s.(*ast.DeclStmt)
	if !ok {
		return false
	}
	gd, ok := ds.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return false
	}
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range vs.Names {
			if name.Name == "err" {
				return true
			}
		}
	}
	return false
}

// declaresVarStringDeclNamed reports whether s is a `var <name> string`
// declaration for the given name.
func declaresVarStringDeclNamed(s ast.Stmt, name string) bool {
	ds, ok := s.(*ast.DeclStmt)
	if !ok {
		return false
	}
	gd, ok := ds.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return false
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
		for _, n := range vs.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

// declaresAnyVarStringDecl reports whether s is a `var <name> string`
// declaration for any name in names.
func declaresAnyVarStringDecl(s ast.Stmt, names map[string]bool) bool {
	for name := range names {
		if declaresVarStringDeclNamed(s, name) {
			return true
		}
	}
	return false
}

// assignTargetName reports the name of as's first LHS identifier — the
// variable an `=`-form EnvProvider discard assigns into.
func assignTargetName(as *ast.AssignStmt) (string, bool) {
	ident, ok := as.Lhs[0].(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

// stmtIndex returns the index of target within list, or -1 if not found.
func stmtIndex(list []ast.Stmt, target ast.Stmt) int {
	for i, s := range list {
		if s == target {
			return i
		}
	}
	return -1
}

// blockDeclaresStringVarBefore reports whether list contains a
// `var <name> string` declaration at an index strictly before targetIdx.
func blockDeclaresStringVarBefore(list []ast.Stmt, name string, targetIdx int) bool {
	for i := 0; i < targetIdx; i++ {
		if declaresVarStringDeclNamed(list[i], name) {
			return true
		}
	}
	return false
}

// errGuard builds `if err != nil { return err }` for a function that
// returns only error, or, for a function with additional result types
// preceding the trailing error, declares a zero value for each of them and
// returns those alongside err — e.g. `if err != nil { var zero0 T; return
// zero0, err }`. results is the enclosing function's result list (as
// returned by funcReturnsError's caller); it must have a trailing `error`
// result. Preceding result types errGuard cannot safely zero (anything but
// an identifier, selector, pointer, slice, array, or map type) fall back to
// the single-value `return err`, which remains correct only when there are
// no other results — callers must not invoke errGuard for a multi-return
// function whose non-error result types it cannot zero; rewriteEnvProviderErrors
// enforces this via canZeroResults.
func errGuard(results *ast.FieldList) *ast.IfStmt {
	body := []ast.Stmt{}
	values := []ast.Expr{}

	if results != nil && len(results.List) > 1 {
		zeroIdx := 0
		for _, field := range results.List[:len(results.List)-1] {
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				name := "zero" + strconv.Itoa(zeroIdx)
				zeroIdx++
				body = append(body, &ast.DeclStmt{Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{&ast.ValueSpec{
						Names: []*ast.Ident{ast.NewIdent(name)},
						Type:  cloneTypeExpr(field.Type),
					}},
				}})
				values = append(values, ast.NewIdent(name))
			}
		}
	}
	values = append(values, ast.NewIdent("err"))
	body = append(body, &ast.ReturnStmt{Results: values})

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{List: body},
	}
}

// cloneTypeExpr deep-copies the type expression shapes errGuard needs to
// emit a `var zeroN T` declaration without sharing AST nodes across
// multiple inserted guards.
func cloneTypeExpr(t ast.Expr) ast.Expr {
	switch v := t.(type) {
	case *ast.Ident:
		return ast.NewIdent(v.Name)
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: cloneTypeExpr(v.X), Sel: ast.NewIdent(v.Sel.Name)}
	case *ast.StarExpr:
		return &ast.StarExpr{X: cloneTypeExpr(v.X)}
	case *ast.ArrayType:
		var len_ ast.Expr
		if v.Len != nil {
			len_ = cloneTypeExpr(v.Len)
		}
		return &ast.ArrayType{Len: len_, Elt: cloneTypeExpr(v.Elt)}
	case *ast.MapType:
		return &ast.MapType{Key: cloneTypeExpr(v.Key), Value: cloneTypeExpr(v.Value)}
	case *ast.InterfaceType:
		return &ast.InterfaceType{Methods: &ast.FieldList{}}
	case *ast.BasicLit:
		return &ast.BasicLit{Kind: v.Kind, Value: v.Value}
	default:
		// Unhandled type shape: return it as-is. Sharing the node across
		// guards is a formatting/positional wrinkle at worst, not a
		// correctness one, since each guard's `var zeroN T` only reads the
		// shared node rather than mutating it.
		return t
	}
}

// canZeroResults reports whether errGuard can safely synthesize a zero
// value for every result preceding the trailing error in results.
func canZeroResults(results *ast.FieldList) bool {
	if results == nil || len(results.List) <= 1 {
		return true
	}
	for _, field := range results.List[:len(results.List)-1] {
		if !isZeroableType(field.Type) {
			return false
		}
	}
	return true
}

// isZeroableType reports whether cloneTypeExpr (and therefore a `var zeroN
// T` declaration) can handle t.
func isZeroableType(t ast.Expr) bool {
	switch v := t.(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr, *ast.ArrayType, *ast.MapType, *ast.InterfaceType:
		return true
	case *ast.Ellipsis:
		return false
	default:
		_ = v
		return false
	}
}

// varErrDecl builds `var err error`.
func varErrDecl() *ast.DeclStmt {
	return &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("err")},
			Type:  ast.NewIdent("error"),
		}},
	}}
}

// rewriteEnvProviderErrors implements R3: for every assignment whose RHS is
// `topic.EnvProvider(...)(...)()` and whose second LHS element is the blank
// identifier, propagate the error instead of discarding it. Assignments
// inside a function that does not return error, or not directly inside a
// block statement, are reported as residue rather than rewritten.
func rewriteEnvProviderErrors(fset *token.FileSet, f *ast.File) (changed bool, residue []Finding) {
	type match struct {
		block   *ast.BlockStmt
		assign  *ast.AssignStmt
		results *ast.FieldList
	}
	var matches []match
	var funcStack []ast.Node

	funcResults := func(n ast.Node) *ast.FieldList {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			return fn.Type.Results
		case *ast.FuncLit:
			return fn.Type.Results
		}
		return nil
	}

	astutil.Apply(f, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.FuncDecl:
			funcStack = append(funcStack, n)
		case *ast.FuncLit:
			funcStack = append(funcStack, n)
		case *ast.AssignStmt:
			if !isEnvProviderBlankAssign(n) {
				return true
			}
			block, ok := c.Parent().(*ast.BlockStmt)
			if !ok {
				residue = append(residue, Finding{
					Pos:    fset.Position(n.Pos()),
					Rule:   "R3",
					Reason: "assignment not directly in a block statement",
				})
				return true
			}
			if len(funcStack) == 0 || !funcReturnsError(funcStack[len(funcStack)-1]) {
				residue = append(residue, Finding{
					Pos:    fset.Position(n.Pos()),
					Rule:   "R3",
					Reason: "enclosing function does not return error",
				})
				return true
			}
			results := funcResults(funcStack[len(funcStack)-1])
			if !canZeroResults(results) {
				residue = append(residue, Finding{
					Pos:    fset.Position(n.Pos()),
					Rule:   "R3",
					Reason: "enclosing function's non-error result type cannot be safely zeroed",
				})
				return true
			}
			matches = append(matches, match{block: block, assign: n, results: results})
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch c.Node().(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			funcStack = funcStack[:len(funcStack)-1]
		}
		return true
	})

	if len(matches) == 0 {
		return false, residue
	}

	byBlock := map[*ast.BlockStmt][]*ast.AssignStmt{}
	blockResults := map[*ast.BlockStmt]*ast.FieldList{}
	var blockOrder []*ast.BlockStmt
	for _, m := range matches {
		if _, ok := byBlock[m.block]; !ok {
			blockOrder = append(blockOrder, m.block)
			blockResults[m.block] = m.results
		}
		byBlock[m.block] = append(byBlock[m.block], m.assign)
	}

	for _, block := range blockOrder {
		targets := make(map[*ast.AssignStmt]bool, len(byBlock[block]))
		for _, a := range byBlock[block] {
			if a.Tok == token.ASSIGN {
				name, ok := assignTargetName(a)
				idx := stmtIndex(block.List, a)
				if !ok || idx < 0 || !blockDeclaresStringVarBefore(block.List, name, idx) {
					// The `=`-form hoist heuristic only knows how to insert
					// `var err error` after a `var <name> string`
					// declaration for the exact variable this assignment's
					// LHS reuses. Without one — a named return value, a
					// variable from an outer scope, or simply not present
					// before this site in the block — there is nowhere safe
					// to hoist, so this site is residue rather than a guess.
					residue = append(residue, Finding{
						Pos:    fset.Position(a.Pos()),
						Rule:   "R3",
						Reason: "assignment-form EnvProvider discard has no preceding `var <name> string` declaration in the same block to hoist `var err error` after",
					})
					continue
				}
			}
			targets[a] = true
		}
		if len(targets) == 0 {
			continue
		}
		block.List = rewriteBlockList(block.List, targets, blockResults[block])
		changed = true
	}

	return changed, residue
}

// rewriteBlockList rewrites list, replacing each target assignment's blank
// second LHS with `err` and inserting an `if err != nil { return err }`
// guard after it. If any target is an `=` assignment, a `var err error`
// declaration is inserted immediately after the block's `var <name> string`
// declaration for that assignment's own LHS identifier (verified by the
// caller to exist), unless a `var err error` already exists.
func rewriteBlockList(list []ast.Stmt, targets map[*ast.AssignStmt]bool, results *ast.FieldList) []ast.Stmt {
	hasVarErr := false
	for _, s := range list {
		if declaresVarErrError(s) {
			hasVarErr = true
			break
		}
	}

	hoistNames := map[string]bool{}
	for as := range targets {
		if as.Tok == token.ASSIGN {
			if name, ok := assignTargetName(as); ok {
				hoistNames[name] = true
			}
		}
	}
	needsHoist := len(hoistNames) > 0

	out := make([]ast.Stmt, 0, len(list)+2)
	for _, s := range list {
		out = append(out, s)
		if needsHoist && !hasVarErr && declaresAnyVarStringDecl(s, hoistNames) {
			out = append(out, varErrDecl())
			hasVarErr = true
		}
		if as, ok := s.(*ast.AssignStmt); ok && targets[as] {
			as.Lhs[1] = ast.NewIdent("err")
			out = append(out, errGuard(results))
		}
	}
	return out
}

// matchStringParam reports whether ft has exactly one parameter named name
// of type `string`.
func matchStringParam(ft *ast.FuncType, name string) bool {
	if ft.Params == nil || len(ft.Params.List) != 1 {
		return false
	}
	field := ft.Params.List[0]
	if len(field.Names) != 1 || field.Names[0].Name != name {
		return false
	}
	ident, ok := field.Type.(*ast.Ident)
	return ok && ident.Name == "string"
}

// isConsumerConfigSelector reports whether t is `consumer.Config`.
func isConsumerConfigSelector(t ast.Expr) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "consumer" && sel.Sel.Name == "Config"
}

// isNewConfigCurriedChain reports whether t is the curried
// `func(name string) func(token string) func(groupId string) consumer.Config`
// chain R4 targets.
func isNewConfigCurriedChain(t ast.Expr) bool {
	outer, ok := t.(*ast.FuncType)
	if !ok || !matchStringParam(outer, "name") || outer.Results == nil || len(outer.Results.List) != 1 {
		return false
	}
	middle, ok := outer.Results.List[0].Type.(*ast.FuncType)
	if !ok || !matchStringParam(middle, "token") || middle.Results == nil || len(middle.Results.List) != 1 {
		return false
	}
	inner, ok := middle.Results.List[0].Type.(*ast.FuncType)
	if !ok || !matchStringParam(inner, "groupId") || inner.Results == nil || len(inner.Results.List) != 1 {
		return false
	}
	return isConsumerConfigSelector(inner.Results.List[0].Type)
}

// retypeTokenParams retypes every `token string` parameter reachable from
// fd (both the declared curried signature and any matching func literal in
// the body) to `token topic.Token`.
func retypeTokenParams(fd *ast.FuncDecl) {
	ast.Inspect(fd, func(n ast.Node) bool {
		ft, ok := n.(*ast.FuncType)
		if !ok || !matchStringParam(ft, "token") {
			return true
		}
		field := ft.Params.List[0]
		pos := field.Type.Pos()
		field.Type = &ast.SelectorExpr{
			X:   &ast.Ident{NamePos: pos, Name: "topic"},
			Sel: &ast.Ident{NamePos: pos, Name: "Token"},
		}
		return true
	})
}

// fatalGuard builds `if err != nil { l.WithError(err).Fatalf("unresolvable
// topic token [%s]", token) }`.
func fatalGuard(errIdent, tokenIdent *ast.Ident) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.CallExpr{
						Fun:  &ast.SelectorExpr{X: ast.NewIdent("l"), Sel: ast.NewIdent("WithError")},
						Args: []ast.Expr{errIdent},
					},
					Sel: ast.NewIdent("Fatalf"),
				},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: `"unresolvable topic token [%s]"`},
					tokenIdent,
				},
			}},
		}},
	}
}

// rewriteNewConfigAssignment finds `t, _ := topic.EnvProvider(l)(token)()`
// inside fd's body and rewrites it to propagate the error to a fatal log
// instead of discarding it. It reports whether it found and rewrote a
// match.
func rewriteNewConfigAssignment(fd *ast.FuncDecl) bool {
	changed := false
	astutil.Apply(fd.Body, func(c *astutil.Cursor) bool {
		as, ok := c.Node().(*ast.AssignStmt)
		if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 2 || len(as.Rhs) != 1 {
			return true
		}
		blank, ok := as.Lhs[1].(*ast.Ident)
		if !ok || blank.Name != "_" {
			return true
		}
		if !isEnvProviderCall(as.Rhs[0]) {
			return true
		}
		call := as.Rhs[0].(*ast.CallExpr)
		mid := call.Fun.(*ast.CallExpr)
		tokenIdent, ok := mid.Args[0].(*ast.Ident)
		if !ok {
			return true
		}

		errIdent := ast.NewIdent("err")
		as.Lhs[1] = errIdent
		c.InsertAfter(fatalGuard(errIdent, tokenIdent))
		changed = true
		return false
	}, nil)
	return changed
}

// isNewConfigDelegate reports whether fd's body is a thin one-statement
// delegate — `return <expr>.NewConfig(<l>)` — forwarding to another
// package's `NewConfig` with fd's own (single) logger parameter. This is
// the shape of the ~60 per-service `NewConfig` wrappers that just call the
// real implementation another R4 match already retypes; retyping its
// `token string` parameter to `topic.Token` keeps the delegate's signature
// in lockstep with the real implementation without touching its body.
func isNewConfigDelegate(fd *ast.FuncDecl) bool {
	if fd.Body == nil || len(fd.Body.List) != 1 {
		return false
	}
	ret, ok := fd.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "NewConfig" {
		return false
	}
	arg, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return false
	}
	if fd.Type.Params == nil || len(fd.Type.Params.List) != 1 {
		return false
	}
	param := fd.Type.Params.List[0]
	if len(param.Names) != 1 {
		return false
	}
	return arg.Name == param.Names[0].Name
}

// rewriteNewConfig implements R4: retype the `NewConfig` curried token
// wrapper's `token string` parameter to `topic.Token`, and turn its
// discarded EnvProvider error into a fatal log. A `NewConfig` whose
// signature matches the curried chain but whose body is neither the direct
// `EnvProvider` assignment nor a thin delegate to another `NewConfig` is
// reported as residue rather than silently left untouched.
func rewriteNewConfig(fset *token.FileSet, f *ast.File) (changed bool, residue []Finding) {
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Name.Name != "NewConfig" {
			continue
		}
		if fd.Type.Results == nil || len(fd.Type.Results.List) != 1 {
			continue
		}
		if !isNewConfigCurriedChain(fd.Type.Results.List[0].Type) {
			continue
		}
		switch {
		case rewriteNewConfigAssignment(fd):
			retypeTokenParams(fd)
			changed = true
		case isNewConfigDelegate(fd):
			retypeTokenParams(fd)
			changed = true
		default:
			residue = append(residue, Finding{
				Pos:    fset.Position(fd.Pos()),
				Rule:   "R4",
				Reason: "NewConfig signature matches curried chain but body is neither the direct EnvProvider assignment nor a thin delegate to another NewConfig",
			})
		}
	}
	return changed, residue
}
