package occurrence

import (
	"atlas-events/event/definition"
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, MigrateTable, transition.MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// testTenantId is a fixed tenant shared by every occurrence package test that
// calls testCtx. Each test gets its own fresh in-memory DB (newTestDB), so a
// shared id across tests is safe; a fixed id (rather than a fresh uuid.New()
// per call) is what lets the REST harness's doGET issue an HTTP request
// against the same tenant a test wrote its fixtures under, without threading
// a tenant id through every helper call.
var testTenantId = uuid.New()

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(testTenantId)
}

// testDefinition builds a definition.Model with a deterministic id for
// theType, so two calls with the same type within a test represent the SAME
// definition — required for the concurrency-key uniqueness tests, which
// create occurrences from "separate" calls that must still collide.
func testDefinition(t *testing.T, theType string) definition.Model {
	t.Helper()
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(theType))
	m, err := definition.NewBuilder(theType, theType).SetId(id).SetEnabled(true).Build()
	if err != nil {
		t.Fatalf("testDefinition: %v", err)
	}
	return m
}

// design §686: both completion paths — Complete() and a terminal
// ApplyProgress — converge on ONE guarded transition. A terminal
// ApplyProgress against an occurrence Complete() already completed must NOT
// overwrite the first completion's reason/timestamp, and must surface
// ErrAlreadyCompleted (distinct from "no such occurrence") rather than
// silently winning the race.
func TestTerminalApplyProgressLosesToAnEarlierComplete(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}

	won, err := p.Complete(o.Id(), "MONSTERS_ELIMINATED", transition.TriggerTypeMonsterKilled, "u1")
	if err != nil || !won {
		t.Fatalf("Complete: won=%v err=%v, want true/nil", won, err)
	}

	_, err = p.ApplyProgress(o, registry.Progress{
		Stage:            "DONE",
		Terminal:         true,
		CompletionReason: "WINDOW_ELAPSED",
	}, transition.TriggerTypeScheduledWork, "poll-1")
	if !errors.Is(err, ErrAlreadyCompleted) {
		t.Fatalf("terminal ApplyProgress err = %v, want ErrAlreadyCompleted", err)
	}

	final, err := p.GetById(o.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if final.CompletionReason() != "MONSTERS_ELIMINATED" {
		t.Fatalf("completion reason = %q, want %q (must not be overwritten by the losing racer)",
			final.CompletionReason(), "MONSTERS_ELIMINATED")
	}

	var trans int64
	db.Model(&transition.Entity{}).Where("occurrence_id = ?", o.Id()).Count(&trans)
	// OCCURRENCE_CREATED + the winning Complete() transition. The losing
	// ApplyProgress must write no transition of its own.
	if trans != 2 {
		t.Fatalf("expected 2 transition rows, got %d", trans)
	}
}

// The non-terminal path is unaffected by the terminal guard: a progress
// update that does not complete the occurrence must still succeed.
func TestNonTerminalApplyProgressStillSucceeds(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)
	o, err := p.CreateFromSeed(testDefinition(t, "CRIMSON_BALROG"),
		registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "k"}, "w")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}

	updated, err := p.ApplyProgress(o, registry.Progress{Stage: "FLEEING"}, transition.TriggerTypeScheduledWork, "poll-1")
	if err != nil {
		t.Fatalf("ApplyProgress: %v", err)
	}
	if updated.Stage() != "FLEEING" || updated.State() != StateActive {
		t.Fatalf("updated = %s/%s, want FLEEING/ACTIVE", updated.State(), updated.Stage())
	}
}
