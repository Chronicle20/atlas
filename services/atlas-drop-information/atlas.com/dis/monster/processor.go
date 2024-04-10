package monster

import (
	drop2 "atlas-dis/monster/drop"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetAll(l logrus.FieldLogger, db *gorm.DB) []Model {
	ms := make(map[uint32]Model)
	drops := drop2.GetAll(l, db)

	for _, d := range drops {
		if _, ok := ms[d.MonsterId()]; !ok {
			m := Model{
				id:    d.MonsterId(),
				drops: make([]drop2.Model, 0),
			}
			ms[d.MonsterId()] = m
		}
		m := ms[d.MonsterId()]
		m.drops = append(m.drops, d)
		ms[d.MonsterId()] = m
	}

	results := make([]Model, 0)
	for _, m := range ms {
		results = append(results, m)
	}
	return results
}

func GetWhoDropItem(l logrus.FieldLogger, db *gorm.DB) func(itemId uint32) []Model {
	return func(itemId uint32) []Model {
		mids := make(map[uint32]bool)
		drops := drop2.GetAll(l, db)
		for _, d := range drops {
			if d.ItemId() == itemId {
				mids[d.MonsterId()] = true
			}
		}

		ms := make(map[uint32]Model)
		for _, d := range drops {
			if _, ok := mids[d.MonsterId()]; !ok {
				continue
			}

			if _, ok := ms[d.MonsterId()]; !ok {
				m := Model{
					id:    d.MonsterId(),
					drops: make([]drop2.Model, 0),
				}
				ms[d.MonsterId()] = m
			}
			m := ms[d.MonsterId()]
			m.drops = append(m.drops, d)
			ms[d.MonsterId()] = m
		}

		results := make([]Model, 0)
		for _, m := range ms {
			results = append(results, m)
		}
		return results
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB) func(monsterId uint32) (Model, error) {
	return func(monsterId uint32) (Model, error) {
		drops := drop2.GetForMonster(l, db)(monsterId)
		m := Model{
			id:    monsterId,
			drops: drops,
		}
		return m, nil
	}
}
