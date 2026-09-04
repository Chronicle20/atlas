package _map

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const objTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="effect.img">
  <imgdir name="quest">
    <imgdir name="gate">
      <imgdir name="0"><int name="obstacle" value="1"/></imgdir>
      <imgdir name="1"></imgdir>
    </imgdir>
  </imgdir>
</imgdir>
`

func writeObjFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "effect.img.xml"), []byte(objTestXML), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
}

func TestInitObjIndexesObstacles(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	dir := t.TempDir()
	writeObjFixture(t, dir)

	indexed, err := InitObj(ten, dir)
	if err != nil {
		t.Fatalf("InitObj returned error: %v", err)
	}
	if indexed != 1 {
		t.Fatalf("expected 1 indexed obstacle, got %d", indexed)
	}

	def, err := GetMapObjectRegistry().Get(ten, "effect/quest/gate/0")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !def.Obstacle() {
		t.Fatalf("expected Obstacle() to be true")
	}

	if _, err := GetMapObjectRegistry().Get(ten, "effect/quest/gate/1"); err == nil {
		t.Fatalf("expected error for effect/quest/gate/1, got nil")
	}

	if k := ResolveObjKind(ten, "effect", "quest", "gate", "0"); k != ObjKindObstacle {
		t.Fatalf("expected %s, got %s", ObjKindObstacle, k)
	}
	if k := ResolveObjKind(ten, "effect", "quest", "gate", "1"); k != ObjKindEnvironment {
		t.Fatalf("expected %s, got %s", ObjKindEnvironment, k)
	}
	if k := ResolveObjKind(ten, "nothing", "at", "all", "0"); k != ObjKindEnvironment {
		t.Fatalf("expected %s, got %s", ObjKindEnvironment, k)
	}
}

func TestInitObjMissingDirectory(t *testing.T) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	_, err = InitObj(ten, filepath.Join(t.TempDir(), "absent"))
	if err == nil {
		t.Fatalf("expected error for missing directory, got nil")
	}

	if k := ResolveObjKind(ten, "effect", "quest", "gate", "0"); k != ObjKindEnvironment {
		t.Fatalf("expected %s, got %s", ObjKindEnvironment, k)
	}
}

func TestInitObjTenantIsolation(t *testing.T) {
	tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to create tenant A: %v", err)
	}
	tenB, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("failed to create tenant B: %v", err)
	}

	dir := t.TempDir()
	writeObjFixture(t, dir)

	if _, err := InitObj(tenA, dir); err != nil {
		t.Fatalf("InitObj returned error: %v", err)
	}

	if k := ResolveObjKind(tenB, "effect", "quest", "gate", "0"); k != ObjKindEnvironment {
		t.Fatalf("expected %s, got %s", ObjKindEnvironment, k)
	}
}
