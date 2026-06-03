package monster

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSpawnFieldCommandProvider_EmitsCountMessages(t *testing.T) {
	inst := uuid.New()
	msgs, err := SpawnFieldCommandProvider(1, 2, 100000000, inst, 100100, 250, -130, 7, 0, 3)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len(msgs) = %d, want 3", len(msgs))
	}

	var cmd FieldCommand[SpawnFieldBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != CommandTypeSpawnField {
		t.Errorf("Type = %q, want %q", cmd.Type, CommandTypeSpawnField)
	}
	if cmd.MapId != 100000000 || cmd.Instance != inst {
		t.Errorf("envelope mismatch: mapId=%d instance=%s", cmd.MapId, cmd.Instance)
	}
	if cmd.Body.MonsterId != 100100 || cmd.Body.X != 250 || cmd.Body.Y != -130 || cmd.Body.Fh != 7 || cmd.Body.Team != 0 {
		t.Errorf("body mismatch: %+v", cmd.Body)
	}
}
