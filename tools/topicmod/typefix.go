// This file implements the second half of the topicmod codemod: a
// compiler-diagnostic-driven pass that retypes the test-file (and residual
// production-file) call sites `go vet` reports once R1-R4 have retyped a
// package's topic-token constants, Buffer, and NewConfig wrapper.
//
// R1-R4 are shape-driven: they recognize a fixed AST pattern and rewrite it
// regardless of whether doing so is currently necessary. That approach does
// not scale to the ~190 test-file compile errors R1-R4's `string` ->
// `topic.Token` retype produces downstream — the callee names, receiver
// types, and surrounding shapes differ in every package. Rather than
// enumerate every one of those shapes by hand, FixModule asks the Go
// compiler what is still wrong (via `go vet`, in the same text format as
// task-4c-compile-errors.txt), locates the exact AST node the diagnostic
// names, and applies one of a small number of generic, direction-derived
// edits:
//
//   - "cannot use <expr> (... topic.Token) as string value in <context>"
//     — <expr> is already correctly typed; the declaration <context> names
//     (a callee's parameter, a map's key type, a struct field, a named
//     local func-type's parameter) is still `string` and needs retyping.
//     If <context> is not a local declaration (e.g. a stdlib call like
//     os.LookupEnv or (*testing.T).Setenv), the token cannot be widened
//     back to string at its own type, so the argument is wrapped in
//     `string(...)` instead — the one documented exception.
//   - "cannot use <expr> (... string) as topic.Token value in <context>"
//     (or as a map[topic.Token]..., a named Provider-shaped type, etc.) —
//     <expr> itself, found directly at the diagnostic's file:line:col, is
//     still string-shaped and is retyped or wrapped in `topic.Token(...)`.
//   - "invalid operation: A != B (mismatched types string and topic.Token)"
//     — whichever operand's reported type is "string" is fixed the same
//     way as the topic.Token-expected case above.
//
// FixModule iterates `go vet ./...` + fix + rewrite-to-disk until a run
// produces no further edits, then returns whatever diagnostics remain as
// residue for a human (or a future rule) to resolve by hand.
package topicmod

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// maxFixIterations bounds FixModule's compile-fix-recompile loop so a
// pathological case (an edit that keeps producing a new, different
// diagnostic at the same site) fails loudly instead of looping forever.
const maxFixIterations = 25

// Diagnostic is one `<path>:<line>:<col>: <message>` line of `go vet`
// output.
type Diagnostic struct {
	Path    string
	Line    int
	Col     int
	Message string
}

// diagLineRe matches a `go vet ./...` diagnostic line. Compiler/type-checker
// failures that abort vet before its own analyzers run are prefixed "vet: "
// (analyzer findings proper are not); both are accepted.
var diagLineRe = regexp.MustCompile(`^(?:vet: )?(\S+\.go):(\d+):(\d+): (.+)$`)

// ParseDiagnostics extracts every `[vet: ]path:line:col: message` line from
// output (a `go vet ./...` run's combined stdout+stderr), ignoring package
// header lines ("# github.com/...") and any other line that doesn't match
// the shape.
func ParseDiagnostics(output string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(output, "\n") {
		m := diagLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		out = append(out, Diagnostic{Path: m[1], Line: line, Col: col, Message: m[4]})
	}
	return out
}

// FixModule runs `go vet ./...` in dir (a Go module root) and applies
// AST edits for every diagnostic it can resolve, repeating until a round
// makes no further edits. It returns the diagnostics from the final round
// as residue.
func FixModule(dir string) ([]Finding, error) {
	var residue []Finding
	for i := 0; i < maxFixIterations; i++ {
		out, _ := runGoVet(dir)
		diags := ParseDiagnostics(out)
		if len(diags) == 0 {
			return nil, nil
		}

		byDir := map[string][]Diagnostic{}
		for _, d := range diags {
			full := filepath.Join(dir, d.Path)
			byDir[filepath.Dir(full)] = append(byDir[filepath.Dir(full)], d)
		}

		changedAny := false
		residue = residue[:0]
		var dirs []string
		for pkgDir := range byDir {
			dirs = append(dirs, pkgDir)
		}
		sort.Strings(dirs)
		for _, pkgDir := range dirs {
			changed, res, err := fixPackageDir(pkgDir, byDir[pkgDir])
			if err != nil {
				return nil, err
			}
			if changed {
				changedAny = true
			}
			residue = append(residue, res...)
		}
		if !changedAny {
			return residue, nil
		}
	}
	return residue, nil
}

