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
	cases := []string{"r1_decl", "r2_buffer", "r3_propagate", "r3_propagate_decl", "r4_newconfig"}

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

// TestRewriteR3Residue pins the one documented sweep residue (design §11):
// an EnvProvider error discarded inside a method that does not return
// error. R3 must classify it as residue and leave it untouched rather than
// mangle it.
func TestRewriteR3Residue(t *testing.T) {
	const src = `package frederick

func (t *NotificationTask) Run() {
	_, _ = topic.EnvProvider(t.l)(merchant.EnvStatusEventTopic)()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "run.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	changed, residue := Rewrite(fset, f, "run.go")
	if changed {
		t.Fatalf("Rewrite() reported a change; want none since Run() does not return error")
	}
	if len(residue) != 1 {
		t.Fatalf("Rewrite() residue = %+v, want exactly one finding", residue)
	}
	if residue[0].Rule != "R3" || residue[0].Reason != "enclosing function does not return error" {
		t.Fatalf("Rewrite() residue = %+v, want {Rule: R3, Reason: enclosing function does not return error}", residue[0])
	}
}
