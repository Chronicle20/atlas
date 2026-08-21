package orchestration

import (
	"atlas-events/event/definition"
	"atlas-events/event/registry"
	"atlas-events/event/scheduling"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// stubType is a throwaway definition type this test registers a trivial
// handler for, purely so definition.Processor.Create's
// registry.Get(m.Type()).ValidateConfiguration lookup succeeds. Nothing
// about SetEnabled's own behavior depends on what this handler does.
const stubType = "ORCHESTRATION_TEST_STUB"

type stubHandler struct{}

func (stubHandler) Type() string                                { return stubType }
func (stubHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (stubHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (stubHandler) ConcurrencyKeyIsConstant() bool { return false }
func (stubHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (stubHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (stubHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

var registerOnce sync.Once

func ensureStubRegistered() {
	registerOnce.Do(func() { registry.Register(stubHandler{}) })
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, definition.MigrateTable, scheduling.MigrateTable)
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(uuid.MustParse("22222222-3333-4444-5555-666666666666"))
}

func seedDefinition(t *testing.T, db *gorm.DB, startEnabled bool) definition.Model {
	t.Helper()
	ensureStubRegistered()

	l, _ := test.NewNullLogger()
	m, err := definition.NewBuilder(stubType, "orchestration-test").SetConfiguration(json.RawMessage("{}")).Build()
	if err != nil {
		t.Fatalf("build definition: %v", err)
	}

	p := definition.NewProcessor(l, testCtx(t), db)
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if startEnabled {
		created, err = p.SetEnabled(created.Id(), true)
		if err != nil {
			t.Fatalf("SetEnabled(seed): %v", err)
		}
	}
	return created
}

func scheduledFor(t *testing.T, db *gorm.DB, dedupeKey string) []scheduling.Entity {
	t.Helper()
	var out []scheduling.Entity
	if err := db.Where("dedupe_key = ?", dedupeKey).Find(&out).Error; err != nil {
		t.Fatalf("query scheduled_event_work: %v", err)
	}
	return out
}

// FR-A2: a false->true transition schedules exactly one TRIGGER_EVALUATION
// row, with the enable:<definitionId> dedupe key SetEnabled's own doc
// comment promises.
func TestSetEnabledFalseToTrueSchedulesExactlyOneRow(t *testing.T) {
	db := newTestDB(t)
	l, _ := test.NewNullLogger()
	d := seedDefinition(t, db, false)

	before := time.Now()
	updated, err := SetEnabled(l, testCtx(t), db)(d.Id(), true)
	if err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if !updated.Enabled() {
		t.Fatalf("updated.Enabled() = false, want true")
	}

	dedupeKey := "enable:" + d.Id().String()
	rows := scheduledFor(t, db, dedupeKey)
	if len(rows) != 1 {
		t.Fatalf("scheduled %d rows for key %q, want 1", len(rows), dedupeKey)
	}
	if rows[0].Type != scheduling.WorkTypeTriggerEvaluation {
		t.Fatalf("type = %s, want %s", rows[0].Type, scheduling.WorkTypeTriggerEvaluation)
	}
	if rows[0].ExecuteAt.Before(before.Add(-1 * time.Minute)) {
		t.Fatalf("executeAt = %s, want ~now (%s)", rows[0].ExecuteAt, before)
	}
}

// A true->true, true->false, or false->false transition schedules nothing —
// disabling must never touch work, and re-enabling an already-enabled
// definition is a no-op PATCH, not a re-trigger.
func TestSetEnabledOtherTransitionsScheduleNothing(t *testing.T) {
	cases := []struct {
		name         string
		startEnabled bool
		setTo        bool
	}{
		{"true to true", true, true},
		{"true to false", true, false},
		{"false to false", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			l, _ := test.NewNullLogger()
			d := seedDefinition(t, db, tc.startEnabled)

			if _, err := SetEnabled(l, testCtx(t), db)(d.Id(), tc.setTo); err != nil {
				t.Fatalf("SetEnabled: %v", err)
			}

			dedupeKey := "enable:" + d.Id().String()
			rows := scheduledFor(t, db, dedupeKey)
			if len(rows) != 0 {
				t.Fatalf("scheduled %d rows for key %q, want 0", len(rows), dedupeKey)
			}
		})
	}
}

// Redelivery of the same false->true PATCH (e.g. a retried request) must not
// double-schedule — the dedupe key collapses it to the same row
// scheduling.Administrator.Schedule already dedupes on.
func TestSetEnabledRedeliveredEnableDoesNotDoubleSchedule(t *testing.T) {
	db := newTestDB(t)
	l, _ := test.NewNullLogger()
	d := seedDefinition(t, db, false)

	if _, err := SetEnabled(l, testCtx(t), db)(d.Id(), true); err != nil {
		t.Fatalf("first SetEnabled: %v", err)
	}
	// Second call: already enabled, so this is the true->true branch — but
	// exercised via the same entrypoint a redelivered PATCH would hit.
	if _, err := SetEnabled(l, testCtx(t), db)(d.Id(), true); err != nil {
		t.Fatalf("second SetEnabled: %v", err)
	}

	rows := scheduledFor(t, db, "enable:"+d.Id().String())
	if len(rows) != 1 {
		t.Fatalf("scheduled %d rows after redelivery, want 1", len(rows))
	}
}
