package itemmake

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// Builder constructs a Model.
type Builder struct {
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

func NewBuilder(id item.Id) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetGroup(group uint32) *Builder {
	b.group = group
	return b
}

func (b *Builder) SetReqLevel(reqLevel uint32) *Builder {
	b.reqLevel = reqLevel
	return b
}

func (b *Builder) SetReqSkillLevel(reqSkillLevel uint32) *Builder {
	b.reqSkillLevel = reqSkillLevel
	return b
}

func (b *Builder) SetItemNum(itemNum uint32) *Builder {
	b.itemNum = itemNum
	return b
}

func (b *Builder) SetTuc(tuc uint32) *Builder {
	b.tuc = tuc
	return b
}

func (b *Builder) SetMeso(meso uint32) *Builder {
	b.meso = meso
	return b
}

func (b *Builder) SetCatalyst(catalyst uint32) *Builder {
	b.catalyst = catalyst
	return b
}

func (b *Builder) SetReqItem(reqItem uint32) *Builder {
	b.reqItem = reqItem
	return b
}

func (b *Builder) SetReqEquip(reqEquip uint32) *Builder {
	b.reqEquip = reqEquip
	return b
}

func (b *Builder) SetRecipe(recipe []Material) *Builder {
	b.recipe = recipe
	return b
}

func (b *Builder) SetRandomReward(randomReward []Reward) *Builder {
	b.randomReward = randomReward
	return b
}

func (b *Builder) SetReqQuest(reqQuest []QuestReq) *Builder {
	b.reqQuest = reqQuest
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		id:            b.id,
		group:         b.group,
		reqLevel:      b.reqLevel,
		reqSkillLevel: b.reqSkillLevel,
		itemNum:       b.itemNum,
		tuc:           b.tuc,
		meso:          b.meso,
		catalyst:      b.catalyst,
		reqItem:       b.reqItem,
		reqEquip:      b.reqEquip,
		recipe:        b.recipe,
		randomReward:  b.randomReward,
		reqQuest:      b.reqQuest,
	}, nil
}
