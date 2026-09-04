package ops

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestStartQuest(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		defaults    QuestDefaults
		wantErr     string
		wantParam   *ParamError
		wantPayload saga.StartQuestPayload
	}{
		{
			name:     "param questId",
			params:   map[string]string{"questId": "2000"},
			defaults: QuestDefaults{},
			wantPayload: saga.StartQuestPayload{
				CharacterId: 7,
				WorldId:     0,
				QuestId:     2000,
				NpcId:       0,
			},
		},
		{
			name:     "param npcId",
			params:   map[string]string{"questId": "2000", "npcId": "1063017"},
			defaults: QuestDefaults{},
			wantPayload: saga.StartQuestPayload{
				CharacterId: 7,
				WorldId:     0,
				QuestId:     2000,
				NpcId:       1063017,
			},
		},
		{
			name:     "default questId",
			params:   map[string]string{},
			defaults: QuestDefaults{QuestId: 2000, NpcId: 9010000},
			wantPayload: saga.StartQuestPayload{
				CharacterId: 7,
				WorldId:     0,
				QuestId:     2000,
				NpcId:       9010000,
			},
		},
		{
			name:     "param wins over default",
			params:   map[string]string{"questId": "3000"},
			defaults: QuestDefaults{QuestId: 2000, NpcId: 9010000},
			wantPayload: saga.StartQuestPayload{
				CharacterId: 7,
				WorldId:     0,
				QuestId:     3000,
				NpcId:       9010000,
			},
		},
		{
			name:     "no questId anywhere",
			params:   map[string]string{},
			defaults: QuestDefaults{NpcId: 9010000},
			wantErr:  `start_quest: parameter "questId" is required`,
		},
		{
			name:      "bad questId",
			params:    map[string]string{"questId": "abc"},
			defaults:  QuestDefaults{},
			wantParam: &ParamError{Op: "start_quest", Param: "questId", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := StartQuest(tt.params, DirectResolver{}, plain, 7, tt.defaults)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
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
			if step.Action() != saga.StartQuest {
				t.Fatalf("got action %v, want %v", step.Action(), saga.StartQuest)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.StartQuestPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload.CharacterId != tt.wantPayload.CharacterId ||
				payload.WorldId != tt.wantPayload.WorldId ||
				payload.QuestId != tt.wantPayload.QuestId ||
				payload.NpcId != tt.wantPayload.NpcId {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestStageClearAttemptPq(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()
	instanceId := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name        string
		characterId uint32
		instanceId  uuid.UUID
		wantErr     string
		wantPayload saga.StageClearAttemptPqPayload
	}{
		{
			name:        "reactor path",
			characterId: 7,
			instanceId:  instanceId,
			wantPayload: saga.StageClearAttemptPqPayload{InstanceId: instanceId, CharacterId: 0},
		},
		{
			name:        "npc path",
			characterId: 7,
			instanceId:  uuid.Nil,
			wantPayload: saga.StageClearAttemptPqPayload{InstanceId: uuid.Nil, CharacterId: 7},
		},
		{
			name:        "neither set",
			characterId: 0,
			instanceId:  uuid.Nil,
			wantErr:     "stage_clear_attempt_pq: exactly one of instanceId or characterId must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := StageClearAttemptPq(plain, tt.characterId, tt.instanceId)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if step.Action() != saga.StageClearAttemptPq {
				t.Fatalf("got action %v, want %v", step.Action(), saga.StageClearAttemptPq)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.StageClearAttemptPqPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}
