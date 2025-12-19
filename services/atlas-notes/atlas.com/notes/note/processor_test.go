package note_test

import (
	"atlas-notes/kafka/message"
	"atlas-notes/note"
	"context"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, note.Migration)

	for _, migrator := range migrators {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestProcessorImpl_CRUD(t *testing.T) {
	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t)

	np := note.NewProcessor(l, ctx, db)

	characterId := uint32(1)
	senderId := uint32(2)
	msg := "Hello!"
	flag := byte(0)

	mb := message.NewBuffer()
	nm, err := np.Create(mb)(characterId)(senderId)(msg)(flag)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	if nm.CharacterId() != characterId {
		t.Fatalf("Unexpected characterId")
	}
	if nm.SenderId() != senderId {
		t.Fatalf("Unexpected senderId")
	}
	if nm.Message() != msg {
		t.Fatalf("Unexpected message")
	}
	if nm.Flag() != flag {
		t.Fatalf("Unexpected flag")
	}
}
