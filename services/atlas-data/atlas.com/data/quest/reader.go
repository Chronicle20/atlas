package quest

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
)

// ReadQuestInfo reads quest info from QuestInfo.img.xml
func ReadQuestInfo(l logrus.FieldLogger) func(np model.Provider[xml.Node]) map[uint32]RestModel {
	return func(np model.Provider[xml.Node]) map[uint32]RestModel {
		result := make(map[uint32]RestModel)

		n, err := np()
		if err != nil {
			l.WithError(err).Errorf("Failed to read QuestInfo.img.xml")
			return result
		}

		for _, questNode := range n.ChildNodes {
			questId, err := strconv.Atoi(questNode.Name)
			if err != nil {
				continue
			}

			m := RestModel{
				Id:              uint32(questId),
				Name:            questNode.GetString("name", ""),
				Parent:          questNode.GetString("parent", ""),
				Area:            uint32(questNode.GetIntegerWithDefault("area", 0)),
				Order:           uint32(questNode.GetIntegerWithDefault("order", 0)),
				AutoStart:       questNode.GetBool("autoStart", false),
				AutoPreComplete: questNode.GetBool("autoPreComplete", false),
				AutoComplete:    questNode.GetBool("autoComplete", false),
				TimeLimit:       uint32(questNode.GetIntegerWithDefault("timeLimit", 0)),
				TimeLimit2:      uint32(questNode.GetIntegerWithDefault("timeLimit2", 0)),
				SelectedMob:     questNode.GetBool("selectedMob", false),
				Summary:         questNode.GetString("summary", ""),
				DemandSummary:   questNode.GetString("demandSummary", ""),
				RewardSummary:   questNode.GetString("rewardSummary", ""),
			}

			result[uint32(questId)] = m
		}

		return result
	}
}

// ReadQuestCheck reads quest requirements from Check.img.xml
func ReadQuestCheck(l logrus.FieldLogger) func(np model.Provider[xml.Node]) func(quests map[uint32]RestModel) map[uint32]RestModel {
	return func(np model.Provider[xml.Node]) func(quests map[uint32]RestModel) map[uint32]RestModel {
		return func(quests map[uint32]RestModel) map[uint32]RestModel {
			n, err := np()
			if err != nil {
				l.WithError(err).Errorf("Failed to read Check.img.xml")
				return quests
			}

			for _, questNode := range n.ChildNodes {
				questId, err := strconv.Atoi(questNode.Name)
				if err != nil {
					continue
				}

				quest, exists := quests[uint32(questId)]
				if !exists {
					quest = RestModel{Id: uint32(questId)}
				}

				// Read start requirements (phase 0)
				if startNode, err := questNode.ChildByName("0"); err == nil {
					quest.StartRequirements = readRequirements(startNode)
				}

				// Read end/completion requirements (phase 1)
				if endNode, err := questNode.ChildByName("1"); err == nil {
					quest.EndRequirements = readRequirements(endNode)
				}

				quests[uint32(questId)] = quest
			}

			return quests
		}
	}
}

// ReadQuestAct reads quest actions from Act.img.xml
func ReadQuestAct(l logrus.FieldLogger) func(np model.Provider[xml.Node]) func(quests map[uint32]RestModel) map[uint32]RestModel {
	return func(np model.Provider[xml.Node]) func(quests map[uint32]RestModel) map[uint32]RestModel {
		return func(quests map[uint32]RestModel) map[uint32]RestModel {
			n, err := np()
			if err != nil {
				l.WithError(err).Errorf("Failed to read Act.img.xml")
				return quests
			}

			for _, questNode := range n.ChildNodes {
				questId, err := strconv.Atoi(questNode.Name)
				if err != nil {
					continue
				}

				quest, exists := quests[uint32(questId)]
				if !exists {
					quest = RestModel{Id: uint32(questId)}
				}

				// Read start actions (phase 0)
				if startNode, err := questNode.ChildByName("0"); err == nil {
					quest.StartActions = readActions(startNode)
				}

				// Read end/completion actions (phase 1)
				if endNode, err := questNode.ChildByName("1"); err == nil {
					quest.EndActions = readActions(endNode)
				}

				quests[uint32(questId)] = quest
			}

			return quests
		}
	}
}

