package validation

import (
	"atlas-query-aggregator/area_info"
	"atlas-query-aggregator/buddy"
	"atlas-query-aggregator/buff"
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/item"
	npcMap "atlas-query-aggregator/map"
	"atlas-query-aggregator/marriage"
	"atlas-query-aggregator/monsterbook"
	"atlas-query-aggregator/party"
	"atlas-query-aggregator/party_quest"
	"atlas-query-aggregator/playernpc"
	"atlas-query-aggregator/quest"
	"atlas-query-aggregator/skill"
	"atlas-query-aggregator/transport"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ValidationContextBuilder provides a builder pattern for creating validation contexts
type ValidationContextBuilder struct {
	character   character.Model
	quests      map[uint32]quest.Model
	skills      map[uint32]skill.Model
	marriage    marriage.Model
	buddyList   buddy.Model
	party       party.Model
	petCount    int
	spawnedPets []SpawnedPet
	mapP        npcMap.Processor
	itemP       item.Processor
	transportP  transport.Processor
	skillP      skill.Processor
	buffP       buff.Processor
	partyP      party.Processor
	partyQuestP party_quest.Processor
	mbP         monsterbook.Processor
	playerNpcP  playernpc.Processor
	areaInfoP   area_info.Processor
	l           logrus.FieldLogger
	ctx         context.Context
}

// NewValidationContextBuilder creates a new validation context builder
func NewValidationContextBuilder(char character.Model) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character:   char,
		quests:      make(map[uint32]quest.Model),
		skills:      make(map[uint32]skill.Model),
		marriage:    marriage.NewModel(char.Id(), false),
		buddyList:   buddy.NewModel(char.Id(), 0),
		petCount:    0,
		spawnedPets: nil,
		mapP:        nil,
		itemP:       nil,
		transportP:  nil,
		skillP:      nil,
		buffP:       nil,
		partyP:      nil,
		partyQuestP: nil,
		mbP:         nil,
		playerNpcP:  nil,
		areaInfoP:   nil,
		l:           nil,
		ctx:         nil,
	}
}

