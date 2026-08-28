package monster

import (
	"atlas-monsters/monster/mobskill"
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// TestExecuteStatBuff_Carnival_NegativeXSurvives pins FR-4.3: a negative X
// value from the mob skill (e.g. the Lich AVOIDABILITY debuff, skill 155
// level 2) must survive executeStatBuff unmodified — not clamped, not
// normalized to zero or its absolute value.
func TestExecuteStatBuff_Carnival_NegativeXSurvives(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	sd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalEVA).
		SetLevel(2).
		SetX(-990).
		SetDuration(180_000).
		Build()

	p.executeStatBuff(m, sd, byte(monster2.SkillTypeCarnivalEVA), 2)

	got, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if len(got.StatusEffects()) != 1 {
		t.Fatalf("expected 1 status effect, got %d", len(got.StatusEffects()))
	}
	se := got.StatusEffects()[0]
	if v := se.Statuses()[string(monster2.TemporaryStatTypeAvoidability)]; v != -990 {
		t.Errorf("AVOIDABILITY = %d, want -990", v)
	}
}

// TestExecuteStatBuff_Carnival_DurationIsMilliseconds pins FR-4.4: atlas-data
// ingests the WZ time=1200 (seconds) as 1_200_000 (ms), and executeStatBuff
// must interpret sd.Duration() as milliseconds without any further rescaling.
func TestExecuteStatBuff_Carnival_DurationIsMilliseconds(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	sd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalPAD).
		SetLevel(1).
		SetX(40).
		SetDuration(1_200_000).
		Build()

	p.executeStatBuff(m, sd, byte(monster2.SkillTypeCarnivalPAD), 1)

	got, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if len(got.StatusEffects()) != 1 {
		t.Fatalf("expected 1 status effect, got %d", len(got.StatusEffects()))
	}
	se := got.StatusEffects()[0]
	if se.Duration() != 20*time.Minute {
		t.Errorf("Duration() = %v, want %v", se.Duration(), 20*time.Minute)
	}
}

// TestExecuteStatBuff_Carnival_RecastRefreshesValueAndExpiry pins FR-4.1
// (design D4): recasting the same carnival skill on the same monster
// refreshes the existing status effect's value and expiry rather than
// stacking a second effect, because Builder.AddStatusEffect removes any
// existing effect of the same status type before appending.
func TestExecuteStatBuff_Carnival_RecastRefreshesValueAndExpiry(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	m := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	firstSd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalPAD).
		SetLevel(1).
		SetX(40).
		SetDuration(60_000).
		Build()

	p.executeStatBuff(m, firstSd, byte(monster2.SkillTypeCarnivalPAD), 1)

	afterFirst, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if len(afterFirst.StatusEffects()) != 1 {
		t.Fatalf("expected 1 status effect after first cast, got %d", len(afterFirst.StatusEffects()))
	}
	firstSe := afterFirst.StatusEffects()[0]
	if v := firstSe.Statuses()[string(monster2.TemporaryStatTypePowerUp)]; v != 40 {
		t.Fatalf("POWER_UP after first cast = %d, want 40", v)
	}
	firstExpiresAt := firstSe.ExpiresAt()

	secondSd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalPAD).
		SetLevel(1).
		SetX(99).
		SetDuration(120_000).
		Build()

	// Re-read the model from the registry between casts.
	m2, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	p.executeStatBuff(m2, secondSd, byte(monster2.SkillTypeCarnivalPAD), 1)

	afterSecond, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if len(afterSecond.StatusEffects()) != 1 {
		t.Fatalf("expected exactly 1 status effect after recast, got %d", len(afterSecond.StatusEffects()))
	}
	secondSe := afterSecond.StatusEffects()[0]
	if v := secondSe.Statuses()[string(monster2.TemporaryStatTypePowerUp)]; v != 99 {
		t.Errorf("POWER_UP after recast = %d, want 99", v)
	}
	if secondSe.Duration() != 2*time.Minute {
		t.Errorf("Duration() after recast = %v, want %v", secondSe.Duration(), 2*time.Minute)
	}
	if !secondSe.ExpiresAt().After(firstExpiresAt) {
		t.Errorf("ExpiresAt() after recast = %v, want strictly after first ExpiresAt() %v", secondSe.ExpiresAt(), firstExpiresAt)
	}
}

// TestExecuteStatBuff_Carnival_NoBoundingBox_CasterOnly pins FR-5.2: a mob
// skill definition built without SetBoundingBox has HasBoundingBox()==false,
// so executeStatBuff's AoE loop never runs and only the caster receives the
// buff, even when another monster is in the same field.
func TestExecuteStatBuff_Carnival_NoBoundingBox_CasterOnly(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	caster := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50, "", "")
	other := r.CreateMonster(ctx, tm, f, 9300018, 30, 10, 0, 5, 0, 3000, 100, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	sd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalPAD).
		SetLevel(1).
		SetX(40).
		SetDuration(60_000).
		Build()

	p.executeStatBuff(caster, sd, byte(monster2.SkillTypeCarnivalPAD), 1)

	gotCaster, err := r.GetMonster(tm, caster.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster(caster): %v", err)
	}
	if !gotCaster.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP on caster")
	}

	gotOther, err := r.GetMonster(tm, other.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster(other): %v", err)
	}
	if gotOther.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected no POWER_UP on other monster without a bounding box")
	}
}

// TestExecuteStatBuff_Carnival_WithBoundingBox_InBoxOnly pins the AoE
// containment test in executeStatBuff: with a bounding box set, monsters
// whose dx/dy fall within [LtX,RbX]x[LtY,RbY] (inclusive) also receive the
// buff, and monsters outside the box do not.
func TestExecuteStatBuff_Carnival_WithBoundingBox_InBoxOnly(t *testing.T) {
	r := GetMonsterRegistry()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	r.Clear(ctx)

	f := testField()
	caster := r.CreateMonster(ctx, tm, f, 9300018, 0, 0, 0, 0, 0, 1000, 50, "", "")
	inBox := r.CreateMonster(ctx, tm, f, 9300018, 30, 10, 0, 5, 0, 3000, 100, "", "")
	outOfBox := r.CreateMonster(ctx, tm, f, 9300018, 200, 0, 0, 5, 0, 3000, 100, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		t:   tm,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	sd := mobskill.NewBuilder().
		SetSkillId(monster2.SkillTypeCarnivalPAD).
		SetLevel(1).
		SetX(40).
		SetDuration(60_000).
		SetBoundingBox(-50, -30, 50, 30).
		Build()

	p.executeStatBuff(caster, sd, byte(monster2.SkillTypeCarnivalPAD), 1)

	gotCaster, err := r.GetMonster(tm, caster.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster(caster): %v", err)
	}
	if !gotCaster.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP on caster")
	}

	gotInBox, err := r.GetMonster(tm, inBox.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster(inBox): %v", err)
	}
	if !gotInBox.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP on in-box monster")
	}

	gotOutOfBox, err := r.GetMonster(tm, outOfBox.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster(outOfBox): %v", err)
	}
	if gotOutOfBox.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected no POWER_UP on out-of-box monster")
	}
}
