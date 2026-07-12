package character

import (
	"encoding/json"
	"testing"

	messagechar "atlas-channel/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestSetHPCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(3), 100000000).Build()
	msgs, err := SetHPCommandProvider(f, 4242, 0xFFFF)()
	if err != nil {
		t.Fatalf("provider err: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd messagechar.Command[messagechar.SetHPCommandBody]
	if uErr := json.Unmarshal(msgs[0].Value, &cmd); uErr != nil {
		t.Fatalf("unmarshal: %v", uErr)
	}
	if cmd.Type != messagechar.CommandSetHP {
		t.Fatalf("Type = %q, want %q", cmd.Type, messagechar.CommandSetHP)
	}
	if cmd.CharacterId != 4242 {
		t.Fatalf("CharacterId = %d, want 4242", cmd.CharacterId)
	}
	if cmd.Body.ChannelId != channel.Id(3) {
		t.Fatalf("Body.ChannelId = %d, want 3", cmd.Body.ChannelId)
	}
	if cmd.Body.Amount != 0xFFFF {
		t.Fatalf("Body.Amount = %d, want 65535", cmd.Body.Amount)
	}
}
