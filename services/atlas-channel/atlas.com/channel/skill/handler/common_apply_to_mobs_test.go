package handler

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// applyCall captures one ApplyStatus invocation so tests can assert on it.
type applyCall struct {
	monsterId uint32
	skillId   uint32
	statuses  map[string]int32
	duration  uint32
}

// cancelCall captures one CancelStatus invocation.
type cancelCall struct {
	monsterId uint32
	skillId   uint32
	class     string
}

// fakes wires deterministic seams for one test.
type fakes struct {
	applies []applyCall
	cancels []cancelCall
}

// installFakes replaces all six seam vars with deterministic implementations
// returning the supplied data. `mobs` controls rectQueryFunc; `reflects`
// controls reflectLookupFunc (keyed by monster id, value is the reflect
// kind that should be reported as active); `propWillFire` controls
// propRollFunc; `caster` is what loadCasterFunc returns; `casterErr` /
// `rectErr` short-circuit those seams.
func installFakes(t *testing.T, caster character.Model, casterErr error, mobs []monster.Model, rectErr error, reflects map[uint32]string, propWillFire bool) *fakes {
	t.Helper()
	f := &fakes{}

	prevLoad := loadCasterFunc
	prevRect := rectQueryFunc
	prevProp := propRollFunc
	prevReflect := reflectLookupFunc
	prevApply := applyStatusFunc
	prevCancel := cancelStatusFunc

	loadCasterFunc = func(_ character.Processor, _ uint32) (character.Model, error) {
		if casterErr != nil {
			return character.Model{}, casterErr
		}
		return caster, nil
	}
	rectQueryFunc = func(_ *monster.Processor, _ field.Model, _, _, _, _ int16, _ uint32) ([]monster.Model, error) {
		if rectErr != nil {
			return nil, rectErr
		}
		return mobs, nil
	}
	propRollFunc = func(_ float64) bool { return propWillFire }
	reflectLookupFunc = func(_ tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
		if want, ok := reflects[monsterId]; ok && want == kind {
			return monster.ReflectInfo{Kind: kind, Percent: 30, ExpiresAt: time.Now().Add(time.Minute)}, true
		}
		return monster.ReflectInfo{}, false
	}
	applyStatusFunc = func(_ *monster.Processor, _ field.Model, monsterId, _, skillId, _ uint32, statuses map[string]int32, duration uint32) error {
		f.applies = append(f.applies, applyCall{monsterId: monsterId, skillId: skillId, statuses: statuses, duration: duration})
		return nil
	}
	cancelStatusFunc = func(_ *monster.Processor, _ field.Model, monsterId uint32, _ []string, _, skillId uint32, class string) error {
		f.cancels = append(f.cancels, cancelCall{monsterId: monsterId, skillId: skillId, class: class})
		return nil
	}

	t.Cleanup(func() {
		loadCasterFunc = prevLoad
		rectQueryFunc = prevRect
		propRollFunc = prevProp
		reflectLookupFunc = prevReflect
		applyStatusFunc = prevApply
		cancelStatusFunc = prevCancel
	})
	return f
}

func mkField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
}

func mkMob(uniqueId uint32) monster.Model {
	return monster.NewModelBuilder(uniqueId, mkField(), 9300018).MustBuild()
}

func mkCaster(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).MustBuild()
}

// mkInfo builds a SkillUsageInfo with the given skill id, level, and affected
// mob ids. The wire decoder is exercised in its own test suite — here we
// build through the public Builder.
func mkInfo(skillId uint32, level byte, mobIds []uint32) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(skillId).
		SetSkillLevel(level).
		SetAffectedMobIds(mobIds).
		Build()
}

// withRect returns the effect with non-zero LT/RB so hasEffectBbox is true.
func withRect(rm effect.RestModel) effect.RestModel {
	rm.LT = &effect.PointRestModel{X: -200, Y: -100}
	rm.RB = &effect.PointRestModel{X: 200, Y: 100}
	return rm
}

func newDoomEffect(prop float64) effect.Model {
	rm := withRect(effect.RestModel{
		Duration:      60000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
		MobCount:      6,
		Prop:          prop,
	})
	se, _ := effect.Extract(rm)
	return se
}

// newDoomEffectNoBbox is a Doom-shaped effect with no rect (FR-4.2 fallback).
func newDoomEffectNoBbox(prop float64) effect.Model {
	se, _ := effect.Extract(effect.RestModel{
		Duration:      60000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
		MobCount:      6,
		Prop:          prop,
	})
	return se
}

