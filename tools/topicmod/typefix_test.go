package topicmod

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseDiagnostics(t *testing.T) {
	const out = `# github.com/example/pkg [github.com/example/pkg.test]
character/meso_award_test.go:40:37: cannot use dropmsg.EnvCommandTopic (constant "COMMAND_TOPIC_DROP" of string type topic.Token) as string value in argument to capture.Messages
kafka/consumer/character/consumer_test.go:92:17: invalid operation: r.Topic != trademsg.EnvEventTopicStatus (mismatched types string and topic.Token)
FAIL	github.com/example/pkg [build failed]
`
	got := ParseDiagnostics(out)
	if len(got) != 2 {
		t.Fatalf("ParseDiagnostics() = %+v, want 2 diagnostics", got)
	}
	if got[0].Path != "character/meso_award_test.go" || got[0].Line != 40 || got[0].Col != 37 {
		t.Fatalf("ParseDiagnostics()[0] = %+v, want path/line/col character/meso_award_test.go:40:37", got[0])
	}
	if got[1].Message != `invalid operation: r.Topic != trademsg.EnvEventTopicStatus (mismatched types string and topic.Token)` {
		t.Fatalf("ParseDiagnostics()[1].Message = %q", got[1].Message)
	}
}

// TestFixModule pins the end-to-end diagnostic-driven pass: a same-file
// test-only mock declares its callee parameter as `string`, the test calls
// it with an already-migrated topic.Token constant, and go vet reports
// exactly the "cannot use ... as string value in argument to F" shape R1-R4
// leave behind. FixModule must retype the callee's parameter so the module
// builds and vets clean, without hand-editing the call site.
func TestFixModule(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	dir := t.TempDir()
	// The module path matches the real repo's so the R1-added
	// `github.com/Chronicle20/atlas/libs/atlas-kafka/topic` import resolves
	// within this temp module, against a minimal stand-in `topic` package
	// declared below, instead of needing network/module-cache access to the
	// real one.
	const goMod = "module github.com/Chronicle20/atlas\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	topicDir := filepath.Join(dir, "libs", "atlas-kafka", "topic")
	if err := os.MkdirAll(topicDir, 0o755); err != nil {
		t.Fatalf("mkdir topic stand-in: %v", err)
	}
	const topicSrc = "package topic\n\ntype Token string\n"
	if err := os.WriteFile(filepath.Join(topicDir, "topic.go"), []byte(topicSrc), 0o644); err != nil {
		t.Fatalf("write topic stand-in: %v", err)
	}

	const src = `package fixtest

const EnvCommandTopic = "COMMAND_TOPIC_DROP"

type mock struct{ seen []string }

func (m *mock) Messages(token string) []string {
	return m.seen
}
`
	const testSrc = `package fixtest

import "testing"

func TestMessages(t *testing.T) {
	m := &mock{}
	m.seen = append(m.seen, "x")
	if got := m.Messages(EnvCommandTopic); len(got) == 0 {
		t.Fatal("expected messages")
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "fixtest.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write fixtest.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixtest_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatalf("write fixtest_test.go: %v", err)
	}

	// R1 first: retype the topic-token constant, as it would already be by
	// the time -fix-tests runs against a real package.
	if _, err := Run([]string{dir}, false); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	residue, err := FixModule(dir)
	if err != nil {
		t.Fatalf("FixModule() error = %v", err)
	}
	if len(residue) != 0 {
		t.Fatalf("FixModule() residue = %+v, want none", residue)
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet after FixModule() failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(filepath.Join(dir, "fixtest.go"))
	if err != nil {
		t.Fatalf("read fixtest.go: %v", err)
	}
	if !contains(string(got), "token topic.Token") {
		t.Fatalf("fixtest.go was not retyped to topic.Token:\n%s", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
