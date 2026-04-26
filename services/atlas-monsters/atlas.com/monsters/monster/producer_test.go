package monster

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStartControlBodyEncodesControllerHasAggro(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := Clone(NewMonster(f, 1, 9300018, 0, 0, 0, 5, 0, 100, 50)).
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
		Type string                       `json:"type"`
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
	m := Clone(NewMonster(f, 5, 9300018, 0, 0, 0, 5, 0, 100, 50)).
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
		Type string                        `json:"type"`
		Body statusEventAggroChangedBody  `json:"body"`
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