// newCrashEffect models Crusader Armor Crash: cancel branch, no MonsterStatus.
func newCrashEffect(prop float64) effect.Model {
	rm := withRect(effect.RestModel{
		MobCount: 6,
		Prop:     prop,
	})
	se, _ := effect.Extract(rm)
	return se
}

func newCtx(t *testing.T) (context.Context, tenant.Model) {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm
}

func nullLogger() *logrus.Logger {
	l := logrus.New()
	l.Out = io.Discard
	return l
}

// Reference imports so unused-import linters do not fire when individual
// tests are commented out for triage.
var (
	_ = errors.New
	_ point.Model
	_ skill2.Id
)

func TestApplyToMobs_EmptyClientList_NoOp(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil, nil, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001, mkInfo(uint32(skill2.PriestDoomId), 30, nil), newDoomEffect(1.0))
	if len(f.applies) != 0 || len(f.cancels) != 0 {
		t.Fatalf("seam calls = (%d apply, %d cancel), want both 0", len(f.applies), len(f.cancels))
	}
}

func TestApplyToMobs_OverCap_Drops_AndWarns(t *testing.T) {
	// Effect.MobCount = 6 by default in newDoomEffect; client sends 7.
	f := installFakes(t, mkCaster(1001), nil, []monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3, 4, 5, 6, 7}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (over-cap should drop)", len(f.applies))
	}
	if len(f.cancels) != 0 {
		t.Fatalf("cancels = %d, want 0 (over-cap should drop)", len(f.cancels))
	}
}

func TestApplyToMobs_NoBbox_TrustsClient(t *testing.T) {
	// effect with all-zero LT/RB → fallback path; rect query is NOT called.
	rectCalled := false
	prevRect := rectQueryFunc
	t.Cleanup(func() { rectQueryFunc = prevRect })

	f := installFakes(t, mkCaster(1001), nil, nil, nil, nil, true)
	rectQueryFunc = func(_ *monster.Processor, _ field.Model, _, _, _, _ int16, _ uint32) ([]monster.Model, error) {
		rectCalled = true
		return nil, nil
	}

	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffectNoBbox(1.0))

	if rectCalled {
		t.Fatalf("rectQueryFunc called; expected fallback to skip rect query")
	}
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3 (no-bbox fallback applies to client list)", len(f.applies))
	}
}

func TestApplyToMobs_CasterLoadFails_Drops(t *testing.T) {
	f := installFakes(t, mkCaster(1001), errors.New("boom"), nil, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 || len(f.cancels) != 0 {
		t.Fatalf("seam calls = (%d apply, %d cancel), want both 0 on caster-load fail", len(f.applies), len(f.cancels))
	}
}

func TestApplyToMobs_RectQueryFails_Drops(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil, nil, errors.New("boom"), nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 on rect-query fail", len(f.applies))
	}
}

func TestApplyToMobs_RectIntersectionApplied(t *testing.T) {
	// server returns 1, 2, 3; client lists 1, 2, 3, 99 (extra).
	// Expectation: 3 applies (in client order); 99 dropped silently from
	// the applied set (and surfaced in the warn log; we do not assert log
	// content here since the file uses a discarding logger).
	f := installFakes(t, mkCaster(1001), nil, []monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3, 99}),
		newDoomEffect(1.0))
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3", len(f.applies))
	}
	want := []uint32{1, 2, 3}
	for i, c := range f.applies {
		if c.monsterId != want[i] {
			t.Errorf("apply[%d] = %d, want %d", i, c.monsterId, want[i])
		}
	}
}

func TestApplyToMobs_DoomMagicReflectSkipped(t *testing.T) {
	// mob 2 has MAGICAL reflect → Doom must skip it.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil,
		map[uint32]string{2: monster2.ReflectKindMagical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffect(1.0))
	if len(f.applies) != 2 {
		t.Fatalf("applies = %d, want 2", len(f.applies))
	}
	if f.applies[0].monsterId != 1 || f.applies[1].monsterId != 3 {
		t.Errorf("applies = [%d, %d], want [1, 3]", f.applies[0].monsterId, f.applies[1].monsterId)
	}
}

