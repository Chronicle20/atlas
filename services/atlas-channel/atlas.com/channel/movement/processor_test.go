package movement

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/monster"
	"atlas-channel/monster/information"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

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
