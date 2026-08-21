package playernpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testDatabase mirrors services/atlas-notes/atlas.com/notes/note/processor_test.go's
// setup, with the driver's own foreign-key DSN param turned on (mattn/go-sqlite3's
// `_foreign_keys=1`, not the `_pragma=foreign_keys(1)` form other services'
// tests use, which that driver does not recognize) so SQLite actually
// enforces the player_npc_equipment -> player_npcs cascade.
func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	l := testLogger()
	database.RegisterTenantCallbacks(l, db)

	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

// buildDeployedNpc returns a Model suitable for a create, with the given
// world/map/name/scriptId/objectId as the fields the DB constraints
// exercise and every other field filled with valid placeholder values, plus
// one equipment row.
func buildDeployedNpc(t *testing.T, worldId byte, mapId uint32, name string, scriptId uint32, objectId uint32) Model {
	t.Helper()

	em, err := NewEquipmentBuilder().
		SetSlot(-1).
		SetItemId(1002140).
		Build()
	if err != nil {
		t.Fatalf("Failed to build equipment: %v", err)
	}

	m, err := NewBuilder().
		SetCharacterId(1).
		SetName(name).
		SetWorldId(worldId).
		SetMapId(mapId).
		SetScriptId(scriptId).
		SetObjectId(objectId).
		SetGender(0).
		SetSkin(1).
		SetFace(20000).
		SetHair(30000).
		SetX(100).
		SetCy(200).
		SetFh(17).
		SetEquipment([]EquipmentModel{em}).
		Build()
	if err != nil {
		t.Fatalf("Failed to build model: %v", err)
	}
	return m
}

func TestPlayerNpcAdministrator(t *testing.T) {
	t.Run("create persists the row and its equipment", func(t *testing.T) {
		db := testDatabase(t)
		te := testTenant(t)
		ctx := tenant.WithContext(context.Background(), te)

		m := buildDeployedNpc(t, 0, 102000004, "Statue Guy", 9901000, 1)

		created, err := createPlayerNpc(db.WithContext(ctx), te.Id(), m)
		if err != nil {
			t.Fatalf("createPlayerNpc() unexpected err = %v", err)
		}
		if created.Id() == uuid.Nil {
			t.Fatalf("createPlayerNpc() did not assign an id")
		}
		if created.Name() != "Statue Guy" {
			t.Fatalf("Name() = %v, want Statue Guy", created.Name())
		}
		if len(created.Equipment()) != 1 {
			t.Fatalf("len(Equipment()) = %v, want 1", len(created.Equipment()))
		}
		if created.Equipment()[0].ItemId() != 1002140 {
			t.Fatalf("Equipment()[0].ItemId() = %v, want 1002140", created.Equipment()[0].ItemId())
		}
	})

	t.Run("duplicate name on the same map is rejected", func(t *testing.T) {
		db := testDatabase(t)
		te := testTenant(t)
		ctx := tenant.WithContext(context.Background(), te)

		first := buildDeployedNpc(t, 0, 102000004, "Duplicate", 9901000, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), first); err != nil {
			t.Fatalf("first createPlayerNpc() unexpected err = %v", err)
		}

		second := buildDeployedNpc(t, 0, 102000004, "Duplicate", 9901001, 2)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), second); err == nil {
			t.Fatalf("second createPlayerNpc() with duplicate (world, map, name) expected err, got nil")
		}
	})

	t.Run("duplicate script id in the same world is rejected", func(t *testing.T) {
		db := testDatabase(t)
		te := testTenant(t)
		ctx := tenant.WithContext(context.Background(), te)

		first := buildDeployedNpc(t, 0, 102000004, "First", 9901000, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), first); err != nil {
			t.Fatalf("first createPlayerNpc() unexpected err = %v", err)
		}

		second := buildDeployedNpc(t, 0, 102000005, "Second", 9901000, 2)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), second); err == nil {
			t.Fatalf("second createPlayerNpc() with duplicate (world, script_id) expected err, got nil")
		}
	})

	t.Run("duplicate object id on the same map is rejected", func(t *testing.T) {
		db := testDatabase(t)
		te := testTenant(t)
		ctx := tenant.WithContext(context.Background(), te)

		first := buildDeployedNpc(t, 0, 102000004, "First", 9901000, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), first); err != nil {
			t.Fatalf("first createPlayerNpc() unexpected err = %v", err)
		}

		second := buildDeployedNpc(t, 0, 102000004, "Second", 9901001, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx), te.Id(), second); err == nil {
			t.Fatalf("second createPlayerNpc() with duplicate (world, map, object_id) expected err, got nil")
		}
	})

	t.Run("cascade delete removes equipment rows", func(t *testing.T) {
		db := testDatabase(t)
		te := testTenant(t)
		ctx := tenant.WithContext(context.Background(), te)

		m := buildDeployedNpc(t, 0, 102000004, "Statue Guy", 9901000, 1)
		created, err := createPlayerNpc(db.WithContext(ctx), te.Id(), m)
		if err != nil {
			t.Fatalf("createPlayerNpc() unexpected err = %v", err)
		}

		if err := deletePlayerNpc(db.WithContext(ctx), created.Id()); err != nil {
			t.Fatalf("deletePlayerNpc() unexpected err = %v", err)
		}

		var count int64
		if err := db.Table("player_npc_equipment").Where("player_npc_id = ?", created.Id()).Count(&count).Error; err != nil {
			t.Fatalf("Failed to count equipment rows: %v", err)
		}
		if count != 0 {
			t.Fatalf("player_npc_equipment rows remaining after delete = %v, want 0", count)
		}
	})

	t.Run("cross-tenant isolation", func(t *testing.T) {
		db := testDatabase(t)
		te1 := testTenant(t)
		te2 := testTenant(t)
		ctx1 := tenant.WithContext(context.Background(), te1)
		ctx2 := tenant.WithContext(context.Background(), te2)

		m1 := buildDeployedNpc(t, 0, 102000004, "Tenant One", 9901000, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx1), te1.Id(), m1); err != nil {
			t.Fatalf("tenant 1 createPlayerNpc() unexpected err = %v", err)
		}

		m2 := buildDeployedNpc(t, 0, 102000004, "Tenant Two", 9901001, 1)
		if _, err := createPlayerNpc(db.WithContext(ctx2), te2.Id(), m2); err != nil {
			t.Fatalf("tenant 2 createPlayerNpc() unexpected err = %v", err)
		}

		rows1, err := playerNpcsByMap(db.WithContext(ctx1), 0, 102000004, model.Page{Number: 1, Size: 50})
		if err != nil {
			t.Fatalf("tenant 1 playerNpcsByMap() unexpected err = %v", err)
		}
		rows2, err := playerNpcsByMap(db.WithContext(ctx2), 0, 102000004, model.Page{Number: 1, Size: 50})
		if err != nil {
			t.Fatalf("tenant 2 playerNpcsByMap() unexpected err = %v", err)
		}

		if len(rows1) != 1 || rows1[0].Name() != "Tenant One" {
			t.Fatalf("tenant 1 rows = %+v, want exactly [Tenant One]", rows1)
		}
		if len(rows2) != 1 || rows2[0].Name() != "Tenant Two" {
			t.Fatalf("tenant 2 rows = %+v, want exactly [Tenant Two]", rows2)
		}
	})
}
