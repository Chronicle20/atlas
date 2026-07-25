package pointreset

import (
	"atlas-channel/character"
	"testing"

	charskill "atlas-channel/character/skill"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// --- test fixtures -----------------------------------------------------
//
// Skill ids below are the real atlas-constants/skill numeric ids (verified
// against libs/atlas-constants/skill/constants.go) for the Warrior branch
// (Fighter=110/Crusader=111/Hero=112), the Noblesse beginner branch, the
// Evan stage-1 branch, and the Magician branch (used as an out-of-tree
// probe). job.IdFromSkillId derives the owning job purely by
// floor(skillId/10000), so these ids exercise the real job-tree arithmetic
// rather than invented numbers.

func mustSkill(id uint32, level byte, masterLevel byte) charskill.Model {
	m, err := charskill.Extract(charskill.RestModel{Id: id, Level: level, MasterLevel: masterLevel})
	if err != nil {
		panic(err)
	}
	return m
}

// baseCharacter is a safe-default Hero (job 112) character used as the
// starting point for the AP-transfer matrix: every primary stat and pool is
// mid-range so only the field under test trips a rule.
func baseCharacter() character.Model {
	return character.NewModelBuilder().
		SetId(1).
		SetJobId(job.HeroId).
		SetLevel(200).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetHp(100).
		SetMaxHp(100).
		SetMp(100).
		SetMaxMp(100).
		SetHpMpUsed(5).
		MustBuild()
}

// heroCharacter carries skill points in Hero's own tier-4 skill
// (HeroBrandish, level 5 / master 20) and its tier-3/tier-2 ancestor skills
// (CrusaderArmorCrash level 10, FighterSwordMastery level 10) — real
// in-tree SP-reset partners for a Hero per job.Is's branch arithmetic.
func heroCharacter() character.Model {
	return character.NewModelBuilder().
		SetId(1).
		SetJobId(job.HeroId).
		SetLevel(200).
		SetSkills([]charskill.Model{
			mustSkill(uint32(skill.FighterSwordMasteryId), 10, 0),
			mustSkill(uint32(skill.CrusaderArmorCrashId), 10, 0),
			mustSkill(uint32(skill.HeroBrandishId), 5, 20),
		}).
		MustBuild()
}

// jobLineCharacter builds a minimal character on an arbitrary job id,
// carrying the given skills. Used for the beginner/Evan/excluded-skill
// probes where the character's own job line (not Hero's) must match the
// probe skills' derived job.
func jobLineCharacter(jobId job.Id, skills []charskill.Model) character.Model {
	return character.NewModelBuilder().
		SetId(1).
		SetJobId(jobId).
		SetLevel(200).
		SetSkills(skills).
		MustBuild()
}

func TestAbilityFromWireFlag(t *testing.T) {
	cases := []struct {
		flag uint32
		want string
		ok   bool
	}{
		{64, AbilityStrength, true},
		{128, AbilityDexterity, true},
		{256, AbilityIntelligence, true},
		{512, AbilityLuck, true},
		{2048, AbilityHp, true},
		{8192, AbilityMp, true},
		{1, "", false},
		{0, "", false},
	}
	for _, tc := range cases {
		got, ok := AbilityFromWireFlag(tc.flag)
		if ok != tc.ok || got != tc.want {
			t.Errorf("AbilityFromWireFlag(%d) = (%q, %v), want (%q, %v)", tc.flag, got, ok, tc.want, tc.ok)
		}
	}
}

func TestSpResetTier(t *testing.T) {
	cases := []struct {
		id       item.Id
		wantTier byte
		wantOk   bool
	}{
		{item.Id(5050001), 1, true},
		{item.Id(5050002), 2, true},
		{item.Id(5050003), 3, true},
		{item.Id(5050004), 4, true},
		{item.Id(5050000), 0, false},
		{item.Id(5050005), 0, false},
	}
	for _, tc := range cases {
		tier, ok := SpResetTier(tc.id)
		if ok != tc.wantOk || (tc.wantOk && tier != tc.wantTier) {
			t.Errorf("SpResetTier(%d) = (%d, %v), want (%d, %v)", tc.id, tier, ok, tc.wantTier, tc.wantOk)
		}
	}
}

func TestValidateApTransfer(t *testing.T) {
	t.Run("source primary at floor(4) rejects", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetStrength(4).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityDexterity)
		want := &ValidationError{Code: ErrorCodeStatAtMinimum, Detail: AbilityStrength}
		assertValidationError(t, got, want)
	})

	t.Run("source primary at 5 (or above) passes the source check", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetStrength(5).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityDexterity)
		assertValidationError(t, got, nil)
	})

	t.Run("HP source with HpMpUsed 0 rejects", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetHpMpUsed(0).MustBuild()
		got := ValidateApTransfer(c, AbilityHp, AbilityDexterity)
		want := &ValidationError{Code: ErrorCodeInsufficientHpMpApUsed, Detail: AbilityHp}
		assertValidationError(t, got, want)
	})

	t.Run("HP source with HpMpUsed >0 passes the source check", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetHpMpUsed(1).MustBuild()
		got := ValidateApTransfer(c, AbilityHp, AbilityDexterity)
		assertValidationError(t, got, nil)
	})

	t.Run("target primary at cap rejects", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetDexterity(32767).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityDexterity)
		want := &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: AbilityDexterity}
		assertValidationError(t, got, want)
	})

	t.Run("target MaxHp at pool cap rejects", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetMaxHp(30000).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityHp)
		want := &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: AbilityHp}
		assertValidationError(t, got, want)
	})

	t.Run("target MaxMp at pool cap rejects", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetMaxMp(30000).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityMp)
		want := &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: AbilityMp}
		assertValidationError(t, got, want)
	})

	t.Run("unknown source ability is INVALID_TARGET", func(t *testing.T) {
		c := baseCharacter()
		got := ValidateApTransfer(c, "BOGUS", AbilityDexterity)
		want := &ValidationError{Code: ErrorCodeInvalidTarget, Detail: "BOGUS"}
		assertValidationError(t, got, want)
	})

	t.Run("unknown target ability is INVALID_TARGET", func(t *testing.T) {
		c := baseCharacter()
		got := ValidateApTransfer(c, AbilityStrength, "BOGUS")
		want := &ValidationError{Code: ErrorCodeInvalidTarget, Detail: "BOGUS"}
		assertValidationError(t, got, want)
	})

	t.Run("From==To STR->STR at STR 10 passes (pool-minimum NOT checked here)", func(t *testing.T) {
		c := character.CloneModel(baseCharacter()).SetStrength(10).MustBuild()
		got := ValidateApTransfer(c, AbilityStrength, AbilityStrength)
		assertValidationError(t, got, nil)
	})
}