// NewValidationContextBuilderWithLogger creates a new validation context builder with logger and context
func NewValidationContextBuilderWithLogger(char character.Model, l logrus.FieldLogger, ctx context.Context) *ValidationContextBuilder {
	return &ValidationContextBuilder{
		character:   char,
		quests:      make(map[uint32]quest.Model),
		skills:      make(map[uint32]skill.Model),
		marriage:    marriage.NewModel(char.Id(), false),
		buddyList:   buddy.NewModel(char.Id(), 0),
		petCount:    0,
		spawnedPets: nil,
		mapP:        npcMap.NewProcessor(l, ctx),
		itemP:       item.NewProcessor(l, ctx),
		transportP:  transport.NewProcessor(l, ctx),
		skillP:      skill.NewProcessor(l, ctx),
		buffP:       buff.NewProcessor(l, ctx),
		partyP:      party.NewProcessor(l, ctx),
		partyQuestP: party_quest.NewProcessor(l, ctx),
		mbP:         monsterbook.NewProcessor(l, ctx),
		playerNpcP:  playernpc.NewProcessor(l, ctx),
		areaInfoP:   area_info.NewProcessor(l, ctx),
		l:           l,
		ctx:         ctx,
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

// SetParty sets the party data for the context being built
func (b *ValidationContextBuilder) SetParty(partyModel party.Model) *ValidationContextBuilder {
	b.party = partyModel
	return b
}

// SetPetCount sets the pet count for the context being built
func (b *ValidationContextBuilder) SetPetCount(count int) *ValidationContextBuilder {
	b.petCount = count
	return b
}

// SetSpawnedPets sets the spawned-pet detail for the context being built.
func (b *ValidationContextBuilder) SetSpawnedPets(pets []SpawnedPet) *ValidationContextBuilder {
	b.spawnedPets = pets
	return b
}

// SetMonsterBookProcessor overrides the monster book processor on the
// builder, primarily so tests can inject a fake.
func (b *ValidationContextBuilder) SetMonsterBookProcessor(p monsterbook.Processor) *ValidationContextBuilder {
	b.mbP = p
	return b
}

// SetPlayerNpcProcessor overrides the player-npc processor on the builder,
// primarily so tests can inject a fake.
func (b *ValidationContextBuilder) SetPlayerNpcProcessor(p playernpc.Processor) *ValidationContextBuilder {
	b.playerNpcP = p
	return b
}

// Build creates a validation context from the builder
func (b *ValidationContextBuilder) Build() ValidationContext {
	return ValidationContext{
		character:   b.character,
		quests:      b.quests,
		skills:      b.skills,
		marriage:    b.marriage,
		buddyList:   b.buddyList,
		party:       b.party,
		petCount:    b.petCount,
		spawnedPets: b.spawnedPets,
		mapP:        b.mapP,
		itemP:       b.itemP,
		transportP:  b.transportP,
		skillP:      b.skillP,
		buffP:       b.buffP,
		partyP:      b.partyP,
		partyQuestP: b.partyQuestP,
		mbP:         b.mbP,
		playerNpcP:  b.playerNpcP,
		areaInfoP:   b.areaInfoP,
		l:           b.l,
		ctx:         b.ctx,
	}
}

// ConditionBuilder is used to safely construct Condition objects
type ConditionBuilder struct {
	conditionType   ConditionType
	operator        Operator
	value           int
	values          []int
	referenceId     *uint32
	step            string
	worldId         world.Id
	channelId       channel.Id
	includeEquipped bool
	valueString     string
	err             error
}

// NewConditionBuilder creates a new condition builder
func NewConditionBuilder() *ConditionBuilder {
	return &ConditionBuilder{}
}

// SetType sets the condition type
func (b *ConditionBuilder) SetType(condType string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch ConditionType(condType) {
	case JobCondition, MesoCondition, MapCondition, FameCondition, ItemCondition, GenderCondition, LevelCondition, RebornsCondition, DojoPointsCondition, VanquisherKillsCondition, GmLevelCondition, GuildIdCondition, GuildRankCondition, QuestStatusCondition, QuestProgressCondition, UnclaimedMarriageGiftsCondition, StrengthCondition, DexterityCondition, IntelligenceCondition, LuckCondition, GuildLeaderCondition, BuddyCapacityCondition, PetCountCondition, MapCapacityCondition, InventorySpaceCondition, TransportAvailableCondition, TransportInTransitCondition, SkillLevelCondition, HpCondition, MaxHpCondition, BuffCondition, ExcessSPCondition, PartyIdCondition, PartyLeaderCondition, PartySizeCondition, PqCustomDataCondition, MonsterBookCountCondition, PetTamenessCondition, CanSpawnPlayerNpcCondition, AreaInfoCondition:
		b.conditionType = ConditionType(condType)
	default:
		b.err = fmt.Errorf("unsupported condition type: %s", condType)
	}
	return b
}

// SetOperator sets the operator
func (b *ConditionBuilder) SetOperator(op string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch Operator(op) {
	case Equals, GreaterThan, LessThan, GreaterEqual, LessEqual, In:
		b.operator = Operator(op)
	default:
		b.err = fmt.Errorf("unsupported operator: %s", op)
	}
	return b
}

// SetValue sets the value
func (b *ConditionBuilder) SetValue(value int) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.value = value
	return b
}

// SetValues sets the values for "in" operator
func (b *ConditionBuilder) SetValues(values []int) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.values = values
	return b
}

// SetReferenceId sets the reference ID (for quest validation, item conditions, etc.)
func (b *ConditionBuilder) SetReferenceId(referenceId uint32) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.referenceId = &referenceId
	return b
}

// SetStep sets the step for quest progress validation
func (b *ConditionBuilder) SetStep(step string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.step = step
	return b
}

// SetValueString sets the string value (for areaInfo conditions)
func (b *ConditionBuilder) SetValueString(valueString string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.valueString = valueString
	return b
}

// SetIncludeEquipped sets whether to include equipped items in item condition checks
func (b *ConditionBuilder) SetIncludeEquipped(includeEquipped bool) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.includeEquipped = includeEquipped
	return b
}

