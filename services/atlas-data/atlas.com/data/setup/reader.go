package setup

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strconv"
)

func parseSetupId(name string) (uint32, error) {
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
			setupId, err := parseSetupId(cxml.Name)
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}
			l.Debugf("Processing setup [%d].", setupId)

			i, err := cxml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			m := RestModel{
				Id: setupId,
			}
			m.Price = uint32(i.GetIntegerWithDefault("price", 0))
			m.SlotMax = uint32(i.GetIntegerWithDefault("slotMax", 0))
			m.RecoveryHP = uint32(i.GetIntegerWithDefault("recoveryHP", 0))
			m.TradeBlock = i.GetBool("tradeBlock", false)
			m.NotSale = i.GetBool("notSale", false)
			m.ReqLevel = uint32(i.GetIntegerWithDefault("reqLevel", 0))
			m.DistanceX = uint32(i.GetIntegerWithDefault("distanceX", 0))
			m.DistanceY = uint32(i.GetIntegerWithDefault("distanceY", 0))
			m.MaxDiff = uint32(i.GetIntegerWithDefault("maxDiff", 0))
			m.Direction = uint32(i.GetIntegerWithDefault("direction", 0))

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}