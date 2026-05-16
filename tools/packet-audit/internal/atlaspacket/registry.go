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
type TypeRegistry struct {
	types map[string]*TypeEntry
}

// TypeEntry holds the pre-analyzed wire shape for one struct type.
type TypeEntry struct {
	File       string
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
	reg := &TypeRegistry{types: map[string]*TypeEntry{}}

	type fileCtx struct {
		path string
		file *ast.File
		fset *token.FileSet
	}

	// Pass 1: collect struct declarations.
	var files []fileCtx
	err := filepath.WalkDir(atlasPacketRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // ignore broken files
		}
		files = append(files, fileCtx{path: path, file: f, fset: fset})
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
				reg.types[ts.Name.Name] = &TypeEntry{
					File:       path,
					StructDecl: st,
				}
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
			entry, ok := reg.types[recvType]
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
				entry.Calls = collectCallsWithCtx(body, fc.fset, reg, recvType)
			case "EncodeForeign":
				// Register under the "<Type>::EncodeForeign" key so callers can pick it
				// explicitly without colliding with the primary Encode entry.
				body := findReturnClosure(fd.Body)
				if body == nil {
					body = fd.Body
				}
				altKey := recvType + "::EncodeForeign"
				reg.types[altKey] = &TypeEntry{
					File:       entry.File,
					StructDecl: entry.StructDecl,
					Calls:      collectCallsWithCtx(body, fc.fset, reg, recvType),
				}
			case "Write":
				// Write methods have a flat body (no closure return) and accept *response.Writer.
				// Only register if no Encode method was already found.
				if entry.Calls == nil {
					entry.Calls = collectCallsWithCtx(fd.Body, fc.fset, reg, recvType)
				}
			}
		}
	}
	return reg, nil
}

// Calls returns the pre-analyzed Call list for the named type, if registered.
func (r *TypeRegistry) Calls(typeName string) ([]Call, bool) {
	e, ok := r.types[typeName]
	if !ok {
		return nil, false
	}
	return e.Calls, e.Calls != nil
}

// FieldType returns the type name of the named field on a struct type, with
// slice/array/pointer/package indirection stripped. Returns ("", false) if unknown.
func (r *TypeRegistry) FieldType(typeName, fieldName string) (string, bool) {
	e, ok := r.types[typeName]
	if !ok {
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
func (r *TypeRegistry) HasType(name string) bool {
	_, ok := r.types[name]
	return ok
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
