package validation

import (
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/inventory"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/party"
	"atlas-query-aggregator/pet"
	"atlas-query-aggregator/quest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// ValidateStructured validates a list of structured condition inputs against a character
	ValidateStructured(decorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error)

	// ValidateWithContext validates a list of structured condition inputs using a validation context
	ValidateWithContext(decorators ...model.Decorator[ValidationResult]) func(ctx ValidationContext, conditionInputs []ConditionInput) (ValidationResult, error)
}

// ProcessorImpl handles validation logic
type ProcessorImpl struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	characterProcessor character.Processor
	inventoryProcessor inventory.Processor
	questProcessor     quest.Processor
	marriageProcessor  marriage.Processor
	buddyProcessor     buddy.Processor
	petProcessor       pet.Processor
	partyProcessor     party.Processor
}

// NewProcessor creates a new validation processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:                  l,
		ctx:                ctx,
		characterProcessor: character.NewProcessor(l, ctx),
		inventoryProcessor: inventory.NewProcessor(l, ctx),
		questProcessor:     quest.NewProcessor(l, ctx),
		marriageProcessor:  marriage.NewProcessor(l, ctx),
		buddyProcessor:     buddy.NewProcessor(l, ctx),
		petProcessor:       pet.NewProcessor(l, ctx),
		partyProcessor:     party.NewProcessor(l, ctx),
	}
}

// ValidateStructured validates a list of structured condition inputs against a character
func (p *ProcessorImpl) ValidateStructured(resultDecorators ...model.Decorator[ValidationResult]) func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error) {
	return func(characterId uint32, conditionInputs []ConditionInput) (ValidationResult, error) {
		// Create a new validation result
		result := NewValidationResult(characterId)

		// Parse all conditions and check which data sources we need
		conditions := make([]Condition, 0, len(conditionInputs))
		needsInventory := false
		needsGuild := false
		needsContext := false

		for _, input := range conditionInputs {
			condition, err := NewConditionBuilder().FromInput(input).Build()
			if err != nil {
				return result, fmt.Errorf("invalid condition: %w", err)
			}

			conditions = append(conditions, condition)

			// Check if this condition requires inventory data
			if condition.conditionType == ItemCondition {
				needsInventory = true
			}

			// Check if this condition requires guild data
			if condition.conditionType == GuildLeaderCondition {
				needsGuild = true
			}

			// Check if this condition requires context-based evaluation
			if condition.conditionType == QuestStatusCondition || condition.conditionType == QuestProgressCondition || condition.conditionType == BuffCondition || condition.conditionType == PartyIdCondition || condition.conditionType == PartyLeaderCondition || condition.conditionType == PartySizeCondition || condition.conditionType == PqCustomDataCondition {
				needsContext = true
			}
		}

		// If we need context-based evaluation, use the context provider
		if needsContext {
			ctx, err := p.GetValidationContextProvider().GetValidationContext(characterId)()
			if err != nil {
				return result, fmt.Errorf("failed to get validation context: %w", err)
			}
			return p.ValidateWithContext(resultDecorators...)(ctx, conditionInputs)
		}

		// Get character data with inventory and/or guild if needed
		var characterData character.Model
		var err error
		var charDecorators []model.Decorator[character.Model]

		if needsInventory {
			charDecorators = append(charDecorators, p.characterProcessor.InventoryDecorator)
		}

		if needsGuild {
			charDecorators = append(charDecorators, p.characterProcessor.GuildDecorator)
		}

		if len(charDecorators) > 0 {
			characterData, err = p.characterProcessor.GetById(charDecorators...)(characterId)
		} else {
			characterData, err = p.characterProcessor.GetById()(characterId)
		}

		if err != nil {
			return result, fmt.Errorf("failed to get character data: %w", err)
		}

		// Evaluate each condition
		for _, condition := range conditions {
			conditionResult := condition.Evaluate(characterData)
			result.AddConditionResult(conditionResult)
		}

		// Apply decorators
		return model.Map(model.Decorate(resultDecorators))(func() (ValidationResult, error) {
			return result, nil
		})()
	}
}

// ValidateWithContext validates a list of structured condition inputs using a validation context
func (p *ProcessorImpl) ValidateWithContext(decorators ...model.Decorator[ValidationResult]) func(ctx ValidationContext, conditionInputs []ConditionInput) (ValidationResult, error) {
	return func(ctx ValidationContext, conditionInputs []ConditionInput) (ValidationResult, error) {
		// Create a new validation result
		result := NewValidationResult(ctx.Character().Id())

		// Parse all conditions
		conditions := make([]Condition, 0, len(conditionInputs))

		for _, input := range conditionInputs {
			condition, err := NewConditionBuilder().FromInput(input).Build()
			if err != nil {
				return result, fmt.Errorf("invalid condition: %w", err)
			}
			conditions = append(conditions, condition)
		}

		// Evaluate each condition using the context
		for _, condition := range conditions {
			conditionResult := condition.EvaluateWithContext(ctx)
			result.AddConditionResult(conditionResult)
		}

		// Apply decorators
		return model.Map(model.Decorate(decorators))(func() (ValidationResult, error) {
			return result, nil
		})()
	}
}

// GetValidationContextProvider returns a provider that can create validation contexts
func (p *ProcessorImpl) GetValidationContextProvider() ValidationContextProvider {
	return NewContextBuilderProvider(
		func(characterId uint32) model.Provider[character.Model] {
			return func() (character.Model, error) {
				return p.characterProcessor.GetById(p.characterProcessor.InventoryDecorator)(characterId)
			}
		},
		func(characterId uint32) model.Provider[map[uint32]quest.Model] {
			return func() (map[uint32]quest.Model, error) {
				quests, err := p.questProcessor.GetQuestsByCharacter(characterId)()
				if err != nil {
					return nil, err
				}
				questMap := make(map[uint32]quest.Model, len(quests))
				for _, q := range quests {
					questMap[q.QuestId()] = q
				}
				return questMap, nil
			}
		},
		func(characterId uint32) model.Provider[marriage.Model] {
			return p.marriageProcessor.GetMarriageGifts(characterId)
		},
		func(characterId uint32) model.Provider[buddy.Model] {
			return p.buddyProcessor.GetBuddyList(characterId)
		},
		func(characterId uint32) model.Provider[int] {
			return p.petProcessor.GetSpawnedPetCount(characterId)
		},
		func(characterId uint32) model.Provider[party.Model] {
			return p.partyProcessor.GetPartyByCharacter(characterId)
		},
		p.l,
		p.ctx,
	)
}
