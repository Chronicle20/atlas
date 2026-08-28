package itemmake

import "strconv"

type RestModel struct {
	Id            uint32              `json:"-"`
	Group         uint32              `json:"group"`
	ReqLevel      uint32              `json:"reqLevel"`
	ReqSkillLevel uint32              `json:"reqSkillLevel"`
	ItemNum       uint32              `json:"itemNum"`
	Tuc           uint32              `json:"tuc"`
	Meso          uint32              `json:"meso"`
	Catalyst      uint32              `json:"catalyst"`
	ReqItem       uint32              `json:"reqItem"`
	ReqEquip      uint32              `json:"reqEquip"`
	Recipe        []MaterialRestModel `json:"recipe"`
	RandomReward  []RewardRestModel   `json:"randomReward"`
	ReqQuest      []QuestReqRestModel `json:"reqQuest"`
}

// MaterialRestModel is one ordered (item, count) entry of a recipe's `recipe`
// child list (FR-1.3). Order is document order and is load-bearing.
type MaterialRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
}

// RewardRestModel is one ordered (item, itemNum, prob) entry of a recipe's
// optional `randomReward` child list (FR-1.4). Prob is a relative weight, not a
// percentage; it is never sent to the client.
type RewardRestModel struct {
	ItemId  uint32 `json:"itemId"`
	ItemNum uint32 `json:"itemNum"`
	Prob    uint32 `json:"prob"`
}

// QuestReqRestModel is one (questId, state) entry of a recipe's optional
// `reqQuest` child list (design C-5). Only two recipes in the reference archive
// carry one.
type QuestReqRestModel struct {
	QuestId uint32 `json:"questId"`
	State   uint32 `json:"state"`
}

func (r RestModel) GetName() string {
	return "itemMakes"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
