package skill_test

import (
	"encoding/json"
	"testing"
	"time"

	"atlas-skills/kafka/message"
	macro2 "atlas-skills/kafka/message/macro"
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/macro"
	"atlas-skills/skill"
	"atlas-skills/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	constskill "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// --- Fixture skill ids (verified against libs/atlas-constants/skill/constants.go
// and libs/atlas-constants/job/constants.go) ---
//
//	WarriorIronBodyId       = 1001003  tier1 (job.IdFromSkillId -> 100  = job.WarriorId)
//	WarriorPowerStrikeId    = 1001004  tier1 (-> 100)
//	WarriorSlashBlastId     = 1001005  tier1 (-> 100)
//	FighterRageId           = 1101006  tier2 (-> 110 = job.FighterId)
//	HeroAdvancedComboAttackId = 1120003 tier4 (-> 112 = job.HeroId, fourth job)
//	MagicianImprovedMaxMpIncreaseId = 2000001 (-> 200 = job.MagicianId, different branch)
//	BeginnerThreeSnailsId   = 1000     tier0 (-> 0 = job.BeginnerId)
//	BeginnerRecoveryId      = 1001     tier0 (-> 0)
//	EvanStage2FireCircleId  = 22101000 (-> 2210 = job.EvanStage2Id, Advancement = -1)
//	EvanStage3LightningBoltId = 22111000 (-> 2211 = job.EvanStage3Id, Advancement = -1)
//	Aran hidden combo id 21110007 (-> 2111 = job.AranStage3Id) is point-reset-excluded
//	AranStage4MapleWarriorId = 21121000 (-> 2112 = job.AranStage4Id), not excluded
const (
	warriorIronBodyId    = uint32(1001003)
	warriorPowerStrikeId = uint32(1001004)
	warriorSlashBlastId  = uint32(1001005)
	fighterRageId        = uint32(1101006)
	heroAdvComboId       = uint32(1120003)
	magicianSkillId      = uint32(2000001)
	beginnerThreeSnails  = uint32(1000)
	beginnerRecovery     = uint32(1001)
	evanStage2SkillId    = uint32(22101000)
	evanStage3SkillId    = uint32(22111000)
	aranExcludedSkillId  = uint32(21110007)
	aranStage4SkillId    = uint32(21121000)
)

// setupTransferSpFixture wires a skill.Processor and macro.Processor against
// the same in-memory DB/tenant context, mirroring skill/processor_test.go's
// setupProcessor plus a macro.Processor for macro-cleanup assertions.
func setupTransferSpFixture(t *testing.T) (skill.Processor, macro.Processor, func()) {
	t.Helper()
	setupCooldownRegistry(t)
	db := test.SetupTestDB(t)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	sp := skill.NewProcessor(logger, ctx, db)
	mp := macro.NewProcessor(logger, ctx, db)

	cleanup := func() {
		test.CleanupTestDB(db)
	}
	return sp, mp, cleanup
}

// --- kafka.Message decoding helpers -----------------------------------------

type envelope struct {
	SkillId uint32          `json:"skillId"`
	Type    string          `json:"type"`
	Body    json.RawMessage `json:"body"`
}

func decodeSkillEnvelopes(t *testing.T, mb *message.Buffer) []envelope {
	t.Helper()
	var out []envelope
	for _, m := range mb.GetAll()[skill2.EnvStatusEventTopic] {
		var e envelope
		if err := json.Unmarshal(m.Value, &e); err != nil {
			t.Fatalf("failed to decode skill status envelope: %v", err)
		}
		out = append(out, e)
	}
	return out
}

func findEnvelope(envs []envelope, eventType string) (envelope, bool) {
	for _, e := range envs {
		if e.Type == eventType {
			return e, true
		}
	}
	return envelope{}, false
}

func countEnvelopes(envs []envelope, eventType string) int {
	n := 0
	for _, e := range envs {
		if e.Type == eventType {
			n++
		}
	}
	return n
}

type macroEnvelope struct {
	Type string                        `json:"type"`
	Body macro2.StatusEventUpdatedBody `json:"body"`
}

func decodeMacroEnvelopes(t *testing.T, mb *message.Buffer) []macroEnvelope {
	t.Helper()
	var out []macroEnvelope
	for _, m := range mb.GetAll()[macro2.EnvStatusEventTopic] {
		var e macroEnvelope
		if err := json.Unmarshal(m.Value, &e); err != nil {
			t.Fatalf("failed to decode macro status envelope: %v", err)
		}
		out = append(out, e)
	}
	return out
}

func macroTopicMessageCount(mb *message.Buffer) int {
	return len(mb.GetAll()[macro2.EnvStatusEventTopic])
}

// mustCreateSkill seeds a skill row directly (bypassing TransferSp validation)
// for test fixtures.
func mustCreateSkill(t *testing.T, sp skill.Processor, characterId uint32, skillId uint32, level byte, masterLevel byte) {
	t.Helper()
	scratch := message.NewBuffer()
	_, err := sp.Create(scratch)(uuid.New(), world.Id(0), characterId, skillId, level, masterLevel, time.Time{})
	if err != nil {
		t.Fatalf("setup: create skill %d: %v", skillId, err)
	}
}

func mustCreateMacro(t *testing.T, mp macro.Processor, characterId uint32, macros []macro.Model) {
	t.Helper()
	scratch := message.NewBuffer()
	_, err := mp.Update(scratch)(uuid.New(), world.Id(0), characterId, macros)
	if err != nil {
		t.Fatalf("setup: create macros: %v", err)
	}
}

func buildMacro(t *testing.T, id uint32, name string, skillId1, skillId2, skillId3 uint32) macro.Model {
	t.Helper()
	m, err := macro.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetShout(false).
		SetSkillId1(constskill.Id(skillId1)).
		SetSkillId2(constskill.Id(skillId2)).
		SetSkillId3(constskill.Id(skillId3)).
		Build()
	if err != nil {
		t.Fatalf("buildMacro: %v", err)
	}
	return m
}

// --- Test matrix -------------------------------------------------------------

func TestTransferSp(t *testing.T) {
	characterId := uint32(555555)
	transactionId := uuid.New()
	worldId := world.Id(0)

	t.Run("tier1 success", func(t *testing.T) {
		sp, _, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, warriorIronBodyId, 3, 0)
		mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 1, 0)

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		from, ferr := sp.ByIdProvider(characterId, warriorIronBodyId)()
		if ferr != nil {
			t.Fatalf("from skill missing: %v", ferr)
		}
		if from.Level() != 2 {
			t.Errorf("from.Level() = %d, want 2", from.Level())
		}
		to, terr := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
		if terr != nil {
			t.Fatalf("to skill missing: %v", terr)
		}
		if to.Level() != 2 {
			t.Errorf("to.Level() = %d, want 2", to.Level())
		}

		envs := decodeSkillEnvelopes(t, mb)
		spEnv, ok := findEnvelope(envs, skill2.StatusEventTypeSpTransferred)
		if !ok {
			t.Fatal("expected SP_TRANSFERRED event")
		}
		if spEnv.SkillId != warriorPowerStrikeId {
			t.Errorf("SP_TRANSFERRED envelope SkillId = %d, want %d (target)", spEnv.SkillId, warriorPowerStrikeId)
		}
		var spBody skill2.StatusEventSpTransferredBody
		if err := json.Unmarshal(spEnv.Body, &spBody); err != nil {
			t.Fatalf("failed to decode SP_TRANSFERRED body: %v", err)
		}
		if spBody.FromSkillId != warriorIronBodyId {
			t.Errorf("SP_TRANSFERRED FromSkillId = %d, want %d", spBody.FromSkillId, warriorIronBodyId)
		}
		if spBody.FromLevel != 2 {
			t.Errorf("SP_TRANSFERRED FromLevel = %d, want 2", spBody.FromLevel)
		}
		if spBody.ToLevel != 2 {
			t.Errorf("SP_TRANSFERRED ToLevel = %d, want 2", spBody.ToLevel)
		}
		if countEnvelopes(envs, skill2.StatusEventTypeUpdated) != 2 {
			t.Errorf("expected 2 UPDATED events, got %d", countEnvelopes(envs, skill2.StatusEventTypeUpdated))
		}
	})

	t.Run("target absent is created at level 1", func(t *testing.T) {
		sp, _, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 0)

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorSlashBlastId, 1, 20)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		to, terr := sp.ByIdProvider(characterId, warriorSlashBlastId)()
		if terr != nil {
			t.Fatalf("target should have been created: %v", terr)
		}
		if to.Level() != 1 {
			t.Errorf("to.Level() = %d, want 1", to.Level())
		}
		if to.MasterLevel() != 0 {
			t.Errorf("to.MasterLevel() = %d, want 0", to.MasterLevel())
		}
		if !to.Expiration().IsZero() {
			t.Errorf("to.Expiration() = %v, want zero", to.Expiration())
		}

		envs := decodeSkillEnvelopes(t, mb)
		if _, ok := findEnvelope(envs, skill2.StatusEventTypeCreated); !ok {
			t.Error("expected CREATED event for the target skill")
		}
		if countEnvelopes(envs, skill2.StatusEventTypeUpdated) != 1 {
			t.Errorf("expected 1 UPDATED event (source only), got %d", countEnvelopes(envs, skill2.StatusEventTypeUpdated))
		}
	})

	t.Run("source at zero rejects", func(t *testing.T) {
		t.Run("row absent", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 5, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			envs := decodeSkillEnvelopes(t, mb)
			e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
			if !ok {
				t.Fatal("expected ERROR event")
			}
			var body skill2.StatusEventErrorBody
			_ = json.Unmarshal(e.Body, &body)
			if body.Error != skill2.StatusEventErrorTypeSkillAtZero {
				t.Errorf("Error = %s, want SKILL_AT_ZERO", body.Error)
			}

			to, _ := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
			if to.Level() != 5 {
				t.Errorf("target should be unchanged, level = %d, want 5", to.Level())
			}
		})

		t.Run("row at level zero", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 0, 0)
			mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 5, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			envs := decodeSkillEnvelopes(t, mb)
			e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
			if !ok {
				t.Fatal("expected ERROR event")
			}
			var body skill2.StatusEventErrorBody
			_ = json.Unmarshal(e.Body, &body)
			if body.Error != skill2.StatusEventErrorTypeSkillAtZero {
				t.Errorf("Error = %s, want SKILL_AT_ZERO", body.Error)
			}

			from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
			if from.Level() != 0 {
				t.Errorf("source should be unchanged, level = %d, want 0", from.Level())
			}
			to, _ := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
			if to.Level() != 5 {
				t.Errorf("target should be unchanged, level = %d, want 5", to.Level())
			}
		})
	})

	t.Run("target at max level rejects with SKILL_AT_CAP", func(t *testing.T) {
		sp, _, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 0)
		mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 20, 0)

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		envs := decodeSkillEnvelopes(t, mb)
		e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
		if !ok {
			t.Fatal("expected ERROR event")
		}
		var body skill2.StatusEventErrorBody
		_ = json.Unmarshal(e.Body, &body)
		if body.Error != skill2.StatusEventErrorTypeSkillAtCap {
			t.Errorf("Error = %s, want SKILL_AT_CAP", body.Error)
		}

		from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
		if from.Level() != 5 {
			t.Errorf("source should be unchanged, level = %d, want 5", from.Level())
		}
		to, _ := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
		if to.Level() != 20 {
			t.Errorf("target should be unchanged, level = %d, want 20", to.Level())
		}
	})

	t.Run("4th job cap uses own master level", func(t *testing.T) {
		t.Run("at master level caps", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, fighterRageId, 5, 0)
			mustCreateSkill(t, sp, characterId, heroAdvComboId, 10, 10)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.HeroId, fighterRageId, heroAdvComboId, 4, 99)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			envs := decodeSkillEnvelopes(t, mb)
			e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
			if !ok {
				t.Fatal("expected ERROR event")
			}
			var body skill2.StatusEventErrorBody
			_ = json.Unmarshal(e.Body, &body)
			if body.Error != skill2.StatusEventErrorTypeSkillAtCap {
				t.Errorf("Error = %s, want SKILL_AT_CAP", body.Error)
			}

			to, _ := sp.ByIdProvider(characterId, heroAdvComboId)()
			if to.Level() != 10 {
				t.Errorf("target should be unchanged, level = %d, want 10", to.Level())
			}
		})

		t.Run("below master level succeeds", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, fighterRageId, 5, 0)
			mustCreateSkill(t, sp, characterId, heroAdvComboId, 5, 10)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.HeroId, fighterRageId, heroAdvComboId, 4, 99)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			to, terr := sp.ByIdProvider(characterId, heroAdvComboId)()
			if terr != nil {
				t.Fatalf("to skill missing: %v", terr)
			}
			if to.Level() != 6 {
				t.Errorf("to.Level() = %d, want 6", to.Level())
			}
			if to.MasterLevel() != 10 {
				t.Errorf("to.MasterLevel() = %d, want 10 (untouched)", to.MasterLevel())
			}
		})
	})

	t.Run("4th job target absent or master level zero always caps", func(t *testing.T) {
		sp, _, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, fighterRageId, 5, 0)
		// heroAdvComboId row intentionally absent -> masterLevel treated as 0.

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.HeroId, fighterRageId, heroAdvComboId, 4, 99)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		envs := decodeSkillEnvelopes(t, mb)
		e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
		if !ok {
			t.Fatal("expected ERROR event")
		}
		var body skill2.StatusEventErrorBody
		_ = json.Unmarshal(e.Body, &body)
		if body.Error != skill2.StatusEventErrorTypeSkillAtCap {
			t.Errorf("Error = %s, want SKILL_AT_CAP", body.Error)
		}

		if _, terr := sp.ByIdProvider(characterId, heroAdvComboId)(); terr == nil {
			t.Error("target skill should still be absent")
		}
	})

	t.Run("wrong tier rejects", func(t *testing.T) {
		t.Run("target tier does not match item tier", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 0)
			mustCreateSkill(t, sp, characterId, fighterRageId, 5, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.FighterId, warriorIronBodyId, fighterRageId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertWrongTier(t, mb)
		})

		t.Run("source tier exceeds item tier", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, fighterRageId, 5, 0)
			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 1, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.FighterId, fighterRageId, warriorIronBodyId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertWrongTier(t, mb)
		})

		t.Run("beginner tier-0 skills reject", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, beginnerThreeSnails, 5, 0)
			mustCreateSkill(t, sp, characterId, beginnerRecovery, 1, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.BeginnerId, beginnerThreeSnails, beginnerRecovery, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertWrongTier(t, mb)
		})

		t.Run("evan job rejects", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, evanStage2SkillId, 5, 0)
			mustCreateSkill(t, sp, characterId, evanStage3SkillId, 1, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.EvanStage3Id, evanStage2SkillId, evanStage3SkillId, 2, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertWrongTier(t, mb)
		})
	})

	t.Run("invalid target rejects", func(t *testing.T) {
		t.Run("skill outside job tree", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, magicianSkillId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertInvalidTarget(t, mb)

			from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
			if from.Level() != 5 {
				t.Errorf("source should be unchanged, level = %d, want 5", from.Level())
			}
		})

		t.Run("excluded skill", func(t *testing.T) {
			sp, _, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, aranExcludedSkillId, 5, 0)
			mustCreateSkill(t, sp, characterId, aranStage4SkillId, 1, 0)

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.AranStage4Id, aranExcludedSkillId, aranStage4SkillId, 4, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}
			assertInvalidTarget(t, mb)

			from, _ := sp.ByIdProvider(characterId, aranExcludedSkillId)()
			if from.Level() != 5 {
				t.Errorf("source should be unchanged, level = %d, want 5", from.Level())
			}
		})
	})

	t.Run("macro cleanup", func(t *testing.T) {
		t.Run("source drops to zero clears referencing slot only", func(t *testing.T) {
			sp, mp, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 1, 0)
			mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 1, 0)

			mustCreateMacro(t, mp, characterId, []macro.Model{
				buildMacro(t, 0, "Combo", warriorSlashBlastId, warriorIronBodyId, warriorPowerStrikeId),
			})

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
			if from.Level() != 0 {
				t.Fatalf("from.Level() = %d, want 0", from.Level())
			}

			macros, merr := mp.ByCharacterIdProvider(characterId)()
			if merr != nil {
				t.Fatalf("ByCharacterIdProvider() unexpected error: %v", merr)
			}
			if len(macros) != 1 {
				t.Fatalf("expected 1 macro, got %d", len(macros))
			}
			m := macros[0]
			if uint32(m.SkillId1()) != warriorSlashBlastId {
				t.Errorf("SkillId1 = %d, want %d (untouched)", m.SkillId1(), warriorSlashBlastId)
			}
			if uint32(m.SkillId2()) != 0 {
				t.Errorf("SkillId2 = %d, want 0 (cleared)", m.SkillId2())
			}
			if uint32(m.SkillId3()) != warriorPowerStrikeId {
				t.Errorf("SkillId3 = %d, want %d (untouched)", m.SkillId3(), warriorPowerStrikeId)
			}

			macroEnvs := decodeMacroEnvelopes(t, mb)
			found := false
			for _, e := range macroEnvs {
				if e.Type == macro2.StatusEventTypeUpdated {
					found = true
				}
			}
			if !found {
				t.Error("expected macro UPDATED event")
			}
		})

		t.Run("source drops to one leaves macros untouched", func(t *testing.T) {
			sp, mp, cleanup := setupTransferSpFixture(t)
			defer cleanup()

			mustCreateSkill(t, sp, characterId, warriorIronBodyId, 2, 0)
			mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 1, 0)

			mustCreateMacro(t, mp, characterId, []macro.Model{
				buildMacro(t, 0, "Combo", warriorSlashBlastId, warriorIronBodyId, warriorPowerStrikeId),
			})

			mb := message.NewBuffer()
			err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
			if err != nil {
				t.Fatalf("TransferSp() unexpected error: %v", err)
			}

			from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
			if from.Level() != 1 {
				t.Fatalf("from.Level() = %d, want 1", from.Level())
			}

			macros, merr := mp.ByCharacterIdProvider(characterId)()
			if merr != nil {
				t.Fatalf("ByCharacterIdProvider() unexpected error: %v", merr)
			}
			if len(macros) != 1 {
				t.Fatalf("expected 1 macro, got %d", len(macros))
			}
			m := macros[0]
			if uint32(m.SkillId2()) != warriorIronBodyId {
				t.Errorf("SkillId2 = %d, want %d (untouched, source not at 0)", m.SkillId2(), warriorIronBodyId)
			}

			if n := macroTopicMessageCount(mb); n != 0 {
				t.Errorf("expected 0 macro events, got %d", n)
			}
		})
	})

	t.Run("validation failure leaves skills and macros unchanged", func(t *testing.T) {
		sp, mp, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 0)
		mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 20, 0)
		mustCreateMacro(t, mp, characterId, []macro.Model{
			buildMacro(t, 0, "Combo", warriorIronBodyId, 0, 0),
		})

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		envs := decodeSkillEnvelopes(t, mb)
		e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
		if !ok || e.Type == "" {
			t.Fatal("expected ERROR event (SKILL_AT_CAP)")
		}

		from, _ := sp.ByIdProvider(characterId, warriorIronBodyId)()
		if from.Level() != 5 {
			t.Errorf("source should be unchanged, level = %d, want 5", from.Level())
		}
		to, _ := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
		if to.Level() != 20 {
			t.Errorf("target should be unchanged, level = %d, want 20", to.Level())
		}

		macros, merr := mp.ByCharacterIdProvider(characterId)()
		if merr != nil {
			t.Fatalf("ByCharacterIdProvider() unexpected error: %v", merr)
		}
		if len(macros) != 1 || uint32(macros[0].SkillId1()) != warriorIronBodyId {
			t.Errorf("macros should be unchanged, got %+v", macros)
		}
		if n := macroTopicMessageCount(mb); n != 0 {
			t.Errorf("expected 0 macro events on a rejected transfer, got %d", n)
		}
	})

	t.Run("master level of source is untouched (FR-15)", func(t *testing.T) {
		sp, _, cleanup := setupTransferSpFixture(t)
		defer cleanup()

		mustCreateSkill(t, sp, characterId, warriorIronBodyId, 5, 20)
		mustCreateSkill(t, sp, characterId, warriorPowerStrikeId, 1, 0)

		mb := message.NewBuffer()
		err := sp.TransferSp(mb)(transactionId, worldId, characterId, job.WarriorId, warriorIronBodyId, warriorPowerStrikeId, 1, 20)
		if err != nil {
			t.Fatalf("TransferSp() unexpected error: %v", err)
		}

		from, ferr := sp.ByIdProvider(characterId, warriorIronBodyId)()
		if ferr != nil {
			t.Fatalf("from skill missing: %v", ferr)
		}
		if from.Level() != 4 {
			t.Errorf("from.Level() = %d, want 4", from.Level())
		}
		if from.MasterLevel() != 20 {
			t.Errorf("from.MasterLevel() = %d, want 20 (untouched, FR-15)", from.MasterLevel())
		}

		to, terr := sp.ByIdProvider(characterId, warriorPowerStrikeId)()
		if terr != nil {
			t.Fatalf("to skill missing: %v", terr)
		}
		if to.MasterLevel() != 0 {
			t.Errorf("to.MasterLevel() = %d, want 0 (untouched, FR-16)", to.MasterLevel())
		}
	})
}

func assertWrongTier(t *testing.T, mb *message.Buffer) {
	t.Helper()
	envs := decodeSkillEnvelopes(t, mb)
	e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
	if !ok {
		t.Fatal("expected ERROR event")
	}
	var body skill2.StatusEventErrorBody
	_ = json.Unmarshal(e.Body, &body)
	if body.Error != skill2.StatusEventErrorTypeWrongTier {
		t.Errorf("Error = %s, want WRONG_TIER", body.Error)
	}
}

func assertInvalidTarget(t *testing.T, mb *message.Buffer) {
	t.Helper()
	envs := decodeSkillEnvelopes(t, mb)
	e, ok := findEnvelope(envs, skill2.StatusEventTypeError)
	if !ok {
		t.Fatal("expected ERROR event")
	}
	var body skill2.StatusEventErrorBody
	_ = json.Unmarshal(e.Body, &body)
	if body.Error != skill2.StatusEventErrorTypeInvalidTarget {
		t.Errorf("Error = %s, want INVALID_TARGET", body.Error)
	}
}