func runGoVet(dir string) (string, error) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run() // a non-zero exit is expected while diagnostics remain
	return buf.String(), nil
}

// pkgScope holds every file parsed for one directory's fix pass, so
// resolvers can search across files in the same package (a mock type
// declared in one _test.go and used from another, for instance) rather
// than being limited to the single file a diagnostic names.
type pkgScope struct {
	fset     *token.FileSet
	files    map[string]*ast.File // absolute path -> parsed file
	filePath map[*ast.File]string // reverse lookup, for marking a file dirty from wherever it was matched
	dirty    map[string]bool      // absolute path -> needs writing back
}

func newPkgScope(fset *token.FileSet, files map[string]*ast.File) *pkgScope {
	filePath := make(map[*ast.File]string, len(files))
	for path, f := range files {
		filePath[f] = path
	}
	return &pkgScope{fset: fset, files: files, filePath: filePath, dirty: map[string]bool{}}
}

// markDirty records that f was mutated and must be written back, regardless
// of which file's diagnostic triggered the edit — a fix can retype a
// callee's declaration in a different file than the call site the
// diagnostic named.
func (p *pkgScope) markDirty(f *ast.File) {
	if path, ok := p.filePath[f]; ok {
		p.dirty[path] = true
	}
}

// fixPackageDir parses every .go file directly inside dir (non-recursive —
// each diagnostic's directory is already one package), applies a fix for
// each diagnostic it can resolve, and writes back every file the pass
// touched. It reports residue for diagnostics it could not resolve.
func fixPackageDir(dir string, diags []Diagnostic) (changed bool, residue []Finding, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, nil, err
	}

	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			// A file that fails to parse is not one topicmod introduced;
			// skip it rather than aborting the whole run.
			continue
		}
		files[path] = f
	}
	pkg := newPkgScope(fset, files)

	for _, d := range diags {
		full := filepath.Join(dir, filepath.Base(d.Path))
		f, ok := files[full]
		if !ok {
			continue
		}
		if applyDiagnosticFix(pkg, f, d) {
			changed = true
			// Every mutating path marks the file(s) it actually touched —
			// which is not always the diagnostic's own file: a "cannot use
			// ... as string value in argument to F" fix retypes F's
			// parameter in whichever file declares F, and never touches
			// the call site itself.
		} else {
			residue = append(residue, Finding{
				Pos:    token.Position{Filename: full, Line: d.Line, Column: d.Col},
				Rule:   "typefix",
				Reason: d.Message,
			})
		}
	}

	if !changed {
		return false, residue, nil
	}

	for path := range pkg.dirty {
		astutil.AddImport(fset, files[path], topicImportPath)
	}
	for path, f := range files {
		if !pkg.dirty[path] {
			continue
		}
		var buf bytes.Buffer
		if ferr := format.Node(&buf, fset, f); ferr != nil {
			return changed, residue, ferr
		}
		info, statErr := os.Stat(path)
		mode := os.FileMode(0o644)
		if statErr == nil {
			mode = info.Mode().Perm()
		}
		if werr := os.WriteFile(path, buf.Bytes(), mode); werr != nil {
			return changed, residue, werr
		}
	}
	return changed, residue, nil
}

