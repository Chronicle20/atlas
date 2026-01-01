package validation

import (
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/character"
	npcMap "atlas-query-aggregator/map"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/quest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// ValidationContext provides all the data needed for validation
type ValidationContext struct {
	character character.Model
	quests    map[uint32]quest.Model
	marriage  marriage.Model
	buddyList buddy.Model
	petCount  int
	mapP      npcMap.Processor
	l         logrus.FieldLogger
	ctx       context.Context
}

// NewValidationContext creates a new validation context with the provided character
func NewValidationContext(char character.Model) ValidationContext {
	return ValidationContext{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
		petCount:  0,
		mapP:      nil,
		l:         nil,
		ctx:       nil,
	}
}

// NewValidationContextWithLogger creates a new validation context with logger and context for map queries
func NewValidationContextWithLogger(char character.Model, l logrus.FieldLogger, ctx context.Context) ValidationContext {
	return ValidationContext{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
		petCount:  0,
		mapP:      npcMap.NewProcessor(l, ctx),
		l:         l,
		ctx:       ctx,
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

// PetCount returns the count of spawned pets
func (ctx ValidationContext) PetCount() int {
	return ctx.petCount
}

// GetPlayerCountInMap returns the player count for a given map
// Returns 0 if map processor is not available or on error (graceful degradation)
func (ctx ValidationContext) GetPlayerCountInMap(worldId byte, channelId byte, mapId uint32) int {
	// If no map processor available, return 0 (graceful degradation)
	if ctx.mapP == nil {
		if ctx.l != nil {
			ctx.l.Warnf("Map processor not available, returning 0 for map [%d]", mapId)
		}
		return 0
	}

	// If worldId is not set, try to get from character
	if worldId == 0 {
		worldId = byte(ctx.character.WorldId())
	}

	// Query player count
	count, err := ctx.mapP.GetPlayerCountInMap(worldId, channelId, mapId)
	if err != nil {
		if ctx.l != nil {
			ctx.l.WithError(err).Warnf("Failed to get player count for map [%d], using 0", mapId)
		}
		return 0
	}

	return count
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
		petCount:  ctx.petCount,
		mapP:      ctx.mapP,
		l:         ctx.l,
		ctx:       ctx.ctx,
	}
}

// WithMarriage adds marriage data to the context
func (ctx ValidationContext) WithMarriage(marriageModel marriage.Model) ValidationContext {
	return ValidationContext{
		character: ctx.character,
		quests:    ctx.quests,
		marriage:  marriageModel,
		buddyList: ctx.buddyList,
		petCount:  ctx.petCount,
		mapP:      ctx.mapP,
		l:         ctx.l,
		ctx:       ctx.ctx,
	}
}

// WithBuddyList adds buddy list data to the context
func (ctx ValidationContext) WithBuddyList(buddyListModel buddy.Model) ValidationContext {
	return ValidationContext{
		character: ctx.character,
		quests:    ctx.quests,
		marriage:  ctx.marriage,
		buddyList: buddyListModel,
		petCount:  ctx.petCount,
		mapP:      ctx.mapP,
		l:         ctx.l,
		ctx:       ctx.ctx,
	}
}

// WithPetCount sets the pet count in the context
func (ctx ValidationContext) WithPetCount(count int) ValidationContext {
	return ValidationContext{
		character: ctx.character,
		quests:    ctx.quests,
		marriage:  ctx.marriage,
		buddyList: ctx.buddyList,
		petCount:  count,
		mapP:      ctx.mapP,
		l:         ctx.l,
		ctx:       ctx.ctx,
	}
}

// ValidationContextBuilder provides a builder pattern for creating validation contexts
type ValidationContextBuilder struct {
	character character.Model
	quests    map[uint32]quest.Model
	marriage  marriage.Model
	buddyList buddy.Model
	petCount  int
	mapP      npcMap.Processor
	l         logrus.FieldLogger
	ctx       context.Context
}

// NewValidationContextBuilder creates a new validation context builder
func NewValidationContextBuilder(char character.Model) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
		petCount:  0,
		mapP:      nil,
		l:         nil,
		ctx:       nil,
	}
}

// NewValidationContextBuilderWithLogger creates a new validation context builder with logger and context
func NewValidationContextBuilderWithLogger(char character.Model, l logrus.FieldLogger, ctx context.Context) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character: char,
		quests:    make(map[uint32]quest.Model),
		marriage:  marriage.NewModel(char.Id(), false),
		buddyList: buddy.NewModel(char.Id(), 0),
		petCount:  0,
		mapP:      npcMap.NewProcessor(l, ctx),
		l:         l,
		ctx:       ctx,
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

// SetPetCount sets the pet count for the context being built
func (b *ValidationContextBuilder) SetPetCount(count int) *ValidationContextBuilder {
	b.petCount = count
	return b
}

// Build creates a validation context from the builder
func (b *ValidationContextBuilder) Build() ValidationContext {
	return ValidationContext{
		character: b.character,
		quests:    b.quests,
		marriage:  b.marriage,
		buddyList: b.buddyList,
		petCount:  b.petCount,
		mapP:      b.mapP,
		l:         b.l,
		ctx:       b.ctx,
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
	petCountProvider  func(uint32) model.Provider[int]
	l                 logrus.FieldLogger
	ctx               context.Context
}

// NewContextBuilderProvider creates a new context builder provider
func NewContextBuilderProvider(
	characterProvider func(uint32) model.Provider[character.Model],
	questProvider func(uint32) model.Provider[map[uint32]quest.Model],
	marriageProvider func(uint32) model.Provider[marriage.Model],
	buddyProvider func(uint32) model.Provider[buddy.Model],
	petCountProvider func(uint32) model.Provider[int],
	l logrus.FieldLogger,
	ctx context.Context,
) *ContextBuilderProvider {
	return &ContextBuilderProvider{
		characterProvider: characterProvider,
		questProvider:     questProvider,
		marriageProvider:  marriageProvider,
		buddyProvider:     buddyProvider,
		petCountProvider:  petCountProvider,
		l:                 l,
		ctx:               ctx,
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

		// Start building context with logger and context for map queries
		builder := NewValidationContextBuilderWithLogger(char, p.l, p.ctx)

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

		// Get pet count data if available
		if p.petCountProvider != nil {
			petCount, err := p.petCountProvider(characterId)()
			if err != nil {
				return ValidationContext{}, fmt.Errorf("failed to get pet count data: %w", err)
			}
			builder.SetPetCount(petCount)
		}

		return builder.Build(), nil
	}
}