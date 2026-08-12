package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/equipment"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"

	chskill "atlas-channel/character/combo"
	skillmodel "atlas-channel/character/skill"

	equipslot "atlas-channel/equipment/slot"

	"github.com/google/uuid"

	slottype "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	aranTestPolearmId = uint32(1442000)
	aranTestSwordId   = uint32(1302000)
)

// aranTestCharacter mirrors the combo package's buildCharacter helper (Task
// 6), plus an explicit character id so each test uses a distinct one and the
// process-wide combo Mirror singleton cannot leak state between them.
func aranTestCharacter(t *testing.T, id uint32, jobId job.Id, skillId skill3.Id, level byte, weaponTemplateId uint32) character.Model {
	t.Helper()
	sm, err := skillmodel.Extract(skillmodel.RestModel{Id: uint32(skillId), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	eq := equipment.NewModel()
	if weaponTemplateId != 0 {
		w := asset.NewBuilder(uuid.New(), weaponTemplateId).SetId(1).MustBuild()
		eq.Set(slottype.Type("weapon"), equipslot.Model{Equipable: &w})
	}
	return character.NewModelBuilder().
		SetId(id).
		SetJobId(jobId).
		SetSkills([]skillmodel.Model{sm}).
		SetEquipment(eq).
		MustBuild()
}

// aranTestEffectLookup delegates to the existing comboTestEffect helper.
func aranTestEffectLookup(t *testing.T, x int16) func(uint32, byte) (effect.Model, error) {
	return func(uint32, byte) (effect.Model, error) { return comboTestEffect(t, x, 0), nil }
}

// An eligible Aran's gate result lands in the mirror when the attack pipeline
// runs, so the ARAN_COMBO_COUNTER packets that follow cost zero REST calls.
func TestAranComboRefreshCachesEligibility(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	c := aranTestCharacter(t, 21, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)

	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))

	el, ok := chskill.GetMirror().Eligibility(tn, 21, time.Now(), time.Minute)
	if !ok {
		t.Fatal("want a cached eligibility after the refresh, got none")
	}
	if el.ComboId() != skill3.AranStage1ComboAbilityId || el.ComboLevel() != 5 || el.StatAmount() != 5 {
		t.Errorf("cached eligibility = (%d,%d,%d), want (%d,5,5)",
			el.ComboId(), el.ComboLevel(), el.StatAmount(), skill3.AranStage1ComboAbilityId)
	}
}

// An ineligible character leaves no entry behind: a stale-eligible cache
// would let a modified client keep incrementing after unequipping.
func TestAranComboRefreshClearsIneligible(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	eligible := aranTestCharacter(t, 22, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)
	aranComboRefreshEligibility(l, ctx, comboTestField(), eligible, aranTestEffectLookup(t, 5))

	swapped := aranTestCharacter(t, 22, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestSwordId)
	aranComboRefreshEligibility(l, ctx, comboTestField(), swapped, aranTestEffectLookup(t, 5))

	if _, ok := chskill.GetMirror().Eligibility(tn, 22, time.Now(), time.Minute); ok {
		t.Error("swapping to a non-polearm must clear the cached eligibility")
	}
}

// The refresh must never advance the count -- only ARAN_COMBO_COUNTER does.
func TestAranComboRefreshDoesNotIncrement(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	c := aranTestCharacter(t, 23, job.AranStage1Id, skill3.AranStage1ComboAbilityId, 5, aranTestPolearmId)

	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))
	aranComboRefreshEligibility(l, ctx, comboTestField(), c, aranTestEffectLookup(t, 5))

	count, seeded := chskill.GetMirror().Increment(tn, 23, comboTestField(), chskill.DefaultIdleWindow, time.Now())
	if count != 1 || !seeded {
		t.Fatalf("refresh must not advance the count: want (1,true) on the first increment, got (%d,%v)", count, seeded)
	}
}
