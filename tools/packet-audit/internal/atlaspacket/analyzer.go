package atlaspacket

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
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
	// Opaque marks a KindRecurse whose target type is a registered but
	// non-decomposable boundary (task-080 §4.7 Pass-3): it has no encode method
	// and its struct layout could not be synthesized. The flatten step sets this
	// so the diff engine emits a STABLE deferred row keyed on "opaque" — the
	// curation target for the docs opaque-type registry — rather than a generic
	// unresolved-recurse row.
	Opaque bool
}

// AnalyzeFile parses a single .go (or .go.txt) file and extracts the ordered
// sequence of w.Write*/r.Read* calls inside the named method's outer return closure.
// Optional registry enables sub-struct type resolution; pass nil to keep the
// legacy variable-name behavior (used by existing tests).
func AnalyzeFile(path, typeName, methodName string) ([]Call, error) {
	return AnalyzeFileWithRegistry(path, typeName, methodName, nil)
}

// AnalyzeFileWithRegistry is like AnalyzeFile but accepts a TypeRegistry for
// sub-struct type resolution. Pass nil for the legacy no-resolution path.
func AnalyzeFileWithRegistry(path, typeName, methodName string, reg *TypeRegistry) ([]Call, error) {
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
	enclosing := qualifiedEnclosingFromPath(path, typeName, reg)
	return collectCallsWithCtx(inner, fset, reg, enclosing), nil
}

