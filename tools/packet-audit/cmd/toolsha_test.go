package cmd

import "testing"

// TestHashGoTreeEntriesStableAcrossNonGoChange proves the ToolSHA is derived
// only from the tool's .go sources: editing a non-.go file (README/docs) must
// not move the SHA, while editing a .go source must. The computation is also
// order-stable and deterministic. (task-169 T2.0)
func TestHashGoTreeEntriesStableAcrossNonGoChange(t *testing.T) {
	base := "100644 blob aaa\ttools/packet-audit/cmd/matrix.go\n" +
		"100644 blob bbb\ttools/packet-audit/README.md\n"

	// A non-.go change (README blob differs) must NOT move the SHA.
	readmeChanged := "100644 blob aaa\ttools/packet-audit/cmd/matrix.go\n" +
		"100644 blob zzz\ttools/packet-audit/README.md\n"
	if hashGoTreeEntries(base) != hashGoTreeEntries(readmeChanged) {
		t.Fatal("ToolSHA changed on a non-.go (README) edit — churn trap not closed")
	}

	// A .go source change MUST move the SHA.
	goChanged := "100644 blob ccc\ttools/packet-audit/cmd/matrix.go\n" +
		"100644 blob bbb\ttools/packet-audit/README.md\n"
	if hashGoTreeEntries(base) == hashGoTreeEntries(goChanged) {
		t.Fatal("ToolSHA did not change on a .go source edit")
	}

	// Order-stable: reordering ls-tree lines yields the same SHA.
	reordered := "100644 blob bbb\ttools/packet-audit/README.md\n" +
		"100644 blob aaa\ttools/packet-audit/cmd/matrix.go\n"
	if hashGoTreeEntries(base) != hashGoTreeEntries(reordered) {
		t.Fatal("ToolSHA not order-stable")
	}

	// testdata .go fixtures are excluded.
	withTestdata := base + "100644 blob ddd\ttools/packet-audit/cmd/testdata/x.go\n"
	if hashGoTreeEntries(base) != hashGoTreeEntries(withTestdata) {
		t.Fatal("testdata .go fixture should be excluded from ToolSHA")
	}

	// Deterministic across calls.
	if hashGoTreeEntries(base) != hashGoTreeEntries(base) {
		t.Fatal("ToolSHA computation is not deterministic")
	}
}
