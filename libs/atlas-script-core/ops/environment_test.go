package ops

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestMoveEnvironment(t *testing.T) {
	instID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	instanced := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).Build()

	tests := []struct {
		name         string
		params       map[string]string
		wantErr      string
		wantParam    *ParamError
		wantRangeErr string
		wantPayload  saga.MoveEnvironmentPayload
	}{
		{
			name:    "missing name",
			params:  map[string]string{"value": "3"},
			wantErr: `move_environment: parameter "name" is required`,
		},
		{
			name:    "blank name",
			params:  map[string]string{"name": "  ", "value": "3"},
			wantErr: `move_environment: parameter "name" is required`,
		},
		{
			name:    "missing value",
			params:  map[string]string{"name": "gate01"},
			wantErr: `move_environment: parameter "value" is required`,
		},
		{
			name:      "bad value",
			params:    map[string]string{"name": "gate01", "value": "abc"},
			wantParam: &ParamError{Op: "move_environment", Param: "value", Value: "abc"},
		},
		{
			name:   "defaults kind",
			params: map[string]string{"name": "gate01", "value": "3"},
			wantPayload: saga.MoveEnvironmentPayload{
				WorldId:   0,
				ChannelId: 1,
				MapId:     910010000,
				Instance:  instID,
				Kind:      field.ObjectKindEnvironment,
				Name:      "gate01",
				State:     3,
			},
		},
		{
			name:   "explicit ENVIRONMENT",
			params: map[string]string{"name": "g", "value": "1", "kind": "ENVIRONMENT"},
			wantPayload: saga.MoveEnvironmentPayload{
				WorldId:   0,
				ChannelId: 1,
				MapId:     910010000,
				Instance:  instID,
				Kind:      field.ObjectKindEnvironment,
				Name:      "g",
				State:     1,
			},
		},
		{
			name:   "explicit OBSTACLE",
			params: map[string]string{"name": "g", "value": "1", "kind": "OBSTACLE"},
			wantPayload: saga.MoveEnvironmentPayload{
				WorldId:   0,
				ChannelId: 1,
				MapId:     910010000,
				Instance:  instID,
				Kind:      field.ObjectKindObstacle,
				Name:      "g",
				State:     1,
			},
		},
		{
			name:      "bad kind",
			params:    map[string]string{"name": "g", "value": "1", "kind": "BOGUS"},
			wantParam: &ParamError{Op: "move_environment", Param: "kind", Value: "BOGUS"},
		},
		{
			name:         "negative value out of range",
			params:       map[string]string{"name": "g", "value": "-1"},
			wantRangeErr: "out of range for uint32",
		},
		{
			name:   "name sent raw, not trimmed",
			params: map[string]string{"name": " gate01 ", "value": "3"},
			wantPayload: saga.MoveEnvironmentPayload{
				WorldId:   0,
				ChannelId: 1,
				MapId:     910010000,
				Instance:  instID,
				Kind:      field.ObjectKindEnvironment,
				Name:      " gate01 ",
				State:     3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := MoveEnvironment(tt.params, DirectResolver{}, instanced, 7)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if tt.wantRangeErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantRangeErr)
				}
				if !strings.Contains(err.Error(), tt.wantRangeErr) {
					t.Fatalf("got error %q, want it to contain %q", err.Error(), tt.wantRangeErr)
				}
				return
			}
			if tt.wantParam != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var pe *ParamError
				if !errors.As(err, &pe) {
					t.Fatalf("expected *ParamError, got %T: %v", err, err)
				}
				if pe.Op != tt.wantParam.Op || pe.Param != tt.wantParam.Param || pe.Value != tt.wantParam.Value {
					t.Fatalf("got ParamError{Op:%q,Param:%q,Value:%q}, want {Op:%q,Param:%q,Value:%q}",
						pe.Op, pe.Param, pe.Value, tt.wantParam.Op, tt.wantParam.Param, tt.wantParam.Value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.MoveEnvironment {
				t.Fatalf("got action %v, want %v", step.Action(), saga.MoveEnvironment)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.MoveEnvironmentPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestResetEnvironment(t *testing.T) {
	instID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	instanced := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).Build()

	step, err := ResetEnvironment(map[string]string{}, DirectResolver{}, instanced, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step.Action() != saga.ResetEnvironment {
		t.Fatalf("got action %v, want %v", step.Action(), saga.ResetEnvironment)
	}
	if step.Status() != saga.Pending {
		t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
	}
	payload, err := PayloadOf[saga.ResetEnvironmentPayload](step)
	if err != nil {
		t.Fatalf("unexpected payload type error: %v", err)
	}
	want := saga.ResetEnvironmentPayload{
		WorldId:   0,
		ChannelId: 1,
		MapId:     910010000,
		Instance:  instID,
	}
	if payload != want {
		t.Fatalf("got payload %+v, want %+v", payload, want)
	}
}
