package ops

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestSpawnMonster(t *testing.T) {
	instID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()
	instanced := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).SetInstance(instID).Build()).SetPosition(-120, 33).Build()

	tests := []struct {
		name        string
		target      Target
		params      map[string]string
		wantErr     string
		wantParam   *ParamError
		wantPayload saga.SpawnMonsterPayload
	}{
		{
			name:    "missing monsterId",
			target:  plain,
			params:  map[string]string{},
			wantErr: `spawn_monster: parameter "monsterId" is required`,
		},
		{
			name:      "bad monsterId",
			target:    plain,
			params:    map[string]string{"monsterId": "abc"},
			wantParam: &ParamError{Op: "spawn_monster", Param: "monsterId", Value: "abc"},
		},
		{
			name:   "defaults, no position",
			target: plain,
			params: map[string]string{"monsterId": "100100"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    uuid.Nil,
				MonsterId:   100100,
				X:           0,
				Y:           0,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:   "position default from target",
			target: instanced,
			params: map[string]string{"monsterId": "100100"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    instID,
				MonsterId:   100100,
				X:           -120,
				Y:           33,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:   "explicit x/y override target",
			target: instanced,
			params: map[string]string{"monsterId": "100100", "x": "5", "y": "6"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    instID,
				MonsterId:   100100,
				X:           5,
				Y:           6,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:      "FR-15 bad x hard-errors",
			target:    instanced,
			params:    map[string]string{"monsterId": "100100", "x": "abc"},
			wantParam: &ParamError{Op: "spawn_monster", Param: "x", Value: "abc"},
		},
		{
			name:      "FR-15 bad y hard-errors",
			target:    instanced,
			params:    map[string]string{"monsterId": "100100", "y": "abc"},
			wantParam: &ParamError{Op: "spawn_monster", Param: "y", Value: "abc"},
		},
		{
			name:      "FR-15 bad count hard-errors",
			target:    plain,
			params:    map[string]string{"monsterId": "100100", "count": "abc"},
			wantParam: &ParamError{Op: "spawn_monster", Param: "count", Value: "abc"},
		},
		{
			name:   "count",
			target: plain,
			params: map[string]string{"monsterId": "100100", "count": "3"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    uuid.Nil,
				MonsterId:   100100,
				X:           0,
				Y:           0,
				Team:        0,
				Count:       3,
			},
		},
		{
			name:   "team",
			target: plain,
			params: map[string]string{"monsterId": "100100", "team": "1"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    uuid.Nil,
				MonsterId:   100100,
				X:           0,
				Y:           0,
				Team:        1,
				Count:       1,
			},
		},
		{
			name:    "team out of range",
			target:  plain,
			params:  map[string]string{"monsterId": "100100", "team": "200"},
			wantErr: `spawn_monster: parameter "team" value "200": out of range for int8`,
		},
		{
			name:    "x out of range",
			target:  plain,
			params:  map[string]string{"monsterId": "100100", "x": "40000"},
			wantErr: `spawn_monster: parameter "x" value "40000": out of range for int16`,
		},
		{
			name:   "OQ-3 same map keeps instance",
			target: instanced,
			params: map[string]string{"monsterId": "100100", "mapId": "910010000"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    instID,
				MonsterId:   100100,
				X:           -120,
				Y:           33,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:   "OQ-3 cross map drops instance",
			target: instanced,
			params: map[string]string{"monsterId": "100100", "mapId": "910510202"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910510202,
				Instance:    uuid.Nil,
				MonsterId:   100100,
				X:           -120,
				Y:           33,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:   "mapId default keeps instance",
			target: instanced,
			params: map[string]string{"monsterId": "100100"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    instID,
				MonsterId:   100100,
				X:           -120,
				Y:           33,
				Team:        0,
				Count:       1,
			},
		},
		{
			name:   "arithmetic count",
			target: plain,
			params: map[string]string{"monsterId": "100100", "count": "2 * 3"},
			wantPayload: saga.SpawnMonsterPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				MapId:       910010000,
				Instance:    uuid.Nil,
				MonsterId:   100100,
				X:           0,
				Y:           0,
				Team:        0,
				Count:       6,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := SpawnMonster(tt.params, DirectResolver{}, tt.target, 7)
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
			if step.Action() != saga.SpawnMonster {
				t.Fatalf("got action %v, want %v", step.Action(), saga.SpawnMonster)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.SpawnMonsterPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}
