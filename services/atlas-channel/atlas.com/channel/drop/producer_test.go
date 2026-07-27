package drop_test

import (
	"atlas-channel/drop"
	drop2 "atlas-channel/kafka/message/drop"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
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

// One CONSUME command per exploded drop, keyed by dropId, all in a single
// buffered provider (task-150 design §4.3-A / FR-8).
func TestConsumeAllCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	txId := uuid.New()
	dropIds := []uint32{11, 22, 33}

	msgs, err := drop.ConsumeAllCommandProvider(txId, f, dropIds)()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != len(dropIds) {
		t.Fatalf("got %d messages, want %d", len(msgs), len(dropIds))
	}
	for i, want := range dropIds {
		if !bytes.Equal(msgs[i].Key, producer.CreateKey(int(want))) {
			t.Errorf("message %d key mismatch", i)
		}
		var cmd drop2.Command[drop2.ConsumeCommandBody]
		if err := json.Unmarshal(msgs[i].Value, &cmd); err != nil {
			t.Fatalf("message %d unmarshal: %v", i, err)
		}
		if cmd.Type != drop2.CommandTypeConsume {
			t.Errorf("message %d type = %q, want %q", i, cmd.Type, drop2.CommandTypeConsume)
		}
		if cmd.Body.DropId != want {
			t.Errorf("message %d dropId = %d, want %d", i, cmd.Body.DropId, want)
		}
		if cmd.TransactionId != txId {
			t.Errorf("message %d transactionId = %s, want %s", i, cmd.TransactionId, txId)
		}
		if cmd.WorldId != f.WorldId() || cmd.ChannelId != f.ChannelId() || cmd.MapId != f.MapId() || cmd.Instance != f.Instance() {
			t.Errorf("message %d field envelope mismatch: %+v", i, cmd)
		}
	}
}