// qualifiedEnclosingFromPath derives the qualified registry key for a struct
// at the given file path. When the registry is non-nil and the path lives
// under "libs/atlas-packet/<sub>/<direction>/file.go", the result is
// "<sub>/<direction>.<typeName>". Falls back to the bare type name when the
// path doesn't conform or the registry is nil.
func qualifiedEnclosingFromPath(path, typeName string, reg *TypeRegistry) string {
	if reg == nil {
		return typeName
	}
	norm := filepath.ToSlash(path)
	const marker = "libs/atlas-packet/"
	idx := strings.Index(norm, marker)
	if idx < 0 {
		return typeName
	}
	relDir := strings.TrimSuffix(norm[idx+len(marker):], filepath.Base(norm))
	relDir = strings.TrimSuffix(relDir, "/")
	if relDir == "" {
		return typeName
	}
	return relDir + "." + typeName
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

// callCtx holds the context needed for registry-aware call collection.
type callCtx struct {
	reg *TypeRegistry
	// enclosing is the (qualified, when known) registry key of the struct
	// whose Encode/Write body is being walked. Used for FieldType lookups.
	enclosing string
	// pkg is the pkgPath portion of enclosing (e.g. "monster/clientbound"),
	// used as the same-package preference when qualifying short-name recurse
	// targets via reg.Qualify. Empty when enclosing is itself unqualified.
	pkg string
	// rangeVars maps a range-loop variable name to the field name it iterates over.
	// E.g. for "for _, c := range m.characters" → rangeVars["c"] = "characters".
	rangeVars map[string]string
	// fieldVars maps a local variable name to a resolved Go type name.
	// Used when a field is bound to a variable before calling .Write on it.
	fieldVars         map[string]string
	out               *[]Call
	stack             *[]*GuardExpr
	suffixGuards      []*GuardExpr // implicit guards from preceding if-returns at this scope
	unreachableSuffix bool         // true when both branches of a preceding if returned
	fset              *token.FileSet
}

// resolveRecurse attempts to resolve a variable-name hint to an actual Go type
// name using range-var bindings and field-type lookups on the enclosing struct.
// Falls back to returning the hint unchanged.
//
// When the registry is qualified (post task-065 item 4), the returned value
// is always passed through reg.Qualify so RecurseType carries the package-
// qualified key that the diff engine can look up unambiguously.
func resolveRecurse(hint string, cc *callCtx) string {
	if cc == nil || cc.reg == nil || cc.enclosing == "" {
		return hint
	}
	resolved := hint
	switch {
	case cc.reg.HasType(hint):
		// Hint already names a registered type — keep it.
	default:
		if fieldName, ok := cc.rangeVars[hint]; ok {
			if r, ok := cc.reg.FieldType(cc.enclosing, fieldName); ok && r != "" {
				resolved = r
				break
			}
		}
		if r, ok := cc.reg.FieldType(cc.enclosing, hint); ok && r != "" {
			resolved = r
		}
	}
	return cc.reg.Qualify(resolved, cc.pkg)
}

// collectCallsWithCtx walks a block and collects all w.Write*/r.Read* primitive
// calls in order, with optional registry-aware sub-struct type resolution.
// Pass reg=nil and enclosing="" to get the legacy variable-name behavior.
//
// `enclosing` should be the qualified registry key (e.g. "monster/clientbound.Spawn")
// when the caller has it; unqualified short names still work for backward
// compatibility but lose same-package preference during sub-struct resolution.
func collectCallsWithCtx(b *ast.BlockStmt, fset *token.FileSet, reg *TypeRegistry, enclosing string) []Call {
	var out []Call
	var stack []*GuardExpr
	pkg := ""
	if i := strings.LastIndex(enclosing, "."); i > 0 && reg != nil {
		// Strip the EncodeForeign suffix before treating ".X" as the type segment.
		head := enclosing
		if strings.HasSuffix(head, "::EncodeForeign") {
			head = strings.TrimSuffix(head, "::EncodeForeign")
		}
		if j := strings.LastIndex(head, "."); j > 0 {
			pkg = head[:j]
		}
	}
	cc := &callCtx{
		reg:       reg,
		enclosing: enclosing,
		pkg:       pkg,
		rangeVars: map[string]string{},
		fieldVars: map[string]string{},
		out:       &out,
		stack:     &stack,
		fset:      fset,
	}
	cc.walk(b)
	return out
}

// collectCalls is the legacy no-context wrapper — preserves the existing API.
func collectCalls(b *ast.BlockStmt, fset *token.FileSet) []Call {
	return collectCallsWithCtx(b, fset, nil, "")
}

func (cc *callCtx) appendCall(c Call) {
	*cc.out = append(*cc.out, c)
}

// pushSuffixGuard appends g to this scope's suffix-guard accumulator, used to
// taint sibling calls after an if-block whose body or else terminates with return.
// Nil guards are dropped — they would AND-out into the existing stack as no-ops.
func (cc *callCtx) pushSuffixGuard(g *GuardExpr) {
	if g == nil {
		return
	}
	cc.suffixGuards = append(cc.suffixGuards, g)
}

// conjoin returns the active combined guard for the current call site, AND-ing the
// explicit if-stack with any suffix guards accumulated from preceding if-returns
// at this scope. Delegates to the package-level conjoin once the slices are merged.
func (cc *callCtx) conjoin() *GuardExpr {
	// Combine explicit stack and any accumulated suffix guards.
	if len(cc.suffixGuards) == 0 {
		return conjoin(*cc.stack)
	}
	combined := append([]*GuardExpr{}, *cc.stack...)
	combined = append(combined, cc.suffixGuards...)
	return conjoin(combined)
}

// isIfWireMutex reports whether both branches of an if statement produce the
// same wire shape — same Call.Kind+Op (and RecurseType) at every position. It
// peeks at the branches via a scratch ctx without emitting into the parent's
// call list. Returns false when there is no else, when one branch is empty
// while the other isn't, or when the shapes diverge in any position.
//
// Examples that ARE mutex (collapsed to a single position):
//
//   if isMeso { w.WriteInt(meso)   } else { w.WriteInt(itemId)  }
//   if owned  { w.WriteByte(1)     } else { w.WriteByte(5)      }
//   if isSkill { w.WriteInt(1)     } else { w.WriteInt(0)       }
//
// Examples that are NOT mutex (still emit per-branch with guards):
//
//   if extended { w.WriteInt(x); w.WriteByte(y) } else { w.WriteInt(x) }   // different lengths
//   if narrow   { w.WriteByte(x) } else { w.WriteInt(x) }                  // different widths
//   if x.Encode { w.WriteByte(x) }                                         // no else
func (cc *callCtx) isIfWireMutex(n *ast.IfStmt) bool {
	if n.Else == nil {
		return false
	}
	bodyCalls := cc.scratchWalk(n.Body)
	var elseCalls []Call
	switch e := n.Else.(type) {
	case *ast.BlockStmt:
		elseCalls = cc.scratchWalk(e)
	case *ast.IfStmt:
		elseCalls = cc.scratchWalk(&ast.BlockStmt{List: []ast.Stmt{e}})
	default:
		return false
	}
	if len(bodyCalls) == 0 || len(bodyCalls) != len(elseCalls) {
		return false
	}
	for i, b := range bodyCalls {
		e := elseCalls[i]
		if b.Kind != e.Kind || b.Op != e.Op || b.RecurseType != e.RecurseType {
			return false
		}
	}
	return true
}

// scratchWalk runs the call collector against a sub-tree without emitting into
// the parent ctx. Used by isIfWireMutex to peek at branch shapes before
// deciding whether to collapse or expand them.
//
// Important: the scratch ctx is given a fresh stack/out/suffix-state so it
// observes only the calls *inside* the sub-tree. It shares the parent's
// registry, range/field-var maps, and enclosing context so resolveRecurse
// continues to work — those are read-only during a wire-shape peek.
func (cc *callCtx) scratchWalk(b ast.Node) []Call {
	var out []Call
	var stack []*GuardExpr
	scratch := &callCtx{
		reg:       cc.reg,
		enclosing: cc.enclosing,
		pkg:       cc.pkg,
		rangeVars: cc.rangeVars,
		fieldVars: cc.fieldVars,
		out:       &out,
		stack:     &stack,
		fset:      cc.fset,
	}
	scratch.walk(b)
	return out
}

// blockTerminatesWithReturn reports whether b's final statement is an *ast.ReturnStmt,
// either at top level or as the terminator of every branch of a terminal IfStmt.
// Loops are not descended (design §3.3 — loop-internal early-return is out of scope).
func blockTerminatesWithReturn(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	last := b.List[len(b.List)-1]
	switch s := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.IfStmt:
		if s.Else == nil {
			return false
		}
		elseBlock, ok := s.Else.(*ast.BlockStmt)
		if !ok {
			// else if — descend into the inner IfStmt's body and walk its else recursively.
			innerIf, ok := s.Else.(*ast.IfStmt)
			if !ok {
				return false
			}
			wrapped := &ast.BlockStmt{List: []ast.Stmt{innerIf}}
			return blockTerminatesWithReturn(s.Body) && blockTerminatesWithReturn(wrapped)
		}
		return blockTerminatesWithReturn(s.Body) && blockTerminatesWithReturn(elseBlock)
	}
	return false
}

