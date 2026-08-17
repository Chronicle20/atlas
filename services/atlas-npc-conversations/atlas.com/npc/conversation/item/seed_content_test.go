package item

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// seedCatalogRoot resolves deploy/seed from this package's location: item/ is
// six levels below the repo root.
func seedCatalogRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "..", "deploy", "seed")
}

// TestSeedCatalog_ItemConversationsBuild walks every version's authored item
// conversations and drives them through the exact path the seeder uses —
// Decode then Build. A file that decodes but fails Build (e.g. a sendOk with
// one choice instead of two) is not a seeding error the operator ever sees:
// runSubdomain records it in seed_state and the group still logs
// "Seed complete" with a zero created count, so the dialogue is simply absent
// at runtime and the saga fails with NO_CONVERSATION_AUTHORED. This test is
// where that class of defect is caught instead.
func TestSeedCatalog_ItemConversationsBuild(t *testing.T) {
	root := seedCatalogRoot(t)
	sd := ItemConversationSubdomain{}

	matches, err := filepath.Glob(filepath.Join(root, "*", "*", sd.Path(), "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no item conversation seed files found under %s", root)
	}

	for _, path := range matches {
		rel, _ := filepath.Rel(root, path)
		t.Run(rel, func(t *testing.T) {
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			rm, err := sd.Decode(b)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if _, err := sd.Build(tenant.Model{}, "", rm); err != nil {
				t.Fatalf("build: %v", err)
			}
		})
	}
}
