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
// emissions, so a test can assert whether ControlOnEnter emitted a StartControl.
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

// TestControlOnEnter_EnteringPlayerAssignedInPlaceNoEmit is the crash/fall-through
// fix: when the entering character is the chosen controller of a previously
// uncontrolled mob, the assignment is applied in-place WITHOUT emitting a
// StartControl event. No early MonsterControl packet is produced for the
// still-loading client, so the channel's spawnMonsterForSession sends
// Spawn-then-Control and the client never materializes the mob from a Control
// body (0/1-stance crash + slope fall-through).
func TestControlOnEnter_EnteringPlayerAssignedInPlaceNoEmit(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	const enter = uint32(7)
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	m := mons[0]

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	// Entering player (7) is the only field candidate → chosen controller == entering.
	if err := p.ControlOnEnter(enter, model.FixedProvider([]uint32{enter}))(m); err != nil {
		t.Fatalf("ControlOnEnter: %v", err)
	}

	if emitted != 0 {
		t.Fatalf("entering player must be assigned in-place with NO StartControl event; got %d emissions", emitted)
	}
	got, err := p.GetById(m.UniqueId())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.ControlCharacterId() != enter {
		t.Fatalf("expected controlCharacterId=%d assigned in-place; got %d", enter, got.ControlCharacterId())
	}
}

// TestControlOnEnter_AlreadyPresentPlayerEmitsStartControl verifies the other
// branch: when the chosen controller is an already-present player (not the
// entering one), the normal StartControl path — with event emission — is used,
// because that client already has the mob spawned and Control-first is safe.
func TestControlOnEnter_AlreadyPresentPlayerEmitsStartControl(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	const enter = uint32(7)
	const existing = uint32(9)
	r.CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	mons := r.GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	m := mons[0]

	emitted := 0
	p := recordingProcessor(ctx, tm, &emitted)

	// Candidate pool = [existing(9)]; entering is 7 → chosen controller (9) != entering.
	if err := p.ControlOnEnter(enter, model.FixedProvider([]uint32{existing}))(m); err != nil {
		t.Fatalf("ControlOnEnter: %v", err)
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