var (
	reArgument  = regexp.MustCompile(`^cannot use .+ as (.+) value in argument to (\S+)$`)
	reMapIndex  = regexp.MustCompile(`^cannot use .+ as (.+) value in map index$`)
	reStructLit = regexp.MustCompile(`^cannot use .+ as (.+) value in struct literal$`)
	reReturn    = regexp.MustCompile(`^cannot use .+ as (.+) value in return statement$`)
	reVarDecl   = regexp.MustCompile(`^cannot use .+ as (.+) value in variable declaration$`)
	reMismatch  = regexp.MustCompile(`^invalid operation: .+ \(mismatched types (\S+) and (\S+)\)$`)
	identOnlyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// applyDiagnosticFix dispatches a single Diagnostic to the resolver for its
// message shape, mutating pkg's parsed files in place. It reports whether
// it found and applied a fix.
func applyDiagnosticFix(pkg *pkgScope, file *ast.File, d Diagnostic) bool {
	if m := reArgument.FindStringSubmatch(d.Message); m != nil {
		expected, calleeText := m[1], m[2]
		if fixExpectedNamedType(pkg, expected) {
			return true
		}
		call, argIdx := findCallArgAt(pkg.fset, file, d.Line, d.Col)
		if call == nil {
			return false
		}
		if expected == "string" {
			return retypeCalleeParam(pkg, file, call, argIdx, calleeText)
		}
		return genericTopicFix(pkg, file, &call.Args[argIdx])
	}

	if m := reMapIndex.FindStringSubmatch(d.Message); m != nil {
		expected := m[1]
		if fixExpectedNamedType(pkg, expected) {
			return true
		}
		idx := findIndexExprAt(pkg.fset, file, d.Line, d.Col)
		if idx == nil {
			return false
		}
		if expected == "string" {
			body, ftype := enclosingFunc(file, idx)
			slot, ok := resolveDeclTypeSlot(pkg, body, ftype, idx.X)
			if !ok || !retypeTypeExpr(slot) {
				return false
			}
			// resolveDeclTypeSlot's SelectorExpr case already marked its
			// (possibly different) owning file dirty; an Ident-resolved
			// param/local var slot lives in file itself.
			pkg.markDirty(file)
			return true
		}
		return genericTopicFix(pkg, file, &idx.Index)
	}

	if m := reStructLit.FindStringSubmatch(d.Message); m != nil {
		expected := m[1]
		if fixExpectedNamedType(pkg, expected) {
			return true
		}
		keyName, slot := findCompositeElementAt(pkg.fset, file, d.Line, d.Col)
		if slot == nil {
			return false
		}
		if expected == "string" {
			if keyName == "" {
				return false
			}
			return retypeFieldsNamed(pkg, keyName)
		}
		return genericTopicFix(pkg, file, slot)
	}

	if m := reReturn.FindStringSubmatch(d.Message); m != nil {
		expected := m[1]
		if fixExpectedNamedType(pkg, expected) {
			return true
		}
		slot, node := findReturnResultAt(pkg.fset, file, d.Line, d.Col)
		if slot == nil {
			return false
		}
		if expected == "string" {
			_, ftype := enclosingFunc(file, node)
			if ftype == nil || ftype.Results == nil {
				return false
			}
			field, _ := fieldAtIndex(ftype.Results, returnResultIndex(node, slot))
			if field == nil || !retypeTypeExpr(&field.Type) {
				return false
			}
			pkg.markDirty(file)
			return true
		}
		return genericTopicFix(pkg, file, slot)
	}

	if m := reVarDecl.FindStringSubmatch(d.Message); m != nil {
		expected := m[1]
		if fixExpectedNamedType(pkg, expected) {
			return true
		}
		slot := findVarDeclValueAt(pkg.fset, file, d.Line, d.Col)
		if slot == nil {
			return false
		}
		if expected == "string" {
			return false
		}
		return genericTopicFix(pkg, file, slot)
	}

	if m := reMismatch.FindStringSubmatch(d.Message); m != nil {
		t1, t2 := m[1], m[2]
		be := findBinaryExprAt(pkg.fset, file, d.Line, d.Col)
		if be == nil {
			return false
		}
		switch {
		case t1 == "string":
			return genericTopicFix(pkg, file, &be.X)
		case t2 == "string":
			return genericTopicFix(pkg, file, &be.Y)
		default:
			return false
		}
	}

	return false
}

// fixExpectedNamedType handles the "expected type" naming a local,
// non-builtin type — e.g. a service's own `type localProvider func(token
// string) producer.MessageProducer` — by retyping that named func type's
// first parameter directly, rather than trying to coerce whatever
// expression the diagnostic points at into matching it structurally.
func fixExpectedNamedType(pkg *pkgScope, expected string) bool {
	if expected == "string" || !identOnlyRe.MatchString(expected) {
		return false
	}
	for path, f := range pkg.files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != expected {
					continue
				}
				ft, ok := ts.Type.(*ast.FuncType)
				if !ok {
					continue
				}
				if retypeFirstStringParam(ft) {
					pkg.dirty[path] = true
					return true
				}
			}
		}
	}
	return false
}

