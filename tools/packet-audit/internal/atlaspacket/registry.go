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
	return reg, nil
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
