package ops

import (
	"errors"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestShowIntro(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		wantErr     string
		wantPayload saga.ShowIntroPayload
	}{
		{
			name:    "missing path",
			params:  map[string]string{},
			wantErr: `show_intro: parameter "path" is required`,
		},
		{
			name:   "ok",
			params: map[string]string{"path": "Effect/Direction1.img/aranTutorial/ClickPoleArm"},
			wantPayload: saga.ShowIntroPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				Path:        "Effect/Direction1.img/aranTutorial/ClickPoleArm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := ShowIntro(tt.params, DirectResolver{}, plain, 7)
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
			if step.Action() != saga.ShowIntro {
				t.Fatalf("got action %v, want %v", step.Action(), saga.ShowIntro)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.ShowIntroPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestShowHint(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name         string
		params       map[string]string
		wantErr      string
		wantParam    *ParamError
		wantRangeErr string
		wantPayload  saga.ShowHintPayload
	}{
		{
			name:    "missing hint",
			params:  map[string]string{},
			wantErr: `show_hint: parameter "hint" is required`,
		},
		{
			name:   "defaults",
			params: map[string]string{"hint": "go left"},
			wantPayload: saga.ShowHintPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				Hint:        "go left",
				Width:       0,
				Height:      0,
			},
		},
		{
			name:   "width/height",
			params: map[string]string{"hint": "go left", "width": "200", "height": "50"},
			wantPayload: saga.ShowHintPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				Hint:        "go left",
				Width:       200,
				Height:      50,
			},
		},
		{
			name:      "bad width",
			params:    map[string]string{"hint": "h", "width": "abc"},
			wantParam: &ParamError{Op: "show_hint", Param: "width", Value: "abc"},
		},
		{
			name:         "width out of range",
			params:       map[string]string{"hint": "h", "width": "65536"},
			wantRangeErr: "out of range for uint16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := ShowHint(tt.params, DirectResolver{}, plain, 7)
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
			if step.Action() != saga.ShowHint {
				t.Fatalf("got action %v, want %v", step.Action(), saga.ShowHint)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.ShowHintPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}

func TestPlayPortalSound(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	step, err := PlayPortalSound(map[string]string{}, DirectResolver{}, plain, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step.Action() != saga.PlayPortalSound {
		t.Fatalf("got action %v, want %v", step.Action(), saga.PlayPortalSound)
	}
	if step.Status() != saga.Pending {
		t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
	}
	payload, err := PayloadOf[saga.PlayPortalSoundPayload](step)
	if err != nil {
		t.Fatalf("unexpected payload type error: %v", err)
	}
	want := saga.PlayPortalSoundPayload{
		CharacterId: 7,
		WorldId:     0,
		ChannelId:   1,
	}
	if payload != want {
		t.Fatalf("got payload %+v, want %+v", payload, want)
	}
}

func TestApplyConsumableEffect(t *testing.T) {
	plain := NewTargetBuilder(field.NewBuilder(0, 1, 910010000).Build()).Build()

	tests := []struct {
		name        string
		params      map[string]string
		wantErr     string
		wantParam   *ParamError
		wantPayload saga.ApplyConsumableEffectPayload
	}{
		{
			name:    "missing itemId",
			params:  map[string]string{},
			wantErr: `apply_consumable_effect: parameter "itemId" is required`,
		},
		{
			name:   "ok",
			params: map[string]string{"itemId": "2000000"},
			wantPayload: saga.ApplyConsumableEffectPayload{
				CharacterId: 7,
				WorldId:     0,
				ChannelId:   1,
				ItemId:      2000000,
			},
		},
		{
			name:      "bad itemId",
			params:    map[string]string{"itemId": "abc"},
			wantParam: &ParamError{Op: "apply_consumable_effect", Param: "itemId", Value: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := ApplyConsumableEffect(tt.params, DirectResolver{}, plain, 7)
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
			if step.Action() != saga.ApplyConsumableEffect {
				t.Fatalf("got action %v, want %v", step.Action(), saga.ApplyConsumableEffect)
			}
			if step.Status() != saga.Pending {
				t.Fatalf("got status %v, want %v", step.Status(), saga.Pending)
			}
			payload, err := PayloadOf[saga.ApplyConsumableEffectPayload](step)
			if err != nil {
				t.Fatalf("unexpected payload type error: %v", err)
			}
			if payload != tt.wantPayload {
				t.Fatalf("got payload %+v, want %+v", payload, tt.wantPayload)
			}
		})
	}
}
