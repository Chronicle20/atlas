package script

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
		t.Fatalf("Failed to migrate reactor_scripts: %v", err)
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

func insertCountScript(t *testing.T, p Processor, reactorId string) {
	t.Helper()
	m := NewReactorScriptBuilder().
		SetReactorId(reactorId).
		SetDescription("count test script").
		Build()
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create reactor script %s: %v", reactorId, err)
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
	insertCountScript(t, p, "reactor_a")
	insertCountScript(t, p, "reactor_b")

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

	insertCountScript(t, p1, "tenant1_a")
	insertCountScript(t, p1, "tenant1_b")
	insertCountScript(t, p2, "tenant2_a")
	insertCountScript(t, p2, "tenant2_b")
	insertCountScript(t, p2, "tenant2_c")

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}

func TestProcessTouch(t *testing.T) {
	tests := []struct {
		name        string
		reactorId   string
		storeScript bool
		touchRules  []Rule
		hitRules    []Rule
		wantRule    string
		wantErr     bool
	}{
		{
			name:        "touchRules wins",
			reactorId:   "reactor_touch_wins",
			storeScript: true,
			touchRules:  []Rule{NewRuleBuilder().SetId("t1").Build()},
			hitRules:    []Rule{NewRuleBuilder().SetId("h1").Build()},
			wantRule:    "t1",
		},
		{
			name:        "falls back to hitRules",
			reactorId:   "reactor_touch_fallback",
			storeScript: true,
			touchRules:  nil,
			hitRules:    []Rule{NewRuleBuilder().SetId("h1").Build()},
			wantRule:    "h1",
		},
		{
			name:        "no rules at all",
			reactorId:   "reactor_touch_no_rules",
			storeScript: true,
			touchRules:  nil,
			hitRules:    nil,
			wantRule:    "no_match",
		},
		{
			name:        "no script",
			reactorId:   "reactor_touch_no_script",
			storeScript: false,
			wantRule:    "no_script",
		},
	}

	l, _ := test.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := newCountTestDB(t)
	p := NewProcessor(l, ctx, db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.storeScript {
				b := NewReactorScriptBuilder().SetReactorId(tt.reactorId)
				for _, r := range tt.touchRules {
					b.AddTouchRule(r)
				}
				for _, r := range tt.hitRules {
					b.AddHitRule(r)
				}
				if _, err := p.Create(b.Build()); err != nil {
					t.Fatalf("Create reactor script %s: %v", tt.reactorId, err)
				}
			}

			result := p.ProcessTouch(tt.reactorId, 0, 1)
			if result.MatchedRule != tt.wantRule {
				t.Errorf("ProcessTouch() MatchedRule = %v, want %v", result.MatchedRule, tt.wantRule)
			}
			if tt.wantErr && result.Error == nil {
				t.Errorf("ProcessTouch() expected error, got nil")
			}
			if !tt.wantErr && result.Error != nil {
				t.Errorf("ProcessTouch() unexpected error: %v", result.Error)
			}
		})
	}
}

func TestHandleCommandFuncRoutesTouch(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := newCountTestDB(t)

	p := NewProcessor(l, ctx, db)
	m := NewReactorScriptBuilder().
		SetReactorId("reactor_touch_route").
		AddTouchRule(NewRuleBuilder().SetId("t1").Build()).
		Build()
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create reactor script: %v", err)
	}

	handler := handleCommandFunc(l, db)

	t.Run("TOUCH reaches ProcessTouch", func(t *testing.T) {
		hook.Reset()
		command := commandEvent[interface{}]{
			ReactorId:      1,
			Classification: "reactor_touch_route",
			ReactorState:   0,
			Type:           CommandTypeTouch,
			Body:           map[string]interface{}{"characterId": float64(1)},
		}
		handler(l, ctx, command)

		found := false
		for _, entry := range hook.Entries {
			if entry.Message == "Reactor touch script [reactor_touch_route] result: matchedRule=t1, operations=0" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected touch result log entry with matchedRule=t1, got entries: %v", hook.Entries)
		}
	})

	t.Run("unknown type warns and is ignored", func(t *testing.T) {
		hook.Reset()
		command := commandEvent[interface{}]{
			ReactorId:      1,
			Classification: "reactor_touch_route",
			ReactorState:   0,
			Type:           "BOGUS",
			Body:           map[string]interface{}{"characterId": float64(1)},
		}
		handler(l, ctx, command)

		if len(hook.Entries) != 1 {
			t.Fatalf("Expected exactly 1 log entry for unknown command type, got %d", len(hook.Entries))
		}
		if hook.LastEntry().Message != "Unknown command type: BOGUS" {
			t.Errorf("Expected warn log for unknown command type, got %q", hook.LastEntry().Message)
		}
	})
}
