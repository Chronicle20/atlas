package definition

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"fmt"
)

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	QuestId string   `json:"questId"`
	Name    string   `json:"name"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

var validFieldLocks = map[string]bool{
	"none":     true,
	"channel":  true,
	"instance": true,
}

var validRegTypes = map[string]bool{
	"party":      true,
	"individual": true,
}

var validAffinities = map[string]bool{
	"none":  true,
	"guild": true,
	"party": true,
}

var validRegModes = map[string]bool{
	"instant": true,
	"timed":   true,
}

var validStageTypes = map[string]bool{
	stage.TypeItemCollection:     true,
	stage.TypeMonsterKilling:     true,
	stage.TypeCombinationPuzzle:  true,
	stage.TypeReactorTrigger:     true,
	stage.TypeWarpPuzzle:         true,
	stage.TypeSequenceMemoryGame: true,
	stage.TypeBoss:               true,
}

var validBonusEntryModes = map[string]bool{
	"auto":   true,
	"manual": true,
}

var validRequirementConditionTypes = map[string]bool{
	"party_size": true,
	"level_min":  true,
	"level_max":  true,
}

var validRequirementOperators = map[string]bool{
	"eq":  true,
	"gte": true,
	"lte": true,
	"gt":  true,
	"lt":  true,
}

var validClearConditionTypes = map[string]bool{
	"item":         true,
	"item_count":   true,
	"monster_kill": true,
	"custom_data":  true,
}

var validClearOperators = map[string]bool{
	">=": true,
	"<=": true,
	"=":  true,
	">":  true,
	"<":  true,
}

var validClearActions = map[string]bool{
	"destroy_monsters": true,
}

var validRewardTypes = map[string]bool{
	"experience":  true,
	"item":        true,
	"random_item": true,
}

var validWarpTypes = map[string]bool{
	"all":  true,
	"none": true,
}

func Validate(rm RestModel) ValidationResult {
	result := ValidationResult{
		Valid:   true,
		QuestId: rm.QuestId,
		Name:    rm.Name,
	}

	if rm.QuestId == "" {
		result.addError("questId is required")
	}
	if rm.Name == "" {
		result.addError("name is required")
	}

	if rm.FieldLock != "" && !validFieldLocks[rm.FieldLock] {
		result.addError(fmt.Sprintf("invalid fieldLock %q, must be one of: none, channel, instance", rm.FieldLock))
	}

	validateRegistration(&result, rm.Registration)
	validateRequirementConditions(&result, rm.StartRequirements, "startRequirements")
	validateRequirementConditions(&result, rm.FailRequirements, "failRequirements")
	validateBonus(&result, rm.Bonus)
	validateStages(&result, rm.Stages)
	validateRewardModels(&result, rm.Rewards, "rewards")

	if rm.Exit == 0 && len(rm.Stages) > 0 {
		result.addWarning("exit map is 0, characters may not warp out properly")
	}

	return result
}

func validateRegistration(result *ValidationResult, reg RegistrationRestModel) {
	if reg.Type != "" && !validRegTypes[reg.Type] {
		result.addError(fmt.Sprintf("invalid registration type %q, must be one of: party, individual", reg.Type))
	}
	if reg.Mode != "" && !validRegModes[reg.Mode] {
		result.addError(fmt.Sprintf("invalid registration mode %q, must be one of: instant, timed", reg.Mode))
	}
	if reg.Mode == "timed" && reg.Duration <= 0 {
		result.addError("timed registration mode requires duration > 0")
	}
	if reg.Affinity != "" && !validAffinities[reg.Affinity] {
		result.addError(fmt.Sprintf("invalid registration affinity %q, must be one of: none, guild, party", reg.Affinity))
	}
}

func validateBonus(result *ValidationResult, bonus *BonusRestModel) {
	if bonus == nil {
		return
	}
	if bonus.MapId == 0 {
		result.addError("bonus.mapId is required")
	}
	if bonus.Entry == "" {
		result.addError("bonus.entry is required (auto or manual)")
	} else if !validBonusEntryModes[bonus.Entry] {
		result.addError(fmt.Sprintf("bonus.entry has invalid value %q, must be one of: auto, manual", bonus.Entry))
	}
}

func validateRequirementConditions(result *ValidationResult, conditions []condition.RestModel, context string) {
	for i, c := range conditions {
		if !validRequirementConditionTypes[c.Type] {
			result.addError(fmt.Sprintf("%s[%d] has invalid type %q", context, i, c.Type))
		}
		if !validRequirementOperators[c.Operator] {
			result.addError(fmt.Sprintf("%s[%d] has invalid operator %q", context, i, c.Operator))
		}
	}
}

func validateStages(result *ValidationResult, stages []stage.RestModel) {
	for i, s := range stages {
		if s.Index != uint32(i) {
			result.addError(fmt.Sprintf("stage[%d] has index %d, expected sequential index %d", i, s.Index, i))
		}

		if s.Name == "" {
			result.addWarning(fmt.Sprintf("stage[%d] has no name", i))
		}

		if !validStageTypes[s.Type] {
			result.addError(fmt.Sprintf("stage[%d] has invalid type %q", i, s.Type))
		}

		if len(s.MapIds) == 0 {
			result.addWarning(fmt.Sprintf("stage[%d] %q has no mapIds", i, s.Name))
		}

		if s.WarpType != "" && !validWarpTypes[s.WarpType] {
			result.addError(fmt.Sprintf("stage[%d] has invalid warpType %q", i, s.WarpType))
		}

		for j, c := range s.ClearConditions {
			if !validClearConditionTypes[c.Type] {
				result.addError(fmt.Sprintf("stage[%d].clearConditions[%d] has invalid type %q", i, j, c.Type))
			}
			if !validClearOperators[c.Operator] {
				result.addError(fmt.Sprintf("stage[%d].clearConditions[%d] has invalid operator %q", i, j, c.Operator))
			}
		}

		for j, a := range s.ClearActions {
			if !validClearActions[a] {
				result.addError(fmt.Sprintf("stage[%d].clearActions[%d] has invalid action %q", i, j, a))
			}
		}

		for j, r := range s.Rewards {
			if !validRewardTypes[r.Type] {
				result.addError(fmt.Sprintf("stage[%d].rewards[%d] has invalid type %q", i, j, r.Type))
			}
			if r.Type == "random_item" && len(r.Items) == 0 {
				result.addError(fmt.Sprintf("stage[%d].rewards[%d] is random_item but has no items", i, j))
			}
		}
	}
}

func validateRewardModels(result *ValidationResult, rewards []reward.RestModel, context string) {
	for i, r := range rewards {
		if !validRewardTypes[r.Type] {
			result.addError(fmt.Sprintf("%s[%d] has invalid type %q", context, i, r.Type))
		}
		if r.Type == "random_item" && len(r.Items) == 0 {
			result.addError(fmt.Sprintf("%s[%d] is random_item but has no items", context, i))
		}
	}
}

func (r *ValidationResult) addError(msg string) {
	r.Valid = false
	r.Errors = append(r.Errors, msg)
}

func (r *ValidationResult) addWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}
