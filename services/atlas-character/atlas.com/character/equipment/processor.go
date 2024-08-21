package equipment

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/equipment/slot/information"
	"atlas-character/equipment/statistics"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
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
		return equipable.DeleteByReferenceId(l)(span)(db)(tenant)(e.ReferenceId())
	}
}

type DestinationProvider func(itemId uint32) model.Provider[int16]

func FixedDestinationProvider(destination int16) DestinationProvider {
	return func(itemId uint32) model.Provider[int16] {
		return func() (int16, error) {
			return destination, nil
		}
	}
}

func GetEquipmentDestination(l logrus.FieldLogger) func(span opentracing.Span) func(tenant tenant.Model) DestinationProvider {
	return func(span opentracing.Span) func(tenant tenant.Model) DestinationProvider {
		return func(tenant tenant.Model) DestinationProvider {
			return func(itemId uint32) model.Provider[int16] {
				slots, err := information.GetById(l, span, tenant)(itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve destination slots for item [%d].", itemId)
					return model.ErrorProvider[int16](err)
				} else if len(slots) <= 0 {
					l.Errorf("Unable to retrieve destination slots for item [%d].", itemId)
					return model.ErrorProvider[int16](err)
				}
				is, err := statistics.GetById(l, span, tenant)(itemId)
				if err != nil {
					return model.ErrorProvider[int16](err)
				}

				destination := int16(0)
				if is.Cash() {
					destination = slots[0].Slot() - 100
				} else {
					destination = slots[0].Slot()
				}
				return model.FixedProvider(destination)
			}
		}
	}
}
