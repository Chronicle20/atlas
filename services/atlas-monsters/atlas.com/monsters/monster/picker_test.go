package monster

import (
	"atlas-monsters/monster/information"
	"atlas-monsters/monster/mobskill"
	"context"
	"errors"
	"testing"
	"time"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type fakeRand struct {
	values []int
	idx    int
}

func (f *fakeRand) Intn(n int) int {
	if f.idx >= len(f.values) {
		return 0
	}
	v := f.values[f.idx]
	f.idx++
	return v
}

type fakeCooldown struct {
	on        map[byte]bool
	remaining map[byte]time.Duration
}

func (f *fakeCooldown) IsOnCooldown(_ context.Context, _ tenant.Model, _ uint32, skillId byte) bool {
	return f.on[skillId]
}

func (f *fakeCooldown) Remaining(_ context.Context, _ tenant.Model, _ uint32, skillId byte) time.Duration {
	return f.remaining[skillId]
}

func newPickerLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func skillsOnly(skills []information.Skill) monsterInfoFetcher {
	// Builds a minimal information.Model carrying only the Skills field.
	// information.Model has private fields; use the existing builder if
	// available, otherwise construct via a helper. For tests we synthesize a
	// model by leaning on the Extract pipeline used in production.
	return func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSkills(skills).Build(), nil
	}
}

func mobSkillTable(table map[uint32]mobskill.Model) mobSkillFetcher {
	return func(id, lvl uint16) (mobskill.Model, error) {
		k := uint32(id)*1000 + uint32(lvl)
		if m, ok := table[k]; ok {
			return m, nil
		}
		return mobskill.Model{}, errors.New("not found")
	}
}

func mskill(t *testing.T, id, lvl uint16, prop, mpCon, hp uint32, interval uint32) mobskill.Model {
	t.Helper()
	return mobskill.NewModelBuilder().
		SetSkillId(id).SetLevel(lvl).
		SetProp(prop).SetMpCon(mpCon).SetHp(hp).SetInterval(interval).
		Build()
}

func newPickerTestMonster(t *testing.T, hp, mp uint32) Model {
	t.Helper()
	return NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, hp, mp)
}

func TestPicker_EmptySkillList_ReturnsSentinel(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(nil), mobSkillTable(nil),
		&fakeCooldown{}, &fakeRand{}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel; got %+v", d)
	}
}

func TestPicker_SealedMonster_ReturnsSentinel(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	m = Clone(m).AddStatusEffect(NewStatusEffect("MONSTER_SKILL", 0, 100, 1, map[string]int32{"SEAL": 1}, time.Minute, 0)).Build()

	skills := []information.Skill{{Id: 100, Level: 1}}
	skillTable := map[uint32]mobskill.Model{100*1000 + 1: mskill(t, 100, 1, 100, 0, 0, 0)}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{0}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel for sealed monster; got %+v", d)
	}
}

func TestPicker_HpThresholdGated_Skipped(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50) // HP 100% / max 100
	skills := []information.Skill{{Id: 100, Level: 1}}
	// hp threshold 30 means skill is only eligible at <= 30% HP; we're at 100%.
	skillTable := map[uint32]mobskill.Model{100*1000 + 1: mskill(t, 100, 1, 100, 0, 30, 0)}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{0}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel for HP-gated skill; got %+v", d)
	}
}

func TestPicker_MpInsufficient_Skipped(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 5) // mp = 5, skill needs 10
	skills := []information.Skill{{Id: 100, Level: 1}}
	skillTable := map[uint32]mobskill.Model{100*1000 + 1: mskill(t, 100, 1, 100, 10, 0, 0)}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{0}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel for MP-gated skill; got %+v", d)
	}
}

func TestPicker_CooldownGated_NextEligibleRepickAtSet(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{{Id: 100, Level: 1}}
	skillTable := map[uint32]mobskill.Model{100*1000 + 1: mskill(t, 100, 1, 100, 0, 0, 5)}

	cd := &fakeCooldown{
		on:        map[byte]bool{100: true},
		remaining: map[byte]time.Duration{100: 3 * time.Second},
	}

	now := int64(1_000_000)
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		cd, &fakeRand{values: []int{0}}, now)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel decision; got %+v", d)
	}
	if d.NextEligibleRepickAtMs != now+3000 {
		t.Fatalf("NextEligibleRepickAtMs=%d, want %d", d.NextEligibleRepickAtMs, now+3000)
	}
}

func TestPicker_AreaPoisonExcluded(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{{Id: uint32(monster2.SkillTypeAreaPoison), Level: 1}}
	skillTable := map[uint32]mobskill.Model{
		uint32(monster2.SkillTypeAreaPoison)*1000 + 1: mskill(t, monster2.SkillTypeAreaPoison, 1, 100, 0, 0, 0),
	}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{0}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel; AREA_POISON should be excluded; got %+v", d)
	}
}

