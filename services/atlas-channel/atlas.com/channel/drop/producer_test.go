package drop_test

import (
	"atlas-channel/drop"
	drop2 "atlas-channel/kafka/message/drop"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestSpawnMesoCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	msgs, err := drop.SpawnMesoCommandProvider(f, 123, 45, -67, 999, 700001, 40, -67)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d; want 1", len(msgs))
	}

	var cmd drop2.Command[drop2.SpawnCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != drop2.CommandTypeSpawn {
		t.Fatalf("Type = %q; want %q", cmd.Type, drop2.CommandTypeSpawn)
	}
	if cmd.WorldId != f.WorldId() || cmd.ChannelId != f.ChannelId() || cmd.MapId != f.MapId() {
		t.Fatalf("field routing = %d/%d/%d; want %d/%d/%d",
			cmd.WorldId, cmd.ChannelId, cmd.MapId, f.WorldId(), f.ChannelId(), f.MapId())
	}

	b := cmd.Body
	if b.Mesos != 123 || b.X != 45 || b.Y != -67 {
		t.Fatalf("mesos/x/y = %d/%d/%d; want 123/45/-67", b.Mesos, b.X, b.Y)
	}
	if b.OwnerId != 999 || b.OwnerPartyId != 0 {
		t.Fatalf("owner = %d/%d; want 999/0", b.OwnerId, b.OwnerPartyId)
	}
	if b.DropperId != 700001 || b.DropperX != 40 || b.DropperY != -67 {
		t.Fatalf("dropper = %d@(%d,%d); want 700001@(40,-67)", b.DropperId, b.DropperX, b.DropperY)
	}
	if b.ItemId != 0 || b.Quantity != 0 {
		t.Fatalf("itemId/quantity = %d/%d; want 0/0 (meso-only drop)", b.ItemId, b.Quantity)
	}
	if b.DropType != 2 {
		t.Fatalf("DropType = %d; want 2 (FFA)", b.DropType)
	}
	if !b.PlayerDrop {
		t.Fatal("PlayerDrop = false; want true (universal pickup)")
	}
	if b.Mod != 0 {
		t.Fatalf("Mod = %d; want 0", b.Mod)
	}
}
