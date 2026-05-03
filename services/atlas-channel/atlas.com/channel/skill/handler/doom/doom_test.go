package doom

import (
	"context"
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

// installFakes wires deterministic seams for one test. Returns a cleanup that
// restores the production implementations.
func installFakes(t *testing.T, mobs []monster.Model, reflects map[uint32]monster.ReflectInfo, propWillFire bool) *[]applyCall {
	t.Helper()
	var calls []applyCall

	prevLoad := loadCasterFunc
	prevRect := rectQueryFunc
	prevProp := propRollFunc
	prevReflect := reflectLookupFunc
	prevApply := applyStatusFunc

	loadCasterFunc = func(_ character.Processor, characterId uint32) (character.Model, error) {
		return character.NewModelBuilder().SetId(characterId).Build()
	}
	rectQueryFunc = func(_ *monster.Processor, _ field.Model, _, _, _, _ int16, _ uint32) ([]monster.Model, error) {
		return mobs, nil
	}
	propRollFunc = func(_ float64) bool { return propWillFire }
	reflectLookupFunc = func(_ tenant.Model, monsterId uint32, _ string) (monster.ReflectInfo, bool) {
		info, ok := reflects[monsterId]
		return info, ok
	}
	applyStatusFunc = func(_ *monster.Processor, _ field.Model, monsterId, _, skillId, _ uint32, statuses map[string]int32, duration uint32) error {
		calls = append(calls, applyCall{monsterId: monsterId, skillId: skillId, statuses: statuses, duration: duration})
		return nil
	}

	t.Cleanup(func() {
		loadCasterFunc = prevLoad
		rectQueryFunc = prevRect
		propRollFunc = prevProp
		reflectLookupFunc = prevReflect
		applyStatusFunc = prevApply
	})
	return &calls
}

func mkMob(uniqueId uint32) monster.Model {
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	return monster.NewModelBuilder(uniqueId, f, 9300018).MustBuild()
}

func newDoomEffect(prop float64) effect.Model {
	se, _ := effect.Extract(effect.RestModel{
		Duration:      60000,
		MonsterStatus: map[string]uint32{monster2.StatusDoom: 1},
		MobCount:      6,
		Prop:          prop,
	})
	return se
}

func newSkillUsageInfo() packetmodel.SkillUsageInfo {
	// Zero value is sufficient — the handler only reads SkillLevel(), and
	// the wire decoder is exercised in its own test suite.
	var info packetmodel.SkillUsageInfo
	return info
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

func TestDoom_Apply_AppliesToAllInRectMobs(t *testing.T) {
	mobs := []monster.Model{mkMob(1), mkMob(2), mkMob(3)}
	calls := installFakes(t, mobs, nil, true)

	ctx, _ := newCtx(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	_ = Apply(nullLogger())(ctx)(nil, f, 1001, newSkillUsageInfo(), newDoomEffect(1.0))

	if len(*calls) != 3 {
		t.Fatalf("apply calls = %d, want 3 (mobs %v)", len(*calls), *calls)
	}
}

func TestDoom_Apply_SkipsMagicReflectMobs(t *testing.T) {
	mobs := []monster.Model{mkMob(1), mkMob(2), mkMob(3)}
	reflects := map[uint32]monster.ReflectInfo{
		2: {Kind: monster2.ReflectKindMagical, Percent: 30, ExpiresAt: time.Now().Add(time.Minute)},
	}
	calls := installFakes(t, mobs, reflects, true)

	ctx, _ := newCtx(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	_ = Apply(nullLogger())(ctx)(nil, f, 1001, newSkillUsageInfo(), newDoomEffect(1.0))

	if len(*calls) != 2 {
		t.Fatalf("apply calls = %d, want 2 (reflect on mob 2 skips)", len(*calls))
	}
	if (*calls)[0].monsterId != 1 || (*calls)[1].monsterId != 3 {
		t.Errorf("apply targets = [%d, %d], want [1, 3]", (*calls)[0].monsterId, (*calls)[1].monsterId)
	}
}

func TestDoom_Apply_RespectsPropZero(t *testing.T) {
	mobs := []monster.Model{mkMob(1), mkMob(2)}
	calls := installFakes(t, mobs, nil /*propWillFire*/, false)

	ctx, _ := newCtx(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	_ = Apply(nullLogger())(ctx)(nil, f, 1001, newSkillUsageInfo(), newDoomEffect(0.0))

	if len(*calls) != 0 {
		t.Fatalf("apply calls = %d, want 0 (prop=0 should skip every mob)", len(*calls))
	}
}

func TestDoom_Apply_PassesDoomStatusAndDuration(t *testing.T) {
	mobs := []monster.Model{mkMob(99)}
	calls := installFakes(t, mobs, nil, true)

	ctx, _ := newCtx(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	_ = Apply(nullLogger())(ctx)(nil, f, 1001, newSkillUsageInfo(), newDoomEffect(1.0))

	if len(*calls) != 1 {
		t.Fatalf("apply calls = %d, want 1", len(*calls))
	}
	got := (*calls)[0]
	if got.statuses[monster2.StatusDoom] != 1 {
		t.Errorf("statuses[DOOM] = %d, want 1", got.statuses[monster2.StatusDoom])
	}
	if got.duration != 60000 {
		t.Errorf("duration = %d, want 60000", got.duration)
	}
	if got.skillId != 2311005 {
		t.Errorf("skillId = %d, want 2311005", got.skillId)
	}
}