// retypeCalleeParam resolves calleeText (as it appears in a "argument to
// X" diagnostic — a bare function name, or "recv.Method") to a FuncDecl
// declared somewhere in pkg's files and retypes its parameter at argIdx
// from `string` to `topic.Token`. If no matching local declaration exists
// (the callee is an external/stdlib function whose signature topicmod
// cannot change — os.LookupEnv, (*testing.T).Setenv), it instead narrows
// the already-topic.Token-typed argument at the call site with an explicit
// `string(...)` conversion, the one documented exception to retyping
// upward.
func retypeCalleeParam(pkg *pkgScope, file *ast.File, call *ast.CallExpr, argIdx int, calleeText string) bool {
	name := calleeText
	isMethod := false
	if i := strings.LastIndex(calleeText, "."); i >= 0 {
		name = calleeText[i+1:]
		isMethod = true
	}
	fd, fdFile := findFuncDeclByName(pkg, name, isMethod)
	if fd == nil {
		call.Args[argIdx] = wrapConversion("string", call.Args[argIdx])
		pkg.markDirty(file)
		return true
	}
	field, _ := fieldAtIndex(fd.Type.Params, argIdx)
	if field == nil {
		return false
	}
	if !retypeTypeExpr(&field.Type) {
		return false
	}
	pkg.markDirty(fdFile)
	return true
}

// genericTopicFix makes the expression at *slot (found directly at a
// diagnostic's file:line:col) satisfy an expected topic.Token-shaped type:
// a func literal's leading parameter is retyped in place; a call to a
// locally declared function/method is resolved to that declaration's
// returned func-type parameter; a map composite literal's key is retyped;
// anything else (a plain identifier, a selector, another call) is wrapped
// in an explicit `topic.Token(...)` conversion at the use site.
func genericTopicFix(pkg *pkgScope, file *ast.File, slot *ast.Expr) bool {
	switch v := (*slot).(type) {
	case *ast.FuncLit:
		if !retypeFirstStringParam(v.Type) {
			return false
		}
		pkg.markDirty(file)
		return true
	case *ast.CallExpr:
		if retypeCalleeReturnedFuncTypeParam(pkg, v) {
			return true
		}
		if isConversionOf(v, "topic", "Token") {
			return false
		}
		*slot = wrapConversion("topic.Token", v)
		pkg.markDirty(file)
		return true
	case *ast.CompositeLit:
		if v.Type != nil && retypeTypeExpr(&v.Type) {
			pkg.markDirty(file)
			return true
		}
		return false
	default:
		if isConversionOf(v, "topic", "Token") {
			return false
		}
		*slot = wrapConversion("topic.Token", v)
		pkg.markDirty(file)
		return true
	}
}

// isConversionOf reports whether e is already `pkg.Sel(...)` — used to
// avoid re-wrapping an expression a previous round already converted.
func isConversionOf(e ast.Expr, pkgName, sel string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	s, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := s.X.(*ast.Ident)
	return ok && x.Name == pkgName && s.Sel.Name == sel
}

// wrapConversion builds `<typeName>(e)`, where typeName is "string" or
// "topic.Token".
func wrapConversion(typeName string, e ast.Expr) ast.Expr {
	var fn ast.Expr
	if typeName == "string" {
		fn = ast.NewIdent("string")
	} else {
		fn = topicTokenType()
	}
	return &ast.CallExpr{Fun: fn, Args: []ast.Expr{e}}
}

// retypeFirstStringParam retypes ft's first parameter to topic.Token if it
// is currently `string`. Every provider/emitter closure shape this codemod
// targets carries its topic-token parameter first.
func retypeFirstStringParam(ft *ast.FuncType) bool {
	if ft.Params == nil || len(ft.Params.List) == 0 {
		return false
	}
	return retypeTypeExpr(&ft.Params.List[0].Type)
}

