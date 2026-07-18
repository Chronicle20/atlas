package game_test

import (
	"atlas-rps/game"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestModelBuilderRoundTripsThroughJSON(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	m := game.NewModelBuilder(ten).
		SetCharacterId(100).SetWorldId(0).SetChannelId(1).SetNpcId(9000019).
		SetRung(2).SetStatus(game.StatusAwaitingDecision).MustBuild()

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out game.Model
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.CharacterId() != 100 || out.Rung() != 2 || out.Status() != game.StatusAwaitingDecision {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestModelBuilderRejectsZeroCharacter(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	if _, err := game.NewModelBuilder(ten).SetCharacterId(0).Build(); err == nil {
		t.Fatal("expected error for characterId 0")
	}
}