func (cc *callCtx) walk(node ast.Node) {
	switch n := node.(type) {
	case *ast.IfStmt:
		// Wire-mutex collapse (task-065 item 5):
		// Detect `if x { WriteByte(a) } else { WriteByte(b) }` patterns where
		// both branches emit the same wire shape (same Kind+Op+RecurseType at
		// every position). Treat them as a single unconditional position rather
		// than two consecutive entries that misalign the diff. Mutually-
		// exclusive branches that write divergent wire shapes fall through to
		// the standard per-branch walk with guards intact.
		if cc.isIfWireMutex(n) {
			cc.walk(n.Body)
			return
		}
		g := guardFromIf(n, cc.fset)
		*cc.stack = append(*cc.stack, g)
		cc.walk(n.Body)
		*cc.stack = (*cc.stack)[:len(*cc.stack)-1]
		thenReturns := blockTerminatesWithReturn(n.Body)
		elseReturns := false
		if n.Else != nil {
			ng := negate(g)
			*cc.stack = append(*cc.stack, ng)
			cc.walk(n.Else)
			*cc.stack = (*cc.stack)[:len(*cc.stack)-1]
			switch e := n.Else.(type) {
			case *ast.BlockStmt:
				elseReturns = blockTerminatesWithReturn(e)
			case *ast.IfStmt:
				// else if — wrap and check.
				elseReturns = blockTerminatesWithReturn(&ast.BlockStmt{List: []ast.Stmt{e}})
			}
		}
		// Suffix-taint: when one branch returns, push an implicit guard for the surviving branch
		// onto cc.suffixGuards so any sibling calls after this if-block inherit it.
		switch {
		case thenReturns && elseReturns:
			// Both branches return — unreachable suffix. Mark and skip.
			cc.unreachableSuffix = true // unreachableSuffix is read by the enclosing *ast.BlockStmt loop to skip dead code following a both-branch-return if. See the BlockStmt arm.
		case thenReturns:
			cc.pushSuffixGuard(negate(g))
		case elseReturns && n.Else != nil:
			cc.pushSuffixGuard(g)
		}
	case *ast.BlockStmt:
		// Each block scope owns its own suffix-guard accumulator.
		savedSuffix := cc.suffixGuards
		savedUnreachable := cc.unreachableSuffix
		cc.suffixGuards = nil
		cc.unreachableSuffix = false
		for _, s := range n.List {
			if cc.unreachableSuffix {
				// Dead code after both branches returned; skip remaining statements in this block.
				break
			}
			cc.walk(s)
		}
		cc.suffixGuards = savedSuffix
		cc.unreachableSuffix = savedUnreachable
	case *ast.ExprStmt:
		cc.walk(n.X)
	case *ast.AssignStmt:
		// atlas Decode methods write to receiver fields:
		//   m.field = r.ReadByte()
		// The wire op lives on the RHS as a CallExpr. Walk each RHS so we
		// pick up the primitive — required for task-065 item 7
		// (Encode↔Decode equivalence) where serverbound packets' runtime
		// path is Decode rather than Encode.
		for _, rhs := range n.Rhs {
			cc.walk(rhs)
		}
	case *ast.RangeStmt:
		// Record range variable binding for type resolution.
		// Pattern: for _, varName := range m.<fieldName> { ... }
		rangeVarName := ""
		if n.Value != nil {
			if id, ok := n.Value.(*ast.Ident); ok && id.Name != "_" {
				rangeVarName = id.Name
			}
		}
		if rangeVarName != "" && cc.reg != nil {
			fieldName := rangeFieldName(n.X)
			if fieldName != "" {
				cc.rangeVars[rangeVarName] = fieldName
			}
		}
		sub := cc.collectSub(n.Body)
		cc.appendCall(Call{
			Kind:  KindRepeat,
			Body:  sub,
			Line:  cc.fset.Position(n.Pos()).Line,
			Guard: cc.conjoin(),
		})
		if rangeVarName != "" {
			delete(cc.rangeVars, rangeVarName)
		}
	case *ast.ForStmt:
		sub := cc.collectSub(n.Body)
		cc.appendCall(Call{
			Kind:  KindRepeat,
			Body:  sub,
			Line:  cc.fset.Position(n.Pos()).Line,
			Guard: cc.conjoin(),
		})
	case *ast.CallExpr:
		sel, ok := n.Fun.(*ast.SelectorExpr)
		if !ok {
			// Handle free-function helpers like WritePaddedString(w, name, n) that
			// atlas uses for fixed-length string fields. Treat as EncodeBuf since
			// the IDA side typically models them as DecodeBuffer(buf, n).
			if id, ok := n.Fun.(*ast.Ident); ok {
				if p, ok := freeFnPrimitive(id.Name); ok {
					cc.appendCall(Call{
						Kind:  KindWrite,
						Op:    p,
						Line:  cc.fset.Position(n.Pos()).Line,
						Guard: cc.conjoin(),
					})
					return
				}
			}
			// Handle chained calls like m.sub.Encode(l, ctx)(opts):
			// the outer CallExpr has Fun = inner CallExpr; recurse into Fun.
			if inner, ok := n.Fun.(*ast.CallExpr); ok {
				cc.walk(inner)
			}
			return
		}
		// Recurse marker: x.Encode(l, ctx) or x.Decode(l, ctx)
		// Exclude common false-positives (writer/reader/logger receivers).
		if sel.Sel.Name == "Encode" || sel.Sel.Name == "Decode" {
			recv := receiverTypeHint(sel.X)
			if !isWriterReaderReceiver(recv) {
				resolved := resolveRecurse(recv, cc)
				cc.appendCall(Call{
					Kind:        KindRecurse,
					RecurseType: resolved,
					Line:        cc.fset.Position(n.Pos()).Line,
					Guard:       cc.conjoin(),
				})
				return
			}
		}
		// Recurse marker: x.EncodeForeign(l, ctx) — routes to the alternate "<Type>::EncodeForeign" key.
		if sel.Sel.Name == "EncodeForeign" {
			recv := receiverTypeHint(sel.X)
			if !isWriterReaderReceiver(recv) {
				resolved := resolveRecurse(recv, cc)
				// Annotate the recurse with the EncodeForeign variant so the diff
				// engine resolves via the alternate registry key.
				cc.appendCall(Call{
					Kind:        KindRecurse,
					RecurseType: resolved + "::EncodeForeign",
					Line:        cc.fset.Position(n.Pos()).Line,
					Guard:       cc.conjoin(),
				})
				return
			}
		}
		// Detect x.Write(w) pattern: a one-arg Write call whose receiver is not a
		// writer/reader variable. This covers atlas's WorldRecommendation pattern
		// where sub-structs expose Write(*response.Writer) instead of Encode().
		if sel.Sel.Name == "Write" && len(n.Args) == 1 {
			recv := receiverTypeHint(sel.X)
			if !isWriterReaderReceiver(recv) {
				resolved := resolveRecurse(recv, cc)
				cc.appendCall(Call{
					Kind:        KindRecurse,
					RecurseType: resolved,
					Line:        cc.fset.Position(n.Pos()).Line,
					Guard:       cc.conjoin(),
				})
				return
			}
		}
		// Compound writer method: WriteKeyValue(byte, int32) is atlas's helper for
		// equipment-slot maps. Decompose into two primitive writes so the diff
		// engine aligns row-by-row against the IDA loop body's Decode1 + Decode4.
		if sel.Sel.Name == "WriteKeyValue" {
			line := cc.fset.Position(n.Pos()).Line
			g := cc.conjoin()
			cc.appendCall(Call{Kind: KindWrite, Op: Encode1, Line: line, Guard: g})
			cc.appendCall(Call{Kind: KindWrite, Op: Encode4, Line: line, Guard: g})
			return
		}
		// Wrapped recurse marker: WriteByteArray(c.Encode(l, ctx)(opts)) — atlas's
		// CharacterList encoder pre-encodes a sub-struct to []byte then writes the
		// buffer verbatim. The wire shape is identical to inlining c.Encode's calls,
		// so model this as KindRecurse rather than KindWrite EncodeBuf.
		if p, ok := primFromName(sel.Sel.Name); ok {
			// Check for WriteByteArray(<sub>.EncodeForeign(...)(...)) first, since
			// wrappedRecurseType only detects Encode/Decode. Use the alt-key.
			if recv, ok := wrappedEncodeForeignRecurseType(sel.Sel.Name, n.Args); ok {
				resolved := resolveRecurse(recv, cc)
				cc.appendCall(Call{
					Kind:        KindRecurse,
					RecurseType: resolved + "::EncodeForeign",
					Line:        cc.fset.Position(n.Pos()).Line,
					Guard:       cc.conjoin(),
				})
				return
			}
			if recv, ok := wrappedRecurseType(sel.Sel.Name, n.Args); ok {
				resolved := resolveRecurse(recv, cc)
				cc.appendCall(Call{
					Kind:        KindRecurse,
					RecurseType: resolved,
					Line:        cc.fset.Position(n.Pos()).Line,
					Guard:       cc.conjoin(),
				})
				return
			}
			cc.appendCall(Call{
				Kind:  KindWrite,
				Op:    p,
				Line:  cc.fset.Position(n.Pos()).Line,
				Guard: cc.conjoin(),
			})
		}
	default:
		ast.Inspect(node, func(c ast.Node) bool {
			if c == node {
				return true
			}
			if _, ok := c.(*ast.IfStmt); ok {
				cc.walk(c)
				return false
			}
			if _, ok := c.(*ast.RangeStmt); ok {
				cc.walk(c)
				return false
			}
			if _, ok := c.(*ast.ForStmt); ok {
				cc.walk(c)
				return false
			}
			if ce, ok := c.(*ast.CallExpr); ok {
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
					// Recurse marker check
					if sel.Sel.Name == "Encode" || sel.Sel.Name == "Decode" {
						recv := receiverTypeHint(sel.X)
						if !isWriterReaderReceiver(recv) {
							resolved := resolveRecurse(recv, cc)
							cc.appendCall(Call{
								Kind:        KindRecurse,
								RecurseType: resolved,
								Line:        cc.fset.Position(ce.Pos()).Line,
								Guard:       cc.conjoin(),
							})
							return false
						}
					}
					// Recurse marker: x.EncodeForeign(l, ctx) in nested contexts.
					if sel.Sel.Name == "EncodeForeign" {
						recv := receiverTypeHint(sel.X)
						if !isWriterReaderReceiver(recv) {
							resolved := resolveRecurse(recv, cc)
							cc.appendCall(Call{
								Kind:        KindRecurse,
								RecurseType: resolved + "::EncodeForeign",
								Line:        cc.fset.Position(ce.Pos()).Line,
								Guard:       cc.conjoin(),
							})
							return false
						}
					}
					// Detect x.Write(w) in nested contexts too.
					if sel.Sel.Name == "Write" && len(ce.Args) == 1 {
						recv := receiverTypeHint(sel.X)
						if !isWriterReaderReceiver(recv) {
							resolved := resolveRecurse(recv, cc)
							cc.appendCall(Call{
								Kind:        KindRecurse,
								RecurseType: resolved,
								Line:        cc.fset.Position(ce.Pos()).Line,
								Guard:       cc.conjoin(),
							})
							return false
						}
					}
					if p, ok := primFromName(sel.Sel.Name); ok {
						cc.appendCall(Call{
							Kind:  KindWrite,
							Op:    p,
							Line:  cc.fset.Position(ce.Pos()).Line,
							Guard: cc.conjoin(),
						})
					}
				}
			}
			return true
		})
	}
}

