package ops

import (
	"errors"
	"strings"
	"testing"
	"time"

	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// withFixedNow overrides the package clock for the duration of the test, so
// expiration-bearing payloads are deterministic.
func withFixedNow(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}

func TestCreateSkill(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	withFixedNow(t, base)

	tests := []struct {
		name         string
		params       map[string]string
		wantErr      string
		wantParam    *ParamError
		wantRangeErr string
		wantPayload  saga.CreateSkillPayload
	}{
		{
			name:    "missing skillId",
			params:  map[string]string{},
			wantErr: `create_skill: parameter "skillId" is required`,
		},
		{
			name:      "bad skillId",
			params:    map[string]string{"skillId": "abc"},
			wantParam: &ParamError{Op: "create_skill", Param: "skillId", Value: "abc"},
		},
		{
			name:   "defaults",
			params: map[string]string{"skillId": "1001003"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:   "level/masterLevel",
			params: map[string]string{"skillId": "1001003", "level": "5", "masterLevel": "20"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       5,
				MasterLevel: 20,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:   "level widened past 127",
			params: map[string]string{"skillId": "1001003", "level": "200"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       200,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:         "level out of byte range",
			params:       map[string]string{"skillId": "1001003", "level": "256"},
			wantRangeErr: "out of range for byte",
		},
		{
			name:   "expiration -1 sentinel",
			params: map[string]string{"skillId": "1001003", "expiration": "-1"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(100 * 365 * 24 * time.Hour),
			},
		},
		{
			name:   "expiration epoch ms",
			params: map[string]string{"skillId": "1001003", "expiration": "1767225600000"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  time.UnixMilli(1767225600000),
			},
		},
		{
			name:   "expiration zero falls back",
			params: map[string]string{"skillId": "1001003", "expiration": "0"},
			wantPayload: saga.CreateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:      "bad expiration",
			params:    map[string]string{"skillId": "1001003", "expiration": "abc"},
			wantParam: &ParamError{Op: "create_skill", Param: "expiration", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := CreateSkill(tt.params, DirectResolver{}, Target{}, 7)
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
			if step.Action() != saga.CreateSkill {
				t.Fatalf("got action %v, want %v", step.Action(), saga.CreateSkill)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.CreateSkillPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload.CharacterId != tt.wantPayload.CharacterId ||
				payload.SkillId != tt.wantPayload.SkillId ||
				payload.Level != tt.wantPayload.Level ||
				payload.MasterLevel != tt.wantPayload.MasterLevel ||
				!payload.Expiration.Equal(tt.wantPayload.Expiration) {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestUpdateSkill(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	withFixedNow(t, base)

	tests := []struct {
		name         string
		params       map[string]string
		wantErr      string
		wantParam    *ParamError
		wantRangeErr string
		wantPayload  saga.UpdateSkillPayload
	}{
		{
			name:    "missing skillId",
			params:  map[string]string{},
			wantErr: `update_skill: parameter "skillId" is required`,
		},
		{
			name:      "bad skillId",
			params:    map[string]string{"skillId": "abc"},
			wantParam: &ParamError{Op: "update_skill", Param: "skillId", Value: "abc"},
		},
		{
			name:   "defaults",
			params: map[string]string{"skillId": "1001003"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:   "level/masterLevel",
			params: map[string]string{"skillId": "1001003", "level": "5", "masterLevel": "20"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       5,
				MasterLevel: 20,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:   "level widened past 127",
			params: map[string]string{"skillId": "1001003", "level": "200"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       200,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:         "level out of byte range",
			params:       map[string]string{"skillId": "1001003", "level": "256"},
			wantRangeErr: "out of range for byte",
		},
		{
			name:   "expiration -1 sentinel",
			params: map[string]string{"skillId": "1001003", "expiration": "-1"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(100 * 365 * 24 * time.Hour),
			},
		},
		{
			name:   "expiration epoch ms",
			params: map[string]string{"skillId": "1001003", "expiration": "1767225600000"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  time.UnixMilli(1767225600000),
			},
		},
		{
			name:   "expiration zero falls back",
			params: map[string]string{"skillId": "1001003", "expiration": "0"},
			wantPayload: saga.UpdateSkillPayload{
				CharacterId: 7,
				SkillId:     1001003,
				Level:       1,
				MasterLevel: 1,
				Expiration:  base.Add(365 * 24 * time.Hour),
			},
		},
		{
			name:      "bad expiration",
			params:    map[string]string{"skillId": "1001003", "expiration": "abc"},
			wantParam: &ParamError{Op: "update_skill", Param: "expiration", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := UpdateSkill(tt.params, DirectResolver{}, Target{}, 7)
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
			if step.Action() != saga.UpdateSkill {
				t.Fatalf("got action %v, want %v", step.Action(), saga.UpdateSkill)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.UpdateSkillPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload.CharacterId != tt.wantPayload.CharacterId ||
				payload.SkillId != tt.wantPayload.SkillId ||
				payload.Level != tt.wantPayload.Level ||
				payload.MasterLevel != tt.wantPayload.MasterLevel ||
				!payload.Expiration.Equal(tt.wantPayload.Expiration) {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}
