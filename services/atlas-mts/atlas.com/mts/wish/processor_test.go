package wish_test

import (
	"atlas-mts/test"
	"atlas-mts/wish"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"gorm.io/gorm"
)

// resetWishes clears the wish_entries table. The shared in-memory SQLite DB is
// reused across tests in the process, and these processor tests all run under
// the fixed test tenant, so rows from prior tests would otherwise leak into
// GetByCharacter/GetAll counts.
func resetWishes(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM wish_entries").Error; err != nil {
		t.Fatalf("reset wish_entries: %v", err)
	}
}

// buildProcessorWish builds a wish entry for the test tenant. The tenant id MUST
// match the processor's context tenant (test.TestTenantId) so the row is visible
// through the processor's tenant-scoped queries.
func buildProcessorWish(t *testing.T, characterId uint32, itemId uint32) wish.Model {
	t.Helper()
	m, err := wish.NewBuilder(test.TestTenantId, characterId, itemId).Build()
	if err != nil {
		t.Fatalf("Failed to build wish: %v", err)
	}
	return m
}

// TestProcessorCreateGetById asserts a created wish entry round-trips through
// the processor's GetById.
func TestProcessorCreateGetById(t *testing.T) {
	p, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	resetWishes(t, db)

	created, err := p.Create(buildProcessorWish(t, 100, 1302000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Id().String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("Create did not assign an id")
	}

	got, err := p.GetById(created.Id().String())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Id() != created.Id() {
		t.Errorf("id = %s, want %s", got.Id(), created.Id())
	}
	if got.CharacterId() != 100 {
		t.Errorf("characterId = %d, want 100", got.CharacterId())
	}
	if got.ItemId() != 1302000 {
		t.Errorf("itemId = %d, want 1302000", got.ItemId())
	}
}

// TestProcessorGetByCharacter asserts GetByCharacter filters by character.
func TestProcessorGetByCharacter(t *testing.T) {
	p, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	resetWishes(t, db)

	if _, err := p.Create(buildProcessorWish(t, 100, 1302000)); err != nil {
		t.Fatalf("Create c100 #1: %v", err)
	}
	if _, err := p.Create(buildProcessorWish(t, 100, 1302001)); err != nil {
		t.Fatalf("Create c100 #2: %v", err)
	}
	if _, err := p.Create(buildProcessorWish(t, 101, 1302000)); err != nil {
		t.Fatalf("Create c101: %v", err)
	}

	got, err := p.GetByCharacter(100)
	if err != nil {
		t.Fatalf("GetByCharacter: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetByCharacter(100) returned %d rows, want 2", len(got))
	}
	for _, w := range got {
		if w.CharacterId() != 100 {
			t.Errorf("GetByCharacter returned wrong row: character=%d", w.CharacterId())
		}
	}
}

// buildWantedWish builds a world-scoped want-ad (type=wanted) for the test
// tenant. GetWantedByWorld filters on (world_id, type=wanted), so the test seeds
// must carry both a world and the wanted type.
func buildWantedWish(t *testing.T, worldId byte, characterId uint32, itemId uint32) wish.Model {
	t.Helper()
	m, err := wish.NewBuilder(test.TestTenantId, characterId, itemId).
		SetWorldId(world.Id(worldId)).
		SetType(wish.TypeWanted).
		Build()
	if err != nil {
		t.Fatalf("Failed to build wanted wish: %v", err)
	}
	return m
}

// TestProcessorGetWantedByWorld asserts GetWantedByWorld returns every want-ad in
// a world across ALL characters, and excludes both cart entries and other worlds'
// want-ads.
func TestProcessorGetWantedByWorld(t *testing.T) {
	p, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	resetWishes(t, db)

	// Two characters' want-ads in world 0.
	if _, err := p.Create(buildWantedWish(t, 0, 100, 1302000)); err != nil {
		t.Fatalf("Create wanted c100: %v", err)
	}
	if _, err := p.Create(buildWantedWish(t, 0, 101, 1302001)); err != nil {
		t.Fatalf("Create wanted c101: %v", err)
	}
	// A cart entry in world 0 (type=cart, default builder) must NOT surface.
	if _, err := p.Create(buildProcessorWish(t, 102, 1302002)); err != nil {
		t.Fatalf("Create cart c102: %v", err)
	}
	// A want-ad in world 1 must NOT surface for a world-0 query.
	if _, err := p.Create(buildWantedWish(t, 1, 103, 1302003)); err != nil {
		t.Fatalf("Create wanted world1 c103: %v", err)
	}

	got, err := p.GetWantedByWorld(world.Id(0))
	if err != nil {
		t.Fatalf("GetWantedByWorld: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetWantedByWorld(0) returned %d rows, want 2 (cross-character want-ads only)", len(got))
	}
	for _, w := range got {
		if w.Type() != wish.TypeWanted {
			t.Errorf("GetWantedByWorld returned a non-wanted row (type=%s)", w.Type())
		}
		if w.WorldId() != world.Id(0) {
			t.Errorf("GetWantedByWorld returned a row from world %d, want 0", byte(w.WorldId()))
		}
		if w.CharacterId() != 100 && w.CharacterId() != 101 {
			t.Errorf("GetWantedByWorld returned unexpected character %d", w.CharacterId())
		}
	}
}

// TestProcessorDelete asserts Delete removes the wish: the first call returns
// true and the row vanishes from GetByCharacter; a second call returns false.
func TestProcessorDelete(t *testing.T) {
	p, db, cleanup := test.CreateWishProcessor(t)
	defer cleanup()
	resetWishes(t, db)

	created, err := p.Create(buildProcessorWish(t, 100, 1302000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := p.Delete(created.Id().String())
	if err != nil {
		t.Fatalf("Delete first: %v", err)
	}
	if !ok {
		t.Error("first Delete returned false, want true")
	}

	got, err := p.GetByCharacter(100)
	if err != nil {
		t.Fatalf("GetByCharacter after delete: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("GetByCharacter after delete returned %d rows, want 0", len(got))
	}

	ok, err = p.Delete(created.Id().String())
	if err != nil {
		t.Fatalf("Delete second: %v", err)
	}
	if ok {
		t.Error("second Delete returned true, want false")
	}
}
