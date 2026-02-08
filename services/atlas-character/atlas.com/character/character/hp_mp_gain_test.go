package character

import (
	skill3 "atlas-character/data/skill"
	"atlas-character/data/skill/effect"
	cskill "atlas-character/skill"
	"testing"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/skill"
)

type mockSkillDataProcessor struct {
	getEffectFunc func(uniqueId uint32, level byte) (effect.Model, error)
	callCount     int
}

func (m *mockSkillDataProcessor) GetById(uniqueId uint32) (skill3.Model, error) {
	return skill3.Model{}, nil
}

func (m *mockSkillDataProcessor) GetEffect(uniqueId uint32, level byte) (effect.Model, error) {
	m.callCount++
	if m.getEffectFunc != nil {
		return m.getEffectFunc(uniqueId, level)
	}
	return effect.Model{}, nil
}

func newTestProcessor(sdp skill3.Processor) *ProcessorImpl {
	return &ProcessorImpl{
		sdp: sdp,
	}
}

func buildCharacter(jobId job.Id, skills []cskill.Model) Model {
	return NewModelBuilder().SetJobId(jobId).SetSkills(skills).Build()
}

func buildSkill(id uint32, level byte) cskill.Model {
	m, _ := cskill.Extract(cskill.RestModel{Id: id, Level: level})
	return m
}

