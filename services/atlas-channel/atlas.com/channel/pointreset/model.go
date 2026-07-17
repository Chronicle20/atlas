// Package pointreset holds the channel-side pre-validation and player-facing
// messages for AP Reset (5050000) and SP Reset (5050001-5050004) cash items.
// The numeric job policy tables (take/gain/min-pool) deliberately live in
// atlas-character (design §7); this package checks only the structural rules
// and the floors/caps/gates visible on the channel character model.
package pointreset

import (
	"atlas-channel/character"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Ability enum strings — must match atlas-character's
// CommandDistributeApAbility* constants (its processor.go:45-50).
const (
	AbilityStrength     = "STRENGTH"
	AbilityDexterity    = "DEXTERITY"
	AbilityIntelligence = "INTELLIGENCE"
	AbilityLuck         = "LUCK"
	AbilityHp           = "HP"
	AbilityMp           = "MP"
)

// Machine-readable rejection codes — shared strings with the services' ERROR
// status events and the saga-failed ErrorCode.
const (
	ErrorCodeStatAtMinimum          = "STAT_AT_MINIMUM"
	ErrorCodeStatAtMaximum          = "STAT_AT_MAXIMUM"
	ErrorCodeInsufficientHpMpApUsed = "INSUFFICIENT_HPMP_AP_USED"
	ErrorCodePoolBelowJobMinimum    = "POOL_BELOW_JOB_MINIMUM"
	ErrorCodeSkillAtZero            = "SKILL_AT_ZERO"
	ErrorCodeSkillAtCap             = "SKILL_AT_CAP"
	ErrorCodeWrongTier              = "WRONG_TIER"
	ErrorCodeInvalidTarget          = "INVALID_TARGET"
)

// Fixed server policy (design §2.2): source floor 4 (must be >= 5 to move
// out), primary cap 32767, pool cap 30000.
const (
	primaryFloor = uint16(4)
	primaryCap   = uint16(32767)
	poolCap      = uint16(30000)
)

// ApResetItemId is the cash item id for AP Reset.
var ApResetItemId = item.Id(5050000)

// SpResetTier returns the SP Reset job-advancement tier for items
// 5050001-5050004 and false for anything else.
func SpResetTier(itemId item.Id) (byte, bool) {
	if itemId >= item.Id(5050001) && itemId <= item.Id(5050004) {
		return byte(itemId % 10), true
	}
	return 0, false
}

// AbilityFromWireFlag maps the client stat-flag encoding of the AP Reset body
// (the client's stat-update bitmask values) to an ability enum string.
func AbilityFromWireFlag(flag uint32) (string, bool) {
	switch flag {
	case 64:
		return AbilityStrength, true
	case 128:
		return AbilityDexterity, true
	case 256:
		return AbilityIntelligence, true
	case 512:
		return AbilityLuck, true
	case 2048:
		return AbilityHp, true
	case 8192:
		return AbilityMp, true
	}
	return "", false
}

// ValidationError is a structural pre-validation rejection: a machine
// Code paired with the ability/skill Detail it applies to.
type ValidationError struct {
	Code   string
	Detail string
}

func primaryValue(c character.Model, ability string) (uint16, bool) {
	switch ability {
	case AbilityStrength:
		return c.Strength(), true
	case AbilityDexterity:
		return c.Dexterity(), true
	case AbilityIntelligence:
		return c.Intelligence(), true
	case AbilityLuck:
		return c.Luck(), true
	}
	return 0, false
}

// ValidateApTransfer checks the structural AP-reset rules the channel can see
// cheaply. The job pool-minimum check (minHp/minMp tables) is atlas-character's
// alone and is NOT mirrored here.
func ValidateApTransfer(c character.Model, from string, to string) *ValidationError {
	// Source.
	if v, ok := primaryValue(c, from); ok {
		if v < primaryFloor+1 {
			return &ValidationError{Code: ErrorCodeStatAtMinimum, Detail: from}
		}
	} else if from == AbilityHp || from == AbilityMp {
		if c.HpMpUsed() < 1 {
			return &ValidationError{Code: ErrorCodeInsufficientHpMpApUsed, Detail: from}
		}
	} else {
		return &ValidationError{Code: ErrorCodeInvalidTarget, Detail: from}
	}
	// Target.
	if v, ok := primaryValue(c, to); ok {
		if v >= primaryCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else if to == AbilityHp {
		if c.MaxHp() >= poolCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else if to == AbilityMp {
		if c.MaxMp() >= poolCap {
			return &ValidationError{Code: ErrorCodeStatAtMaximum, Detail: to}
		}
	} else {
		return &ValidationError{Code: ErrorCodeInvalidTarget, Detail: to}
	}
	return nil
}

// ValidateSpTransfer checks the full SP-reset rule set (design §4.3 arm 5).
// gameDataMaxLevel is len(Effects()) from atlas-data for the target skill;
// the 4th-job cap is the character's own master level.
func ValidateSpTransfer(c character.Model, fromId skill.Id, toId skill.Id, tier byte, gameDataMaxLevel byte) *ValidationError {
	fromJob := job.IdFromSkillId(fromId)
	toJob := job.IdFromSkillId(toId)
	if !job.Is(c.JobId(), fromJob) || !job.Is(c.JobId(), toJob) {
		return &ValidationError{Code: ErrorCodeInvalidTarget}
	}
	if skill.IsPointResetExcluded(fromId) || skill.IsPointResetExcluded(toId) {
		return &ValidationError{Code: ErrorCodeInvalidTarget}
	}
	fromTier := job.Advancement(fromJob)
	toTier := job.Advancement(toJob)
	if toTier != int(tier) || fromTier < 1 || fromTier > int(tier) {
		return &ValidationError{Code: ErrorCodeWrongTier}
	}
	fromSkill, err := c.SkillById(fromId)
	if err != nil || fromSkill.Level() == 0 {
		return &ValidationError{Code: ErrorCodeSkillAtZero}
	}
	var toLevel, toMaster byte
	if toSkill, err := c.SkillById(toId); err == nil {
		toLevel, toMaster = toSkill.Level(), toSkill.MasterLevel()
	}
	levelCap := gameDataMaxLevel
	if job.IsFourthJob(toJob) {
		levelCap = toMaster
	}
	if toLevel >= levelCap {
		return &ValidationError{Code: ErrorCodeSkillAtCap}
	}
	return nil
}

// abilityDisplay maps ability enum strings to the short names used in the
// player-facing rejection messages.
var abilityDisplay = map[string]string{
	AbilityStrength:     "STR",
	AbilityDexterity:    "DEX",
	AbilityIntelligence: "INT",
	AbilityLuck:         "LUK",
	AbilityHp:           "HP",
	AbilityMp:           "MP",
}

// ErrorMessage renders the player-facing pink-text message for a rejection
// code. detail is the ability enum (or, on the saga path, the failed event's
// Reason field, which the compensator sets to the service's errorDetail).
func ErrorMessage(code string, detail string) string {
	disp, known := abilityDisplay[detail]
	if !known {
		disp = detail
	}
	switch code {
	case ErrorCodeStatAtMinimum:
		if disp != "" && (known || disp == "STR" || disp == "DEX" || disp == "INT" || disp == "LUK" || disp == "HP" || disp == "MP") {
			return fmt.Sprintf("You don't have the minimum %s required to swap.", disp)
		}
	case ErrorCodeInsufficientHpMpApUsed:
		return "You don't have enough HPMP stat points to spend on AP Reset."
	case ErrorCodePoolBelowJobMinimum:
		if disp == "HP" || disp == "MP" {
			return fmt.Sprintf("You don't have the minimum %s pool required to swap.", disp)
		}
	case ErrorCodeSkillAtZero:
		return "There are no points in that skill to move."
	case ErrorCodeSkillAtCap:
		return "That skill cannot be raised any further."
	case ErrorCodeWrongTier:
		return "That SP Reset cannot move points into that skill."
	case ErrorCodeInvalidTarget:
		return "That skill's points cannot be moved."
	}
	return "Couldn't execute AP reset operation."
}
