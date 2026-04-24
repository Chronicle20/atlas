package script

import (
	"context"
	"fmt"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCountTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := MigrateTable(db); err != nil {
		t.Fatalf("Failed to migrate map_scripts: %v", err)
	}
	return db
}

func countTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func insertCountScript(t *testing.T, p ScriptProcessor, scriptName, scriptType string) {
	t.Helper()
	m := NewMapScriptBuilder().
		SetScriptName(scriptName).
		SetScriptType(scriptType).
		SetDescription("count test script").
		Build()
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create map script %s/%s: %v", scriptName, scriptType, err)
	}
}

func TestProcessorImpl_Count_Empty(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := newCountTestDB(t)

	p := NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := test.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := newCountTestDB(t)

	p := NewProcessor(l, ctx, db)
	insertCountScript(t, p, "map_a", "onEnter")
	insertCountScript(t, p, "map_b", "onFirstEnter")

	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	if updated == nil {
		t.Fatalf("updatedAt is nil; expected non-nil")
	}
	if time.Since(*updated) > 5*time.Second {
		t.Errorf("updatedAt too old: %v", *updated)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := newCountTestDB(t)

	p1 := NewProcessor(l, ctx1, db)
	p2 := NewProcessor(l, ctx2, db)

	insertCountScript(t, p1, "tenant1_a", "onEnter")
	insertCountScript(t, p1, "tenant1_b", "onEnter")
	insertCountScript(t, p2, "tenant2_a", "onEnter")
	insertCountScript(t, p2, "tenant2_b", "onEnter")
	insertCountScript(t, p2, "tenant2_c", "onEnter")

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}
