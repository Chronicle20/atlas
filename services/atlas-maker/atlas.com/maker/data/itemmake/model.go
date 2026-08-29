package itemmake

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// Material is one ordered (item, count) entry of a recipe's `recipe` child
// list (design FR-1.3). Order is document order and is load-bearing.
type Material struct {
	itemId item.Id
	count  uint32
}

func (m Material) ItemId() item.Id {
	return m.itemId
}

func (m Material) Count() uint32 {
	return m.count
}

// Reward is one ordered (item, itemNum, prob) entry of a recipe's optional
// `randomReward` child list (design FR-1.4). Prob is a relative weight, not a
// percentage; it is never sent to the client.
type Reward struct {
	itemId  item.Id
	itemNum uint32
	prob    uint32
}

func (r Reward) ItemId() item.Id {
	return r.itemId
}

func (r Reward) ItemNum() uint32 {
	return r.itemNum
}

func (r Reward) Prob() uint32 {
	return r.prob
}

// QuestReq is one (questId, state) entry of a recipe's optional `reqQuest`
// child list (design C-5). Only two recipes in the reference archive carry
// one.
type QuestReq struct {
	questId uint32
	state   uint32
}

func (q QuestReq) QuestId() uint32 {
	return q.questId
}

func (q QuestReq) State() uint32 {
	return q.state
}

// Model is one atlas-data item-make recipe (design §4.2's "recipes" table),
// as read by atlas-maker to evaluate and consume a craft.
type Model struct {
	id            item.Id
	group         uint32
	reqLevel      uint32
	reqSkillLevel uint32
	itemNum       uint32
	tuc           uint32
	meso          uint32
	catalyst      uint32
	reqItem       uint32
	reqEquip      uint32
	recipe        []Material
	randomReward  []Reward
	reqQuest      []QuestReq
}

// Id is the produced item's template id (the item-make resource's own id).
func (m Model) Id() item.Id {
	return m.id
}

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

func (m Model) Catalyst() uint32 {
	return m.catalyst
}

func (m Model) ReqItem() uint32 {
	return m.reqItem
}

func (m Model) ReqEquip() uint32 {
	return m.reqEquip
}

func (m Model) Recipe() []Material {
	return m.recipe
}

func (m Model) RandomReward() []Reward {
	return m.randomReward
}

func (m Model) ReqQuest() []QuestReq {
	return m.reqQuest
}
