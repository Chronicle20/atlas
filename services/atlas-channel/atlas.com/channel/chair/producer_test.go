package chair

import (
	"encoding/json"
	"testing"

	chair2 "atlas-channel/kafka/message/chair"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestRecoveryCommandProvider(t *testing.T) {
	f := field.NewBuilder(0, 1, 100000000).Build()
	msgs, err := RecoveryCommandProvider(f, 12345, 50, -3)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var c chair2.Command[chair2.RecoveryCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != chair2.CommandRecovery {
		t.Errorf("type: got %s, want RECOVERY", c.Type)
	}
	if c.WorldId != 0 || c.ChannelId != 1 || c.MapId != 100000000 {
		t.Errorf("field routing: got %d/%d/%d", c.WorldId, c.ChannelId, c.MapId)
	}
	if c.Body.CharacterId != 12345 || c.Body.Hp != 50 || c.Body.Mp != -3 {
		t.Errorf("body: got %+v", c.Body)
	}
}
