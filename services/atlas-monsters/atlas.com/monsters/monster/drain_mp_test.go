package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"errors"
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

// newDrainMpProcessor creates a ProcessorImpl with a recording emitter that
// captures the full envelope (type + body) for each emitted Kafka message.
func newDrainMpProcessor(t *testing.T, ten tenant.Model) (*ProcessorImpl, *[]emittedBody) {
	t.Helper()
	var events []emittedBody
	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: context.Background(),
		t:   ten,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider error in DrainMp test: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string          `json:"type"`
					Body json.RawMessage `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode emitted DrainMp message: %v", err)
				}
				events = append(events, emittedBody{Topic: topic, Type: env.Type, Body: env.Body})
			}
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) {
			return nil, nil
		},
	}
	return p, &events
}

// TestDrainMp_HappyPath_EmitsMpChanged verifies that DrainMp on a non-boss
// monster with MaxMp=1000, Mp=1000 deducts 100 MP and emits a single
// MP_CHANGED event with Reason=MP_EATER, Amount=100, MonsterMpAfter=900.
func TestDrainMp_HappyPath_EmitsMpChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	// Stub information lookup to return a non-boss monster.
	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 42, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}

	// Verify registry state.
	got, getErr := r.GetMonster(ten, uniqueId)
	if getErr != nil {
		t.Fatalf("GetMonster: %v", getErr)
	}
	if got.Mp() != 900 {
		t.Errorf("Mp after DrainMp = %d, want 900", got.Mp())
	}

	// Verify exactly one MP_CHANGED event.
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(*events), *events)
	}
	ev := (*events)[0]
	if ev.Type != EventMonsterStatusMpChanged {
		t.Errorf("event type = %q, want %q", ev.Type, EventMonsterStatusMpChanged)
	}
	if ev.Topic != EnvEventTopicMonsterStatus {
		t.Errorf("event topic = %q, want %q", ev.Topic, EnvEventTopicMonsterStatus)
	}

	var body statusEventMpChangedBody
	if err := json.Unmarshal(ev.Body, &body); err != nil {
		t.Fatalf("decode MP_CHANGED body: %v", err)
	}
	if body.Reason != MpChangeReasonMpEater {
		t.Errorf("body.Reason = %q, want %q", body.Reason, MpChangeReasonMpEater)
	}
	if body.Amount != 100 {
		t.Errorf("body.Amount = %d, want 100", body.Amount)
	}
	if body.MonsterMpAfter != 900 {
		t.Errorf("body.MonsterMpAfter = %d, want 900", body.MonsterMpAfter)
	}
	if body.CharacterId != 42 {
		t.Errorf("body.CharacterId = %d, want 42", body.CharacterId)
	}
	if body.SkillId != 2300000 {
		t.Errorf("body.SkillId = %d, want 2300000", body.SkillId)
	}
}

// TestDrainMp_ClampsAtZero verifies that when the monster has Mp=50 and the
// request is 200, the drain clamps at 0 (actual drain = 50) and the event
// Amount=50 with MonsterMpAfter=0.
func TestDrainMp_ClampsAtZero(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()
	// Manually reduce MP to 50 by deducting 950.
	if _, err := r.DeductMp(ten, uniqueId, 950); err != nil {
		t.Fatalf("seed DeductMp: %v", err)
	}

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 1, 2300000, 200)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}

	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 0 {
		t.Errorf("Mp after clamp-at-zero DrainMp = %d, want 0", got.Mp())
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}
	var body statusEventMpChangedBody
	if err := json.Unmarshal((*events)[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Amount != 50 {
		t.Errorf("body.Amount = %d, want 50", body.Amount)
	}
	if body.MonsterMpAfter != 0 {
		t.Errorf("body.MonsterMpAfter = %d, want 0", body.MonsterMpAfter)
	}
}

// TestDrainMp_SkipsZeroMaxMp verifies that DrainMp emits nothing when
// MaxMp == 0 (monster was created with zero max MP).
func TestDrainMp_SkipsZeroMaxMp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	// Create with mp=0 → MaxMp==0.
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 0)
	uniqueId := m.UniqueId()

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for MaxMp=0 monster, got %d: %v", len(*events), *events)
	}
}

// TestDrainMp_SkipsZeroCurrentMp verifies that DrainMp emits nothing when the
// monster's current Mp == 0 (already drained).
func TestDrainMp_SkipsZeroCurrentMp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()
	// Drain all MP first.
	if _, err := r.DeductMp(ten, uniqueId, 1000); err != nil {
		t.Fatalf("seed DeductMp: %v", err)
	}

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for Mp=0 monster, got %d", len(*events))
	}
}

// TestDrainMp_SkipsZeroRequest verifies that DrainMp with requestedAmount=0
// emits nothing and leaves Mp unchanged.
func TestDrainMp_SkipsZeroRequest(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(false).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 1000000, 0, 0, 0, 5, 0, 5000, 1000)
	uniqueId := m.UniqueId()

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 1, 2300000, 0)
	if err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for requestedAmount=0, got %d", len(*events))
	}
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 1000 {
		t.Errorf("Mp after zero-request DrainMp = %d, want 1000 (unchanged)", got.Mp())
	}
}

// TestDrainMp_MissingMonster verifies that DrainMp returns nil and emits no
// event when the uniqueId is not present in the registry.
func TestDrainMp_MissingMonster(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(99999999, 1, 2300000, 100)
	if err != nil {
		t.Fatalf("DrainMp returned error for missing monster: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for missing monster, got %d", len(*events))
	}
}

// TestDrainMp_SkipsBoss verifies that DrainMp emits nothing and leaves Mp
// unchanged when the information lookup indicates the monster is a boss.
func TestDrainMp_SkipsBoss(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetBoss(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 8800000, 0, 0, 0, 5, 0, 50000, 3000)
	uniqueId := m.UniqueId()

	p, events := newDrainMpProcessor(t, ten)
	err := p.DrainMp(uniqueId, 1, 2300000, 500)
	if err != nil {
		t.Fatalf("DrainMp returned error for boss: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("expected 0 events for boss monster, got %d", len(*events))
	}
	got, _ := r.GetMonster(ten, uniqueId)
	if got.Mp() != 3000 {
		t.Errorf("Mp after boss DrainMp = %d, want 3000 (unchanged)", got.Mp())
	}
}

// Compile-time sentinel — keeps the unused-import error from appearing while
// the test file exists without all referenced packages being used.
var _ = errors.New