func TestPicker_ByteOverflow_Skipped(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{{Id: 65536, Level: 1}}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(nil),
		&fakeCooldown{}, &fakeRand{values: []int{0}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel for byte-overflow; got %+v", d)
	}
}

func TestPicker_FirstHit_Wins(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{
		{Id: 100, Level: 1},
		{Id: 101, Level: 1},
	}
	skillTable := map[uint32]mobskill.Model{
		100*1000 + 1: mskill(t, 100, 1, 100, 0, 0, 0),
		101*1000 + 1: mskill(t, 101, 1, 100, 0, 0, 0),
	}

	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{50, 50}}, 1000)
	if d.SkillId != 100 {
		t.Fatalf("expected first-eligible (100) to win; got %d", d.SkillId)
	}
	if d.SkillLevel != 1 {
		t.Fatalf("expected level 1; got %d", d.SkillLevel)
	}
}

func TestPicker_PropFails_NoSkill(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{{Id: 100, Level: 1}}
	skillTable := map[uint32]mobskill.Model{100*1000 + 1: mskill(t, 100, 1, 25, 0, 0, 0)}

	// rand returns 50, prop = 25, so 50 < 25 is false ⇒ skipped.
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{50}}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel when prop roll fails; got %+v", d)
	}
}

func TestPicker_NextEligibleMinimumAcrossCooldownGated(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{
		{Id: 100, Level: 1},
		{Id: 101, Level: 1},
	}
	skillTable := map[uint32]mobskill.Model{
		100*1000 + 1: mskill(t, 100, 1, 100, 0, 0, 0),
		101*1000 + 1: mskill(t, 101, 1, 100, 0, 0, 0),
	}
	cd := &fakeCooldown{
		on:        map[byte]bool{100: true, 101: true},
		remaining: map[byte]time.Duration{100: 8 * time.Second, 101: 3 * time.Second},
	}
	now := int64(1_000_000)
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		cd, &fakeRand{values: []int{0, 0}}, now)
	if d.NextEligibleRepickAtMs != now+3000 {
		t.Fatalf("expected min-cooldown expiry %d; got %d", now+3000, d.NextEligibleRepickAtMs)
	}
}

func TestPicker_InfoFetchError_ReturnsSentinel(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	errInfo := func(_ uint32) (information.Model, error) {
		return information.Model{}, errors.New("atlas-data down")
	}
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		errInfo, mobSkillTable(nil),
		&fakeCooldown{}, &fakeRand{}, 1000)
	if !d.IsSentinel() {
		t.Fatalf("expected sentinel on info fetch error; got %+v", d)
	}
}

func TestPicker_SecondSkillWinsAfterFirstPropFails(t *testing.T) {
	tm := newTestTenant(t)
	m := newPickerTestMonster(t, 100, 50)
	skills := []information.Skill{
		{Id: 100, Level: 1},
		{Id: 101, Level: 1},
	}
	skillTable := map[uint32]mobskill.Model{
		100*1000 + 1: mskill(t, 100, 1, 10, 0, 0, 0),  // 10% prop
		101*1000 + 1: mskill(t, 101, 1, 100, 0, 0, 0), // 100% prop
	}

	// First roll = 95: 95 < 10 false, skill 100 skipped.
	// Second roll = 0:  0 < 100 true, skill 101 wins.
	d := pickNextSkill(newPickerLogger(), context.Background(), tm, m,
		skillsOnly(skills), mobSkillTable(skillTable),
		&fakeCooldown{}, &fakeRand{values: []int{95, 0}}, 1000)
	if d.SkillId != 101 {
		t.Fatalf("expected skill 101 to win; got %d", d.SkillId)
	}
}

func TestRepickAndEmit_AlwaysEmits(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	GetMonsterRegistry().CreateMonster(ctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	mons := GetMonsterRegistry().GetMonstersInMap(tm, testField())
	if len(mons) != 1 {
		t.Fatalf("expected 1 monster; got %d", len(mons))
	}
	uniqueId := mons[0].UniqueId()

	emitted := 0
	p := &ProcessorImpl{
		l:   newPickerLogger(),
		ctx: ctx,
		t:   tm,
		emit: func(topic string, _ model.Provider[[]kafka.Message]) error {
			if topic == EnvEventTopicMonsterStatus {
				emitted++
			}
			return nil
		},
	}
	if err := p.repickAndEmit(uniqueId, RepickReasonSpawn); err != nil {
		t.Fatalf("repickAndEmit: %v", err)
	}
	if emitted != 1 {
		t.Fatalf("expected 1 emission (always-emit); got %d", emitted)
	}
}
