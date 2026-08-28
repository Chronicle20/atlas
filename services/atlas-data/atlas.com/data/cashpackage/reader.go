package cashpackage

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
		for _, cxml := range exml.ChildNodes {
			id, err := strconv.ParseUint(cxml.Name, 10, 32)
			if err != nil {
				l.WithError(err).Warnf("Unable to parse cash package id [%s].", cxml.Name)
				continue
			}

			m := RestModel{}
			m.Id = uint32(id)
			l.Debugf("Processing cash package [%d].", m.Id)

			m.SerialNumbers = make([]uint32, 0)
			if snNode, err := cxml.ChildByName("SN"); err == nil {
				for _, sn := range snNode.IntegerNodes {
					val, err := strconv.ParseUint(sn.Value, 10, 32)
					if err != nil {
						l.WithError(err).Warnf("Unable to parse cash package [%d] serial number [%s].", m.Id, sn.Value)
						continue
					}
					m.SerialNumbers = append(m.SerialNumbers, uint32(val))
				}
			}

			res = append(res, m)
		}

		return model.FixedProvider(res)
	}
}
