package itemmake

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// RestModel mirrors atlas-data's item-make resource
// (services/atlas-data/atlas.com/data/itemmake/rest.go) field-for-field;
// atlas-maker consumes every one of these to evaluate and consume a craft.
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

type MaterialRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
}

type RewardRestModel struct {
	ItemId  uint32 `json:"itemId"`
	ItemNum uint32 `json:"itemNum"`
	Prob    uint32 `json:"prob"`
}

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

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's
// unmarshal even though this client doesn't care about the item-make
// resource's relationships (libs/atlas-rest gotcha): a target struct must
// implement them or unmarshal errors whenever the upstream response
// includes a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	recipe, err := model.SliceMap(extractMaterial)(model.FixedProvider(rm.Recipe))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	rewards, err := model.SliceMap(extractReward)(model.FixedProvider(rm.RandomReward))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	reqQuest, err := model.SliceMap(extractQuestReq)(model.FixedProvider(rm.ReqQuest))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return NewBuilder(item.Id(rm.Id)).
		SetGroup(rm.Group).
		SetReqLevel(rm.ReqLevel).
		SetReqSkillLevel(rm.ReqSkillLevel).
		SetItemNum(rm.ItemNum).
		SetTuc(rm.Tuc).
		SetMeso(rm.Meso).
		SetCatalyst(rm.Catalyst).
		SetReqItem(rm.ReqItem).
		SetReqEquip(rm.ReqEquip).
		SetRecipe(recipe).
		SetRandomReward(rewards).
		SetReqQuest(reqQuest).
		Build()
}

func extractMaterial(rm MaterialRestModel) (Material, error) {
	return Material{itemId: item.Id(rm.ItemId), count: rm.Count}, nil
}

func extractReward(rm RewardRestModel) (Reward, error) {
	return Reward{itemId: item.Id(rm.ItemId), itemNum: rm.ItemNum, prob: rm.Prob}, nil
}

func extractQuestReq(rm QuestReqRestModel) (QuestReq, error) {
	return QuestReq{questId: rm.QuestId, state: rm.State}, nil
}
