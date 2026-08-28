package topicmod

import (
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Run walks dirs, applying Rewrite to every .go file it finds — including
// _test.go files, since test callers of a retyped API need the same R1-R4
// rewrites production code does — skipping vendor/ and testdata/ paths,
// formats the result with go/format, and writes it back unless check is
// true. It returns every residue Finding collected across all files, plus —
// when check is true — a "check" Finding for every file Rewrite would
// change, so a caller can treat any un-migrated site (residue or
// would-change) as a non-zero exit rather than silently reporting nothing.
func Run(dirs []string, check bool) ([]Finding, error) {
	var findings []Finding

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "vendor" || d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}

			changed, residue := Rewrite(fset, f, path)
			findings = append(findings, residue...)
			if !changed {
				return nil
			}
			if check {
				findings = append(findings, Finding{
					Pos:    fset.Position(f.Package),
					Rule:   "check",
					Reason: "file requires rewriting (run without -check to apply)",
				})
				return nil
			}

			var buf strings.Builder
			if err := format.Node(&buf, fset, f); err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.WriteFile(path, []byte(buf.String()), info.Mode().Perm())
		})
		if err != nil {
			return findings, err
		}
	}

	return findings, nil
}
