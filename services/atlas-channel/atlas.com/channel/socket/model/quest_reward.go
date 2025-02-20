package model

type QuestReward struct {
	itemId uint32
	amount int32
}

func (r QuestReward) ItemId() uint32 {
	return r.itemId
}

func (r QuestReward) Amount() int32 {
	return r.amount
}

func NewQuestReward(itemId uint32, amount int32) QuestReward {
	return QuestReward{itemId: itemId, amount: amount}
}