// retypeCalleeReturnedFuncTypeParam resolves call's callee to a local
// FuncDecl and, if its sole result is a func type (or a locally declared
// named type whose underlying type is a func type), retypes that returned
// function's first string parameter to topic.Token — the shape of a mock's
// `func (m *Mock) Provider() func(token string) producer.MessageProducer`
// factory method.
func retypeCalleeReturnedFuncTypeParam(pkg *pkgScope, call *ast.CallExpr) bool {
	var name string
	var isMethod bool
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		name, isMethod = fn.Name, false
	case *ast.SelectorExpr:
		name, isMethod = fn.Sel.Name, true
	default:
		return false
	}
	fd, fdFile := findFuncDeclByName(pkg, name, isMethod)
	if fd == nil || fd.Type.Results == nil || len(fd.Type.Results.List) != 1 {
		return false
	}
	switch rt := fd.Type.Results.List[0].Type.(type) {
	case *ast.FuncType:
		if !retypeFirstStringParam(rt) {
			return false
		}
		pkg.markDirty(fdFile)
		return true
	case *ast.Ident:
		return fixExpectedNamedType(pkg, rt.Name)
	default:
		return false
	}
}

// retypeFieldsNamed retypes every `<name> string` struct field (in a named
// or anonymous struct type, in any of pkg's files) to topic.Token.
func retypeFieldsNamed(pkg *pkgScope, name string) bool {
	changed := false
	for path, f := range pkg.files {
		ast.Inspect(f, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			for _, field := range st.Fields.List {
				for _, fn := range field.Names {
					if fn.Name == name && retypeTypeExpr(&field.Type) {
						changed = true
						pkg.dirty[path] = true
					}
				}
			}
			return true
		})
	}
	return changed
}

// retypeTypeExpr rewrites *slot from `string` to `topic.Token`, unwrapping
// through `*T` and `map[string]V` (retyping the key) as needed. It reports
// whether it changed anything.
func retypeTypeExpr(slot *ast.Expr) bool {
	switch t := (*slot).(type) {
	case *ast.Ident:
		if t.Name == "string" {
			*slot = topicTokenType()
			return true
		}
	case *ast.StarExpr:
		return retypeTypeExpr(&t.X)
	case *ast.MapType:
		return retypeTypeExpr(&t.Key)
	}
	return false
}

// findFuncDeclByName searches every file in files for a FuncDecl named
// name whose receiver-ness matches isMethod. A package-qualified external
// call (os.LookupEnv, (*testing.T).Setenv) is indistinguishable at the AST
// level from a same-package method call, so this deliberately returns nil
// for both "genuinely no such method" and "it's actually external" —
// callers fall back to a `string(...)` conversion either way, which is
// only reached when there truly is no local declaration to retype.
func findFuncDeclByName(pkg *pkgScope, name string, isMethod bool) (*ast.FuncDecl, *ast.File) {
	for _, f := range pkg.files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name.Name != name {
				continue
			}
			if (fd.Recv != nil) != isMethod {
				continue
			}
			return fd, f
		}
	}
	return nil, nil
}

// fieldAtIndex returns the *ast.Field governing the argIdx'th expanded
// parameter/result in fl (accounting for `a, b string` grouping), and its
// position within that field's Names.
func fieldAtIndex(fl *ast.FieldList, argIdx int) (*ast.Field, int) {
	if fl == nil {
		return nil, 0
	}
	i := 0
	for _, field := range fl.List {
		n := len(field.Names)
		if n == 0 {
			n = 1
		}
		if argIdx < i+n {
			return field, argIdx - i
		}
		i += n
	}
	return nil, 0
}

