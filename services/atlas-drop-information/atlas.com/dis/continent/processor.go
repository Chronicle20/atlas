package continent

import (
	"atlas-dis/continent/drop"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetAll(l logrus.FieldLogger, db *gorm.DB) []Model {
	ms := make(map[int32]Model)
	drops := drop.GetAll(l, db)

	for _, d := range drops {
		if _, ok := ms[d.ContinentId()]; !ok {
			m := Model{
				id:    d.ContinentId(),
				drops: make([]drop.Model, 0),
			}
			ms[d.ContinentId()] = m
		}
		m := ms[d.ContinentId()]
		m.drops = append(m.drops, d)
		ms[d.ContinentId()] = m
	}

	results := make([]Model, 0)
	for _, m := range ms {
		results = append(results, m)
	}
	return results
}