func TestResolveHPMPGainParams_Beginner(t *testing.T) {
	mock := &mockSkillDataProcessor{}
	p := newTestProcessor(mock)
	c := buildCharacter(job.BeginnerId, nil)

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 12 || params.hpUpper != 16 {
		t.Fatalf("Beginner HP range should be 12-16, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 10 || params.mpUpper != 12 {
		t.Fatalf("Beginner MP range should be 10-12, got %d-%d", params.mpLower, params.mpUpper)
	}
	if params.hpBonus != 0 || params.mpBonus != 0 {
		t.Fatalf("Beginner should have no bonuses, got hp=%d mp=%d", params.hpBonus, params.mpBonus)
	}
	if mock.callCount != 0 {
		t.Fatalf("Beginner should not call GetEffect, got %d calls", mock.callCount)
	}
}

func TestResolveHPMPGainParams_Warrior(t *testing.T) {
	bonus := int16(10)
	mock := &mockSkillDataProcessor{
		getEffectFunc: func(uniqueId uint32, level byte) (effect.Model, error) {
			rm := effect.RestModel{X: bonus}
			return effect.Extract(rm)
		},
	}
	p := newTestProcessor(mock)
	c := buildCharacter(job.FighterId, []cskill.Model{
		buildSkill(uint32(skill.WarriorImprovedMaxHpIncreaseId), 5),
	})

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 24 || params.hpUpper != 28 {
		t.Fatalf("Warrior HP range should be 24-28, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 4 || params.mpUpper != 6 {
		t.Fatalf("Warrior MP range should be 4-6, got %d-%d", params.mpLower, params.mpUpper)
	}
	if params.hpBonus != bonus {
		t.Fatalf("Warrior HP bonus should be %d, got %d", bonus, params.hpBonus)
	}
	if params.mpBonus != 0 {
		t.Fatalf("Warrior MP bonus should be 0, got %d", params.mpBonus)
	}
}

func TestResolveHPMPGainParams_Magician(t *testing.T) {
	bonus := int16(15)
	mock := &mockSkillDataProcessor{
		getEffectFunc: func(uniqueId uint32, level byte) (effect.Model, error) {
			rm := effect.RestModel{X: bonus}
			return effect.Extract(rm)
		},
	}
	p := newTestProcessor(mock)
	c := buildCharacter(job.MagicianId, []cskill.Model{
		buildSkill(uint32(skill.MagicianImprovedMaxMpIncreaseId), 3),
	})

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 10 || params.hpUpper != 14 {
		t.Fatalf("Magician HP range should be 10-14, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 22 || params.mpUpper != 24 {
		t.Fatalf("Magician MP range should be 22-24, got %d-%d", params.mpLower, params.mpUpper)
	}
	if params.hpBonus != 0 {
		t.Fatalf("Magician HP bonus should be 0, got %d", params.hpBonus)
	}
	if params.mpBonus != bonus {
		t.Fatalf("Magician MP bonus should be %d, got %d", bonus, params.mpBonus)
	}
}

func TestResolveHPMPGainParams_BowmanRogue(t *testing.T) {
	mock := &mockSkillDataProcessor{}
	p := newTestProcessor(mock)
	c := buildCharacter(job.HunterId, nil)

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 20 || params.hpUpper != 24 {
		t.Fatalf("Bowman HP range should be 20-24, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 14 || params.mpUpper != 16 {
		t.Fatalf("Bowman MP range should be 14-16, got %d-%d", params.mpLower, params.mpUpper)
	}
	if mock.callCount != 0 {
		t.Fatalf("Bowman should not call GetEffect, got %d calls", mock.callCount)
	}
}

func TestResolveHPMPGainParams_GM(t *testing.T) {
	mock := &mockSkillDataProcessor{}
	p := newTestProcessor(mock)
	c := buildCharacter(job.GmId, nil)

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 30000 || params.hpUpper != 30000 {
		t.Fatalf("GM HP range should be 30000-30000, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 30000 || params.mpUpper != 30000 {
		t.Fatalf("GM MP range should be 30000-30000, got %d-%d", params.mpLower, params.mpUpper)
	}
}

func TestResolveHPMPGainParams_Pirate(t *testing.T) {
	bonus := int16(8)
	mock := &mockSkillDataProcessor{
		getEffectFunc: func(uniqueId uint32, level byte) (effect.Model, error) {
			rm := effect.RestModel{X: bonus}
			return effect.Extract(rm)
		},
	}
	p := newTestProcessor(mock)
	c := buildCharacter(job.BrawlerId, []cskill.Model{
		buildSkill(uint32(skill.BrawlerImproveMaxHpId), 5),
	})

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 22 || params.hpUpper != 28 {
		t.Fatalf("Pirate HP range should be 22-28, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 18 || params.mpUpper != 23 {
		t.Fatalf("Pirate MP range should be 18-23, got %d-%d", params.mpLower, params.mpUpper)
	}
	if params.hpBonus != bonus {
		t.Fatalf("Pirate HP bonus should be %d, got %d", bonus, params.hpBonus)
	}
}

func TestResolveHPMPGainParams_Aran(t *testing.T) {
	mock := &mockSkillDataProcessor{}
	p := newTestProcessor(mock)
	c := buildCharacter(job.AranStage2Id, nil)

	params := p.resolveHPMPGainParams(c)

	if params.hpLower != 44 || params.hpUpper != 48 {
		t.Fatalf("Aran HP range should be 44-48, got %d-%d", params.hpLower, params.hpUpper)
	}
	if params.mpLower != 4 || params.mpUpper != 8 {
		t.Fatalf("Aran MP range should be 4-8, got %d-%d", params.mpLower, params.mpUpper)
	}
}

func TestResolveHPMPGainParams_NoSkillLevel(t *testing.T) {
	mock := &mockSkillDataProcessor{}
	p := newTestProcessor(mock)
	// Warrior with no improving skill learned (level 0)
	c := buildCharacter(job.FighterId, nil)

	params := p.resolveHPMPGainParams(c)

	if params.hpBonus != 0 {
		t.Fatalf("No skill should yield 0 HP bonus, got %d", params.hpBonus)
	}
	// GetEffect should still be called (with level 0), returning empty model
	if mock.callCount != 1 {
		t.Fatalf("Should call GetEffect once for HP skill, got %d", mock.callCount)
	}
}

func TestRollHPMPGain_WithinBounds(t *testing.T) {
	params := hpMPGainParams{
		hpLower: 24,
		hpUpper: 28,
		mpLower: 4,
		mpUpper: 6,
		hpBonus: 0,
		mpBonus: 0,
	}

	for i := 0; i < 1000; i++ {
		hp, mp := rollHPMPGain(params)
		if hp < 24 || hp > 28 {
			t.Fatalf("HP %d outside range 24-28", hp)
		}
		if mp < 4 || mp > 6 {
			t.Fatalf("MP %d outside range 4-6", mp)
		}
	}
}

func TestRollHPMPGain_WithBonus(t *testing.T) {
	params := hpMPGainParams{
		hpLower: 24,
		hpUpper: 28,
		mpLower: 4,
		mpUpper: 6,
		hpBonus: 10,
		mpBonus: 5,
	}

	for i := 0; i < 1000; i++ {
		hp, mp := rollHPMPGain(params)
		if hp < 34 || hp > 38 {
			t.Fatalf("HP %d outside range 34-38 (24-28 + 10 bonus)", hp)
		}
		if mp < 9 || mp > 11 {
			t.Fatalf("MP %d outside range 9-11 (4-6 + 5 bonus)", mp)
		}
	}
}

func TestRollHPMPGain_GMFixed(t *testing.T) {
	params := hpMPGainParams{
		hpLower: 30000,
		hpUpper: 30000,
		mpLower: 30000,
		mpUpper: 30000,
	}

	hp, mp := rollHPMPGain(params)
	if hp != 30000 {
		t.Fatalf("GM HP should be exactly 30000, got %d", hp)
	}
	if mp != 30000 {
		t.Fatalf("GM MP should be exactly 30000, got %d", mp)
	}
}

func TestResolveHPMPGainParams_GetEffectCalledOnce(t *testing.T) {
	mock := &mockSkillDataProcessor{
		getEffectFunc: func(uniqueId uint32, level byte) (effect.Model, error) {
			rm := effect.RestModel{X: 5}
			return effect.Extract(rm)
		},
	}
	p := newTestProcessor(mock)
	// Warrior (has HP improving skill)
	c := buildCharacter(job.FighterId, []cskill.Model{
		buildSkill(uint32(skill.WarriorImprovedMaxHpIncreaseId), 5),
	})

	_ = p.resolveHPMPGainParams(c)

	if mock.callCount != 1 {
		t.Fatalf("GetEffect should be called exactly once, got %d", mock.callCount)
	}
}
