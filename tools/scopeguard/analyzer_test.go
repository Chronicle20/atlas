// Internal test package (not scopeguard_test): TestAnalyzerAllowlisted needs
// to substitute a fixture allowlist for EntityAllowlist/CallsiteAllowlist
// without touching the real checked-in allowlist.txt/callsite-allowlist.txt.
package scopeguard

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer exercises both rules' failing and passing shapes against the
// real checked-in allowlist.txt/callsite-allowlist.txt (both effectively
// empty for these fixture packages, since none of their keys appear there).
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer,
		"atlas-example/widget",       // Rule 1: data-plane, no TenantId — fails
		"atlas-example/scoped",       // Rule 1: data-plane, has TenantId — passes
		"atlas-example/lowerentity",  // Rule 1: lowercase `entity` name, no TenantId — fails (fix round 2)
		"atlas-trades/itementity",    // Rule 1: `Entity`-suffixed name (ItemEntity), has TenantId — passes (fix round 2)
		"atlas-configurations/thing", // Rule 2: control-plane, no Environment, no allowlist entry, no unique natural key — fails
		"atlas-tenants/registry",     // Rule 2: control-plane, has Environment — passes
		"atlas-tenants/config",       // has TenantId despite living in a control-plane service — passes (mirrors real configuration.Entity)
		"atlas-callsite/scheduler",   // Rule 2 call-site: the atlas-marriages shape — one violation, one clean call site, in the same package
	)
}

// TestAnalyzerAllowlisted substitutes a fixture allowlist so the guard's
// exemption path is exercised without polluting the real fleet allowlist
// files with test-only entries.
//
// atlas-configurations/auditrow is the fix-round-1 smuggle probe: it gets
// an allowlist entry here exactly like envfixture does, but its Entity has
// no uniquely-constrained natural key (see its own doc comment). Its
// testdata `// want` annotation asserts it is STILL flagged despite the
// entry — this is the test that would have caught the smuggle the reviewer
// found in the allowlist-only version of this check, and is what stops it
// regressing.
//
// atlas-configurations/testfileaudit is the fix-round-2 smuggle probe: the
// same audit-row shape, but declared in an entity_test.go file instead of
// entity.go, added after the real environments/processor_test.go's
// testEntity needed its own allowlist entry keyed off a _test.go file name.
// It proves entityAllowlistKey's file-based derivation (any declaring file
// name, not hardcoded to "entity.go") does not accidentally exempt
// everything in a _test.go file — hasUniqueNaturalKey still gates it.
func TestAnalyzerAllowlisted(t *testing.T) {
	origEntity, origCallsite := EntityAllowlist, CallsiteAllowlist
	EntityAllowlist = map[string]string{
		"atlas-allowedsvc/widget/entity.go":                 "test fixture — see analyzer_test.go",
		"atlas-configurations/envfixture/entity.go":         "test fixture — see analyzer_test.go (task-232 Task 19 control-plane allowlist exception)",
		"atlas-configurations/auditrow/entity.go":           "test fixture — see analyzer_test.go (task-232 Task 19 fix round 1 smuggle probe; must still be flagged)",
		"atlas-configurations/testfileaudit/entity_test.go": "test fixture — see analyzer_test.go (task-232 Task 19 fix round 2 smuggle probe; a _test.go declaration must still be flagged with no unique natural key)",
	}
	CallsiteAllowlist = map[string]string{
		"atlas-callsite-allowed/task/task.go:14": "test fixture — see analyzer_test.go",
	}
	defer func() {
		EntityAllowlist = origEntity
		CallsiteAllowlist = origCallsite
	}()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer,
		"atlas-allowedsvc/widget",
		"atlas-configurations/envfixture",
		"atlas-configurations/auditrow",
		"atlas-configurations/testfileaudit",
		"atlas-callsite-allowed/task",
	)
}

// TestParseAllowlistRequiresReason pins the allowlist file's own lint rule:
// an entry with a key but no reason is a parse error, never silently
// admitted. This is what TestAllowlistEntriesHaveReasons (checking the real
// files parsed cleanly at package init, implicitly, since init would have
// panicked otherwise) depends on actually being enforced.
func TestParseAllowlistRequiresReason(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"empty file", "", false},
		{"comment only", "# just a comment\n", false},
		{"valid entry", "some/path/entity.go # a real reason\n", false},
		{"no hash at all", "some/path/entity.go\n", true},
		{"hash but empty reason", "some/path/entity.go #\n", true},
		{"hash but whitespace reason", "some/path/entity.go #   \n", true},
		{"duplicate key", "a # r1\na # r2\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseAllowlist(c.raw)
			if (err != nil) != c.wantErr {
				t.Fatalf("parseAllowlist(%q) error = %v, wantErr %v", c.raw, err, c.wantErr)
			}
		})
	}
}

// TestAllowlistEntriesHaveReasons re-parses the real, checked-in allowlist
// files directly (rather than relying on package init having already
// succeeded) so a future malformed entry fails this test with a clear
// message instead of a cryptic init panic during `go test`.
func TestAllowlistEntriesHaveReasons(t *testing.T) {
	if _, err := parseAllowlist(allowlistRaw); err != nil {
		t.Fatalf("allowlist.txt: %v", err)
	}
	if _, err := parseAllowlist(callsiteAllowlistRaw); err != nil {
		t.Fatalf("callsite-allowlist.txt: %v", err)
	}
}
