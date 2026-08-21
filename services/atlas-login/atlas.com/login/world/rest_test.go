package world

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestRestModel_HydratesIncludedChannels feeds a JSON:API document for one
// world whose `channels` to-many relationship lists two ids and whose
// `included` array carries their attributes in [1, 0] order (the order
// atlas-world actually returns), then asserts that every extracted channel's
// ChannelId, CurrentCapacity and Port survive jsonapi.Unmarshal in
// relationship order. This is the regression net for the bug where every
// channel came back with ChannelId/CurrentCapacity/Port all zero because
// world.RestModel did not implement jsonapi.UnmarshalIncludedRelations.
func TestRestModel_HydratesIncludedChannels(t *testing.T) {
	doc := []byte(`{
      "data": {
        "type": "worlds",
        "id": "0",
        "attributes": {"name": "Scania"},
        "relationships": {
          "channels": {
            "data": [
              {"type": "channels", "id": "00000000-0000-0000-0000-000000000000"},
              {"type": "channels", "id": "00000000-0000-0000-0000-000000000001"}
            ]
          }
        }
      },
      "included": [
        {
          "type": "channels",
          "id": "00000000-0000-0000-0000-000000000001",
          "attributes": {
            "worldId": 0,
            "channelId": 1,
            "ipAddress": "127.0.0.1",
            "port": 7576,
            "currentCapacity": 12,
            "maxCapacity": 200
          }
        },
        {
          "type": "channels",
          "id": "00000000-0000-0000-0000-000000000000",
          "attributes": {
            "worldId": 0,
            "channelId": 0,
            "ipAddress": "127.0.0.1",
            "port": 7575,
            "currentCapacity": 3,
            "maxCapacity": 200
          }
        }
      ]
    }`)

	var w RestModel
	if err := jsonapi.Unmarshal(doc, &w); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(w.Channels) != 2 {
		t.Fatalf("len(Channels) = %d, want 2", len(w.Channels))
	}

	c0 := w.Channels[0]
	if c0.GetID() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Channels[0].GetID() = %s, want the first relationship id", c0.GetID())
	}
	if c0.ChannelId != 0 {
		t.Errorf("Channels[0].ChannelId = %d, want 0", c0.ChannelId)
	}
	if c0.Port != 7575 {
		t.Errorf("Channels[0].Port = %d, want 7575", c0.Port)
	}
	if c0.CurrentCapacity != 3 {
		t.Errorf("Channels[0].CurrentCapacity = %d, want 3", c0.CurrentCapacity)
	}

	c1 := w.Channels[1]
	if c1.GetID() != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("Channels[1].GetID() = %s, want the second relationship id", c1.GetID())
	}
	if c1.ChannelId != 1 {
		t.Errorf("Channels[1].ChannelId = %d, want 1", c1.ChannelId)
	}
	if c1.Port != 7576 {
		t.Errorf("Channels[1].Port = %d, want 7576", c1.Port)
	}
	if c1.CurrentCapacity != 12 {
		t.Errorf("Channels[1].CurrentCapacity = %d, want 12", c1.CurrentCapacity)
	}
}
