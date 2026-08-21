package monster

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recordingProcessor builds a ProcessorImpl whose emit hook counts MONSTER_STATUS
// emissions, so a test can assert whether a controller assignment announced
// itself on the wire.
func recordingProcessor(ctx context.Context, tm tenant.Model, emitted *int) *ProcessorImpl {
	return &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(topic string, _ model.Provider[[]kafka.Message]) error {
			if topic == EnvEventTopicMonsterStatus {
				*emitted++
			}
			return nil
		},
	}
}

// TestFindNextController_EnteringPlayerEmitsStartControl is the regression test
// for the frozen-monsters-on-re-entry bug.
//
// A controller who leaves a map and walks back in used to be assigned in place
// with NO StartControl event, on the theory that the channel's map-enter spawn
// would observe the registry write and send Spawn-then-Control itself. It
// cannot: the channel spawns off EVENT_TOPIC_CHARACTER_STATUS while this
// assignment runs off EVENT_TOPIC_MAP_STATUS, and the channel routinely reads
// the monster list first — before this service has even released the previous
// map. When it lost that race the mob ended up assigned in the registry and
// permanently frozen on the client, because nothing else ever emits the grant.
//
// The assignment must therefore always announce itself. Control-before-Spawn is
// prevented on the channel side instead (controlGrantFn sends Spawn-then-Control).
func TestFindNextController_EnteringPlayerEmitsStartControl(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	const enter = uint32(7)
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50, "", "")
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	m := mons[0]

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	// Entering player (7) is the only field candidate → chosen controller == entering.
	if err := p.FindNextController(model.FixedProvider([]uint32{enter}))(m); err != nil {
		t.Fatalf("FindNextController: %v", err)
	}

	if emitted != 1 {
		t.Fatalf("the entering player's assignment must emit exactly one StartControl event; got %d", emitted)
	}
	got, err := p.GetById(m.UniqueId())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != enter {
		t.Fatalf("expected controlCharacterId=%d; got %d", enter, got.ControlCharacterId())
	}
}

// TestFindNextController_AlreadyPresentPlayerEmitsStartControl covers the other
// candidate shape — an already-present player — which always emitted and must
// keep doing so.
func TestFindNextController_AlreadyPresentPlayerEmitsStartControl(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	const existing = uint32(9)
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50, "", "")
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	m := mons[0]

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.FindNextController(model.FixedProvider([]uint32{existing}))(m); err != nil {
		t.Fatalf("FindNextController: %v", err)
	}

	if emitted != 1 {
		t.Fatalf("already-present controller must emit exactly one StartControl event; got %d", emitted)
	}
	got, err := p.GetById(m.UniqueId())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != existing {
		t.Fatalf("expected controlCharacterId=%d; got %d", existing, got.ControlCharacterId())
	}
}

// TestFindNextController_NoAggroDoesNotRepick pins the guard that keeps the
// map-enter path quiet, which routing the entering player through StartControl
// now depends on.
//
// StartControl re-picks the mob's next skill on a controller change, but only
// when the new controller has aggro. Without that gate every mob in a map
// decides a skill the moment a player walks in — the incident where twelve
// freshly-spawned Wyverns all cast skill 126 on entry. RepickAndEmit publishes
// NEXT_SKILL_DECIDED on the same MONSTER_STATUS topic as StartControl, so a
// re-pick here would show up as a second emission.
func TestFindNextController_NoAggroDoesNotRepick(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	const enter = uint32(7)
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50, "", "")
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	m := mons[0]
	if m.ControllerHasAggro() {
		t.Fatalf("a freshly created, undamaged monster must not have controller aggro")
	}

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	if err := p.FindNextController(model.FixedProvider([]uint32{enter}))(m); err != nil {
		t.Fatalf("FindNextController: %v", err)
	}

	// Exactly one: the StartControl. A second would be NEXT_SKILL_DECIDED.
	if emitted != 1 {
		t.Fatalf("a no-aggro assignment must emit only StartControl and no re-pick; got %d emissions", emitted)
	}
}
