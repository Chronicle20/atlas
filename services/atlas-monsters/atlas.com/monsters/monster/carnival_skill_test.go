package monster

import (
	"atlas-monsters/monster/mobskill"
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// carnivalCase is a single CARNIVAL_BUFF dispatch table row: casting skillId
// at skillLevel with the given mob skill x/duration must apply exactly the
// expected status name/value pair.
type carnivalCase struct {
	name       string
	skillId    byte
	skillLevel byte
	x          int32
	duration   uint32
	wantStatus monster2.TemporaryStatType
	wantValue  int32
}

var carnivalCases = []carnivalCase{
	{"carnival_pad", byte(monster2.SkillTypeCarnivalPAD), 1, 40, 1_200_000, monster2.TemporaryStatTypePowerUp, 40},
	{"carnival_acc", byte(monster2.SkillTypeCarnivalACC), 1, 50, 1_200_000, monster2.TemporaryStatTypeAccuracy, 50},
	{"carnival_seal_skill", byte(monster2.SkillTypeCarnivalSealSkill), 1, 1, 180_000, monster2.TemporaryStatTypeSealSkill, 1},
}

// newCarnivalTestProcessor sets up a fresh miniredis-backed cooldown
// registry, a testMobSkillLookup hook returning the given x/duration for the
// requested id+level, and a fresh monster, returning the processor and
// registry-read handle for the created monster. Callers must invoke the
// returned cleanup func (via t.Cleanup or defer) to restore the swapped
// package-level state.
func newCarnivalTestProcessor(t *testing.T, c carnivalCase) (*ProcessorImpl, tenant.Model, uint32) {
	t.Helper()
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	t.Cleanup(func() { cooldownReg = prevCooldown })

	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		return mobskill.NewBuilder().
			SetSkillId(skillId).
			SetLevel(level).
			SetX(c.x).
			SetDuration(c.duration).
			Build(), nil
	}
	t.Cleanup(func() { testMobSkillLookup = prevSkill })

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100, "", "")

	p := &ProcessorImpl{
		l:   logrus.New(),
		ctx: tenant.WithContext(ctx, ten),
		t:   ten,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	return p, ten, m.UniqueId()
}

// assertCarnivalStatus re-reads the monster and asserts it carries exactly
// one status effect matching the case's expected status name/value, sourced
// from the cast skill, and not a reflect effect.
func assertCarnivalStatus(t *testing.T, ten tenant.Model, uniqueId uint32, c carnivalCase) {
	t.Helper()
	got, err := GetMonsterRegistry().GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if len(got.StatusEffects()) != 1 {
		t.Fatalf("expected 1 status effect, got %d", len(got.StatusEffects()))
	}
	se := got.StatusEffects()[0]
	if !se.HasStatus(string(c.wantStatus)) {
		t.Errorf("expected status %q to be present", c.wantStatus)
	}
	if se.Statuses()[string(c.wantStatus)] != c.wantValue {
		t.Errorf("status %q value = %d, want %d", c.wantStatus, se.Statuses()[string(c.wantStatus)], c.wantValue)
	}
	if se.SourceSkillId() != uint32(c.skillId) {
		t.Errorf("SourceSkillId = %d, want %d", se.SourceSkillId(), c.skillId)
	}
	if se.IsReflect() {
		t.Errorf("expected IsReflect()=false, got true")
	}
}

// TestUseSkill_Carnival_AppliesMappedStatus verifies that casting each
// CARNIVAL_BUFF skill through UseSkill applies exactly the status Task 1's
// SkillTypeToStatusName maps it to, at the mob skill's X value.
func TestUseSkill_Carnival_AppliesMappedStatus(t *testing.T) {
	for _, c := range carnivalCases {
		t.Run(c.name, func(t *testing.T) {
			p, ten, uniqueId := newCarnivalTestProcessor(t, c)

			p.UseSkill(uniqueId, 1, c.skillId, c.skillLevel)

			assertCarnivalStatus(t, ten, uniqueId, c)
		})
	}
}

// TestUseSkillGM_Carnival_AppliesMappedStatus mirrors
// TestUseSkill_Carnival_AppliesMappedStatus via the GM command entry point.
// It cannot pass until UseSkillGM honors testMobSkillLookup.
func TestUseSkillGM_Carnival_AppliesMappedStatus(t *testing.T) {
	for _, c := range carnivalCases {
		t.Run(c.name, func(t *testing.T) {
			p, ten, uniqueId := newCarnivalTestProcessor(t, c)

			p.UseSkillGM(uniqueId, c.skillId, c.skillLevel)

			assertCarnivalStatus(t, ten, uniqueId, c)
		})
	}
}

