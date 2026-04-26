package monster

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// emittedEvent captures a single Kafka message emitted during a test.
type emittedEvent struct {
	Topic string
	Type  string
}

// newRecordingProcessor constructs a ProcessorImpl with a stub emitter that
// records every emitted event so tests can assert on ordering and topic.
func newRecordingProcessor(t *testing.T, ten tenant.Model) (*ProcessorImpl, *[]emittedEvent) {
	t.Helper()
	var events []emittedEvent
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string `json:"type"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode emitted message: %v", err)
				}
				events = append(events, emittedEvent{Topic: topic, Type: env.Type})
			}
			return nil
		},
	}
	return p, &events
}

// TestDamageMultiLineKillOnLastLine verifies that when the killing blow is the
// final damage line (40+30+50=120 vs 100 HP), both a DAMAGED event and a
// KILLED event are emitted in that order and the monster is removed from the
// registry.
func TestDamageMultiLineKillOnLastLine(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, charId, []uint32{40, 30, 50}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}
	if (*events)[0].Topic != EnvEventTopicMonsterStatus || (*events)[1].Topic != EnvEventTopicMonsterStatus {
		t.Errorf("events emitted to wrong topic: %v", *events)
	}

	// Monster must have been removed from the registry.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageMultiLineKillOnMiddleLine verifies that when the killing blow is
// the second of three lines (40+80=120 vs 100 HP), the third line is NOT
// applied and only 2 events are emitted.
func TestDamageMultiLineKillOnMiddleLine(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	// Line 1: 40 damage (HP→60), Line 2: 80 damage (kills; HP→0), Line 3: 50 (must NOT apply)
	p.Damage(uniqueId, charId, []uint32{40, 80, 50}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}

	// Monster must be gone.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageSingleLineKill verifies that a one-line killing attack emits both
// DAMAGED and KILLED (in that order). This is a deliberate behavior change
// from pre-task-030 code where a one-line kill emitted only KILLED.
func TestDamageSingleLineKill(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(9300018)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 50, 50)
	uniqueId := m.UniqueId()

	charId := uint32(1)
	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, charId, []uint32{200}, 0)

	if len(*events) != 2 {
		t.Fatalf("expected 2 events (damaged+killed), got %d: %v", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusDamaged {
		t.Errorf("event[0]: expected %q, got %q", EventMonsterStatusDamaged, (*events)[0].Type)
	}
	if (*events)[1].Type != EventMonsterStatusKilled {
		t.Errorf("event[1]: expected %q, got %q", EventMonsterStatusKilled, (*events)[1].Type)
	}

	// Monster must be gone.
	if _, err := r.GetMonster(ten, uniqueId); err == nil {
		t.Error("expected monster to be removed from registry after kill, but GetMonster succeeded")
	}
}

// TestDamageEmptySlice verifies that an empty damages slice results in zero
// events and the monster is left untouched.
func TestDamageEmptySlice(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, uint32(9300018), 0, 0, 0, 5, 0, 100, 50)
	uniqueId := m.UniqueId()

	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, 1, []uint32{}, 0)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for empty damages, got %d: %v", len(*events), *events)
	}
	// Monster must still exist and be at full HP.
	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("expected monster to remain in registry, got error: %v", err)
	}
	if got.Hp() != 100 {
		t.Errorf("expected monster HP=100 (unchanged), got %d", got.Hp())
	}
}

// TestDamageAlreadyDeadMonster verifies that Damage against a monster with
// HP=0 emits no events.
func TestDamageAlreadyDeadMonster(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	// Create with HP=1, then kill it directly via the registry so HP=0.
	m := r.CreateMonster(ctx, ten, f, uint32(9300018), 0, 0, 0, 5, 0, 1, 50)
	uniqueId := m.UniqueId()
	// Apply a killing hit directly via the registry (no emit).
	r.ApplyDamage(ten, 1, 999, uniqueId)
	// Do NOT remove the monster — just leave it at HP=0 in the registry so
	// Processor.Damage hits the !m.Alive() early-return path.

	p, events := newRecordingProcessor(t, ten)
	p.Damage(uniqueId, 1, []uint32{100}, 0)

	if len(*events) != 0 {
		t.Fatalf("expected 0 events for already-dead monster, got %d: %v", len(*events), *events)
	}
}
