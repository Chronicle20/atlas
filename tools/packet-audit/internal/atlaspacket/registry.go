package atlaspacket

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// TypeRegistry catalogs struct types in libs/atlas-packet/ that have an
// Encode or Write method, and pre-analyzes their bodies so the diff engine
// can inline KindRecurse markers.
//
// Storage is keyed on a qualified name "<pkgPath>.<structName>" where
// pkgPath is the directory of the struct relative to the atlas-packet root
// (e.g. "monster/clientbound.Spawn"). This prevents the previous
// last-write-wins collision across sub-domains that reuse short struct
// names like Spawn, Destroy, Movement.
//
// A short-name index (byShort) maps unqualified names back to their
// qualified registry keys for callers that don't know the package context.
type TypeRegistry struct {
	types   map[string]*TypeEntry // qualified key
	byShort map[string][]string   // short name → qualified keys
}

// TypeEntry holds the pre-analyzed wire shape for one struct type.
type TypeEntry struct {
	File       string
	PkgPath    string // directory of the struct relative to atlas-packet root, e.g. "monster/clientbound"
	StructDecl *ast.StructType
	// Pre-analyzed calls for whichever method this type exposes for wire encoding.
	// Encode preferred over Write when both exist (atlas convention).
	Calls []Call
	// Opaque is set by Pass-3 when a type has NO encode method AND its struct
	// layout could not be decomposed into known primitives / registered types
	// (e.g. a field of an unmappable type like time.Time, an interface, or an
	// unregistered named type). Such a type is the register boundary: it is the
	// curation target for the docs opaque-type registry, and the diff engine
	// emits a STABLE deferred row for it rather than silently passing it through.
	Opaque bool
}

