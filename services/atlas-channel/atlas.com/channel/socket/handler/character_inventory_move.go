package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	inventory3 "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterInventoryMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := inventory3.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		inventoryType := inventory2.Type(p.InventoryType())
		source := p.Source()
		destination := p.Destination()
		count := p.Count()

		if source < 0 && destination > 0 {
			err := compartment.NewProcessor(l, ctx).Unequip(s.CharacterId(), inventoryType, source, destination)
			if err != nil {
				l.WithError(err).Errorf("Error removing equipment equipped in slot [%d] for character [%d].", source, s.CharacterId())
			}
			return
		}
		if destination < 0 {
			err := compartment.NewProcessor(l, ctx).Equip(s.CharacterId(), inventoryType, source, destination)
			if err != nil {
				l.WithError(err).Errorf("Error equipping equipment from slot [%d] for character [%d].", source, s.CharacterId())
			}
			return
		}
		if destination == 0 {
			c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character [%d] dropping item.", s.CharacterId())
				return
			}
			err = compartment.NewProcessor(l, ctx).Drop(s.Field(), s.CharacterId(), inventoryType, source, count, c.X(), c.Y())
			if err != nil {
				l.WithError(err).Errorf("Error dropping [%d] item from slot [%d] for character [%d].", count, source, s.CharacterId())
			}
			return
		}

		err := compartment.NewProcessor(l, ctx).Move(s.CharacterId(), inventoryType, source, destination)
		if err != nil {
			l.WithError(err).Errorf("Error moving item from slot [%d] to slot [%d] for character [%d].", source, destination, s.CharacterId())
		}
	}
}