// TestUseSkill_Carnival_NoUnknownCategoryWarning verifies that, for every
// CARNIVAL_BUFF skill id (150-157), neither UseSkill nor UseSkillGM logs the
// "unknown skill category" or "No status mapping for skill type" warnings. A
// fresh monster is created per id so that skill 157 (SEAL_SKILL) applied in
// an earlier subtest cannot block a later cast in the same loop.
func TestUseSkill_Carnival_NoUnknownCategoryWarning(t *testing.T) {
	carnivalIds := []uint16{
		monster2.SkillTypeCarnivalPAD,
		monster2.SkillTypeCarnivalMAD,
		monster2.SkillTypeCarnivalPDR,
		monster2.SkillTypeCarnivalMDR,
		monster2.SkillTypeCarnivalACC,
		monster2.SkillTypeCarnivalEVA,
		monster2.SkillTypeCarnivalSpeed,
		monster2.SkillTypeCarnivalSealSkill,
	}

	assertNoWarnings := func(t *testing.T, hook *test.Hook) {
		t.Helper()
		for _, e := range hook.AllEntries() {
			if strings.Contains(e.Message, "unknown skill category") {
				t.Errorf("unexpected log entry: %q", e.Message)
			}
			if strings.Contains(e.Message, "No status mapping for skill type") {
				t.Errorf("unexpected log entry: %q", e.Message)
			}
		}
	}

	t.Run("UseSkill", func(t *testing.T) {
		for _, id := range carnivalIds {
			c := carnivalCase{skillId: byte(id), skillLevel: 1, x: 1, duration: 60_000}
			l, hook := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)
			p, _, uniqueId := newCarnivalTestProcessor(t, c)
			p.l = l

			p.UseSkill(uniqueId, 1, c.skillId, c.skillLevel)

			assertNoWarnings(t, hook)
			hook.Reset()
		}
	})

	t.Run("UseSkillGM", func(t *testing.T) {
		for _, id := range carnivalIds {
			c := carnivalCase{skillId: byte(id), skillLevel: 1, x: 1, duration: 60_000}
			l, hook := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)
			p, _, uniqueId := newCarnivalTestProcessor(t, c)
			p.l = l

			p.UseSkillGM(uniqueId, c.skillId, c.skillLevel)

			assertNoWarnings(t, hook)
			hook.Reset()
		}
	})
}

// TestExecuteStatBuff_Carnival_NoOppositeImmunityPrecancel_NotReflect
// verifies FR-3.3: executeStatBuff's opposite-immunity pre-cancel logic is
// gated on category == SkillCategoryImmunity, so a CARNIVAL_BUFF cast (e.g.
// skill 150, PAD) never cancels an already-active MAGIC_ATTACK_IMMUNE, and
// the resulting POWER_UP effect carries no reflect metadata.
func TestExecuteStatBuff_Carnival_NoOppositeImmunityPrecancel_NotReflect(t *testing.T) {
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

	applyImmunityForTest(t, p, m.UniqueId(), string(monster2.TemporaryStatTypeMagicAttackImmune), 1)

	m, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
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
	if !got.HasStatusEffect(string(monster2.TemporaryStatTypeMagicAttackImmune)) {
		t.Errorf("expected MAGIC_ATTACK_IMMUNE to remain (not pre-cancelled)")
	}
	if !got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP to be applied")
	}
	if len(got.StatusEffects()) != 2 {
		t.Fatalf("expected 2 status effects, got %d", len(got.StatusEffects()))
	}

	var powerUp StatusEffect
	found := false
	for _, se := range got.StatusEffects() {
		if se.HasStatus(string(monster2.TemporaryStatTypePowerUp)) {
			powerUp = se
			found = true
		}
	}
	if !found {
		t.Fatalf("POWER_UP status effect not found")
	}
	if powerUp.IsReflect() {
		t.Errorf("expected IsReflect()=false, got true")
	}
	if powerUp.ReflectKind() != "" {
		t.Errorf("ReflectKind: got %q, want \"\"", powerUp.ReflectKind())
	}
	if powerUp.ReflectPercent() != 0 {
		t.Errorf("ReflectPercent: got %d, want 0", powerUp.ReflectPercent())
	}
	if powerUp.ReflectMaxDamage() != 0 {
		t.Errorf("ReflectMaxDamage: got %d, want 0", powerUp.ReflectMaxDamage())
	}
}
