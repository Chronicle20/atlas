package seeder

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// tenantGMS83 creates a GMS 83.1 tenant for tests.
func tenantGMS83(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "gms", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// goodFixtureRoot returns the absolute path to testdata/good for use in tests.
func goodFixtureRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata", "good")
}

// newTestSource returns a FilesystemCatalogSource whose fallback root points at
// the testdata/good fixture tree embedded in the repo.
func newTestSource(t *testing.T) CatalogSource {
	t.Helper()
	// testdata lives next to catalog_test.go
	return NewFilesystemCatalogSource("ATLAS_SEEDER_CATALOG_TEST_OVERRIDE", "testdata/good")
}

func TestFilesystemCatalogSource_Roots_UsesTenantRegionVersion(t *testing.T) {
	src := newTestSource(t)
	tm := tenantGMS83(t)

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
	want := filepath.Join("gms", "83_1")
	if !strings.HasSuffix(roots[0], want) {
		t.Fatalf("root = %q, want suffix %q", roots[0], want)
	}
}

// Production tenants store the region in uppercase ("GMS") while the
// catalog tree on disk uses lowercase directory names. Roots() must
// normalize to the lowercase layout so the path resolves on a
// case-sensitive filesystem (Linux).
func TestFilesystemCatalogSource_Roots_LowerCasesRegion(t *testing.T) {
	src := newTestSource(t)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
	want := filepath.Join("gms", "83_1")
	if !strings.HasSuffix(roots[0], want) {
		t.Fatalf("root = %q, want suffix %q (region must be lower-cased)", roots[0], want)
	}
}

func TestFilesystemCatalogSource_Revision(t *testing.T) {
	src := newTestSource(t)
	tm := tenantGMS83(t)

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}

	rev, err := src.Revision(roots[0])
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	if rev != "test-rev-abc123" {
		t.Fatalf("revision = %q, want %q", rev, "test-rev-abc123")
	}
}

func TestFilesystemCatalogSource_Revision_MissingReturnsEmptyNoError(t *testing.T) {
	src := newTestSource(t)
	// Use a non-existent path — Revision should return ("", nil).
	rev, err := src.Revision(t.TempDir())
	if err != nil {
		t.Fatalf("Revision on missing file: %v", err)
	}
	if rev != "" {
		t.Fatalf("revision = %q, want empty", rev)
	}
}

func TestFilesystemCatalogSource_Walk_SkipsUnderscorePrefixed(t *testing.T) {
	src := newTestSource(t)
	tm := tenantGMS83(t)

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}

	files, err := src.Walk(roots[0], "widgets")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := []string{"widget-1.json", "widget-2.json"}
	if len(files) != len(want) {
		t.Fatalf("Walk returned %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("files[%d] = %q, want %q", i, f, want[i])
		}
	}
}

func TestFilesystemCatalogSource_Walk_SkipsUnderscoreDir(t *testing.T) {
	src := newTestSource(t)
	tm := tenantGMS83(t)

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}

	files, err := src.Walk(roots[0], "gizmos")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := []string{"gizmo-100.json"}
	if len(files) != len(want) {
		t.Fatalf("Walk returned %v, want %v", files, want)
	}
	if files[0] != want[0] {
		t.Errorf("files[0] = %q, want %q", files[0], want[0])
	}
}

func TestFilesystemCatalogSource_Open(t *testing.T) {
	src := newTestSource(t)
	tm := tenantGMS83(t)

	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}

	rc, err := src.Open(roots[0], filepath.Join("widgets", "widget-1.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(b), `"name":"one"`) {
		t.Fatalf("file content = %q, want to contain %q", string(b), `"name":"one"`)
	}
}

func TestFilesystemCatalogSource_EnvOverridesFallback(t *testing.T) {
	src := NewFilesystemCatalogSource("ATLAS_SEEDER_CATALOG_TEST_OVERRIDE", "/nonexistent/fallback")

	// Point env var at the real testdata directory.
	abs, err := filepath.Abs("testdata/good")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	t.Setenv("ATLAS_SEEDER_CATALOG_TEST_OVERRIDE", abs)

	tm := tenantGMS83(t)
	roots, err := src.Roots(tm)
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
	want := filepath.Join(abs, "gms", "83_1")
	if roots[0] != want {
		t.Fatalf("root = %q, want %q", roots[0], want)
	}
}

func TestFilesystemCatalogSource_Roots_ZeroVersion(t *testing.T) {
	// tenant.Create does not validate zero versions, so we construct one here
	// to verify that Roots() itself rejects zero major/minor version.
	tm, err := tenant.Create(uuid.New(), "gms", 0, 0)
	if err != nil {
		// If Create starts validating, skip.
		t.Skipf("tenant.Create rejected zero version: %v", err)
	}

	src := newTestSource(t)
	_, rootsErr := src.Roots(tm)
	if rootsErr == nil {
		t.Fatal("Roots(zero-version tenant) returned nil error, want error")
	}
}

// newSharedTestSource returns a CatalogSource with both a shared root and
// the tenant's version-specific root, over the testdata/good fixture tree.
func newSharedTestSource(t *testing.T) CatalogSource {
	t.Helper()
	return NewFilesystemCatalogSourceWithShared("ATLAS_SEEDER_CATALOG_TEST_OVERRIDE", "testdata/good", "shared/all")
}

func TestFilesystemCatalogSource_Roots_SharedFirstThenVersion(t *testing.T) {
	roots, err := newSharedTestSource(t).Roots(tenantGMS83(t))
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("len(roots) = %d, want 2", len(roots))
	}
	if !strings.HasSuffix(roots[0], filepath.Join("shared", "all")) {
		t.Errorf("roots[0] = %q, want shared root first", roots[0])
	}
	if !strings.HasSuffix(roots[1], filepath.Join("gms", "83_1")) {
		t.Errorf("roots[1] = %q, want version-specific root second", roots[1])
	}
}

// The nine existing consumers construct the plain source; it must keep
// returning exactly one root so their composite revision never changes.
func TestFilesystemCatalogSource_Roots_PlainSourceStillSingleRoot(t *testing.T) {
	roots, err := newTestSource(t).Roots(tenantGMS83(t))
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("len(roots) = %d, want 1", len(roots))
	}
}

func TestRevisionFor_SingleRootIsUnchanged(t *testing.T) {
	src := newTestSource(t)
	roots, _ := src.Roots(tenantGMS83(t))
	if got := revisionFor(src, roots); got != "test-rev-abc123" {
		t.Fatalf("revisionFor = %q, want %q", got, "test-rev-abc123")
	}
}

func TestRevisionFor_TwoRootsJoinsWithPlus(t *testing.T) {
	src := newSharedTestSource(t)
	roots, _ := src.Roots(tenantGMS83(t))
	want := "shared-rev-xyz789+test-rev-abc123"
	if got := revisionFor(src, roots); got != want {
		t.Fatalf("revisionFor = %q, want %q", got, want)
	}
}

func TestRevisionFor_SkipsEmptyRevisions(t *testing.T) {
	src := newSharedTestSource(t)
	// t.TempDir() has no CATALOG_REVISION → Revision returns ("", nil).
	roots, _ := src.Roots(tenantGMS83(t))
	got := revisionFor(src, []string{t.TempDir(), roots[1]})
	if got != "test-rev-abc123" {
		t.Fatalf("revisionFor = %q, want %q (empty root contributes nothing)", got, "test-rev-abc123")
	}
}
