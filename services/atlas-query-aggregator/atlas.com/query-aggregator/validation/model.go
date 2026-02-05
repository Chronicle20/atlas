package validation

import (
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/quest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

// ConditionType represents the type of condition to validate
type ConditionType string

const (
	JobCondition                    ConditionType = "jobId"
	MesoCondition                   ConditionType = "meso"
	MapCondition                    ConditionType = "mapId"
	FameCondition                   ConditionType = "fame"
	ItemCondition                   ConditionType = "item"
	GenderCondition                 ConditionType = "gender"
	LevelCondition                  ConditionType = "level"
	RebornsCondition                ConditionType = "reborns"
	DojoPointsCondition             ConditionType = "dojoPoints"
	VanquisherKillsCondition        ConditionType = "vanquisherKills"
	GmLevelCondition                ConditionType = "gmLevel"
	GuildIdCondition                ConditionType = "guildId"
	GuildLeaderCondition            ConditionType = "guildLeader"
	GuildRankCondition              ConditionType = "guildRank"
	QuestStatusCondition            ConditionType = "questStatus"
	QuestProgressCondition          ConditionType = "questProgress"
	UnclaimedMarriageGiftsCondition ConditionType = "hasUnclaimedMarriageGifts"
	StrengthCondition               ConditionType = "strength"
	DexterityCondition              ConditionType = "dexterity"
	IntelligenceCondition           ConditionType = "intelligence"
	LuckCondition                   ConditionType = "luck"
	BuddyCapacityCondition          ConditionType = "buddyCapacity"
	PetCountCondition               ConditionType = "petCount"
	MapCapacityCondition            ConditionType = "mapCapacity"
	InventorySpaceCondition         ConditionType = "inventorySpace"
	TransportAvailableCondition     ConditionType = "transportAvailable"
	SkillLevelCondition             ConditionType = "skillLevel"
	HpCondition                     ConditionType = "hp"
	MaxHpCondition                  ConditionType = "maxHp"
	BuffCondition                   ConditionType = "buff"
	ExcessSPCondition               ConditionType = "excessSp"
)

// Operator represents the comparison operator in a condition
type Operator string

const (
	Equals       Operator = "="
	GreaterThan  Operator = ">"
	LessThan     Operator = "<"
	GreaterEqual Operator = ">="
	LessEqual    Operator = "<="
	In           Operator = "in"
)

// ConditionInput represents the structured input for creating a condition
type ConditionInput struct {
	Type            string     `json:"type"`                      // e.g., "jobId", "meso", "item"
	Operator        string     `json:"operator"`                  // e.g., "=", ">=", "<", "in"
	Value           int        `json:"value"`                     // Value or quantity (for single value operators)
	Values          []int      `json:"values,omitempty"`          // Values for "in" operator
	ReferenceId     uint32     `json:"referenceId,omitempty"`     // For quest validation, item checks, etc.
	Step            string     `json:"step,omitempty"`            // For quest progress validation
	ItemId          uint32     `json:"itemId,omitempty"`          // Deprecated: use ReferenceId instead
	WorldId         world.Id   `json:"worldId,omitempty"`         // For mapCapacity conditions
	ChannelId       channel.Id `json:"channelId,omitempty"`       // For mapCapacity conditions
	IncludeEquipped bool       `json:"includeEquipped,omitempty"` // For item conditions: also check equipped items
}

// ConditionResult represents the result of a condition evaluation
type ConditionResult struct {
	Passed      bool
	Description string
	Type        ConditionType
	Operator    Operator
	Value       int
	ItemId      uint32
	ActualValue int
}

// Condition represents a validation condition
type Condition struct {
	conditionType   ConditionType
	operator        Operator
	value           int
	values          []int      // Used for "in" operator
	referenceId     uint32     // Used for quest validation, item conditions, etc.
	step            string     // Used for quest progress validation
	worldId         world.Id   // Used for mapCapacity conditions
	channelId       channel.Id // Used for mapCapacity conditions
	includeEquipped bool       // For item conditions: also check equipped items
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
	case JobCondition, MesoCondition, MapCondition, FameCondition, ItemCondition, GenderCondition, LevelCondition, RebornsCondition, DojoPointsCondition, VanquisherKillsCondition, GmLevelCondition, GuildIdCondition, GuildRankCondition, QuestStatusCondition, QuestProgressCondition, UnclaimedMarriageGiftsCondition, StrengthCondition, DexterityCondition, IntelligenceCondition, LuckCondition, GuildLeaderCondition, BuddyCapacityCondition, PetCountCondition, MapCapacityCondition, InventorySpaceCondition, TransportAvailableCondition, SkillLevelCondition, HpCondition, MaxHpCondition, BuffCondition, ExcessSPCondition:
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

// SetItemId sets the item ID (deprecated: use SetReferenceId instead)
func (b *ConditionBuilder) SetItemId(itemId uint32) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.referenceId = &itemId
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

	// Handle ReferenceId (preferred) or ItemId (deprecated)
	if input.ReferenceId != 0 {
		b.SetReferenceId(input.ReferenceId)
	} else if input.ItemId != 0 {
		b.SetReferenceId(input.ItemId) // Migrate ItemId to ReferenceId
	}

	// Set step for quest progress validation
	if input.Step != "" {
		b.SetStep(input.Step)
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
		if input.ReferenceId == 0 && input.ItemId == 0 {
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
	}

	if b.referenceId != nil {
		condition.referenceId = *b.referenceId
	}

	return condition, nil
}

// Evaluate evaluates the condition against a character model
// Returns a structured ConditionResult with evaluation details
func (c Condition) Evaluate(character character.Model) ConditionResult {
	var actualValue int
	var passed bool
	var description string
	var itemId uint32

	// Get the actual value from the character model based on condition type
	switch c.conditionType {
	case JobCondition:
		actualValue = int(character.JobId())
		description = fmt.Sprintf("Job ID %s %d", c.operator, c.value)
	case MesoCondition:
		actualValue = int(character.Meso())
		description = fmt.Sprintf("Meso %s %d", c.operator, c.value)
	case MapCondition:
		actualValue = int(character.MapId())
		description = fmt.Sprintf("Map ID %s %d", c.operator, c.value)
	case FameCondition:
		actualValue = int(character.Fame())
		description = fmt.Sprintf("Fame %s %d", c.operator, c.value)
	case GenderCondition:
		actualValue = int(character.Gender())
		description = fmt.Sprintf("Gender %s %d", c.operator, c.value)
	case LevelCondition:
		actualValue = int(character.Level())
		description = fmt.Sprintf("Level %s %d", c.operator, c.value)
	case HpCondition:
		actualValue = int(character.Hp())
		description = fmt.Sprintf("HP %s %d", c.operator, c.value)
	case MaxHpCondition:
		actualValue = int(character.MaxHp())
		description = fmt.Sprintf("Max HP %s %d", c.operator, c.value)
	case RebornsCondition:
		actualValue = int(character.Reborns())
		description = fmt.Sprintf("Reborns %s %d", c.operator, c.value)
	case DojoPointsCondition:
		actualValue = int(character.DojoPoints())
		description = fmt.Sprintf("Dojo Points %s %d", c.operator, c.value)
	case VanquisherKillsCondition:
		actualValue = int(character.VanquisherKills())
		description = fmt.Sprintf("Vanquisher Kills %s %d", c.operator, c.value)
	case GmLevelCondition:
		actualValue = character.GmLevel()
		description = fmt.Sprintf("GM Level %s %d", c.operator, c.value)
	case GuildIdCondition:
		actualValue = int(character.Guild().Id())
		if actualValue == 0 {
			description = fmt.Sprintf("Guild ID %s %d (character not in guild)", c.operator, c.value)
		} else {
			description = fmt.Sprintf("Guild ID %s %d", c.operator, c.value)
		}
	case GuildLeaderCondition:
		// For guild leader conditions, we need to check if the character is a guild leader
		// Get the guild from the character model
		guild := character.Guild()

		// Check if the character is the guild leader
		if guild.Id() == 0 {
			// Character has no guild
			actualValue = 0
		} else if guild.LeaderId() == character.Id() {
			// Character is the guild leader
			actualValue = 1
		} else {
			// Character is not the guild leader
			actualValue = 0
		}

		description = fmt.Sprintf("Guild Leader %s %d", c.operator, c.value)
	case GuildRankCondition:
		actualValue = character.Guild().MemberRank(character.Id())
		if character.Guild().Id() == 0 {
			description = fmt.Sprintf("Guild Rank %s %d (character not in guild)", c.operator, c.value)
		} else {
			description = fmt.Sprintf("Guild Rank %s %d", c.operator, c.value)
		}
	case QuestStatusCondition:
		// Quest status validation requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Quest %d Status validation requires ValidationContext", c.referenceId),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: int(quest.StateNotStarted),
		}
	case QuestProgressCondition:
		// Quest progress validation requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Quest %d Progress validation (step: %s) requires ValidationContext", c.referenceId, c.step),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	case UnclaimedMarriageGiftsCondition:
		// Marriage gifts validation requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Unclaimed Marriage Gifts validation requires ValidationContext"),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	case StrengthCondition:
		actualValue = int(character.Strength())
		description = fmt.Sprintf("Strength %s %d", c.operator, c.value)
	case DexterityCondition:
		actualValue = int(character.Dexterity())
		description = fmt.Sprintf("Dexterity %s %d", c.operator, c.value)
	case IntelligenceCondition:
		actualValue = int(character.Intelligence())
		description = fmt.Sprintf("Intelligence %s %d", c.operator, c.value)
	case LuckCondition:
		actualValue = int(character.Luck())
		description = fmt.Sprintf("Luck %s %d", c.operator, c.value)
	case ExcessSPCondition:
		// Calculates excess SP beyond what's expected for the job tier
		// referenceId is the base level for the job tier (30 for 2nd job, 70 for 3rd job, 120 for 4th job)
		// Formula: excessSp = remainingSp - (level - baseLevel) * 3
		baseLevel := int(c.referenceId)
		expectedSp := (int(character.Level()) - baseLevel) * 3
		if expectedSp < 0 {
			expectedSp = 0
		}
		actualValue = int(character.RemainingSp()) - expectedSp
		description = fmt.Sprintf("Excess SP (base level %d) %s %d", baseLevel, c.operator, c.value)
	case BuddyCapacityCondition:
		// Buddy capacity requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Buddy Capacity validation requires ValidationContext"),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	case MapCapacityCondition:
		// Map capacity validation requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Map Capacity validation for map %d requires ValidationContext", c.referenceId),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	case BuffCondition:
		// Buff validation requires context - return error state
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Buff validation for source %d requires ValidationContext", c.referenceId),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	case ItemCondition:
		// For item conditions, we need to check the inventory
		itemQuantity := 0
		it, ok := inventory2.TypeFromItemId(item.Id(c.referenceId))
		if !ok {
			return ConditionResult{
				Passed:      false,
				Description: fmt.Sprintf("Invalid item ID: %d", c.referenceId),
				Type:        c.conditionType,
				Operator:    c.operator,
				Value:       c.value,
				ItemId:      c.referenceId,
				ActualValue: 0,
			}
		}

		compartment := character.Inventory().CompartmentByType(it)
		for _, a := range compartment.Assets() {
			if a.TemplateId() == c.referenceId {
				itemQuantity += int(a.Quantity())
			}
		}

		// If includeEquipped is set, also check equipped items
		if c.includeEquipped {
			for _, slot := range character.Equipment().Slots() {
				if slot.Equipable != nil && slot.Equipable.TemplateId() == c.referenceId {
					itemQuantity++
				}
				if slot.CashEquipable != nil && slot.CashEquipable.TemplateId() == c.referenceId {
					itemQuantity++
				}
			}
		}

		actualValue = itemQuantity
		itemId = c.referenceId
		description = fmt.Sprintf("Item %d quantity %s %d", c.referenceId, c.operator, c.value)
		if c.includeEquipped {
			description = fmt.Sprintf("Item %d quantity (including equipped) %s %d", c.referenceId, c.operator, c.value)
		}
	default:
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Unsupported condition type: %s", c.conditionType),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	}

	// Compare the actual value with the expected value based on the operator
	switch c.operator {
	case Equals:
		passed = actualValue == c.value
	case GreaterThan:
		passed = actualValue > c.value
	case LessThan:
		passed = actualValue < c.value
	case GreaterEqual:
		passed = actualValue >= c.value
	case LessEqual:
		passed = actualValue <= c.value
	case In:
		// Check if actualValue is in the values list
		for _, v := range c.values {
			if actualValue == v {
				passed = true
				break
			}
		}
		description = fmt.Sprintf("%s in %v", c.conditionType, c.values)
	}

	return ConditionResult{
		Passed:      passed,
		Description: description,
		Type:        c.conditionType,
		Operator:    c.operator,
		Value:       c.value,
		ItemId:      itemId,
		ActualValue: actualValue,
	}
}

// EvaluateWithContext evaluates the condition using a validation context
// This method supports additional validation types like quest status, marriage gifts, etc.
func (c Condition) EvaluateWithContext(ctx ValidationContext) ConditionResult {
	var actualValue int
	var passed bool
	var description string
	var itemId uint32

	character := ctx.Character()

	// Handle context-specific conditions first
	switch c.conditionType {
	case QuestStatusCondition:
		questModel, exists := ctx.Quest(c.referenceId)
		if !exists {
			return ConditionResult{
				Passed:      false,
				Description: fmt.Sprintf("Quest %d not found", c.referenceId),
				Type:        c.conditionType,
				Operator:    c.operator,
				Value:       c.value,
				ActualValue: int(quest.StateNotStarted),
			}
		}
		actualValue = int(questModel.State())
		description = fmt.Sprintf("Quest %d Status %s %d", c.referenceId, c.operator, c.value)

	case QuestProgressCondition:
		questModel, exists := ctx.Quest(c.referenceId)
		if !exists {
			return ConditionResult{
				Passed:      false,
				Description: fmt.Sprintf("Quest %d not found", c.referenceId),
				Type:        c.conditionType,
				Operator:    c.operator,
				Value:       c.value,
				ActualValue: 0,
			}
		}
		actualValue = questModel.GetProgressByKey(c.step)
		description = fmt.Sprintf("Quest %d Progress (step: %s) %s %d", c.referenceId, c.step, c.operator, c.value)

	case UnclaimedMarriageGiftsCondition:
		marriageModel := ctx.Marriage()
		if marriageModel.HasUnclaimedGifts() {
			actualValue = 1
		} else {
			actualValue = 0
		}
		description = fmt.Sprintf("Unclaimed Marriage Gifts %s %d", c.operator, c.value)

	case BuddyCapacityCondition:
		buddyList := ctx.BuddyList()
		actualValue = int(buddyList.Capacity())
		description = fmt.Sprintf("Buddy Capacity %s %d", c.operator, c.value)

	case PetCountCondition:
		actualValue = ctx.PetCount()
		description = fmt.Sprintf("Pet Count %s %d", c.operator, c.value)

	case MapCapacityCondition:
		// Get player count for the specified map using worldId/channelId from condition
		actualValue = ctx.GetPlayerCountInMap(c.worldId, c.channelId, _map.Id(c.referenceId))
		description = fmt.Sprintf("Map %d Player Count %s %d (world:%d channel:%d)", c.referenceId, c.operator, c.value, c.worldId, c.channelId)

	case GuildIdCondition:
		actualValue = int(character.Guild().Id())
		if actualValue == 0 {
			description = fmt.Sprintf("Guild ID %s %d (character not in guild)", c.operator, c.value)
		} else {
			description = fmt.Sprintf("Guild ID %s %d", c.operator, c.value)
		}

	case GuildRankCondition:
		actualValue = character.Guild().MemberRank(character.Id())
		if character.Guild().Id() == 0 {
			description = fmt.Sprintf("Guild Rank %s %d (character not in guild)", c.operator, c.value)
		} else {
			description = fmt.Sprintf("Guild Rank %s %d", c.operator, c.value)
		}

	case InventorySpaceCondition:
		// Check if item processor is available
		itemProcessor := ctx.ItemProcessor()
		if itemProcessor == nil {
			return ConditionResult{
				Passed:      false,
				Description: fmt.Sprintf("Item processor not available for inventory space check (item %d)", c.referenceId),
				Type:        c.conditionType,
				Operator:    c.operator,
				Value:       c.value,
				ItemId:      c.referenceId,
				ActualValue: 0,
			}
		}

		// Calculate inventory space
		canHold, slotsRemaining := CalculateInventorySpace(character, c.referenceId, uint32(c.value), itemProcessor)

		// For inventory space, actualValue represents slots remaining after adding items
		// If slotsRemaining is negative, it means we don't have enough space
		actualValue = slotsRemaining
		itemId = c.referenceId
		description = fmt.Sprintf("Inventory Space for item %d (quantity %d) %s required (slots remaining: %d)", c.referenceId, c.value, c.operator, slotsRemaining)

		// Handle the comparison based on canHold result
		// The value in the condition represents the quantity to add
		// We return canHold as the result for >= operator (can we hold this quantity?)
		if c.operator == GreaterEqual {
			passed = canHold
		} else {
			// For other operators, compare slots remaining
			// This allows more flexible conditions if needed
			switch c.operator {
			case Equals:
				passed = actualValue == c.value
			case GreaterThan:
				passed = actualValue > c.value
			case LessThan:
				passed = actualValue < c.value
			case LessEqual:
				passed = actualValue <= c.value
			}
		}

		return ConditionResult{
			Passed:      passed,
			Description: description,
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ItemId:      itemId,
			ActualValue: actualValue,
		}

	case TransportAvailableCondition:
		// Get transport state for the specified start map
		state := ctx.GetTransportState(_map.Id(c.referenceId))

		// Map state to numeric value (open_entry=1, other=0)
		if state == "open_entry" {
			actualValue = 1
		} else {
			actualValue = 0
		}

		description = fmt.Sprintf("Transport for map %d is %s (state: %s)", c.referenceId, func() string {
			if actualValue == 1 {
				return "available"
			}
			return "not available"
		}(), state)

	case SkillLevelCondition:
		// Get skill level for the specified skill ID
		skillLevel := ctx.GetSkillLevel(c.referenceId)
		actualValue = int(skillLevel)
		description = fmt.Sprintf("Skill %d Level %s %d", c.referenceId, c.operator, c.value)

	case BuffCondition:
		// Check if character has an active buff with the specified source ID
		hasBuff := ctx.HasActiveBuff(int32(c.referenceId))
		if hasBuff {
			actualValue = 1
		} else {
			actualValue = 0
		}
		description = fmt.Sprintf("Buff %d Active %s %d", c.referenceId, c.operator, c.value)

	default:
		// For non-context-specific conditions, delegate to the original Evaluate method
		return c.Evaluate(character)
	}

	// Compare the actual value with the expected value based on the operator
	switch c.operator {
	case Equals:
		passed = actualValue == c.value
	case GreaterThan:
		passed = actualValue > c.value
	case LessThan:
		passed = actualValue < c.value
	case GreaterEqual:
		passed = actualValue >= c.value
	case LessEqual:
		passed = actualValue <= c.value
	case In:
		// Check if actualValue is in the values list
		for _, v := range c.values {
			if actualValue == v {
				passed = true
				break
			}
		}
		description = fmt.Sprintf("%s in %v", c.conditionType, c.values)
	}

	return ConditionResult{
		Passed:      passed,
		Description: description,
		Type:        c.conditionType,
		Operator:    c.operator,
		Value:       c.value,
		ItemId:      itemId,
		ActualValue: actualValue,
	}
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	passed      bool
	details     []string
	results     []ConditionResult
	characterId uint32
}

// NewValidationResult creates a new validation result
func NewValidationResult(characterId uint32) ValidationResult {
	return ValidationResult{
		passed:      true,
		details:     []string{},
		results:     []ConditionResult{},
		characterId: characterId,
	}
}

// Passed returns whether the validation passed
func (v ValidationResult) Passed() bool {
	return v.passed
}

// Details returns the details of the validation
func (v ValidationResult) Details() []string {
	return v.details
}

// Results returns the structured condition results
func (v ValidationResult) Results() []ConditionResult {
	return v.results
}

// CharacterId returns the character ID that was validated
func (v ValidationResult) CharacterId() uint32 {
	return v.characterId
}

// AddConditionResult adds a structured condition result to the validation result
func (v *ValidationResult) AddConditionResult(result ConditionResult) {
	if !result.Passed {
		v.passed = false
	}
	status := "Passed"
	if !result.Passed {
		status = "Failed"
	}
	v.details = append(v.details, fmt.Sprintf("%s: %s", status, result.Description))
	v.results = append(v.results, result)
}
