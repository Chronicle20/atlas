package validation

import (
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/quest"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
)

// ValidationContext provides all the data needed for validation
type ValidationContext struct {
	character character.Model
	quests    map[uint32]quest.Model
	marriage  marriage.Model
	buddyList buddy.Model
}

// NewValidationContext creates a new validation context with the provided character
func NewValidationContext(char character.Model) ValidationContext {
	return ValidationContext{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
	}
}

// Character returns the character model
func (ctx ValidationContext) Character() character.Model {
	return ctx.character
}

// Quest returns the quest model for the given quest ID
func (ctx ValidationContext) Quest(questId uint32) (quest.Model, bool) {
	q, exists := ctx.quests[questId]
	return q, exists
}

// Marriage returns the marriage model
func (ctx ValidationContext) Marriage() marriage.Model {
	return ctx.marriage
}

// BuddyList returns the buddy list model
func (ctx ValidationContext) BuddyList() buddy.Model {
	return ctx.buddyList
}

// WithQuest adds a quest to the context
func (ctx ValidationContext) WithQuest(questModel quest.Model) ValidationContext {
	newQuests := make(map[uint32]quest.Model)
	for k, v := range ctx.quests {
		newQuests[k] = v
	}
	newQuests[questModel.Id()] = questModel

	return ValidationContext{
		character: ctx.character,
		quests:    newQuests,
		marriage:  ctx.marriage,
		buddyList: ctx.buddyList,
	}
}

// WithMarriage adds marriage data to the context
func (ctx ValidationContext) WithMarriage(marriageModel marriage.Model) ValidationContext {
	return ValidationContext{
		character: ctx.character,
		quests:    ctx.quests,
		marriage:  marriageModel,
		buddyList: ctx.buddyList,
	}
}

// WithBuddyList adds buddy list data to the context
func (ctx ValidationContext) WithBuddyList(buddyListModel buddy.Model) ValidationContext {
	return ValidationContext{
		character: ctx.character,
		quests:    ctx.quests,
		marriage:  ctx.marriage,
		buddyList: buddyListModel,
	}
}

// ValidationContextBuilder provides a builder pattern for creating validation contexts
type ValidationContextBuilder struct {
	character character.Model
	quests    map[uint32]quest.Model
	marriage  marriage.Model
	buddyList buddy.Model
}

// NewValidationContextBuilder creates a new validation context builder
func NewValidationContextBuilder(char character.Model) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
	}
}

// AddQuest adds a quest to the context being built
func (b *ValidationContextBuilder) AddQuest(questModel quest.Model) *ValidationContextBuilder {
	if b.quests == nil {
		b.quests = make(map[uint32]quest.Model)
	}
	b.quests[questModel.Id()] = questModel
	return b
}

// SetMarriage sets the marriage data for the context being built
func (b *ValidationContextBuilder) SetMarriage(marriageModel marriage.Model) *ValidationContextBuilder {
	b.marriage = marriageModel
	return b
}

// SetBuddyList sets the buddy list data for the context being built
func (b *ValidationContextBuilder) SetBuddyList(buddyListModel buddy.Model) *ValidationContextBuilder {
	b.buddyList = buddyListModel
	return b
}

// Build creates a validation context from the builder
func (b *ValidationContextBuilder) Build() ValidationContext {
	return ValidationContext{
		character: b.character,
		quests:    b.quests,
		marriage:  b.marriage,
		buddyList: b.buddyList,
	}
}

// ValidationContextProvider defines the interface for providing validation contexts
type ValidationContextProvider interface {
	// GetValidationContext returns a provider that builds a validation context for the given character
	GetValidationContext(characterId uint32) model.Provider[ValidationContext]
}

// ContextBuilderProvider provides a way to create validation contexts with data from multiple services
type ContextBuilderProvider struct {
	characterProvider func(uint32) model.Provider[character.Model]
	questProvider     func(uint32) model.Provider[map[uint32]quest.Model]
	marriageProvider  func(uint32) model.Provider[marriage.Model]
	buddyProvider     func(uint32) model.Provider[buddy.Model]
}

// NewContextBuilderProvider creates a new context builder provider
func NewContextBuilderProvider(
	characterProvider func(uint32) model.Provider[character.Model],
	questProvider func(uint32) model.Provider[map[uint32]quest.Model],
	marriageProvider func(uint32) model.Provider[marriage.Model],
	buddyProvider func(uint32) model.Provider[buddy.Model],
) *ContextBuilderProvider {
	return &ContextBuilderProvider{
		characterProvider: characterProvider,
		questProvider:     questProvider,
		marriageProvider:  marriageProvider,
		buddyProvider:     buddyProvider,
	}
}

// GetValidationContext returns a provider that builds a validation context for the given character
func (p *ContextBuilderProvider) GetValidationContext(characterId uint32) model.Provider[ValidationContext] {
	return func() (ValidationContext, error) {
		// Get character data
		char, err := p.characterProvider(characterId)()
		if err != nil {
			return ValidationContext{}, err
		}

		// Start building context
		builder := NewValidationContextBuilder(char)

		// Get quest data if available
		if p.questProvider != nil {
			questsMap, err := p.questProvider(characterId)()
			if err != nil {
				return ValidationContext{}, fmt.Errorf("failed to get quest data: %w", err)
			}
			for _, questModel := range questsMap {
				builder.AddQuest(questModel)
			}
		}

		// Get marriage data if available
		if p.marriageProvider != nil {
			marriageModel, err := p.marriageProvider(characterId)()
			if err != nil {
				return ValidationContext{}, fmt.Errorf("failed to get marriage data: %w", err)
			}
			builder.SetMarriage(marriageModel)
		}

		// Get buddy list data if available
		if p.buddyProvider != nil {
			buddyListModel, err := p.buddyProvider(characterId)()
			if err != nil {
				return ValidationContext{}, fmt.Errorf("failed to get buddy list data: %w", err)
			}
			builder.SetBuddyList(buddyListModel)
		}

		return builder.Build(), nil
	}
}