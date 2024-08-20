package handler

import (
	"atlas-channel/character/inventory"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CharacterInventoryMoveHandle = "CharacterInventoryMoveHandle"

func CharacterInventoryMoveHandleFunc(l logrus.FieldLogger, span opentracing.Span, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		inventoryType := r.ReadByte()
		source := r.ReadInt16()
		destination := r.ReadInt16()
		count := r.ReadInt16()
		l.Debugf("Character [%d] attempting to move [%d] item in inventory [%d]. source [%d] destination [%d] updateTime [%d]", s.CharacterId(), count, inventoryType, source, destination, updateTime)
		if source < 0 && destination > 0 {
			err := inventory.Unequip(l, span, s.Tenant())(s.CharacterId(), source, destination)
			if err != nil {
				l.WithError(err).Errorf("Error removing equipment equipped in slot [%d] for character [%d].", source, s.CharacterId())
			}
			return
		}
		if destination < 0 {
			err := inventory.Equip(l, span, s.Tenant())(s.CharacterId(), source, destination)
			if err != nil {
				l.WithError(err).Errorf("Error equipping equipment from slot [%d] for character [%d].", source, s.CharacterId())
			}
			return
		}
		if destination == 0 {
			// drop
			return
		}
		// move
	}
}