func TestValidateSpTransfer(t *testing.T) {
	t.Run("out-of-job-tree target is INVALID_TARGET", func(t *testing.T) {
		c := heroCharacter()
		got := ValidateSpTransfer(c, skill.HeroBrandishId, skill.MagicianEnergyBoltId, 4, 20)
		want := &ValidationError{Code: ErrorCodeInvalidTarget}
		assertValidationError(t, got, want)
	})

	t.Run("excluded skill (GM range) is INVALID_TARGET", func(t *testing.T) {
		// job.Id(900) is an arbitrary branch-root job (900 % 100 == 0); the
		// job.Is/Advancement arithmetic is pure math and does not require a
		// registered Jobs[] entry. skill.Id(9001000) is the start of the
		// fixed GM-skill exclusion range in skill.IsPointResetExcluded.
		c := jobLineCharacter(job.Id(900), []charskill.Model{
			mustSkill(9000001, 10, 0),
		})
		got := ValidateSpTransfer(c, skill.Id(9000001), skill.Id(9001000), 1, 20)
		want := &ValidationError{Code: ErrorCodeInvalidTarget}
		assertValidationError(t, got, want)
	})

	t.Run("target tier != item tier is WRONG_TIER", func(t *testing.T) {
		c := heroCharacter()
		// CrusaderArmorCrash is tier 3 (job.Advancement(111)==3); item tier 3.
		got := ValidateSpTransfer(c, skill.CrusaderArmorCrashId, skill.HeroBrandishId, 3, 20)
		want := &ValidationError{Code: ErrorCodeWrongTier}
		assertValidationError(t, got, want)
	})

	t.Run("source tier > item tier is WRONG_TIER", func(t *testing.T) {
		c := heroCharacter()
		// HeroBrandish is tier 4; item tier 3 matches the target
		// (CrusaderArmorCrash, tier 3) but the source outranks the item.
		got := ValidateSpTransfer(c, skill.HeroBrandishId, skill.CrusaderArmorCrashId, 3, 20)
		want := &ValidationError{Code: ErrorCodeWrongTier}
		assertValidationError(t, got, want)
	})

	t.Run("beginner-prefix skill is WRONG_TIER", func(t *testing.T) {
		// NoblesseThreeSnails/NoblesseRecovery are Noblesse (job 1000)
		// beginner-branch skills; job.Advancement(1000) == 0 (IsBeginner),
		// which never matches an item tier (1-4).
		c := jobLineCharacter(job.NoblesseId, []charskill.Model{
			mustSkill(uint32(skill.NoblesseThreeSnailsId), 5, 0),
		})
		got := ValidateSpTransfer(c, skill.NoblesseThreeSnailsId, skill.NoblesseRecoveryId, 1, 20)
		want := &ValidationError{Code: ErrorCodeWrongTier}
		assertValidationError(t, got, want)
	})

	t.Run("Evan job is WRONG_TIER", func(t *testing.T) {
		// EvanStage1 (job 2200) falls in job.Advancement's Evan-stage
		// carve-out and always returns -1 (never maps onto tiers 1-4).
		c := jobLineCharacter(job.EvanStage1Id, []charskill.Model{
			mustSkill(uint32(skill.EvanStage1DragonSoulId), 5, 0),
		})
		got := ValidateSpTransfer(c, skill.EvanStage1DragonSoulId, skill.EvanStage1MagicMissileId, 1, 20)
		want := &ValidationError{Code: ErrorCodeWrongTier}
		assertValidationError(t, got, want)
	})

	t.Run("absent source skill is SKILL_AT_ZERO", func(t *testing.T) {
		c := heroCharacter()
		// CrusaderShout is tier 3 (in-tree, not present in heroCharacter's
		// skills slice); target HeroBrandish is tier 4, item tier 4.
		got := ValidateSpTransfer(c, skill.CrusaderShoutId, skill.HeroBrandishId, 4, 20)
		want := &ValidationError{Code: ErrorCodeSkillAtZero}
		assertValidationError(t, got, want)
	})

	t.Run("present source skill at level 0 is SKILL_AT_ZERO", func(t *testing.T) {
		c := jobLineCharacter(job.HeroId, []charskill.Model{
			mustSkill(uint32(skill.CrusaderArmorCrashId), 0, 0),
			mustSkill(uint32(skill.HeroBrandishId), 5, 20),
		})
		got := ValidateSpTransfer(c, skill.CrusaderArmorCrashId, skill.HeroBrandishId, 4, 20)
		want := &ValidationError{Code: ErrorCodeSkillAtZero}
		assertValidationError(t, got, want)
	})

	t.Run("non-4th-job target at gameDataMaxLevel is SKILL_AT_CAP", func(t *testing.T) {
		c := jobLineCharacter(job.HeroId, []charskill.Model{
			mustSkill(uint32(skill.FighterSwordMasteryId), 10, 0),
			mustSkill(uint32(skill.CrusaderArmorCrashId), 20, 0), // == gameDataMaxLevel
		})
		got := ValidateSpTransfer(c, skill.FighterSwordMasteryId, skill.CrusaderArmorCrashId, 3, 20)
		want := &ValidationError{Code: ErrorCodeSkillAtCap}
		assertValidationError(t, got, want)
	})

	t.Run("4th-job target capped by MasterLevel, not gameDataMaxLevel", func(t *testing.T) {
		c := jobLineCharacter(job.HeroId, []charskill.Model{
			mustSkill(uint32(skill.CrusaderArmorCrashId), 10, 0),
			mustSkill(uint32(skill.HeroBrandishId), 25, 25), // masterLevel 25 < gameDataMaxLevel 30
		})
		got := ValidateSpTransfer(c, skill.CrusaderArmorCrashId, skill.HeroBrandishId, 4, 30)
		want := &ValidationError{Code: ErrorCodeSkillAtCap}
		assertValidationError(t, got, want)
	})

	t.Run("happy path passes", func(t *testing.T) {
		c := heroCharacter()
		got := ValidateSpTransfer(c, skill.CrusaderArmorCrashId, skill.HeroBrandishId, 4, 30)
		assertValidationError(t, got, nil)
	})
}

