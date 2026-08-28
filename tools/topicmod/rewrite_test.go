package topicmod

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestRewrite(t *testing.T) {
	cases := []string{"r1_decl", "r2_buffer"}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", name)
			beforePath := filepath.Join(dir, "before.go.txt")
			afterPath := filepath.Join(dir, "after.go.txt")

			want, err := os.ReadFile(afterPath)
			if err != nil {
				t.Fatalf("read %s: %v", afterPath, err)
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, beforePath, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", beforePath, err)
			}

			changed, residue := Rewrite(fset, f, beforePath)
			if !changed {
				t.Fatalf("Rewrite() reported no change for %s", beforePath)
			}
			if len(residue) != 0 {
				t.Fatalf("Rewrite() reported unexpected residue for %s: %+v", beforePath, residue)
			}

			var buf bytes.Buffer
			if err := format.Node(&buf, fset, f); err != nil {
				t.Fatalf("format.Node: %v", err)
			}

			if !bytes.Equal(buf.Bytes(), want) {
				t.Fatalf("rewritten %s does not match %s\n--- got ---\n%s\n--- want ---\n%s", beforePath, afterPath, buf.Bytes(), want)
			}
		})
	}
}
