package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterInventoryMoveHandle = "CharacterInventoryMoveHandle"

func CharacterInventoryMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		inventoryType := inventory2.Type(r.ReadByte())
		source := r.ReadInt16()
		destination := r.ReadInt16()
		count := r.ReadInt16()

		l.Debugf("Character [%d] attempting to move [%d] item in inventory [%d]. source [%d] destination [%d] updateTime [%d]", s.CharacterId(), count, inventoryType, source, destination, updateTime)
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
