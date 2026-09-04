package script

import (
	"context"
	"strings"
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
			// FR-15: error text now uses the shared ops.ParamError format;
			// pass/fail outcome is unchanged.
			name:      "missing name errors",
			params:    map[string]string{"value": "3"},
			wantErr:   `move_environment: parameter "name" is required`,
			wantSagas: 0,
		},
		{
			// FR-15: error text now uses the shared ops.ParamError format;
			// pass/fail outcome is unchanged.
			name:      "blank name errors",
			params:    map[string]string{"name": "", "value": "3"},
			wantErr:   `move_environment: parameter "name" is required`,
			wantSagas: 0,
		},
		{
			// FR-15: error text now uses the shared ops.ParamError format;
			// pass/fail outcome is unchanged.
			name:      "missing value errors",
			params:    map[string]string{"name": "gate01"},
			wantErr:   `move_environment: parameter "value" is required`,
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
			// FR-15: error text now uses the shared ops.ParamError format;
			// pass/fail outcome is unchanged.
			name:      "bad kind errors",
			params:    map[string]string{"name": "gate01", "value": "3", "kind": "GATE"},
			wantErr:   `move_environment: parameter "kind" value "GATE": unrecognized object kind [GATE]`,
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

// TestExecuteSpawnMonsterRejectsBadNumerics verifies that spawn_monster's
// numeric parameters (x, y, count) are hard errors when unparsable — the
// previous local parsing silently kept the default (FR-15) — and that valid
// defaults still use the reactor's position.
func TestExecuteSpawnMonsterRejectsBadNumerics(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]string
		wantErrHas []string
		checkFn    func(t *testing.T, p saga.SpawnMonsterPayload)
	}{
		{
			name:       "bad x errors",
			params:     map[string]string{"monsterId": "100100", "x": "abc"},
			wantErrHas: []string{"spawn_monster", `"x"`, `"abc"`},
		},
		{
			name:       "bad y errors",
			params:     map[string]string{"monsterId": "100100", "y": "abc"},
			wantErrHas: []string{"spawn_monster", `"y"`, `"abc"`},
		},
		{
			name:       "bad count errors",
			params:     map[string]string{"monsterId": "100100", "count": "abc"},
			wantErrHas: []string{"spawn_monster", `"count"`, `"abc"`},
		},
		{
			name:   "good defaults still use reactor position",
			params: map[string]string{"monsterId": "100100"},
			checkFn: func(t *testing.T, p saga.SpawnMonsterPayload) {
				rc := testReactorContext()
				if p.X != rc.X {
					t.Errorf("X = %v, want %v", p.X, rc.X)
				}
				if p.Y != rc.Y {
					t.Errorf("Y = %v, want %v", p.Y, rc.Y)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			rc := testReactorContext()
			op := newOperation(t, "spawn_monster", tt.params)

			err := e.ExecuteOperation(rc, 1234, op)

			if tt.wantErrHas != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				for _, substr := range tt.wantErrHas {
					if !strings.Contains(err.Error(), substr) {
						t.Errorf("error %q does not contain %q", err.Error(), substr)
					}
				}
				if len(d.created) != 0 {
					t.Fatalf("created sagas = %d, want 0", len(d.created))
				}
				return
			}

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
			p, ok := s.Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("payload type = %T, want SpawnMonsterPayload", s.Steps[0].Payload)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, p)
			}
		})
	}
}

// TestExecuteDropMessageAcceptsMessageTypeAlias verifies that drop_message
// accepts messageType (FR-13; reactor-actions previously read only `type`)
// and that the numeric `type` alias still maps to its string form.
func TestExecuteDropMessageAcceptsMessageTypeAlias(t *testing.T) {
	tests := []struct {
		name            string
		params          map[string]string
		wantMessageType string
	}{
		{
			name:            "messageType key",
			params:          map[string]string{"message": "hi", "messageType": "NOTICE"},
			wantMessageType: "NOTICE",
		},
		{
			name:            "numeric type alias",
			params:          map[string]string{"message": "hi", "type": "5"},
			wantMessageType: "PINK_TEXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			rc := testReactorContext()
			op := newOperation(t, "drop_message", tt.params)

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
			p, ok := s.Steps[0].Payload.(saga.SendMessagePayload)
			if !ok {
				t.Fatalf("payload type = %T, want SendMessagePayload", s.Steps[0].Payload)
			}
			if p.MessageType != tt.wantMessageType {
				t.Errorf("MessageType = %q, want %q", p.MessageType, tt.wantMessageType)
			}
		})
	}
}
