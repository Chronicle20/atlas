package equipment

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Delete(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(m Model) error {
	return func(m Model) error {
		var equipables = []slot.Model{m.hat, m.medal, m.forehead, m.ring1, m.ring2, m.eye, m.earring, m.shoulder, m.cape, m.top, m.pendant, m.weapon, m.shield, m.gloves, m.bottom, m.belt, m.ring3, m.ring4, m.shoes}
		for _, e := range equipables {
			err := deleteBySlot(l, db, span, tenant)(e)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func deleteBySlot(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(m slot.Model) error {
	return func(m slot.Model) error {
		e := m.Equipable
		if e == nil {
			return nil
		}
		return equipable.DeleteByReferenceId(l, db, span, tenant)(e.ReferenceId())
	}
}
