package script

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// recordingSagaProcessor is a hand-rolled test recorder implementing
// mapactionsaga.Processor; it captures every saga passed to Create so tests
// can assert on how many operations actually ran.
type recordingSagaProcessor struct {
	created []saga.Saga
}

func (r *recordingSagaProcessor) Create(s saga.Saga) error {
	r.created = append(r.created, s)
	return nil
}

func newTestOperationExecutor() (*OperationExecutor, *recordingSagaProcessor) {
	l, _ := test.NewNullLogger()
	rec := &recordingSagaProcessor{}
	e := NewOperationExecutor(l, context.Background())
	e.sagaP = rec
	return e, rec
}

func TestExecuteOperationUnknownTypeErrors(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()
	op, err := operation.NewBuilder().SetType("play_sound").Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	err = e.ExecuteOperation(f, 1, op)
	if err == nil {
		t.Fatalf("ExecuteOperation() error = nil, want non-nil")
	}
	if err.Error() != "unknown operation type [play_sound]" {
		t.Errorf("ExecuteOperation() error = %q, want %q", err.Error(), "unknown operation type [play_sound]")
	}

	if len(rec.created) != 0 {
		t.Errorf("len(rec.created) = %d, want 0", len(rec.created))
	}
}

func TestExecuteOperationsAbortsAfterUnknownOperation(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()

	fieldEffect, err := operation.NewBuilder().
		SetType("field_effect").
		SetParams(map[string]string{"path": "maplemap/enter/1000000"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}
	playSound, err := operation.NewBuilder().SetType("play_sound").Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}
	unlockUi, err := operation.NewBuilder().SetType("unlock_ui").Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	err = e.ExecuteOperations(f, 1, []operation.Model{fieldEffect, playSound, unlockUi})
	if err == nil {
		t.Fatalf("ExecuteOperations() error = nil, want non-nil")
	}
	if err.Error() != "unknown operation type [play_sound]" {
		t.Errorf("ExecuteOperations() error = %q, want %q", err.Error(), "unknown operation type [play_sound]")
	}

	if len(rec.created) != 1 {
		t.Errorf("len(rec.created) = %d, want 1", len(rec.created))
	}
}
