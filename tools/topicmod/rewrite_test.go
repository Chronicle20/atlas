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
	cases := []string{"r1_decl", "r2_buffer", "r3_propagate", "r3_propagate_decl", "r3_propagate_multi", "r4_newconfig", "r4_newconfig_delegate"}

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

// TestRewriteR3AssignResidueWithoutHoistTarget pins the negative case for
// the `=`-form hoist heuristic: an EnvProvider discard assigning into a
// variable that is not preceded, in the same block, by a `var <name> string`
// declaration for that exact variable (here, a named return value). There is
// nowhere safe to hoist `var err error`, so R3 must report residue instead
// of emitting `t, err = ...` with `err` never declared.
func TestRewriteR3AssignResidueWithoutHoistTarget(t *testing.T) {
	const src = `package character

func InitHandlers(l logrus.FieldLogger) (t string, err error) {
	t, _ = topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
	return t, nil
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "handlers.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	changed, residue := Rewrite(fset, f, "handlers.go")
	if changed {
		t.Fatalf("Rewrite() reported a change; want none since `t` has no preceding `var t string` to hoist `var err error` after")
	}
	if len(residue) != 1 {
		t.Fatalf("Rewrite() residue = %+v, want exactly one finding", residue)
	}
	const wantReason = "assignment-form EnvProvider discard has no preceding `var <name> string` declaration in the same block to hoist `var err error` after"
	if residue[0].Rule != "R3" || residue[0].Reason != wantReason {
		t.Fatalf("Rewrite() residue = %+v, want {Rule: R3, Reason: %s}", residue[0], wantReason)
	}
}

// TestRewriteR4Residue pins R4's silent-skip fix: a NewConfig declaration
// whose signature matches the curried chain but whose body is neither the
// direct EnvProvider assignment nor a thin delegate to another NewConfig
// must be reported as residue rather than left untouched with no findings.
func TestRewriteR4Residue(t *testing.T) {
	const src = `package consumer

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(brokers(), name, token, groupId)
			}
		}
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "consumer.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	changed, residue := Rewrite(fset, f, "consumer.go")
	if changed {
		t.Fatalf("Rewrite() reported a change; want none since the body matches neither R4 shape")
	}
	if len(residue) != 1 {
		t.Fatalf("Rewrite() residue = %+v, want exactly one finding", residue)
	}
	const wantReason = "NewConfig signature matches curried chain but body is neither the direct EnvProvider assignment nor a thin delegate to another NewConfig"
	if residue[0].Rule != "R4" || residue[0].Reason != wantReason {
		t.Fatalf("Rewrite() residue = %+v, want {Rule: R4, Reason: %s}", residue[0], wantReason)
	}
}
