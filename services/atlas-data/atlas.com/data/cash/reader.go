package cash

import (
	"atlas-data/xml"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/pet/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func parseCashId(name string) (uint32, error) {
	id, err := strconv.Atoi(name)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// parseTimeWindow parses a time window string like "MON:18-20" into a TimeWindow struct
func parseTimeWindow(value string) (TimeWindow, bool) {
	// Format: "DAY:START-END" (e.g., "MON:18-20" or "MON:00-24")
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return TimeWindow{}, false
	}

	day := parts[0]
	hourParts := strings.Split(parts[1], "-")
	if len(hourParts) != 2 {
		return TimeWindow{}, false
	}

	startHour, err := strconv.Atoi(hourParts[0])
	if err != nil {
		return TimeWindow{}, false
	}

	endHour, err := strconv.Atoi(hourParts[1])
	if err != nil {
		return TimeWindow{}, false
	}

	return TimeWindow{
		Day:       day,
		StartHour: startHour,
		EndHour:   endHour,
	}, true
}

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)
		for _, cxml := range exml.ChildNodes {
			cashId, err := parseCashId(cxml.Name)
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}
			l.Debugf("Processing cash [%d].", cashId)

			i, err := cxml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			m := RestModel{
				Id:   cashId,
				Spec: make(map[SpecType]int32),
			}
			m.SlotMax = uint32(i.GetIntegerWithDefault("slotMax", 0))
			m.ProtectTime = uint32(i.GetIntegerWithDefault("protectTime", 0))
			m.AddTime = uint32(i.GetIntegerWithDefault("addTime", 0))
			m.MaxDays = uint32(i.GetIntegerWithDefault("maxDays", 0))
			m.Life = uint32(i.GetIntegerWithDefault("life", 0))
			// 0520 meso sacks: the flat award amount. Absent node => 0, which the
			// channel handler treats as "reject, consume nothing" (FR-1.2/FR-2.4).
			// Deliberately NOT a Spec entry — Spec is the consumable effect map and
			// this is an award amount.
			m.Meso = uint32(i.GetIntegerWithDefault("meso", 0))
			m.TradeBlock = i.GetBool("tradeBlock", false)
			m.StateChangeItem = uint32(i.GetIntegerWithDefault("stateChangeItem", 0))
			// info/npc — the shop NPC a remote-merchant item (0545.img) opens.
			// Mirrors consumable/reader.go's identical read.
			m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))
			if i.GetIntegerWithDefault("isBgmOrEffect", 0) == 1 {
				m.BgmPath = i.GetString("bgmPath", "")
			}

			// Parse rate multiplier from info/rate (used for coupon rate display and calculation)
			if rate := i.GetIntegerWithDefault("rate", 0); rate != 0 {
				m.Spec[SpecTypeRate] = rate
			}

			// Parse time windows from info/time imgdir (e.g., "MON:18-20", "TUE:00-24")
			if timeNode, err := i.ChildByName("time"); err == nil && timeNode != nil {
				var windows []TimeWindow
				for _, sn := range timeNode.StringNodes {
					if tw, ok := parseTimeWindow(sn.Value); ok {
						windows = append(windows, tw)
					}
				}
				if len(windows) > 0 {
					m.TimeWindows = windows
				}
			}

			// 0519 pet skill pouches: the skill key(s) and add flag live under
			// info (not spec) — add=1 grants the skill, add=0 removes it.
			for _, k := range skill.All() {
				if i.GetBool(string(k), false) {
					m.PetSkills = append(m.PetSkills, string(k))
				}
			}
			if len(m.PetSkills) > 0 {
				m.PetSkillAdd = i.GetBool("add", false)
			}

			s, err := cxml.ChildByName("spec")
			if err == nil && s != nil {
				// Parse standard spec properties
				m.Spec[SpecTypeInc] = s.GetIntegerWithDefault(string(SpecTypeInc), 0)
				m.Spec[SpecTypeIndexZero] = s.GetIntegerWithDefault(string(SpecTypeIndexZero), 0)
				m.Spec[SpecTypeIndexOne] = s.GetIntegerWithDefault(string(SpecTypeIndexOne), 0)
				m.Spec[SpecTypeIndexTwo] = s.GetIntegerWithDefault(string(SpecTypeIndexTwo), 0)
				m.Spec[SpecTypeIndexThree] = s.GetIntegerWithDefault(string(SpecTypeIndexThree), 0)
				m.Spec[SpecTypeIndexFour] = s.GetIntegerWithDefault(string(SpecTypeIndexFour), 0)
				m.Spec[SpecTypeIndexFive] = s.GetIntegerWithDefault(string(SpecTypeIndexFive), 0)
				m.Spec[SpecTypeIndexSix] = s.GetIntegerWithDefault(string(SpecTypeIndexSix), 0)
				m.Spec[SpecTypeIndexSeven] = s.GetIntegerWithDefault(string(SpecTypeIndexSeven), 0)
				m.Spec[SpecTypeIndexEight] = s.GetIntegerWithDefault(string(SpecTypeIndexEight), 0)
				m.Spec[SpecTypeIndexNine] = s.GetIntegerWithDefault(string(SpecTypeIndexNine), 0)

				// Parse rate coupon properties from spec node (EXP coupons 0521.img, Drop coupons 0536.img)
				if expR := s.GetIntegerWithDefault(string(SpecTypeExpR), 0); expR != 0 {
					m.Spec[SpecTypeExpR] = expR
				}
				if drpR := s.GetIntegerWithDefault(string(SpecTypeDrpR), 0); drpR != 0 {
					m.Spec[SpecTypeDrpR] = drpR
				}
				if time := s.GetIntegerWithDefault(string(SpecTypeTime), 0); time != 0 {
					m.Spec[SpecTypeTime] = time
				}

				// Transformation coupons (0530.img): the Morph.wz creature id and the
				// flat HP heal. Omit-when-zero, matching expR/drpR/time above.
				if morph := s.GetIntegerWithDefault(string(SpecTypeMorph), 0); morph != 0 {
					m.Spec[SpecTypeMorph] = morph
				}
				if hp := s.GetIntegerWithDefault(string(SpecTypeHp), 0); hp != 0 {
					m.Spec[SpecTypeHp] = hp
				}
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