// resolveDeclTypeSlot finds the *ast.Expr type node governing x's static
// type: a function parameter (searched in ftype), a local `var` (searched
// in body), the map/type literal a `:=` short declaration assigns
// (searched in body), or — for a selector — a struct field found by name
// across the whole package.
func resolveDeclTypeSlot(pkg *pkgScope, body *ast.BlockStmt, ftype *ast.FuncType, x ast.Expr) (*ast.Expr, bool) {
	switch e := x.(type) {
	case *ast.Ident:
		if ftype != nil && ftype.Params != nil {
			for _, field := range ftype.Params.List {
				for _, n := range field.Names {
					if n.Name == e.Name {
						return &field.Type, true
					}
				}
			}
		}
		if body == nil {
			return nil, false
		}
		if vs := findVarSpec(body, e.Name); vs != nil && vs.Type != nil {
			return &vs.Type, true
		}
		if as, idx := findAssignRHS(body, e.Name, e.Pos()); as != nil {
			return findTypeNodeInExpr(&as.Rhs[idx])
		}
		return nil, false
	case *ast.SelectorExpr:
		field, ownerFile := findStructFieldSlot(pkg, e.Sel.Name)
		if field == nil {
			return nil, false
		}
		pkg.markDirty(ownerFile)
		return &field.Type, true
	case *ast.StarExpr:
		return resolveDeclTypeSlot(pkg, body, ftype, e.X)
	case *ast.ParenExpr:
		return resolveDeclTypeSlot(pkg, body, ftype, e.X)
	}
	return nil, false
}

// findStructFieldSlot returns the first struct field named name across
// every file in pkg, and the file it was found in.
func findStructFieldSlot(pkg *pkgScope, name string) (*ast.Field, *ast.File) {
	var found *ast.Field
	var foundFile *ast.File
	for _, f := range pkg.files {
		if found != nil {
			break
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if found != nil {
				return false
			}
			st, ok := n.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			for _, field := range st.Fields.List {
				for _, fn := range field.Names {
					if fn.Name == name {
						found = field
						foundFile = f
						return false
					}
				}
			}
			return true
		})
	}
	return found, foundFile
}

// findVarSpec finds a `var <name> T` declaration for name anywhere in
// body.
func findVarSpec(body *ast.BlockStmt, name string) *ast.ValueSpec {
	var found *ast.ValueSpec
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			return true
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range vs.Names {
				if n.Name == name {
					found = vs
					return false
				}
			}
		}
		return true
	})
	return found
}

// findAssignRHS finds the AssignStmt declaring or last assigning name
// before beforePos in body (a short `:=`/`=` decl with matching arity
// between Lhs and Rhs), returning it and name's index in Lhs.
func findAssignRHS(body *ast.BlockStmt, name string, beforePos token.Pos) (*ast.AssignStmt, int) {
	var found *ast.AssignStmt
	var idx int
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || as.Pos() >= beforePos || len(as.Lhs) != len(as.Rhs) {
			return true
		}
		for i, lhs := range as.Lhs {
			id, ok := lhs.(*ast.Ident)
			if ok && id.Name == name {
				found, idx = as, i
			}
		}
		return true
	})
	if found == nil {
		return nil, 0
	}
	return found, idx
}

// findTypeNodeInExpr locates the type-bearing node inside *slot for a map
// or struct literal initializer — `map[string]V{...}`, `make(map[string]V,
// ...)`, or `&map[string]V{...}` — returning a pointer to its Type field so
// retypeTypeExpr can mutate it in place.
func findTypeNodeInExpr(slot *ast.Expr) (*ast.Expr, bool) {
	switch v := (*slot).(type) {
	case *ast.CompositeLit:
		if v.Type == nil {
			return nil, false
		}
		return &v.Type, true
	case *ast.CallExpr:
		fn, ok := v.Fun.(*ast.Ident)
		if !ok || fn.Name != "make" || len(v.Args) == 0 {
			return nil, false
		}
		return &v.Args[0], true
	case *ast.UnaryExpr:
		if v.Op != token.AND {
			return nil, false
		}
		return findTypeNodeInExpr(&v.X)
	}
	return nil, false
}

// enclosingFunc returns the innermost FuncDecl/FuncLit body and signature
// containing target within file.
func enclosingFunc(file *ast.File, target ast.Node) (*ast.BlockStmt, *ast.FuncType) {
	type frame struct {
		body *ast.BlockStmt
		typ  *ast.FuncType
	}
	var stack []frame
	var body *ast.BlockStmt
	var ftype *ast.FuncType

	astutil.Apply(file, func(c *astutil.Cursor) bool {
		switch fn := c.Node().(type) {
		case *ast.FuncDecl:
			stack = append(stack, frame{fn.Body, fn.Type})
		case *ast.FuncLit:
			stack = append(stack, frame{fn.Body, fn.Type})
		}
		if c.Node() == target && len(stack) > 0 {
			top := stack[len(stack)-1]
			body, ftype = top.body, top.typ
		}
		return true
	}, func(c *astutil.Cursor) bool {
		switch c.Node().(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			stack = stack[:len(stack)-1]
		}
		return true
	})
	return body, ftype
}