func TestApplyToMobs_CrashFamily_PhysicalReflectSkipped(t *testing.T) {
	// Crusader Armor Crash → cancel branch with PHYSICAL kind.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil,
		map[uint32]string{1: monster2.ReflectKindPhysical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.CrusaderArmorCrashId), 30, []uint32{1, 2}),
		newCrashEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (cancel branch only)", len(f.applies))
	}
	if len(f.cancels) != 1 || f.cancels[0].monsterId != 2 {
		t.Errorf("cancels = %v, want exactly mob 2", f.cancels)
	}
}

func TestApplyToMobs_PriestDispel_MagicalReflectSkipped(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil,
		map[uint32]string{2: monster2.ReflectKindMagical}, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDispelId), 30, []uint32{1, 2}),
		newCrashEffect(1.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (cancel branch)", len(f.applies))
	}
	if len(f.cancels) != 1 || f.cancels[0].monsterId != 1 {
		t.Errorf("cancels = %v, want exactly mob 1", f.cancels)
	}
}

func TestApplyToMobs_PropZero_AppliesNothing(t *testing.T) {
	// propRollFunc is set to "false"; with prop=0, the effect contract is
	// "always skip". applies should be empty.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil, nil, false)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2}),
		newDoomEffect(0.0))
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (prop=0 should skip every mob)", len(f.applies))
	}
}

func TestApplyToMobs_PropOne_AppliesAll(t *testing.T) {
	// prop=1 with propRollFunc="true" should apply every in-rect mob.
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2), mkMob(3)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1, 2, 3}),
		newDoomEffect(1.0))
	if len(f.applies) != 3 {
		t.Fatalf("applies = %d, want 3 (prop=1 should pass all)", len(f.applies))
	}
}

func TestApplyToMobs_DoomTakesApplyBranch(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{1}),
		newDoomEffect(1.0))
	if len(f.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(f.applies))
	}
	if len(f.cancels) != 0 {
		t.Fatalf("cancels = %d, want 0 (Doom must not take cancel branch)", len(f.cancels))
	}
}

func TestApplyToMobs_CrashTakesCancelBranch(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.CrusaderArmorCrashId), 30, []uint32{1}),
		newCrashEffect(1.0))
	if len(f.cancels) != 1 {
		t.Fatalf("cancels = %d, want 1", len(f.cancels))
	}
	if f.cancels[0].class != "PHYSICAL" {
		t.Errorf("cancel class = %q, want PHYSICAL", f.cancels[0].class)
	}
	if len(f.applies) != 0 {
		t.Fatalf("applies = %d, want 0 (Crash must not take apply branch)", len(f.applies))
	}
}

func TestApplyToMobs_PropCarveOutSuppressesPropOnCancel(t *testing.T) {
	// Install a deny entry for Crusader Armor Crash on the cancel branch.
	// With propRollFunc="false" the cast would normally produce zero
	// cancels; the carve-out flips that to "always pass".
	id := skill2.CrusaderArmorCrashId
	prev := propCarveOut[id]
	propCarveOut[id] = map[propBranch]bool{propBranchCancel: false}
	t.Cleanup(func() {
		if prev == nil {
			delete(propCarveOut, id)
		} else {
			propCarveOut[id] = prev
		}
	})

	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(1), mkMob(2)}, nil, nil, false /* propWillFire */)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(id), 30, []uint32{1, 2}),
		newCrashEffect(0.0)) // prop=0 would force-skip if rolled
	if len(f.cancels) != 2 {
		t.Fatalf("cancels = %d, want 2 (carve-out should bypass prop)", len(f.cancels))
	}
}

func TestApplyToMobs_PassesDoomStatusAndDuration(t *testing.T) {
	f := installFakes(t, mkCaster(1001), nil,
		[]monster.Model{mkMob(99)}, nil, nil, true)
	ctx, _ := newCtx(t)
	applyToMobs(nullLogger(), ctx, mkField(), 1001,
		mkInfo(uint32(skill2.PriestDoomId), 30, []uint32{99}),
		newDoomEffect(1.0))
	if len(f.applies) != 1 {
		t.Fatalf("applies = %d, want 1", len(f.applies))
	}
	got := f.applies[0]
	if got.statuses[monster2.StatusDoom] != 1 {
		t.Errorf("statuses[DOOM] = %d, want 1", got.statuses[monster2.StatusDoom])
	}
	if got.duration != 60000 {
		t.Errorf("duration = %d, want 60000", got.duration)
	}
	if got.skillId != uint32(skill2.PriestDoomId) {
		t.Errorf("skillId = %d, want %d", got.skillId, uint32(skill2.PriestDoomId))
	}
}
