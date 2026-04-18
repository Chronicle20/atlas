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

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

		// Parse all conditions and figure out which data sources we need
		conditions := make([]Condition, 0, len(conditionInputs))
		needsInventory := false
		needsGuild := false
		needsContext := false
		var ctxReqs ContextRequirements

		for _, input := range conditionInputs {
			condition, err := NewConditionBuilder().FromInput(input).Build()
			if err != nil {
				return result, fmt.Errorf("invalid condition: %w", err)
			}

			conditions = append(conditions, condition)

			if condition.conditionType == ItemCondition {
				needsInventory = true
			}
			if condition.conditionType == GuildLeaderCondition {
				needsGuild = true
			}
			if requiresContextPath(condition.conditionType) {
				needsContext = true
			}
			ctxReqs = ctxReqs.union(requirementsFor(condition.conditionType))
		}

		// If we need context-based evaluation, use the context provider
		if needsContext {
			ctx, err := p.GetValidationContextProvider().GetValidationContext(characterId, ctxReqs)()
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

// requiresContextPath reports whether a condition type can only be evaluated through the
// ValidationContext path. The non-context Evaluate falls back to character-only evaluation,
// so any condition reading quests, marriage, buddy/pet/party, lazy processors (map capacity,
// inventory space, transports, skills, buffs), or PQ data must take the context route.
func requiresContextPath(t ConditionType) bool {
	switch t {
	case QuestStatusCondition, QuestProgressCondition,
		UnclaimedMarriageGiftsCondition,
		BuddyCapacityCondition, PetCountCondition,
		MapCapacityCondition, InventorySpaceCondition,
		TransportAvailableCondition, SkillLevelCondition,
		BuffCondition,
		PartyIdCondition, PartyLeaderCondition, PartySizeCondition,
		PqCustomDataCondition:
		return true
	}
	return false
}

// requirementsFor returns the eager-fetch needs for a single condition. Lazy-processor
// conditions return a zero ContextRequirements because their data is fetched on demand
// inside EvaluateWithContext.
func requirementsFor(t ConditionType) ContextRequirements {
	switch t {
	case QuestStatusCondition, QuestProgressCondition:
		return ContextRequirements{Quests: true}
	case UnclaimedMarriageGiftsCondition:
		return ContextRequirements{Marriage: true}
	case BuddyCapacityCondition:
		return ContextRequirements{Buddy: true}
	case PetCountCondition:
		return ContextRequirements{Pets: true}
	case PartyIdCondition, PartyLeaderCondition, PartySizeCondition:
		return ContextRequirements{Party: true}
	}
	return ContextRequirements{}
}

func (r ContextRequirements) union(o ContextRequirements) ContextRequirements {
	return ContextRequirements{
		Quests:   r.Quests || o.Quests,
		Marriage: r.Marriage || o.Marriage,
		Buddy:    r.Buddy || o.Buddy,
		Pets:     r.Pets || o.Pets,
		Party:    r.Party || o.Party,
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
