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

// declaresVarStringDecl reports whether s is a `var <name> string`
// declaration.
func declaresVarStringDecl(s ast.Stmt) bool {
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
		if ident, ok := vs.Type.(*ast.Ident); ok && ident.Name == "string" {
			return true
		}
	}
	return false
}

// errGuard builds `if err != nil { return err }`.
func errGuard() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: ast.NewIdent("err"), Op: token.NEQ, Y: ast.NewIdent("nil")},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("err")}},
		}},
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
		block  *ast.BlockStmt
		assign *ast.AssignStmt
	}
	var matches []match
	var funcStack []ast.Node

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
			matches = append(matches, match{block: block, assign: n})
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
	var blockOrder []*ast.BlockStmt
	for _, m := range matches {
		if _, ok := byBlock[m.block]; !ok {
			blockOrder = append(blockOrder, m.block)
		}
		byBlock[m.block] = append(byBlock[m.block], m.assign)
	}

	for _, block := range blockOrder {
		targets := make(map[*ast.AssignStmt]bool, len(byBlock[block]))
		for _, a := range byBlock[block] {
			targets[a] = true
		}
		block.List = rewriteBlockList(block.List, targets)
	}

	return true, residue
}

// rewriteBlockList rewrites list, replacing each target assignment's blank
// second LHS with `err` and inserting an `if err != nil { return err }`
// guard after it. If any target is an `=` assignment, a `var err error`
// declaration is inserted immediately after the block's existing
// `var t string` declaration, unless one already exists.
func rewriteBlockList(list []ast.Stmt, targets map[*ast.AssignStmt]bool) []ast.Stmt {
	hasVarErr := false
	for _, s := range list {
		if declaresVarErrError(s) {
			hasVarErr = true
			break
		}
	}

	needsHoist := false
	for _, s := range list {
		if as, ok := s.(*ast.AssignStmt); ok && targets[as] && as.Tok == token.ASSIGN {
			needsHoist = true
			break
		}
	}

	out := make([]ast.Stmt, 0, len(list)+2)
	for _, s := range list {
		out = append(out, s)
		if needsHoist && !hasVarErr && declaresVarStringDecl(s) {
			out = append(out, varErrDecl())
			hasVarErr = true
		}
		if as, ok := s.(*ast.AssignStmt); ok && targets[as] {
			as.Lhs[1] = ast.NewIdent("err")
			out = append(out, errGuard())
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

// rewriteNewConfig implements R4: retype the `NewConfig` curried token
// wrapper's `token string` parameter to `topic.Token`, and turn its
// discarded EnvProvider error into a fatal log.
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
		if !rewriteNewConfigAssignment(fd) {
			continue
		}
		retypeTokenParams(fd)
		changed = true
	}
	return changed, residue
}
