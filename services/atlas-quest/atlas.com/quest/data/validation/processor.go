package validation

import (
	"context"

	dataquest "atlas-quest/data/quest"

	"github.com/sirupsen/logrus"
)

// Processor provides validation functionality against query-aggregator
type Processor interface {
	// ValidateStartRequirements checks if a character meets the quest start requirements
	ValidateStartRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error)
	// ValidateEndRequirements checks if a character meets the quest end requirements (items only)
	ValidateEndRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ValidateStartRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error) {
	var conditions []ConditionInput

	req := questDef.StartRequirements

	// Level requirements
	if req.LevelMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     LevelCondition,
			Operator: ">=",
			Value:    int(req.LevelMin),
		})
	}
	if req.LevelMax > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     LevelCondition,
			Operator: "<=",
			Value:    int(req.LevelMax),
		})
	}

	// Job requirements - check if character's job is in the allowed list
	if len(req.Jobs) > 0 {
		// Convert jobs to int slice for "in" operator
		jobValues := make([]int, len(req.Jobs))
		for i, job := range req.Jobs {
			jobValues[i] = int(job)
		}
		conditions = append(conditions, ConditionInput{
			Type:     JobCondition,
			Operator: "in",
			Values:   jobValues,
		})
	}

	// Fame requirement
	if req.FameMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     FameCondition,
			Operator: ">=",
			Value:    int(req.FameMin),
		})
	}

	// Meso requirements
	if req.MesoMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     MesoCondition,
			Operator: ">=",
			Value:    int(req.MesoMin),
		})
	}
	if req.MesoMax > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     MesoCondition,
			Operator: "<=",
			Value:    int(req.MesoMax),
		})
	}

	// Item requirements
	for _, item := range req.Items {
		if item.Count > 0 {
			conditions = append(conditions, ConditionInput{
				Type:        ItemCondition,
				Operator:    ">=",
				Value:       int(item.Count),
				ReferenceId: item.Id,
			})
		}
	}

	// Prerequisite quest requirements
	for _, quest := range req.Quests {
		conditions = append(conditions, ConditionInput{
			Type:        QuestStatusCondition,
			Operator:    "=",
			Value:       int(quest.State),
			ReferenceId: quest.Id,
		})
	}

	// If no conditions, validation passes
	if len(conditions) == 0 {
		return true, nil, nil
	}

	// Call query-aggregator
	result, err := requestValidation(characterId, conditions)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to validate start requirements for character [%d]", characterId)
		return false, nil, err
	}

	if result.AllPassed() {
		return true, nil, nil
	}

	return false, result.GetFailedConditions(), nil
}

func (p *ProcessorImpl) ValidateEndRequirements(characterId uint32, questDef dataquest.RestModel) (bool, []string, error) {
	var conditions []ConditionInput

	req := questDef.EndRequirements

	// Item requirements for completion
	for _, item := range req.Items {
		if item.Count > 0 {
			// Player must have at least this many items
			conditions = append(conditions, ConditionInput{
				Type:        ItemCondition,
				Operator:    ">=",
				Value:       int(item.Count),
				ReferenceId: item.Id,
			})
		} else if item.Count == 0 {
			// Player must NOT have this item (e.g., consumed it)
			conditions = append(conditions, ConditionInput{
				Type:        ItemCondition,
				Operator:    "=",
				Value:       0,
				ReferenceId: item.Id,
			})
		}
	}

	// Meso requirements for completion
	if req.MesoMin > 0 {
		conditions = append(conditions, ConditionInput{
			Type:     MesoCondition,
			Operator: ">=",
			Value:    int(req.MesoMin),
		})
	}

	// If no conditions, validation passes
	if len(conditions) == 0 {
		return true, nil, nil
	}

	// Call query-aggregator
	result, err := requestValidation(characterId, conditions)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to validate end requirements for character [%d]", characterId)
		return false, nil, err
	}

	if result.AllPassed() {
		return true, nil, nil
	}

	return false, result.GetFailedConditions(), nil
}
