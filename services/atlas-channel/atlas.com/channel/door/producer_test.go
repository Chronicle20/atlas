package door

import (
	"encoding/json"
	"testing"

	doormsg "atlas-channel/kafka/message/door"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// TestSpawnCommandProvider verifies that SpawnCommandProvider emits a single
// Kafka message keyed by area map id and carrying the expected command envelope.
func TestSpawnCommandProvider(t *testing.T) {
	areaMapId := _map.Id(105040300)
	f := field.NewBuilder(world.Id(0), channel.Id(1), areaMapId).Build()

	ownerCharacterId := uint32(777888)
	skillId := uint32(2311002)
	level := byte(3)
	x := int16(-120)
	y := int16(85)

	msgs, err := SpawnCommandProvider(f, ownerCharacterId, skillId, level, x, y)()
	if err != nil {
		t.Fatalf("SpawnCommandProvider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	// Verify the key is derived from the area map id.
	expectedKey := producer.CreateKey(int(areaMapId))
	if string(msgs[0].Key) != string(expectedKey) {
		t.Errorf("key: got %v, want %v", msgs[0].Key, expectedKey)
	}

	// Decode and assert the command envelope.
	var cmd doormsg.Command[doormsg.SpawnBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cmd.WorldId != world.Id(0) {
		t.Errorf("worldId: got %d, want 0", cmd.WorldId)
	}
	if cmd.ChannelId != channel.Id(1) {
		t.Errorf("channelId: got %d, want 1", cmd.ChannelId)
	}
	if cmd.MapId != areaMapId {
		t.Errorf("mapId: got %d, want %d", cmd.MapId, areaMapId)
	}
	if cmd.OwnerCharacterId != ownerCharacterId {
		t.Errorf("ownerCharacterId: got %d, want %d", cmd.OwnerCharacterId, ownerCharacterId)
	}
	if cmd.Type != doormsg.CommandTypeSpawn {
		t.Errorf("type: got %q, want %q", cmd.Type, doormsg.CommandTypeSpawn)
	}
	if cmd.Body.SkillId != skillId {
		t.Errorf("skillId: got %d, want %d", cmd.Body.SkillId, skillId)
	}
	if cmd.Body.SkillLevel != level {
		t.Errorf("skillLevel: got %d, want %d", cmd.Body.SkillLevel, level)
	}
	if cmd.Body.X != x {
		t.Errorf("x: got %d, want %d", cmd.Body.X, x)
	}
	if cmd.Body.Y != y {
		t.Errorf("y: got %d, want %d", cmd.Body.Y, y)
	}
}
