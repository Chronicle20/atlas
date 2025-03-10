package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/inventory"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetSpawnHandle = "PetSpawnHandle"

func PetSpawnHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		slot := r.ReadInt16()
		lead := r.ReadBool()
		l.Debugf("Character [%d] triggered PetSpawnHandle. updateTime [%d], slot [%d], lead [%t].", s.CharacterId(), updateTime, slot, lead)

		i, err := character.GetItemInSlot(l)(ctx)(s.CharacterId(), byte(inventory.TypeValueCash), slot)()
		if err != nil {
			return
		}

		c, err := character.GetById(l)(ctx)(character.PetModelDecorator(l)(ctx))(s.CharacterId())
		if err != nil {
			return
		}

		var p *pet.Model
		spawned := false
		for _, rp := range c.Pets() {
			if rp.InventoryItemId() == i.Id() {
				p = &rp
				if p.Slot() != -1 {
					spawned = true
				}
			}
		}
		if p == nil {
			return
		}
		if spawned {
			_ = pet.Despawn(l)(ctx)(s.CharacterId(), p.Id())
		} else {
			_ = pet.Spawn(l)(ctx)(s.CharacterId(), p.Id(), lead)
		}
	}
}
