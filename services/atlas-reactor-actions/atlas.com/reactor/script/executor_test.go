package script

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// captureSagaProcessor is an in-package test double for reactorsaga.Processor
// that records the sagas it was handed.
type captureSagaProcessor struct {
	CreateFunc func(s saga.Saga) error
	created    []saga.Saga
}

func (c *captureSagaProcessor) Create(s saga.Saga) error {
	c.created = append(c.created, s)
	if c.CreateFunc != nil {
		return c.CreateFunc(s)
	}
	return nil
}

func newTestExecutor(t *testing.T) (*OperationExecutor, *captureSagaProcessor) {
	t.Helper()
	l, _ := test.NewNullLogger()
	e := NewOperationExecutor(l, context.Background())
	d := &captureSagaProcessor{}
	e.sagaP = d
	return e, d
}

func newOperation(t *testing.T, opType string, params map[string]string) operation.Model {
	t.Helper()
	op, err := operation.NewBuilder().SetType(opType).SetParams(params).Build()
	if err != nil {
		t.Fatalf("failed to build operation: %v", err)
	}
	return op
}

func testReactorContext() ReactorContext {
	return ReactorContext{
		Field:          field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build(),
		ReactorId:      1,
		Classification: "2001000",
		ReactorName:    "gate",
	}
}

func TestExecuteMoveEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		wantErr   string
		wantSagas int
		checkFn   func(t *testing.T, p saga.MoveEnvironmentPayload)
	}{
		{
			name:      "creates saga step",
			params:    map[string]string{"name": "gate01", "value": "3"},
			wantSagas: 1,
			checkFn: func(t *testing.T, p saga.MoveEnvironmentPayload) {
				if p.WorldId != 0 {
					t.Errorf("WorldId = %v, want 0", p.WorldId)
				}
				if p.ChannelId != 1 {
					t.Errorf("ChannelId = %v, want 1", p.ChannelId)
				}
				if p.MapId != 910010000 {
					t.Errorf("MapId = %v, want 910010000", p.MapId)
				}
				if p.Instance != uuid.Nil {
					t.Errorf("Instance = %v, want uuid.Nil", p.Instance)
				}
				if p.Kind != field.ObjectKindEnvironment {
					t.Errorf("Kind = %v, want ENVIRONMENT", p.Kind)
				}
				if p.Name != "gate01" {
					t.Errorf("Name = %v, want gate01", p.Name)
				}
				if p.State != uint32(3) {
					t.Errorf("State = %v, want 3", p.State)
				}
			},
		},
		{
			name:      "kind obstacle reaches payload",
			params:    map[string]string{"name": "obs3", "value": "1", "kind": "OBSTACLE"},
			wantSagas: 1,
			checkFn: func(t *testing.T, p saga.MoveEnvironmentPayload) {
				if p.Kind != field.ObjectKindObstacle {
					t.Errorf("Kind = %v, want OBSTACLE", p.Kind)
				}
			},
		},
		{
			name:      "omitted kind defaults environment",
			params:    map[string]string{"name": "gate01", "value": "0"},
			wantSagas: 1,
			checkFn: func(t *testing.T, p saga.MoveEnvironmentPayload) {
				if p.Kind != field.ObjectKindEnvironment {
					t.Errorf("Kind = %v, want ENVIRONMENT", p.Kind)
				}
				if p.State != uint32(0) {
					t.Errorf("State = %v, want 0", p.State)
				}
			},
		},
		{
			name:      "missing name errors",
			params:    map[string]string{"value": "3"},
			wantErr:   "move_environment operation missing name parameter",
			wantSagas: 0,
		},
		{
			name:      "blank name errors",
			params:    map[string]string{"name": "", "value": "3"},
			wantErr:   "move_environment operation missing name parameter",
			wantSagas: 0,
		},
		{
			name:      "missing value errors",
			params:    map[string]string{"name": "gate01"},
			wantErr:   "move_environment operation missing value parameter",
			wantSagas: 0,
		},
		{
			name:      "non-numeric value errors",
			params:    map[string]string{"name": "gate01", "value": "up"},
			wantSagas: 0,
		},
		{
			name:      "negative value errors",
			params:    map[string]string{"name": "gate01", "value": "-1"},
			wantSagas: 0,
		},
		{
			name:      "overflow value errors",
			params:    map[string]string{"name": "gate01", "value": "4294967296"},
			wantSagas: 0,
		},
		{
			name:      "max uint32 accepted",
			params:    map[string]string{"name": "gate01", "value": "4294967295"},
			wantSagas: 1,
			checkFn: func(t *testing.T, p saga.MoveEnvironmentPayload) {
				if p.State != uint32(4294967295) {
					t.Errorf("State = %v, want 4294967295", p.State)
				}
			},
		},
		{
			name:      "bad kind errors",
			params:    map[string]string{"name": "gate01", "value": "3", "kind": "GATE"},
			wantErr:   "unrecognized object kind [GATE]",
			wantSagas: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			rc := testReactorContext()
			op := newOperation(t, "move_environment", tt.params)

			err := e.ExecuteOperation(rc, 1234, op)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
				}
			} else if tt.wantSagas > 0 {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			}

			if len(d.created) != tt.wantSagas {
				t.Fatalf("created sagas = %d, want %d", len(d.created), tt.wantSagas)
			}

			if tt.wantSagas == 0 {
				return
			}

			s := d.created[0]
			if len(s.Steps) != 1 {
				t.Fatalf("steps = %d, want 1", len(s.Steps))
			}
			step := s.Steps[0]
			if step.Action != saga.MoveEnvironment {
				t.Errorf("Action = %v, want MoveEnvironment", step.Action)
			}
			if step.Status != saga.Pending {
				t.Errorf("Status = %v, want Pending", step.Status)
			}
			wantStepId := "move-environment-2001000-" + tt.params["name"]
			if step.StepId != wantStepId {
				t.Errorf("StepId = %q, want %q", step.StepId, wantStepId)
			}
			p, ok := step.Payload.(saga.MoveEnvironmentPayload)
			if !ok {
				t.Fatalf("payload type = %T, want MoveEnvironmentPayload", step.Payload)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, p)
			}
		})
	}
}

