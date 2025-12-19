package continent

import (
	"atlas-drops-information/continent/drop"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
		return func(db *gorm.DB) func() ([]Model, error) {
			return func() ([]Model, error) {
				ms := make(map[int32]Model)
				drops, err := drop.GetAll(l)(ctx)(db)()
				if err != nil {
					return nil, err
				}

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
				return results, nil
			}
		}
	}
}
