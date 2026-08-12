package monstermagnet

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"io"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const casterId = uint32(1)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testField() field.Model {
	return field.NewBuilder(0, 0, 1).Build()
}

// magnetEffect builds the WZ effect for a level-30 magnet: mobCount 7,
// range 450 (design section 3). rangeValue 0 exercises the cap-only fallback.
func magnetEffect(t *testing.T, mobCount uint32, skillRange int32) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{MobCount: mobCount, Range: skillRange})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func magnetInfo(grabs ...packetmodel.MagnetGrab) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.HeroMonsterMagnetId)).
		SetSkillLevel(30).
		SetMagnetGrabs(grabs).
		Build()
}

// call records one seam invocation so tests can assert both counts and order.
type call struct {
	kind      string // "announce" | "clear" | "force"
	monsterId uint32
	result    byte
	success   byte
}

// stubs installs all five seams and returns the recorded call log. rectIds is
// what the rect query reports as present server-side; a nil rectErr/casterErr
// means the corresponding seam succeeds.
func stubs(t *testing.T, rectIds []uint32, casterErr, rectErr error, rectCalls *int) *[]call {
	t.Helper()
	origCaster, origRect := loadCasterFunc, rectQueryFunc
	origAnnounce, origClear, origForce := announceCatchFunc, clearAggroFunc, forceControlFunc
	t.Cleanup(func() {
		loadCasterFunc, rectQueryFunc = origCaster, origRect
		announceCatchFunc, clearAggroFunc, forceControlFunc = origAnnounce, origClear, origForce
	})

	var calls []call

	loadCasterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
		if casterErr != nil {
			return character.Model{}, casterErr
		}
		// stance defaults to 0 (facing right); the character builder has no
		// SetStance, and MagnetRegion's left-facing branch is covered by
		// TestMagnetRegionFacingLeftMirrors in skill/handler.
		return character.NewModelBuilder().SetId(characterId).SetX(1000).SetY(500).MustBuild(), nil
	}
	rectQueryFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
		if rectCalls != nil {
			*rectCalls++
		}
		if rectErr != nil {
			return nil, rectErr
		}
		mobs := make([]monster.Model, 0, len(rectIds))
		for _, id := range rectIds {
			// monster.NewModelBuilder takes (uniqueId, field, monsterId).
			m, berr := monster.NewModelBuilder(id, testField(), 9000000).Build()
			if berr != nil {
				t.Fatalf("monster.NewModelBuilder(%d): %v", id, berr)
			}
			mobs = append(mobs, m)
		}
		return mobs, nil
	}
	announceCatchFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, cid uint32, monsterId uint32) error {
		calls = append(calls, call{kind: "announce", monsterId: monsterId, result: grabResultSuccess, success: grabSuccessFlag})
		return nil
	}
	clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
		calls = append(calls, call{kind: "clear", monsterId: monsterId})
		return nil
	}
	forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
		calls = append(calls, call{kind: "force", monsterId: monsterId})
		return nil
	}
	return &calls
}

func countKind(calls []call, kind string) int {
	n := 0
	for _, c := range calls {
		if c.kind == kind {
			n++
		}
	}
	return n
}

func TestMagnetHappyPathEmitsInOrder(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got := *calls
	if len(got) != 6 {
		t.Fatalf("recorded %d calls, want 6 (announce+clear+force per monster): %+v", len(got), got)
	}
	want := []call{
		{kind: "announce", monsterId: 1001, result: 1, success: 1},
		{kind: "clear", monsterId: 1001},
		{kind: "force", monsterId: 1001},
		{kind: "announce", monsterId: 1002, result: 1, success: 1},
		{kind: "clear", monsterId: 1002},
		{kind: "force", monsterId: 1002},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d = %+v, want %+v — CLEAR_AGGRO must precede FORCE_CONTROL per monster", i, got[i], want[i])
		}
	}
}

func TestMagnetSkipsFailedGrabsAndReleasedSlots(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002, 1003}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(
			packetmodel.NewMagnetGrab(1001, true),
			packetmodel.NewMagnetGrab(1002, false), // client reports a failed grab
			packetmodel.NewMagnetGrab(0, true),     // released slot
		),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	for _, c := range *calls {
		if c.monsterId != 1001 {
			t.Fatalf("acted on monster [%d]; only the successful, non-zero grab may be acted on: %+v", c.monsterId, *calls)
		}
	}
	if countKind(*calls, "announce") != 1 || countKind(*calls, "clear") != 1 || countKind(*calls, "force") != 1 {
		t.Fatalf("expected exactly one of each call kind, got %+v", *calls)
	}
}