// FromInput creates a condition builder from a ConditionInput
func (b *ConditionBuilder) FromInput(input ConditionInput) *ConditionBuilder {
	b.SetType(input.Type)
	b.SetOperator(input.Operator)
	b.SetValue(input.Value)

	// Set values for "in" operator
	if len(input.Values) > 0 {
		b.SetValues(input.Values)
	}

	if input.ReferenceId != 0 {
		b.SetReferenceId(input.ReferenceId)
	}

	// Set step for quest progress validation
	if input.Step != "" {
		b.SetStep(input.Step)
	}

	// Set valueString for areaInfo conditions
	if input.ValueString != "" {
		b.SetValueString(input.ValueString)
	}

	// Set worldId and channelId for mapCapacity conditions
	if input.WorldId != 0 {
		b.worldId = input.WorldId
	}
	if input.ChannelId != 0 {
		b.channelId = input.ChannelId
	}

	// Set includeEquipped for item conditions
	b.includeEquipped = input.IncludeEquipped

	// Validate required fields for specific condition types
	switch ConditionType(input.Type) {
	case ItemCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for item conditions")
		}
	case QuestStatusCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for quest conditions")
		}
	case QuestProgressCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for quest conditions")
		}
		if input.Step == "" {
			b.err = fmt.Errorf("step is required for quest progress conditions")
		}
	case MapCapacityCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for mapCapacity conditions")
		}
	case TransportAvailableCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for transportAvailable conditions")
		}
	case TransportInTransitCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for transportInTransit conditions")
		}
	case SkillLevelCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for skillLevel conditions")
		}
	case BuffCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId is required for buff conditions")
		}
	case ExcessSPCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId (base level) is required for excessSp conditions")
		}
	case PqCustomDataCondition:
		if input.Step == "" {
			b.err = fmt.Errorf("step (custom data key) is required for pqCustomData conditions")
		}
	case PetTamenessCondition:
		if len(input.Values) == 0 {
			b.err = fmt.Errorf("values (pet template ids) required for petTameness conditions")
		}
	case CanSpawnPlayerNpcCondition:
		if input.ReferenceId == 0 {
			b.err = fmt.Errorf("referenceId (mapId) is required for canSpawnPlayerNpc conditions")
		}
	case AreaInfoCondition:
		if input.ReferenceId == 0 || input.ValueString == "" {
			b.err = fmt.Errorf("referenceId and valueString are required for areaInfo conditions")
		}
	}

	return b
}

// Validate validates the builder state
func (b *ConditionBuilder) Validate() *ConditionBuilder {
	if b.err != nil {
		return b
	}

	// Check if condition type is set
	if b.conditionType == "" {
		b.err = fmt.Errorf("condition type is required")
		return b
	}

	// Check if operator is set
	if b.operator == "" {
		b.err = fmt.Errorf("operator is required")
		return b
	}

	// Check if referenceId is set for conditions that require it
	switch b.conditionType {
	case ItemCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for item conditions")
			return b
		}
	case QuestStatusCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for quest conditions")
			return b
		}
	case QuestProgressCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for quest conditions")
			return b
		}
		if b.step == "" {
			b.err = fmt.Errorf("step is required for quest progress conditions")
			return b
		}
	case MapCapacityCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for mapCapacity conditions")
			return b
		}
	case InventorySpaceCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for inventorySpace conditions")
			return b
		}
	case TransportAvailableCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for transportAvailable conditions")
			return b
		}
	case TransportInTransitCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for transportInTransit conditions")
			return b
		}
	case SkillLevelCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for skillLevel conditions")
			return b
		}
	case BuffCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId is required for buff conditions")
			return b
		}
	case ExcessSPCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId (base level) is required for excessSp conditions")
			return b
		}
	case PqCustomDataCondition:
		if b.step == "" {
			b.err = fmt.Errorf("step (custom data key) is required for pqCustomData conditions")
			return b
		}
	case CanSpawnPlayerNpcCondition:
		if b.referenceId == nil {
			b.err = fmt.Errorf("referenceId (mapId) is required for canSpawnPlayerNpc conditions")
			return b
		}
	case AreaInfoCondition:
		if b.referenceId == nil || b.valueString == "" {
			b.err = fmt.Errorf("referenceId and valueString are required for areaInfo conditions")
			return b
		}
	}

	return b
}

// Build builds a Condition from the builder
func (b *ConditionBuilder) Build() (Condition, error) {
	b.Validate()

	if b.err != nil {
		return Condition{}, b.err
	}

	condition := Condition{
		conditionType:   b.conditionType,
		operator:        b.operator,
		value:           b.value,
		values:          b.values,
		step:            b.step,
		worldId:         b.worldId,
		channelId:       b.channelId,
		includeEquipped: b.includeEquipped,
		valueString:     b.valueString,
	}

	if b.referenceId != nil {
		condition.referenceId = *b.referenceId
	}

	return condition, nil
}
