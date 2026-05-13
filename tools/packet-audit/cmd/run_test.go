package cmd

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPhaseAExitGate(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	out := t.TempDir()

	args := []string{
		"--csv-clientbound", filepath.Join(repoRoot, "docs/packets/MapleStory Ops - ClientBound.csv"),
		"--csv-serverbound", filepath.Join(repoRoot, "docs/packets/MapleStory Ops - ServerBound.csv"),
		"--template", filepath.Join(repoRoot, "services/atlas-configurations/seed-data/templates/template_gms_95_1.json"),
		"--atlas-packet", filepath.Join(repoRoot, "libs/atlas-packet"),
		"--ida-source", filepath.Join(repoRoot, "docs/packets/ida-exports/gms_v95.json"),
		"--output", out,
	}
	var stderr bytes.Buffer
	rc := Run(args, &stderr)
	if rc == 3 {
		t.Fatalf("runtime error: rc=%d stderr=%q", rc, stderr.String())
	}
	for _, want := range []string{"AuthSuccess.md", "ServerListEntry.md", "ServerIP.md"} {
		matches, _ := filepath.Glob(filepath.Join(out, "*", want))
		if len(matches) == 0 {
			matches, _ = filepath.Glob(filepath.Join(out, want))
		}
		if len(matches) == 0 {
			t.Errorf("missing expected report: %s (out=%s)", want, out)
		}
	}
}