func TestMagnetOverCapRejectsWholeCast(t *testing.T) {
	calls := stubs(t, []uint32{1001, 1002, 1003, 1004}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(
			packetmodel.NewMagnetGrab(1001, true),
			packetmodel.NewMagnetGrab(1002, true),
			packetmodel.NewMagnetGrab(1003, true),
			packetmodel.NewMagnetGrab(1004, true),
		),
		magnetEffect(t, 3, 450)) // cap 3, four claimed
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 — an over-cap cast grabs NOTHING (FR-2.2): %+v", len(*calls), *calls)
	}
}

func TestMagnetOutOfRegionDropsIndividually(t *testing.T) {
	// The server sees only 1001; the client also claimed 1002.
	calls := stubs(t, []uint32{1001}, nil, nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(*calls) != 3 {
		t.Fatalf("recorded %d calls, want 3 — the in-region monster proceeds, the other is dropped: %+v", len(*calls), *calls)
	}
	for _, c := range *calls {
		if c.monsterId != 1001 {
			t.Fatalf("acted on out-of-region monster [%d]", c.monsterId)
		}
	}
}

func TestMagnetCasterLoadFailureDropsWholeCast(t *testing.T) {
	calls := stubs(t, []uint32{1001}, errors.New("boom"), nil, nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply must return nil even on a dropped cast: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 (FR-2.7): %+v", len(*calls), *calls)
	}
}

func TestMagnetRectQueryFailureDropsWholeCast(t *testing.T) {
	calls := stubs(t, nil, nil, errors.New("boom"), nil)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)),
		magnetEffect(t, 7, 450))
	if err != nil {
		t.Fatalf("Apply must return nil even on a dropped cast: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("recorded %d calls, want 0 (FR-2.7): %+v", len(*calls), *calls)
	}
}

// TestMagnetIssuesExactlyOneRectQuery pins the NFR performance contract: one
// rect query per cast, not one lookup per claimed monster.
func TestMagnetIssuesExactlyOneRectQuery(t *testing.T) {
	rectCalls := 0
	ids := []uint32{1001, 1002, 1003, 1004, 1005}
	stubs(t, ids, nil, nil, &rectCalls)

	grabs := make([]packetmodel.MagnetGrab, 0, len(ids))
	for _, id := range ids {
		grabs = append(grabs, packetmodel.NewMagnetGrab(id, true))
	}

	if err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(grabs...), magnetEffect(t, 7, 450)); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if rectCalls != 1 {
		t.Fatalf("issued %d rect queries, want exactly 1 for a 5-monster cast", rectCalls)
	}
}

func TestMagnetNoRangeFallsBackToCapOnly(t *testing.T) {
	rectCalls := 0
	calls := stubs(t, nil, nil, nil, &rectCalls)

	err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true), packetmodel.NewMagnetGrab(1002, true)),
		magnetEffect(t, 7, 0)) // no range in this tenant's WZ data
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if rectCalls != 0 {
		t.Fatalf("issued %d rect queries with no range, want 0", rectCalls)
	}
	if countKind(*calls, "force") != 2 {
		t.Fatalf("expected both capped grabs to proceed on the cap-only path, got %+v", *calls)
	}
}

func TestMagnetReturnsNilWhenCommandsFail(t *testing.T) {
	stubs(t, []uint32{1001}, nil, nil, nil)
	origClear, origForce := clearAggroFunc, forceControlFunc
	t.Cleanup(func() { clearAggroFunc, forceControlFunc = origClear, origForce })
	clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
		return errors.New("kafka down")
	}
	forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
		return errors.New("kafka down")
	}

	// Apply must still return nil: a non-nil return makes UseSkill log a second
	// error, and the caller's EnableActions unlock must never be aborted.
	if err := Apply(tl())(context.Background())(nil, testField(), casterId,
		magnetInfo(packetmodel.NewMagnetGrab(1001, true)), magnetEffect(t, 7, 450)); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
}

func TestMagnetRegisteredOnAllThreeIdentities(t *testing.T) {
	for _, id := range []skill2.Identity{
		skill2.HeroMonsterMagnet,
		skill2.PaladinMonsterMagnet,
		skill2.DarkKnightMonsterMagnet,
	} {
		if h, ok := channelhandler.Lookup(id); !ok || h == nil {
			t.Fatalf("no handler registered for identity [%v]", id)
		}
	}
}
