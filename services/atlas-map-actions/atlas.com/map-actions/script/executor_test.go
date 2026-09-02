package script

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
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

func TestExecuteOperationUnknownType(t *testing.T) {
	tests := []struct {
		name        string
		execute     func(e *OperationExecutor, f field.Model) error
		wantErr     string
		wantCreated int
	}{
		{
			name: "ExecuteOperation errors and creates nothing",
			execute: func(e *OperationExecutor, f field.Model) error {
				op, err := operation.NewBuilder().SetType("play_sound").Build()
				if err != nil {
					t.Fatalf("operation.NewBuilder().Build(): %v", err)
				}
				return e.ExecuteOperation(f, 1, op)
			},
			wantErr:     "unknown operation type [play_sound]",
			wantCreated: 0,
		},
		{
			name: "ExecuteOperations aborts after unknown operation",
			execute: func(e *OperationExecutor, f field.Model) error {
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
				return e.ExecuteOperations(f, 1, []operation.Model{fieldEffect, playSound, unlockUi})
			},
			wantErr:     "unknown operation type [play_sound]",
			wantCreated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()

			err := tt.execute(e, f)
			if err == nil {
				t.Fatalf("error = nil, want non-nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}

			if len(rec.created) != tt.wantCreated {
				t.Errorf("len(rec.created) = %d, want %d", len(rec.created), tt.wantCreated)
			}
		})
	}
}

func TestExecuteSpawnMonsterCarriesFieldInstance(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "carries field instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().
				SetType("spawn_monster").
				SetParams(map[string]string{"monsterId": "9100013", "x": "82", "y": "200"}).
				Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			if err := e.ExecuteOperation(f, 1, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v, want nil", err)
			}

			if len(rec.created) != 1 {
				t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
			}
			payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
			}
			if payload.Instance != inst {
				t.Errorf("payload.Instance = %v, want %v", payload.Instance, inst)
			}
			if payload.MapId != _map.Id(926000000) {
				t.Errorf("payload.MapId = %v, want %v", payload.MapId, _map.Id(926000000))
			}
			if payload.MonsterId != 9100013 {
				t.Errorf("payload.MonsterId = %d, want %d", payload.MonsterId, 9100013)
			}
			if payload.X != 82 {
				t.Errorf("payload.X = %d, want %d", payload.X, 82)
			}
			if payload.Y != 200 {
				t.Errorf("payload.Y = %d, want %d", payload.Y, 200)
			}
			if payload.Count != 1 {
				t.Errorf("payload.Count = %d, want %d", payload.Count, 1)
			}
		})
	}
}

func TestExecuteSpawnMonsterSpawnIfAbsent(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantAbsent    bool
		wantErrSubstr string
	}{
		{name: "absent", params: map[string]string{"monsterId": "9100013"}, wantAbsent: false},
		{name: "true", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "true"}, wantAbsent: true},
		{name: "false", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "false"}, wantAbsent: false},
		{name: "invalid", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "yes"}, wantErrSubstr: "invalid spawnIfAbsent [yes]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().SetType("spawn_monster").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExecuteOperation() error = %v, want nil", err)
			}

			payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
			}
			if payload.SpawnIfAbsent != tt.wantAbsent {
				t.Errorf("payload.SpawnIfAbsent = %t, want %t", payload.SpawnIfAbsent, tt.wantAbsent)
			}
		})
	}
}

func TestExecuteSpawnMonsterMonsterIdsPicksFromSet(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "picks from set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().
				SetType("spawn_monster").
				SetParams(map[string]string{"monsterIds": "3300005,3300006,3300007", "x": "-28", "y": "-67"}).
				Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			seen := map[uint32]bool{}
			for i := 0; i < 200; i++ {
				e, rec := newTestOperationExecutor()
				if err := e.ExecuteOperation(f, 1, op); err != nil {
					t.Fatalf("ExecuteOperation() error = %v, want nil", err)
				}
				payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
				if !ok {
					t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
				}
				if payload.MonsterId != 3300005 && payload.MonsterId != 3300006 && payload.MonsterId != 3300007 {
					t.Fatalf("payload.MonsterId = %d, want one of 3300005, 3300006, 3300007", payload.MonsterId)
				}
				seen[payload.MonsterId] = true
				if payload.X != -28 {
					t.Errorf("payload.X = %d, want %d", payload.X, -28)
				}
				if payload.Y != -67 {
					t.Errorf("payload.Y = %d, want %d", payload.Y, -67)
				}
			}

			for _, id := range []uint32{3300005, 3300006, 3300007} {
				if !seen[id] {
					t.Errorf("monsterId %d never chosen across 200 runs", id)
				}
			}
		})
	}
}

func TestExecuteSpawnMonsterIdParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "neither", params: map[string]string{"x": "0"}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "both", params: map[string]string{"monsterId": "1", "monsterIds": "2,3"}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "empty monsterIds", params: map[string]string{"monsterIds": ""}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "non-numeric entry", params: map[string]string{"monsterIds": "3300005,abc"}, wantErrSubstr: "invalid monsterIds entry [abc]"},
		{name: "non-numeric monsterId", params: map[string]string{"monsterId": "abc"}, wantErrSubstr: "invalid monsterId [abc]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).Build()

			op, err := operation.NewBuilder().SetType("spawn_monster").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}
