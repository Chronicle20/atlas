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

// captureSagaProcessor is an in-package test double for mapactionsaga.Processor
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

func testField() field.Model {
	return field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()
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
			wantErr:   `move_environment: parameter "name" is required`,
			wantSagas: 0,
		},
		{
			name:      "blank name errors",
			params:    map[string]string{"name": "", "value": "3"},
			wantErr:   `move_environment: parameter "name" is required`,
			wantSagas: 0,
		},
		{
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
			name:      "bad kind errors",
			params:    map[string]string{"name": "gate01", "value": "3", "kind": "GATE"},
			wantErr:   `move_environment: parameter "kind" value "GATE": unrecognized object kind [GATE]`,
			wantSagas: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			f := testField()
			op := newOperation(t, "move_environment", tt.params)

			err := e.ExecuteOperation(f, 1234, op)

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
			wantStepId := "move-environment-" + tt.params["name"]
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
			f := testField()
			op := newOperation(t, "reset_environment", tt.params)

			err := e.ExecuteOperation(f, 1234, op)
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
			if step.StepId != "reset-environment-910010000" {
				t.Errorf("StepId = %q, want %q", step.StepId, "reset-environment-910010000")
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

// TestExecuteSpawnMonsterCarriesInstance verifies that a spawn targeting the
// event's own map carries that map's instance, that a spawn aimed at a
// different map drops the instance, and that the team parameter now reaches
// the payload (FR-16 / OQ-3).
func TestExecuteSpawnMonsterCarriesInstance(t *testing.T) {
	instID := uuid.New()

	tests := []struct {
		name    string
		params  map[string]string
		checkFn func(t *testing.T, p saga.SpawnMonsterPayload)
	}{
		{
			name:   "same map carries instance",
			params: map[string]string{"monsterId": "100100"},
			checkFn: func(t *testing.T, p saga.SpawnMonsterPayload) {
				if p.Instance != instID {
					t.Errorf("Instance = %v, want %v", p.Instance, instID)
				}
				if p.MapId != 910010000 {
					t.Errorf("MapId = %v, want 910010000", p.MapId)
				}
			},
		},
		{
			name:   "cross map drops instance",
			params: map[string]string{"monsterId": "100100", "mapId": "910510202"},
			checkFn: func(t *testing.T, p saga.SpawnMonsterPayload) {
				if p.Instance != uuid.Nil {
					t.Errorf("Instance = %v, want uuid.Nil", p.Instance)
				}
				if p.MapId != 910510202 {
					t.Errorf("MapId = %v, want 910510202", p.MapId)
				}
			},
		},
		{
			name:   "team now populated",
			params: map[string]string{"monsterId": "100100", "team": "1"},
			checkFn: func(t *testing.T, p saga.SpawnMonsterPayload) {
				if p.Team != 1 {
					t.Errorf("Team = %v, want 1", p.Team)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			f := field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()
			op := newOperation(t, "spawn_monster", tt.params)

			err := e.ExecuteOperation(f, 1234, op)
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
			if step.Action != saga.SpawnMonster {
				t.Errorf("Action = %v, want SpawnMonster", step.Action)
			}
			if step.Status != saga.Pending {
				t.Errorf("Status = %v, want Pending", step.Status)
			}
			if step.StepId != "spawn-1234-100100" {
				t.Errorf("StepId = %q, want %q", step.StepId, "spawn-1234-100100")
			}
			if s.InitiatedBy != "map-action-spawn" {
				t.Errorf("InitiatedBy = %q, want %q", s.InitiatedBy, "map-action-spawn")
			}
			p, ok := step.Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("payload type = %T, want SpawnMonsterPayload", step.Payload)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, p)
			}
		})
	}
}

// TestExecuteDropMessageAcceptsTypeAlias verifies that drop_message accepts
// both the numeric type alias and the messageType key, with messageType
// winning when both are present (FR-13).
func TestExecuteDropMessageAcceptsTypeAlias(t *testing.T) {
	tests := []struct {
		name            string
		params          map[string]string
		wantMessageType string
	}{
		{
			name:            "numeric type alias",
			params:          map[string]string{"message": "hi", "type": "6"},
			wantMessageType: "BLUE_TEXT",
		},
		{
			name:            "messageType wins over type",
			params:          map[string]string{"message": "hi", "messageType": "NOTICE", "type": "POP_UP"},
			wantMessageType: "NOTICE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, d := newTestExecutor(t)
			f := testField()
			op := newOperation(t, "drop_message", tt.params)

			err := e.ExecuteOperation(f, 1234, op)
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
			if step.Action != saga.SendMessage {
				t.Errorf("Action = %v, want SendMessage", step.Action)
			}
			if step.StepId != "message-1234" {
				t.Errorf("StepId = %q, want %q", step.StepId, "message-1234")
			}
			if s.InitiatedBy != "map-action-message" {
				t.Errorf("InitiatedBy = %q, want %q", s.InitiatedBy, "map-action-message")
			}
			p, ok := step.Payload.(saga.SendMessagePayload)
			if !ok {
				t.Fatalf("payload type = %T, want SendMessagePayload", step.Payload)
			}
			if p.MessageType != tt.wantMessageType {
				t.Errorf("MessageType = %q, want %q", p.MessageType, tt.wantMessageType)
			}
		})
	}
}

func TestExecuteOperation_UnknownTypeStillWarns(t *testing.T) {
	e, d := newTestExecutor(t)
	f := testField()
	op := newOperation(t, "not_a_real_op", map[string]string{})

	err := e.ExecuteOperation(f, 1234, op)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(d.created) != 0 {
		t.Fatalf("created sagas = %d, want 0", len(d.created))
	}
}
