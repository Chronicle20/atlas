package npc

import (
	npcKafka "atlas-maps/kafka/message/npc"
	"context"
	"encoding/json"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestContext() context.Context {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), te)
}

// stubNpcEmit swaps the package-level emit seam for a recording stub that
// captures every EVENT_TOPIC_NPC_STATUS message published during the test,
// decoded as a CREATED status event, without touching a real Kafka
// connection -- mirrors atlas-channel's stubPlayerNpcAnnounce
// (kafka/consumer/map/player_npc_test.go).
func stubNpcEmit(t *testing.T) *[]npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody] {
	t.Helper()
	var captured []npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]
	orig := emit
	emit = func(l logrus.FieldLogger, ctx context.Context, tok topic.Token, prov model.Provider[[]kafka.Message]) error {
		msgs, err := prov()
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			var ev npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]
			if uerr := json.Unmarshal(msg.Value, &ev); uerr == nil {
				captured = append(captured, ev)
			}
		}
		return nil
	}
	t.Cleanup(func() { emit = orig })
	return &captured
}

func TestCreateNpcInField(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	_, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 1 {
		t.Fatalf("Expected 1 npc, got %d", len(ns))
	}
	if ns[0].NpcId() != 1104100 {
		t.Errorf("Expected NpcId 1104100, got %d", ns[0].NpcId())
	}
	if ns[0].X() != 2830 {
		t.Errorf("Expected X 2830, got %d", ns[0].X())
	}
	if ns[0].Y() != 78 {
		t.Errorf("Expected Y 78, got %d", ns[0].Y())
	}
}

func TestCreateNpcSpawnIfAbsentSuppressesWhenPresent(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("First create returned error: %v", err)
	}

	m, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true})
	if err != nil {
		t.Fatalf("Second create returned error: %v", err)
	}
	if m.UniqueId() != 0 {
		t.Errorf("Expected suppressed create to return UniqueId 0, got %d", m.UniqueId())
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 1 {
		t.Fatalf("Expected 1 npc after suppressed create, got %d", len(ns))
	}
}

func TestCreateNpcSpawnIfAbsentIsFieldScoped(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	fa := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).SetInstance(uuid.New()).Build()
	fb := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).SetInstance(uuid.New()).Build()

	if _, err := p.Create(fa, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("Create on field A returned error: %v", err)
	}

	m, err := p.Create(fb, RestModel{NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true})
	if err != nil {
		t.Fatalf("Create on field B returned error: %v", err)
	}
	if m.UniqueId() == 0 {
		t.Errorf("Expected create on field B to happen (different instance), got suppressed")
	}

	nb, err := p.GetInField(fb)
	if err != nil {
		t.Fatalf("GetInField(fb) returned error: %v", err)
	}
	if len(nb) != 1 {
		t.Fatalf("Expected 1 npc on field B, got %d", len(nb))
	}
}

func TestCreateNpcWithoutGuardStacks(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("First create returned error: %v", err)
	}
	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("Second create returned error: %v", err)
	}

	ns, err := p.GetInField(f)
	if err != nil {
		t.Fatalf("GetInField returned error: %v", err)
	}
	if len(ns) != 2 {
		t.Fatalf("Expected 2 npcs, got %d", len(ns))
	}
}

// TestCreateNpcEmitsCreatedStatusEvent asserts the bound path (task-BC):
// Processor.Create publishes one CREATED status event carrying the
// placement it just registered.
func TestCreateNpcEmitsCreatedStatusEvent(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)
	captured := stubNpcEmit(t)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	m, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78, Fh: 5})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(*captured) != 1 {
		t.Fatalf("Expected 1 CREATED status event, got %d", len(*captured))
	}
	ev := (*captured)[0]
	if ev.Type != npcKafka.EventNpcStatusTypeCreated {
		t.Errorf("Expected type %s, got %s", npcKafka.EventNpcStatusTypeCreated, ev.Type)
	}
	if ev.UniqueId != m.UniqueId() {
		t.Errorf("Expected UniqueId %d, got %d", m.UniqueId(), ev.UniqueId)
	}
	if ev.Body.NpcId != 1104100 || ev.Body.X != 2830 || ev.Body.Y != 78 || ev.Body.Fh != 5 {
		t.Errorf("Unexpected event body: %+v", ev.Body)
	}
}

// TestCreateNpcSpawnIfAbsentSuppressedEmitsNothing pins the known
// correctness trap (task-BC brief): when SpawnIfAbsent suppresses a
// duplicate spawn, Processor.Create returns a zero Model and MUST NOT emit
// any status event.
func TestCreateNpcSpawnIfAbsentSuppressedEmitsNothing(t *testing.T) {
	getRegistry().Reset()
	ctx := newTestContext()
	p := NewProcessor(logrus.StandardLogger(), ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()

	if _, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78}); err != nil {
		t.Fatalf("First create returned error: %v", err)
	}

	captured := stubNpcEmit(t)
	m, err := p.Create(f, RestModel{NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true})
	if err != nil {
		t.Fatalf("Suppressed create returned error: %v", err)
	}
	if m.UniqueId() != 0 {
		t.Fatalf("Expected suppressed create to return UniqueId 0, got %d", m.UniqueId())
	}
	if len(*captured) != 0 {
		t.Fatalf("Expected no status event on a suppressed create, got %d", len(*captured))
	}
}