func TestErrorMessage(t *testing.T) {
	cases := []struct {
		name   string
		code   string
		detail string
		want   string
	}{
		{"stat at minimum", ErrorCodeStatAtMinimum, "STR", "You don't have the minimum STR required to swap."},
		{"insufficient hpmp ap used", ErrorCodeInsufficientHpMpApUsed, "HP", "You don't have enough HPMP stat points to spend on AP Reset."},
		{"pool below job minimum", ErrorCodePoolBelowJobMinimum, "HP", "You don't have the minimum HP pool required to swap."},
		{"skill at zero", ErrorCodeSkillAtZero, "", "There are no points in that skill to move."},
		{"skill at cap", ErrorCodeSkillAtCap, "", "That skill cannot be raised any further."},
		{"wrong tier", ErrorCodeWrongTier, "", "That SP Reset cannot move points into that skill."},
		{"invalid target", ErrorCodeInvalidTarget, "", "That skill's points cannot be moved."},
		{"unknown code", "SOMETHING_ELSE", "", "Couldn't execute AP reset operation."},
		{"unrecognized detail", ErrorCodeStatAtMinimum, "", "Couldn't execute AP reset operation."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ErrorMessage(tc.code, tc.detail)
			if got != tc.want {
				t.Errorf("ErrorMessage(%q, %q) = %q, want %q", tc.code, tc.detail, got, tc.want)
			}
		})
	}
}

func assertValidationError(t *testing.T, got *ValidationError, want *ValidationError) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("expected nil (pass), got %+v", got)
		}
		return
	}
	if got == nil {
		t.Fatalf("expected %+v, got nil (pass)", want)
	}
	if got.Code != want.Code || got.Detail != want.Detail {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
