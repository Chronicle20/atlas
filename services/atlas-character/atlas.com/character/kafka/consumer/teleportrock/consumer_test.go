package teleportrock

import (
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"atlas-character/teleport_rock"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testDB(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	// Uniquely-named shared-cache in-memory database, mirroring
	// teleport_rock/administrator_test.go: a bare ":memory:" DB is private to
	// one connection, so a second pooled connection can see an empty schema.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetConnMaxIdleTime(0)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := teleport_rock.Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// A command with the wrong Type must be a no-op (each handler receives every
// message on the topic).
func TestHandleAddMapIgnoresWrongType(t *testing.T) {
	db := testDB(t)
	l, _ := test.NewNullLogger()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)

	handleAddMap(db)(l, ctx, teleportrock2.Command[teleportrock2.AddMapCommandBody]{
		Type: teleportrock2.CommandRemoveMap, // wrong type for this handler
		Body: teleportrock2.AddMapCommandBody{MapId: 100000000, Vip: false},
	})

	m, err := teleport_rock.NewProcessor(l, ctx, db).GetByCharacterId(0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(m.Regular()) != 0 {
		t.Fatalf("wrong-type command must not mutate: %v", m.Regular())
	}
}

// A command with the wrong Type must be a no-op for RemoveMap too.
func TestHandleRemoveMapIgnoresWrongType(t *testing.T) {
	db := testDB(t)
	l, _ := test.NewNullLogger()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)

	handleRemoveMap(db)(l, ctx, teleportrock2.Command[teleportrock2.RemoveMapCommandBody]{
		Type: teleportrock2.CommandAddMap, // wrong type for this handler
		Body: teleportrock2.RemoveMapCommandBody{MapId: 100000000, Vip: false},
	})

	m, err := teleport_rock.NewProcessor(l, ctx, db).GetByCharacterId(0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(m.Regular()) != 0 {
		t.Fatalf("wrong-type command must not mutate: %v", m.Regular())
	}
}
