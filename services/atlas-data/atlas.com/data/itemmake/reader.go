package itemmake

import (
	"atlas-data/xml"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)
		for _, groupNode := range exml.ChildNodes {
			group, err := strconv.Atoi(groupNode.Name)
			if err != nil {
				l.Warnf("Top-level itemMake group [%s] is not numeric, skipping.", groupNode.Name)
				continue
			}

			for _, entryNode := range groupNode.ChildNodes {
				id, err := strconv.Atoi(entryNode.Name)
				if err != nil {
					l.Warnf("itemMake entry [%s] in group [%d] is not numeric, skipping.", entryNode.Name, group)
					continue
				}

				m := RestModel{}
				m.Id = uint32(id)
				m.Group = uint32(group)
				l.Debugf("Processing itemMake [%d].", m.Id)
				m.ReqLevel = uint32(entryNode.GetIntegerWithDefault("reqLevel", 0))
				m.ReqSkillLevel = uint32(entryNode.GetIntegerWithDefault("reqSkillLevel", 0))
				m.ItemNum = uint32(entryNode.GetIntegerWithDefault("itemNum", 0))
				m.Tuc = uint32(entryNode.GetIntegerWithDefault("tuc", 0))
				m.Meso = uint32(entryNode.GetIntegerWithDefault("meso", 0))
				m.Catalyst = uint32(entryNode.GetIntegerWithDefault("catalyst", 0))
				m.ReqItem = uint32(entryNode.GetIntegerWithDefault("reqItem", 0))
				m.ReqEquip = uint32(entryNode.GetIntegerWithDefault("reqEquip", 0))

				m.Recipe = make([]MaterialRestModel, 0)
				if recipeNode, err := entryNode.ChildByName("recipe"); err == nil {
					for _, c := range recipeNode.ChildNodes {
						m.Recipe = append(m.Recipe, MaterialRestModel{
							ItemId: uint32(c.GetIntegerWithDefault("item", 0)),
							Count:  uint32(c.GetIntegerWithDefault("count", 0)),
						})
					}
				}

				m.RandomReward = make([]RewardRestModel, 0)
				if rewardNode, err := entryNode.ChildByName("randomReward"); err == nil {
					for _, c := range rewardNode.ChildNodes {
						m.RandomReward = append(m.RandomReward, RewardRestModel{
							ItemId:  uint32(c.GetIntegerWithDefault("item", 0)),
							ItemNum: uint32(c.GetIntegerWithDefault("itemNum", 0)),
							Prob:    uint32(c.GetIntegerWithDefault("prob", 0)),
						})
					}
				}

				m.ReqQuest = make([]QuestReqRestModel, 0)
				if reqQuestNode, err := entryNode.ChildByName("reqQuest"); err == nil {
					for _, in := range reqQuestNode.IntegerNodes {
						questId, err := strconv.Atoi(in.Name)
						if err != nil {
							l.Warnf("itemMake [%d] reqQuest key [%s] is not numeric, skipping.", m.Id, in.Name)
							continue
						}
						state, err := strconv.Atoi(in.Value)
						if err != nil {
							l.Warnf("itemMake [%d] reqQuest [%d] value [%s] is not numeric, skipping.", m.Id, questId, in.Value)
							continue
						}
						m.ReqQuest = append(m.ReqQuest, QuestReqRestModel{
							QuestId: uint32(questId),
							State:   uint32(state),
						})
					}
				}

				res = append(res, m)
			}
		}

		return model.FixedProvider(res)
	}
}
