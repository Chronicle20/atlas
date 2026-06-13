package food

import (
	"encoding/json"
	"testing"

	foodmsg "atlas-channel/kafka/message/food"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestRequestFeedCommandProvider verifies the channel taming-mob food command
// carries worldId/channelId/mapId from the field plus slot/itemId in the body,
// matching the consumables Command envelope Task 32 decodes.
func TestRequestFeedCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(2), channel.Id(5), _map.Id(910000000)).Build()
	cid := character.Id(12345)

	provider := RequestFeedCommandProvider(f, cid, int16(7), uint32(2000000))
	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd foodmsg.Command[foodmsg.RequestFeedBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cmd.WorldId != world.Id(2) {
		t.Errorf("worldId: got %d, want 2", cmd.WorldId)
	}
	if cmd.ChannelId != channel.Id(5) {
		t.Errorf("channelId: got %d, want 5", cmd.ChannelId)
	}
	if cmd.MapId != _map.Id(910000000) {
		t.Errorf("mapId: got %d, want 910000000", cmd.MapId)
	}
	if cmd.CharacterId != cid {
		t.Errorf("characterId: got %d, want %d", cmd.CharacterId, cid)
	}
	if cmd.Type != foodmsg.CommandRequestFeed {
		t.Errorf("type: got %q, want %q", cmd.Type, foodmsg.CommandRequestFeed)
	}
	if cmd.Body.Slot != 7 {
		t.Errorf("slot: got %d, want 7", cmd.Body.Slot)
	}
	if cmd.Body.ItemId != 2000000 {
		t.Errorf("itemId: got %d, want 2000000", cmd.Body.ItemId)
	}
}