func TestExecuteResetEnvironment(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
	}{
		{name: "creates saga step", params: map[string]string{}},
		{name: "extra params ignored", params: map[string]string{"name": "ignored"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			rc := testReactorContext()
			op := newOperation(t, "reset_environment", tt.params)

			err := e.ExecuteOperation(rc, 1234, op)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if len(d.created) != 1 {
				t.Fatalf("created sagas = %d, want 1", len(d.created))
			}

			s := d.created[0]
			if len(s.Steps) != 1 {
				t.Fatalf("steps = %d, want 1", len(s.Steps))
			}
			step := s.Steps[0]
			if step.Action != saga.ResetEnvironment {
				t.Errorf("Action = %v, want ResetEnvironment", step.Action)
			}
			if step.Status != saga.Pending {
				t.Errorf("Status = %v, want Pending", step.Status)
			}
			if step.StepId != "reset-environment-2001000" {
				t.Errorf("StepId = %q, want %q", step.StepId, "reset-environment-2001000")
			}
			p, ok := step.Payload.(saga.ResetEnvironmentPayload)
			if !ok {
				t.Fatalf("payload type = %T, want ResetEnvironmentPayload", step.Payload)
			}
			if p.WorldId != 0 {
				t.Errorf("WorldId = %v, want 0", p.WorldId)
			}
			if p.ChannelId != 1 {
				t.Errorf("ChannelId = %v, want 1", p.ChannelId)
			}
			if p.MapId != 910010000 {
				t.Errorf("MapId = %v, want 910010000", p.MapId)
			}
			if p.Instance != uuid.Nil {
				t.Errorf("Instance = %v, want uuid.Nil", p.Instance)
			}
		})
	}
}

// TestExecuteMoveEnvironment_NoLongerReturnsNilStub is a regression guard
// against the historical bare `return nil` stub: a valid move_environment
// call must create exactly one saga.
func TestExecuteMoveEnvironment_NoLongerReturnsNilStub(t *testing.T) {
	e, d := newTestExecutor(t)
	rc := testReactorContext()
	op := newOperation(t, "move_environment", map[string]string{"name": "gate01", "value": "3"})

	if err := e.ExecuteOperation(rc, 1234, op); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(d.created) != 1 {
		t.Fatalf("created sagas = %d, want exactly 1", len(d.created))
	}
}

func newTestReactorContext() ReactorContext {
	return ReactorContext{
		Field:          field.NewBuilder(1, 2, 100000000).Build(),
		ReactorId:      1,
		Classification: "test_reactor",
		ReactorName:    "test_reactor",
		X:              100,
		Y:              200,
	}
}

