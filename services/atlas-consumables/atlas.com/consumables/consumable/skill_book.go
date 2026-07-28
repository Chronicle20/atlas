package consumable

import (
	"errors"
	"time"

	"github.com/google/uuid"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Skill-book eligibility sentinels (FR-2.5–2.8). Distinct values so tests and
// warn-logs can name the gate that rejected the request.
var (
	ErrSkillBookSkillNotLearned    = errors.New("mastery book requires the target skill to be learned")
	ErrSkillBookReqSkillLevel      = errors.New("current skill level below the book's required level")
	ErrSkillBookMasterLevelCeiling = errors.New("master level already at or above the book's grant")
)

// SelectSkillBookTargetSkill returns the first entry in the book's skills[]
// whose job prefix equals the character's job id (Cosmic
// ItemInformationProvider rule: skillId/10000 == jobId); no match means the
// book is unusable for this character (FR-2.4).
func SelectSkillBookTargetSkill(skills []uint32, jobId job.Id) (uint32, bool) {
	for _, s := range skills {
		if job.IdFromSkillId(skill.Id(s)) == jobId {
			return s, true
		}
	}
	return 0, false
}

// SkillBookRollPasses reports whether a roll in [0,100) passes for a percent
// success rate. Strictly-less-than: success=0 never passes, success=100
// always passes. Deliberately NOT the scroll path's <= (which gives
// success=0 a 1% pass rate) — design §2 D-3.
func SkillBookRollPasses(roll int32, successRate uint32) bool {
	return roll < int32(successRate)
}

// ValidateSkillBookSkillState enforces the skill-state gates (FR-2.5–2.8):
// a mastery book (229) requires a learned record (level >= 1); a skill book
// (228) may target an unlearned skill only when the book's reqSkillLevel is
// 0 (an absent record counts as level 0); the current level must meet
// reqSkillLevel; and the current master level must be below the book's grant.
func ValidateSkillBookSkillState(isMasteryBook bool, hasRecord bool, currentLevel byte, currentMasterLevel byte, reqSkillLevel uint32, bookMasterLevel uint32) error {
	if isMasteryBook && (!hasRecord || currentLevel < 1) {
		return ErrSkillBookSkillNotLearned
	}
	level := uint32(0)
	if hasRecord {
		level = uint32(currentLevel)
	}
	if level < reqSkillLevel {
		return ErrSkillBookReqSkillLevel
	}
	if uint32(currentMasterLevel) >= bookMasterLevel {
		return ErrSkillBookMasterLevelCeiling
	}
	return nil
}

// BuildSkillBookSaga constructs the skill_book_use saga (design §5.3).
// Step 1 destroys exactly one book from the slot on BOTH outcomes
// (consume-on-fail, Cosmic parity); TemplateId rides along so the
// orchestrator's reverse walk can re-award the book if the skill step fails.
// Step 2 exists only on a passing roll: update_skill carries the CURRENT
// level/expiration (atlas-skills Update clobbers all columns), create_skill
// teaches an unlearned skill at level 0 with zero (permanent) expiration.
func BuildSkillBookSaga(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, rollPassed bool, hasRecord bool, currentLevel byte, currentExpiration time.Time, targetSkillId uint32, bookMasterLevel byte) sharedsaga.Saga {
	b := sharedsaga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(sharedsaga.SkillBookUse).
		SetInitiatedBy("SKILL_BOOK").
		AddStep("destroy_asset_from_slot", sharedsaga.Pending, sharedsaga.DestroyAssetFromSlot, sharedsaga.DestroyAssetFromSlotPayload{
			CharacterId:   characterId,
			InventoryType: byte(inventory2.TypeValueUse),
			Slot:          slot,
			Quantity:      1,
			ShowEffect:    false,
			TemplateId:    uint32(itemId),
		})
	if rollPassed {
		if hasRecord {
			b.AddStep("update_skill", sharedsaga.Pending, sharedsaga.UpdateSkill, sharedsaga.UpdateSkillPayload{
				CharacterId: characterId,
				SkillId:     targetSkillId,
				Level:       currentLevel,
				MasterLevel: bookMasterLevel,
				Expiration:  currentExpiration,
			})
		} else {
			b.AddStep("create_skill", sharedsaga.Pending, sharedsaga.CreateSkill, sharedsaga.CreateSkillPayload{
				CharacterId: characterId,
				SkillId:     targetSkillId,
				Level:       0,
				MasterLevel: bookMasterLevel,
				Expiration:  time.Time{},
			})
		}
	}
	return b.Build()
}
