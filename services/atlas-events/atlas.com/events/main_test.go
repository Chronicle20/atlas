package main

import (
	"atlas-events/event/definition"
	"atlas-events/event/registry"
	"atlas-events/event/scheduling"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// mainWiringTestTenantId is fixed (rather than derived) purely to build a
// tenant-scoped context for this file's DB fixture.
var mainWiringTestTenantId = uuid.MustParse("33333333-4444-5555-6666-777777777777")

// TestEventOrchestrationIsWiredAtStartup pins the seam described in
// event/definition/resource.go's EnabledOrchestrator doc comment: unless
// something assigns it, updateDefinitionHandler silently falls back to the
// plain toggle-only Processor.SetEnabled and FR-A2's schedule-on-enable
// never fires in production — with nothing anywhere failing (task-231
// fix-round-1 finding 2). init() in main.go is the only production assigner,
// and go test runs package init unconditionally before any test body, so
// this fails exactly when that assignment is missing.
func TestEventOrchestrationIsWiredAtStartup(t *testing.T) {
	if definition.EnabledOrchestrator == nil {
		t.Fatal("definition.EnabledOrchestrator is nil — FR-A2 schedule-on-enable is unwired (see main.go's init())")
	}
}

// mainWiringStubType/mainWiringStubHandler exist only so
// definition.Processor.Create's registry.Get(m.Type()).ValidateConfiguration
// lookup succeeds for the definition this test seeds — orchestration.SetEnabled
// itself is what is under test, not this handler's behavior.
const mainWiringStubType = "MAIN_WIRING_TEST_STUB"

type mainWiringStubHandler struct{}

func (mainWiringStubHandler) Type() string                                { return mainWiringStubType }
func (mainWiringStubHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (mainWiringStubHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (mainWiringStubHandler) ConcurrencyKeyIsConstant() bool { return false }
func (mainWiringStubHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (mainWiringStubHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (mainWiringStubHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

// TestEventOrchestrationWiringSchedulesOnEnable is the functional pin: it
// exercises the EXACT call chain updateDefinitionHandler uses
// (definition.EnabledOrchestrator(l, ctx, db)(id, true)) rather than calling
// orchestration.SetEnabled directly, so it fails not only if the var is nil
// but also if it were ever wired to something other than the real
// FR-A2-scheduling implementation.
func TestEventOrchestrationWiringSchedulesOnEnable(t *testing.T) {
	if definition.EnabledOrchestrator == nil {
		t.Fatal("definition.EnabledOrchestrator is nil — cannot exercise the wiring")
	}

	db := newMainWiringTestDB(t)
	ctx := databasetest.TenantContext(mainWiringTestTenantId)
	l, _ := test.NewNullLogger()

	registry.ResetForTest()
	registry.Register(mainWiringStubHandler{})

	m, err := definition.NewBuilder(mainWiringStubType, "main-wiring-test").SetConfiguration(json.RawMessage("{}")).Build()
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}
	created, err := definition.NewProcessor(l, ctx, db).Create(m)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := definition.EnabledOrchestrator(l, ctx, db)(created.Id(), true); err != nil {
		t.Fatalf("EnabledOrchestrator(...)(id, true): %v", err)
	}

	var rows []scheduling.Entity
	if err := db.Where("dedupe_key = ?", "enable:"+created.Id().String()).Find(&rows).Error; err != nil {
		t.Fatalf("query scheduled_event_work: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("scheduled %d rows on enable, want 1 (FR-A2)", len(rows))
	}
	if rows[0].Type != scheduling.WorkTypeTriggerEvaluation {
		t.Fatalf("type = %s, want %s", rows[0].Type, scheduling.WorkTypeTriggerEvaluation)
	}
}

func newMainWiringTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, definition.MigrateTable, scheduling.MigrateTable)
}
