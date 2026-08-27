package movement

import (
	"atlas-channel/character/snapshot"
	"atlas-channel/monster"
	"atlas-channel/monster/information"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	movement2 "atlas-channel/kafka/message/movement"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// drainMovementCommand blocks until ForCharacter's async producer goroutine
// has emitted its command for characterId, so the test returns only after
// that goroutine has finished. sharedCapture (testmain_test.go) is a single
// package-wide singleton, so a still-in-flight goroutine from one test can
// otherwise race a later test's sharedCapture.Reset()/Messages() calls
// (documented on TestTeleportCharacter_EmitsFhZeroOnWire).
func drainMovementCommand(t *testing.T, characterId uint32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		for _, msg := range sharedCapture.Messages(movement2.EnvCommandCharacterMovement) {
			var cmd movement2.Command[any]
			if err := json.Unmarshal(msg.Value, &cmd); err != nil {
				continue
			}
			if cmd.ObjectId == uint64(characterId) {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for movement command for character [%d]", characterId)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestNarrowSkill_HappyPath(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(100, 2)
	if !ok || id != 100 || lvl != 2 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 100 2 true", id, lvl, ok)
	}
}

func TestNarrowSkill_NegativeRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(-1, 1); ok {
		t.Fatalf("expected reject for negative skillId")
	}
	if _, _, ok := narrowSkillBytes(1, -1); ok {
		t.Fatalf("expected reject for negative skillLevel")
	}
}

func TestNarrowSkill_OverflowRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(256, 1); ok {
		t.Fatalf("expected reject for skillId > 255")
	}
	if _, _, ok := narrowSkillBytes(1, 256); ok {
		t.Fatalf("expected reject for skillLevel > 255")
	}
}

func TestNarrowSkill_BoundaryAccepted(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(255, 255)
	if !ok || id != 255 || lvl != 255 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 255 255 true", id, lvl, ok)
	}
}

func TestComputeAckMp_BasicAttackPath_DecrementsByConMp(t *testing.T) {
	atks := []information.AttackInfo{
		{Pos: 2, ConMP: 5, AttackAfter: 1500},
	}
	got := computeAckMp(uint16(100), uint8(1), atks)
	if got != 95 {
		t.Errorf("computeAckMp(100, pos0=1, conMP=5) = %d, want 95", got)
	}
}

func TestComputeAckMp_BasicAttackPath_NoAttackInfo_Untouched(t *testing.T) {
	got := computeAckMp(uint16(100), uint8(0), nil)
	if got != 100 {
		t.Errorf("computeAckMp with no attack info = %d, want 100", got)
	}
}

func TestComputeAckMp_BasicAttackPath_ConMpExceedsMp_ClampsToZero(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 50, AttackAfter: 1500}}
	got := computeAckMp(uint16(10), uint8(0), atks)
	if got != 0 {
		t.Errorf("computeAckMp clamps to zero on overflow, got %d", got)
	}
}

func TestComputeAckMp_BasicAttackPath_PosNotFound_Untouched(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 5, AttackAfter: 1500}}
	got := computeAckMp(uint16(100), uint8(2), atks)
	if got != 100 {
		t.Errorf("computeAckMp with pos not found = %d, want 100", got)
	}
}

func newMovementTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newMovementTestProcessor(t *testing.T) (*ProcessorImpl, tenant.Model) {
	t.Helper()
	tm := newMovementTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	return NewProcessor(logrus.New(), ctx, nil).(*ProcessorImpl), tm
}

func movementTestField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestResolveLiveMonster_WarmPath_ZeroRest(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	monster.GetLiveMirror().Put(tm, 8001, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 44, ControllerHasAggro: true})

	calls := 0
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		calls++
		return monster.Model{}, errors.New("REST must not be called on the warm path")
	}
	defer func() { monsterByIdFn = prev }()

	entry, err := p.resolveLiveMonster(8001)
	if err != nil {
		t.Fatalf("warm path errored: %v", err)
	}
	if calls != 0 {
		t.Fatalf("warm path made %d REST calls, want 0", calls)
	}
	if entry.Mp != 44 || !entry.ControllerHasAggro || entry.MonsterId != 100100 {
		t.Fatalf("entry mismatch: %+v", entry)
	}
}

