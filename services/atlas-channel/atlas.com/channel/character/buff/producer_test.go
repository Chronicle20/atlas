package buff

import (
	"encoding/json"
	"testing"

	buffmsg "atlas-channel/kafka/message/buff"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestCancelByTypesCommandProvider(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	types := []string{"STUN", "POISON"}

	msgs, err := CancelByTypesCommandProvider(f, 42, types)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd buffmsg.Command[buffmsg.CancelByTypesCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cmd.Type != buffmsg.CommandTypeCancelByTypes {
		t.Errorf("Type = %q, want %q", cmd.Type, buffmsg.CommandTypeCancelByTypes)
	}
	if cmd.CharacterId != 42 {
		t.Errorf("CharacterId = %d, want 42", cmd.CharacterId)
	}
	if len(cmd.Body.Types) != 2 || cmd.Body.Types[0] != "STUN" || cmd.Body.Types[1] != "POISON" {
		t.Errorf("Body.Types = %v, want [STUN POISON]", cmd.Body.Types)
	}
}
