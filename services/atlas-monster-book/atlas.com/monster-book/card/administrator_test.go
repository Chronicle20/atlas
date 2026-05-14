package card

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUpsertFirstInsertion(t *testing.T) {
	db, tid, eid := newDB(t), uuid.New(), uuid.New()
	res, err := upsertCard(db, tid, 1, 2380000, eid)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !res.Inserted || res.NewLevel != 1 || res.Duplicate {
		t.Fatalf("got %+v", res)
	}
}

func TestUpsertLevelsUpToFive(t *testing.T) {
	db, tid := newDB(t), uuid.New()
	for i := 1; i <= 7; i++ {
		eid := uuid.New()
		res, err := upsertCard(db, tid, 1, 2380000, eid)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		expectedLevel := uint8(i)
		if expectedLevel > MaxLevel {
			expectedLevel = MaxLevel
		}
		if res.NewLevel != expectedLevel {
			t.Fatalf("step %d: want level %d, got %d", i, expectedLevel, res.NewLevel)
		}
		if res.Inserted != (i == 1) {
			t.Fatalf("step %d: inserted=%v", i, res.Inserted)
		}
		if res.Full != (i >= int(MaxLevel)) {
			t.Fatalf("step %d: full=%v", i, res.Full)
		}
		if res.Duplicate {
			t.Fatalf("step %d: unexpected duplicate", i)
		}
	}
}

func TestUpsertDuplicateEventIdNoOp(t *testing.T) {
	db, tid, eid := newDB(t), uuid.New(), uuid.New()
	if _, err := upsertCard(db, tid, 1, 2380000, eid); err != nil {
		t.Fatalf("first: %v", err)
	}
	res, err := upsertCard(db, tid, 1, 2380000, eid)
	if err != nil {
		t.Fatalf("dup: %v", err)
	}
	if !res.Duplicate || res.NewLevel != 1 {
		t.Fatalf("got %+v", res)
	}
}
