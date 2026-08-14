package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// These tests pin the attack-packet cast dispatcher (task-200). The defect
// they guard against is a silent structural one: Poison Mist's handler was
// registered on the use-skill registry, which processAttack reads but never
// invokes, so casting the skill produced the direct magic-attack damage and
// nothing else -- no mist, no poison, and not a single log line to say so.
//
// attackCastTryApply is the extracted seam (same shape as beaconTryApply
// above it); the one call line inside processAttack's closure is not covered
// here, matching how every other fire-and-forget hook in that file is tested.

type attackCastCall struct {
	characterId uint32
	wireSkillId skill2.Id
	skillLevel  byte
	duration    int32
	castOrigin  *point.Model
}

// registerAttackCastSpy installs a recording handler under id and removes it
// on cleanup. If retErr is non-nil the handler returns it, exercising the
// swallow path.
func registerAttackCastSpy(t *testing.T, id skill2.Identity, retErr error) *[]attackCastCall {
	t.Helper()
	calls := make([]attackCastCall, 0)
	channelhandler.RegisterAttackCast(id, func(_ logrus.FieldLogger) func(_ context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		skillId skill2.Id, skillLevel byte, e effect.Model, castOrigin *point.Model,
	) error {
		return func(_ context.Context) func(
			writer.Producer, field.Model, uint32, skill2.Id, byte, effect.Model, *point.Model,
		) error {
			return func(_ writer.Producer, _ field.Model, characterId uint32,
				skillId skill2.Id, skillLevel byte, e effect.Model, castOrigin *point.Model,
			) error {
				calls = append(calls, attackCastCall{
					characterId: characterId,
					wireSkillId: skillId,
					skillLevel:  skillLevel,
					duration:    e.Duration(),
					castOrigin:  castOrigin,
				})
				return retErr
			}
		}
	})
	t.Cleanup(func() { channelhandler.UnregisterAttackCast(id) })
	return &calls
}

func attackCastTestEffect(t *testing.T, durationMs int32) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{Duration: durationMs})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func attackCastTestField() field.Model {
	return field.NewBuilder(0, 0, 100000000).Build()
}

// TestAttackCastTryApply_Registered_InvokesHandler is the direct regression
// test for the bug: a skill delivered on an attack packet must actually reach
// its registered handler.
func TestAttackCastTryApply_Registered_InvokesHandler(t *testing.T) {
	id := skill2.Identity(900900901)
	calls := registerAttackCastSpy(t, id, nil)

	l, _ := test.NewNullLogger()
	attackCastTryApply(l, context.Background(), nil, attackCastTestField(), 1001, id,
		skill2.Id(2111003), 7, attackCastTestEffect(t, 4000), nil)

	if len(*calls) != 1 {
		t.Fatalf("handler invoked %d times, want 1", len(*calls))
	}
	got := (*calls)[0]
	if got.characterId != 1001 {
		t.Errorf("characterId = %d, want 1001", got.characterId)
	}
	// The WIRE id must be forwarded, not the Identity the registry is keyed
	// on: handlers put it back on the wire for the client to match against
	// its own WZ.
	if got.wireSkillId != skill2.Id(2111003) {
		t.Errorf("wireSkillId = %d, want 2111003", got.wireSkillId)
	}
	if got.skillLevel != 7 {
		t.Errorf("skillLevel = %d, want 7", got.skillLevel)
	}
	// The effect must be the caster's resolved one — a mist handler reads
	// duration/lt/rb off it and rejects the cast when they are zero.
	if got.duration != 4000 {
		t.Errorf("effect duration = %d, want 4000", got.duration)
	}
}

// TestAttackCastTryApply_Unregistered_NoOp pins that ordinary attack skills
// (the overwhelming majority, none of which register here) cost nothing.
func TestAttackCastTryApply_Unregistered_NoOp(t *testing.T) {
	l, hook := test.NewNullLogger()
	attackCastTryApply(l, context.Background(), nil, attackCastTestField(), 1001,
		skill2.Identity(900900902), skill2.Id(2001005), 1, attackCastTestEffect(t, 0), nil)

	if n := len(hook.AllEntries()); n != 0 {
		t.Fatalf("unregistered identity logged %d entries, want 0: %v", n, hook.AllEntries())
	}
}

// TestAttackCastTryApply_HandlerError_SwallowedAndLogged pins the
// fire-and-forget posture: damage is already applied and the attack already
// broadcast by the time this runs, so a handler failure must never propagate.
func TestAttackCastTryApply_HandlerError_SwallowedAndLogged(t *testing.T) {
	id := skill2.Identity(900900903)
	calls := registerAttackCastSpy(t, id, errors.New("kafka down"))

	l, hook := test.NewNullLogger()
	attackCastTryApply(l, context.Background(), nil, attackCastTestField(), 1001, id,
		skill2.Id(2111003), 1, attackCastTestEffect(t, 4000), nil)

	if len(*calls) != 1 {
		t.Fatalf("handler invoked %d times, want 1", len(*calls))
	}
	if len(hook.AllEntries()) == 0 {
		t.Fatal("handler error was swallowed silently; want a logged entry")
	}
}

// TestAttackCastOriginPrefersThePacket pins where an ATTACK-delivered cast
// anchors. Both halves are field-report regressions (task-218).
//
// #4: a thrown-grenade skill must use the bomb's landing point, not the
// caster's feet — the throw distance scales with how long the key was held, so
// anchoring at the caster puts the cloud a whole throw away from the explosion
// the client already drew.
//
// #2: every other attack must use the caster position the PACKET carries, not a
// read-back from atlas-character. That service is updated asynchronously over
// Kafka (movement.ProcessorImpl.ForCharacter emits from a detached goroutine),
// so a lookup during this attack can return a pre-move position and anchor the
// cloud where the caster used to be.
func TestAttackCastOriginPrefersThePacket(t *testing.T) {
	t.Run("grenade skill uses the landing point", func(t *testing.T) {
		ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
		ai.SetSkillId(uint32(skill2.NightWalkerStage3PoisonBombId))
		gx, gy := int16(-540), int16(421)
		ai.SetGrenadePosition(uint16(gx), uint16(gy))
		got := attackCastOrigin(*ai)
		if got == nil {
			t.Fatal("grenade attack must supply an origin")
		}
		if int16(got.X()) != -540 || int16(got.Y()) != 421 {
			t.Errorf("origin = (%d,%d), want the grenade landing point (-540,421)", int16(got.X()), int16(got.Y()))
		}
	})

	t.Run("non-grenade skill uses the packet caster position", func(t *testing.T) {
		ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeMagic)
		ai.SetSkillId(uint32(skill2.BlazeWizardStage3FlameGearId))
		cx, cy := int16(-96), int16(422)
		ai.SetCharacterPosition(uint16(cx), uint16(cy))
		got := attackCastOrigin(*ai)
		if got == nil {
			t.Fatal("attack-delivered cast must supply an origin so the async caster lookup is never used")
		}
		if int16(got.X()) != -96 || int16(got.Y()) != 422 {
			t.Errorf("origin = (%d,%d), want the packet caster position (-96,422)", int16(got.X()), int16(got.Y()))
		}
	})
}
