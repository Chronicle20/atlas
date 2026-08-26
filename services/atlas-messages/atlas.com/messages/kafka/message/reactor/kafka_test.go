package reactor

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestDestroyInFieldCommandProvider_EmitsEnvelope(t *testing.T) {
	inst := uuid.New()
	msgs, err := DestroyInFieldCommandProvider(1, 2, 610030400, inst)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}

	var cmd Command[DestroyInFieldCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != CommandTypeDestroyInField {
		t.Errorf("Type = %q, want %q", cmd.Type, CommandTypeDestroyInField)
	}
	if cmd.WorldId != 1 || cmd.ChannelId != 2 || cmd.MapId != 610030400 || cmd.Instance != inst {
		t.Errorf("envelope mismatch: %+v", cmd)
	}
}
