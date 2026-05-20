package seeder

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SeedState{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestSeedState_TableName(t *testing.T) {
	if got := (SeedState{}).TableName(); got != "seed_state" {
		t.Fatalf("TableName = %q, want seed_state", got)
	}
}

func TestSeedState_UpsertReplacesExistingRow(t *testing.T) {
	db := openTestDB(t)
	tenantID := uuid.New()
	first := SeedState{
		TenantID:        tenantID,
		GroupName:       "drops",
		CatalogRevision: "rev-1",
		SeededAt:        time.Now().UTC(),
		ResultSummary:   datatypes.JSON(`{"groupName":"drops"}`),
	}
	if err := UpsertSeedState(db, &first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second := first
	second.CatalogRevision = "rev-2"
	second.ResultSummary = datatypes.JSON(`{"groupName":"drops","run":2}`)
	if err := UpsertSeedState(db, &second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := ReadSeedState(db, tenantID, "drops")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got == nil || got.CatalogRevision != "rev-2" {
		t.Fatalf("CatalogRevision = %v, want rev-2", got)
	}
	var summary map[string]any
	if err := json.Unmarshal(got.ResultSummary, &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summary["run"] != float64(2) {
		t.Fatalf("ResultSummary not replaced: %v", summary)
	}
}

func TestReadSeedState_NotFoundReturnsNil(t *testing.T) {
	db := openTestDB(t)
	got, err := ReadSeedState(db, uuid.New(), "drops")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("got = %+v, want nil", got)
	}
}
