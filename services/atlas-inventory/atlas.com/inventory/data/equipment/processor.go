package equipment

import (
	"atlas-inventory/data/equipment/slot"
	"atlas-inventory/data/equipment/statistics"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l             logrus.FieldLogger
	ctx           context.Context
	slotProcessor *slot.Processor
	statProcessor *statistics.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:             l,
		ctx:           ctx,
		slotProcessor: slot.NewProcessor(l, ctx),
		statProcessor: statistics.NewProcessor(l, ctx),
	}
	return p
}

type DestinationProvider func(itemId uint32) model.Provider[int16]

func (p *Processor) DestinationSlotProvider(suggested int16) DestinationProvider {
	return func(itemId uint32) model.Provider[int16] {
		slots, err := p.slotProcessor.GetById(itemId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve destination slots for item [%d].", itemId)
			return model.ErrorProvider[int16](err)
		} else if len(slots) <= 0 {
			p.l.Errorf("Unable to retrieve destination slots for item [%d].", itemId)
			return model.ErrorProvider[int16](err)
		}
		is, err := p.statProcessor.GetById(itemId)
		if err != nil {
			return model.ErrorProvider[int16](err)
		}
		slot := slots[0]

		destination := int16(0)
		if is.Cash() {
			if slot.Name() == "PET_EQUIP" {
				destination = suggested
			} else {
				destination = slot.Slot() - 100
			}
		} else {
			destination = slot.Slot()
		}
		return model.FixedProvider(destination)
	}
}