func readRequirements(n *xml.Node) RequirementsRestModel {
	req := RequirementsRestModel{
		NpcId:           uint32(n.GetIntegerWithDefault("npc", 0)),
		LevelMin:        n.GetShort("lvmin", 0),
		LevelMax:        n.GetShort("lvmax", 0),
		FameMin:         int16(n.GetIntegerWithDefault("pop", 0)),
		MesoMin:         uint32(n.GetIntegerWithDefault("money", 0)),
		MesoMax:         uint32(n.GetIntegerWithDefault("moneyMax", 0)),
		PetTamenessMin:  int16(n.GetIntegerWithDefault("pettamenessmin", 0)),
		DayOfWeek:       n.GetString("dayOfWeek", ""),
		Start:           n.GetString("start", ""),
		End:             n.GetString("end", ""),
		Interval:        uint32(n.GetIntegerWithDefault("interval", 0)),
		StartScript:     n.GetString("startscript", ""),
		EndScript:       n.GetString("endscript", ""),
		InfoNumber:      uint32(n.GetIntegerWithDefault("infoNumber", 0)),
		NormalAutoStart: n.GetBool("normalAutoStart", false),
		CompletionCount: uint32(n.GetIntegerWithDefault("completeCount", 0)),
	}

	// Read job requirements
	if jobNode, err := n.ChildByName("job"); err == nil {
		for _, jn := range jobNode.IntegerNodes {
			jobId, err := strconv.ParseUint(jn.Value, 10, 16)
			if err == nil {
				req.Jobs = append(req.Jobs, uint16(jobId))
			}
		}
	}

	// Read quest prerequisites
	if questNode, err := n.ChildByName("quest"); err == nil {
		for _, qn := range questNode.ChildNodes {
			qReq := QuestRequirement{
				Id:    uint32(qn.GetIntegerWithDefault("id", 0)),
				State: uint8(qn.GetIntegerWithDefault("state", 0)),
			}
			if qReq.Id > 0 {
				req.Quests = append(req.Quests, qReq)
			}
		}
	}

	// Read item requirements
	if itemNode, err := n.ChildByName("item"); err == nil {
		for _, in := range itemNode.ChildNodes {
			iReq := ItemRequirement{
				Id:    uint32(in.GetIntegerWithDefault("id", 0)),
				Count: int32(in.GetIntegerWithDefault("count", 0)),
			}
			if iReq.Id > 0 {
				req.Items = append(req.Items, iReq)
			}
		}
	}

	// Read mob requirements
	if mobNode, err := n.ChildByName("mob"); err == nil {
		for _, mn := range mobNode.ChildNodes {
			mReq := MobRequirement{
				Id:    uint32(mn.GetIntegerWithDefault("id", 0)),
				Count: uint32(mn.GetIntegerWithDefault("count", 0)),
			}
			if mReq.Id > 0 {
				req.Mobs = append(req.Mobs, mReq)
			}
		}
	}

	// Read field enter requirements
	if fieldNode, err := n.ChildByName("fieldEnter"); err == nil {
		for _, fn := range fieldNode.IntegerNodes {
			mapId, err := strconv.ParseUint(fn.Value, 10, 32)
			if err == nil && mapId > 0 {
				req.FieldEnter = append(req.FieldEnter, uint32(mapId))
			}
		}
	}

	// Read pet requirements
	if petNode, err := n.ChildByName("pet"); err == nil {
		for _, pn := range petNode.ChildNodes {
			petId := uint32(pn.GetIntegerWithDefault("id", 0))
			if petId > 0 {
				req.Pet = append(req.Pet, petId)
			}
		}
	}

	return req
}

func readActions(n *xml.Node) ActionsRestModel {
	act := ActionsRestModel{
		NpcId:      uint32(n.GetIntegerWithDefault("npc", 0)),
		Exp:        int32(n.GetIntegerWithDefault("exp", 0)),
		Money:      int32(n.GetIntegerWithDefault("money", 0)),
		Fame:       int16(n.GetIntegerWithDefault("pop", 0)),
		NextQuest:  uint32(n.GetIntegerWithDefault("nextQuest", 0)),
		BuffItemId: uint32(n.GetIntegerWithDefault("buffItemID", 0)),
		Interval:   uint32(n.GetIntegerWithDefault("interval", 0)),
		LevelMin:   n.GetShort("lvmin", 0),
	}

	// Read item rewards
	if itemNode, err := n.ChildByName("item"); err == nil {
		for _, in := range itemNode.ChildNodes {
			iReward := ItemReward{
				Id:         uint32(in.GetIntegerWithDefault("id", 0)),
				Count:      int32(in.GetIntegerWithDefault("count", 0)),
				Job:        int32(in.GetIntegerWithDefault("job", 0)),
				Gender:     int8(in.GetIntegerWithDefault("gender", -1)),
				Prop:       int32(in.GetIntegerWithDefault("prop", -1)),
				Period:     uint32(in.GetIntegerWithDefault("period", 0)),
				DateExpire: in.GetString("dateExpire", ""),
				Var:        uint32(in.GetIntegerWithDefault("var", 0)),
			}
			if iReward.Id > 0 {
				act.Items = append(act.Items, iReward)
			}
		}
	}

	// Read skill rewards
	if skillNode, err := n.ChildByName("skill"); err == nil {
		for _, sn := range skillNode.ChildNodes {
			sReward := SkillReward{
				Id:          uint32(sn.GetIntegerWithDefault("id", 0)),
				Level:       int32(sn.GetIntegerWithDefault("skillLevel", 0)),
				MasterLevel: uint32(sn.GetIntegerWithDefault("masterLevel", 0)),
			}

			// Read job requirements for skill
			if jobNode, err := sn.ChildByName("job"); err == nil {
				for _, jn := range jobNode.IntegerNodes {
					jobId, err := strconv.ParseUint(jn.Value, 10, 16)
					if err == nil {
						sReward.Jobs = append(sReward.Jobs, uint16(jobId))
					}
				}
			}

			if sReward.Id > 0 {
				act.Skills = append(act.Skills, sReward)
			}
		}
	}

	return act
}
