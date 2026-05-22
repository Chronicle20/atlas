package tenantpurge

import (
	"context"
	"testing"

	"atlas-data/canonical"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPurgeRefusesCanonical(t *testing.T) {
	err := Purge(context.Background(), logrus.New(), nil, nil, uuid.MustParse(canonical.TenantUUID))
	if err == nil || err != ErrCanonicalRefused {
		t.Fatalf("expected ErrCanonicalRefused, got %v", err)
	}
}

func TestPurgeDeletesRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Create a stripped documents table with the columns we touch.
	if err := db.Exec(`CREATE TABLE documents (id TEXT, tenant_id TEXT, type TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE monster_search_index (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE npc_search_index (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE reactor_search_index (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE map_search_index (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE item_string_search_index (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE tenant_baselines (tenant_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	if err := db.Exec(`INSERT INTO documents (id, tenant_id, type) VALUES ('a', ?, 'item')`, id.String()).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO tenant_baselines (tenant_id) VALUES (?)`, id.String()).Error; err != nil {
		t.Fatal(err)
	}
	if err := Purge(context.Background(), logrus.New(), db, nil, id); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Raw("SELECT COUNT(*) FROM documents").Scan(&n)
	if n != 0 {
		t.Fatalf("expected documents empty, got %d", n)
	}
	db.Raw("SELECT COUNT(*) FROM tenant_baselines").Scan(&n)
	if n != 0 {
		t.Fatalf("expected tenant_baselines empty, got %d", n)
	}
}
