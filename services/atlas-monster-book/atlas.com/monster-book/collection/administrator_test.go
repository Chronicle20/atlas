package collection

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
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

func TestUpsertCreatesAndUpdates(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 1, SpecialCount: 0, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 2, SpecialCount: 1, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, err := getByCharacter(db, tid, 7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.NormalCount != 2 || got.SpecialCount != 1 {
		t.Fatalf("expected (2,1) got (%d,%d)", got.NormalCount, got.SpecialCount)
	}
}

func TestSetCoverIdempotent(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 1, SpecialCount: 0, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	eid := uuid.New()
	changed, err := setCover(db, tid, 7, 2380000, 0, eid)
	if err != nil || !changed {
		t.Fatalf("first set: changed=%v err=%v", changed, err)
	}
	changed, err = setCover(db, tid, 7, 2380001, 0, eid) // same eventId, should be no-op
	if err != nil || changed {
		t.Fatalf("dup eventId: changed=%v err=%v", changed, err)
	}
	got, _ := getByCharacter(db, tid, 7)
	if got.CoverCardId != 2380000 {
		t.Fatalf("cover should still be 2380000, got %d", got.CoverCardId)
	}
}

func TestSetCoverPersistsMobId(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	cid := character.Id(7)

	// setCover updates an existing row; seed one first.
	if _, err := upsertStats(db, tid, cid, statsUpdate{}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ev := uuid.New()
	changed, err := setCover(db, tid, cid, item.Id(2380000), 100100, ev)
	if err != nil || !changed {
		t.Fatalf("setCover: changed=%v err=%v", changed, err)
	}

	e, err := getByCharacter(db, tid, cid)
	if err != nil {
		t.Fatalf("getByCharacter: %v", err)
	}
	if e.CoverCardId != 2380000 || e.CoverMobId != 100100 {
		t.Fatalf("persisted cardId=%d mobId=%d, want 2380000/100100", e.CoverCardId, e.CoverMobId)
	}

	// Duplicate eventId must no-op and must NOT overwrite the stored mob id.
	changed2, err := setCover(db, tid, cid, item.Id(0), 0, ev)
	if err != nil {
		t.Fatalf("setCover dup: %v", err)
	}
	if changed2 {
		t.Fatal("duplicate eventId should report changed=false")
	}
	e2, _ := getByCharacter(db, tid, cid)
	if e2.CoverMobId != 100100 || e2.CoverCardId != 2380000 {
		t.Fatalf("duplicate eventId overwrote cover: cardId=%d mobId=%d", e2.CoverCardId, e2.CoverMobId)
	}
}
