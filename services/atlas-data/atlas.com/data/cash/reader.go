package cash

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
)

func parseCashId(name string) (uint32, error) {
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

			s, err := cxml.ChildByName("spec")
			if err == nil && s != nil {
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
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