// collectSub collects body calls for a loop body, inheriting the registry
// context and current range-var bindings from the parent callCtx.
func (cc *callCtx) collectSub(b *ast.BlockStmt) []Call {
	var out []Call
	var stack []*GuardExpr
	// Copy rangeVars so the child inherits current bindings without polluting parent.
	childRangeVars := make(map[string]string, len(cc.rangeVars))
	for k, v := range cc.rangeVars {
		childRangeVars[k] = v
	}
	child := &callCtx{
		reg:       cc.reg,
		enclosing: cc.enclosing,
		rangeVars: childRangeVars,
		fieldVars: map[string]string{},
		out:       &out,
		stack:     &stack,
		fset:      cc.fset,
	}
	child.walk(b)
	return out
}

// collectGuardedSub collects body calls fresh — the parent Repeat already carries the outer guard.
// Kept for any direct callers outside this file.
func collectGuardedSub(b *ast.BlockStmt, fset *token.FileSet) []Call {
	return collectCalls(b, fset)
}

// rangeFieldName extracts the field name from a range expression like m.fieldName
// or just fieldName. Returns "" if the expression doesn't match either pattern.
func rangeFieldName(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.SelectorExpr:
		// m.fieldName → return fieldName
		return v.Sel.Name
	case *ast.Ident:
		// bare variable — could be a field alias; return as-is
		return v.Name
	}
	return ""
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