func TestResolveLiveMonster_MissFallsBackOnceAndBackfills(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	calls := 0
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, objectId uint32) (monster.Model, error) {
		calls++
		return monster.NewModelBuilder(objectId, f, 100100).
			SetMp(70).
			SetMaxMp(90).
			SetControllerHasAggro(true).
			Build()
	}
	defer func() { monsterByIdFn = prev }()

	entry, err := p.resolveLiveMonster(8002)
	if err != nil {
		t.Fatalf("fallback errored: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first resolve made %d REST calls, want exactly 1", calls)
	}
	if entry.Mp != 70 || !entry.ControllerHasAggro {
		t.Fatalf("fallback entry mismatch: %+v", entry)
	}

	// Second resolve must be served from the backfilled mirror.
	if _, err := p.resolveLiveMonster(8002); err != nil {
		t.Fatalf("second resolve errored: %v", err)
	}
	if calls != 1 {
		t.Fatalf("second resolve made a REST call (total %d), want mirror hit", calls)
	}
	if got, ok := monster.GetLiveMirror().Lookup(tm, 8002); !ok || got.Mp != 70 {
		t.Fatalf("fallback must backfill the mirror, got %+v ok=%v", got, ok)
	}
}

func TestResolveLiveMonster_FallbackError_Propagates(t *testing.T) {
	p, tm := newMovementTestProcessor(t)

	wantErr := errors.New("monsters unavailable")
	prev := monsterByIdFn
	monsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, wantErr
	}
	defer func() { monsterByIdFn = prev }()

	if _, err := p.resolveLiveMonster(8003); !errors.Is(err, wantErr) {
		t.Fatalf("fallback error must propagate unchanged, got %v", err)
	}
	if _, ok := monster.GetLiveMirror().Lookup(tm, 8003); ok {
		t.Fatalf("failed fallback must not backfill the mirror")
	}
}

func TestForCharacter_FeedsSnapshotPositionSynchronously(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	// Entry must exist (events/feeds never create entries): simulate the
	// lazy populate by creating the entry, then validate core via backfill
	// so the position lands on a live entry.
	v := snapshot.GetRegistry().View(tm, 9001)
	_ = v

	mv := model.Movement{StartX: 10, StartY: 20}
	// No elements: the fold returns the start position.
	if err := p.ForCharacter(f, 9001, mv); err != nil {
		t.Fatalf("ForCharacter: %v", err)
	}

	got := snapshot.GetRegistry().View(tm, 9001)
	if !got.PosValid || got.PosX != 10 || got.PosY != 20 {
		t.Fatalf("position must be fed synchronously before ForCharacter returns: %+v", got)
	}
	drainMovementCommand(t, 9001)
}

func TestForCharacter_NoEntryNoCreate(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	if err := p.ForCharacter(f, 9002, model.Movement{StartX: 1, StartY: 2}); err != nil {
		t.Fatalf("ForCharacter: %v", err)
	}
	// 9002 was never Viewed/populated. View returns an existing entry
	// unchanged rather than resetting it: if the feed wrongly created an
	// entry it would carry the fed position with PosValid true; if the feed
	// correctly created nothing, View creates a fresh empty entry here with
	// PosValid false.
	got := snapshot.GetRegistry().View(tm, 9002)
	if got.PosValid {
		t.Fatalf("position feed must never create snapshot entries, got %+v", got)
	}
	if _, ok := snapshot.GetRegistry().ComposedIfValid(tm, 9002); ok {
		t.Fatalf("position feed must never create snapshot entries")
	}
	drainMovementCommand(t, 9002)
}

func TestForMonster_FeedsMirrorPosition(t *testing.T) {
	// ForMonster's ack goroutine reads the process-wide skill inbox
	// singleton; init it here since this package has no production
	// bootstrap (main.go does this at startup) and no prior test in this
	// package exercises ForMonster.
	monster.InitNextSkillInbox()
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	monster.GetLiveMirror().Put(tm, 8101, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 3})

	mv := model.Movement{StartX: 44, StartY: -55}
	_ = p.ForMonster(f, 1, 8101, 0, false, 0, 0, 0, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, mv)

	got, ok := monster.GetLiveMirror().Lookup(tm, 8101)
	if !ok || got.X != 44 || got.Y != -55 {
		t.Fatalf("mirror position not fed: %+v ok=%v", got, ok)
	}
}
