package consumable

import (
	"atlas-data/xml"
	"context"
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

func ReadFromFile(l logrus.FieldLogger) func(ctx context.Context) func(path string) model.Provider[[]Model] {
	return func(ctx context.Context) func(path string) model.Provider[[]Model] {
		return func(path string) model.Provider[[]Model] {
			exml, err := xml.Read(path)
			if err != nil {
				return model.ErrorProvider[[]Model](err)
			}

			res := make([]Model, 0)
			for _, cxml := range exml.ChildNodes {
				consumableId, err := parseConsumableId(cxml.Name)
				if err != nil {
					return model.ErrorProvider[[]Model](err)
				}
				l.Debugf("Processing consumable [%d].", consumableId)

				i, err := cxml.ChildByName("info")
				if err != nil {
					return model.ErrorProvider[[]Model](err)
				}

				m := Model{
					id:             consumableId,
					spec:           make(map[SpecType]int32),
					monsterSummons: make(map[uint32]uint32),
					skills:         make([]uint32, 0),
					morphs:         make(map[uint32]uint32),
					rewards:        make([]RewardModel, 0),
				}
				m.tradeBlock = i.GetBool("tradeBlock", false)
				m.price = uint32(i.GetIntegerWithDefault("price", 0))
				m.unitPrice = uint32(i.GetIntegerWithDefault("unitPrice", 0))
				m.slotMax = uint32(i.GetIntegerWithDefault("slotMax", 0))
				m.timeLimited = i.GetBool("timeLimited", false)
				m.notSale = i.GetBool("notSale", false)
				m.quest = i.GetBool("quest", false)
				m.only = i.GetBool("only", false)
				m.success = uint32(i.GetIntegerWithDefault("success", 0))
				m.cursed = uint32(i.GetIntegerWithDefault("cursed", 0))
				m.create = uint32(i.GetIntegerWithDefault("create", 0))
				m.masterLevel = uint32(i.GetIntegerWithDefault("masterLevel", 0))
				m.reqSkillLevel = uint32(i.GetIntegerWithDefault("reqSkillLevel", 0))
				m.tradeAvailable = i.GetBool("tradeAvailable", false)
				m.noCancelMouse = i.GetBool("noCancelMouse", false)
				m.pquest = i.GetBool("pquest", false)
				m.left = i.GetIntegerWithDefault("left", 0)
				m.right = i.GetIntegerWithDefault("right", 0)
				m.top = i.GetIntegerWithDefault("top", 0)
				m.bottom = i.GetIntegerWithDefault("bottom", 0)
				m.bridleMsgType = uint32(i.GetIntegerWithDefault("bridleMsgType", 0))
				m.bridleProp = uint32(i.GetIntegerWithDefault("bridleProp", 0))
				m.bridlePropChg = i.GetFloatWithDefault("bridlePropChg", 0)
				m.useDelay = uint32(i.GetIntegerWithDefault("useDelay", 0))
				m.delayMsg = i.GetString("delayMsg", "")
				m.incFatigue = i.GetIntegerWithDefault("incFatigue", 0)
				m.npc = uint32(i.GetIntegerWithDefault("npc", 0))
				m.runOnPickup = i.GetBool("runOnPickup", false)
				m.monsterBook = i.GetBool("monsterBook", false)
				m.monsterId = uint32(i.GetIntegerWithDefault("mob", 0))
				m.bigSize = i.GetBool("bigSize", false)
				m.tragetBlock = i.GetBool("tragetBlock", false)
				m.effect = i.GetString("effect", "")
				m.monsterHp = uint32(i.GetIntegerWithDefault("mobHP", 0))
				m.worldMsg = i.GetString("worldMsg", "")
				m.incPDD = uint32(i.GetIntegerWithDefault("incPDD", 0))
				m.incMDD = uint32(i.GetIntegerWithDefault("incMDD", 0))
				m.incACC = uint32(i.GetIntegerWithDefault("incACC", 0))
				m.incMHP = uint32(i.GetIntegerWithDefault("incMHP", 0))
				m.incMMP = uint32(i.GetIntegerWithDefault("incMMP", 0))
				m.incPAD = uint32(i.GetIntegerWithDefault("incPAD", 0))
				m.incMAD = uint32(i.GetIntegerWithDefault("incMAD", 0))
				m.incEVA = uint32(i.GetIntegerWithDefault("incEVA", 0))
				m.incLUK = uint32(i.GetIntegerWithDefault("incLUK", 0))
				m.incDEX = uint32(i.GetIntegerWithDefault("incDEX", 0))
				m.incINT = uint32(i.GetIntegerWithDefault("incINT", 0))
				m.incSTR = uint32(i.GetIntegerWithDefault("incSTR", 0))
				m.incSpeed = uint32(i.GetIntegerWithDefault("incSpeed", 0))

				mos, err := cxml.ChildByName("mob")
				if err == nil && mos != nil {
					for _, mo := range mos.ChildNodes {
						mid := uint32(mo.GetIntegerWithDefault("id", 0))
						prob := uint32(mo.GetIntegerWithDefault("prob", 0))
						m.monsterSummons[mid] = prob
					}
				}

				ss, err := i.ChildByName("skill")
				if err == nil && ss != nil {
					for _, s := range ss.IntegerNodes {
						val, err := strconv.ParseUint(s.Value, 10, 32)
						if err != nil {
							return model.ErrorProvider[[]Model](err)
						}
						m.skills = append(m.skills, uint32(val))
					}
				}

				s, err := cxml.ChildByName("spec")
				if err == nil && s != nil {
					m.spec[SpecTypeHP] = s.GetIntegerWithDefault(string(SpecTypeHP), 0)
					m.spec[SpecTypeMP] = s.GetIntegerWithDefault(string(SpecTypeMP), 0)
					m.spec[SpecTypeHPRecovery] = s.GetIntegerWithDefault(string(SpecTypeHPRecovery), 0)
					m.spec[SpecTypeMPRecovery] = s.GetIntegerWithDefault(string(SpecTypeMPRecovery), 0)
					m.spec[SpecTypeMoveTo] = s.GetIntegerWithDefault(string(SpecTypeMoveTo), 0)
					m.spec[SpecTypeWeaponAttack] = s.GetIntegerWithDefault(string(SpecTypeWeaponAttack), 0)
					m.spec[SpecTypeMagicAttack] = s.GetIntegerWithDefault(string(SpecTypeMagicAttack), 0)
					m.spec[SpecTypeWeaponDefense] = s.GetIntegerWithDefault(string(SpecTypeWeaponDefense), 0)
					m.spec[SpecTypeMagicDefense] = s.GetIntegerWithDefault(string(SpecTypeMagicDefense), 0)
					m.spec[SpecTypeSpeed] = s.GetIntegerWithDefault(string(SpecTypeSpeed), 0)
					m.spec[SpecTypeEvasion] = s.GetIntegerWithDefault(string(SpecTypeEvasion), 0)
					m.spec[SpecTypeAccuracy] = s.GetIntegerWithDefault(string(SpecTypeAccuracy), 0)
					m.spec[SpecTypeJump] = s.GetIntegerWithDefault(string(SpecTypeJump), 0)
					m.spec[SpecTypeTime] = s.GetIntegerWithDefault(string(SpecTypeTime), 0)
					m.spec[SpecTypeThaw] = s.GetIntegerWithDefault(string(SpecTypeThaw), 0)
					m.spec[SpecTypePoison] = s.GetIntegerWithDefault(string(SpecTypePoison), 0)
					m.spec[SpecTypeDarkness] = s.GetIntegerWithDefault(string(SpecTypeDarkness), 0)
					m.spec[SpecTypeWeakness] = s.GetIntegerWithDefault(string(SpecTypeWeakness), 0)
					m.spec[SpecTypeSeal] = s.GetIntegerWithDefault(string(SpecTypeSeal), 0)
					m.spec[SpecTypeCurse] = s.GetIntegerWithDefault(string(SpecTypeCurse), 0)
					m.spec[SpecTypeReturnMap] = s.GetIntegerWithDefault(string(SpecTypeReturnMap), 0)
					m.spec[SpecTypeIgnoreContinent] = s.GetIntegerWithDefault(string(SpecTypeIgnoreContinent), 0)
					m.spec[SpecTypeMorph] = s.GetIntegerWithDefault(string(SpecTypeMorph), 0)
					m.spec[SpecTypeRandomMoveInFieldSet] = s.GetIntegerWithDefault(string(SpecTypeRandomMoveInFieldSet), 0)
					m.spec[SpecTypeExperienceBuff] = s.GetIntegerWithDefault(string(SpecTypeExperienceBuff), 0)
					m.spec[SpecTypeInc] = s.GetIntegerWithDefault(string(SpecTypeInc), 0)
					m.spec[SpecTypeOnlyPickup] = s.GetIntegerWithDefault(string(SpecTypeOnlyPickup), 0)

					ms, err := s.ChildByName("morphRandom")
					if err == nil && ms != nil {
						for _, mo := range ms.ChildNodes {
							id := uint32(mo.GetIntegerWithDefault("morph", 0))
							prob := uint32(mo.GetIntegerWithDefault("prob", 0))
							m.morphs[id] = prob
						}
					}
					m.script = s.GetString("script", "")
				}

				r, err := cxml.ChildByName("reward")
				if err == nil && r != nil {
					for _, ro := range r.ChildNodes {
						itemId := uint32(ro.GetIntegerWithDefault("item", 0))
						count := uint32(ro.GetIntegerWithDefault("count", 0))
						prob := uint32(ro.GetIntegerWithDefault("prob", 0))
						m.rewards = append(m.rewards, RewardModel{itemId, count, prob})
					}
				}

				res = append(res, m)
			}

			return model.FixedProvider(res)
		}
	}
}