func TestExecuteDropItems_MesoParams(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]string
		wantMeso       bool
		wantMesoChance uint32
		wantMesoMin    uint32
		wantMesoMax    uint32
		wantMinItems   uint32
		wantDropType   string
	}{
		{
			name: "all new params",
			params: map[string]string{
				"meso":       "true",
				"mesoChance": "2",
				"mesoMin":    "8",
				"mesoMax":    "15",
				"minItems":   "1",
			},
			wantMeso:       true,
			wantMesoChance: 2,
			wantMesoMin:    8,
			wantMesoMax:    15,
			wantMinItems:   1,
			wantDropType:   "drop",
		},
		{
			name:           "no params",
			params:         map[string]string{},
			wantMeso:       false,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "legacy minMeso ignored",
			params: map[string]string{
				"meso":    "true",
				"minMeso": "2",
				"maxMeso": "8",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "legacy mixed with new",
			params: map[string]string{
				"meso":    "true",
				"mesoMin": "8",
				"minMeso": "999",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    8,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
		{
			name: "unparseable value falls back to default",
			params: map[string]string{
				"meso":       "true",
				"mesoChance": "abc",
			},
			wantMeso:       true,
			wantMesoChance: 1,
			wantMesoMin:    1,
			wantMesoMax:    1,
			wantMinItems:   0,
			wantDropType:   "drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, err := operation.NewBuilder().SetType("drop_items").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("failed to build operation: %v", err)
			}

			executor, fake := newTestExecutor(t)
			rc := newTestReactorContext()

			if err := executor.ExecuteOperation(rc, 12345, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v", err)
			}

			if len(fake.created) != 1 {
				t.Fatalf("expected 1 saga created, got %d", len(fake.created))
			}

			s := fake.created[0]
			if len(s.Steps) != 1 {
				t.Fatalf("expected 1 step, got %d", len(s.Steps))
			}

			payload, ok := s.Steps[0].Payload.(saga.SpawnReactorDropsPayload)
			if !ok {
				t.Fatalf("step payload is not a SpawnReactorDropsPayload: %T", s.Steps[0].Payload)
			}

			if payload.Meso != tt.wantMeso {
				t.Errorf("Meso = %v, want %v", payload.Meso, tt.wantMeso)
			}
			if payload.MesoChance != tt.wantMesoChance {
				t.Errorf("MesoChance = %d, want %d", payload.MesoChance, tt.wantMesoChance)
			}
			if payload.MesoMin != tt.wantMesoMin {
				t.Errorf("MesoMin = %d, want %d", payload.MesoMin, tt.wantMesoMin)
			}
			if payload.MesoMax != tt.wantMesoMax {
				t.Errorf("MesoMax = %d, want %d", payload.MesoMax, tt.wantMesoMax)
			}
			if payload.MinItems != tt.wantMinItems {
				t.Errorf("MinItems = %d, want %d", payload.MinItems, tt.wantMinItems)
			}
			if payload.DropType != tt.wantDropType {
				t.Errorf("DropType = %q, want %q", payload.DropType, tt.wantDropType)
			}
		})
	}
}

// TestExecuteSprayItems covers executeSprayItems's two scenarios:
//
//   - "SetsDropType": a full set of meso params produces a
//     SpawnReactorDropsPayload with DropType "spray" and the meso fields
//     carried through unchanged.
//   - "NoParams" pins the invariant that executeSprayItems injects
//     params["dropType"] = "spray" into whatever map op.Params() returns.
//     operation.NewBuilder always seeds a non-nil params map, and
//     convertJsonOperation only calls SetParams when the seed JSON has a
//     non-nil params object, so a spray_items operation decoded from a seed
//     with no params key must still execute without panicking.
func TestExecuteSprayItems(t *testing.T) {
	tests := []struct {
		name             string
		params           map[string]string
		wantDropType     string
		checkMesoDetails bool
		wantMesoMin      uint32
		wantMesoMax      uint32
		wantMinItems     uint32
	}{
		{
			name: "SetsDropType",
			params: map[string]string{
				"meso":       "true",
				"mesoChance": "1",
				"mesoMin":    "50",
				"mesoMax":    "100",
				"minItems":   "15",
			},
			wantDropType:     "spray",
			checkMesoDetails: true,
			wantMesoMin:      50,
			wantMesoMax:      100,
			wantMinItems:     15,
		},
		{
			name:         "NoParams",
			params:       nil,
			wantDropType: "spray",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := operation.NewBuilder().SetType("spray_items")
			if tt.params != nil {
				builder = builder.SetParams(tt.params)
			}
			op, err := builder.Build()
			if err != nil {
				t.Fatalf("failed to build operation: %v", err)
			}

			executor, fake := newTestExecutor(t)
			rc := newTestReactorContext()

			if err := executor.ExecuteOperation(rc, 12345, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v", err)
			}

			if len(fake.created) != 1 {
				t.Fatalf("expected 1 saga created, got %d", len(fake.created))
			}

			s := fake.created[0]
			if len(s.Steps) != 1 {
				t.Fatalf("expected 1 step, got %d", len(s.Steps))
			}

			payload, ok := s.Steps[0].Payload.(saga.SpawnReactorDropsPayload)
			if !ok {
				t.Fatalf("step payload is not a SpawnReactorDropsPayload: %T", s.Steps[0].Payload)
			}

			if payload.DropType != tt.wantDropType {
				t.Errorf("DropType = %q, want %q", payload.DropType, tt.wantDropType)
			}

			if tt.checkMesoDetails {
				if payload.MesoMin != tt.wantMesoMin {
					t.Errorf("MesoMin = %d, want %d", payload.MesoMin, tt.wantMesoMin)
				}
				if payload.MesoMax != tt.wantMesoMax {
					t.Errorf("MesoMax = %d, want %d", payload.MesoMax, tt.wantMesoMax)
				}
				if payload.MinItems != tt.wantMinItems {
					t.Errorf("MinItems = %d, want %d", payload.MinItems, tt.wantMinItems)
				}
			}
		})
	}
}
