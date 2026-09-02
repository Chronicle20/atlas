package topicmod

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunCheckReportsUnmigratedSite pins the -check contract: a file whose
// only site is a rule match that Rewrite would apply (no residue) must
// still surface as a Finding under check mode, so a caller that treats any
// non-empty Finding list as a failure exits non-zero rather than silently
// reporting nothing. It also asserts the file is left untouched.
func TestRunCheckReportsUnmigratedSite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kafka.go")
	const src = `package character

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated = "CREATED"
)
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	findings, err := Run([]string{dir}, true)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("Run(check=true) reported no findings for an un-migrated topic-shaped file; want at least one")
	}

	found := false
	for _, f := range findings {
		if f.Pos.Filename == path {
			found = true
		}
	}
	if !found {
		t.Fatalf("Run(check=true) findings = %+v, want one naming %s", findings, path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != src {
		t.Fatalf("Run(check=true) modified the file; want it left untouched\ngot:\n%s", got)
	}
}

// TestRunCheckNoFindingsWhenNothingToMigrate confirms Run(check=true)
// reports nothing for a file with no topic-shaped sites at all.
func TestRunCheckNoFindingsWhenNothingToMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.go")
	const src = `package character

const StatusEventTypeCreated = "CREATED"
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	findings, err := Run([]string{dir}, true)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("Run(check=true) findings = %+v, want none", findings)
	}
}