// NewTypeRegistry walks atlasPacketRoot, discovers struct types with Encode/Write
// methods, and pre-analyzes their bodies into a Call list. Two passes:
//  1. Catalog all struct declarations across all files.
//  2. For each Encode/Write method, locate its containing struct, parse its body,
//     and analyze it (including resolving its own KindRecurse markers via this
//     registry).
func NewTypeRegistry(atlasPacketRoot string) (*TypeRegistry, error) {
	reg := &TypeRegistry{
		types:   map[string]*TypeEntry{},
		byShort: map[string][]string{},
	}

	type fileCtx struct {
		path    string
		pkgPath string
		file    *ast.File
		fset    *token.FileSet
	}

	// Pass 1: collect struct declarations.
	var files []fileCtx
	err := filepath.WalkDir(atlasPacketRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(atlasPacketRoot, filepath.Dir(path))
		if relErr != nil {
			rel = ""
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // ignore broken files
		}
		files = append(files, fileCtx{path: path, pkgPath: rel, file: f, fset: fset})
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				qual := qualify(rel, ts.Name.Name)
				reg.types[qual] = &TypeEntry{
					File:       path,
					PkgPath:    rel,
					StructDecl: st,
				}
				reg.byShort[ts.Name.Name] = append(reg.byShort[ts.Name.Name], qual)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Pass 2: for each file, find Encode/Write methods and analyze their bodies.
	// Encode wins over Write.
	for _, fc := range files {
		for _, decl := range fc.file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			recvType := receiverIdent(fd.Recv.List[0].Type)
			if recvType == "" {
				continue
			}
			qual := qualify(fc.pkgPath, recvType)
			entry, ok := reg.types[qual]
			if !ok {
				continue
			}
			// Skip if we already have an Encode entry (Encode wins over Write).
			// EncodeForeign always proceeds since it registers under its own alt-key.
			if entry.Calls != nil && fd.Name.Name != "Encode" && fd.Name.Name != "EncodeForeign" {
				continue
			}
			switch fd.Name.Name {
			case "Encode":
				body := findReturnClosure(fd.Body)
				if body == nil {
					body = fd.Body
				}
				entry.Calls = collectCallsWithCtx(body, fc.fset, reg, qual)
			case "EncodeEntry":
				// EncodeEntry returns a closure (same shape as Encode) but the method
				// name differs because the type is a list-entry sub-struct (e.g.
				// inventory/change_entry.go's AddEntry/MoveEntry). Encode wins over
				// EncodeEntry per the precedence below.
				if entry.Calls == nil {
					body := findReturnClosure(fd.Body)
					if body == nil {
						body = fd.Body
					}
					entry.Calls = collectCallsWithCtx(body, fc.fset, reg, qual)
				}
			case "EncodeBytes":
				// EncodeBytes returns a flat []byte (no closure). Used for sub-structs
				// embedded inside a top-level Encode via WriteByteArray (e.g.
				// cash/clientbound/shop_inventory.go's CashInventoryItem).
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, qual)
				}
			case "EncodeForeign":
				// Register under the "<Type>::EncodeForeign" key so callers can pick it
				// explicitly without colliding with the primary Encode entry.
				body := findReturnClosure(fd.Body)
				if body == nil {
					body = fd.Body
				}
				altQual := qual + "::EncodeForeign"
				altShort := recvType + "::EncodeForeign"
				reg.types[altQual] = &TypeEntry{
					File:       entry.File,
					PkgPath:    fc.pkgPath,
					StructDecl: entry.StructDecl,
					Calls:      collectCallsWithCtx(body, fc.fset, reg, altQual),
				}
				reg.byShort[altShort] = append(reg.byShort[altShort], altQual)
			case "Write":
				// Write methods have a flat body (no closure return) and accept *response.Writer.
				// Only register if no Encode method was already found.
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, qual)
				}
			}
		}
	}

	// Pass 3: fallback descent for self-describing sub-structs (task-080 §4.7).
	//
	// A struct that has NO encode method (Calls still nil after Pass-2) but whose
	// parent encoder writes its fields inline — e.g. model.Asset / GW_ItemSlotBase
	// style flat layouts, npc ShopCommodity — leaves a KindRecurse marker the diff
	// engine cannot inline, surfacing as a 🔍 deferred false-positive.
	//
	// When EVERY field of such a type decomposes into a known wire primitive or an
	// already-registered (and itself-resolved) type, synthesize Calls by walking
	// the fields in declaration order. This is deliberately conservative: if ANY
	// field is unmappable (time.Time, interface, map, embedded, channel, or a
	// named type that is not registered with non-nil Calls), the WHOLE type is
	// left Calls==nil and flagged Opaque — the explicit register boundary — rather
	// than guessing a partial/wrong layout that could mask a real mismatch.
	//
	// Synthesis runs to a fixed point so a decomposable type that depends on
	// another decomposable type (both method-less) can resolve once its dependency
	// has been synthesized. Types still unresolved after the fixed point are
	// flagged Opaque.
	for {
		progressed := false
		for _, e := range reg.types {
			if e.Calls != nil || e.Opaque || e.StructDecl == nil {
				continue
			}
			calls, ok := reg.synthesizeFieldCalls(e)
			if ok {
				e.Calls = calls
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}
	// Any remaining method-less, unsynthesized struct is the opaque boundary.
	for _, e := range reg.types {
		if e.Calls == nil && e.StructDecl != nil {
			e.Opaque = true
		}
	}

	return reg, nil
}

// synthesizeFieldCalls attempts to derive a wire-shape Call list for a struct
// that has no encode method, by decomposing its fields in declaration order.
// Returns (calls, true) only when EVERY field resolves to a known primitive or
// an already-resolved registered type; otherwise (nil, false) — the caller
// leaves the entry unresolved (eventually flagged Opaque). Embedded fields,
// blank-named fields, and any field whose type is neither a mappable Go builtin
// nor a registered type with non-nil Calls cause an immediate bail-out.
func (r *TypeRegistry) synthesizeFieldCalls(e *TypeEntry) ([]Call, bool) {
	if e.StructDecl.Fields == nil {
		return nil, false
	}
	var out []Call
	for _, field := range e.StructDecl.Fields.List {
		// Embedded field (no names) — cannot reason about its inline layout safely.
		if len(field.Names) == 0 {
			return nil, false
		}
		for range field.Names {
			c, ok := r.fieldCall(field.Type, e.PkgPath)
			if !ok {
				return nil, false
			}
			out = append(out, c...)
		}
	}
	if len(out) == 0 {
		// A zero-field (or all-skipped) struct yields no wire shape; treat as
		// opaque rather than synthesizing an empty (and meaningless) inline.
		return nil, false
	}
	return out, true
}

// fieldCall maps a single struct-field type expression to the Call(s) that its
// inline encoding would emit. Returns (calls, true) for a known Go primitive or
// an already-resolved registered named type; (nil, false) for anything the
// analyzer cannot decompose safely. Slices/arrays/pointers are NOT descended:
// their wire shape depends on a length-prefix + loop the field type alone does
// not capture, so they are treated as unmappable (→ Opaque) rather than guessed.
func (r *TypeRegistry) fieldCall(t ast.Expr, contextPkg string) ([]Call, bool) {
	switch v := t.(type) {
	case *ast.Ident:
		if p, ok := goPrimitive(v.Name); ok {
			return []Call{{Kind: KindWrite, Op: p}}, true
		}
		// A named, in-package struct type — decompose only if already registered
		// with a concrete (non-nil) Call list. A KindRecurse keeps the diff engine
		// inlining via the registry, preserving cycle/guard handling.
		if r.resolveResolvedEntry(v.Name, contextPkg) != nil {
			return []Call{{Kind: KindRecurse, RecurseType: r.Qualify(v.Name, contextPkg)}}, true
		}
		return nil, false
	case *ast.SelectorExpr:
		// pkg.Type — e.g. another package's named type. Decompose only if its bare
		// name resolves to a registered, resolved entry.
		name := v.Sel.Name
		if r.resolveResolvedEntry(name, contextPkg) != nil {
			return []Call{{Kind: KindRecurse, RecurseType: r.Qualify(name, contextPkg)}}, true
		}
		return nil, false
	}
	// StarExpr (pointer), ArrayType (slice/array), MapType, InterfaceType,
	// ChanType, FuncType, StructType (anonymous) — all unmappable from the field
	// type alone. Bail out so the enclosing type is flagged Opaque.
	return nil, false
}

// resolveResolvedEntry returns a registered TypeEntry for name that already has
// a concrete (non-nil) Call list, preferring a same-package match. Returns nil
// when no such resolved entry exists (unknown, ambiguous, opaque, or not yet
// synthesized). Used by Pass-3 so synthesis only descends into types that are
// themselves already decomposed.
func (r *TypeRegistry) resolveResolvedEntry(name, contextPkg string) *TypeEntry {
	qual := r.Qualify(name, contextPkg)
	if e, ok := r.types[qual]; ok && e.Calls != nil {
		return e
	}
	if quals, ok := r.byShort[name]; ok && len(quals) == 1 {
		if e := r.types[quals[0]]; e != nil && e.Calls != nil {
			return e
		}
	}
	return nil
}

// goPrimitive maps a Go builtin numeric/bool/string type name to its wire
// Primitive. Only fixed-width builtins atlas writes 1:1 are mapped; anything
// else returns false so the field is treated as unmappable. Two exclusions are
// load-bearing and deliberate, not oversights:
//   - int / uint are excluded because their wire width is platform-ambiguous
//     (32- vs 64-bit), so the field type alone cannot pin a byte count.
//   - float32 / float64 are excluded because the Atlas response.Writer never
//     writes raw floats — float-valued fields are converted to fixed-width ints
//     before encoding — so a float field must force Opaque rather than let the
//     analyzer guess a width.
func goPrimitive(name string) (Primitive, bool) {
	switch name {
	case "bool", "byte", "int8", "uint8":
		return Encode1, true
	case "int16", "uint16":
		return Encode2, true
	case "int32", "uint32", "rune":
		return Encode4, true
	case "int64", "uint64":
		return Encode8, true
	case "string":
		return EncodeStr, true
	}
	return 0, false
}

// Calls returns the pre-analyzed Call list for the named type, if registered.
// Accepts either a qualified key ("<pkgPath>.<name>") or an unqualified short
// name. For short-name lookups, only returns calls when there is exactly one
// qualified match — ambiguous short names return (nil, false) so callers must
// disambiguate via Qualify or a context-aware resolver.
func (r *TypeRegistry) Calls(typeName string) ([]Call, bool) {
	if e, ok := r.types[typeName]; ok {
		return e.Calls, e.Calls != nil
	}
	if quals, ok := r.byShort[typeName]; ok && len(quals) == 1 {
		e := r.types[quals[0]]
		return e.Calls, e.Calls != nil
	}
	return nil, false
}

// IsOpaque reports whether the named type is a registered Pass-3 opaque
// boundary: it has no encode method and its layout could not be synthesized.
// Accepts a qualified key or an unambiguously-short name. Returns false for
// unknown or ambiguous names.
func (r *TypeRegistry) IsOpaque(typeName string) bool {
	if e, ok := r.types[typeName]; ok {
		return e.Opaque
	}
	if quals, ok := r.byShort[typeName]; ok && len(quals) == 1 {
		e := r.types[quals[0]]
		return e.Opaque
	}
	return false
}

// FieldType returns the type name of the named field on a struct type, with
// slice/array/pointer/package indirection stripped. Returns the bare Go
// identifier (no package qualifier) — callers needing the qualified form
// should pass the result through Qualify with the enclosing pkgPath.
// Accepts qualified or unambiguously-short typeName.
func (r *TypeRegistry) FieldType(typeName, fieldName string) (string, bool) {
	e := r.resolveEntry(typeName)
	if e == nil {
		return "", false
	}
	for _, field := range e.StructDecl.Fields.List {
		for _, fname := range field.Names {
			if fname.Name == fieldName {
				return resolveTypeName(field.Type), true
			}
		}
	}
	return "", false
}

// HasType reports whether the registry knows about the named type at all.
// Accepts qualified or unqualified — for unqualified, returns true if any
// package has a struct by that name.
func (r *TypeRegistry) HasType(name string) bool {
	if _, ok := r.types[name]; ok {
		return true
	}
	_, ok := r.byShort[name]
	return ok
}

// Qualify resolves a possibly-unqualified type hint to a qualified registry
// key, preferring same-package matches over global short-name matches.
// Returns the hint unchanged when no registered type matches.
//
// EncodeForeign suffixes ("::EncodeForeign") are preserved across the
// qualification.
func (r *TypeRegistry) Qualify(hint, contextPkg string) string {
	if hint == "" {
		return hint
	}
	if _, ok := r.types[hint]; ok {
		return hint
	}
	// Strip an EncodeForeign suffix for the lookup, re-attach at the end.
	suffix := ""
	name := hint
	if strings.HasSuffix(hint, "::EncodeForeign") {
		suffix = "::EncodeForeign"
		name = strings.TrimSuffix(hint, "::EncodeForeign")
	}
	// Try direct qualified lookup with the context package.
	if contextPkg != "" {
		if cand := qualify(contextPkg, name) + suffix; ifTypeExists(r, cand) {
			return cand
		}
	}
	// Fall back to short-name index.
	if quals, ok := r.byShort[name+suffix]; ok && len(quals) > 0 {
		if contextPkg != "" {
			// Prefer a same-pkg or sibling-pkg match.
			for _, q := range quals {
				if strings.HasPrefix(q, contextPkg+".") {
					return q
				}
			}
		}
		return quals[0]
	}
	return hint
}

func ifTypeExists(r *TypeRegistry, key string) bool {
	_, ok := r.types[key]
	return ok
}

// resolveEntry looks up an entry by qualified key, or by unambiguously-short
// name. Returns nil for ambiguous short names or unknown keys.
func (r *TypeRegistry) resolveEntry(name string) *TypeEntry {
	if e, ok := r.types[name]; ok {
		return e
	}
	if quals, ok := r.byShort[name]; ok && len(quals) == 1 {
		return r.types[quals[0]]
	}
	return nil
}

// qualify joins a pkgPath and struct name into the canonical registry key.
// Falls back to just the name when pkgPath is empty (root of atlas-packet).
func qualify(pkgPath, name string) string {
	if pkgPath == "" {
		return name
	}
	return pkgPath + "." + name
}

// receiverIdent extracts the type name from a receiver type expression.
func receiverIdent(t ast.Expr) string {
	switch rt := t.(type) {
	case *ast.Ident:
		return rt.Name
	case *ast.StarExpr:
		if id, ok := rt.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// resolveTypeName strips slice/array/pointer/package indirection from a type
// expression and returns the unqualified Go identifier.
func resolveTypeName(t ast.Expr) string {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return resolveTypeName(v.X)
	case *ast.ArrayType:
		return resolveTypeName(v.Elt)
	case *ast.SelectorExpr:
		// package.Type — return the unqualified type name
		return v.Sel.Name
	}
	return ""
}
