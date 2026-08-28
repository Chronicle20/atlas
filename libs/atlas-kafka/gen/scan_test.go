package main

import (
	"sort"
	"strings"
	"testing"
)

// TestScan runs Scan against the real repository root and asserts
// properties of the result rather than a frozen token list, so it does not
// become a second manifest to maintain.
func TestScan(t *testing.T) {
	repoRoot, err := repoRootFromGit()
	if err != nil {
		t.Fatalf("resolving repo root failed: %v", err)
	}

	m, err := Scan(repoRoot)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(m.Topics) <= 100 {
		t.Fatalf("len(m.Topics) = %d, want > 100 (a partial load collapses this)", len(m.Topics))
	}

	if !sort.SliceIsSorted(m.Topics, func(i, j int) bool { return m.Topics[i].Token < m.Topics[j].Token }) {
		t.Error("m.Topics is not sorted by Token")
	}

	wantCompact := map[string]bool{
		"EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS": true,
		"EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS":     true,
		"EVENT_TOPIC_CONFIGURATION_TENANT_STATUS":      true,
	}
	gotCompact := make(map[string]bool)

	byToken := make(map[string]Entry, len(m.Topics))
	for _, e := range m.Topics {
		byToken[e.Token] = e

		switch e.Cleanup {
		case "delete":
			// ok
		case "compact":
			gotCompact[e.Token] = true
		default:
			t.Errorf("token %s: Cleanup = %q, want \"delete\" or \"compact\"", e.Token, e.Cleanup)
		}

		if len(e.Packages) == 0 {
			t.Errorf("token %s: Packages is empty", e.Token)
		}
		if !sort.StringsAreSorted(e.Packages) {
			t.Errorf("token %s: Packages is not sorted: %v", e.Token, e.Packages)
		}
	}

	if len(gotCompact) != len(wantCompact) {
		t.Errorf("compact tokens = %v, want exactly %v", gotCompact, wantCompact)
	}
	for tok := range wantCompact {
		if !gotCompact[tok] {
			t.Errorf("token %s: want Cleanup = \"compact\", got %v", tok, byToken[tok])
		}
	}

	for _, tok := range []string{"STATUS_TOPIC_CASH_ITEM", "STATUS_EVENT_TOPIC_SKILL_MACRO"} {
		if _, ok := byToken[tok]; !ok {
			t.Errorf("expected token %s to be present (FR-6.1)", tok)
		}
	}

	// FR-1.7: tokens declared only in _test.go files must not appear.
	for _, tok := range []string{"EVENT_TOPIC_TEST", "EVENT_TOPIC_FAKE", "ANY_TOPIC", "MY_TOPIC", "RACE_TOPIC", "TEST_TOPIC"} {
		if _, ok := byToken[tok]; ok {
			t.Errorf("token %s from a _test.go file must not appear in the manifest", tok)
		}
	}
}

// TestStalePolicyIsAnError asserts that applyPolicy rejects a policy
// naming a token the scan did not find, and that the error names it.
func TestStalePolicyIsAnError(t *testing.T) {
	tokens := map[string][]string{
		"EVENT_TOPIC_REAL": {"atlas-example/kafka/message"},
	}
	pol := policy{Compact: []string{"EVENT_TOPIC_DOES_NOT_EXIST"}}

	_, err := applyPolicy(tokens, pol)
	if err == nil {
		t.Fatal("applyPolicy returned nil error for a stale policy token, want an error")
	}
	if !strings.Contains(err.Error(), "EVENT_TOPIC_DOES_NOT_EXIST") {
		t.Errorf("applyPolicy error = %q, want it to name EVENT_TOPIC_DOES_NOT_EXIST", err.Error())
	}
}

// TestDrift mirrors libs/atlas-constants/gen/drift_test.go's
// TestGeneratedFilesMatchSource: it re-scans the repository, re-emits
// topics.yaml in-process, and diffs it byte-for-byte against what's
// checked in via the same checkDrift helper `go run . -check` uses.
func TestDrift(t *testing.T) {
	repoRoot, err := repoRootFromGit()
	if err != nil {
		t.Fatalf("resolving repo root failed: %v", err)
	}

	m, err := Scan(repoRoot)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	want := m.EmitTopicsYAML()
	if err := checkDrift("topics.yaml", want); err != nil {
		t.Error(err)
	}
}
