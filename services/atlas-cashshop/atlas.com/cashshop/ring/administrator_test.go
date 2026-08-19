package ring

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// testDatabase returns a fresh sqlite-in-memory *gorm.DB migrated for this
// package, via the module's established in-memory test-database helper.
func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration)
}

func TestCreatePair(t *testing.T) {
	db := testDatabase(t)
	a := uuid.New()

	var pairId uuid.UUID

	t.Run("creates two halves sharing a pair id", func(t *testing.T) {
		var err error
		pairId, err = CreatePair(db, a, TypeCouple, Half{CharacterId: 42, AssetId: 1001, ItemTemplateId: 1112000}, Half{CharacterId: 77, AssetId: 1002, ItemTemplateId: 1112000})
		if err != nil {
			t.Fatalf("CreatePair: %v", err)
		}
		if pairId == uuid.Nil {
			t.Fatalf("CreatePair: got nil pair id")
		}

		rows, err := GetByCharacterId(db, a, 42)
		if err != nil {
			t.Fatalf("GetByCharacterId: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("GetByCharacterId(42) = %d rows, want 1", len(rows))
		}
		m := rows[0]
		if m.PairId() != pairId {
			t.Fatalf("PairId() = %s, want %s", m.PairId(), pairId)
		}
		if m.CharacterId() != 42 {
			t.Fatalf("CharacterId() = %d, want 42", m.CharacterId())
		}
		if m.PartnerCharacterId() != 77 {
			t.Fatalf("PartnerCharacterId() = %d, want 77", m.PartnerCharacterId())
		}
		if m.AssetId() != 1001 {
			t.Fatalf("AssetId() = %d, want 1001", m.AssetId())
		}
		if m.Type() != TypeCouple {
			t.Fatalf("Type() = %s, want %s", m.Type(), TypeCouple)
		}
		if m.State() != StateActive {
			t.Fatalf("State() = %s, want %s", m.State(), StateActive)
		}
	})

	t.Run("the partner half mirrors it", func(t *testing.T) {
		rows, err := GetByCharacterId(db, a, 77)
		if err != nil {
			t.Fatalf("GetByCharacterId: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("GetByCharacterId(77) = %d rows, want 1", len(rows))
		}
		m := rows[0]
		if m.PairId() != pairId {
			t.Fatalf("PairId() = %s, want %s", m.PairId(), pairId)
		}
		if m.CharacterId() != 77 {
			t.Fatalf("CharacterId() = %d, want 77", m.CharacterId())
		}
		if m.PartnerCharacterId() != 42 {
			t.Fatalf("PartnerCharacterId() = %d, want 42", m.PartnerCharacterId())
		}
		if m.AssetId() != 1002 {
			t.Fatalf("AssetId() = %d, want 1002", m.AssetId())
		}
	})

	t.Run("friendship pairs are distinguishable", func(t *testing.T) {
		_, err := CreatePair(db, a, TypeFriendship, Half{CharacterId: 42, AssetId: 1003, ItemTemplateId: 1112800}, Half{CharacterId: 88, AssetId: 1004, ItemTemplateId: 1112800})
		if err != nil {
			t.Fatalf("CreatePair: %v", err)
		}

		rows, err := GetByCharacterId(db, a, 42)
		if err != nil {
			t.Fatalf("GetByCharacterId: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("GetByCharacterId(42) = %d rows, want 2", len(rows))
		}
		friendshipCount := 0
		for _, m := range rows {
			if m.Type() == TypeFriendship {
				friendshipCount++
			}
		}
		if friendshipCount != 1 {
			t.Fatalf("friendship halves = %d, want 1", friendshipCount)
		}
	})

	t.Run("another tenant sees nothing", func(t *testing.T) {
		rows, err := GetByCharacterId(db, uuid.New(), 42)
		if err != nil {
			t.Fatalf("GetByCharacterId: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("GetByCharacterId = %d rows, want 0", len(rows))
		}
	})

	t.Run("a character with no rings is not an error", func(t *testing.T) {
		rows, err := GetByCharacterId(db, a, 999)
		if err != nil {
			t.Fatalf("GetByCharacterId: want nil error, got %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("GetByCharacterId = %d rows, want 0", len(rows))
		}
	})
}

// failWritesForCharacter fails any write to cash_rings that includes a row
// for the given characterId, whether that row arrives as its own db.Create
// call or as one element of a batched db.Create(&[]Entity{...}) call. This
// is deliberately more targeted than databasetest.FailWritesOn(table): a
// blanket "every write to this table fails" would fail the very first
// statement no matter how CreatePair is implemented, and so could never
// distinguish "two separate Create calls, first one committed" from "one
// atomic batch insert, nothing committed" -- exactly the distinction
// FR-RING-4 requires this test to make.
func failWritesForCharacter(t *testing.T, db *gorm.DB, characterId uint32) {
	t.Helper()
	fail := func(d *gorm.DB) {
		if d.Statement == nil || d.Statement.Table != "cash_rings" {
			return
		}
		switch v := d.Statement.Dest.(type) {
		case *Entity:
			if v.CharacterId == characterId {
				_ = d.AddError(fmt.Errorf("ring: injected failure for character %d", characterId))
			}
		case []Entity:
			for _, e := range v {
				if e.CharacterId == characterId {
					_ = d.AddError(fmt.Errorf("ring: injected failure for character %d", characterId))
					return
				}
			}
		case *[]Entity:
			for _, e := range *v {
				if e.CharacterId == characterId {
					_ = d.AddError(fmt.Errorf("ring: injected failure for character %d", characterId))
					return
				}
			}
		}
	}
	name := fmt.Sprintf("ring_test:fail_char_%d", characterId)
	if err := db.Callback().Create().Before("gorm:create").Register(name, fail); err != nil {
		t.Fatalf("register callback: %v", err)
	}
}

// TestCreatePairIsAtomic proves a partial pair cannot be persisted
// (FR-RING-4). It forces the write for ONE half (character 77) to fail and
// asserts that the OTHER half (character 42) was not left behind. A
// two-separate-db.Create-calls implementation would commit the first half
// before the injected failure ever fires, since neither call is itself
// wrapped in a transaction here; a single batched db.Create over both rows
// fails the whole statement before either row is written.
func TestCreatePairIsAtomic(t *testing.T) {
	db := testDatabase(t)
	a := uuid.New()

	failWritesForCharacter(t, db, 77)

	_, err := CreatePair(db, a, TypeCouple, Half{CharacterId: 42, AssetId: 1001, ItemTemplateId: 1112000}, Half{CharacterId: 77, AssetId: 1002, ItemTemplateId: 1112000})
	if err == nil {
		t.Fatalf("CreatePair: want error from injected failure, got nil")
	}

	rows42, err := GetByCharacterId(db, a, 42)
	if err != nil {
		t.Fatalf("GetByCharacterId(42): %v", err)
	}
	if len(rows42) != 0 {
		t.Fatalf("GetByCharacterId(42) = %d rows, want 0 (partial pair persisted)", len(rows42))
	}

	rows77, err := GetByCharacterId(db, a, 77)
	if err != nil {
		t.Fatalf("GetByCharacterId(77): %v", err)
	}
	if len(rows77) != 0 {
		t.Fatalf("GetByCharacterId(77) = %d rows, want 0", len(rows77))
	}
}