// freeFnPrimitive returns a Primitive for known free-function helpers that
// atlas uses outside the w.Write*/r.Read* method convention.
//   WritePaddedString(w, str, n) writes a fixed-length buffer.
//   ReadPaddedString(r, n) reads a fixed-length buffer.
func freeFnPrimitive(name string) (Primitive, bool) {
	switch name {
	case "WritePaddedString", "ReadPaddedString":
		return EncodeBuf, true
	}
	return 0, false
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

// wrappedRecurseType detects the WriteByteArray(<sub>.Encode(...)(...)) pattern
// used by atlas's CharacterList encoder. Returns the sub-receiver type hint and
// true when the outer primitive is WriteByteArray and its sole arg is a call
// chain whose terminal selector is Encode or Decode.
func wrappedRecurseType(outerMethod string, args []ast.Expr) (string, bool) {
	if outerMethod != "WriteByteArray" && outerMethod != "ReadByteArray" {
		return "", false
	}
	if len(args) != 1 {
		return "", false
	}
	// Unwrap: args[0] might be c.Encode(l, ctx)(opts) — a CallExpr whose Fun is
	// itself a CallExpr whose Fun is a SelectorExpr ending in Encode/Decode.
	expr := args[0]
	for {
		ce, ok := expr.(*ast.CallExpr)
		if !ok {
			return "", false
		}
		if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Encode" || sel.Sel.Name == "Decode" {
				recv := receiverTypeHint(sel.X)
				if !isWriterReaderReceiver(recv) {
					return recv, true
				}
			}
			return "", false
		}
		expr = ce.Fun
	}
}

