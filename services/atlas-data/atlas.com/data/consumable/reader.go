package consumable

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
)

func parseConsumableId(name string) (uint32, error) {
	id, err := strconv.Atoi(name)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)
		for _, cxml := range exml.ChildNodes {
			consumableId, err := parseConsumableId(cxml.Name)
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}
			l.Debugf("Processing consumable [%d].", consumableId)

			i, err := cxml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			m := RestModel{
				Id:             consumableId,
				Spec:           make(map[SpecType]int32),
				MonsterSummons: make(map[uint32]uint32),
				Skills:         make([]uint32, 0),
				Morphs:         make(map[uint32]uint32),
				Rewards:        make([]RewardRestModel, 0),
			}
			m.TradeBlock = i.GetBool("tradeBlock", false)
			m.Price = uint32(i.GetIntegerWithDefault("price", 0))
			m.UnitPrice = i.GetDouble("unitPrice", 0)
			m.SlotMax = uint32(i.GetIntegerWithDefault("slotMax", 0))
			m.TimeLimited = i.GetBool("timeLimited", false)
			m.NotSale = i.GetBool("notSale", false)
			m.Quest = i.GetBool("quest", false)
			m.Only = i.GetBool("only", false)
			m.Success = uint32(i.GetIntegerWithDefault("success", 0))
			m.Cursed = uint32(i.GetIntegerWithDefault("cursed", 0))
			m.Create = uint32(i.GetIntegerWithDefault("create", 0))
			m.MasterLevel = uint32(i.GetIntegerWithDefault("masterLevel", 0))
			m.ReqSkillLevel = uint32(i.GetIntegerWithDefault("reqSkillLevel", 0))
			m.TradeAvailable = i.GetBool("tradeAvailable", false)
			m.NoCancelMouse = i.GetBool("noCancelMouse", false)
			m.Pquest = i.GetBool("pquest", false)
			m.Left = i.GetIntegerWithDefault("left", 0)
			m.Right = i.GetIntegerWithDefault("right", 0)
			m.Top = i.GetIntegerWithDefault("top", 0)
			m.Bottom = i.GetIntegerWithDefault("bottom", 0)
			m.BridleMsgType = uint32(i.GetIntegerWithDefault("bridleMsgType", 0))
			m.BridleProp = uint32(i.GetIntegerWithDefault("bridleProp", 0))
			m.BridlePropChg = i.GetFloatWithDefault("bridlePropChg", 0)
			m.UseDelay = uint32(i.GetIntegerWithDefault("useDelay", 0))
			m.DelayMsg = i.GetString("delayMsg", "")
			m.IncFatigue = i.GetIntegerWithDefault("incFatigue", 0)
			m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))
			m.RunOnPickup = i.GetBool("runOnPickup", false)
			m.MonsterBook = i.GetBool("monsterBook", false)
			m.MonsterId = uint32(i.GetIntegerWithDefault("mob", 0))
			m.BigSize = i.GetBool("bigSize", false)
			m.TargetBlock = i.GetBool("tragetBlock", false)
			m.Effect = i.GetString("effect", "")
			m.MonsterHP = uint32(i.GetIntegerWithDefault("mobHP", 0))
			m.WorldMsg = i.GetString("worldMsg", "")
			m.IncreasePDD = uint32(i.GetIntegerWithDefault("incPDD", 0))
			m.IncreaseMDD = uint32(i.GetIntegerWithDefault("incMDD", 0))
			m.IncreaseACC = uint32(i.GetIntegerWithDefault("incACC", 0))
			m.IncreaseMHP = uint32(i.GetIntegerWithDefault("incMHP", 0))
			m.IncreaseMMP = uint32(i.GetIntegerWithDefault("incMMP", 0))
			m.IncreasePAD = uint32(i.GetIntegerWithDefault("incPAD", 0))
			m.IncreaseMAD = uint32(i.GetIntegerWithDefault("incMAD", 0))
			m.IncreaseJump = uint32(i.GetIntegerWithDefault("incJump", 0))
			m.IncreaseEVA = uint32(i.GetIntegerWithDefault("incEVA", 0))
			m.IncreaseLUK = uint32(i.GetIntegerWithDefault("incLUK", 0))
			m.IncreaseDEX = uint32(i.GetIntegerWithDefault("incDEX", 0))
			m.IncreaseINT = uint32(i.GetIntegerWithDefault("incINT", 0))
			m.IncreaseSTR = uint32(i.GetIntegerWithDefault("incSTR", 0))
			m.IncreaseSpeed = uint32(i.GetIntegerWithDefault("incSpeed", 0))

			mos, err := cxml.ChildByName("mob")
			if err == nil && mos != nil {
				for _, mo := range mos.ChildNodes {
					mid := uint32(mo.GetIntegerWithDefault("id", 0))
					prob := uint32(mo.GetIntegerWithDefault("prob", 0))
					m.MonsterSummons[mid] = prob
				}
			}

			ss, err := i.ChildByName("skill")
			if err == nil && ss != nil {
				for _, s := range ss.IntegerNodes {
					val, err := strconv.ParseUint(s.Value, 10, 32)
					if err != nil {
						return model.ErrorProvider[[]RestModel](err)
					}
					m.Skills = append(m.Skills, uint32(val))
				}
			}

			s, err := cxml.ChildByName("spec")
			if err == nil && s != nil {
				m.Spec[SpecTypeHP] = s.GetIntegerWithDefault(string(SpecTypeHP), 0)
				m.Spec[SpecTypeMP] = s.GetIntegerWithDefault(string(SpecTypeMP), 0)
				m.Spec[SpecTypeHPRecovery] = s.GetIntegerWithDefault(string(SpecTypeHPRecovery), 0)
				m.Spec[SpecTypeMPRecovery] = s.GetIntegerWithDefault(string(SpecTypeMPRecovery), 0)
				m.Spec[SpecTypeMoveTo] = s.GetIntegerWithDefault(string(SpecTypeMoveTo), 0)
				m.Spec[SpecTypeWeaponAttack] = s.GetIntegerWithDefault(string(SpecTypeWeaponAttack), 0)
				m.Spec[SpecTypeMagicAttack] = s.GetIntegerWithDefault(string(SpecTypeMagicAttack), 0)
				m.Spec[SpecTypeWeaponDefense] = s.GetIntegerWithDefault(string(SpecTypeWeaponDefense), 0)
				m.Spec[SpecTypeMagicDefense] = s.GetIntegerWithDefault(string(SpecTypeMagicDefense), 0)
				m.Spec[SpecTypeSpeed] = s.GetIntegerWithDefault(string(SpecTypeSpeed), 0)
				m.Spec[SpecTypeEvasion] = s.GetIntegerWithDefault(string(SpecTypeEvasion), 0)
				m.Spec[SpecTypeAccuracy] = s.GetIntegerWithDefault(string(SpecTypeAccuracy), 0)
				m.Spec[SpecTypeJump] = s.GetIntegerWithDefault(string(SpecTypeJump), 0)
				m.Spec[SpecTypeTime] = s.GetIntegerWithDefault(string(SpecTypeTime), 0)
				m.Spec[SpecTypeThaw] = s.GetIntegerWithDefault(string(SpecTypeThaw), 0)
				m.Spec[SpecTypePoison] = s.GetIntegerWithDefault(string(SpecTypePoison), 0)
				m.Spec[SpecTypeDarkness] = s.GetIntegerWithDefault(string(SpecTypeDarkness), 0)
				m.Spec[SpecTypeWeakness] = s.GetIntegerWithDefault(string(SpecTypeWeakness), 0)
				m.Spec[SpecTypeSeal] = s.GetIntegerWithDefault(string(SpecTypeSeal), 0)
				m.Spec[SpecTypeCurse] = s.GetIntegerWithDefault(string(SpecTypeCurse), 0)
				m.Spec[SpecTypeReturnMap] = s.GetIntegerWithDefault(string(SpecTypeReturnMap), 0)
				m.Spec[SpecTypeIgnoreContinent] = s.GetIntegerWithDefault(string(SpecTypeIgnoreContinent), 0)
				m.Spec[SpecTypeMorph] = s.GetIntegerWithDefault(string(SpecTypeMorph), 0)
				m.Spec[SpecTypeRandomMoveInFieldSet] = s.GetIntegerWithDefault(string(SpecTypeRandomMoveInFieldSet), 0)
				m.Spec[SpecTypeExperienceBuff] = s.GetIntegerWithDefault(string(SpecTypeExperienceBuff), 0)
				m.Spec[SpecTypeInc] = s.GetIntegerWithDefault(string(SpecTypeInc), 0)
				m.Spec[SpecTypeOnlyPickup] = s.GetIntegerWithDefault(string(SpecTypeOnlyPickup), 0)

				ms, err := s.ChildByName("morphRandom")
				if err == nil && ms != nil {
					for _, mo := range ms.ChildNodes {
						id := uint32(mo.GetIntegerWithDefault("morph", 0))
						prob := uint32(mo.GetIntegerWithDefault("prob", 0))
						m.Morphs[id] = prob
					}
				}
				m.Script = s.GetString("script", "")
			}

			r, err := cxml.ChildByName("reward")
			if err == nil && r != nil {
				for _, ro := range r.ChildNodes {
					itemId := uint32(ro.GetIntegerWithDefault("item", 0))
					count := uint32(ro.GetIntegerWithDefault("count", 0))
					prob := uint32(ro.GetIntegerWithDefault("prob", 0))
					m.Rewards = append(m.Rewards, RewardRestModel{itemId, count, prob})
				}
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
