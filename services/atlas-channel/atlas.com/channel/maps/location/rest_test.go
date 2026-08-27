package location

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransform(t *testing.T) {
	instance := uuid.New()
	m := NewModelForTest(100, world.Id(1), channel.Id(2), _map.Id(3), instance, characterconst.PresenceStateInField)

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != 100 {
		t.Errorf("Id mismatch. Expected %v, got %v", 100, rm.Id)
	}

	if rm.WorldId != world.Id(1) {
		t.Errorf("WorldId mismatch. Expected %v, got %v", world.Id(1), rm.WorldId)
	}

	if rm.ChannelId != channel.Id(2) {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", channel.Id(2), rm.ChannelId)
	}

	if rm.MapId != _map.Id(3) {
		t.Errorf("MapId mismatch. Expected %v, got %v", _map.Id(3), rm.MapId)
	}

	if rm.Instance != instance {
		t.Errorf("Instance mismatch. Expected %v, got %v", instance, rm.Instance)
	}

	if rm.State != string(characterconst.PresenceStateInField) {
		t.Errorf("State mismatch. Expected %v, got %v", string(characterconst.PresenceStateInField), rm.State)
	}
}