// wrappedEncodeForeignRecurseType detects WriteByteArray(<sub>.EncodeForeign(...)(...))
// pattern — the foreign-receiver variant of wrappedRecurseType. Returns the
// sub-receiver type hint (raw, without the "::EncodeForeign" suffix) and true
// when the inner selector is EncodeForeign. The caller appends the suffix.
func wrappedEncodeForeignRecurseType(outerMethod string, args []ast.Expr) (string, bool) {
	if outerMethod != "WriteByteArray" && outerMethod != "ReadByteArray" {
		return "", false
	}
	if len(args) != 1 {
		return "", false
	}
	expr := args[0]
	for {
		ce, ok := expr.(*ast.CallExpr)
		if !ok {
			return "", false
		}
		if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "EncodeForeign" {
				recv := receiverTypeHint(sel.X)
				if !isWriterReaderReceiver(recv) {
					return recv, true
				}
			}
			return "", false
		}
		expr = ce.Fun
	}
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
//
// If the stack contains an unparseable guard (e.g. <unparsed:...> for
// expressions like `len(x) > 0` that the DSL parser can't model), the
// text-based reparse fails. In that case we synthesize a GuardExpr whose
// Eval AND-s the stack's eval functions directly — preserves the
// outer-guard semantics that a "return last guard" fallback would lose.
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
	if combined, err := ParseGuard(strings.Join(parts, " && ")); err == nil {
		return combined
	}
	snapshot := make([]*GuardExpr, len(s))
	copy(snapshot, s)
	return &GuardExpr{
		eval: func(c GuardContext) bool {
			for _, g := range snapshot {
				if !g.eval(c) {
					return false
				}
			}
			return true
		},
		text: strings.Join(parts, " && "),
	}
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
	case "WriteByte", "WriteBool", "WriteInt8", "ReadByte", "ReadBool", "ReadInt8":
		return Encode1, true
	case "WriteShort", "WriteInt16", "ReadUint16", "ReadInt16":
		return Encode2, true
	case "WriteInt", "WriteInt32", "ReadUint32", "ReadInt32":
		return Encode4, true
	case "WriteLong", "WriteInt64", "ReadUint64", "ReadInt64":
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
