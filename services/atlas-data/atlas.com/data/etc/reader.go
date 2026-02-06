package etc

import (
	"atlas-data/xml"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func parseEtcId(name string) (uint32, error) {
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
			etcId, err := parseEtcId(cxml.Name)
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}
			l.Debugf("Processing etc [%d].", etcId)

			i, err := cxml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			m := RestModel{
				Id: etcId,
			}
			m.Price = uint32(i.GetIntegerWithDefault("price", 0))
			m.UnitPrice = i.GetDouble("unitPrice", 0)
			m.SlotMax = uint32(i.GetIntegerWithDefault("slotMax", 100))
			m.TimeLimited = i.GetBool("timeLimited", false)

			// Parse replace attributes if present
			if replaceNode, err := i.ChildByName("replace"); err == nil && replaceNode != nil {
				m.ReplaceItemId = uint32(replaceNode.GetIntegerWithDefault("itemid", 0))
				m.ReplaceMessage = replaceNode.GetString("msg", "")
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
