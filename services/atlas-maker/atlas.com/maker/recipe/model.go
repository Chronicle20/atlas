package recipe

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// Material is one ordered (item, count) entry of a recipe's material list.
// Order is document order, preserved from the atlas-data item-make model;
// never sorted.
type Material struct {
	ItemId item.Id
	Count  uint32
}

// Reward is one ordered (item, itemNum, prob) entry of a recipe's optional
// random-reward list. Order is preserved from the atlas-data item-make
// model; never sorted.
type Reward struct {
	ItemId  item.Id
	ItemNum uint32
	Prob    uint32
}

// QuestRequirement is one (questId, state) entry of a recipe's optional
// quest-requirement list. Order is preserved from the atlas-data item-make
// model; never sorted.
type QuestRequirement struct {
	QuestId uint32
	State   uint32
}

// Model is one atlas-maker recipe, indexed from the atlas-data item-make
// catalog (design §4.2.1). It is immutable once built.
type Model struct {
	id                item.Id
	group             uint32
	reqLevel          uint32
	reqSkillLevel     uint32
	itemNum           uint32
	tuc               uint32
	meso              uint32
	catalyst          item.Id
	reqItem           item.Id
	reqEquip          item.Id
	materials         []Material
	randomRewards     []Reward
	questRequirements []QuestRequirement
}

// Id is the produced item's template id.
func (m Model) Id() item.Id {
	return m.id
}

// Group is the item-make archive's top-level group digit. Group 0 is the
// crystallization group that byLeftover indexes; every other group is
// ignored by that index (design C-6).
func (m Model) Group() uint32 {
	return m.group
}

func (m Model) ReqLevel() uint32 {
	return m.reqLevel
}

func (m Model) ReqSkillLevel() uint32 {
	return m.reqSkillLevel
}

func (m Model) ItemNum() uint32 {
	return m.itemNum
}

func (m Model) Tuc() uint32 {
	return m.tuc
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) Catalyst() item.Id {
	return m.catalyst
}

func (m Model) ReqItem() item.Id {
	return m.reqItem
}

func (m Model) ReqEquip() item.Id {
	return m.reqEquip
}

// Materials is the ordered list of items and counts consumed by this
// recipe. Never sorted.
func (m Model) Materials() []Material {
	return m.materials
}

// RandomRewards is the ordered list of possible outputs for this recipe.
// Never sorted.
func (m Model) RandomRewards() []Reward {
	return m.randomRewards
}

// QuestRequirements is the ordered list of quests this recipe requires.
// Never sorted.
func (m Model) QuestRequirements() []QuestRequirement {
	return m.questRequirements
}
