package character

import (
	"encoding/json"
	"strings"
	"testing"

	character2 "atlas-saga-orchestrator/kafka/message/character"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestRequestCreateCharacterProvider_GmMeso(t *testing.T) {
	p := RequestCreateCharacterProvider(uuid.New(), 1, world.Id(0), "Hero", 200, 0, 0, 0, 0, 0, 0, job.Id(112), 0, 0, 0, 0, _map.Id(0), 2, 12345, 0, "")
	msgs, err := p()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	body := string(msgs[0].Value)
	if !strings.Contains(body, `"gm":2`) || !strings.Contains(body, `"meso":12345`) {
		t.Fatalf("expected gm/meso in body, got %s", body)
	}
}

// AP and SP are additive (task-246 amendment 1): the provider must carry
// them through to the produced command body unchanged.
func TestRequestCreateCharacterProviderCarriesApAndSp(t *testing.T) {
	p := RequestCreateCharacterProvider(uuid.New(), 1, world.Id(2), "Hero", 30, 4, 5, 6, 7, 8, 9, job.Id(112), 1, 10, 11, 3, _map.Id(100000), 2, 12345, 12, "3,0,0,0,0,0,0,0,0,0")
	msgs, err := p()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd character2.Command[character2.CreateCharacterCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Body.AP != 12 {
		t.Errorf("AP: got %d, want 12", cmd.Body.AP)
	}
	if cmd.Body.SP != "3,0,0,0,0,0,0,0,0,0" {
		t.Errorf("SP: got %q, want %q", cmd.Body.SP, "3,0,0,0,0,0,0,0,0,0")
	}
}
