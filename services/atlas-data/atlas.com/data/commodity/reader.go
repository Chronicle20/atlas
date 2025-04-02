package commodity

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)
		for _, cxml := range exml.ChildNodes {
			m := RestModel{}
			m.Id = uint32(cxml.GetIntegerWithDefault("SN", 0))
			l.Debugf("Processing commodity [%d].", m.Id)
			m.ItemId = uint32(cxml.GetIntegerWithDefault("ItemId", 0))
			m.Count = uint32(cxml.GetIntegerWithDefault("Count", 0))
			m.Price = uint32(cxml.GetIntegerWithDefault("Price", 0))
			m.Period = uint32(cxml.GetIntegerWithDefault("Period", 0))
			m.Priority = uint32(cxml.GetIntegerWithDefault("Priority", 0))
			m.Gender = byte(cxml.GetIntegerWithDefault("Gender", 0))
			m.OnSale = cxml.GetBool("OnSale", false)
			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
