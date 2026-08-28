package monster

import (
	"atlas-monsters/monster/information"
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

// TestUseSkill_SealSkill_RejectsAndLogsDistinctly verifies FR-6.3/FR-7.2: a
// monster with SEAL_SKILL active rejects UseSkill before the skill executes,
// and the rejection log line names SEAL_SKILL specifically (not the bare
// SEAL wording), mirroring the picker's gate from Task 4.
func TestUseSkill_SealSkill_RejectsAndLogsDistinctly(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	defer func() { cooldownReg = prevCooldown }()

	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		return mobskill.NewBuilder().
			SetSkillId(skillId).
			SetLevel(level).
			SetX(40).
			SetDuration(1_200_000).
			Build(), nil
	}
	defer func() { testMobSkillLookup = prevSkill }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100, "", "")

	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	p := &ProcessorImpl{
		l:   l,
		ctx: tenant.WithContext(ctx, ten),
		t:   ten,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	applyImmunityForTest(t, p, m.UniqueId(), string(monster2.TemporaryStatTypeSealSkill), 1)

	p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalPAD), 1)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP not applied while SEAL_SKILL is active")
	}
	if len(got.StatusEffects()) != 1 {
		t.Fatalf("expected 1 status effect (the seeded SEAL_SKILL), got %d", len(got.StatusEffects()))
	}

	found := false
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "SEAL_SKILL") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a logged message naming SEAL_SKILL, got entries: %+v", hook.AllEntries())
	}
}

// TestUseSkill_Seal_StillRejects is the D7 regression: seeding the
// pre-existing SEAL status (with no SEAL_SKILL present) must still reject
// UseSkill and log a message containing "SEAL". The gate added in Task 5 is
// strictly additive to the existing SEAL check.
func TestUseSkill_Seal_StillRejects(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	defer func() { cooldownReg = prevCooldown }()

	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		return mobskill.NewBuilder().
			SetSkillId(skillId).
			SetLevel(level).
			SetX(40).
			SetDuration(1_200_000).
			Build(), nil
	}
	defer func() { testMobSkillLookup = prevSkill }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 5100004, 0, 0, 0, 5, 0, 3000, 100, "", "")

	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	p := &ProcessorImpl{
		l:   l,
		ctx: tenant.WithContext(ctx, ten),
		t:   ten,
		emit: func(_ string, _ model.Provider[[]kafka.Message]) error {
			return nil
		},
		inFieldFn: func(_ field.Model) ([]uint32, error) { return nil, nil },
	}

	applyImmunityForTest(t, p, m.UniqueId(), string(monster2.TemporaryStatTypeSeal), 1)

	p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalPAD), 1)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP not applied while SEAL is active")
	}

	found := false
	var matched string
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "SEAL") {
			found = true
			matched = e.Message
			break
		}
	}
	if !found {
		t.Errorf("expected a logged message naming SEAL, got entries: %+v", hook.AllEntries())
	}
	if strings.Contains(matched, "SEAL_SKILL") {
		t.Errorf("expected the SEAL rejection log message to not name SEAL_SKILL, got: %s", matched)
	}
}

// TestUseBasicAttack_SealSkill_StillSucceeds verifies FR-6.6: SEAL_SKILL
// gates UseSkill only. UseBasicAttack has no suppression gate and must
// still deduct MP and register cooldown while SEAL_SKILL is active.
func TestUseBasicAttack_SealSkill_StillSucceeds(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevAttackReg := attackCooldownReg
	attackCooldownReg = &attackCooldownRegistry{reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-attack-cooldown", func(s string) string { return s })}
	defer func() { attackCooldownReg = prevAttackReg }()

	prevHook := testInformationLookup
	testInformationLookup = func(monsterId uint32) (information.Model, error) {
		return information.NewBuilder().
			SetAttacks([]information.AttackInfo{{Pos: 2, ConMP: 5, AttackAfter: 1500}}).
			Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	monsterId := uint32(5100004)
	m := r.CreateMonster(ctx, ten, f, monsterId, 0, 0, 0, 5, 0, 3000, 100, "", "")
	uniqueId := m.UniqueId()

	p := &ProcessorImpl{l: logrus.New(), ctx: tenant.WithContext(ctx, ten), t: ten, emit: func(string, model.Provider[[]kafka.Message]) error { return nil }}

	applyImmunityForTest(t, p, uniqueId, string(monster2.TemporaryStatTypeSealSkill), 1)

	p.UseBasicAttack(uniqueId, uint8(1)) // 0-indexed, matches Pos=2 (1+1)

	got, err := r.GetMonster(ten, uniqueId)
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.Mp() != 95 {
		t.Errorf("Mp after UseBasicAttack = %d, want 95 (100-5)", got.Mp())
	}
	if !attackCooldownReg.IsOnCooldown(ctx, ten, uniqueId, uint8(1)) {
		t.Errorf("expected attack pos 1 to be on cooldown after happy-path UseBasicAttack")
	}
}

// TestUseSkill_Skill157ThenAnySkill_RejectedEndToEnd is the end-to-end case:
// casting skill 157 (CARNIVAL_SEAL_SKILL) applies SEAL_SKILL, and a
// subsequent UseSkill for any other skill (150, CARNIVAL_PAD) is then
// rejected at cast time by the same gate.
func TestUseSkill_Skill157ThenAnySkill_RejectedEndToEnd(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	prevCooldown := cooldownReg
	cooldownReg = &cooldownRegistry{reg: atlasredis.NewTenantRegistry[string, int64](rc, "monster-cooldown", func(s string) string { return s })}
	defer func() { cooldownReg = prevCooldown }()

	prevSkill := testMobSkillLookup
	testMobSkillLookup = func(skillId uint16, level uint16) (mobskill.Model, error) {
		switch skillId {
		case 157:
			return mobskill.NewBuilder().
				SetSkillId(157).
				SetLevel(1).
				SetX(1).
				SetDuration(180_000).
				Build(), nil
		case 150:
			return mobskill.NewBuilder().
				SetSkillId(150).
				SetLevel(1).
				SetX(40).
				SetDuration(1_200_000).
				Build(), nil
		}
		t.Fatalf("unexpected skill lookup for id %d", skillId)
		return mobskill.Model{}, nil
	}
	defer func() { testMobSkillLookup = prevSkill }()

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

	p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalSealSkill), 1)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if !got.HasStatusEffect(string(monster2.TemporaryStatTypeSealSkill)) {
		t.Fatalf("expected SEAL_SKILL applied after casting skill 157")
	}

	p.UseSkill(m.UniqueId(), 1, byte(monster2.SkillTypeCarnivalPAD), 1)

	got, err = r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.HasStatusEffect(string(monster2.TemporaryStatTypePowerUp)) {
		t.Errorf("expected POWER_UP not applied; skill 150 should be rejected by the SEAL_SKILL gate")
	}
	if len(got.StatusEffects()) != 1 {
		t.Errorf("expected only the SEAL_SKILL status effect, got %d", len(got.StatusEffects()))
	}
}