// findCallArgAt finds the CallExpr in file with an argument starting at
// exactly (line, col), returning it and the argument's index.
func findCallArgAt(fset *token.FileSet, file *ast.File, line, col int) (*ast.CallExpr, int) {
	var found *ast.CallExpr
	var idx int
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		for i, a := range call.Args {
			p := fset.Position(a.Pos())
			if p.Line == line && p.Column == col {
				found, idx = call, i
			}
		}
		return true
	})
	return found, idx
}

// findIndexExprAt finds the IndexExpr in file whose index key expression
// starts at exactly (line, col).
func findIndexExprAt(fset *token.FileSet, file *ast.File, line, col int) *ast.IndexExpr {
	var found *ast.IndexExpr
	ast.Inspect(file, func(n ast.Node) bool {
		ix, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}
		p := fset.Position(ix.Index.Pos())
		if p.Line == line && p.Column == col {
			found = ix
		}
		return true
	})
	return found
}

// findCompositeElementAt finds the composite-literal element (keyed or
// positional) whose value expression starts at exactly (line, col),
// returning the key's identifier name (empty for a positional element) and
// a settable slot for the value.
func findCompositeElementAt(fset *token.FileSet, file *ast.File, line, col int) (string, *ast.Expr) {
	var keyName string
	var slot *ast.Expr
	ast.Inspect(file, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for i := range cl.Elts {
			if kv, ok := cl.Elts[i].(*ast.KeyValueExpr); ok {
				p := fset.Position(kv.Value.Pos())
				if p.Line == line && p.Column == col {
					if id, ok := kv.Key.(*ast.Ident); ok {
						keyName = id.Name
					}
					slot = &kv.Value
				}
				continue
			}
			p := fset.Position(cl.Elts[i].Pos())
			if p.Line == line && p.Column == col {
				keyName = ""
				slot = &cl.Elts[i]
			}
		}
		return true
	})
	return keyName, slot
}

// findReturnResultAt finds the ReturnStmt result expression starting at
// exactly (line, col), returning a settable slot for it and the ReturnStmt
// itself (for enclosing-function/result-index resolution).
func findReturnResultAt(fset *token.FileSet, file *ast.File, line, col int) (*ast.Expr, *ast.ReturnStmt) {
	var slot *ast.Expr
	var stmt *ast.ReturnStmt
	ast.Inspect(file, func(n ast.Node) bool {
		rs, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for i := range rs.Results {
			p := fset.Position(rs.Results[i].Pos())
			if p.Line == line && p.Column == col {
				slot, stmt = &rs.Results[i], rs
			}
		}
		return true
	})
	return slot, stmt
}

// returnResultIndex reports slot's index within stmt.Results.
func returnResultIndex(stmt *ast.ReturnStmt, slot *ast.Expr) int {
	for i := range stmt.Results {
		if &stmt.Results[i] == slot {
			return i
		}
	}
	return 0
}

// findVarDeclValueAt finds the initializer expression starting at exactly
// (line, col) inside a `var x T = <expr>` ValueSpec or a `x := <expr>`
// short declaration, returning a settable slot for it.
func findVarDeclValueAt(fset *token.FileSet, file *ast.File, line, col int) *ast.Expr {
	var slot *ast.Expr
	match := func(exprs []ast.Expr) {
		for i := range exprs {
			p := fset.Position(exprs[i].Pos())
			if p.Line == line && p.Column == col {
				slot = &exprs[i]
			}
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.ValueSpec:
			match(v.Values)
		case *ast.AssignStmt:
			match(v.Rhs)
		}
		return true
	})
	return slot
}

// findBinaryExprAt finds the BinaryExpr in file starting at exactly (line,
// col).
func findBinaryExprAt(fset *token.FileSet, file *ast.File, line, col int) *ast.BinaryExpr {
	var found *ast.BinaryExpr
	ast.Inspect(file, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		p := fset.Position(be.Pos())
		if p.Line == line && p.Column == col {
			found = be
		}
		return true
	})
	return found
}
