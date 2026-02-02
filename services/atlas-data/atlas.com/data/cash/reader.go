package cash

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
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
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
