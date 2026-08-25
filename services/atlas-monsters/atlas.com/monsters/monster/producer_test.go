package monster

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestStartControlBodyEncodesControllerHasAggro(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := Clone(NewMonster(f, 1, 9300018, 0, 0, 0, 5, 0, 100, 50, "", "")).
		SetControlCharacterId(42).
		SetControllerHasAggro(true).
		Build()
	msgs, err := startControlStatusEventProvider(m)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env struct {
		Type string                      `json:"type"`
		Body statusEventStartControlBody `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Type != EventMonsterStatusStartControl {
		t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusStartControl)
	}
	if env.Body.ActorId != 42 {
		t.Errorf("ActorId=%d, want 42", env.Body.ActorId)
	}
	if !env.Body.ControllerHasAggro {
		t.Errorf("ControllerHasAggro=%v, want true", env.Body.ControllerHasAggro)
	}
}

func TestAggroChangedBodyEncoding(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).
		SetInstance(uuid.Nil).Build()
	m := Clone(NewMonster(f, 5, 9300018, 0, 0, 0, 5, 0, 100, 50, "", "")).
		SetControlCharacterId(7).
		SetControllerHasAggro(true).
		Build()
	msgs, err := aggroChangedStatusEventProvider(m, 7, true)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env struct {
		Type string                      `json:"type"`
		Body statusEventAggroChangedBody `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Type != EventMonsterStatusAggroChanged {
		t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusAggroChanged)
	}
	if env.Body.ControllerCharacterId != 7 || !env.Body.ControllerHasAggro {
		t.Errorf("body unexpected: %+v", env.Body)
	}
}

func TestKilledBodyCarriesDeathType(t *testing.T) {
	tests := []struct {
		name          string
		deathType     string
		killerId      uint32
		wantDeathType string
		wantActorId   uint32
	}{
		{"ordinary kill", DeathTypeFadeOut, 42, DeathTypeFadeOut, 42},
		{"self-destruct action 3", DeathTypeDestructByMiss, 42, DeathTypeDestructByMiss, 42},
		{"no killer", DeathTypeSelfDestruct, 0, DeathTypeSelfDestruct, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Clone(NewMonster(field.NewBuilder(0, 0, 40000).Build(), 1, 5100002, 0, 0, 0, 5, 0, 100, 50, "", "")).Build()
			msgs, err := killedStatusEventProvider(m, tt.killerId, false, nil, tt.deathType)()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			var env struct {
				Type string                `json:"type"`
				Body statusEventKilledBody `json:"body"`
			}
			if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Type != EventMonsterStatusKilled {
				t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusKilled)
			}
			if env.Body.DeathType != tt.wantDeathType {
				t.Errorf("DeathType=%s, want %s", env.Body.DeathType, tt.wantDeathType)
			}
			if env.Body.ActorId != tt.wantActorId {
				t.Errorf("ActorId=%d, want %d", env.Body.ActorId, tt.wantActorId)
			}
		})
	}
}

func TestDestroyedBodyCarriesDeathType(t *testing.T) {
	m := Clone(NewMonster(field.NewBuilder(0, 0, 40000).Build(), 1, 5100002, 0, 0, 0, 5, 0, 100, 50, "", "")).Build()
	msgs, err := destroyedStatusEventProvider(m, DeathTypeUnset)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env struct {
		Type string                   `json:"type"`
		Body statusEventDestroyedBody `json:"body"`
	}
	if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Type != EventMonsterStatusDestroyed {
		t.Errorf("type=%s, want %s", env.Type, EventMonsterStatusDestroyed)
	}
	if env.Body.DeathType != "" {
		t.Errorf("DeathType=%q, want empty", env.Body.DeathType)
	}
}

func TestKilledBodyDeathTypeIsOmittedShapeCompatible(t *testing.T) {
	raw := `{"x":0,"y":0,"actorId":9,"boss":false,"damageEntries":null}`
	var body statusEventKilledBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DeathType != "" {
		t.Errorf("DeathType=%q, want empty", body.DeathType)
	}
}
