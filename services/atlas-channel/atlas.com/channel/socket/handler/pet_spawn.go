package handler

import (
	"atlas-channel/character"
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

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator)(s.CharacterId())
		if err != nil {
			return
		}
		a, ok := c.Inventory().Cash().FindBySlot(slot)
		if !ok {
			return
		}
		if !a.IsPet() {
			return
		}
		spawned := a.PetSlot() != -1
		if spawned {
			_ = pet.NewProcessor(l, ctx).Despawn(s.CharacterId(), a.PetId())
		} else {
			_ = pet.NewProcessor(l, ctx).Spawn(s.CharacterId(), a.PetId(), lead)
		}
	}
}
