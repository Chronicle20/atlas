package validation

import (
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/item"
	npcMap "atlas-query-aggregator/map"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/quest"
	"atlas-query-aggregator/skill"
	"atlas-query-aggregator/transport"
	"context"
	"fmt"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// ValidationContext provides all the data needed for validation
type ValidationContext struct {
	character  character.Model
	quests     map[uint32]quest.Model
	skills     map[uint32]skill.Model
	marriage   marriage.Model
	buddyList  buddy.Model
	petCount   int
	mapP       npcMap.Processor
	itemP      item.Processor
	transportP transport.Processor
	skillP     skill.Processor
	l          logrus.FieldLogger
	ctx        context.Context
}

// NewValidationContext creates a new validation context with the provided character
func NewValidationContext(char character.Model) ValidationContext {
	return ValidationContext{
		character:  char,
		quests:     make(map[uint32]quest.Model),
		skills:     make(map[uint32]skill.Model),
		marriage:   marriage.NewModel(char.Id(), false),
		buddyList:  buddy.NewModel(char.Id(), 0),
		petCount:   0,
		mapP:       nil,
		itemP:      nil,
		transportP: nil,
		skillP:     nil,
		l:          nil,
		ctx:        nil,
	}
}

// NewValidationContextWithLogger creates a new validation context with logger and context for map queries
func NewValidationContextWithLogger(char character.Model, l logrus.FieldLogger, ctx context.Context) ValidationContext {
	return ValidationContext{
		character:  char,
		quests:     make(map[uint32]quest.Model),
		skills:     make(map[uint32]skill.Model),
		marriage:   marriage.NewModel(char.Id(), false),
		buddyList:  buddy.NewModel(char.Id(), 0),
		petCount:   0,
		mapP:       npcMap.NewProcessor(l, ctx),
		itemP:      item.NewProcessor(l, ctx),
		transportP: transport.NewProcessor(l, ctx),
		skillP:     skill.NewProcessor(l, ctx),
		l:          l,
		ctx:        ctx,
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

// Skill returns the skill model for the given skill ID
func (ctx ValidationContext) Skill(skillId uint32) (skill.Model, bool) {
	s, exists := ctx.skills[skillId]
	return s, exists
}

// GetSkillLevel returns the level of a skill, or 0 if not found
// This method supports lazy loading via the skill processor if available
func (ctx ValidationContext) GetSkillLevel(skillId uint32) byte {
	// First check local cache
	if s, exists := ctx.skills[skillId]; exists {
		return s.Level()
	}

	// If skill processor is available, query it
	if ctx.skillP != nil {
		level, err := ctx.skillP.GetSkillLevel(ctx.character.Id(), skillId)()
		if err != nil {
			if ctx.l != nil {
				ctx.l.WithError(err).Debugf("Failed to get skill level for skill %d", skillId)
			}
			return 0
		}
		return level
	}

	return 0
}

// SkillProcessor returns the skill processor for querying skill data
// Returns nil if not available (graceful degradation)
func (ctx ValidationContext) SkillProcessor() skill.Processor {
	return ctx.skillP
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

// ItemProcessor returns the item processor for querying item data
// Returns nil if not available (graceful degradation)
func (ctx ValidationContext) ItemProcessor() item.Processor {
	return ctx.itemP
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

// GetTransportState returns the transport state for a given start map ID
// Returns "unknown" if transport processor is not available or on error (graceful degradation)
func (ctx ValidationContext) GetTransportState(mapId _map.Id) string {
	// If no transport processor available, return "unknown" (graceful degradation)
	if ctx.transportP == nil {
		if ctx.l != nil {
			ctx.l.Warnf("Transport processor not available, returning 'unknown' for map [%d]", mapId)
		}
		return "unknown"
	}

	// Query transport route
	route, err := ctx.transportP.GetRouteByStartMap(mapId)
	if err != nil {
		if ctx.l != nil {
			ctx.l.WithError(err).Warnf("Failed to get transport state for map [%d], using 'unknown'", mapId)
		}
		return "unknown"
	}

	return route.State()
}

// WithQuest adds a quest to the context
func (ctx ValidationContext) WithQuest(questModel quest.Model) ValidationContext {
	newQuests := make(map[uint32]quest.Model)
	for k, v := range ctx.quests {
		newQuests[k] = v
	}
	newQuests[questModel.QuestId()] = questModel

	return ValidationContext{
		character:  ctx.character,
		quests:     newQuests,
		skills:     ctx.skills,
		marriage:   ctx.marriage,
		buddyList:  ctx.buddyList,
		petCount:   ctx.petCount,
		mapP:       ctx.mapP,
		itemP:      ctx.itemP,
		transportP: ctx.transportP,
		skillP:     ctx.skillP,
		l:          ctx.l,
		ctx:        ctx.ctx,
	}
}

// WithSkill adds a skill to the context
func (ctx ValidationContext) WithSkill(skillModel skill.Model) ValidationContext {
	newSkills := make(map[uint32]skill.Model)
	for k, v := range ctx.skills {
		newSkills[k] = v
	}
	newSkills[skillModel.Id()] = skillModel

	return ValidationContext{
		character:  ctx.character,
		quests:     ctx.quests,
		skills:     newSkills,
		marriage:   ctx.marriage,
		buddyList:  ctx.buddyList,
		petCount:   ctx.petCount,
		mapP:       ctx.mapP,
		itemP:      ctx.itemP,
		transportP: ctx.transportP,
		skillP:     ctx.skillP,
		l:          ctx.l,
		ctx:        ctx.ctx,
	}
}

// WithMarriage adds marriage data to the context
func (ctx ValidationContext) WithMarriage(marriageModel marriage.Model) ValidationContext {
	return ValidationContext{
		character:  ctx.character,
		quests:     ctx.quests,
		skills:     ctx.skills,
		marriage:   marriageModel,
		buddyList:  ctx.buddyList,
		petCount:   ctx.petCount,
		mapP:       ctx.mapP,
		itemP:      ctx.itemP,
		transportP: ctx.transportP,
		skillP:     ctx.skillP,
		l:          ctx.l,
		ctx:        ctx.ctx,
	}
}

// WithBuddyList adds buddy list data to the context
func (ctx ValidationContext) WithBuddyList(buddyListModel buddy.Model) ValidationContext {
	return ValidationContext{
		character:  ctx.character,
		quests:     ctx.quests,
		skills:     ctx.skills,
		marriage:   ctx.marriage,
		buddyList:  buddyListModel,
		petCount:   ctx.petCount,
		mapP:       ctx.mapP,
		itemP:      ctx.itemP,
		transportP: ctx.transportP,
		skillP:     ctx.skillP,
		l:          ctx.l,
		ctx:        ctx.ctx,
	}
}

// WithPetCount sets the pet count in the context
func (ctx ValidationContext) WithPetCount(count int) ValidationContext {
	return ValidationContext{
		character:  ctx.character,
		quests:     ctx.quests,
		skills:     ctx.skills,
		marriage:   ctx.marriage,
		buddyList:  ctx.buddyList,
		petCount:   count,
		mapP:       ctx.mapP,
		itemP:      ctx.itemP,
		transportP: ctx.transportP,
		skillP:     ctx.skillP,
		l:          ctx.l,
		ctx:        ctx.ctx,
	}
}

// ValidationContextBuilder provides a builder pattern for creating validation contexts
type ValidationContextBuilder struct {
	character  character.Model
	quests     map[uint32]quest.Model
	skills     map[uint32]skill.Model
	marriage   marriage.Model
	buddyList  buddy.Model
	petCount   int
	mapP       npcMap.Processor
	itemP      item.Processor
	transportP transport.Processor
	skillP     skill.Processor
	l          logrus.FieldLogger
	ctx        context.Context
}

// NewValidationContextBuilder creates a new validation context builder
func NewValidationContextBuilder(char character.Model) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character:  char,
		quests:     make(map[uint32]quest.Model),
		skills:     make(map[uint32]skill.Model),
		marriage:   marriage.NewModel(char.Id(), false),
		buddyList:  buddy.NewModel(char.Id(), 0),
		petCount:   0,
		mapP:       nil,
		itemP:      nil,
		transportP: nil,
		skillP:     nil,
		l:          nil,
		ctx:        nil,
	}
}

// NewValidationContextBuilderWithLogger creates a new validation context builder with logger and context
func NewValidationContextBuilderWithLogger(char character.Model, l logrus.FieldLogger, ctx context.Context) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character:  char,
		quests:     make(map[uint32]quest.Model),
		skills:     make(map[uint32]skill.Model),
		marriage:   marriage.NewModel(char.Id(), false),
		buddyList:  buddy.NewModel(char.Id(), 0),
		petCount:   0,
		mapP:       npcMap.NewProcessor(l, ctx),
		itemP:      item.NewProcessor(l, ctx),
		transportP: transport.NewProcessor(l, ctx),
		skillP:     skill.NewProcessor(l, ctx),
		l:          l,
		ctx:        ctx,
	}
}

// AddQuest adds a quest to the context being built
func (b *ValidationContextBuilder) AddQuest(questModel quest.Model) *ValidationContextBuilder {
	if b.quests == nil {
		b.quests = make(map[uint32]quest.Model)
	}
	b.quests[questModel.QuestId()] = questModel
	return b
}

// AddSkill adds a skill to the context being built
func (b *ValidationContextBuilder) AddSkill(skillModel skill.Model) *ValidationContextBuilder {
	if b.skills == nil {
		b.skills = make(map[uint32]skill.Model)
	}
	b.skills[skillModel.Id()] = skillModel
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
		character:  b.character,
		quests:     b.quests,
		skills:     b.skills,
		marriage:   b.marriage,
		buddyList:  b.buddyList,
		petCount:   b.petCount,
		mapP:       b.mapP,
		itemP:      b.itemP,
		transportP: b.transportP,
		skillP:     b.skillP,
		l:          b.l,
		ctx:        b.ctx,
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