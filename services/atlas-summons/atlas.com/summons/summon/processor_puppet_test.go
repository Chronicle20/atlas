package summon

import (
	"context"
	"encoding/json"
	"testing"

	"atlas-summons/data/skill/effect"
	monstermsg "atlas-summons/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// newPuppetProcessor wires a ProcessorImpl backed by miniredis, a stub effect
// source, and a capturing emitter so puppet ADD/REMOVE signaling can be asserted
// without kafka.
func newPuppetProcessor(t *testing.T, eff effect.Model) (*ProcessorImpl, *[]capturedMessage) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	registry = newRegistry(rc)
	idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	captured := &[]capturedMessage{}
	p := &ProcessorImpl{
		l:       logrus.New(),
		ctx:     ctx,
		t:       ten,
		effects: stubEffectSource{eff: eff},
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, perr := provider()
			if perr != nil {
				return perr
			}
			for _, msg := range msgs {
				var payload map[string]any
				_ = json.Unmarshal(msg.Value, &payload)
				*captured = append(*captured, capturedMessage{topic: topic, payload: payload})
			}
			return nil
		},
	}
	return p, captured
}

// findCommand returns the first captured message on the monster command topic
// whose "type" equals the given command type, or nil.
func findCommand(captured *[]capturedMessage, commandType string) *capturedMessage {
	for i := range *captured {
		c := &(*captured)[i]
		if c.topic == monstermsg.EnvCommandTopic && c.payload["type"] == commandType {
			return c
		}
	}
	return nil
}

func TestSpawnPuppetEmitsAddPuppet(t *testing.T) {
	p, captured := newPuppetProcessor(t, effectWithX(800, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	// 3111002 = Fire Puppet (PUPPET).
	m, err := p.Spawn(f, 42, 3111002, 20, 100, -50)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if !m.IsPuppet() {
		t.Fatalf("expected a puppet summon")
	}

	add := findCommand(captured, monstermsg.CommandTypeAddPuppet)
	if add == nil {
		t.Fatalf("expected an ADD_PUPPET command on %s", monstermsg.EnvCommandTopic)
	}
	if got := uint32(add.payload["ownerCharacterId"].(float64)); got != 42 {
		t.Fatalf("ADD_PUPPET ownerCharacterId: got %d want 42", got)
	}
	if got := int16(add.payload["x"].(float64)); got != 100 {
		t.Fatalf("ADD_PUPPET x: got %d want 100", got)
	}
	if got := int16(add.payload["y"].(float64)); got != -50 {
		t.Fatalf("ADD_PUPPET y: got %d want -50", got)
	}
}

func TestDespawnPuppetEmitsRemovePuppet(t *testing.T) {
	p, captured := newPuppetProcessor(t, effectWithX(800, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	m, err := p.Spawn(f, 42, 3111002, 20, 100, -50)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if err := p.Despawn(m.Id(), true); err != nil {
		t.Fatalf("Despawn returned error: %v", err)
	}

	remove := findCommand(captured, monstermsg.CommandTypeRemovePuppet)
	if remove == nil {
		t.Fatalf("expected a REMOVE_PUPPET command on %s", monstermsg.EnvCommandTopic)
	}
	if got := uint32(remove.payload["ownerCharacterId"].(float64)); got != 42 {
		t.Fatalf("REMOVE_PUPPET ownerCharacterId: got %d want 42", got)
	}
}

func TestSpawnNonPuppetEmitsNeitherPuppetSignal(t *testing.T) {
	p, captured := newPuppetProcessor(t, effectWithX(0, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	// 3111005 = Silver Hawk (ATTACKER, not a puppet).
	m, err := p.Spawn(f, 42, 3111005, 10, 100, -50)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if m.IsPuppet() {
		t.Fatalf("3111005 should not be a puppet")
	}
	if err := p.Despawn(m.Id(), true); err != nil {
		t.Fatalf("Despawn returned error: %v", err)
	}

	if add := findCommand(captured, monstermsg.CommandTypeAddPuppet); add != nil {
		t.Fatalf("non-puppet must not emit ADD_PUPPET")
	}
	if remove := findCommand(captured, monstermsg.CommandTypeRemovePuppet); remove != nil {
		t.Fatalf("non-puppet must not emit REMOVE_PUPPET")
	}
}
